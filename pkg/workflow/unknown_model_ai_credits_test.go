//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// TestUnknownModelAICreditsOutput tests that the agent job outputs unknown_model_ai_credits
// from the parse-mcp-gateway step and that the conclusion job receives it as an env var.
func TestUnknownModelAICreditsOutput(t *testing.T) {
	t.Parallel()
	lockStr := compileWorkflowAndReadLock(t, `---
on: workflow_dispatch
engine: copilot
---

Test workflow`)

	if !strings.Contains(lockStr, "unknown_model_ai_credits: ${{ steps.parse-mcp-gateway.outputs.unknown_model_ai_credits || 'false' }}") {
		t.Error("Expected agent job outputs to include unknown_model_ai_credits from parse-mcp-gateway step")
	}
	if !strings.Contains(lockStr, "GH_AW_UNKNOWN_MODEL_AI_CREDITS: ${{ needs.agent.outputs.unknown_model_ai_credits || 'false' }}") {
		t.Error("Expected conclusion job env to include GH_AW_UNKNOWN_MODEL_AI_CREDITS")
	}
}
