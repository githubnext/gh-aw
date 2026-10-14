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
		config        string
		customSetup   string
		trial         bool
		logicalRepo   string
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
			name:        "disabled checkout uses default",
			goMod:       true,
			config:      "checkout: false\n",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:        "skipped default checkout uses default",
			goMod:       true,
			config:      "permissions:\n  contents: none\n",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:          "custom checkout uses go.mod",
			goMod:         true,
			customSetup:   "  - uses: actions/checkout@v7\n    with:\n      persist-credentials: false\n",
			wantGoModFile: true,
		},
		{
			name:        "custom subdirectory checkout uses default",
			goMod:       true,
			customSetup: "  - uses: actions/checkout@v7\n    with:\n      persist-credentials: false\n      path: src\n",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:        "custom external checkout uses default",
			goMod:       true,
			customSetup: "  - uses: actions/checkout@v7\n    with:\n      persist-credentials: false\n      repository: example/other\n",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:        "configured external root checkout uses default",
			goMod:       true,
			config:      "checkout:\n  repository: example/other\n  path: .\n",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:        "root wiki checkout uses default",
			goMod:       true,
			config:      "checkout:\n  wiki: true\n",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:          "configured current repository expression",
			goMod:         true,
			config:        "checkout:\n  repository: ${{github.repository}}\n  path: .\n",
			wantGoModFile: true,
		},
		{
			name:        "configured checkout ref uses default",
			goMod:       true,
			config:      "checkout:\n  ref: other-branch\n",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:        "custom checkout ref uses default",
			goMod:       true,
			customSetup: "  - uses: actions/checkout@v7\n    with:\n      persist-credentials: false\n      ref: other-branch\n",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:          "configured sparse cone checkout retains root files",
			goMod:         true,
			config:        "checkout:\n  sparse-checkout: docs\n",
			wantGoModFile: true,
		},
		{
			name:          "custom sparse cone checkout retains root files",
			goMod:         true,
			customSetup:   "  - uses: actions/checkout@v7\n    with:\n      persist-credentials: false\n      sparse-checkout: docs\n",
			wantGoModFile: true,
		},
		{
			name:          "custom non-cone checkout includes go.mod",
			goMod:         true,
			customSetup:   "  - uses: actions/checkout@v7\n    with:\n      persist-credentials: false\n      sparse-checkout-cone-mode: false\n      sparse-checkout: |\n        /go.mod\n        /docs/\n",
			wantGoModFile: true,
		},
		{
			name:        "custom non-cone checkout excludes go.mod",
			goMod:       true,
			customSetup: "  - uses: actions/checkout@v7\n    with:\n      persist-credentials: false\n      sparse-checkout-cone-mode: false\n      sparse-checkout: docs\n",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:        "trial logical repository uses default",
			goMod:       true,
			trial:       true,
			logicalRepo: "example/other",
			wantVersion: string(constants.DefaultGoVersion),
		},
		{
			name:          "trial without logical repository retains current checkout",
			goMod:         true,
			trial:         true,
			wantGoModFile: true,
		},
		{
			name:          "trial ignores configured ref override",
			goMod:         true,
			trial:         true,
			config:        "checkout:\n  ref: other-branch\n",
			wantGoModFile: true,
		},
		{
			name:        "conditional custom checkout uses default",
			goMod:       true,
			customSetup: "  - uses: actions/checkout@v7\n    if: github.event_name == 'push'\n    with:\n      persist-credentials: false\n",
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
%s%ssteps:
%s  - run: go mod vendor
---

# Test Go version selection
`, runtimes, tt.config, tt.customSetup)
			require.NoError(t, os.WriteFile(workflowPath, []byte(content), 0644))

			compiler := NewCompiler()
			compiler.gitRoot = root
			compiler.trialMode = tt.trial
			compiler.trialLogicalRepoSlug = tt.logicalRepo
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

func TestSparseCheckoutIncludesRootGoMod(t *testing.T) {
	tests := []struct {
		patterns string
		coneMode string
		want     bool
	}{
		{"docs", "", true},
		{"docs", "true", true},
		{"docs", "false", false},
		{"go.mod", "false", true},
		{"/*.mod", "false", true},
		{"**/go.mod", "false", true},
		{"*\n!**/go.mod", "false", false},
		{"*\n!**/*.mod", "false", false},
		{"*\n!go.mod", "false", false},
		{"*\n!*.mod", "false", false},
		{"*\n!*.mod\ngo.mod", "false", true},
		{"# go.mod\ndocs/", "false", false},
		{"go.mod/", "false", false},
		{"${{ inputs.paths }}", "", false},
		{"docs", "${{ inputs.cone }}", false},
		{"[", "false", false},
		{"*\n![[:alpha:]]*", "false", false},
		{"*\n![!a]*", "false", false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s/%s", tt.patterns, tt.coneMode), func(t *testing.T) {
			assert.Equal(t, tt.want, sparseCheckoutIncludesRootGoMod(tt.patterns, tt.coneMode))
		})
	}
}

func TestCheckoutStepsHaveRootGoMod(t *testing.T) {
	tests := []struct {
		name  string
		steps string
		first bool
		want  bool
	}{
		{"wrapped current expression", "steps:\n  - uses: actions/checkout@v7\n    with:\n      repository: ${{  github.repository  }}\n", true, true},
		{"malformed steps", "steps: [", true, false},
		{"first checkout is subdirectory", "- uses: actions/checkout@v7\n  with:\n    path: src\n- uses: actions/checkout@v7\n", true, false},
		{"last root checkout is external", "- uses: actions/checkout@v7\n- uses: actions/checkout@v7\n  with:\n    repository: example/other\n", false, false},
		{"additional subdirectory preserves root", "- uses: actions/checkout@v7\n- uses: actions/checkout@v7\n  with:\n    repository: example/other\n    path: src\n", false, true},
		{"last root checkout has ref", "- uses: actions/checkout@v7\n- uses: actions/checkout@v7\n  with:\n    ref: other-branch\n", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, checkoutStepsHaveRootGoMod(tt.steps, tt.first))
		})
	}
}
