//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommentMemoryToolConfig(t *testing.T) {
	tests := []struct {
		name                     string
		commentMemoryConfigYAML  string
		expectedCommentMemorySet bool
	}{
		{
			name:                     "defaults disabled when key absent",
			commentMemoryConfigYAML:  "",
			expectedCommentMemorySet: false,
		},
		{
			name:                     "explicit false disables comment-memory",
			commentMemoryConfigYAML:  "  comment-memory: false\n",
			expectedCommentMemorySet: false,
		},
		{
			name:                     "explicit null disables comment-memory",
			commentMemoryConfigYAML:  "  comment-memory: null\n",
			expectedCommentMemorySet: false,
		},
		{
			name:                     "explicit true enables with defaults",
			commentMemoryConfigYAML:  "  comment-memory: true\n",
			expectedCommentMemorySet: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := testutil.TempDir(t, "comment-memory-config-test")
			toolsSection := ""
			if tt.commentMemoryConfigYAML != "" {
				toolsSection = "tools:\n" + tt.commentMemoryConfigYAML
			}
			workflow := `---
on: issues
engine: copilot
permissions:
  contents: read
` + toolsSection + `safe-outputs:
  add-comment:
    max: 1
---

# Test
`
			testFile := filepath.Join(tmpDir, "workflow.md")
			err := os.WriteFile(testFile, []byte(workflow), 0644)
			require.NoError(t, err, "Failed to write test workflow")

			compiler := NewCompiler(WithVersion("1.0.0"))
			workflowData, err := compiler.ParseWorkflowFile(testFile)
			require.NoError(t, err, "Failed to parse workflow")
			require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should be present")

			if tt.expectedCommentMemorySet {
				require.NotNil(t, workflowData.SafeOutputs.CommentMemory, "CommentMemory should be enabled")
				assert.Equal(t, strPtr("1"), workflowData.SafeOutputs.CommentMemory.Max, "CommentMemory max should default to 1")
				assert.Equal(t, "default", workflowData.SafeOutputs.CommentMemory.MemoryID, "CommentMemory memory-id should default to 'default'")
			} else {
				assert.Nil(t, workflowData.SafeOutputs.CommentMemory, "CommentMemory should be disabled")
			}
		})
	}
}
