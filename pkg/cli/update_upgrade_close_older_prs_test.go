//go:build !integration

package cli

import (
	"context"
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateUpgradeCloseOlderPRsFlag(t *testing.T) {
	t.Parallel()
	for _, newCommand := range []func(func(string) error) *cobra.Command{NewUpdateCommand, NewUpgradeCommand} {
		cmd := newCommand(func(string) error { return nil })
		t.Run(cmd.Name(), func(t *testing.T) {
			flag := cmd.Flags().Lookup("close-older-prs")
			require.NotNil(t, flag)
			assert.Equal(t, "false", flag.DefValue)
			assert.Contains(t, flag.Usage, "--pr or --create-pull-request")
			assert.Contains(t, cmd.Example, "--pr --close-older-prs")
		})
	}
}

func TestUpdateUpgradeCloseOlderPRsRequiresPREarly(t *testing.T) {
	t.Parallel()
	for _, newCommand := range []func(func(string) error) *cobra.Command{NewUpdateCommand, NewUpgradeCommand} {
		name := newCommand(func(string) error { return nil }).Name()
		t.Run(name, func(t *testing.T) {
			scenarios := [][]string{
				{"--close-older-prs"},
				{"--close-older-prs", "--repo", "owner/repo"},
				{"--close-older-prs", "--org", "my-org"},
				{"--close-older-prs", "--org", "my-org", "--create-issue"},
				{"--close-older-prs", "--pr=false"},
				{"--close-older-prs", "--create-pull-request=false"},
			}
			if name == "upgrade" {
				scenarios = append(scenarios, []string{"--close-older-prs", "--audit"})
			}
			for _, args := range scenarios {
				t.Run(fmt.Sprint(args), func(t *testing.T) {
					validated := false
					cmd := newCommand(func(string) error {
						validated = true
						return assert.AnError
					})
					cmd.SetArgs(args)
					require.EqualError(t, cmd.Execute(), "--close-older-prs requires --pr or --create-pull-request")
					assert.False(t, validated, "validation must precede engine checks and side effects")
				})
			}
		})
	}
}

func TestUpdateUpgradeCloseOlderPRsAcceptsPRAliases(t *testing.T) {
	t.Parallel()
	for _, newCommand := range []func(func(string) error) *cobra.Command{NewUpdateCommand, NewUpgradeCommand} {
		name := newCommand(func(string) error { return nil }).Name()
		t.Run(name, func(t *testing.T) {
			for _, prFlag := range []string{"--pr", "--create-pull-request"} {
				t.Run(prFlag, func(t *testing.T) {
					cmd := newCommand(func(string) error { return assert.AnError })
					cmd.SetArgs([]string{prFlag, "--close-older-prs", "--auto-merge"})
					require.ErrorIs(t, cmd.Execute(), assert.AnError)
				})
			}
		})
	}
}

func TestUpdateUpgradeCloseOlderPRsDispatch(t *testing.T) {
	originalUpdate := runUpdateForTargetRepoFn
	originalUpgrade := runUpgradeForTargetRepoFn
	originalUpdateSearch := searchOrgWorkflowReposFn
	originalUpgradeSearch := searchOrgLockWorkflowReposFn
	originalPreview := previewOrgRepoUpdatesFn
	originalScan := scanUpgradeRepoFn
	originalWait := waitForOrgRateLimitFn
	t.Cleanup(func() {
		runUpdateForTargetRepoFn = originalUpdate
		runUpgradeForTargetRepoFn = originalUpgrade
		searchOrgWorkflowReposFn = originalUpdateSearch
		searchOrgLockWorkflowReposFn = originalUpgradeSearch
		previewOrgRepoUpdatesFn = originalPreview
		scanUpgradeRepoFn = originalScan
		waitForOrgRateLimitFn = originalWait
	})
	searchOrgWorkflowReposFn = func(context.Context, string, []string, bool) ([]string, error) {
		return []string{"octo/repo"}, nil
	}
	searchOrgLockWorkflowReposFn = func(context.Context, string, bool) ([]string, error) {
		return []string{"octo/repo"}, nil
	}
	previewOrgRepoUpdatesFn = func(_ context.Context, repo string, opts UpdateWorkflowsOptions, _ bool) (orgRepoPreview, error) {
		return orgRepoPreview{
			Repo:           repo,
			TotalWorkflows: 1,
			Workflows:      []orgWorkflowPreview{{Name: "repo-assist", CurrentRef: "v1.0.0", LatestRef: "v1.1.0"}},
		}, nil
	}
	scanUpgradeRepoFn = func(_ context.Context, repo string, _ bool) (orgRepoPreview, bool, error) {
		return orgRepoPreview{Repo: repo, TotalWorkflows: 1}, true, nil
	}
	waitForOrgRateLimitFn = func(context.Context, string, bool) error { return nil }

	for _, newCommand := range []func(func(string) error) *cobra.Command{NewUpdateCommand, NewUpgradeCommand} {
		name := newCommand(func(string) error { return nil }).Name()
		for _, mode := range []string{"--repo", "--org"} {
			for _, prFlag := range []string{"--pr", "--create-pull-request"} {
				for _, closeOlder := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/%s/%s/close=%t", name, mode, prFlag, closeOlder), func(t *testing.T) {
						called := false
						check := func(repo string, createPR, gotCloseOlder bool) error {
							called = true
							assert.Equal(t, "octo/repo", repo)
							assert.True(t, createPR)
							assert.Equal(t, closeOlder, gotCloseOlder)
							return nil
						}
						runUpdateForTargetRepoFn = func(_ context.Context, repo string, opts UpdateWorkflowsOptions, createPR, _ bool) error {
							return check(repo, createPR, opts.CloseOlderPRs)
						}
						runUpgradeForTargetRepoFn = func(_ context.Context, repo string, opts upgradeOptions, createPR, _ bool) error {
							return check(repo, createPR, opts.closeOlderPRs)
						}
						target := "octo/repo"
						if mode == "--org" {
							target = "octo"
						}
						args := []string{mode, target, prFlag, "--yes"}
						if closeOlder {
							args = append(args, "--close-older-prs")
						}
						cmd := newCommand(func(string) error { return nil })
						cmd.SetArgs(args)
						require.NoError(t, cmd.Execute())
						assert.True(t, called)
					})
				}
			}
		}
	}
}
