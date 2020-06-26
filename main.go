package main

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

	// RepoName is the repo name to be processed
	RepoName = "repo"
)

func main() {
	Generartor().run()
}
