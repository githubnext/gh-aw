package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetActionFolders tests the getActionFolders helper function
func TestGetActionFolders(t *testing.T) {
	tests := []struct {
		name     string
		data     *WorkflowData
		expected []string
	}{
		{
			name:     "nil data",
			data:     nil,
			expected: nil,
		},
		{
			name:     "nil features and no engine",
			data:     &WorkflowData{Features: nil},
			expected: nil,
		},
		{
			name:     "empty features and no engine",
			data:     &WorkflowData{Features: map[string]any{}},
			expected: nil,
		},
		{
			name:     "action-folder not specified, no engine",
			data:     &WorkflowData{Features: map[string]any{"other": "value"}},
			expected: nil,
		},
		{
			name:     "action-folder is nil, no engine",
			data:     &WorkflowData{Features: map[string]any{"action-folder": nil}},
			expected: nil,
		},
		{
			name:     "action-folder is empty string, no engine",
			data:     &WorkflowData{Features: map[string]any{"action-folder": ""}},
			expected: nil,
		},
		{
			name:     "claude engine only (no action-folder)",
			data:     &WorkflowData{EngineConfig: &EngineConfig{ID: "claude"}},
			expected: []string{".claude"},
		},
		{
			name:     "codex engine only (no action-folder)",
			data:     &WorkflowData{EngineConfig: &EngineConfig{ID: "codex"}},
			expected: []string{".codex"},
		},
		{
			name:     "copilot engine only (no action-folder)",
			data:     &WorkflowData{EngineConfig: &EngineConfig{ID: "copilot"}},
			expected: nil,
		},
		{
			name:     "custom engine only (no action-folder)",
			data:     &WorkflowData{EngineConfig: &EngineConfig{ID: "custom"}},
			expected: nil,
		},
		{
			name:     "claude engine with single folder",
			data:     &WorkflowData{EngineConfig: &EngineConfig{ID: "claude"}, Features: map[string]any{"action-folder": "custom-actions"}},
			expected: []string{".claude", "custom-actions"},
		},
		{
			name:     "codex engine with array of folders",
			data:     &WorkflowData{EngineConfig: &EngineConfig{ID: "codex"}, Features: map[string]any{"action-folder": []string{"folder1", "folder2"}}},
			expected: []string{".codex", "folder1", "folder2"},
		},
		{
			name:     "single folder as string (no engine)",
			data:     &WorkflowData{Features: map[string]any{"action-folder": "custom-actions"}},
			expected: []string{"custom-actions"},
		},
		{
			name:     "array of strings ([]any) (no engine)",
			data:     &WorkflowData{Features: map[string]any{"action-folder": []any{"folder1", "folder2"}}},
			expected: []string{"folder1", "folder2"},
		},
		{
			name:     "array of strings ([]string) (no engine)",
			data:     &WorkflowData{Features: map[string]any{"action-folder": []string{"folder1", "folder2"}}},
			expected: []string{"folder1", "folder2"},
		},
		{
			name:     "array with empty strings filtered out (no engine)",
			data:     &WorkflowData{Features: map[string]any{"action-folder": []any{"folder1", "", "folder2"}}},
			expected: []string{"folder1", "folder2"},
		},
		{
			name:     "folder with path separators (no engine)",
			data:     &WorkflowData{Features: map[string]any{"action-folder": ".github/custom"}},
			expected: []string{".github/custom"},
		},
		{
			name:     "legacy AI field with claude",
			data:     &WorkflowData{AI: "claude"},
			expected: []string{".claude"},
		},
		{
			name:     "legacy AI field with codex",
			data:     &WorkflowData{AI: "codex"},
			expected: []string{".codex"},
		},
		{
			name:     "EngineConfig takes precedence over AI field",
			data:     &WorkflowData{EngineConfig: &EngineConfig{ID: "claude"}, AI: "codex"},
			expected: []string{".claude"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getActionFolders(tt.data)
			assert.Equal(t, tt.expected, result, "Should extract folders correctly")
		})
	}
}

// TestGenerateCheckoutActionsFolder_WithActionFolder tests the generateCheckoutActionsFolder function
// with the action-folder feature
func TestGenerateCheckoutActionsFolder_WithActionFolder(t *testing.T) {
	tests := []struct {
		name             string
		actionMode       ActionMode
		engineID         string
		features         map[string]any
		expectedContains []string
		expectedNotNil   bool
		expectedFolders  []string // folders that should be in sparse-checkout
	}{
		{
			name:       "dev mode without action-folder",
			actionMode: ActionModeDev,
			features:   map[string]any{},
			expectedContains: []string{
				"Checkout actions folder",
				"sparse-checkout: |",
				"actions",
			},
			expectedNotNil:  true,
			expectedFolders: []string{"actions"},
		},
		{
			name:       "dev mode with claude engine (auto .claude folder)",
			actionMode: ActionModeDev,
			engineID:   "claude",
			features:   map[string]any{},
			expectedContains: []string{
				"Checkout actions folder",
				"sparse-checkout: |",
				"actions",
				".claude",
			},
			expectedNotNil:  true,
			expectedFolders: []string{"actions", ".claude"},
		},
		{
			name:       "dev mode with codex engine (auto .codex folder)",
			actionMode: ActionModeDev,
			engineID:   "codex",
			features:   map[string]any{},
			expectedContains: []string{
				"Checkout actions folder",
				"sparse-checkout: |",
				"actions",
				".codex",
			},
			expectedNotNil:  true,
			expectedFolders: []string{"actions", ".codex"},
		},
		{
			name:       "dev mode with copilot engine (no additional folder)",
			actionMode: ActionModeDev,
			engineID:   "copilot",
			features:   map[string]any{},
			expectedContains: []string{
				"Checkout actions folder",
				"sparse-checkout: |",
				"actions",
			},
			expectedNotNil:  true,
			expectedFolders: []string{"actions"},
		},
		{
			name:       "dev mode with claude + custom action-folder",
			actionMode: ActionModeDev,
			engineID:   "claude",
			features:   map[string]any{"action-folder": "custom-actions"},
			expectedContains: []string{
				"Checkout actions folder",
				"sparse-checkout: |",
				"actions",
				".claude",
				"custom-actions",
			},
			expectedNotNil:  true,
			expectedFolders: []string{"actions", ".claude", "custom-actions"},
		},
		{
			name:       "dev mode with single action-folder",
			actionMode: ActionModeDev,
			features:   map[string]any{"action-folder": "custom-actions"},
			expectedContains: []string{
				"Checkout actions folder",
				"sparse-checkout: |",
				"actions",
				"custom-actions",
			},
			expectedNotNil:  true,
			expectedFolders: []string{"actions", "custom-actions"},
		},
		{
			name:       "dev mode with multiple action-folders (array)",
			actionMode: ActionModeDev,
			features:   map[string]any{"action-folder": []string{"folder1", "folder2"}},
			expectedContains: []string{
				"Checkout actions folder",
				"sparse-checkout: |",
				"actions",
				"folder1",
				"folder2",
			},
			expectedNotNil:  true,
			expectedFolders: []string{"actions", "folder1", "folder2"},
		},
		{
			name:       "script mode with action-folder",
			actionMode: ActionModeScript,
			features:   map[string]any{"action-folder": "custom-actions"},
			expectedContains: []string{
				"Checkout actions folder",
				"repository: githubnext/gh-aw",
				"sparse-checkout: |",
				"actions",
				"custom-actions",
				"path: /tmp/gh-aw/actions-source",
			},
			expectedNotNil:  true,
			expectedFolders: []string{"actions", "custom-actions"},
		},
		{
			name:           "release mode with action-folder (should not checkout)",
			actionMode:     ActionModeRelease,
			features:       map[string]any{"action-folder": "custom-actions"},
			expectedNotNil: false,
		},
		{
			name:           "dev mode with action-tag (should not checkout)",
			actionMode:     ActionModeDev,
			features:       map[string]any{"action-tag": "v1.0.0"},
			expectedNotNil: false,
		},
		{
			name:           "dev mode with action-tag and action-folder (action-tag takes precedence)",
			actionMode:     ActionModeDev,
			features:       map[string]any{"action-tag": "v1.0.0", "action-folder": "custom-actions"},
			expectedNotNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{
				actionMode: tt.actionMode,
			}

			data := &WorkflowData{
				Features: tt.features,
			}

			// Add engine config if specified
			if tt.engineID != "" {
				data.EngineConfig = &EngineConfig{ID: tt.engineID}
			}

			result := compiler.generateCheckoutActionsFolder(data)

			if tt.expectedNotNil {
				require.NotNil(t, result, "Should generate checkout steps")
				require.NotEmpty(t, result, "Should have at least one step")

				// Join all result strings to check content
				fullYAML := strings.Join(result, "")

				// Check that all expected strings are present
				for _, expected := range tt.expectedContains {
					assert.Contains(t, fullYAML, expected, "Should contain expected string: %s", expected)
				}

				// Verify folder presence in the full YAML output
				// Since folders are included in the sparse-checkout section as separate lines
				// (e.g., "            actions\n"), we can simply check the full YAML
				if len(tt.expectedFolders) > 0 {
					for _, folder := range tt.expectedFolders {
						assert.Contains(t, fullYAML, folder,
							"YAML should include folder: %s", folder)
					}
				}
			} else {
				assert.Nil(t, result, "Should not generate checkout steps")
			}
		})
	}
}

// TestGenerateCheckoutActionsFolder_FolderFormats tests various input formats
func TestGenerateCheckoutActionsFolder_FolderFormats(t *testing.T) {
	tests := []struct {
		name            string
		actionFolderVal any
		expectedFolders []string
	}{
		{
			name:            "string: single folder",
			actionFolderVal: "custom",
			expectedFolders: []string{"actions", "custom"},
		},
		{
			name:            "array: []any",
			actionFolderVal: []any{"x", "y"},
			expectedFolders: []string{"actions", "x", "y"},
		},
		{
			name:            "array: []string",
			actionFolderVal: []string{"p", "q"},
			expectedFolders: []string{"actions", "p", "q"},
		},
		{
			name:            "path with slashes",
			actionFolderVal: ".github/custom-actions",
			expectedFolders: []string{"actions", ".github/custom-actions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := &Compiler{
				actionMode: ActionModeDev,
			}

			data := &WorkflowData{
				Features: map[string]any{"action-folder": tt.actionFolderVal},
			}

			result := compiler.generateCheckoutActionsFolder(data)
			require.NotNil(t, result, "Should generate checkout steps")

			fullYAML := strings.Join(result, "")

			// Verify all expected folders are present
			for _, folder := range tt.expectedFolders {
				assert.Contains(t, fullYAML, folder, "Should contain folder: %s", folder)
			}
		})
	}
}
