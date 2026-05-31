//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
)

func TestResolveMaxDailyEffectiveWorkflow(t *testing.T) {
	t.Parallel()

	t.Run("prefers top-level literal value", func(t *testing.T) {
		t.Parallel()
		got := resolveMaxDailyEffectiveWorkflow(map[string]any{"max-daily-effective-workflow": 1234}, `"999"`)
		if got == nil || *got != "1234" {
			t.Fatalf("expected literal top-level value, got %v", got)
		}
	})

	t.Run("falls back to imported expression", func(t *testing.T) {
		t.Parallel()
		got := resolveMaxDailyEffectiveWorkflow(map[string]any{}, `"${{ inputs.max-daily-effective-workflow }}"`)
		if got == nil || *got != "${{ inputs.max-daily-effective-workflow }}" {
			t.Fatalf("expected imported expression, got %v", got)
		}
	})
}

func TestDailyEffectiveWorkflowGuardrailInCompiledWorkflow(t *testing.T) {
	testDir := testutil.TempDir(t, "daily-effective-workflow-guardrail-*")
	workflowFile := filepath.Join(testDir, "daily-guardrail.md")

	workflow := `---
on:
  workflow_dispatch:
  stale-check: false
max-daily-effective-workflow: 1234
safe-outputs:
  add-comment:
    max: 1
---

Guardrail test workflow`

	if err := os.WriteFile(workflowFile, []byte(workflow), 0o644); err != nil {
		t.Fatalf("failed to write test workflow: %v", err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("failed to compile workflow: %v", err)
	}

	lockFile := stringutil.MarkdownToLockFile(workflowFile)
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("failed to read lock file: %v", err)
	}
	lockStr := string(lockContent)

	if !strings.Contains(lockStr, "id: daily-effective-workflow-guardrail") {
		t.Fatal("expected activation job to include the daily workflow ET guardrail step")
	}
	if !strings.Contains(lockStr, "check_daily_effective_workflow_guardrail.cjs") {
		t.Fatal("expected activation job to call check_daily_effective_workflow_guardrail.cjs")
	}
	if !strings.Contains(lockStr, `GH_AW_MAX_DAILY_EFFECTIVE_WORKFLOW: "1234"`) {
		t.Fatal("expected activation guardrail step to receive the configured threshold")
	}
	if !strings.Contains(lockStr, "daily_effective_workflow_exceeded: ${{ steps.daily-effective-workflow-guardrail.outputs.daily_effective_workflow_exceeded == 'true' }}") {
		t.Fatal("expected activation job to expose daily_effective_workflow_exceeded output")
	}
	if !strings.Contains(lockStr, "daily_effective_workflow_total_effective_tokens: ${{ steps.daily-effective-workflow-guardrail.outputs.daily_effective_workflow_total_effective_tokens || '' }}") {
		t.Fatal("expected activation job to expose the aggregated ET total output")
	}
	if !strings.Contains(lockStr, "if: ${{ needs.activation.outputs.daily_effective_workflow_exceeded != 'true' }}") {
		t.Fatal("expected the agent job to be skipped when the daily workflow ET guardrail is exceeded")
	}
	if !strings.Contains(lockStr, "GH_AW_DAILY_EFFECTIVE_WORKFLOW_EXCEEDED: ${{ needs.activation.outputs.daily_effective_workflow_exceeded }}") {
		t.Fatal("expected the conclusion job to receive the daily workflow ET guardrail output")
	}
	if !strings.Contains(lockStr, "needs.activation.outputs.daily_effective_workflow_exceeded == 'true'") {
		t.Fatal("expected the conclusion job condition to allow activation guardrail failures through")
	}
	if !strings.Contains(lockStr, "actions: read") {
		t.Fatal("expected activation permissions to include actions: read for workflow run inspection")
	}
	if !strings.Contains(lockStr, "issues: write") {
		t.Fatal("expected activation permissions to include issues: write for guardrail issue creation")
	}
}

