/*
MIT License
Copyright (c) 2020 akhettar

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/crypto/openpgp"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	git "gopkg.in/src-d/go-git.v4"
)

const (
	// GithubUser for oauth2 authentication
	GithubUser string = "githubTokenAuth"

	// GithubAuthToken environment variable
	GithubAuthToken string = "GITHUB_AUTH_TOKEN"

	// GPGPassword the gpg password
	GPGPassword = "GPG_PASSWORD"

	// GithubEmail the github email of the user running the action generator
	GithubEmail = "GITHUB_EMAIL"

	// DryRun flag to disable commiting the changes to the branch
	DryRun = "DryRun"
)

var regex = regexp.MustCompile(".+(\\/)(.+)")

// GitClient type
type GitClient struct {
	client     *github.Client
	repo       *git.Repository
	tree       *git.Worktree
	gpgEnabled bool
}

// NewGitClient create a new instance of the GitClient
func NewGitClient() *GitClient {
	//create the PR in GitHub
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: os.Getenv(GithubAuthToken)},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)
	return &GitClient{client: client}
}

func (g *GitClient) prepareRepo(repoLocation string, branchName string, url string) error {

	//clone the repo
	r, err := git.PlainClone(repoLocation, false, &git.CloneOptions{
		Auth: getGitHubAuth(),
		URL:  url,
	})
	if err != nil {
		log.Printf("failed to clone the repository: %s with error:%v", url, err.Error())
		return err
	}

	err = r.Fetch(&git.FetchOptions{
		Auth:     getGitHubAuth(),
		RefSpecs: []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
	})
	if err != nil {
		println("failed fetching repository", err.Error())
	}

	//get the work tree
	worktree, err := r.Worktree()

	//create the new branch
	branch := fmt.Sprintf("refs/heads/%s", branchName)
	b := plumbing.ReferenceName(branch)

	err = worktree.Checkout(&git.CheckoutOptions{Create: true, Force: true, Branch: b})
	if err != nil {
		log.Printf("looks like the repo already exist: %v\n", err.Error())
		err = worktree.Checkout(&git.CheckoutOptions{Create: false, Force: false, Branch: b})

		if err != nil {
			log.Println("failed to checkout the repo the branch")
			return err
		}
	}
	g.repo, g.tree = r, worktree
	return nil
}

func (g *GitClient) push(branchName string, url string) error {
	//add the change to the stage

	_, err := g.tree.Add(".github/")
	if err != nil {
		log.Printf("failed to stage the file: %v", err.Error())
		return err
	}

	cmtOptions, err := commitOptions()
	if err != nil {
		return err
	}

	//commit the change
	if _, err = g.tree.Commit("Updating .github/workflows/config.yml", cmtOptions); err != nil {
		log.Println("failed to commit the changes")
		return err
	}

	if dryRun, _ := strconv.ParseBool(os.Getenv(DryRun)); !dryRun {
		//push the change up to the remote
		err = g.repo.Push(&git.PushOptions{Auth: getGitHubAuth()})
		if err != nil {
			log.Println("failed pushing the changes")
			return err
		}

		pl, _, err := g.client.PullRequests.Create(oauth2.NoContext, "cirtak", getRepoName(url), &github.NewPullRequest{
			Title:               github.String("Github action config update"),
			Head:                github.String(branchName),
			Base:                github.String("master"),
			Body:                github.String("Updating Github Action config file"),
			MaintainerCanModify: github.Bool(true),
		})
		if err != nil {
			log.Printf("failed to create a PR with erro: %v. Repo not found", err.Error())
			return err
		}
		log.Printf("Pull request has been successfuly created: %s", *pl.HTMLURL)

	}
	return nil
}

func commitOptions() (*git.CommitOptions, error) {

	// if GPG is eanbled set the signing key
	if os.Getenv(GPGPassword) != "" {

		//grab the private key, decrypt it
		privKey, err := os.Open("gpg-private-key")
		if err != nil {
			log.Printf("failed to read pgp private key: %v", err.Error())
			return nil, err
		}

		es, err := openpgp.ReadArmoredKeyRing(privKey)
		if err != nil {
			log.Printf("failed to read pgp private key: %v", err.Error())
			return nil, err
		}
		key := es[0]
		if err = key.PrivateKey.Decrypt([]byte(os.Getenv(GPGPassword))); err != nil {
			return nil, err
		}

		return &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Github Action Templater",
				Email: os.Getenv(GithubEmail),
				When:  time.Now(),
			},
			All:     true,
			SignKey: key,
		}, nil
	}

	// return without signature
	return &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Github Action Templater",
			Email: os.Getenv(GithubEmail),
			When:  time.Now(),
		},
		All: true,
	}, nil
}

func getRepoName(url string) string {
	parts := regex.FindAllStringSubmatch(url, -1)[0]
	return parts[len(parts)-1]
}

func getGitHubAuth() *http.BasicAuth {
	return &http.BasicAuth{
		Username: GithubUser,
		Password: os.Getenv(GithubAuthToken),
	}
}
