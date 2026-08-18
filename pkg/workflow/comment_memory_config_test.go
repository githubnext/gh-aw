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
		name                    string
		commentMemoryConfigYAML string
		expectedCommentMemory   *CommentMemoryConfig
	}{
		{
			name:                    "defaults disabled when key absent",
			commentMemoryConfigYAML: "",
		},
		{
			name:                    "explicit false disables comment-memory",
			commentMemoryConfigYAML: "  comment-memory: false\n",
		},
		{
			name:                    "explicit null disables comment-memory",
			commentMemoryConfigYAML: "  comment-memory: null\n",
		},
		{
			name:                    "explicit true enables with defaults",
			commentMemoryConfigYAML: "  comment-memory: true\n",
			expectedCommentMemory: &CommentMemoryConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				MemoryID: "default",
			},
		},
		{
			name: "map config sets fields",
			commentMemoryConfigYAML: `  comment-memory:
    max: 3
    memory-id: triage
    target: "42"
    target-repo: github/docs
    allowed-repos:
      - github/docs
      - github/gh-aw
    footer: false
`,
			expectedCommentMemory: &CommentMemoryConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("3"),
				},
				MemoryID:       "triage",
				Target:         "42",
				TargetRepoSlug: "github/docs",
				AllowedRepos:   []string{"github/docs", "github/gh-aw"},
				Footer:         strPtr("false"),
			},
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
			if tt.expectedCommentMemory != nil {
				require.NotNil(t, workflowData.CommentMemoryConfig, "CommentMemory should be enabled")
				assert.Equal(t, tt.expectedCommentMemory, workflowData.CommentMemoryConfig)
			} else {
				assert.Nil(t, workflowData.CommentMemoryConfig, "CommentMemory should be disabled")
			}
		})
	}
}

func TestExtractCommentMemoryConfigInvalidFields(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tests := []struct {
		name      string
		rawConfig any
	}{
		{
			name: "invalid max disables comment-memory",
			rawConfig: map[string]any{
				"max": "invalid",
			},
		},
		{
			name: "invalid footer disables comment-memory",
			rawConfig: map[string]any{
				"footer": "invalid",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.extractCommentMemoryConfig(&ToolsConfig{
				CommentMemory: &CommentMemoryToolConfig{
					Raw: tt.rawConfig,
				},
			})
			assert.Nil(t, config, "CommentMemory should be disabled")
		})
	}
}

func TestParseCommentMemoryConfigValueBoolAndNil(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tests := []struct {
		name                  string
		rawConfig             any
		expectedCommentMemory *CommentMemoryConfig
	}{
		{
			name:      "nil disables comment-memory",
			rawConfig: nil,
		},
		{
			name:      "false disables comment-memory",
			rawConfig: false,
		},
		{
			name:      "true enables defaults",
			rawConfig: true,
			expectedCommentMemory: &CommentMemoryConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				MemoryID: "default",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := compiler.parseCommentMemoryConfigValue(tt.rawConfig)
			assert.Equal(t, tt.expectedCommentMemory, config)
		})
	}
}
