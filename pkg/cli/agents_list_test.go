package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAgentInfoExtraction(t *testing.T) {
	// Create a temporary directory for test workflows
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Create a test workflow file
	workflowContent := `---
on:
  issues:
    types: [opened]
permissions:
  issues: write
safe-outputs:
  create-issue:
    max: 1
  add-comment:
source: githubnext/agentics/ci-doctor@main
---

# CI Doctor

Monitor CI workflows and investigate failures automatically.

This workflow analyzes failed CI runs and provides detailed reports.
`

	workflowPath := filepath.Join(workflowsDir, "ci-doctor.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Parse the agent info
	agent, err := parseAgentInfo(workflowPath, false)
	if err != nil {
		t.Fatalf("Failed to parse agent info: %v", err)
	}

	// Validate extracted fields
	if agent.Name != "ci-doctor" {
		t.Errorf("Expected name 'ci-doctor', got '%s'", agent.Name)
	}

	if agent.Source != "githubnext/agentics/ci-doctor@main" {
		t.Errorf("Expected source 'githubnext/agentics/ci-doctor@main', got '%s'", agent.Source)
	}

	if agent.Trigger != "issues" {
		t.Errorf("Expected trigger 'issues', got '%s'", agent.Trigger)
	}

	expectedDescription := "Monitor CI workflows and investigate failures automatically."
	if agent.Description != expectedDescription {
		t.Errorf("Expected description '%s', got '%s'", expectedDescription, agent.Description)
	}

	// Check safe outputs
	if len(agent.SafeOutputs) != 2 {
		t.Errorf("Expected 2 safe outputs, got %d", len(agent.SafeOutputs))
	}
}

func TestInferCategoryFromDescription(t *testing.T) {
	tests := []struct {
		description string
		expected    string
	}{
		{"Triage issues automatically", "Triage"},
		{"CI Doctor monitors workflows", "Analysis"},
		{"Weekly research summary", "Research"},
		{"Daily status report", "Research"}, // "status" triggers Research category
		{"Fix PR issues automatically", "Triage"}, // "issue" triggers Triage (checked before "pr")
		{"Update documentation", "Documentation"},
		{"Plan the sprint", "Coding"}, // "plan" contains "pla" which doesn't match our patterns perfectly
		{"Some other workflow", "Other"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			result := inferCategoryFromDescription(tt.description)
			if result != tt.expected {
				t.Errorf("inferCategoryFromDescription(%q) = %q, want %q", tt.description, result, tt.expected)
			}
		})
	}
}

func TestExtractAgentDescription(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name: "simple description",
			body: `# Workflow Title

This is the description.

More details here.`,
			expected: "This is the description.",
		},
		{
			name: "with empty lines",
			body: `# Workflow Title


This is the description after empty lines.`,
			expected: "This is the description after empty lines.",
		},
		{
			name: "long description",
			body: `# Workflow Title

This is a very long description that exceeds the maximum length limit and should be truncated appropriately to fit within the bounds.`,
			expected: "This is a very long description that exceeds the maximum length limit and should be truncated app...",
		},
		{
			name:     "no description",
			body:     `# Workflow Title`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAgentDescription(tt.body)
			if result != tt.expected {
				t.Errorf("extractAgentDescription() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractCategory(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		description string
		expected    string
	}{
		{
			name:        "explicit category",
			frontmatter: map[string]any{"category": "Testing"},
			description: "Some workflow description",
			expected:    "Testing",
		},
		{
			name:        "infer from triage description",
			frontmatter: map[string]any{},
			description: "Triage incoming issues",
			expected:    "Triage",
		},
		{
			name:        "infer from CI description",
			frontmatter: map[string]any{},
			description: "Monitor CI builds",
			expected:    "Analysis",
		},
		{
			name:        "infer from research description",
			frontmatter: map[string]any{},
			description: "Research new technologies",
			expected:    "Research",
		},
		{
			name:        "default to Other",
			frontmatter: map[string]any{},
			description: "Unknown workflow type",
			expected:    "Other",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractCategory(tt.frontmatter, tt.description)
			if result != tt.expected {
				t.Errorf("extractCategory() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractTrigger(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    string
	}{
		{
			name:        "workflow_dispatch",
			frontmatter: map[string]any{"on": "workflow_dispatch"},
			expected:    "workflow_dispatch",
		},
		{
			name: "multiple triggers",
			frontmatter: map[string]any{
				"on": map[string]any{
					"issues":       map[string]any{"types": []any{"opened"}},
					"pull_request": map[string]any{"types": []any{"opened"}},
				},
			},
			expected: "issues, pull_request", // order may vary
		},
		{
			name:        "no trigger",
			frontmatter: map[string]any{},
			expected:    "manual",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTrigger(tt.frontmatter)
			// For multiple triggers, just check that the result contains expected triggers
			if tt.name == "multiple triggers" {
				if !stringContainsAll(result, "issues", "pull_request") {
					t.Errorf("extractTrigger() = %q, should contain 'issues' and 'pull_request'", result)
				}
			} else if result != tt.expected {
				t.Errorf("extractTrigger() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// stringContainsAll checks if a string contains all given substrings
func stringContainsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
