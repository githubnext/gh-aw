//go:build !integration

package cli

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRepoGlobs(t *testing.T) {
	require.NoError(t, validateRepoGlobs([]string{"api-*", "octo/*"}))

	err := validateRepoGlobs([]string{"["})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --repos pattern")
}

func TestFilterOrgRepos(t *testing.T) {
	repos := []string{"octo/api-service", "octo/web", "octo/worker"}

	assert.Equal(t, repos, filterOrgRepos(repos, nil))
	assert.Equal(t, []string{"octo/api-service"}, filterOrgRepos(repos, []string{"api-*"}))
	assert.Equal(t, []string{"octo/web"}, filterOrgRepos(repos, []string{"octo/web"}))
}

func TestNewUpdateCommandOrgFlags(t *testing.T) {
	cmd := NewUpdateCommand(func(string) error { return nil })

	require.NotNil(t, cmd.Flags().Lookup("org"))
	require.NotNil(t, cmd.Flags().Lookup("repos"))
	require.NotNil(t, cmd.Flags().Lookup("create-issue"))
	assert.Contains(t, cmd.Example, "--org my-org")
	assert.Contains(t, cmd.Example, "--repos '*-service'")
	assert.Contains(t, cmd.Example, "--create-issue")
}

func TestRunUpdateForOrgDryRun(t *testing.T) {
	origSearch := searchOrgWorkflowReposFn
	origPreview := previewOrgRepoUpdatesFn
	origUpdate := runUpdateForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	searchOrgWorkflowReposFn = func(ctx context.Context, org string, verbose bool) ([]string, error) {
		return []string{"octo/api", "octo/web"}, nil
	}
	previewOrgRepoUpdatesFn = func(ctx context.Context, repo string, opts UpdateWorkflowsOptions, verbose bool) (orgRepoPreview, error) {
		if repo == "octo/api" {
			return orgRepoPreview{
				Repo:       repo,
				OldestEdit: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				Workflows: []orgWorkflowPreview{{
					Name:       "repo-assist",
					CurrentRef: "v1.0.0",
					LatestRef:  "v1.1.0",
				}},
			}, nil
		}
		return orgRepoPreview{Repo: repo}, nil
	}
	runUpdateForTargetRepoFn = func(ctx context.Context, targetRepo string, opts UpdateWorkflowsOptions, createPR bool, verbose bool) error {
		t.Fatalf("unexpected update call for %s", targetRepo)
		return nil
	}
	waitForOrgRateLimitFn = func(ctx context.Context, resource string, verbose bool) error { return nil }
	defer func() {
		searchOrgWorkflowReposFn = origSearch
		previewOrgRepoUpdatesFn = origPreview
		runUpdateForTargetRepoFn = origUpdate
		waitForOrgRateLimitFn = origWait
	}()

	output := captureUpdateOrgStderr(t, func() {
		err := runUpdateForOrg(context.Background(), "octo", nil, UpdateWorkflowsOptions{}, false, false, false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run preview of update pull requests")
	assert.Contains(t, output, "octo/api")
	assert.Contains(t, output, "repo-assist: v1.0.0 -> v1.1.0")
	assert.NotContains(t, output, "octo/web")
}

func TestRunUpdateForOrgCreatePRSortsOldestFirst(t *testing.T) {
	origSearch := searchOrgWorkflowReposFn
	origPreview := previewOrgRepoUpdatesFn
	origUpdate := runUpdateForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	searchOrgWorkflowReposFn = func(ctx context.Context, org string, verbose bool) ([]string, error) {
		return []string{"octo/newer", "octo/older"}, nil
	}
	previewOrgRepoUpdatesFn = func(ctx context.Context, repo string, opts UpdateWorkflowsOptions, verbose bool) (orgRepoPreview, error) {
		edited := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		if strings.HasSuffix(repo, "older") {
			edited = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		return orgRepoPreview{
			Repo:       repo,
			OldestEdit: edited,
			Workflows: []orgWorkflowPreview{{
				Name:       "repo-assist",
				CurrentRef: "v1.0.0",
				LatestRef:  "v1.1.0",
			}},
		}, nil
	}
	var processed []string
	runUpdateForTargetRepoFn = func(ctx context.Context, targetRepo string, opts UpdateWorkflowsOptions, createPR bool, verbose bool) error {
		processed = append(processed, targetRepo)
		return nil
	}
	waitForOrgRateLimitFn = func(ctx context.Context, resource string, verbose bool) error { return nil }
	defer func() {
		searchOrgWorkflowReposFn = origSearch
		previewOrgRepoUpdatesFn = origPreview
		runUpdateForTargetRepoFn = origUpdate
		waitForOrgRateLimitFn = origWait
	}()

	err := runUpdateForOrg(context.Background(), "octo", nil, UpdateWorkflowsOptions{}, true, false, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"octo/older", "octo/newer"}, processed)
}

func TestRunUpdateForOrgCreateIssueSortsOldestFirst(t *testing.T) {
	origSearch := searchOrgWorkflowReposFn
	origPreview := previewOrgRepoUpdatesFn
	origUpdate := runUpdateForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	origCreateIssue := createIssueForOrgRepoFn
	searchOrgWorkflowReposFn = func(ctx context.Context, org string, verbose bool) ([]string, error) {
		return []string{"octo/newer", "octo/older"}, nil
	}
	previewOrgRepoUpdatesFn = func(ctx context.Context, repo string, opts UpdateWorkflowsOptions, verbose bool) (orgRepoPreview, error) {
		edited := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		if strings.HasSuffix(repo, "older") {
			edited = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		}
		return orgRepoPreview{
			Repo:       repo,
			OldestEdit: edited,
			Workflows: []orgWorkflowPreview{{
				Name:       "repo-assist",
				CurrentRef: "v1.0.0",
				LatestRef:  "v1.1.0",
			}},
		}, nil
	}
	runUpdateForTargetRepoFn = func(ctx context.Context, targetRepo string, opts UpdateWorkflowsOptions, createPR bool, verbose bool) error {
		t.Fatalf("unexpected update call for %s", targetRepo)
		return nil
	}
	var issuedFor []string
	createIssueForOrgRepoFn = func(ctx context.Context, preview orgRepoPreview, verbose bool) error {
		issuedFor = append(issuedFor, preview.Repo)
		return nil
	}
	waitForOrgRateLimitFn = func(ctx context.Context, resource string, verbose bool) error { return nil }
	defer func() {
		searchOrgWorkflowReposFn = origSearch
		previewOrgRepoUpdatesFn = origPreview
		runUpdateForTargetRepoFn = origUpdate
		waitForOrgRateLimitFn = origWait
		createIssueForOrgRepoFn = origCreateIssue
	}()

	err := runUpdateForOrg(context.Background(), "octo", nil, UpdateWorkflowsOptions{}, false, true, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"octo/older", "octo/newer"}, issuedFor)
}

func captureUpdateOrgStderr(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	defer func() {
		_ = r.Close()
		os.Stderr = orig
	}()

	fn()

	require.NoError(t, w.Close())
	data, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(data)
}
