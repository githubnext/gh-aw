//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestDocsAgenticsCheckoutAvoidsDefaultBranchAPI(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "docs.yml"))
	require.NoError(t, err)

	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Name string
				With map[string]any
			}
		}
	}
	require.NoError(t, yaml.Unmarshal(content, &workflow))

	for _, step := range workflow.Jobs["build"].Steps {
		if step.Name == "Checkout agentics workflows" {
			// An explicit branch skips checkout's token-authenticated REST lookup.
			require.Equal(t, "main", step.With["ref"])
			return
		}
	}
	t.Fatal("agentics checkout step not found")
}
