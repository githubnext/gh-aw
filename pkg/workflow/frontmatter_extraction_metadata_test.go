//go:build !integration

package workflow

import (
	"testing"
)

func TestExtractFeatures(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name        string
		frontmatter map[string]any
		expected    map[string]any
	}{
		{
			name: "valid features map with boolean values",
			frontmatter: map[string]any{
				"features": map[string]any{
					"feature1": true,
					"feature2": false,
					"feature3": true,
				},
			},
			expected: map[string]any{
				"feature1": true,
				"feature2": false,
				"feature3": true,
			},
		},
		{
			name:        "features key not present",
			frontmatter: map[string]any{"other": "value"},
			expected:    nil,
		},
		{
			name:        "empty frontmatter",
			frontmatter: map[string]any{},
			expected:    nil,
		},
		{
			name: "features is not a map",
			frontmatter: map[string]any{
				"features": "not a map",
			},
			expected: nil,
		},
		{
			name: "features map with mixed value types",
			frontmatter: map[string]any{
				"features": map[string]any{
					"feature1":   true,
					"feature2":   "string value",
					"action-tag": "2d4c6ce24c55704d72ec674d1f5c357831435180",
				},
			},
			expected: map[string]any{
				"feature1":   true,
				"feature2":   "string value",
				"action-tag": "2d4c6ce24c55704d72ec674d1f5c357831435180",
			},
		},
		{
			name: "empty features map",
			frontmatter: map[string]any{
				"features": map[string]any{},
			},
			expected: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compiler.extractFeatures(tt.frontmatter)

			if result == nil && tt.expected == nil {
				return
			}

			if (result == nil) != (tt.expected == nil) {
				t.Errorf("extractFeatures() = %v, want %v", result, tt.expected)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("extractFeatures() length = %d, want %d", len(result), len(tt.expected))
				return
			}

			for key, expectedVal := range tt.expected {
				if actualVal, ok := result[key]; !ok || actualVal != expectedVal {
					t.Errorf("extractFeatures()[%q] = %v, want %v", key, actualVal, expectedVal)
				}
			}
		})
	}
}

func TestExtractToolsStartupTimeout(t *testing.T) {
	compiler := &Compiler{}

	tests := []struct {
		name        string
		tools       map[string]any
		expected    string
		shouldError bool
	}{
		{
			name: "int timeout",
			tools: map[string]any{
				"startup-timeout": 30,
			},
			expected: "30",
		},
		{
			name: "int64 timeout",
			tools: map[string]any{
				"startup-timeout": int64(60),
			},
			expected: "60",
		},
		{
			name: "uint timeout",
			tools: map[string]any{
				"startup-timeout": uint(45),
			},
			expected: "45",
		},
		{
			name: "uint64 timeout",
			tools: map[string]any{
				"startup-timeout": uint64(90),
			},
			expected: "90",
		},
		{
			name: "float64 timeout",
			tools: map[string]any{
				"startup-timeout": 120.0,
			},
			expected: "120",
		},
		{
			name:     "nil tools",
			tools:    nil,
			expected: "",
		},
		{
			name:     "empty tools map",
			tools:    map[string]any{},
			expected: "",
		},
		{
			name: "startup-timeout not present",
			tools: map[string]any{
				"other-field": "value",
			},
			expected: "",
		},
		{
			name: "invalid type (string) - should error",
			tools: map[string]any{
				"startup-timeout": "not a number",
			},
			expected:    "",
			shouldError: true,
		},
		{
			name: "invalid type (array) - should error",
			tools: map[string]any{
				"startup-timeout": []int{1, 2, 3},
			},
			expected:    "",
			shouldError: true,
		},
		{
			name: "zero timeout - should fail validation",
			tools: map[string]any{
				"startup-timeout": 0,
			},
			expected:    "",
			shouldError: true,
		},
		{
			name: "negative timeout - should fail validation",
			tools: map[string]any{
				"startup-timeout": -5,
			},
			expected:    "",
			shouldError: true,
		},
		{
			name: "minimum valid timeout (1)",
			tools: map[string]any{
				"startup-timeout": 1,
			},
			expected: "1",
		},
		{
			name: "expression - should be accepted",
			tools: map[string]any{
				"startup-timeout": "${{ inputs.startup-timeout }}",
			},
			expected: "${{ inputs.startup-timeout }}",
		},
		{
			name: "expression with vars - should be accepted",
			tools: map[string]any{
				"startup-timeout": "${{ vars.STARTUP_TIMEOUT }}",
			},
			expected: "${{ vars.STARTUP_TIMEOUT }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compiler.extractToolsStartupTimeout(tt.tools)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if result != tt.expected {
					t.Errorf("extractToolsStartupTimeout() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

// TestBuildLocalWorkflowSourceURL verifies that buildLocalWorkflowSourceURL derives
// a correct GitHub URL for a local workflow markdown file.
func TestBuildLocalWorkflowSourceURL(t *testing.T) {
	tests := []struct {
		name         string
		markdownPath string
		expected     string
	}{
		{
			name:         "empty path returns empty string",
			markdownPath: "",
			expected:     "",
		},
		{
			name:         "absolute path with .github workflows directory",
			markdownPath: "/home/runner/work/repo/.github/workflows/linter-miner.md",
			expected:     "${{ github.server_url }}/${{ github.repository }}/blob/${{ github.ref_name }}/.github/workflows/linter-miner.md",
		},
		{
			name:         "relative path starting with .github",
			markdownPath: ".github/workflows/my-workflow.md",
			expected:     "${{ github.server_url }}/${{ github.repository }}/blob/${{ github.ref_name }}/.github/workflows/my-workflow.md",
		},
		{
			name:         "path where repo is named .github (uses last .github dir)",
			markdownPath: "/root/.github/.github/workflows/file.md",
			expected:     "${{ github.server_url }}/${{ github.repository }}/blob/${{ github.ref_name }}/.github/workflows/file.md",
		},
		{
			name:         "temp path returns empty string",
			markdownPath: "/tmp/test.md",
			expected:     "",
		},
		{
			name:         "path not containing .github returns empty string",
			markdownPath: "/home/user/custom-dir/workflow.md",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildLocalWorkflowSourceURL(tt.markdownPath)
			if result != tt.expected {
				t.Errorf("buildLocalWorkflowSourceURL(%q) = %q, want %q", tt.markdownPath, result, tt.expected)
			}
		})
	}
}
