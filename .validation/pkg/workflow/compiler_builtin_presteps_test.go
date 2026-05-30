//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestBuiltinJobsPreStepsInsertionOrder(t *testing.T) {
	tmpDir := testutil.TempDir(t, "builtin-pre-steps")

	workflowContent := `---
on:
  issue_comment:
    types: [created]
  roles: [admin]
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: claude
strict: false
jobs:
  pre-activation:
    pre-steps:
      - name: Pre-activation pre-step
        run: echo "pre-activation"
  activation:
    pre-steps:
      - name: Activation pre-step
        run: echo "activation"
---

# Builtin pre-steps ordering

Run builtin pre-step ordering checks.
`

	workflowFile := filepath.Join(tmpDir, "builtin-pre-steps.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("CompileWorkflow() returned error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "builtin-pre-steps.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockYAML := string(lockContent)

	activationSection := extractJobSection(lockYAML, "activation")
	if activationSection == "" {
		t.Fatal("Expected activation job section")
	}
	assertStepOrderInSection(t, activationSection,
		"id: setup",
		"- name: Activation pre-step",
		"- name: Checkout .github and .agents folders",
	)

	preActivationSection := extractJobSection(lockYAML, "pre_activation")
	if preActivationSection == "" {
		t.Fatal("Expected pre_activation job section")
	}
	assertStepOrderInSection(t, preActivationSection,
		"id: setup",
		"- name: Pre-activation pre-step",
		"- name: Check team membership",
	)

}
