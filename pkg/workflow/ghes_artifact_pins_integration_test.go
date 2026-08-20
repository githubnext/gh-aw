//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGHESArtifactPinsIntegration(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*testing.T, *Compiler, string)
	}{
		{
			name: "compile flag",
			configure: func(_ *testing.T, compiler *Compiler, _ string) {
				compiler.SetGHESCompat(true)
			},
		},
		{
			name: "repository config",
			configure: func(t *testing.T, compiler *Compiler, root string) {
				configPath := filepath.Join(root, RepoConfigFileName)
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o755))
				require.NoError(t, os.WriteFile(configPath, []byte(`{"ghes":true}`), 0o600))
				compiler.gitRoot = root
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := testutil.TempDir(t, "ghes-artifact-pins")
			workflowPath := filepath.Join(root, "artifact-pins.md")
			require.NoError(t, os.WriteFile(workflowPath, []byte(`---
on: workflow_dispatch
permissions:
  contents: read
engine: copilot
strict: false
steps:
  - name: Upload test artifact
    uses: actions/upload-artifact@v7
    with:
      name: test
      path: test.txt
---
# GHES artifact pins
`), 0o600))

			compiler := NewCompiler()
			tt.configure(t, compiler, root)
			require.NoError(t, compiler.CompileWorkflow(workflowPath))

			lock, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
			require.NoError(t, err)
			contents := string(lock)
			assert.Contains(t, contents, "actions/upload-artifact@c6a366c94c3e0affe28c06c8df20a878f24da3cf # v3.2.2")
			assert.Contains(t, contents, "actions/download-artifact@a9bc5e6ef2cb54c177f32aa5726adaa15e7e2d59 # v3.1.0")
			assert.NotContains(t, contents, "actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1")
			assert.NotContains(t, contents, "actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c # v8.0.1")
		})
	}
}
