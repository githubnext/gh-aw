//go:build !integration

package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDeployForOrgEmptyOrg(t *testing.T) {
	err := runDeployForOrg(context.Background(), " ", nil, []string{"githubnext/agentics/ci-doctor"}, AddOptions{}, time.Hour, false, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "--org cannot be empty")
}

func TestRunDeployForOrgInvalidRepoGlob(t *testing.T) {
	err := runDeployForOrg(context.Background(), "octo", []string{"["}, []string{"githubnext/agentics/ci-doctor"}, AddOptions{}, time.Hour, false, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid --repos pattern")
}

func TestRunDeployForOrgCreatePRRequiresYesInCI(t *testing.T) {
	origIsCI := isRunningInCIFn
	isRunningInCIFn = func() bool { return true }
	defer func() { isRunningInCIFn = origIsCI }()

	err := runDeployForOrg(context.Background(), "octo", nil, []string{"githubnext/agentics/ci-doctor"}, AddOptions{}, time.Hour, false, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "--yes")
}

func TestRunDeployForOrgAppliesAcrossRepositories(t *testing.T) {
	origSearch := searchOrgDeployReposFn
	origRunDeploy := runDeployForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	searchOrgDeployReposFn = func(context.Context, string, bool) ([]string, error) {
		return []string{"octo/api", "octo/web"}, nil
	}
	var deployed []string
	runDeployForTargetRepoFn = func(_ context.Context, targetRepo string, _ []string, _ AddOptions, _ time.Duration) error {
		deployed = append(deployed, targetRepo)
		return nil
	}
	waitForOrgRateLimitFn = func(context.Context, string, bool) error { return nil }
	defer func() {
		searchOrgDeployReposFn = origSearch
		runDeployForTargetRepoFn = origRunDeploy
		waitForOrgRateLimitFn = origWait
	}()

	err := runDeployForOrg(context.Background(), "octo", nil, []string{"githubnext/agentics/ci-doctor"}, AddOptions{}, time.Hour, true, false)
	require.NoError(t, err)
	assert.Equal(t, []string{"octo/api", "octo/web"}, deployed)
}

func TestRunDeployForOrgReportsFailureWhenAllReposFail(t *testing.T) {
	origSearch := searchOrgDeployReposFn
	origRunDeploy := runDeployForTargetRepoFn
	origWait := waitForOrgRateLimitFn
	searchOrgDeployReposFn = func(context.Context, string, bool) ([]string, error) {
		return []string{"octo/api", "octo/web"}, nil
	}
	runDeployForTargetRepoFn = func(_ context.Context, targetRepo string, _ []string, _ AddOptions, _ time.Duration) error {
		return errors.New("boom")
	}
	waitForOrgRateLimitFn = func(context.Context, string, bool) error { return nil }
	defer func() {
		searchOrgDeployReposFn = origSearch
		runDeployForTargetRepoFn = origRunDeploy
		waitForOrgRateLimitFn = origWait
	}()

	err := runDeployForOrg(context.Background(), "octo", nil, []string{"githubnext/agentics/ci-doctor"}, AddOptions{}, time.Hour, true, false)
	require.Error(t, err)
	require.ErrorContains(t, err, "failed to deploy workflows to any repository")
}
