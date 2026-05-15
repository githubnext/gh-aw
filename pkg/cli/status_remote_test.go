//go:build !integration

package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBuildRemoteWorkflowStatuses tests the helper that builds status entries
// from GitHub API data when --repo is specified (no local files required).
func TestBuildRemoteWorkflowStatuses(t *testing.T) {
	githubWorkflows := map[string]*GitHubWorkflow{
		"smoke-copilot": {ID: 1, Name: "Smoke Copilot", Path: ".github/workflows/smoke-copilot.lock.yml", State: "active"},
		"weekly-report": {ID: 2, Name: "Weekly Report", Path: ".github/workflows/weekly-report.lock.yml", State: "disabled_manually"},
		"ci-check":      {ID: 3, Name: "CI Check", Path: ".github/workflows/ci-check.lock.yml", State: "active"},
	}

	t.Run("returns all workflows when no pattern", func(t *testing.T) {
		statuses := buildRemoteWorkflowStatuses("", githubWorkflows, nil)
		assert.Len(t, statuses, 3)
	})

	t.Run("filters by pattern case-insensitively", func(t *testing.T) {
		statuses := buildRemoteWorkflowStatuses("smoke", githubWorkflows, nil)
		assert.Len(t, statuses, 1)
		assert.Equal(t, "smoke-copilot", statuses[0].Workflow)
	})

	t.Run("translates disabled_manually to disabled", func(t *testing.T) {
		statuses := buildRemoteWorkflowStatuses("weekly", githubWorkflows, nil)
		assert.Len(t, statuses, 1)
		assert.Equal(t, "disabled", statuses[0].Status)
	})

	t.Run("preserves active state", func(t *testing.T) {
		statuses := buildRemoteWorkflowStatuses("ci-check", githubWorkflows, nil)
		assert.Len(t, statuses, 1)
		assert.Equal(t, "active", statuses[0].Status)
	})

	t.Run("includes run status when ref runs provided", func(t *testing.T) {
		latestRuns := map[string]*WorkflowRun{
			"smoke-copilot": {Status: "completed", Conclusion: "success"},
		}
		statuses := buildRemoteWorkflowStatuses("smoke", githubWorkflows, latestRuns)
		assert.Len(t, statuses, 1)
		assert.Equal(t, "completed", statuses[0].RunStatus)
		assert.Equal(t, "success", statuses[0].RunConclusion)
	})

	t.Run("empty run status when no matching run", func(t *testing.T) {
		latestRuns := map[string]*WorkflowRun{}
		statuses := buildRemoteWorkflowStatuses("smoke", githubWorkflows, latestRuns)
		assert.Len(t, statuses, 1)
		assert.Empty(t, statuses[0].RunStatus)
		assert.Empty(t, statuses[0].RunConclusion)
	})

	t.Run("returns empty slice when no workflows match pattern", func(t *testing.T) {
		statuses := buildRemoteWorkflowStatuses("nonexistent", githubWorkflows, nil)
		assert.Empty(t, statuses)
	})

	t.Run("returns empty slice for empty workflow map", func(t *testing.T) {
		statuses := buildRemoteWorkflowStatuses("", map[string]*GitHubWorkflow{}, nil)
		assert.Empty(t, statuses)
	})

	t.Run("local-only fields are empty for remote statuses", func(t *testing.T) {
		statuses := buildRemoteWorkflowStatuses("smoke", githubWorkflows, nil)
		assert.Len(t, statuses, 1)
		assert.Empty(t, statuses[0].EngineID, "EngineID should be empty for remote workflows")
		assert.Empty(t, statuses[0].Compiled, "Compiled should be empty for remote workflows")
		assert.Empty(t, statuses[0].TimeRemaining, "TimeRemaining should be empty for remote workflows")
		assert.Nil(t, statuses[0].Labels, "Labels should be nil for remote workflows")
		assert.Nil(t, statuses[0].On, "On should be nil for remote workflows")
	})
}

// TestGetWorkflowStatuses_WithRepoFlag_SkipsLocalFiles verifies that GetWorkflowStatuses does not
// attempt local filesystem access when repoOverride is provided. It changes to a
// directory that has no .github/workflows folder; any accidental local-file lookup
// would cause an error that would propagate to the caller.
func TestGetWorkflowStatuses_WithRepoFlag_SkipsLocalFiles(t *testing.T) {
	tmpDir := t.TempDir()

	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(tmpDir))
	defer func() { _ = os.Chdir(origDir) }()

	// With repoOverride set, GetWorkflowStatuses must not error even though
	// there is no local .github/workflows directory. The GitHub API call will
	// fail (no real token in tests) but that is swallowed and an empty result
	// is returned rather than a "no .github/workflows directory found" error.
	statuses, err := GetWorkflowStatuses("", "", "", "owner/repo")
	assert.NoError(t, err, "GetWorkflowStatuses with --repo should not propagate a 'missing local dir' error")
	// statuses may be nil (no API mock) or an empty slice; either is acceptable.
	_ = statuses
}

// TestGetWorkflowStatuses_LabelFilterWithRepo verifies that combining --label
// with --repo returns a clear error, since label information is not exposed by
// the GitHub Actions workflow API.
func TestGetWorkflowStatuses_LabelFilterWithRepo(t *testing.T) {
	_, err := GetWorkflowStatuses("", "", "my-label", "owner/repo")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--label filter is not supported with --repo")
}
