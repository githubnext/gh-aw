//go:build !integration

package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileWorkflowGoModVersion(t *testing.T) {
	tests := []struct {
		name          string
		goMod         bool
		goModDir      bool
		version       string
		customSetup   string
		wantVersion   string
		wantGoModFile bool
	}{
		{
			name:          "root go.mod",
			goMod:         true,
			wantGoModFile: true,
		},
		{
			name:        "explicit version overrides go.mod",
			goMod:       true,
			version:     "1.25",
			wantVersion: "1.25",
		},
		{
			name:        "no go.mod uses default",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:        "go.mod directory uses default",
			goModDir:    true,
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:  "deduplicated root go.mod setup",
			goMod: true,
			customSetup: `  - uses: actions/setup-go@v7
    with:
      go-version-file: go.mod
      cache: false
`,
			wantGoModFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if tt.goMod {
				require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n\ngo 1.27.0\n\ntoolchain go1.27.2\n"), 0644))
			}
			if tt.goModDir {
				require.NoError(t, os.Mkdir(filepath.Join(root, "go.mod"), 0755))
			}

			workflowDir := filepath.Join(root, ".github", "workflows")
			require.NoError(t, os.MkdirAll(workflowDir, 0755))
			workflowPath := filepath.Join(workflowDir, "go-mod.md")
			runtimes := ""
			if tt.version != "" {
				runtimes = fmt.Sprintf("runtimes:\n  go:\n    version: '%s'\n", tt.version)
			}
			content := fmt.Sprintf(`---
on: push
engine: copilot
%ssteps:
%s  - run: go mod vendor
---

# Test Go version selection
`, runtimes, tt.customSetup)
			require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0644))

			compiler := NewCompiler()
			compiler.gitRoot = root
			require.NoError(t, compiler.CompileWorkflow(workflowPath))
			lock, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
			require.NoError(t, err)
			lockContent := string(lock)
			setupStart := strings.Index(lockContent, "      - name: Setup Go\n")
			require.NotEqual(t, -1, setupStart)
			setup := strings.SplitN(lockContent[setupStart:], "\n      - ", 2)[0]
			assert.Contains(t, setup, "uses: actions/setup-go@")
			assert.Contains(t, setup, "cache: false")
			if tt.wantGoModFile {
				assert.Contains(t, setup, "go-version-file: go.mod")
				assert.NotContains(t, setup, "go-version:")
			} else {
				assert.Contains(t, setup, fmt.Sprintf("go-version: '%s'", tt.wantVersion))
				assert.NotContains(t, setup, "go-version-file:")
			}
			assert.Equal(t, 1, strings.Count(lockContent, "        uses: actions/setup-go@"))
			assert.Less(t, setupStart, strings.Index(lockContent, "run: go mod vendor"))
		})
	}
}
