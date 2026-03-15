//go:build !integration

package parser

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateWorkflowFrontmatter(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := testutil.TempDir(t, "test-*")

	tests := []struct {
		name            string
		initialContent  string
		updateFunc      func(frontmatter map[string]any) error
		expectedContent string
		expectError     bool
		verbose         bool
	}{
		{
			name: "Add tool to existing tools section",
			initialContent: `---
tools:
  existing: {}
---
# Test Workflow
Some content`,
			updateFunc: func(frontmatter map[string]any) error {
				tools := EnsureToolsSection(frontmatter)
				tools["new-tool"] = map[string]any{"type": "test"}
				return nil
			},
			expectedContent: `---
existing: {}
new-tool:
  type: test
tools:
  existing: {}
  new-tool:
    type: test
---
# Test Workflow
Some content`,
			expectError: false,
		},
		{
			name: "Create tools section if missing",
			initialContent: `---
engine: claude
---
# Test Workflow
Some content`,
			updateFunc: func(frontmatter map[string]any) error {
				tools := EnsureToolsSection(frontmatter)
				tools["new-tool"] = map[string]any{"type": "test"}
				return nil
			},
			expectedContent: `---
engine: claude
tools:
  new-tool:
    type: test
---
# Test Workflow
Some content`,
			expectError: false,
		},
		{
			name: "Handle empty frontmatter",
			initialContent: `---
---
# Test Workflow
Some content`,
			updateFunc: func(frontmatter map[string]any) error {
				tools := EnsureToolsSection(frontmatter)
				tools["new-tool"] = map[string]any{"type": "test"}
				return nil
			},
			expectedContent: `---
tools:
  new-tool:
    type: test
---
# Test Workflow
Some content`,
			expectError: false,
		},
		{
			name: "Handle file with no frontmatter",
			initialContent: `# Test Workflow
Some content without frontmatter`,
			updateFunc: func(frontmatter map[string]any) error {
				tools := EnsureToolsSection(frontmatter)
				tools["new-tool"] = map[string]any{"type": "test"}
				return nil
			},
			expectedContent: `---
tools:
  new-tool:
    type: test
---
# Test Workflow
Some content without frontmatter`,
			expectError: false,
		},
		{
			name: "Update function returns error",
			initialContent: `---
tools: {}
---
# Test Workflow`,
			updateFunc: func(frontmatter map[string]any) error {
				return errors.New("test error")
			},
			expectedContent: "",
			expectError:     true,
		},
		{
			name: "Verbose mode does not affect output",
			initialContent: `---
engine: claude
---
# Test`,
			updateFunc: func(frontmatter map[string]any) error {
				frontmatter["engine"] = "copilot"
				return nil
			},
			expectError: false,
			verbose:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			testFile := filepath.Join(tempDir, "test-workflow.md")
			err := os.WriteFile(testFile, []byte(tt.initialContent), 0644)
			require.NoError(t, err, "test file setup should succeed")

			// Run the update function
			err = UpdateWorkflowFrontmatter(testFile, tt.updateFunc, tt.verbose)

			if tt.expectError {
				assert.Error(t, err, "UpdateWorkflowFrontmatter should return an error")
				return
			}

			require.NoError(t, err, "UpdateWorkflowFrontmatter should succeed")

			// Read the updated content
			updatedContent, err := os.ReadFile(testFile)
			require.NoError(t, err, "reading updated file should succeed")

			content := string(updatedContent)
			if tt.verbose {
				// Verbose mode: verify the update was applied correctly (engine changed to copilot)
				assert.Contains(t, content, "engine: copilot", "verbose mode should still update frontmatter correctly")
				assert.Contains(t, content, "---", "verbose mode should preserve frontmatter delimiters")
			} else {
				assert.Contains(t, content, "new-tool:", "updated file should contain the new tool key")
				assert.Contains(t, content, "type: test", "updated file should contain the tool type")
				assert.Contains(t, content, "---", "updated file should preserve frontmatter delimiters")
			}
		})
	}
}

func TestEnsureToolsSection(t *testing.T) {
	tests := []struct {
		name          string
		frontmatter   map[string]any
		expectedTools map[string]any
	}{
		{
			name:          "Create tools section when missing",
			frontmatter:   map[string]any{},
			expectedTools: map[string]any{},
		},
		{
			name: "Return existing tools section",
			frontmatter: map[string]any{
				"tools": map[string]any{
					"existing": map[string]any{"type": "test"},
				},
			},
			expectedTools: map[string]any{
				"existing": map[string]any{"type": "test"},
			},
		},
		{
			name: "Replace invalid tools section",
			frontmatter: map[string]any{
				"tools": "invalid",
			},
			expectedTools: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tools := EnsureToolsSection(tt.frontmatter)

			require.NotNil(t, tools, "EnsureToolsSection should never return nil")

			// Verify frontmatter was updated to contain the tools map
			frontmatterTools, ok := tt.frontmatter["tools"].(map[string]any)
			require.True(t, ok, "frontmatter['tools'] should be a map[string]any")

			// Verify returned tools matches the frontmatter entry and expected content
			assert.Equal(t, frontmatterTools, tools, "returned tools should be the same reference as frontmatter['tools']")
			assert.Equal(t, tt.expectedTools, tools, "tools section should match expected state")
		})
	}
}

func TestReconstructWorkflowFile(t *testing.T) {
	tests := []struct {
		name            string
		frontmatterYAML string
		markdownContent string
		expectedResult  string
	}{
		{
			name:            "With frontmatter and markdown",
			frontmatterYAML: "engine: claude\ntools: {}",
			markdownContent: "# Test Workflow\nSome content",
			expectedResult:  "---\nengine: claude\ntools: {}\n---\n# Test Workflow\nSome content",
		},
		{
			name:            "Empty frontmatter with markdown",
			frontmatterYAML: "",
			markdownContent: "# Test Workflow\nSome content",
			expectedResult:  "---\n---\n# Test Workflow\nSome content",
		},
		{
			name:            "Frontmatter with no markdown",
			frontmatterYAML: "engine: claude",
			markdownContent: "",
			expectedResult:  "---\nengine: claude\n---",
		},
		{
			name:            "Frontmatter with trailing newline",
			frontmatterYAML: "engine: claude\n",
			markdownContent: "# Test",
			expectedResult:  "---\nengine: claude\n---\n# Test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reconstructWorkflowFile(tt.frontmatterYAML, tt.markdownContent)
			require.NoError(t, err, "reconstructWorkflowFile should succeed")
			assert.Equal(t, tt.expectedResult, result, "reconstructed file content should match")
		})
	}
}
