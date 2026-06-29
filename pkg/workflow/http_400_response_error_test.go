//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

// TestHTTP400ResponseErrorDetectionStep tests that engines with detect-agent-errors support
// expose http_400_response_error from that step.
func TestHTTP400ResponseErrorDetectionStep(t *testing.T) {
	t.Parallel()
	engines := []string{"copilot", "codex", "claude"}
	for _, engine := range engines {
		t.Run(engine, func(t *testing.T) {
			t.Parallel()
			lockStr := compileWorkflowAndReadLock(t, `---
on: workflow_dispatch
engine: `+engine+`
---

Test workflow`)
			if !strings.Contains(lockStr, "id: detect-agent-errors") {
				t.Error("Expected agent job to have a separate detect-agent-errors step")
			}
			if !strings.Contains(lockStr, "http_400_response_error: ${{ steps.detect-agent-errors.outputs.http_400_response_error || 'false' }}") {
				t.Error("Expected agent job to have http_400_response_error output from detect-agent-errors step")
			}
		})
	}
}

// TestHTTP400ResponseErrorInConclusionJob tests that the conclusion job receives the
// http_400_response_error env var when the engine provides detect-agent-errors support.
func TestHTTP400ResponseErrorInConclusionJob(t *testing.T) {
	t.Parallel()
	engines := []string{"copilot", "codex", "claude"}
	for _, engine := range engines {
		t.Run(engine, func(t *testing.T) {
			t.Parallel()
			lockStr := compileWorkflowAndReadLock(t, `---
on: workflow_dispatch
engine: `+engine+`
safe-outputs:
  add-comment:
    max: 5
---

Test workflow`)
			if !strings.Contains(lockStr, "GH_AW_HTTP_400_RESPONSE_ERROR: ${{ needs.agent.outputs.http_400_response_error }}") {
				t.Error("Expected conclusion job to receive http_400_response_error from agent job")
			}
		})
	}
}

// TestHTTP400ResponseErrorNotInEngineWithoutDetectionScript tests that engines without
// detect-agent-errors support do not include http_400_response_error output.
func TestHTTP400ResponseErrorNotInEngineWithoutDetectionScript(t *testing.T) {
	t.Parallel()

	lockStr := compileWorkflowAndReadLock(t, `---
on: workflow_dispatch
engine: gemini
---

Test workflow`)
	if strings.Contains(lockStr, "http_400_response_error:") {
		t.Error("Expected engine without detection script to NOT have http_400_response_error output")
	}
}
