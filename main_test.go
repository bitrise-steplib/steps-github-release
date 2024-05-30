package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/bitrise-io/go-utils/log"
	"github.com/google/go-github/v57/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRepo(t *testing.T) {
	t.Log("Parses: https://hostname/owner/repository.git")
	{
		host, owner, name := parseRepo("https://github.com/bitrise/steps-github-release.git")
		require.Equal(t, host, "github.com")
		require.Equal(t, owner, "bitrise")
		require.Equal(t, name, "steps-github-release")
	}

	t.Log("Parses: git@hostname:owner/repository.git")
	{
		host, owner, name := parseRepo("git@github.com:bitrise/steps-github-release.git")
		require.Equal(t, host, "github.com")
		require.Equal(t, owner, "bitrise")
		require.Equal(t, name, "steps-github-release")
	}

	t.Log("Parses: ssh://git@hostname:port/owner/repository.git")
	{
		host, owner, name := parseRepo("ssh://git@github.com:port/bitrise/steps-github-release.git")
		require.Equal(t, host, "github.com")
		require.Equal(t, owner, "bitrise")
		require.Equal(t, name, "steps-github-release")
	}
}

func TestRetryUpload(t *testing.T) {
	t.Log("Tests retry should fail if no connection")
	{
		var buf bytes.Buffer
		writer := bufio.NewWriter(&buf)
		log.SetOutWriter(writer)
		err := uploadFileWithRetry(GetUploader(mockUploadAsset, 3, 1), "", "", nil, nil, "", "", 0)
		assert.Error(t, err, "Could not connect")
		if err := writer.Flush(); err != nil {
			failf("Could not flush buffer: %s", err)
		}
		expected := buf.String()
		buf.Reset()
		log.Warnf("1. attempt failed: failed to upload file (): Could not connect")
		log.Warnf("2. attempt failed: failed to upload file (): Could not connect")
		log.Warnf("3. attempt failed: failed to upload file (): Could not connect")
		if err := writer.Flush(); err != nil {
			failf("Could not flush buffer: %s", err)
		}
		require.Equal(t, expected, buf.String())
	}
}

func mockUploadAsset(filePath string, fileName string, fi *os.File, client *github.Client, owner string, repo string, id int64) (*github.ReleaseAsset, *github.Response, error) {
	return nil, nil, fmt.Errorf("Could not connect")
}
