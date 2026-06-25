//go:build !integration

package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpgradeCommandOrgFlags(t *testing.T) {
	cmd := NewUpgradeCommand()

	require.NotNil(t, cmd.Flags().Lookup("org"))
	require.NotNil(t, cmd.Flags().Lookup("repos"))
	require.NotNil(t, cmd.Flags().Lookup("create-issue"))
	assert.Contains(t, cmd.Example, "--org my-org")
	assert.Contains(t, cmd.Example, "--repos '*-service'")
	assert.Contains(t, cmd.Example, "--create-issue")
}

func TestRunUpgradeForOrgEmptyOrg(t *testing.T) {
	err := runUpgradeForOrg(context.Background(), "  ", nil, upgradeOptions{ctx: context.Background()}, false, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--org cannot be empty")
}

func TestRunUpgradeForOrgInvalidRepoGlob(t *testing.T) {
	err := runUpgradeForOrg(context.Background(), "octo", []string{"["}, upgradeOptions{ctx: context.Background()}, false, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --repos pattern")
}

func TestRunUpgradeForOrgNoReposFound(t *testing.T) {
	origSearch := searchOrgAnyWorkflowReposFn
	searchOrgAnyWorkflowReposFn = func(ctx context.Context, org string, verbose bool) ([]string, error) {
		return nil, nil
	}
	defer func() { searchOrgAnyWorkflowReposFn = origSearch }()

	output := captureUpgradeOrgStderr(t, func() {
		err := runUpgradeForOrg(context.Background(), "octo", nil, upgradeOptions{ctx: context.Background()}, false, false, false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No repositories found with agentic workflows")
}

func TestRunUpgradeForOrgNoReposMatchFilter(t *testing.T) {
	origSearch := searchOrgAnyWorkflowReposFn
	searchOrgAnyWorkflowReposFn = func(ctx context.Context, org string, verbose bool) ([]string, error) {
		return []string{"octo/api"}, nil
	}
	defer func() { searchOrgAnyWorkflowReposFn = origSearch }()

	output := captureUpgradeOrgStderr(t, func() {
		err := runUpgradeForOrg(context.Background(), "octo", []string{"nomatch-*"}, upgradeOptions{ctx: context.Background()}, false, false, false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "No repositories matched the requested --repos filters")
}

func TestRunUpgradeForOrgDryRun(t *testing.T) {
	origSearch := searchOrgAnyWorkflowReposFn
	origUpgrade := runUpgradeForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	searchOrgAnyWorkflowReposFn = func(ctx context.Context, org string, verbose bool) ([]string, error) {
		return []string{"octo/api", "octo/web"}, nil
	}
	runUpgradeForTargetRepoFn = func(ctx context.Context, repo string, opts upgradeOptions, verbose bool) error {
		t.Fatalf("unexpected upgrade call for %s", repo)
		return nil
	}
	waitForOrgRateLimitFn = func(ctx context.Context, resource string, verbose bool) error { return nil }
	defer func() {
		searchOrgAnyWorkflowReposFn = origSearch
		runUpgradeForTargetRepoFn = origUpgrade
		waitForOrgRateLimitFn = origWait
	}()

	output := captureUpgradeOrgStderr(t, func() {
		err := runUpgradeForOrg(context.Background(), "octo", nil, upgradeOptions{ctx: context.Background()}, false, false, false)
		require.NoError(t, err)
	})

	assert.Contains(t, output, "Dry-run preview of upgrade pull requests")
	assert.Contains(t, output, "octo/api")
	assert.Contains(t, output, "octo/web")
}

func TestRunUpgradeForOrgCreatePR(t *testing.T) {
	origSearch := searchOrgAnyWorkflowReposFn
	origUpgrade := runUpgradeForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	searchOrgAnyWorkflowReposFn = func(ctx context.Context, org string, verbose bool) ([]string, error) {
		return []string{"octo/api", "octo/web"}, nil
	}
	var upgraded []string
	runUpgradeForTargetRepoFn = func(ctx context.Context, repo string, opts upgradeOptions, verbose bool) error {
		upgraded = append(upgraded, repo)
		return nil
	}
	waitForOrgRateLimitFn = func(ctx context.Context, resource string, verbose bool) error { return nil }
	defer func() {
		searchOrgAnyWorkflowReposFn = origSearch
		runUpgradeForTargetRepoFn = origUpgrade
		waitForOrgRateLimitFn = origWait
	}()

	err := runUpgradeForOrg(context.Background(), "octo", nil, upgradeOptions{ctx: context.Background()}, true, false, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"octo/api", "octo/web"}, upgraded)
}

func TestRunUpgradeForOrgRepoFilter(t *testing.T) {
	origSearch := searchOrgAnyWorkflowReposFn
	origUpgrade := runUpgradeForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	searchOrgAnyWorkflowReposFn = func(ctx context.Context, org string, verbose bool) ([]string, error) {
		return []string{"octo/api-service", "octo/web", "octo/worker-service"}, nil
	}
	var upgraded []string
	runUpgradeForTargetRepoFn = func(ctx context.Context, repo string, opts upgradeOptions, verbose bool) error {
		upgraded = append(upgraded, repo)
		return nil
	}
	waitForOrgRateLimitFn = func(ctx context.Context, resource string, verbose bool) error { return nil }
	defer func() {
		searchOrgAnyWorkflowReposFn = origSearch
		runUpgradeForTargetRepoFn = origUpgrade
		waitForOrgRateLimitFn = origWait
	}()

	err := runUpgradeForOrg(context.Background(), "octo", []string{"*-service"}, upgradeOptions{ctx: context.Background()}, true, false, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"octo/api-service", "octo/worker-service"}, upgraded)
}

func TestRunUpgradeForOrgCreateIssue(t *testing.T) {
	origSearch := searchOrgAnyWorkflowReposFn
	origUpgrade := runUpgradeForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	origIssue := createIssueForUpgradeOrgRepoFn
	searchOrgAnyWorkflowReposFn = func(ctx context.Context, org string, verbose bool) ([]string, error) {
		return []string{"octo/api", "octo/web"}, nil
	}
	runUpgradeForTargetRepoFn = func(ctx context.Context, repo string, opts upgradeOptions, verbose bool) error {
		t.Fatalf("unexpected upgrade call for %s", repo)
		return nil
	}
	var issuedRepos []string
	createIssueForUpgradeOrgRepoFn = func(ctx context.Context, repo string, verbose bool) error {
		issuedRepos = append(issuedRepos, repo)
		return nil
	}
	waitForOrgRateLimitFn = func(ctx context.Context, resource string, verbose bool) error { return nil }
	defer func() {
		searchOrgAnyWorkflowReposFn = origSearch
		runUpgradeForTargetRepoFn = origUpgrade
		waitForOrgRateLimitFn = origWait
		createIssueForUpgradeOrgRepoFn = origIssue
	}()

	err := runUpgradeForOrg(context.Background(), "octo", nil, upgradeOptions{ctx: context.Background()}, false, true, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"octo/api", "octo/web"}, issuedRepos)
}

func TestRunUpgradeCommandCreateIssueRequiresOrg(t *testing.T) {
	cmd := NewUpgradeCommand()
	cmd.SetArgs([]string{"--create-issue"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--create-issue requires --org")
}

func TestRunUpgradeCommandCreateIssueAndPRMutuallyExclusive(t *testing.T) {
	cmd := NewUpgradeCommand()
	cmd.SetArgs([]string{"--org", "octo", "--create-issue", "--create-pull-request"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify both --create-pull-request and --create-issue")
}

func TestRunUpgradeCommandReposRequiresOrg(t *testing.T) {
	cmd := NewUpgradeCommand()
	cmd.SetArgs([]string{"--repos", "*-svc"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--repos requires --org")
}

func TestRunUpgradeForOrgStopsOnFirstError(t *testing.T) {
	origSearch := searchOrgAnyWorkflowReposFn
	origUpgrade := runUpgradeForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	searchOrgAnyWorkflowReposFn = func(_ context.Context, _ string, _ bool) ([]string, error) {
		return []string{"octo/api", "octo/web"}, nil
	}
	boom := errors.New("upgrade failed")
	var called []string
	runUpgradeForTargetRepoFn = func(_ context.Context, repo string, _ upgradeOptions, _ bool) error {
		called = append(called, repo)
		return boom
	}
	waitForOrgRateLimitFn = func(_ context.Context, _ string, _ bool) error { return nil }
	defer func() {
		searchOrgAnyWorkflowReposFn = origSearch
		runUpgradeForTargetRepoFn = origUpgrade
		waitForOrgRateLimitFn = origWait
	}()

	err := runUpgradeForOrg(context.Background(), "octo", nil, upgradeOptions{ctx: context.Background()}, true, false, false)
	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{"octo/api"}, called, "should stop after first failure")
}

func captureUpgradeOrgStderr(t *testing.T, fn func()) string {
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
