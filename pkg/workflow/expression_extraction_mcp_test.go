package workflow

import (
	"strings"
	"testing"
)

func TestExtractExpressionsFromTools(t *testing.T) {
	tests := []struct {
		name                string
		tools               map[string]any
		expectedExpressions []string
	}{
		{
			name: "Extract from args",
			tools: map[string]any{
				"custom-tool": map[string]any{
					"args": []any{
						"--project",
						"${{ github.workspace }}",
						"--repo",
						"${{ github.repository }}",
					},
				},
			},
			expectedExpressions: []string{
				"${{ github.workspace }}",
				"${{ github.repository }}",
			},
		},
		{
			name: "Extract from env",
			tools: map[string]any{
				"custom-tool": map[string]any{
					"env": map[string]any{
						"REPO": "${{ github.repository }}",
						"USER": "${{ github.actor }}",
					},
				},
			},
			expectedExpressions: []string{
				"${{ github.repository }}",
				"${{ github.actor }}",
			},
		},
		{
			name: "Extract from headers",
			tools: map[string]any{
				"http-tool": map[string]any{
					"headers": map[string]any{
						"Authorization": "Bearer ${{ secrets.TOKEN }}",
						"X-Repo":        "${{ github.repository }}",
					},
				},
			},
			expectedExpressions: []string{
				"${{ secrets.TOKEN }}",
				"${{ github.repository }}",
			},
		},
		{
			name: "Extract from multiple tools",
			tools: map[string]any{
				"tool1": map[string]any{
					"args": []any{"${{ github.workspace }}"},
				},
				"tool2": map[string]any{
					"env": map[string]any{
						"REPO": "${{ github.repository }}",
					},
				},
			},
			expectedExpressions: []string{
				"${{ github.workspace }}",
				"${{ github.repository }}",
			},
		},
		{
			name: "No expressions",
			tools: map[string]any{
				"tool": map[string]any{
					"args": []any{"--project", "/tmp/project"},
				},
			},
			expectedExpressions: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewExpressionExtractor()
			extractor.ExtractExpressionsFromTools(tt.tools)

			mappings := extractor.GetMappings()

			if len(mappings) != len(tt.expectedExpressions) {
				t.Errorf("Expected %d expressions, got %d", len(tt.expectedExpressions), len(mappings))
			}

			// Check that all expected expressions are present
			for _, expected := range tt.expectedExpressions {
				found := false
				for _, mapping := range mappings {
					if mapping.Original == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected expression %q not found in mappings", expected)
				}
			}
		})
	}
}

func TestReplaceExpressionsInTools(t *testing.T) {
	extractor := NewExpressionExtractor()

	tools := map[string]any{
		"serena": map[string]any{
			"args": []any{
				"--project",
				"${{ github.workspace }}",
			},
			"env": map[string]any{
				"REPO": "${{ github.repository }}",
			},
		},
	}

	// Extract expressions
	extractor.ExtractExpressionsFromTools(tools)

	// Replace expressions
	modifiedTools := extractor.ReplaceExpressionsInTools(tools)

	// Check that args are replaced
	serenaConfig := modifiedTools["serena"].(map[string]any)
	args := serenaConfig["args"].([]any)
	projectArg := args[1].(string)

	if strings.Contains(projectArg, "${{") {
		t.Errorf("Expected expression to be replaced, but got: %s", projectArg)
	}

	if !strings.HasPrefix(projectArg, "${GH_AW_EXPR_") {
		t.Errorf("Expected env var reference, but got: %s", projectArg)
	}

	// Check that env is replaced
	env := serenaConfig["env"].(map[string]any)
	repoEnv := env["REPO"].(string)

	if strings.Contains(repoEnv, "${{") {
		t.Errorf("Expected expression to be replaced, but got: %s", repoEnv)
	}

	if !strings.HasPrefix(repoEnv, "${GH_AW_EXPR_") {
		t.Errorf("Expected env var reference, but got: %s", repoEnv)
	}
}

func TestExpressionExtractionPreservesOriginalTools(t *testing.T) {
	originalTools := map[string]any{
		"tool": map[string]any{
			"args": []any{"${{ github.workspace }}"},
		},
	}

	extractor := NewExpressionExtractor()
	extractor.ExtractExpressionsFromTools(originalTools)
	modifiedTools := extractor.ReplaceExpressionsInTools(originalTools)

	// Original should still have the expression
	originalArgs := originalTools["tool"].(map[string]any)["args"].([]any)
	if !strings.Contains(originalArgs[0].(string), "${{") {
		t.Errorf("Original tools should not be modified")
	}

	// Modified should have env var reference
	modifiedArgs := modifiedTools["tool"].(map[string]any)["args"].([]any)
	if strings.Contains(modifiedArgs[0].(string), "${{") {
		t.Errorf("Modified tools should have expression replaced")
	}
}
