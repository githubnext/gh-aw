//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestStepSummaryIncludesProcessedOutput(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "step-summary-test")

	// Test case with Claude engine
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
  pull-requests: read
tools:
  github:
    allowed: [list_issues]
engine: claude
features:
  dangerous-permissions-write: true
strict: false
safe-outputs:
  create-issue:
---

# Test Step Summary with Processed Output

This workflow tests that the step summary includes both JSONL and processed output.
`

	testFile := filepath.Join(tmpDir, "test-step-summary.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-step-summary.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify that the "Print sanitized agent output" step no longer exists (moved to JavaScript)
	if strings.Contains(lockContent, "- name: Print sanitized agent output") {
		t.Error("Did not expect 'Print sanitized agent output' step (should be in JavaScript now)")
	}

	// Verify that the threat detection setup requires the .cjs file
	// (The .addRaw call for threat detection is now in setup_threat_detection.cjs, not inline)
	if strings.Contains(lockContent, "setup_threat_detection.cjs") {
		t.Log("✓ Threat detection setup correctly requires .cjs file")
	}

	t.Log("Step summary correctly includes processed output sections")
}

func TestStepSummaryIncludesAgenticRunInfo(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "agentic-run-info-test")

	// Test case with Claude engine including extended configuration
	testContent := `---
on: push
permissions:
  contents: read
  issues: write
  pull-requests: read
strict: false
features:
  dangerous-permissions-write: true
tools:
  github:
    allowed: [list_issues]
engine:
  id: claude
  model: claude-3-5-sonnet-20241022
  version: beta
---

# Test Agentic Run Info Step Summary

This workflow tests that the step summary includes agentic run information.
`

	testFile := filepath.Join(tmpDir, "test-agentic-run-info.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler()

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Unexpected error compiling workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-agentic-run-info.lock.yml")
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read generated lock file: %v", err)
	}

	lockContent := string(content)

	// Verify that the "Generate agentic run info" step exists
	if !strings.Contains(lockContent, "- name: Generate agentic run info") {
		t.Error("Expected 'Generate agentic run info' step")
	}

	// Verify that the step does NOT include the "Agentic Run Information" section in step summary
	if strings.Contains(lockContent, "## Agentic Run Information") {
		t.Error("Did not expect '## Agentic Run Information' section in step summary (it should only be in action logs)")
	}

	// Verify that the aw_info.json file is still referenced in the workflow
	if !strings.Contains(lockContent, "aw_info.json") {
		t.Error("Expected 'aw_info.json' to be referenced in the workflow")
	}

	// Verify that the generate_aw_info.cjs helper is invoked from the step
	if !strings.Contains(lockContent, "require('/opt/gh-aw/actions/generate_aw_info.cjs')") {
		t.Error("Expected generate_aw_info.cjs require call in 'Generate agentic run info' step")
	}

	t.Log("Step correctly creates aw_info.json without adding to step summary")
}

func TestStepSummaryIncludesWorkflowOverview(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "workflow-overview-test")

	tests := []struct {
		name                 string
		workflowContent      string
		expectEngineID       string
		expectEngineName     string
		expectModel          string
		expectFirewall       bool
		expectAllowedDomains []string
	}{
		{
			name: "copilot engine with firewall",
			workflowContent: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
network:
  allowed:
    - defaults
    - node
  firewall: true
---

# Test Workflow Overview

This workflow tests the workflow overview step summary.
`,
			expectEngineID:       "copilot",
			expectEngineName:     "GitHub Copilot CLI",
			expectModel:          "",
			expectFirewall:       true,
			expectAllowedDomains: []string{"defaults", "node"},
		},
		{
			name: "claude engine with model",
			workflowContent: `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
strict: false
features:
  dangerous-permissions-write: true
engine:
  id: claude
  model: claude-sonnet-4-20250514
---

# Test Claude Workflow Overview

This workflow tests the workflow overview for Claude engine.
`,
			expectEngineID:       "claude",
			expectEngineName:     "Claude Code",
			expectModel:          "claude-sonnet-4-20250514",
			expectFirewall:       true, // Claude now has firewall enabled by default
			expectAllowedDomains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".md")
			if err := os.WriteFile(testFile, []byte(tt.workflowContent), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler()

			// Compile the workflow
			if err := compiler.CompileWorkflow(testFile); err != nil {
				t.Fatalf("Unexpected error compiling workflow: %v", err)
			}

			// Read the generated lock file
			lockFile := filepath.Join(tmpDir, tt.name+".lock.yml")
			content, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read generated lock file: %v", err)
			}

			lockContent := string(content)

			// Verify that the "Generate agentic run info" step exists and contains network config
			if !strings.Contains(lockContent, "- name: Generate agentic run info") {
				t.Error("Expected 'Generate agentic run info' step")
			}

			// Verify workflow overview is merged into the generate_aw_info step (no separate step)
			if strings.Contains(lockContent, "- name: Generate workflow overview") {
				t.Error("Expected no separate 'Generate workflow overview' step (should be merged into 'Generate agentic run info')")
			}

			// Verify workflow overview call is present in the generate_aw_info step
			if !strings.Contains(lockContent, "require('/opt/gh-aw/actions/generate_aw_info.cjs')") {
				t.Error("Expected generate_aw_info.cjs require call inside 'Generate agentic run info' step")
			}

			// Verify engine ID is set as an env var in the generate_aw_info step
			if !strings.Contains(lockContent, "GH_AW_INFO_ENGINE_ID: \""+tt.expectEngineID+"\"") {
				t.Errorf("Expected GH_AW_INFO_ENGINE_ID: %q in generate_aw_info step", tt.expectEngineID)
			}

			// Verify engine name is set as an env var in the generate_aw_info step
			if !strings.Contains(lockContent, "GH_AW_INFO_ENGINE_NAME: \""+tt.expectEngineName+"\"") {
				t.Errorf("Expected GH_AW_INFO_ENGINE_NAME: %q in generate_aw_info step", tt.expectEngineName)
			}

			// Verify model is set as an env var in the generate_aw_info step
			if tt.expectModel == "" {
				// For empty model, check for the complete vars expression (with fallback)
				if !strings.Contains(lockContent, "GH_AW_INFO_MODEL: ${{ vars.GH_AW_MODEL_AGENT_COPILOT || '' }}") &&
					!strings.Contains(lockContent, "GH_AW_INFO_MODEL: ${{ vars.GH_AW_MODEL_DETECTION_COPILOT || '' }}") {
					t.Errorf("Expected GH_AW_INFO_MODEL to use vars expression in generate_aw_info step")
				}
			} else {
				// For non-empty model, check for the literal value
				if !strings.Contains(lockContent, "GH_AW_INFO_MODEL: \""+tt.expectModel+"\"") {
					t.Errorf("Expected GH_AW_INFO_MODEL: %q in generate_aw_info step", tt.expectModel)
				}
			}

			// Verify firewall status is set as an env var in the generate_aw_info step
			expectedFirewall := "false"
			if tt.expectFirewall {
				expectedFirewall = "true"
			}
			if !strings.Contains(lockContent, "GH_AW_INFO_FIREWALL_ENABLED: \""+expectedFirewall+"\"") {
				t.Errorf("Expected GH_AW_INFO_FIREWALL_ENABLED: %q in generate_aw_info step", expectedFirewall)
			}

			// Verify allowed domains if specified (in aw_info.json)
			if len(tt.expectAllowedDomains) > 0 {
				for _, domain := range tt.expectAllowedDomains {
					if !strings.Contains(lockContent, domain) {
						t.Errorf("Expected allowed domain: %q in aw_info.json", domain)
					}
				}
			}

			// Verify step runs before "Download activation artifact" (activation job appears before agent job in YAML)
			// Note: "Generate agentic run info" (which includes the overview) is in the activation job,
			// and "Download activation artifact" is in the agent job, which follows activation in the YAML.
			awInfoIdx := strings.Index(lockContent, "- name: Generate agentic run info")
			promptIdx := strings.Index(lockContent, "- name: Download activation artifact")
			if awInfoIdx >= promptIdx {
				t.Error("Expected 'Generate agentic run info' step to run BEFORE 'Download activation artifact' step")
			}

			// Note: HTML details/summary format is now in generate_workflow_overview.cjs
			// The compiled workflow will call the function via require
			// The actual HTML generation is tested in generate_workflow_overview.test.cjs

			t.Logf("✓ Workflow overview step correctly generated for %s", tt.name)
		})
	}
}
