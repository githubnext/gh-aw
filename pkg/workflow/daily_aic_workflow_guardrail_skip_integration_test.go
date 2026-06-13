//go:build integration

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

func TestDailyAICGuardrailSkipScenariosIntegration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		frontmatterOn string
		wantSlashFlag bool
		wantLabelFlag bool
	}{
		{
			name: "manual workflow_dispatch",
			frontmatterOn: `on:
  workflow_dispatch:`,
			wantSlashFlag: false,
			wantLabelFlag: false,
		},
		{
			name: "slash command centralized",
			frontmatterOn: `on:
  slash_command:
    name: fix
    strategy: centralized`,
			wantSlashFlag: true,
			wantLabelFlag: false,
		},
		{
			name: "slash command non centralized",
			frontmatterOn: `on:
  slash_command:
    name: fix`,
			wantSlashFlag: true,
			wantLabelFlag: false,
		},
		{
			name: "label command centralized",
			frontmatterOn: `on:
  label_command:
    name: ci-doctor`,
			wantSlashFlag: false,
			wantLabelFlag: true,
		},
		{
			name: "label command non centralized",
			frontmatterOn: `on:
  label_command:
    name: ci-doctor
    strategy: decentralized`,
			wantSlashFlag: false,
			wantLabelFlag: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			testDir := t.TempDir()
			workflowFile := filepath.Join(testDir, "daily-aic-skip.md")
			workflow := fmt.Sprintf(`---
%s
max-daily-ai-credits: 5000
safe-outputs:
  add-comment:
    max: 1
---

Daily AIC skip integration test
`, tt.frontmatterOn)

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

			if !strings.Contains(lockStr, "GH_AW_WORKFLOW_DISPATCH_AW_CONTEXT: ${{ github.event.inputs.aw_context || '' }}") {
				t.Fatal("expected daily guardrail step to expose workflow_dispatch aw_context input")
			}

			wantSlash := fmt.Sprintf(`GH_AW_HAS_SLASH_COMMAND: "%t"`, tt.wantSlashFlag)
			if !strings.Contains(lockStr, wantSlash) {
				t.Fatalf("expected lock file to contain %q", wantSlash)
			}

			wantLabel := fmt.Sprintf(`GH_AW_HAS_LABEL_COMMAND: "%t"`, tt.wantLabelFlag)
			if !strings.Contains(lockStr, wantLabel) {
				t.Fatalf("expected lock file to contain %q", wantLabel)
			}
		})
	}
}
