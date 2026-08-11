//go:build !integration

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTempFolderPromptIncluded(t *testing.T) {
	engines := []string{"codex", "claude", "copilot"}

	for _, engine := range engines {
		t.Run(engine, func(t *testing.T) {
			tmpDir := t.TempDir()
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			testContent := fmt.Sprintf(`---
on: push
engine: %s
---

# Test Workflow

This is a test workflow to verify temp folder instructions are included.
`, engine)

			require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

			compiler := NewCompiler()
			require.NoError(t, compiler.CompileWorkflow(testFile))

			lockFile := stringutil.MarkdownToLockFile(testFile)
			lockContent, err := os.ReadFile(lockFile)
			require.NoError(t, err)

			lockStr := string(lockContent)
			tests := []struct {
				name     string
				contains string
			}{
				{"prompt step name", "- name: Create prompt with built-in context"},
				{"temp folder prompt config", `\"file\":\"temp_folder_prompt.md\"`},
				{"JavaScript prompt renderer", "create_prompt.cjs"},
			}
			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					assert.Contains(t, lockStr, tt.contains)
				})
			}
		})
	}
}
