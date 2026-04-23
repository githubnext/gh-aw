//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestPreAgentStepsGeneration(t *testing.T) {
	tmpDir := testutil.TempDir(t, "pre-agent-steps-test")

	testContent := `---
on: push
permissions:
  contents: read
pre-agent-steps:
  - name: Finalize prompt context
    run: echo "finalize"
engine: claude
strict: false
---

Test pre-agent-steps.
`

	testFile := filepath.Join(tmpDir, "test-pre-agent-steps.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow with pre-agent-steps: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "test-pre-agent-steps.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	lockContent := string(content)

	if !strings.Contains(lockContent, "- name: Finalize prompt context") {
		t.Error("Expected pre-agent-step to be in generated workflow")
	}

	startMCPGatewayIndex := indexInNonCommentLines(lockContent, "- name: Start MCP Gateway")
	preAgentStepIndex := indexInNonCommentLines(lockContent, "- name: Finalize prompt context")
	aiStepIndex := indexInNonCommentLines(lockContent, "- name: Execute Claude Code CLI")
	if startMCPGatewayIndex == -1 || preAgentStepIndex == -1 || aiStepIndex == -1 {
		t.Fatal("Could not find expected steps in generated workflow")
	}
	if preAgentStepIndex >= startMCPGatewayIndex {
		t.Errorf("Pre-agent-step (%d) should appear before Start MCP Gateway (%d)", preAgentStepIndex, startMCPGatewayIndex)
	}
	if preAgentStepIndex >= aiStepIndex {
		t.Errorf("Pre-agent-step (%d) should appear before AI execution step (%d)", preAgentStepIndex, aiStepIndex)
	}
}

func TestPreAgentStepsImportsMergeOrder(t *testing.T) {
	tmpDir := testutil.TempDir(t, "pre-agent-steps-imports-test")

	sharedContent := `---
pre-agent-steps:
  - name: Imported pre-agent step
    run: echo "imported"
---

Shared steps.
`
	sharedFile := filepath.Join(tmpDir, "shared.md")
	if err := os.WriteFile(sharedFile, []byte(sharedContent), 0644); err != nil {
		t.Fatal(err)
	}

	mainContent := `---
on: issues
permissions:
  contents: read
imports:
  - ./shared.md
pre-agent-steps:
  - name: Main pre-agent step
    run: echo "main"
engine: claude
strict: false
---

Main workflow.
`
	mainFile := filepath.Join(tmpDir, "main.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(mainFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow with imported pre-agent-steps: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "main.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	lockContent := string(content)

	importedIdx := indexInNonCommentLines(lockContent, "- name: Imported pre-agent step")
	mainIdx := indexInNonCommentLines(lockContent, "- name: Main pre-agent step")
	startMCPGatewayIdx := indexInNonCommentLines(lockContent, "- name: Start MCP Gateway")
	aiStepIdx := indexInNonCommentLines(lockContent, "- name: Execute Claude Code CLI")
	if importedIdx == -1 || mainIdx == -1 || startMCPGatewayIdx == -1 || aiStepIdx == -1 {
		t.Fatal("Could not find expected pre-agent, MCP gateway, and AI steps in generated workflow")
	}
	if importedIdx >= mainIdx {
		t.Errorf("Imported pre-agent-step (%d) should appear before main pre-agent-step (%d)", importedIdx, mainIdx)
	}
	if mainIdx >= startMCPGatewayIdx {
		t.Errorf("Main pre-agent-step (%d) should appear before Start MCP Gateway (%d)", mainIdx, startMCPGatewayIdx)
	}
	if mainIdx >= aiStepIdx {
		t.Errorf("Main pre-agent-step (%d) should appear before AI execution step (%d)", mainIdx, aiStepIdx)
	}
}

func TestImportedPreAgentStepsRunAfterPRBaseRestore(t *testing.T) {
	tmpDir := testutil.TempDir(t, "pre-agent-steps-pr-restore-test")

	sharedDir := filepath.Join(tmpDir, "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}

	sharedContent := `---
pre-agent-steps:
  - name: Restore APM packages
    run: echo "restore apm"
---

Shared APM-style steps.
`
	sharedFile := filepath.Join(sharedDir, "apm.md")
	if err := os.WriteFile(sharedFile, []byte(sharedContent), 0644); err != nil {
		t.Fatal(err)
	}

	mainContent := `---
on:
  pull_request:
    types: [opened]
permissions:
  contents: read
  issues: read
  pull-requests: read
imports:
  - ./shared/apm.md
engine: claude
strict: false
---

Main workflow.
`
	mainFile := filepath.Join(tmpDir, "main.md")
	if err := os.WriteFile(mainFile, []byte(mainContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(mainFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow with imported pre-agent-steps in PR context: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "main.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}
	lockContent := string(content)

	restoreAPMIdx := indexInNonCommentLines(lockContent, "- name: Restore APM packages")
	startMCPGatewayIdx := indexInNonCommentLines(lockContent, "- name: Start MCP Gateway")
	aiStepIdx := indexInNonCommentLines(lockContent, "- name: Execute Claude Code CLI")
	if restoreAPMIdx == -1 || startMCPGatewayIdx == -1 || aiStepIdx == -1 {
		t.Fatal("Could not find expected pre-agent, MCP gateway, and AI steps in generated workflow")
	}
	if restoreAPMIdx >= startMCPGatewayIdx {
		t.Errorf("Imported pre-agent step (%d) should appear before Start MCP Gateway (%d)", restoreAPMIdx, startMCPGatewayIdx)
	}
	if restoreAPMIdx >= aiStepIdx {
		t.Errorf("Imported pre-agent step (%d) should appear before AI execution step (%d)", restoreAPMIdx, aiStepIdx)
	}
}
