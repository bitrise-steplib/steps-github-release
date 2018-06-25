package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/google/go-github/github"
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
	Username      stepconf.Secret `env:"username,required"`
	RepositoryURL string          `env:"repository_url,required"`
	Tag           string          `env:"tag,required"`
	Commit        string          `env:"commit,required"`
	Name          string          `env:"name,required"`
	Body          string          `env:"body,required"`
	Draft         string          `env:"draft,opt[yes,no]"`
}

// RoundTrip ...
func (c Config) RoundTrip(req *http.Request) (*http.Response, error) {
	userPassPair := []byte(fmt.Sprintf("%s:%s", string(c.Username), string(c.APIToken)))
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(userPassPair))
	return http.DefaultTransport.RoundTrip(req)
}

func main() {
	var c Config
	if err := stepconf.Parse(&c); err != nil {
		failf("Issue with input: %s", err)
	}
	stepconf.Print(c)

	basicAuthClient := &http.Client{Transport: c}
	client := github.NewClient(basicAuthClient)

	isDraft := (c.Draft == "yes")
	release := &github.RepositoryRelease{
		TagName:         &c.Tag,
		TargetCommitish: &c.Commit,
		Name:            &c.Name,
		Body:            &c.Body,
		Draft:           &isDraft,
	}

	_, owner, repo := parseRepo(c.RepositoryURL)
	newRelease, _, err := client.Repositories.CreateRelease(context.Background(), owner, repo, release)
	if err != nil {
		failf("Failed to create release: %s\n", err)
	}

	fmt.Println()
	log.Infof("Release created:")
	log.Printf(newRelease.GetHTMLURL())
}
