//go:build !integration

package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateWorkflowMaintenancePRLifecycle(t *testing.T) {
	originalCreate := createMaintenancePRWithChanges
	t.Cleanup(func() { createMaintenancePRWithChanges = originalCreate })

	for _, operation := range []string{"update", "upgrade"} {
		for _, tc := range []struct {
			name       string
			closeOlder bool
			url        string
			err        error
			wantClose  bool
		}{
			{name: "default preserves older PRs", url: "https://ghe.example.com/octo/repo/pull/42"},
			{name: "successful replacement", closeOlder: true, url: "https://ghe.example.com/octo/repo/pull/42", wantClose: true},
			{name: "creation fails", closeOlder: true, err: assert.AnError},
			{name: "no replacement URL", closeOlder: true},
			{name: "auto-merge fails after creation", closeOlder: true, url: "https://ghe.example.com/octo/repo/pull/42", err: assert.AnError},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				dir := t.TempDir()
				logPath := filepath.Join(dir, "calls.log")
				markerPrefix := ghawUpdateMarkerPrefix
				if operation == "upgrade" {
					markerPrefix = ghawUpgradeMarkerPrefix
				}
				script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
case "$*" in
  *"--method PATCH"*) printf '{}' ;;
  *) printf '%%s' '[{"number":1,"body":"%s v1.0.0 -->"}]' ;;
esac
`, logPath, markerPrefix)
				require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755))
				t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
				createCalls := 0
				createMaintenancePRWithChanges = func(_ context.Context, branchPrefix, commitMessage, title, body string, verbose, autoMerge bool) (string, error) {
					createCalls++
					assert.True(t, verbose)
					assert.True(t, autoMerge)
					assert.Contains(t, body, "Summary\n\nCustom message")
					assert.Equal(t, 1, strings.Count(body, markerPrefix))
					assert.True(t, strings.HasSuffix(body, buildOrgXMLMarker(markerPrefix, normalizeDisplayVersion(GetVersion()))))
					if operation == "update" {
						assert.Equal(t, "update-workflows", branchPrefix)
						assert.Equal(t, "chore: update workflows", commitMessage)
						assert.Equal(t, "Update workflows from source", title)
						assert.NotContains(t, body, ghawUpgradeMarkerPrefix)
					} else {
						assert.Equal(t, "upgrade-agentic-workflows", branchPrefix)
						assert.Equal(t, "chore: upgrade agentic workflows", commitMessage)
						assert.Equal(t, "Upgrade agentic workflows", title)
						assert.NotContains(t, body, ghawUpdateMarkerPrefix)
					}
					_, err := os.Stat(logPath)
					assert.True(t, os.IsNotExist(err), "cleanup must never run before replacement creation succeeds")
					return tc.url, tc.err
				}

				prURL, err := createWorkflowMaintenancePR(context.Background(), operation, "Summary\n\nCustom message", true, true, tc.closeOlder)
				assert.Equal(t, tc.url, prURL)
				require.ErrorIs(t, err, tc.err)
				assert.Equal(t, 1, createCalls)
				log, readErr := os.ReadFile(logPath)
				if tc.wantClose {
					require.NoError(t, readErr)
					assert.Contains(t, string(log), "api --hostname ghe.example.com --method PATCH /repos/octo/repo/pulls/1 -f state=closed")
				} else {
					assert.True(t, os.IsNotExist(readErr), "default mode and unsuccessful replacements must not even query existing PRs")
				}
			})
		}
	}
}

func TestCreateWorkflowMaintenancePRNoChangesPreservesOlderPRs(t *testing.T) {
	repoDir := initPRTransferRepo(t)
	t.Chdir(repoDir)
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte(fmt.Sprintf("#!/bin/sh\nprintf called >> %q\n", logPath)), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	prURL, err := createWorkflowMaintenancePR(context.Background(), "update", "Summary", false, false, true)

	require.ErrorContains(t, err, "failed to commit")
	assert.Empty(t, prURL)
	assert.Equal(t, "main", strings.TrimSpace(gitOutput(t, repoDir, "branch", "--show-current")))
	_, err = os.Stat(logPath)
	assert.True(t, os.IsNotExist(err), "an unchanged repository must not create or close any PR")
}

func TestCloseExistingOrgPRsByMarkerRejectsInvalidReplacement(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte(fmt.Sprintf("#!/bin/sh\nprintf called >> %q\n", logPath)), 0o755))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, prURL := range []string{
		"", "not a URL", "/octo/repo/pull/42",
		"http://github.com/octo/repo/pull/42",
		"https://user@github.com/octo/repo/pull/42",
		"https://github.com/octo/repo/issues/42",
		"https://github.com/octo/repo/pull/0",
		"https://github.com/octo/repo/pull/-1",
		"https://github.com/octo/repo/pull/not-a-number",
		"https://github.com/octo/repo/pull/99999999999999999999",
		"https://%zz/octo/repo/pull/42",
	} {
		t.Run(prURL, func(t *testing.T) {
			closeExistingOrgPRsByMarker(context.Background(), prURL, ghawUpdateMarkerPrefix, false)
		})
	}
	_, err := os.Stat(logPath)
	assert.True(t, os.IsNotExist(err), "unsafe or unidentifiable replacements must not trigger cleanup")
}
