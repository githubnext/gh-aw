package workflow

import (
	"strings"
	"testing"
)

// TestCustomMCPExpressionExtraction verifies that expressions in custom MCP servers
// are extracted and replaced with environment variable references
func TestCustomMCPExpressionExtraction(t *testing.T) {
	tests := []struct {
		name         string
		toolConfig   map[string]any
		expectEnvVar bool
		expectCount  int
	}{
		{
			name: "MCP server with expression in args",
			toolConfig: map[string]any{
				"command": "uvx",
				"args": []any{
					"--project",
					"${{ github.workspace }}",
				},
			},
			expectEnvVar: true,
			expectCount:  1,
		},
		{
			name: "MCP server with expression in env",
			toolConfig: map[string]any{
				"command": "node",
				"env": map[string]any{
					"API_KEY": "${{ secrets.MY_API_KEY }}",
				},
			},
			expectEnvVar: true,
			expectCount:  1,
		},
		{
			name: "MCP server with multiple expressions",
			toolConfig: map[string]any{
				"command": "python",
				"args": []any{
					"--workspace",
					"${{ github.workspace }}",
				},
				"env": map[string]any{
					"REPO":  "${{ github.repository }}",
					"TOKEN": "${{ secrets.GITHUB_TOKEN }}",
				},
			},
			expectEnvVar: true,
			expectCount:  3,
		},
		{
			name: "MCP server without expressions",
			toolConfig: map[string]any{
				"command": "docker",
				"args": []any{
					"run",
					"-i",
				},
			},
			expectEnvVar: false,
			expectCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expressions := extractExpressionsFromCustomMCP(tt.toolConfig)

			if tt.expectEnvVar && len(expressions) == 0 {
				t.Errorf("Expected to extract expressions, but got none")
			}

			if !tt.expectEnvVar && len(expressions) > 0 {
				t.Errorf("Expected no expressions, but got %d", len(expressions))
			}

			if len(expressions) != tt.expectCount {
				t.Errorf("Expected %d expressions, got %d", tt.expectCount, len(expressions))
			}

			// Verify that extracted expressions follow the expected format
			for envVar, originalExpr := range expressions {
				if !strings.HasPrefix(envVar, "GH_AW_EXPR_") {
					t.Errorf("Expected env var to start with GH_AW_EXPR_, got: %s", envVar)
				}
				if !strings.Contains(originalExpr, "${{") {
					t.Errorf("Expected original expression to contain ${{, got: %s", originalExpr)
				}
			}
		})
	}
}

// TestReplaceExpressionsInString verifies that template expressions are replaced
// with environment variable references
func TestReplaceExpressionsInString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple expression",
			input:    "${{ github.workspace }}",
			expected: "${GH_AW_EXPR_",
		},
		{
			name:     "Multiple expressions",
			input:    "${{ github.repository }} - ${{ github.workspace }}",
			expected: "${GH_AW_EXPR_",
		},
		{
			name:     "No expressions",
			input:    "/home/user/project",
			expected: "/home/user/project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := replaceExpressionsInString(tt.input)

			if strings.Contains(tt.input, "${{") {
				// If input had expressions, result should have env var references
				if !strings.Contains(result, tt.expected) {
					t.Errorf("Expected result to contain %s, got: %s", tt.expected, result)
				}
				// And should NOT have template expressions
				if strings.Contains(result, "${{") {
					t.Errorf("Expected result to not contain ${}, got: %s", result)
				}
			} else {
				// If input had no expressions, result should be unchanged
				if result != tt.expected {
					t.Errorf("Expected %s, got: %s", tt.expected, result)
				}
			}
		})
	}
}
