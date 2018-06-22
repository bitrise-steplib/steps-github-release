package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

// formats:
// https://hostname/owner/repository.git
// git@hostname:owner/repository.git
// ssh://git@hostname:port/owner/repository.git
func parseRepo(url string) (host string, owner string, name string) {
	url = strings.TrimSuffix(url, ".git")

	var repo string
	switch {
	case strings.HasPrefix(url, "https://"):
		url = strings.TrimPrefix(url, "https://")
		idx := strings.Index(url, "/")
		host, repo = url[:idx], url[idx+1:]
	case strings.HasPrefix(url, "git@"):
		url = url[strings.Index(url, "@")+1:]
		idx := strings.Index(url, ":")
		host, repo = url[:idx], url[idx+1:]
	case strings.HasPrefix(url, "ssh://"):
		url = url[strings.Index(url, "@")+1:]
		host = url[:strings.Index(url, ":")]
		repo = url[strings.Index(url, "/")+1:]
	}

	split := strings.Split(repo, "/")
	return host, split[0], split[1]
}

func failf(format string, args ...interface{}) {
	log.Errorf(format, args...)
	os.Exit(1)
}

// Config ...
type Config struct {
	APIToken      stepconf.Secret `env:"api_token,required"`
	RepositoryURL string          `env:"repository_url,required"`
	Tag           string          `env:"tag,required"`
	Commit        string          `env:"commit,required"`
	Name          string          `env:"name,required"`
	Body          string          `env:"body,required"`
	Draft         string          `env:"draft,opt[yes,no]"`
}

func main() {
	var c Config
	if err := stepconf.Parse(&c); err != nil {
		failf("Issue with input: %s")
	}
	stepconf.Print(c)

	ctx := context.Background()
	token := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.APIToken.String()})
	authClient := oauth2.NewClient(ctx, token)
	client := github.NewClient(authClient)

	isDraft := (c.Draft == "yes")
	release := &github.RepositoryRelease{
		TagName:         &c.Tag,
		TargetCommitish: &c.Commit,
		Name:            &c.Name,
		Body:            &c.Body,
		Draft:           &isDraft,
	}

	_, owner, repo := parseRepo(c.RepositoryURL)
	newRelease, _, err := client.Repositories.CreateRelease(ctx, owner, repo, release)
	if err != nil {
		failf("Failed to create release: %s", err)
	}

	fmt.Println()
	log.Infof("Release created:")
	log.Printf(newRelease.GetHTMLURL())
}
