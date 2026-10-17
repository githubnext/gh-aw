//go:build !integration

package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
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

func TestHasCompleteXMLMarker(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		body string
		want bool
	}{
		{name: "version marker", body: "<!-- gh-aw-update: v1.2.3 -->", want: true},
		{name: "placeholder marker", body: "<!-- gh-aw-update: latest -->", want: true},
		{name: "surrounding text", body: "Summary\n<!-- gh-aw-update: v1.2.3 -->\nFooter", want: true},
		{name: "preceding unrelated comment", body: "<!-- unrelated -->\n<!-- gh-aw-update: v1.2.3 -->", want: true},
		{name: "unmarked", body: "Update workflows"},
		{name: "opposite command", body: "<!-- gh-aw-upgrade: v1.2.3 -->"},
		{name: "stray prefix", body: "Please add <!-- gh-aw-update: to the body"},
		{name: "unterminated marker", body: "<!-- gh-aw-update: v1.2.3"},
		{name: "empty marker", body: "<!-- gh-aw-update: -->"},
		{name: "nested comment", body: "<!-- example: <!-- gh-aw-update: v1.2.3 --> -->"},
		{name: "malformed comment", body: "<!-- gh-aw-update: v1.2.3 -- ignored -->"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, hasCompleteXMLMarker(tc.body, ghawUpdateMarkerPrefix))
		})
	}
}

func TestGetGhawReleaseInfoCachesSuccessfulLookup(t *testing.T) {
	originalLookup := getLatestOrgReleaseFunc
	resetGhawReleaseInfoCacheForTest()
	t.Cleanup(func() {
		getLatestOrgReleaseFunc = originalLookup
		resetGhawReleaseInfoCacheForTest()
	})

	lookups := 0
	getLatestOrgReleaseFunc = func(_ context.Context, includePrereleases bool) (string, error) {
		lookups++
		return "v1.2.3", nil
	}

	tag, releaseURL := getGhawReleaseInfo(context.Background())
	require.Equal(t, "v1.2.3", tag)
	require.Equal(t, "https://github.com/github/gh-aw/releases/tag/v1.2.3", releaseURL)

	tag, releaseURL = getGhawReleaseInfo(context.Background())
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
	getLatestOrgReleaseFunc = func(_ context.Context, includePrereleases bool) (string, error) {
		lookups++
		if lookups == 1 {
			return "", errors.New("temporary failure")
		}
		return "v1.2.4", nil
	}

	tag, releaseURL := getGhawReleaseInfo(context.Background())
	require.Empty(t, tag)
	require.Empty(t, releaseURL)

	tag, releaseURL = getGhawReleaseInfo(context.Background())
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

	// Isolate getHostFromOriginRemote() from the real git remote by running inside
	// a temporary repository whose origin points to github.com.
	fakeGitDir := t.TempDir()
	require.NoError(t, exec.Command("git", "-C", fakeGitDir, "init").Run())
	require.NoError(t, exec.Command("git", "-C", fakeGitDir, "remote", "add", "origin", "https://github.com/octo/repo.git").Run())
	t.Chdir(fakeGitDir)

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

func TestCloseExistingOrgPRsByMarkerFiltersAndPaginates(t *testing.T) {
	for _, operation := range []string{"update", "upgrade"} {
		t.Run(operation, func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "gh-args.log")
			marker, otherMarker := ghawUpdateMarkerPrefix, ghawUpgradeMarkerPrefix
			if operation == "upgrade" {
				marker, otherMarker = otherMarker, marker
			}
			page1 := []orgListItem{
				{Number: 1, Body: buildOrgXMLMarker(marker, "v1.0.0")},
				{Number: 200, Body: buildOrgXMLMarker(marker, "v2.0.0")},
				{Number: 201, Body: buildOrgXMLMarker(marker, "v3.0.0")},
				{Number: 2, Body: buildOrgXMLMarker(otherMarker, "v1.0.0")},
				{Number: 3, Body: "unrelated PR"},
				{Number: 0, Body: buildOrgXMLMarker(marker, "v1.0.0")},
				{Number: 98, Body: marker + " v1.0.0"},
				{Number: 99, Body: marker + " -->"},
			}
			for i := 4; len(page1) < 100; i++ {
				page1 = append(page1, orgListItem{Number: i, Body: "no marker"})
			}
			page1JSON, err := json.Marshal(page1)
			require.NoError(t, err)
			page2JSON, err := json.Marshal([]orgListItem{{Number: 101, Body: buildOrgXMLMarker(marker, "v1.0.0")}})
			require.NoError(t, err)
			page1Path, page2Path := filepath.Join(dir, "page1.json"), filepath.Join(dir, "page2.json")
			require.NoError(t, os.WriteFile(page1Path, page1JSON, 0o644))
			require.NoError(t, os.WriteFile(page2Path, page2JSON, 0o644))
			// One failed close must not prevent closing other obsolete PRs.
			script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
case "$*" in
  *"/pulls?state=open&per_page=100&page=1"*) cat %q ;;
  *"/pulls?state=open&per_page=100&page=2"*) cat %q ;;
  *"/pulls/1 "*) exit 1 ;;
  *"/pulls/101 "*) printf '{}' ;;
  *) exit 1 ;;
esac
`, logPath, page1Path, page2Path)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

			closeExistingOrgPRsByMarker(context.Background(), "https://ghe.example.com/octo/repo/pull/200", marker, false)

			log, err := os.ReadFile(logPath)
			require.NoError(t, err)
			lines := strings.Split(strings.TrimSpace(string(log)), "\n")
			require.Len(t, lines, 4)
			assert.Equal(t, "api --hostname ghe.example.com /repos/octo/repo/pulls?state=open&per_page=100&page=1", lines[0])
			assert.Equal(t, "api --hostname ghe.example.com /repos/octo/repo/pulls?state=open&per_page=100&page=2", lines[1])
			assert.Equal(t, "api --hostname ghe.example.com --method PATCH /repos/octo/repo/pulls/1 -f state=closed", lines[2])
			assert.Equal(t, "api --hostname ghe.example.com --method PATCH /repos/octo/repo/pulls/101 -f state=closed", lines[3])
		})
	}
}

func TestCloseExistingOrgPRsByMarkerListErrorsAreNonFatal(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response string
	}{
		{name: "API failure", response: "exit 1"},
		{name: "invalid JSON", response: "printf 'not json'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "gh-args.log")
			script := fmt.Sprintf("#!/bin/sh\nprintf '%%s\\n' \"$*\" >> %q\n%s\n", logPath, tc.response)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "gh"), []byte(script), 0o755))
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

			closeExistingOrgPRsByMarker(context.Background(), "https://github.com/octo/repo/pull/42", ghawUpdateMarkerPrefix, false)

			log, err := os.ReadFile(logPath)
			require.NoError(t, err)
			assert.NotContains(t, string(log), "PATCH")
		})
	}
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
