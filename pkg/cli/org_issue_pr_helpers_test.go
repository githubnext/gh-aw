//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildOrgXMLMarker(t *testing.T) {
	t.Run("with tag", func(t *testing.T) {
		marker := buildOrgXMLMarker(ghawUpgradeMarkerPrefix, "v1.2.3")
		assert.Equal(t, "<!-- gh-aw-upgrade: v1.2.3 -->", marker, "marker should include the provided release tag")
	})

	t.Run("without tag uses latest placeholder", func(t *testing.T) {
		marker := buildOrgXMLMarker(ghawUpgradeMarkerPrefix, "")
		assert.Equal(t, "<!-- gh-aw-upgrade: latest -->", marker, "empty tags should still produce a searchable marker")
	})

	t.Run("update prefix", func(t *testing.T) {
		marker := buildOrgXMLMarker(ghawUpdateMarkerPrefix, "v2.0.0")
		assert.Equal(t, "<!-- gh-aw-update: v2.0.0 -->", marker, "update markers should use the update prefix")
	})
}

func TestMarkerPrefixesAreDistinct(t *testing.T) {
	assert.NotEqual(t, ghawUpgradeMarkerPrefix, ghawUpdateMarkerPrefix, "upgrade and update marker prefixes must stay distinct")
}

func TestGetGhawReleaseInfoCachesSuccessfulLookup(t *testing.T) {
	originalLookup := getLatestOrgReleaseFunc
	resetGhawReleaseInfoCacheForTest()
	t.Cleanup(func() {
		getLatestOrgReleaseFunc = originalLookup
		resetGhawReleaseInfoCacheForTest()
	})

	lookups := 0
	getLatestOrgReleaseFunc = func(includePrereleases bool) (string, error) {
		lookups++
		return "v1.2.3", nil
	}

	tag, releaseURL := getGhawReleaseInfo()
	require.Equal(t, "v1.2.3", tag)
	require.Equal(t, "https://github.com/github/gh-aw/releases/tag/v1.2.3", releaseURL)

	tag, releaseURL = getGhawReleaseInfo()
	require.Equal(t, "v1.2.3", tag)
	require.Equal(t, "https://github.com/github/gh-aw/releases/tag/v1.2.3", releaseURL)
	assert.Equal(t, 1, lookups, "successful release lookups should be cached")
}

func TestGetGhawReleaseInfoRetriesAfterFailure(t *testing.T) {
	originalLookup := getLatestOrgReleaseFunc
	resetGhawReleaseInfoCacheForTest()
	t.Cleanup(func() {
		getLatestOrgReleaseFunc = originalLookup
		resetGhawReleaseInfoCacheForTest()
	})

	lookups := 0
	getLatestOrgReleaseFunc = func(includePrereleases bool) (string, error) {
		lookups++
		if lookups == 1 {
			return "", errors.New("temporary failure")
		}
		return "v1.2.4", nil
	}

	tag, releaseURL := getGhawReleaseInfo()
	require.Empty(t, tag)
	require.Empty(t, releaseURL)

	tag, releaseURL = getGhawReleaseInfo()
	require.Equal(t, "v1.2.4", tag)
	require.Equal(t, "https://github.com/github/gh-aw/releases/tag/v1.2.4", releaseURL)
	assert.Equal(t, 2, lookups, "failed release lookups should be retried")
}

func resetGhawReleaseInfoCacheForTest() {
	ghawReleaseTagCache.Reset()
}

func TestCloseExistingOrgIssuesByMarkerSkipsPRsAndPaginates(t *testing.T) {
	fakeBinDir := t.TempDir()
	argsLogPath := filepath.Join(fakeBinDir, "gh-args.log")
	page1Path := filepath.Join(fakeBinDir, "page1.json")
	page2Path := filepath.Join(fakeBinDir, "page2.json")
	fakeGH := filepath.Join(fakeBinDir, "gh")
	page1Items := make([]orgListItem, 0, 100)
	page1Items = append(page1Items,
		orgListItem{Number: 1, Body: "<!-- gh-aw-update: v1.2.3 -->"},
		orgListItem{Number: 2, Body: "no marker"},
		orgListItem{Number: 3, Body: "<!-- gh-aw-update: v1.2.3 -->", PullRequest: &orgPullRequest{}},
	)
	for i := 4; i <= 100; i++ {
		page1Items = append(page1Items, orgListItem{Number: i, Body: "no marker"})
	}
	page1JSON, err := json.Marshal(page1Items)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(page1Path, page1JSON, 0o644))
	require.NoError(t, os.WriteFile(page2Path, []byte(`[{"number":101,"body":"<!-- gh-aw-update: v1.2.3 -->"}]`), 0o644))

	fakeGHScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"" + argsLogPath + "\"\n" +
		"case \"$*\" in\n" +
		"  *\"/repos/octo/repo/issues?state=open&per_page=100&page=1\"*)\n" +
		"    cat \"" + page1Path + "\"\n" +
		"    ;;\n" +
		"  *\"/repos/octo/repo/issues?state=open&per_page=100&page=2\"*)\n" +
		"    cat \"" + page2Path + "\"\n" +
		"    ;;\n" +
		"  *\"/repos/octo/repo/issues/1\"*|*\"/repos/octo/repo/issues/101\"*)\n" +
		"    printf '%s' '{}'\n" +
		"    ;;\n" +
		"  *)\n" +
		"    printf '%s\\n' \"unexpected gh args: $*\" >&2\n" +
		"    exit 1\n" +
		"    ;;\n" +
		"esac\n"
	require.NoError(t, os.WriteFile(fakeGH, []byte(fakeGHScript), 0o755))

	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	closeExistingOrgIssuesByMarker(context.Background(), "octo/repo", ghawUpdateMarkerPrefix, false)

	argsLog, err := os.ReadFile(argsLogPath)
	require.NoError(t, err)
	logText := string(argsLog)
	assert.Contains(t, logText, "api --hostname github.com /repos/octo/repo/issues?state=open&per_page=100&page=1", "first page should be queried")
	assert.Contains(t, logText, "api --hostname github.com /repos/octo/repo/issues?state=open&per_page=100&page=2", "second page should be queried")
	assert.Contains(t, logText, "api --hostname github.com --method PATCH /repos/octo/repo/issues/1 -f state=closed -f state_reason=not_planned", "matching issues should be closed")
	assert.Contains(t, logText, "api --hostname github.com --method PATCH /repos/octo/repo/issues/101 -f state=closed -f state_reason=not_planned", "matching issues on later pages should also be closed")
	assert.NotContains(t, logText, "/repos/octo/repo/issues/3", "items flagged as pull requests must not be closed via the issues endpoint")
}

func TestCreateOrgIssueRetriesOnlyForMissingLabel(t *testing.T) {
	fakeBinDir := t.TempDir()
	argsLogPath := filepath.Join(fakeBinDir, "gh-args.log")
	fakeGH := filepath.Join(fakeBinDir, "gh")
	fakeGHScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"" + argsLogPath + "\"\n" +
		"case \"$*\" in\n" +
		"  *\"labels[]=agentic-workflows\"*)\n" +
		"    printf '%s\\n' 'HTTP 422: Validation Failed (label does not exist)' >&2\n" +
		"    exit 1\n" +
		"    ;;\n" +
		"  *)\n" +
		"    printf '%s' '{}'\n" +
		"    ;;\n" +
		"esac\n"
	require.NoError(t, os.WriteFile(fakeGH, []byte(fakeGHScript), 0o755))

	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := createOrgIssue(context.Background(), "octo/repo", "title", "body", agenticWorkflowsLabel)
	require.NoError(t, err)

	argsLog, err := os.ReadFile(argsLogPath)
	require.NoError(t, err)
	assert.Contains(t, string(argsLog), "labels[]=agentic-workflows", "the initial attempt should include the label")
	assert.Equal(t, 2, strings.Count(string(argsLog), "/repos/octo/repo/issues"), "a label validation error should trigger exactly one retry")
}

func TestCreateOrgIssueDoesNotRetryNonLabelErrors(t *testing.T) {
	fakeBinDir := t.TempDir()
	argsLogPath := filepath.Join(fakeBinDir, "gh-args.log")
	fakeGH := filepath.Join(fakeBinDir, "gh")
	fakeGHScript := "#!/bin/sh\n" +
		"printf '%s\\n' \"$*\" >> \"" + argsLogPath + "\"\n" +
		"printf '%s\\n' 'HTTP 500: Internal Server Error' >&2\n" +
		"exit 1\n"
	require.NoError(t, os.WriteFile(fakeGH, []byte(fakeGHScript), 0o755))

	t.Setenv("PATH", fakeBinDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := createOrgIssue(context.Background(), "octo/repo", "title", "body", agenticWorkflowsLabel)
	require.Error(t, err)

	argsLog, readErr := os.ReadFile(argsLogPath)
	require.NoError(t, readErr)
	assert.Equal(t, 1, strings.Count(string(argsLog), "/repos/octo/repo/issues"), "non-label failures must not be retried")
}
