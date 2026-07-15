//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestCustomRunnerNodeValidationStepPrecedesJavaScriptActions(t *testing.T) {
	tmpDir := testutil.TempDir(t, "custom-runner-node-validation")
	workflowFile := filepath.Join(tmpDir, "custom-runner-node-validation.md")

	workflowContent := `---
on: push
engine: copilot
runs-on: self-hosted
---

# Custom runner validation
`

	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	compiler.SetActionMode(ActionModeDev)
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("CompileWorkflow() returned error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "custom-runner-node-validation.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	agentSection := extractJobSection(string(lockContent), "agent")
	if agentSection == "" {
		t.Fatal("Expected agent job section")
	}

	assertStepOrderInSection(t, agentSection,
		"- name: Validate Node.js on custom runner",
		"- name: Checkout actions folder",
		"- name: Setup Scripts",
	)

	if !strings.Contains(agentSection, "This self-hosted or GPU runner is missing 'node' on PATH.") {
		t.Fatalf("Expected actionable missing-node guidance in agent section:\n%s", agentSection)
	}
	if !strings.Contains(agentSection, "The workflow 'runtimes.node' setting applies later in the job and cannot satisfy this prerequisite.") {
		t.Fatalf("Expected runtimes.node remediation guidance in agent section:\n%s", agentSection)
	}
	if !strings.Contains(agentSection, "exit 127") {
		t.Fatalf("Expected missing-node validation step to fail with exit 127:\n%s", agentSection)
	}
}

func TestStandardRunnerSkipsCustomRunnerNodeValidationStep(t *testing.T) {
	tmpDir := testutil.TempDir(t, "standard-runner-node-validation")
	workflowFile := filepath.Join(tmpDir, "standard-runner-node-validation.md")

	workflowContent := `---
on: push
engine: copilot
runs-on: ubuntu-latest
---

# Standard runner validation
`

	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()
	compiler.SetActionMode(ActionModeDev)
	if err := compiler.CompileWorkflow(workflowFile); err != nil {
		t.Fatalf("CompileWorkflow() returned error: %v", err)
	}

	lockFile := filepath.Join(tmpDir, "standard-runner-node-validation.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	agentSection := extractJobSection(string(lockContent), "agent")
	if agentSection == "" {
		t.Fatal("Expected agent job section")
	}

	if strings.Contains(agentSection, "Validate Node.js on custom runner") {
		t.Fatalf("Did not expect custom-runner node validation step for ubuntu-latest:\n%s", agentSection)
	}
}
