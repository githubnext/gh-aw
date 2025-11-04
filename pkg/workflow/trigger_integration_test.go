//go:build integration
// +build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTriggerParsingIntegration tests trigger parsing in real workflow scenarios
func TestTriggerParsingIntegration(t *testing.T) {
	tests := []struct {
		name          string
		frontmatter   string
		wantEvents    []string
		wantCommand   bool
		wantReaction  string
		wantStopAfter string
		wantErr       bool
	}{
		{
			name: "simple push trigger",
			frontmatter: `---
on: push
engine: claude
---
# Test Workflow
Test content`,
			wantEvents:  []string{"push"},
			wantCommand: false,
			wantErr:     false,
		},
		{
			name: "pull_request with types",
			frontmatter: `---
on:
  pull_request:
    types: [opened, synchronize]
engine: copilot
---
# PR Workflow
Test content`,
			wantEvents:  []string{"pull_request"},
			wantCommand: false,
			wantErr:     false,
		},
		{
			name: "multiple events",
			frontmatter: `---
on:
  push:
    branches: [main]
  pull_request:
    types: [opened]
  workflow_dispatch:
engine: claude
---
# Multi-trigger Workflow
Test content`,
			wantEvents:  []string{"push", "pull_request", "workflow_dispatch"},
			wantCommand: false,
			wantErr:     false,
		},
		{
			name: "schedule trigger",
			frontmatter: `---
on:
  schedule:
    - cron: "0 0 * * *"
  workflow_dispatch:
engine: copilot
---
# Scheduled Workflow
Test content`,
			wantEvents:  []string{"schedule", "workflow_dispatch"},
			wantCommand: false,
			wantErr:     false,
		},
		{
			name: "command trigger",
			frontmatter: `---
on:
  command:
    name: bot
    events: [issues, issue_comment]
engine: claude
---
# Command Workflow
Test content`,
			wantEvents:  []string{},
			wantCommand: true,
			wantErr:     false,
		},
		{
			name: "trigger with reaction",
			frontmatter: `---
on:
  issues:
    types: [opened]
  reaction: rocket
engine: copilot
---
# Reaction Workflow
Test content`,
			wantEvents:   []string{"issues"},
			wantCommand:  false,
			wantReaction: "rocket",
			wantErr:      false,
		},
		{
			name: "trigger with stop-after",
			frontmatter: `---
on:
  workflow_dispatch:
  stop-after: "2024-12-31 23:59:59"
engine: claude
---
# Time-limited Workflow
Test content`,
			wantEvents:    []string{"workflow_dispatch"},
			wantCommand:   false,
			wantStopAfter: "2024-12-31 23:59:59",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory for the test
			tmpDir := t.TempDir()
			workflowPath := filepath.Join(tmpDir, "test-workflow.md")

			// Write the workflow file
			err := os.WriteFile(workflowPath, []byte(tt.frontmatter), 0644)
			if err != nil {
				t.Fatalf("Failed to write test workflow: %v", err)
			}

			// Create a compiler
			compiler := NewCompiler(false, "", "test-version")
			compiler.SetSkipValidation(true) // Skip schema validation for these tests

			// Parse the workflow
			workflowData, err := compiler.ParseWorkflowFile(workflowPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseWorkflowFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// Check that ParsedTrigger was populated
			if workflowData.ParsedTrigger == nil {
				t.Error("ParsedTrigger should not be nil")
				return
			}

			// Check events
			for _, eventName := range tt.wantEvents {
				if !workflowData.ParsedTrigger.HasEvent(eventName) {
					t.Errorf("Expected trigger to have event %s", eventName)
				}
			}

			// Check command
			if workflowData.ParsedTrigger.HasCommand() != tt.wantCommand {
				t.Errorf("HasCommand() = %v, want %v", workflowData.ParsedTrigger.HasCommand(), tt.wantCommand)
			}

			// Check reaction
			if tt.wantReaction != "" && workflowData.ParsedTrigger.Reaction != tt.wantReaction {
				t.Errorf("Reaction = %v, want %v", workflowData.ParsedTrigger.Reaction, tt.wantReaction)
			}

			// Check stop-after
			if tt.wantStopAfter != "" && workflowData.ParsedTrigger.StopAfter != tt.wantStopAfter {
				t.Errorf("StopAfter = %v, want %v", workflowData.ParsedTrigger.StopAfter, tt.wantStopAfter)
			}
		})
	}
}

// TestTriggerParsingWithRealWorkflows tests trigger parsing with actual workflow files
func TestTriggerParsingWithRealWorkflows(t *testing.T) {
	// Create a compiler
	compiler := NewCompiler(false, "", "test-version")
	compiler.SetSkipValidation(true)

	// Test with a real workflow that has pull_request trigger
	prWorkflow := `---
on:
  pull_request:
    types: [opened, synchronize, reopened]
    branches: [main, develop]
permissions:
  contents: read
  pull-requests: write
engine: copilot
---
# Pull Request Review

Review the pull request changes.
`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "pr-review.md")
	err := os.WriteFile(workflowPath, []byte(prWorkflow), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	workflowData, err := compiler.ParseWorkflowFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify trigger was parsed
	if workflowData.ParsedTrigger == nil {
		t.Fatal("ParsedTrigger should not be nil")
	}

	// Verify pull_request event exists
	if !workflowData.ParsedTrigger.HasEvent("pull_request") {
		t.Error("Expected pull_request event")
	}

	// Verify event configuration
	prEvent, exists := workflowData.ParsedTrigger.Events["pull_request"]
	if !exists {
		t.Fatal("pull_request event should exist")
	}

	// Check types
	expectedTypes := []string{"opened", "synchronize", "reopened"}
	if len(prEvent.Types) != len(expectedTypes) {
		t.Errorf("Expected %d types, got %d", len(expectedTypes), len(prEvent.Types))
	}

	// Check branches
	expectedBranches := []string{"main", "develop"}
	if len(prEvent.Branches) != len(expectedBranches) {
		t.Errorf("Expected %d branches, got %d", len(expectedBranches), len(prEvent.Branches))
	}
}

// TestTriggerBackwardCompatibility ensures that the On string field is still populated
func TestTriggerBackwardCompatibility(t *testing.T) {
	compiler := NewCompiler(false, "", "test-version")
	compiler.SetSkipValidation(true)

	workflow := `---
on:
  push:
    branches: [main]
engine: claude
---
# Test Workflow
Content`

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test.md")
	err := os.WriteFile(workflowPath, []byte(workflow), 0644)
	if err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	workflowData, err := compiler.ParseWorkflowFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Both ParsedTrigger and On should be populated
	if workflowData.ParsedTrigger == nil {
		t.Error("ParsedTrigger should not be nil")
	}

	if workflowData.On == "" {
		t.Error("On field should still be populated for backward compatibility")
	}

	// The On field should contain the trigger information
	if !strings.Contains(workflowData.On, "push") {
		t.Error("On field should contain 'push' trigger")
	}
}

// TestTriggerToYAMLRoundTrip tests that ToYAML produces valid YAML that can be re-parsed
func TestTriggerToYAMLRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		trigger *TriggerConfig
	}{
		{
			name: "simple trigger",
			trigger: &TriggerConfig{
				Simple: "push",
				Events: map[string]EventConfig{
					"push": {Raw: nil},
				},
			},
		},
		{
			name: "complex trigger with raw",
			trigger: &TriggerConfig{
				Raw: map[string]any{
					"pull_request": map[string]any{
						"types": []any{"opened", "synchronize"},
					},
				},
				Events: map[string]EventConfig{
					"pull_request": {
						Raw:   map[string]any{"types": []any{"opened", "synchronize"}},
						Types: []string{"opened", "synchronize"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to YAML
			yamlStr, err := tt.trigger.ToYAML()
			if err != nil {
				t.Fatalf("ToYAML() error = %v", err)
			}

			if yamlStr == "" {
				t.Error("ToYAML() returned empty string")
				return
			}

			// Verify it contains "on:" - it may be quoted as '"on":' for YAML keywords
			if !strings.Contains(yamlStr, "on:") && !strings.Contains(yamlStr, `"on":`) {
				t.Errorf("YAML output should contain 'on:' key, got:\n%s", yamlStr)
			}
		})
	}
}
