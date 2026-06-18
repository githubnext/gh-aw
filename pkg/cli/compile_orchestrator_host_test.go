//go:build !integration

package cli

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileWorkflowsAutoDetectsDefaultGHHost(t *testing.T) {
	t.Run("sets default host from GHE origin when GH_HOST is unset", func(t *testing.T) {
		workflow.SetDefaultGHHost("")
		t.Cleanup(func() { workflow.SetDefaultGHHost("") })
		if prev, ok := os.LookupEnv("GH_HOST"); ok {
			t.Cleanup(func() { _ = os.Setenv("GH_HOST", prev) })
		} else {
			t.Cleanup(func() { _ = os.Unsetenv("GH_HOST") })
		}
		require.NoError(t, os.Unsetenv("GH_HOST"))

		runCompileWorkflowsHostDetectionCheck(t, "https://ghes.example.com/owner/repo.git")

		assert.Equal(t, "ghes.example.com", getGHHostFromCommandEnv(workflow.ExecGH("auth", "status")))
	})

	t.Run("does not overwrite default host when GH_HOST is already set", func(t *testing.T) {
		workflow.SetDefaultGHHost("existing.default.test")
		t.Cleanup(func() { workflow.SetDefaultGHHost("") })
		t.Setenv("GH_HOST", "env.ghe.test")

		runCompileWorkflowsHostDetectionCheck(t, "https://ghes.example.com/owner/repo.git")

		// Unset GH_HOST so that ExecGH reads defaultGHHost via getDefaultGHHost rather than
		// the env var, which would shadow the in-process default in the command Environ().
		require.NoError(t, os.Unsetenv("GH_HOST"))
		assert.Equal(t, "existing.default.test", getGHHostFromCommandEnv(workflow.ExecGH("auth", "status")))
	})

	t.Run("resets default host to empty for github.com remotes", func(t *testing.T) {
		workflow.SetDefaultGHHost("existing.default.test")
		t.Cleanup(func() { workflow.SetDefaultGHHost("") })
		if prev, ok := os.LookupEnv("GH_HOST"); ok {
			t.Cleanup(func() { _ = os.Setenv("GH_HOST", prev) })
		} else {
			t.Cleanup(func() { _ = os.Unsetenv("GH_HOST") })
		}
		require.NoError(t, os.Unsetenv("GH_HOST"))

		runCompileWorkflowsHostDetectionCheck(t, "https://github.com/owner/repo.git")

		// For github.com remotes, compile resets defaultGHHost to "" so that repeated
		// calls in watch mode do not inherit a stale GHES host from a previous invocation.
		assert.Empty(t, getGHHostFromCommandEnv(workflow.ExecGH("auth", "status")))
	})

	t.Run("sets default host from GHE origin (SSH remote) when GH_HOST is unset", func(t *testing.T) {
		workflow.SetDefaultGHHost("")
		t.Cleanup(func() { workflow.SetDefaultGHHost("") })
		if prev, ok := os.LookupEnv("GH_HOST"); ok {
			t.Cleanup(func() { _ = os.Setenv("GH_HOST", prev) })
		} else {
			t.Cleanup(func() { _ = os.Unsetenv("GH_HOST") })
		}
		require.NoError(t, os.Unsetenv("GH_HOST"))

		runCompileWorkflowsHostDetectionCheck(t, "git@ghes.example.com:owner/repo.git")

		assert.Equal(t, "ghes.example.com", getGHHostFromCommandEnv(workflow.ExecGH("auth", "status")))
	})
}

func runCompileWorkflowsHostDetectionCheck(t *testing.T, remoteURL string) {
	t.Helper()

	tempDir := testutil.TempDir(t, "compile-gh-host-*")
	require.NoError(t, initTestGitRepo(tempDir))
	require.NoError(t, addOriginRemoteToTestRepo(tempDir, remoteURL))
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, ".github", "workflows"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, ".github", "workflows", "host-test.md"), []byte(`---
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Host Detection Test

Verify compile host detection.
`), 0644))

	t.Chdir(tempDir)

	_, err := CompileWorkflows(context.Background(), CompileConfig{
		MarkdownFiles: []string{"host-test"},
		NoEmit:        true,
	})
	require.NoError(t, err)
}

func addOriginRemoteToTestRepo(dir string, remoteURL string) error {
	configPath := filepath.Join(dir, ".git", "config")
	config, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	config = append(config, []byte("\n[remote \"origin\"]\n\turl = "+remoteURL+"\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n")...)
	return os.WriteFile(configPath, config, 0644)
}

func getGHHostFromCommandEnv(cmd interface{ Environ() []string }) string {
	for _, entry := range cmd.Environ() {
		if value, ok := strings.CutPrefix(entry, "GH_HOST="); ok {
			return value
		}
	}
	return ""
}
