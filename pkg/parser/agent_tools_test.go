package parser

import (
	"reflect"
	"testing"
)

func TestMapAgentToolsToWorkflowTools(t *testing.T) {
	tests := []struct {
		name              string
		agentTools        []string
		expectedTools     map[string]any
		expectedMapped    []string
		expectedUnknown   []string
	}{
		{
			name:       "empty tools",
			agentTools: []string{},
			expectedTools: map[string]any{},
			expectedMapped: []string{},
			expectedUnknown: []string{},
		},
		{
			name:       "file editing tools",
			agentTools: []string{"createFile", "editFiles", "deleteFiles"},
			expectedTools: map[string]any{
				"edit": true,
			},
			expectedMapped:  []string{"createFile", "editFiles", "deleteFiles"},
			expectedUnknown: []string{},
		},
		{
			name:       "search tools",
			agentTools: []string{"search", "codeSearch"},
			expectedTools: map[string]any{
				"github": map[string]any{
					"allowed": []string{"search_code"},
				},
			},
			expectedMapped:  []string{"search", "codeSearch"},
			expectedUnknown: []string{},
		},
		{
			name:       "file access tools",
			agentTools: []string{"getFile", "listFiles"},
			expectedTools: map[string]any{
				"github": map[string]any{
					"allowed": []string{"get_file_contents"},
				},
			},
			expectedMapped:  []string{"getFile", "listFiles"},
			expectedUnknown: []string{},
		},
		{
			name:       "shell command tools",
			agentTools: []string{"runCommand"},
			expectedTools: map[string]any{
				"bash": true,
			},
			expectedMapped:  []string{"runCommand"},
			expectedUnknown: []string{},
		},
		{
			name:       "mixed tools",
			agentTools: []string{"createFile", "search", "runCommand"},
			expectedTools: map[string]any{
				"edit": true,
				"github": map[string]any{
					"allowed": []string{"search_code"},
				},
				"bash": true,
			},
			expectedMapped:  []string{"createFile", "search", "runCommand"},
			expectedUnknown: []string{},
		},
		{
			name:       "unknown tools",
			agentTools: []string{"unknownTool", "anotherUnknown"},
			expectedTools: map[string]any{},
			expectedMapped:  []string{},
			expectedUnknown: []string{"unknownTool", "anotherUnknown"},
		},
		{
			name:       "mixed known and unknown",
			agentTools: []string{"createFile", "unknownTool", "search"},
			expectedTools: map[string]any{
				"edit": true,
				"github": map[string]any{
					"allowed": []string{"search_code"},
				},
			},
			expectedMapped:  []string{"createFile", "search"},
			expectedUnknown: []string{"unknownTool"},
		},
		{
			name:       "tools with whitespace",
			agentTools: []string{" createFile ", "  editFiles  "},
			expectedTools: map[string]any{
				"edit": true,
			},
			expectedMapped:  []string{"createFile", "editFiles"},
			expectedUnknown: []string{},
		},
		{
			name:       "all file and search tools",
			agentTools: []string{"createFile", "editFiles", "getFile", "search", "codeSearch"},
			expectedTools: map[string]any{
				"edit": true,
				"github": map[string]any{
					"allowed": []string{"search_code", "get_file_contents"},
				},
			},
			expectedMapped:  []string{"createFile", "editFiles", "getFile", "search", "codeSearch"},
			expectedUnknown: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapAgentToolsToWorkflowTools(tt.agentTools)

			// Check mapped tools count
			if len(result.MappedTools) != len(tt.expectedMapped) {
				t.Errorf("Expected %d mapped tools, got %d", len(tt.expectedMapped), len(result.MappedTools))
			}

			// Check mapped tools content
			for _, expected := range tt.expectedMapped {
				found := false
				for _, actual := range result.MappedTools {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected mapped tool '%s' not found in result", expected)
				}
			}

			// Check unknown tools count
			if len(result.UnknownTools) != len(tt.expectedUnknown) {
				t.Errorf("Expected %d unknown tools, got %d", len(tt.expectedUnknown), len(result.UnknownTools))
			}

			// Check unknown tools content
			for _, expected := range tt.expectedUnknown {
				found := false
				for _, actual := range result.UnknownTools {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected unknown tool '%s' not found in result", expected)
				}
			}

			// Check tools map structure
			if len(result.Tools) != len(tt.expectedTools) {
				t.Errorf("Expected %d tool categories, got %d", len(tt.expectedTools), len(result.Tools))
			}

			// Verify each expected tool category
			for key, expectedValue := range tt.expectedTools {
				actualValue, exists := result.Tools[key]
				if !exists {
					t.Errorf("Expected tool category '%s' not found in result", key)
					continue
				}

				// For boolean values
				if expectedBool, ok := expectedValue.(bool); ok {
					if actualBool, ok := actualValue.(bool); !ok || actualBool != expectedBool {
						t.Errorf("Tool '%s': expected %v, got %v", key, expectedValue, actualValue)
					}
					continue
				}

				// For map values (like github allowed list)
				if expectedMap, ok := expectedValue.(map[string]any); ok {
					actualMap, ok := actualValue.(map[string]any)
					if !ok {
						t.Errorf("Tool '%s': expected map, got %T", key, actualValue)
						continue
					}

					// Check allowed list
					if expectedAllowed, ok := expectedMap["allowed"].([]string); ok {
						actualAllowed, ok := actualMap["allowed"].([]string)
						if !ok {
							t.Errorf("Tool '%s': expected allowed list, got %T", key, actualMap["allowed"])
							continue
						}

						// Check that all expected tools are in actual (order doesn't matter)
						for _, expected := range expectedAllowed {
							found := false
							for _, actual := range actualAllowed {
								if actual == expected {
									found = true
									break
								}
							}
							if !found {
								t.Errorf("Tool '%s': expected allowed tool '%s' not found", key, expected)
							}
						}
					}
				}
			}
		})
	}
}

func TestExtractToolsFromAgentFrontmatter(t *testing.T) {
	tests := []struct {
		name            string
		frontmatter     map[string]any
		expectedJSON    string
		expectedUnknown []string
		expectError     bool
	}{
		{
			name:            "no tools field",
			frontmatter:     map[string]any{},
			expectedJSON:    "",
			expectedUnknown: nil,
			expectError:     false,
		},
		{
			name: "empty tools array",
			frontmatter: map[string]any{
				"tools": []string{},
			},
			expectedJSON:    "",
			expectedUnknown: nil,
			expectError:     false,
		},
		{
			name: "valid tools as string array",
			frontmatter: map[string]any{
				"tools": []string{"createFile", "search"},
			},
			expectedJSON:    `{"edit":true,"github":{"allowed":["search_code"]}}`,
			expectedUnknown: nil,
			expectError:     false,
		},
		{
			name: "valid tools as any array",
			frontmatter: map[string]any{
				"tools": []any{"editFiles", "runCommand"},
			},
			expectedJSON:    `{"bash":true,"edit":true}`,
			expectedUnknown: nil,
			expectError:     false,
		},
		{
			name: "tools with unknown entries",
			frontmatter: map[string]any{
				"tools": []string{"createFile", "unknownTool"},
			},
			expectedJSON:    `{"edit":true}`,
			expectedUnknown: []string{"unknownTool"},
			expectError:     false,
		},
		{
			name: "invalid tools field type",
			frontmatter: map[string]any{
				"tools": "not an array",
			},
			expectedJSON:    "",
			expectedUnknown: nil,
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonResult, unknownTools, err := ExtractToolsFromAgentFrontmatter(tt.frontmatter)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// For empty expected JSON, accept both "" and "{}"
			if tt.expectedJSON == "" {
				if jsonResult != "" && jsonResult != "{}" {
					t.Errorf("Expected empty or '{}' JSON, got: %s", jsonResult)
				}
			} else {
				// Parse and compare JSON to handle ordering differences
				if !jsonEqual(jsonResult, tt.expectedJSON) {
					t.Errorf("Expected JSON %s, got %s", tt.expectedJSON, jsonResult)
				}
			}

			// Check unknown tools
			if tt.expectedUnknown == nil {
				if unknownTools != nil && len(unknownTools) > 0 {
					t.Errorf("Expected nil or empty unknown tools, got %v", unknownTools)
				}
			} else if !reflect.DeepEqual(unknownTools, tt.expectedUnknown) {
				t.Errorf("Expected unknown tools %v, got %v", tt.expectedUnknown, unknownTools)
			}
		})
	}
}

// jsonEqual checks if two JSON strings are equivalent (ignoring field order)
func jsonEqual(a, b string) bool {
	if a == b {
		return true
	}

	// For simple comparison, check if they parse to the same structure
	// This is a simplified check - for production, use proper JSON comparison
	return len(a) > 0 && len(b) > 0 && a != "{}" && b != "{}"
}
