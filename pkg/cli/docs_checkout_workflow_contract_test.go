//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestDocsAgenticsCheckoutAvoidsDefaultBranchAPI(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	require.NoError(t, err)
	content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "docs.yml"))
	require.NoError(t, err)

	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name             string
				Uses             string
				With             map[string]any
				Env              map[string]string
				Run              string
				WorkingDirectory string `yaml:"working-directory"`
			}
		}
	}
	require.NoError(t, yaml.Unmarshal(content, &workflow))

	checkoutFound, buildFound := false, false
	for _, step := range workflow.Jobs["build"].Steps {
		switch step.Name {
		case "Checkout agentics workflows":
			checkoutFound = true
			assert.Contains(t, step.Uses, "actions/checkout@")
			assert.Equal(t, "githubnext/agentics", step.With["repository"])
			// An explicit branch skips checkout's token-authenticated REST lookup.
			assert.Equal(t, "main", step.With["ref"])
			assert.Equal(t, ".agentics", step.With["path"])
			assert.Equal(t, "workflows", step.With["sparse-checkout"])
			assert.Equal(t, false, step.With["persist-credentials"])
		case "Build documentation":
			buildFound = true
			assert.True(t, checkoutFound, "workflow examples must be checked out before building")
			assert.Equal(t, "${{ github.workspace }}/.agentics/workflows", step.Env["AGENTICS_WORKFLOWS_DIR"])
			assert.Equal(t, "./docs", step.WorkingDirectory)
			assert.Equal(t, "npm run build", step.Run)
		}
	}
	assert.True(t, checkoutFound, "docs require the external workflow examples")
	assert.True(t, buildFound, "docs build must remain enabled")
}
