package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/retry"
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
	PreRelease    string          `env:"pre_release,opt[yes,no]"`
	FilesToUpload string          `env:"files_to_upload"`
}

// RoundTrip ...
func (c Config) RoundTrip(req *http.Request) (*http.Response, error) {
	req.SetBasicAuth(string(c.Username), string(c.APIToken))
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

	isDraft := c.Draft == "yes"
	isPreRelease := c.PreRelease == "yes"

	release := &github.RepositoryRelease{
		TagName:         &c.Tag,
		TargetCommitish: &c.Commit,
		Name:            &c.Name,
		Body:            &c.Body,
		Draft:           &isDraft,
		Prerelease:      &isPreRelease,
	}

	_, owner, repo := parseRepo(c.RepositoryURL)
	newRelease, _, err := client.Repositories.CreateRelease(context.Background(), owner, repo, release)
	if err != nil {
		failf("Failed to create release: %s\n", err)
	}

	fmt.Println()
	log.Infof("Release created:")
	log.Printf(newRelease.GetHTMLURL())

	if filelist := strings.TrimSpace(c.FilesToUpload); filelist != "" {
		if err := uploadFileListWithRetry(filelist, client, owner, repo, newRelease.GetID()); err != nil {
			failf("Error during upload: %s", err)
		}
	}
}

func uploadFileListWithRetry(filelist string, client *github.Client, owner string, repo string, id int64) error {
	fmt.Println()
	log.Infof("Uploading assets:")
	files := strings.Split(filelist, "\n")
	for i, filePath := range files {
		if strings.TrimSpace(filePath) == "" {
			continue
		}

		fileName, filePath, err := getFileNameFromPath(filePath)
		if err != nil {
			return err
		}
		log.Printf("(%d/%d) Uploading: %s - %s", i+1, len(files), fileName, filePath)
		fi, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("Failed to open file (%s), error: %s", filePath, err)
		}

		if err := uploadFileWithRetry(filePath, fileName, fi, client, owner, repo, id); err != nil {
			return err
		}
	}
	return nil
}

func uploadFileWithRetry(filePath string, fileName string, fi *os.File, client *github.Client, owner string, repo string, id int64) error {
	if err := retry.Times(3).Wait(5 * time.Second).Try(func(attempt uint) error {
		if attempt > 0 {
			log.Warnf("%d attempt failed", attempt)
		}
		if _, _, err := client.Repositories.UploadReleaseAsset(context.Background(), owner, repo, id, &github.UploadOptions{Name: fileName}, fi); err != nil {
			return fmt.Errorf("Failed to upload file (%s), error: %s", filePath, err)
		}
		log.Donef("- Done")
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func getFileNameFromPath(filePath string) (string, string, error) {
	var fileName string
	if s := strings.Split(filePath, "|"); len(s) > 1 {
		if strings.TrimSpace(s[0]) != "" {
			filePath = s[0]
		} else {
			return "", "", fmt.Errorf("Invalid file path configuration: %s", filePath)
		}
		if strings.TrimSpace(s[1]) != "" {
			fileName = s[1]
		} else {
			return "", "", fmt.Errorf("Invalid file name configuration: %s", filePath)
		}
	} else {
		fileName = filepath.Base(filePath)
	}
	return fileName, filePath, nil
}
