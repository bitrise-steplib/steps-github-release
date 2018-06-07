package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseRepo(t *testing.T) {
	t.Log("Parses: https://hostname/owner/repository.git")
	{
		host, owner, name := parseRepo("https://github.com/godrei/steps-github-release.git")
		require.Equal(t, host, "github.com")
		require.Equal(t, owner, "godrei")
		require.Equal(t, name, "steps-github-release")
	}

	t.Log("Parses: git@hostname:owner/repository.git")
	{
		host, owner, name := parseRepo("git@github.com:godrei/steps-github-release.git")
		require.Equal(t, host, "github.com")
		require.Equal(t, owner, "godrei")
		require.Equal(t, name, "steps-github-release")
	}

	t.Log("Parses: ssh://git@hostname:port/owner/repository.git")
	{
		host, owner, name := parseRepo("ssh://git@github.com:port/godrei/steps-github-release.git")
		require.Equal(t, host, "github.com")
		require.Equal(t, owner, "godrei")
		require.Equal(t, name, "steps-github-release")
	}
}
