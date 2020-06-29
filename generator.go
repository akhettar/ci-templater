package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"
)

// RepoName is the repo name to be processed
const RepoName = "repo"

// Config the Github action config
type Config struct {
	Description  string         `yaml:"description"`
	Repositories []Repositories `yaml:"repositories"`
}

// Repository type
type Repository = map[string]string

// Repositories type
type Repositories struct {
	Repo Repository `yaml:"repo,omitempty"`
}

// TemplateGenerator type
type TemplateGenerator struct {
	conf   *Config
	client *GitClient
}

// Report holds information about the processing of the repo
type Report struct {
	name string
	err  error
}

// Generartor creates an instance of the Templater
func Generartor() *TemplateGenerator {

	// make sure we are always starting from fresh git repo
	clearUp("repos/")

	// assert the presence of the required environment variable
	validate()

	// read the Github action template. At the moment only one config with all the defined the repositories to be processed
	data, err := ioutil.ReadFile("./config.yml")
	if err != nil {
		log.Fatal(err)
	}
	var config Config

	if err = yaml.NewDecoder(bytes.NewReader(data)).Decode(&config); err != nil {
		log.Fatal(err)
	}

	return &TemplateGenerator{conf: &config, client: NewGitClient()}
}

// generates all github action config for all the repos
func (t *TemplateGenerator) run() []Report {

	var wg sync.WaitGroup
	reportChan := make(chan Report, len(t.conf.Repositories))
	repo := os.Getenv(RepoName)

	for _, r := range t.conf.Repositories {
		repoName := getRepoName(r.Repo["url"])
		repoPath := fmt.Sprintf("repos/%s", repoName)
		branch := fmt.Sprintf("%s-%d", "githubAction", time.Now().UnixNano())

		// escaping the repos if the flag is set to process only one given repository
		if repo != "" && repoName != repo {
			log.Printf("[repo flag] %s", repo)
			log.Printf("[repo] %s", repoName)

			continue
		}

		log.Printf("processing repo: %s", r.Repo["url"])
		wg.Add(1)

		// process all the repositories in parallel
		go func(wg *sync.WaitGroup, result chan Report, r Repository) {
			defer wg.Done()
			//1. Prepare the repo
			url := r["url"]
			if err := t.client.prepareRepo(repoPath, branch, url); err != nil {
				log.Println("failed to prepare the repo")
				result <- Report{url, err}
				clearUp(repoPath)
				return
			}

			// 2. render the template write it the repo path
			if err := renderTemplate(r, repoPath); err != nil {
				result <- Report{url, err}
				clearUp(repoPath)
				return
			}

			// 3. push and commit
			if err := t.client.push(branch, url); err != nil {
				result <- Report{url, err}
				clearUp(repoPath)
				return
			}

			// clean up
			if dryRun, _ := strconv.ParseBool(os.Getenv(DryRun)); !dryRun {
				clearUp(repoPath)
			}

			// everything went well publish report in the chanel
			result <- Report{url, nil}

		}(&wg, reportChan, r.Repo)
	}

	// wait for all the reports to come through
	wg.Wait()
	close(reportChan)

	// print out the report
	reports := make([]Report, len(t.conf.Repositories), len(t.conf.Repositories))
	for r := range reportChan {
		if r.err != nil {
			log.Printf("failed to process the repo: %v with error: %v %v", r.name, r.err, "❌")
			reports = append(reports, r)
		} else {
			log.Printf("repo: %s was successfully processed: %v", r.name, "✅")
		}
	}
	log.Println("successfully completed processing of all the repositories")
	return reports
}

func renderTemplate(repo Repository, repoLocation string) error {

	gitHubDir := fmt.Sprintf("%s/%s", repoLocation, ".github")
	workflowDir := fmt.Sprintf("%s/%s", repoLocation, ".github/workflows")
	os.Mkdir(gitHubDir, os.ModePerm)
	os.Mkdir(workflowDir, os.ModePerm)

	templateBytes, err := ioutil.ReadFile(fmt.Sprintf("templates/%s-template.yml", repo["lang"]))
	t := template.Must(template.New("githubAction").Funcs(templateFunc()).Delims("{[", "]}").Parse(string(templateBytes)))

	//overwrite the config file with the template
	filePath := fmt.Sprintf("%s/%s", repoLocation, ".github/workflows/config.yml")
	fileToWrite, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("failed to render the template, error: %v", err.Error())
		return err
	}
	err = t.Execute(fileToWrite, repo)
	return err
}

// Template function to be called from the template
// exp: {[template "DEPLOYMENT" map "Env" "dev" "Name" .Name  "AWSAccountNumber" "326458601802" ]}
func templateFunc() template.FuncMap {
	return template.FuncMap{
		"ToUpper": strings.ToUpper,
		"map": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid map call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("map keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}
}

// Asserts the presence of all the required environment variables
func validate() {

	// overriding the environement variables if new ones set as flags
	token := flag.String(GithubAuthToken, os.Getenv(GithubAuthToken), "Github token")
	if *token == "" {
		log.Fatalf("%s is required", GithubAuthToken)
	}

	email := flag.String(GithubEmail, os.Getenv(GithubEmail), "Github email")
	if *email == "" {
		log.Fatalf("%s is required", GithubEmail)
	}

	pwd := flag.String(GPGPassword, os.Getenv(GPGPassword), "GPG password")
	repo := flag.String(RepoName, "", "repo name to be processed")
	dryrun := flag.String(DryRun, "false", "flag to disable pushing the changes to the branch and creating pull request")

	flag.Parse()

	os.Setenv(GPGPassword, *pwd)
	os.Setenv(RepoName, *repo)
	os.Setenv(GithubAuthToken, *token)
	os.Setenv(GithubEmail, *email)
	os.Setenv(DryRun, *dryrun)

}

func clearUp(path string) {
	if err := os.RemoveAll(path); err != nil {
		log.Printf("failed to delete the generated local repository: %s", path)
	}
}
