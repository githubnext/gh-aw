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

// TestSmokeCopilotAOAIWorkflowDispatchContract guards against the chronic
// "Smoke Copilot - AOAI" failures where the agent called dispatch_workflow for
// haiku-printer without the required `message` input, hard-failing the
// safe_outputs job after an otherwise successful (and expensive) run.
func TestSmokeCopilotAOAIWorkflowDispatchContract(t *testing.T) {
	repoRoot, err := gitutil.FindGitRoot()
	if err != nil {
		t.Skipf("Skipping test: not in a git repository: %v", err)
	}

	for _, workflow := range []string{"smoke-copilot-aoai-apikey.md", "smoke-copilot-aoai-entra.md"} {
		t.Run(workflow, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(repoRoot, ".github", "workflows", workflow))
			require.NoError(t, err, "Should read %s workflow", workflow)

			text := string(content)
			assert.Contains(t, text, "include `inputs.message` with an original testing/automation haiku (non-empty string)",
				"Workflow must require the haiku-printer required `message` dispatch input")
			assert.Contains(t, text, "set top-level `ref` to `${{ github.event.repository.default_branch }}`",
				"Workflow must pin the dispatch ref to the default branch")
			assert.Contains(t, text, "max-ai-credits: 60",
				"Workflow must cap per-run AI credits so failed runs cannot burn an unbounded token budget")
			assert.Contains(t, text, "timeout-minutes: 15",
				"Workflow must keep the agent runtime capped")
		})
	}
}
