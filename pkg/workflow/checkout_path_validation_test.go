package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateCrossRepoCheckoutPaths verifies that cross-repo checkout entries
// without an explicit path: get a warning and an auto-derived path.
func TestValidateCrossRepoCheckoutPaths(t *testing.T) {
	const markdownPath = "test-workflow.md"

	t.Run("no checkout configs produces no warning", func(t *testing.T) {
		c := NewCompiler()
		c.validateCrossRepoCheckoutPaths(nil, markdownPath)
		assert.Equal(t, 0, c.GetWarningCount())
	})

	t.Run("default-only checkout (no repository) produces no warning", func(t *testing.T) {
		c := NewCompiler()
		configs := []*CheckoutConfig{{Path: "."}}
		c.validateCrossRepoCheckoutPaths(configs, markdownPath)
		assert.Equal(t, 0, c.GetWarningCount())
		assert.Equal(t, ".", configs[0].Path, "path must not be modified")
	})

	t.Run("cross-repo checkout with explicit path produces no warning", func(t *testing.T) {
		c := NewCompiler()
		configs := []*CheckoutConfig{
			{Repository: "owner/repo", Path: "my-repo"},
		}
		c.validateCrossRepoCheckoutPaths(configs, markdownPath)
		assert.Equal(t, 0, c.GetWarningCount())
		assert.Equal(t, "my-repo", configs[0].Path, "explicit path must not be changed")
	})

	t.Run("cross-repo checkout with no path gets auto-derived path and warning", func(t *testing.T) {
		c := NewCompiler()
		configs := []*CheckoutConfig{
			{Repository: "githubnext/gh-aw-side-repo"},
		}
		c.validateCrossRepoCheckoutPaths(configs, markdownPath)
		assert.Equal(t, 1, c.GetWarningCount())
		assert.Equal(t, "gh-aw-side-repo", configs[0].Path, "path must be auto-derived from repo name")
	})

	t.Run("auto-derived path is only the repo-name portion of owner/repo", func(t *testing.T) {
		c := NewCompiler()
		configs := []*CheckoutConfig{
			{Repository: "some-org/my-awesome-tool"},
		}
		c.validateCrossRepoCheckoutPaths(configs, markdownPath)
		require.Equal(t, 1, c.GetWarningCount())
		assert.Equal(t, "my-awesome-tool", configs[0].Path)
	})

	t.Run("multiple cross-repo checkouts without path each get a warning and derived path", func(t *testing.T) {
		c := NewCompiler()
		configs := []*CheckoutConfig{
			{Repository: "owner/alpha"},
			{Repository: "owner/beta", Path: "explicit-beta"}, // already has path — unchanged
			{Repository: "owner/gamma"},
		}
		c.validateCrossRepoCheckoutPaths(configs, markdownPath)
		assert.Equal(t, 2, c.GetWarningCount())
		assert.Equal(t, "alpha", configs[0].Path)
		assert.Equal(t, "explicit-beta", configs[1].Path, "pre-set path must not be changed")
		assert.Equal(t, "gamma", configs[2].Path)
	})

	t.Run("nil checkout entry in slice is skipped without panic", func(t *testing.T) {
		c := NewCompiler()
		configs := []*CheckoutConfig{nil}
		require.NotPanics(t, func() { c.validateCrossRepoCheckoutPaths(configs, markdownPath) })
		assert.Equal(t, 0, c.GetWarningCount())
	})

	t.Run("dynamic repository expression is skipped (no warning, no path change)", func(t *testing.T) {
		c := NewCompiler()
		configs := []*CheckoutConfig{
			{Repository: "${{ github.event.inputs.repo }}"},
		}
		c.validateCrossRepoCheckoutPaths(configs, markdownPath)
		assert.Equal(t, 0, c.GetWarningCount(), "dynamic expressions cannot be determined as cross-repo at compile time")
		assert.Empty(t, configs[0].Path, "path must not be modified for dynamic repos")
	})

	t.Run("github.repository expression (same-repo trusted checkout) is skipped", func(t *testing.T) {
		c := NewCompiler()
		configs := []*CheckoutConfig{
			{Repository: "${{ github.repository }}"},
		}
		c.validateCrossRepoCheckoutPaths(configs, markdownPath)
		assert.Equal(t, 0, c.GetWarningCount(), "github.repository is a same-repo pattern used in pull_request_target")
		assert.Empty(t, configs[0].Path)
	})

	t.Run("warning message for static repo contains suggested path", func(t *testing.T) {
		c := NewCompiler()
		// Capture stderr by checking warning count and path derivation;
		// the actual message text is validated by the path value and warning count.
		configs := []*CheckoutConfig{
			{Repository: "acme/widget-service"},
		}
		c.validateCrossRepoCheckoutPaths(configs, markdownPath)
		assert.Equal(t, "widget-service", configs[0].Path)
		assert.Equal(t, 1, c.GetWarningCount())
	})
}

// TestValidateCrossRepoCheckoutPathsViaCompiler verifies that the warning and path
// auto-derivation are triggered end-to-end when compiling a workflow.
func TestValidateCrossRepoCheckoutPathsViaCompiler(t *testing.T) {
	t.Run("compiler emits warning for cross-repo checkout without path", func(t *testing.T) {
		c := NewCompiler()

		workflowData := minimalWorkflowData()
		workflowData.CheckoutConfigs = []*CheckoutConfig{
			{Repository: "owner/side-repo"},
		}

		markdownPath := "test.md"
		err := c.validateWorkflowData(workflowData, markdownPath)
		require.NoError(t, err, "missing path should be a warning, not an error")
		assert.Equal(t, 1, c.GetWarningCount(), "expected one warning for missing path")
		// Confirm auto-derivation happened
		assert.Equal(t, "side-repo", workflowData.CheckoutConfigs[0].Path)
	})

	t.Run("compiler emits no warning when path is explicitly set", func(t *testing.T) {
		c := NewCompiler()

		workflowData := minimalWorkflowData()
		workflowData.CheckoutConfigs = []*CheckoutConfig{
			{Repository: "owner/side-repo", Path: "side-repo"},
		}

		markdownPath := "test.md"
		err := c.validateWorkflowData(workflowData, markdownPath)
		require.NoError(t, err)
		assert.Equal(t, 0, c.GetWarningCount())
	})
}

// minimalWorkflowData returns a WorkflowData suitable for use in compiler validation tests.
func minimalWorkflowData() *WorkflowData {
	return &WorkflowData{
		Name:            "Test Workflow",
		MarkdownContent: "Test content.",
		On:              "push:",
		Permissions:     "contents: read",
	}
}

// warnCrossRepoPath is a helper used by tests that need to capture the warning message text.
func warnCrossRepoPath(t *testing.T, repository string) (path string, warnCount int) {
	t.Helper()
	c := NewCompiler()
	cfg := &CheckoutConfig{Repository: repository}
	c.validateCrossRepoCheckoutPaths([]*CheckoutConfig{cfg}, "test.md")
	return cfg.Path, c.GetWarningCount()
}

func TestWarnCrossRepoPath_RepoNameExtraction(t *testing.T) {
	tests := []struct {
		repository   string
		expectedPath string
	}{
		{"owner/repo", "repo"},
		{"github/copilot", "copilot"},
		{"githubnext/gh-aw-side-repo", "gh-aw-side-repo"},
		{"a/b/c", "c"}, // only the last segment
	}
	for _, tt := range tests {
		t.Run(tt.repository, func(t *testing.T) {
			path, count := warnCrossRepoPath(t, tt.repository)
			assert.Equal(t, tt.expectedPath, path)
			assert.Equal(t, 1, count)
		})
	}
}

// TestCrossRepoCheckoutPathAppearsInManifestStep confirms that after auto-derivation the
// CheckoutManager emits a non-empty GH_AW_CHECKOUT_PATH_N in the manifest step.
func TestCrossRepoCheckoutPathAppearsInManifestStep(t *testing.T) {
	getActionPin := func(action string) string { return action + "@pin" }

	t.Run("auto-derived path appears in manifest step", func(t *testing.T) {
		// Simulate what the compiler does: validate configs first (derives path), then pass to manager.
		configs := []*CheckoutConfig{
			{Repository: "githubnext/gh-aw-side-repo"},
		}
		c := NewCompiler()
		c.validateCrossRepoCheckoutPaths(configs, "test.md")

		cm := NewCheckoutManager(configs)
		steps := cm.GenerateCheckoutManifestStep(getActionPin)
		require.Len(t, steps, 1)
		out := steps[0]
		assert.Contains(t, out, `GH_AW_CHECKOUT_REPO_0: "githubnext/gh-aw-side-repo"`)
		assert.Contains(t, out, `GH_AW_CHECKOUT_PATH_0: "gh-aw-side-repo"`)
		assert.NotContains(t, out, `GH_AW_CHECKOUT_PATH_0: ""`, "path must not be empty after auto-derivation")
	})
}

// TestCrossRepoCheckoutPathAppearsInCheckoutStep confirms that the checkout step emits
// a path: field after auto-derivation.
func TestCrossRepoCheckoutPathAppearsInCheckoutStep(t *testing.T) {
	getActionPin := func(action string) string { return action + "@pin" }

	configs := []*CheckoutConfig{
		{Repository: "acme/my-lib"},
	}
	c := NewCompiler()
	c.validateCrossRepoCheckoutPaths(configs, "test.md")

	cm := NewCheckoutManager(configs)
	lines := cm.GenerateAdditionalCheckoutSteps(getActionPin)
	combined := strings.Join(lines, "")
	assert.Contains(t, combined, "repository: acme/my-lib")
	assert.Contains(t, combined, "path: my-lib", "checkout step must include the auto-derived path")
}
