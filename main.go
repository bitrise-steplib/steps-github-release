package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/log"
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

func main() {
	apiToken := os.Getenv("api_token")
	repositoryURL := os.Getenv("repository_url")
	tag := os.Getenv("tag")
	commit := os.Getenv("commit")
	name := os.Getenv("name")
	body := os.Getenv("body")
	draft := os.Getenv("draft")

	log.Infof("Configs:")
	log.Printf("- api_token: %s", apiToken)
	log.Printf("- repository_url: %s", repositoryURL)
	log.Printf("- tag: %s", tag)
	log.Printf("- commit: %s", commit)
	log.Printf("- name: %s", name)
	log.Printf("- body: %s", body)
	log.Printf("- draft: %s", draft)

	if apiToken == "" {
		failf("api_token not defined")
	}
	if repositoryURL == "" {
		failf("repository_url not defined")
	}
	if tag == "" {
		failf("tag not defined")
	}
	if commit == "" {
		failf("commit not defined")
	}
	if name == "" {
		failf("name not defined")
	}
	if body == "" {
		failf("body not defined")
	}
	if draft == "" {
		failf("draft not defined")
	}

	ctx := context.Background()
	token := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: apiToken})
	authClient := oauth2.NewClient(ctx, token)
	client := github.NewClient(authClient)

	isDraft := (draft == "yes")
	release := &github.RepositoryRelease{
		TagName:         &tag,
		TargetCommitish: &commit,
		Name:            &name,
		Body:            &body,
		Draft:           &isDraft,
	}

	_, owner, name := parseRepo(repositoryURL)
	newRelease, _, err := client.Repositories.CreateRelease(ctx, owner, name, release)
	if err != nil {
		log.Errorf("Failed to create release: %s", err)
		os.Exit(1)
	}

	printableRelease := newRelease.String()

	b, err := json.MarshalIndent(newRelease, "", "  ")
	if err == nil {
		printableRelease = string(b)
	}

	fmt.Println()
	log.Infof("Release created:")
	log.Printf(printableRelease)
}
