package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
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

	compiler := NewCompiler(false, "", "test")

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

	// Verify that the JavaScript uses addRaw to build the summary
	if strings.Count(lockContent, ".addRaw(") < 2 {
		t.Error("Expected at least 2 '.addRaw(' calls in JavaScript code for summary building")
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

	compiler := NewCompiler(false, "", "test")

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

	// Verify that the aw_info.json file is still created and logged to console
	if !strings.Contains(lockContent, "aw_info.json") {
		t.Error("Expected 'aw_info.json' to be created")
	}

	if !strings.Contains(lockContent, "console.log('Generated aw_info.json at:', tmpPath);") {
		t.Error("Expected console.log output for aw_info.json")
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
engine:
  id: claude
  model: claude-sonnet-4-20250514
---

# Test Claude Workflow Overview

This workflow tests the workflow overview for Claude engine.
`,
			expectEngineID:       "claude",
			expectEngineName:     "Anthropic Claude Code",
			expectModel:          "claude-sonnet-4-20250514",
			expectFirewall:       false,
			expectAllowedDomains: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(tmpDir, tt.name+".md")
			if err := os.WriteFile(testFile, []byte(tt.workflowContent), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "test")

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

			// Verify that the "Generate workflow overview" step exists
			if !strings.Contains(lockContent, "- name: Generate workflow overview") {
				t.Error("Expected 'Generate workflow overview' step")
			}

			// Verify engine ID is present
			if !strings.Contains(lockContent, "engine_id: \""+tt.expectEngineID+"\"") {
				t.Errorf("Expected engine_id: %q", tt.expectEngineID)
			}

			// Verify engine name is present
			if !strings.Contains(lockContent, "engine_name: \""+tt.expectEngineName+"\"") {
				t.Errorf("Expected engine_name: %q", tt.expectEngineName)
			}

			// Verify model is present
			if !strings.Contains(lockContent, "model: \""+tt.expectModel+"\"") {
				t.Errorf("Expected model: %q", tt.expectModel)
			}

			// Verify firewall status
			expectedFirewall := "false"
			if tt.expectFirewall {
				expectedFirewall = "true"
			}
			if !strings.Contains(lockContent, "firewall_enabled: "+expectedFirewall) {
				t.Errorf("Expected firewall_enabled: %s", expectedFirewall)
			}

			// Verify allowed domains if specified
			if len(tt.expectAllowedDomains) > 0 {
				for _, domain := range tt.expectAllowedDomains {
					if !strings.Contains(lockContent, domain) {
						t.Errorf("Expected allowed domain: %q", domain)
					}
				}
			}

			// Verify step runs before "Create prompt"
			overviewIdx := strings.Index(lockContent, "- name: Generate workflow overview")
			promptIdx := strings.Index(lockContent, "- name: Create prompt")
			if overviewIdx >= promptIdx {
				t.Error("Expected 'Generate workflow overview' step to run BEFORE 'Create prompt' step")
			}

			// Verify HTML details/summary format
			if !strings.Contains(lockContent, "<details>") {
				t.Error("Expected HTML <details> tag for collapsible summary")
			}
			if !strings.Contains(lockContent, "<summary>🤖 Agentic Workflow Run Overview</summary>") {
				t.Error("Expected HTML <summary> tag with workflow overview title")
			}
			if !strings.Contains(lockContent, "</details>") {
				t.Error("Expected HTML </details> closing tag")
			}

			t.Logf("✓ Workflow overview step correctly generated for %s", tt.name)
		})
	}
}
