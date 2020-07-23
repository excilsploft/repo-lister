package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

type Repos struct {
	Repos []Repo `yaml:"repos"`
}

type Repo struct {
	Name     string   `yaml:"name"`
	GitURL   string   `yaml:"giturl"`
	CloneURL string   `yaml:"cloneurl"`
	Branches []string `yaml:"branches"`
}

func main() {

	var token string
	var org string
	var debug bool

	flag.StringVar(&token, "t", lookupStringVar("GITHUB_AUTH_TOKEN", ""), "github token")
	flag.StringVar(&org, "o", lookupStringVar("GITHUB_ORG", ""), "github organization")
	flag.BoolVar(&debug, "d", false, "debug mode")
	flag.Parse()

	if len(token) < 1 {
		fmt.Fprintf(os.Stderr, "No value for '-t' or GITHUB_AUTH_TOKEN\n")
		os.Exit(1)
	}

	if len(org) < 1 {
		fmt.Fprintf(os.Stderr, "No value for '-o' or GITHUB_ORG\n")
		os.Exit(1)
	}

	orgRepos := Repos{}

	// set the context to a background context: https://golang.org/pkg/context/#Background
	ctx := context.Background()
	// get the oauth client witht the ghub token and context
	client := getOauthClient(ctx, token)

	// set our repo and branch list options
	options := github.RepositoryListByOrgOptions{Type: "all", Sort: "full_name"}
	blOptions := github.BranchListOptions{}

	// get the repository list for our org
	repos, resp, err := client.Repositories.ListByOrg(ctx, org, &options)
	if err != nil {
		fmt.Fprint(os.Stderr, "An Error Occurred: %s\n", err)
		os.Exit(1)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Repository List Response: %s\n", resp)
	}

	for _, v := range repos {

		repo := Repo{Name: *v.Name, CloneURL: *v.CloneURL, GitURL: *v.GitURL}
		branches, resp, err := client.Repositories.ListBranches(ctx, org, *v.Name, &blOptions)
		if err != nil {
			fmt.Fprint(os.Stderr, "An Error Occurred: %s\n", err)
			if debug {
				fmt.Fprintf(os.Stderr, "Branch List Response: %s\n", resp)
			}
			continue
		}

		for _, b := range branches {
			repo.Branches = append(repo.Branches, *b.Name)
		}

		orgRepos.Repos = append(orgRepos.Repos, repo)
	}

	// get a yaml encoder
	yEncoder := yaml.NewEncoder(os.Stdout)
	err = yEncoder.Encode(&orgRepos)
	if err != nil {
		fmt.Fprintf(os.Stderr, "An Error Occurred in Encoding: %s\n", err)
	}

	yEncoder.Close()

	os.Exit(0)

}

// function to get a string from an env var if it exists but return a default value if it does not
func lookupStringVar(key string, defVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defVal
}

// private function to get a github httpclient with oauth headers and tokens
func getOauthClient(ctx context.Context, token string) *github.Client {

	st := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tokenClient := oauth2.NewClient(ctx, st)

	client := github.NewClient(tokenClient)
	return client
}
