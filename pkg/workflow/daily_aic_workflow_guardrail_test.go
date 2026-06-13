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

func TestResolveMaxDailyAIC(t *testing.T) {
	t.Run("prefers top-level literal value", func(t *testing.T) {
		t.Parallel()
		got := resolveMaxDailyAIC(map[string]any{"max-daily-ai-credits": 1234}, `"999"`)
		if got == nil || *got != "1234" {
			t.Fatalf("expected literal top-level value, got %v", got)
		}

	})

	t.Run("falls back to imported expression", func(t *testing.T) {
		t.Parallel()
		got := resolveMaxDailyAIC(map[string]any{}, `"${{ inputs.max-daily-ai-credits }}"`)
		if got == nil || *got != "${{ inputs.max-daily-ai-credits }}" {
			t.Fatalf("expected imported expression, got %v", got)
		}
	})

	t.Run("emits runtime expression when no frontmatter", func(t *testing.T) {
		t.Parallel()
		got := resolveMaxDailyAIC(map[string]any{}, "")
		wantExpr := "${{ vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS || '5000' }}"
		if got == nil || *got != wantExpr {
			t.Fatalf("expected runtime expression %q, got %v", wantExpr, got)
		}
	})

	t.Run("frontmatter value takes precedence over runtime default expression", func(t *testing.T) {
		got := resolveMaxDailyAIC(map[string]any{"max-daily-ai-credits": 1234}, "")
		if got == nil || *got != "1234" {
			t.Fatalf("expected frontmatter value to override runtime default expression, got %v", got)
		}
	})

	t.Run("normalizes suffix strings", func(t *testing.T) {
		t.Parallel()
		got := resolveMaxDailyAIC(map[string]any{"max-daily-ai-credits": "100M"}, "")
		if got == nil || *got != "100000000" {
			t.Fatalf("expected normalized suffix string, got %v", got)
		}
	})

	t.Run("explicit disable skips guardrail", func(t *testing.T) {
		t.Parallel()
		got := resolveMaxDailyAIC(map[string]any{"max-daily-ai-credits": -1}, "")
		if got != nil {
			t.Fatalf("expected explicit disable to skip the guardrail, got %v", *got)
		}
	})

	// T-AIC-DG-007: Imported workflow max-daily-ai-credits used when no frontmatter value;
	// frontmatter takes precedence over imports (spec §9.3 (2)).
	t.Run("spec §9.3(2) / T-AIC-DG-007: imported config used when no frontmatter value", func(t *testing.T) {
		t.Parallel()
		got := resolveMaxDailyAIC(map[string]any{}, `"2000"`)
		if got == nil || *got != "2000" {
			t.Fatalf("spec §9.3(2): expected imported config value %q, got %v", "2000", got)
		}
	})

	t.Run("spec §9.3(2) / T-AIC-DG-007: frontmatter takes precedence over imported config", func(t *testing.T) {
		t.Parallel()
		got := resolveMaxDailyAIC(map[string]any{"max-daily-ai-credits": 9999}, `"2000"`)
		if got == nil || *got != "9999" {
			t.Fatalf("spec §9.3(2): expected frontmatter value to override imported config, got %v", got)
		}
	})
}

func TestDailyAICWorkflowGuardrailInCompiledWorkflow(t *testing.T) {
	testDir := testutil.TempDir(t, "daily-effective-workflow-guardrail-*")
	workflowFile := filepath.Join(testDir, "daily-guardrail.md")

	workflow := `---
on:
  workflow_dispatch:
  stale-check: false
max-daily-ai-credits: 100_000_000
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
	activationStart := strings.Index(lockStr, "\n  activation:\n")
	if activationStart == -1 {
		t.Fatal("expected compiled workflow to include an activation job")
	}
	activationSection := lockStr[activationStart:]
	if nextJob := strings.Index(activationSection, "\n  agent:\n"); nextJob != -1 {
		activationSection = activationSection[:nextJob]
	}

	if !strings.Contains(lockStr, "id: daily-effective-workflow-guardrail") {
		t.Fatal("expected activation job to include the daily AI Credits guardrail step")
	}
	if !strings.Contains(lockStr, "if: ${{ env.GH_AW_MAX_DAILY_AI_CREDITS != '' }}") {
		t.Fatal("expected frontmatter-configured guardrail step to use env-based runtime gating")
	}
	if !strings.Contains(lockStr, "check_daily_aic_workflow_guardrail.cjs") {
		t.Fatal("expected activation job to call check_daily_aic_workflow_guardrail.cjs")
	}
	if !strings.Contains(lockStr, `GH_AW_MAX_DAILY_AI_CREDITS: "100000000"`) {
		t.Fatal("expected activation job env to include normalized guardrail threshold")
	}
	if !strings.Contains(lockStr, "daily_ai_credits_exceeded: ${{ steps.daily-effective-workflow-guardrail.outputs.daily_ai_credits_exceeded == 'true' }}") {
		t.Fatal("expected activation job to expose daily_ai_credits_exceeded output")
	}
	if !strings.Contains(lockStr, "daily_ai_credits_total_effective_tokens: ${{ steps.daily-effective-workflow-guardrail.outputs.daily_ai_credits_total_effective_tokens || '' }}") {
		t.Fatal("expected activation job to expose the aggregated AI Credits total output")
	}
	if strings.Contains(lockStr, "daily_ai_credits_issue_url") {
		t.Fatal("expected activation job to avoid surfacing a separate daily AI Credits issue URL")
	}
	if !strings.Contains(lockStr, "if: needs.activation.outputs.daily_ai_credits_exceeded != 'true'") {
		t.Fatal("expected the agent job to be skipped when the daily AI Credits guardrail is exceeded")
	}
	if !strings.Contains(lockStr, "GH_AW_DAILY_AI_CREDITS_EXCEEDED: ${{ needs.activation.outputs.daily_ai_credits_exceeded }}") {
		t.Fatal("expected the conclusion job to receive the daily AI Credits guardrail output")
	}
	if !strings.Contains(lockStr, "needs.activation.outputs.daily_ai_credits_exceeded == 'true'") {
		t.Fatal("expected the conclusion job condition to allow activation guardrail failures through")
	}
	if !strings.Contains(activationSection, "actions: read") {
		t.Fatal("expected activation permissions to include actions: read for workflow run inspection")
	}
	if strings.Contains(activationSection, "issues: write") {
		t.Fatal("expected activation permissions to avoid issues: write for the daily AI Credits guardrail")
	}
	if !strings.Contains(activationSection, "safe-output-artifact-client: ${{ env.GH_AW_MAX_DAILY_AI_CREDITS != '' }}") {
		t.Fatal("expected frontmatter-configured guardrail to gate artifact client installation dynamically")
	}
	if !strings.Contains(activationSection, "restore_aic_usage_cache_fallback.cjs") {
		t.Fatal("expected activation job to call restore_aic_usage_cache_fallback.cjs for cross-branch cache fallback")
	}
	if !strings.Contains(activationSection, "id: restore-daily-aic-cache-fallback") {
		t.Fatal("expected activation job to include the artifact-based AIC cache fallback step")
	}
	if !strings.Contains(lockStr, "id: upload-daily-aic-cache") {
		t.Fatal("expected conclusion job to include the AIC usage cache artifact upload step")
	}
	if !strings.Contains(lockStr, "name: aic-usage-cache") {
		t.Fatal("expected conclusion job to upload artifact named aic-usage-cache")
	}
}

func TestDailyETGuardrailDynamicGate(t *testing.T) {
	testDir := testutil.TempDir(t, "daily-effective-workflow-no-guardrail-*")
	workflowFile := filepath.Join(testDir, "no-daily-guardrail.md")

	workflow := `---
on:
  workflow_dispatch:
  stale-check: false
safe-outputs:
  add-comment:
    max: 1
---

No daily guardrail`

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
		t.Fatal("expected activation job to emit the daily AI Credits guardrail step even when threshold is unset")
	}
	if !strings.Contains(lockStr, "if: ${{ env.GH_AW_MAX_DAILY_AI_CREDITS != '' }}") {
		t.Fatal("expected emitted daily AI Credits guardrail step to be dynamically skipped when threshold is unset")
	}
	if !strings.Contains(lockStr, "daily_ai_credits_exceeded") {
		t.Fatal("expected workflows to continue wiring daily AI Credits outputs when guardrail step is emitted")
	}
	if !strings.Contains(lockStr, "safe-output-artifact-client: ${{ env.GH_AW_MAX_DAILY_AI_CREDITS != '' }}") {
		t.Fatal("expected emitted guardrail to gate artifact client installation dynamically")
	}
}

func TestDailyAICWorkflowGuardrailConfiguredViaEnvVar(t *testing.T) {
	testDir := testutil.TempDir(t, "daily-effective-workflow-env-guardrail-*")
	workflowFile := filepath.Join(testDir, "daily-guardrail-env.md")

	workflow := `---
on:
  workflow_dispatch:
  stale-check: false
env:
  GH_AW_MAX_DAILY_AI_CREDITS: "5000000"
safe-outputs:
  add-comment:
    max: 1
---

Daily guardrail via env var`

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
		t.Fatal("expected activation job to include the daily AI Credits guardrail step when env var is configured")
	}
	if !strings.Contains(lockStr, "if: ${{ env.GH_AW_MAX_DAILY_AI_CREDITS != '' }}") {
		t.Fatal("expected daily AI Credits guardrail step to gate execution on GH_AW_MAX_DAILY_AI_CREDITS")
	}
	if !strings.Contains(lockStr, "safe-output-artifact-client: ${{ env.GH_AW_MAX_DAILY_AI_CREDITS != '' }}") {
		t.Fatal("expected setup step to conditionally install artifact client when daily AI Credits guardrail is env-configured")
	}
}

func TestDailyETGuardrailNegativeValueRejected(t *testing.T) {
	testDir := testutil.TempDir(t, "daily-effective-workflow-explicit-disable-*")
	workflowFile := filepath.Join(testDir, "daily-guardrail-explicit-disable.md")

	// -2 is below the minimum of -1 (the explicit disable sentinel) and must be rejected.
	workflow := `---
on:
  workflow_dispatch:
  stale-check: false
max-daily-ai-credits: -2
safe-outputs:
  add-comment:
    max: 1
---

Invalid negative daily guardrail value`

	if err := os.WriteFile(workflowFile, []byte(workflow), 0o644); err != nil {
		t.Fatalf("failed to write test workflow: %v", err)
	}

	compiler := NewCompiler()
	err := compiler.CompileWorkflow(workflowFile)
	if err == nil {
		t.Fatal("expected compile to fail for invalid negative max-daily-ai-credits")
	}
	// Schema validation or frontmatter validation may produce the error; either
	// correctly rejects values below -1.
	if !strings.Contains(err.Error(), "must be -1") && !strings.Contains(err.Error(), "minimum") {
		t.Fatalf("expected validation error rejecting -2, got: %v", err)
	}
}
