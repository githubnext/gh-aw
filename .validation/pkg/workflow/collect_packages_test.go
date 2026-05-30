//go:build !integration

package workflow

import (
	"reflect"
	"strings"
	"testing"
)

// TestCollectPackagesFromWorkflow_EmptyWorkflow tests that an empty workflow returns no packages
func TestCollectPackagesFromWorkflow_EmptyWorkflow(t *testing.T) {
	workflowData := &WorkflowData{}

	extractor := func(s string) []string {
		return []string{}
	}

	packages := collectPackagesFromWorkflow(workflowData, extractor, "")

	if len(packages) != 0 {
		t.Errorf("Expected no packages, got %v", packages)
	}
}

// TestCollectPackagesFromWorkflow_CustomSteps tests package extraction from custom steps
func TestCollectPackagesFromWorkflow_CustomSteps(t *testing.T) {
	tests := []struct {
		name        string
		customSteps string
		extractor   func(string) []string
		toolCommand string
		expected    []string
	}{
		{
			name:        "Single package in custom steps",
			customSteps: "npm install axios",
			extractor: func(s string) []string {
				return []string{"axios"}
			},
			toolCommand: "",
			expected:    []string{"axios"},
		},
		{
			name:        "Multiple packages in custom steps",
			customSteps: "npm install axios lodash",
			extractor: func(s string) []string {
				return []string{"axios", "lodash"}
			},
			toolCommand: "",
			expected:    []string{"axios", "lodash"},
		},
		{
			name:        "Empty custom steps",
			customSteps: "",
			extractor: func(s string) []string {
				return []string{}
			},
			toolCommand: "",
			expected:    []string{},
		},
		{
			name:        "Duplicate packages in custom steps",
			customSteps: "npm install axios axios",
			extractor: func(s string) []string {
				return []string{"axios", "axios"}
			},
			toolCommand: "",
			expected:    []string{"axios"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				CustomSteps: tt.customSteps,
			}

			packages := collectPackagesFromWorkflow(workflowData, tt.extractor, tt.toolCommand)

			if len(packages) != len(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, packages)
				return
			}
			if len(packages) > 0 && !reflect.DeepEqual(packages, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, packages)
			}
		})
	}
}

// TestCollectPackagesFromWorkflow_MCPConfig tests package extraction from MCP server configurations
func TestCollectPackagesFromWorkflow_MCPConfig(t *testing.T) {
	tests := []struct {
		name        string
		tools       map[string]any
		toolCommand string
		extractor   func(string) []string
		expected    []string
	}{
		{
			name: "Structured MCP config with matching command",
			tools: map[string]any{
				"server1": map[string]any{
					"command": "npx",
					"args":    []any{"-y", "axios"},
				},
			},
			toolCommand: "npx",
			extractor: func(s string) []string {
				return []string{}
			},
			expected: []string{"axios"},
		},
		{
			name: "Structured MCP config with flags before package",
			tools: map[string]any{
				"server1": map[string]any{
					"command": "npm",
					"args":    []any{"install", "-g", "typescript"},
				},
			},
			toolCommand: "npm",
			extractor: func(s string) []string {
				return []string{}
			},
			expected: []string{"install"},
		},
		{
			name: "Structured MCP config with non-matching command",
			tools: map[string]any{
				"server1": map[string]any{
					"command": "pip",
					"args":    []any{"install", "flask"},
				},
			},
			toolCommand: "npm",
			extractor: func(s string) []string {
				return []string{}
			},
			expected: []string{},
		},
		{
			name: "String-format MCP tool",
			tools: map[string]any{
				"server1": "npx -y axios",
			},
			toolCommand: "npx",
			extractor: func(s string) []string {
				if s == "npx -y axios" {
					return []string{"axios"}
				}
				return []string{}
			},
			expected: []string{"axios"},
		},
		{
			name: "Empty tool command",
			tools: map[string]any{
				"server1": map[string]any{
					"command": "npx",
					"args":    []any{"-y", "axios"},
				},
			},
			toolCommand: "",
			extractor: func(s string) []string {
				return []string{}
			},
			expected: []string{},
		},
		{
			name: "Args with only flags",
			tools: map[string]any{
				"server1": map[string]any{
					"command": "npx",
					"args":    []any{"-y", "--verbose"},
				},
			},
			toolCommand: "npx",
			extractor: func(s string) []string {
				return []string{}
			},
			expected: []string{},
		},
		{
			name: "Args with non-string values",
			tools: map[string]any{
				"server1": map[string]any{
					"command": "npm",
					"args":    []any{123, 456},
				},
			},
			toolCommand: "npm",
			extractor: func(s string) []string {
				return []string{}
			},
			expected: []string{},
		},
		{
			name:        "Nil tools map",
			tools:       nil,
			toolCommand: "npm",
			extractor: func(s string) []string {
				return []string{}
			},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Tools: tt.tools,
			}

			packages := collectPackagesFromWorkflow(workflowData, tt.extractor, tt.toolCommand)

			if len(packages) != len(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, packages)
				return
			}
			if len(packages) > 0 && !reflect.DeepEqual(packages, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, packages)
			}
		})
	}
}

// TestCollectPackagesFromWorkflow_Combined tests package extraction from multiple sources
func TestCollectPackagesFromWorkflow_Combined(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		extractor    func(string) []string
		toolCommand  string
		expected     []string
	}{
		{
			name: "Packages from all sources with deduplication",
			workflowData: &WorkflowData{
				CustomSteps: "npm install axios\nnpm install lodash",
				Tools: map[string]any{
					"server1": map[string]any{
						"command": "npx",
						"args":    []any{"-y", "axios"}, // Duplicate
					},
				},
			},
			extractor: func(s string) []string {
				var result []string
				if strings.Contains(s, "npm install axios") {
					result = append(result, "axios")
				}
				if strings.Contains(s, "npm install lodash") {
					result = append(result, "lodash")
				}
				return result
			},
			toolCommand: "npx",
			expected:    []string{"axios", "lodash"},
		},
		{
			name: "Empty sources",
			workflowData: &WorkflowData{
				CustomSteps: "",
				Tools:       map[string]any{},
			},
			extractor: func(s string) []string {
				return []string{}
			},
			toolCommand: "",
			expected:    []string{},
		},
		{
			name: "Nil engine config",
			workflowData: &WorkflowData{
				CustomSteps:  "npm install axios",
				EngineConfig: nil,
				Tools:        nil,
			},
			extractor: func(s string) []string {
				return []string{"axios"}
			},
			toolCommand: "",
			expected:    []string{"axios"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packages := collectPackagesFromWorkflow(tt.workflowData, tt.extractor, tt.toolCommand)

			if len(packages) != len(tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, packages)
				return
			}
			if len(packages) > 0 && !reflect.DeepEqual(packages, tt.expected) {
				t.Errorf("Expected %v, got %v", tt.expected, packages)
			}
		})
	}
}
