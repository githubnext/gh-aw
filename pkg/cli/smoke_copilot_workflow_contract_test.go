//go:build !integration

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSmokeCopilotGitHubURLPermission(t *testing.T) {
	t.Parallel()
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	source, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "smoke-copilot.md"))
	require.NoError(t, err)
	assert.Contains(t, string(source), `args: ["--allow-url", "github.com"]`,
		"GitHub curl smoke test must not require interactive URL approval")
	assert.NotContains(t, string(source), "--allow-all-urls",
		"URL approval must remain scoped to the tested host")
	assert.Contains(t, string(source), `requiredSubAgents: ["file-summarizer"]`,
		"Required delegation must still be checked")

	compiled, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", "smoke-copilot.lock.yml"))
	require.NoError(t, err)
	assert.Contains(t, string(compiled), "--allow-url github.com",
		"Compiled Copilot invocation must include the scoped URL permission")
	assert.NotContains(t, string(compiled), "--allow-all-urls")
}
