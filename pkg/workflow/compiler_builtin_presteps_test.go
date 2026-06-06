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

func TestBuiltinJobsSetupStepsRunBeforeTokenMinting(t *testing.T) {
	tmpDir := testutil.TempDir(t, "builtin-setup-steps-token-order")

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
safe-outputs:
  github-app:
    app-id: "${{ vars.ACTIONS_APP_ID }}"
    private-key: "${{ secrets.ACTIONS_PRIVATE_KEY }}"
  add-comment:
jobs:
  agent:
    setup-steps:
      - name: Agent setup-step
        run: echo "agent-setup"
  safe_outputs:
    setup-steps:
      - name: Safe outputs setup-step
        run: echo "safe-outputs-setup"
  conclusion:
    setup-steps:
      - name: Conclusion setup-step
        run: echo "conclusion-setup"
---

# Builtin setup-steps ordering
`

	workflowFile := filepath.Join(tmpDir, "builtin-setup-steps.md")
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("CompileWorkflow() returned error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "builtin-setup-steps.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lockYAML := string(lockContent)

	agentSection := extractJobSection(lockYAML, "agent")
	if agentSection == "" {
		t.Fatal("Expected agent job section")
	}
	if !containsInNonCommentLines(agentSection, "- name: Agent setup-step") {
		t.Fatalf("Expected agent setup-step in agent section:\n%s", agentSection)
	}

	safeOutputsSection := extractJobSection(lockYAML, "safe_outputs")
	if safeOutputsSection == "" {
		t.Fatal("Expected safe_outputs job section")
	}
	assertStepOrderInSection(t, safeOutputsSection,
		"- name: Safe outputs setup-step",
		"- name: Generate GitHub App token",
	)

	conclusionSection := extractJobSection(lockYAML, "conclusion")
	if conclusionSection == "" {
		t.Fatal("Expected conclusion job section")
	}
	assertStepOrderInSection(t, conclusionSection,
		"- name: Conclusion setup-step",
		"- name: Generate GitHub App token",
	)
}
