//go:build integration

package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompileServicePortsWorkflow compiles the canonical test-service-ports.md workflow
// and verifies that the generated lock file contains --allow-host-service-ports with the
// correct ${{ job.services['<id>'].ports['<port>'] }} expressions for every service port.
func TestCompileServicePortsWorkflow(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	srcPath := filepath.Join(projectRoot, "pkg/cli/workflows/test-service-ports.md")
	dstPath := filepath.Join(setup.workflowsDir, "test-service-ports.md")

	srcContent, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("Failed to read source workflow %s: %v", srcPath, err)
	}
	if err := os.WriteFile(dstPath, srcContent, 0644); err != nil {
		t.Fatalf("Failed to write workflow to test dir: %v", err)
	}

	cmd := exec.Command(setup.binaryPath, "compile", dstPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Compile failed: %v\nOutput: %s", err, string(output))
	}

	lockFilePath := filepath.Join(setup.workflowsDir, "test-service-ports.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lock := string(lockContent)

	// The compiler must emit --allow-host-service-ports
	if !strings.Contains(lock, "--allow-host-service-ports") {
		t.Errorf("Lock file missing --allow-host-service-ports\nLock content:\n%s", lock)
	}

	// Bracket-notation expressions must be present for both services
	for _, expr := range []string{
		"job.services['postgres'].ports['5432']",
		"job.services['redis'].ports['6379']",
	} {
		if !strings.Contains(lock, expr) {
			t.Errorf("Lock file missing expected expression %q\nLock content:\n%s", expr, lock)
		}
	}

	t.Logf("test-service-ports.md compiled successfully; --allow-host-service-ports verified")
}

// TestCompileServicePorts_NoServices verifies that a workflow with no services block
// compiles without errors and does NOT emit --allow-host-service-ports.
func TestCompileServicePorts_NoServices(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	testWorkflow := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
---

# No Services Workflow

This workflow has no services block and should not include --allow-host-service-ports.
`
	testPath := filepath.Join(setup.workflowsDir, "no-services.md")
	if err := os.WriteFile(testPath, []byte(testWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	cmd := exec.Command(setup.binaryPath, "compile", testPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Compile failed: %v\nOutput: %s", err, string(output))
	}

	lockFilePath := filepath.Join(setup.workflowsDir, "no-services.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	if strings.Contains(string(lockContent), "--allow-host-service-ports") {
		t.Errorf("Lock file should NOT contain --allow-host-service-ports when no services are defined")
	}
}

// TestCompileServicePorts_HyphenatedServiceID verifies that service IDs containing
// hyphens are emitted with bracket notation (not dot notation) in the compiled lock file.
func TestCompileServicePorts_HyphenatedServiceID(t *testing.T) {
	setup := setupIntegrationTest(t)
	defer setup.cleanup()

	testWorkflow := `---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
services:
  my-postgres:
    image: postgres:15
    ports:
      - 5432:5432
---

# Hyphenated Service ID Workflow

Verifies bracket notation for hyphenated service IDs.
`
	testPath := filepath.Join(setup.workflowsDir, "hyphenated-service.md")
	if err := os.WriteFile(testPath, []byte(testWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write workflow: %v", err)
	}

	cmd := exec.Command(setup.binaryPath, "compile", testPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Compile failed: %v\nOutput: %s", err, string(output))
	}

	lockFilePath := filepath.Join(setup.workflowsDir, "hyphenated-service.lock.yml")
	lockContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}
	lock := string(lockContent)

	// Must use bracket notation, not dot notation
	bracketNotation := "job.services['my-postgres'].ports['5432']"
	dotNotation := "job.services.my-postgres.ports"

	if !strings.Contains(lock, bracketNotation) {
		t.Errorf("Lock file missing bracket-notation expression %q\nLock content:\n%s", bracketNotation, lock)
	}
	if strings.Contains(lock, dotNotation) {
		t.Errorf("Lock file must NOT use dot notation for hyphenated service IDs; found %q\nLock content:\n%s", dotNotation, lock)
	}
}
