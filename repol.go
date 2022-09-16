package main

import (
	"context"
	"flag"
	"fmt"
	"os"
    "sync"

	"github.com/google/go-github/v32/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v3"
)

// Struct to hold a top level list of repos
type Repos struct {
	Repos []Repo `yaml:"repos"`
}

// Struct to hold the fields we care about in a repo
type Repo struct {
	Name     string   `yaml:"name"`
	GitURL   string   `yaml:"giturl"`
	CloneURL string   `yaml:"cloneurl"`
	Branches []string `yaml:"branches"`
}

func main() {

    var enterpriseUrl string
	var token string
	var org string
	var debug bool

	flag.StringVar(&enterpriseUrl, "e", "", "Github Enterprise Url")
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


	// set the context to a background context: https://golang.org/pkg/context/#Background
	ctx := context.Background()

    var client *github.Client
    var clientErr error
	// if enterprise get enterprise client else get the oauth client with the ghub token and context
    if len(enterpriseUrl) > 0 {
        client, clientErr = getEnterpriseClient(ctx, enterpriseUrl, token)
        if clientErr != nil {
            fmt.Fprintf(os.Stderr, "Error Getting Enterprise Client: %s\n", clientErr)
            os.Exit(1)
        }
    } else {
	    client = getOauthClient(ctx, token)
    }

	// set our repo and branch list options
    options := github.RepositoryListByOrgOptions{Type: "all", Sort: "full_name", ListOptions: github.ListOptions{PerPage: 100}}

	// get the repository list for our org
	repos, resp, err := client.Repositories.ListByOrg(ctx, org, &options)
	if err != nil {
		fmt.Fprint(os.Stderr, "An Error Occurred: %s\n", err)
		os.Exit(1)
	}

	if debug {
		fmt.Fprintf(os.Stderr, "Repository List Response: %s\n", resp)
	}

    blOptions := github.BranchListOptions{ListOptions: github.ListOptions{PerPage: 200}}
    orgRepos := Repos{}
    ch  := make(chan Repo)
    var wg sync.WaitGroup

    for _, v := range repos {
        wg.Add(1)
        go func(v *github.Repository, ch chan <- Repo, wg *sync.WaitGroup) {
            defer wg.Done()
            repo := Repo{Name: *v.Name, CloneURL: *v.CloneURL, GitURL: *v.GitURL}
            branches, resp, err := client.Repositories.ListBranches(ctx, org, *v.Name, &blOptions)
            if err != nil {
                fmt.Fprint(os.Stderr, "An Error Occurred: %s\n", err)
                if debug {
                    fmt.Fprintf(os.Stderr, "Branch List Response: %s\n", resp)
                }
            }

            for _, b := range branches {
                repo.Branches = append(repo.Branches, *b.Name)
            }

            // orgRepos.Repos = append(orgRepos.Repos, repo)
            ch <- repo
        }(v, ch, &wg)
    }

    go func() {
        wg.Wait()
        close(ch)
    }()

    for r := range ch {
        orgRepos.Repos = append(orgRepos.Repos, r)
    }

    if debug {
        fmt.Fprintf(os.Stderr, "%+v\n", orgRepos.Repos)
    }
	// get a yaml encoder
	yEncoder := yaml.NewEncoder(os.Stdout)
	err = yEncoder.Encode(&orgRepos.Repos)
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

// private function to wrap getting an enterprise client
func getEnterpriseClient(ctx context.Context, enterpriseUrl string, token string) (*github.Client, error) {

	st := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tokenClient := oauth2.NewClient(ctx, st)

	client, err := github.NewEnterpriseClient(enterpriseUrl, enterpriseUrl, tokenClient)

	return client, err

}
