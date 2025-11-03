package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaintAnalysis(t *testing.T) {
	// Create temporary directory with test workflows
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		t.Fatalf("Failed to create workflows directory: %v", err)
	}

	// Test workflow with agentic engine and edit tool (unsafe)
	agenticWorkflow := `---
on: issues
engine: claude
tools:
  edit:
---

# Test Agentic Workflow

Process issues and edit files.
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "agentic-edit.md"), []byte(agenticWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Test workflow with web-search and bash (highly unsafe)
	webSearchWorkflow := `---
on: workflow_dispatch
engine: copilot
tools:
  web-search:
  bash:
---

# Web Search with Bash

Search web and execute commands.
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "web-bash.md"), []byte(webSearchWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Test workflow with safe-outputs (should be safe)
	safeWorkflow := `---
on: issues
engine: claude
tools:
  github:
safe-outputs:
  create-issue:
---

# Safe Workflow

Process issues safely with safe-outputs.
`
	if err := os.WriteFile(filepath.Join(workflowsDir, "safe-workflow.md"), []byte(safeWorkflow), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Run taint analysis
	result, err := performTaintAnalysis(workflowsDir, false)
	if err != nil {
		t.Fatalf("Taint analysis failed: %v", err)
	}

	// Verify results
	if result.WorkflowCount != 3 {
		t.Errorf("Expected 3 workflows, got %d", result.WorkflowCount)
	}

	if len(result.Sources) == 0 {
		t.Error("Expected to find taint sources")
	}

	if len(result.Sinks) == 0 {
		t.Error("Expected to find taint sinks")
	}

	if len(result.UnsafePaths) == 0 {
		t.Error("Expected to find unsafe paths")
	}

	// Verify at least one unsafe path is detected
	foundUnsafe := false
	for _, path := range result.UnsafePaths {
		if path.IsUnsafe {
			foundUnsafe = true
			break
		}
	}

	if !foundUnsafe {
		t.Error("Expected to find at least one unsafe path")
	}
}

func TestIdentifyTaintSources(t *testing.T) {
	tests := []struct {
		name           string
		workflow       WorkflowInfo
		expectedCount  int
		expectedTypes  []string
	}{
		{
			name: "agentic workflow",
			workflow: WorkflowInfo{
				Name:      "test.md",
				IsAgentic: true,
				Engine:    "claude",
			},
			expectedCount: 1,
			expectedTypes: []string{"agentic_workflow"},
		},
		{
			name: "web-search tool",
			workflow: WorkflowInfo{
				Name:      "test.md",
				IsAgentic: true,
				Tools:     []string{"web-search"},
			},
			expectedCount: 2,
			expectedTypes: []string{"agentic_workflow", "web_search"},
		},
		{
			name: "user input detection",
			workflow: WorkflowInfo{
				Name:      "test.md",
				IsAgentic: true,
				Markdown:  "Process issue and pull request comments",
			},
			expectedCount: 2,
			expectedTypes: []string{"agentic_workflow", "user_input"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources := identifyTaintSources(tt.workflow, false)
			if len(sources) != tt.expectedCount {
				t.Errorf("Expected %d sources, got %d", tt.expectedCount, len(sources))
			}

			// Verify expected types are present
			typeMap := make(map[string]bool)
			for _, source := range sources {
				typeMap[source.Type] = true
			}

			for _, expectedType := range tt.expectedTypes {
				if !typeMap[expectedType] {
					t.Errorf("Expected to find source type %s", expectedType)
				}
			}
		})
	}
}

func TestIdentifyTaintSinks(t *testing.T) {
	tests := []struct {
		name           string
		workflow       WorkflowInfo
		expectedCount  int
		expectedTypes  []string
	}{
		{
			name: "edit tool",
			workflow: WorkflowInfo{
				Name:  "test.md",
				Tools: []string{"edit"},
			},
			expectedCount: 1,
			expectedTypes: []string{"file_write"},
		},
		{
			name: "bash tool",
			workflow: WorkflowInfo{
				Name:  "test.md",
				Tools: []string{"bash"},
			},
			expectedCount: 1,
			expectedTypes: []string{"command_execution"},
		},
		{
			name: "safe-outputs",
			workflow: WorkflowInfo{
				Name:        "test.md",
				HasSafeOuts: true,
			},
			expectedCount: 1,
			expectedTypes: []string{"github_api"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sinks := identifyTaintSinks(tt.workflow, false)
			if len(sinks) != tt.expectedCount {
				t.Errorf("Expected %d sinks, got %d", tt.expectedCount, len(sinks))
			}

			// Verify expected types are present
			typeMap := make(map[string]bool)
			for _, sink := range sinks {
				typeMap[sink.Type] = true
			}

			for _, expectedType := range tt.expectedTypes {
				if !typeMap[expectedType] {
					t.Errorf("Expected to find sink type %s", expectedType)
				}
			}
		})
	}
}

func TestEvaluatePathSafety(t *testing.T) {
	tests := []struct {
		name           string
		source         TaintSource
		sink           TaintSink
		expectedUnsafe bool
		reasonContains string
	}{
		{
			name: "web-search to command execution",
			source: TaintSource{
				Type: "web_search",
			},
			sink: TaintSink{
				Type: "command_execution",
			},
			expectedUnsafe: true,
			reasonContains: "code injection",
		},
		{
			name: "user input to command execution",
			source: TaintSource{
				Type: "user_input",
			},
			sink: TaintSink{
				Type: "command_execution",
			},
			expectedUnsafe: true,
			reasonContains: "code injection",
		},
		{
			name: "agentic workflow to file write",
			source: TaintSource{
				Type: "agentic_workflow",
			},
			sink: TaintSink{
				Type: "file_write",
			},
			expectedUnsafe: true,
			reasonContains: "content validation",
		},
		{
			name: "safe-outputs sink",
			source: TaintSource{
				Type: "agentic_workflow",
			},
			sink: TaintSink{
				Type: "github_api",
				Name: "test:safe-outputs",
			},
			expectedUnsafe: false,
			reasonContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isUnsafe, reason := evaluatePathSafety(tt.source, tt.sink)
			if isUnsafe != tt.expectedUnsafe {
				t.Errorf("Expected unsafe=%v, got %v", tt.expectedUnsafe, isUnsafe)
			}

			if tt.reasonContains != "" && reason == "" {
				t.Errorf("Expected reason to contain '%s', got empty", tt.reasonContains)
			}
		})
	}
}

func TestGenerateMermaidGraph(t *testing.T) {
	analysis := &TaintAnalysisResult{
		Sources: []TaintSource{
			{
				Type:         "agentic_workflow",
				Name:         "test.md",
				WorkflowFile: "/path/to/test.md",
				Severity:     "high",
			},
		},
		Sinks: []TaintSink{
			{
				Type:         "file_write",
				Name:         "test.md:edit",
				WorkflowFile: "/path/to/test.md",
				Severity:     "high",
			},
		},
		Paths: []TaintPath{
			{
				Source: TaintSource{
					Type: "agentic_workflow",
					Name: "test.md",
				},
				Sink: TaintSink{
					Type: "file_write",
					Name: "test.md:edit",
				},
				IsUnsafe: true,
			},
		},
		WorkflowCount: 1,
	}

	graph := generateMermaidGraph(analysis)

	// Verify graph contains expected elements
	if graph == "" {
		t.Error("Expected non-empty Mermaid graph")
	}

	// Check for mermaid code block
	if !stringContains(graph, "```mermaid") {
		t.Error("Expected Mermaid code block")
	}

	// Check for flowchart definition
	if !stringContains(graph, "flowchart TD") {
		t.Error("Expected flowchart definition")
	}

	// Check for source and sink nodes
	if !stringContains(graph, "Taint Sources") {
		t.Error("Expected taint sources section")
	}

	if !stringContains(graph, "Taint Sinks") {
		t.Error("Expected taint sinks section")
	}
}

func TestGenerateFullReport(t *testing.T) {
	analysis := &TaintAnalysisResult{
		WorkflowCount: 1,
		Sources: []TaintSource{
			{
				Type:         "agentic_workflow",
				Name:         "test.md",
				WorkflowFile: "/path/to/test.md",
				Severity:     "high",
			},
		},
		Sinks: []TaintSink{
			{
				Type:         "file_write",
				Name:         "test.md:edit",
				WorkflowFile: "/path/to/test.md",
				Severity:     "high",
			},
		},
		UnsafePaths: []TaintPath{
			{
				Source: TaintSource{
					Type: "agentic_workflow",
					Name: "test.md",
				},
				Sink: TaintSink{
					Type: "file_write",
					Name: "test.md:edit",
				},
				IsUnsafe: true,
				Reason:   "Test reason",
			},
		},
	}

	report := generateFullReport(analysis)

	// Verify report contains expected sections
	expectedSections := []string{
		"# Taint Flow Analysis Report",
		"## Executive Summary",
		"## Taint Sources",
		"## Taint Sinks",
		"## ⚠️ Unsafe Taint Paths",
		"## Taint Flow Visualization",
		"## General Recommendations",
	}

	for _, section := range expectedSections {
		if !stringContains(report, section) {
			t.Errorf("Expected report to contain section: %s", section)
		}
	}

	// Verify statistics
	if !stringContains(report, "Workflows Analyzed**: 1") {
		t.Error("Expected workflow count in report")
	}

	if !stringContains(report, "Unsafe Paths**: 1") {
		t.Error("Expected unsafe paths count in report")
	}
}

// Helper function to check if string contains substring
func stringContains(s, substr string) bool {
	return strings.Contains(s, substr)
}
