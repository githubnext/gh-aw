//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/parser"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/require"
)

func TestRepositoryProjectInstallsUseNpmCI(t *testing.T) {
	tests := []struct {
		workflow string
		section  string
		stepName string
		project  string
	}{
		{"jsweep", "steps", "Install Node.js dependencies", "actions/setup/js"},
		{"daily-multi-device-docs-tester", "pre-agent-steps", "Install and build docs", "docs"},
	}
	for _, tt := range tests {
		t.Run(tt.workflow, func(t *testing.T) {
			workflowDir := "../../.github/workflows"
			source, err := os.ReadFile(filepath.Join(workflowDir, tt.workflow+".md"))
			require.NoError(t, err)
			frontmatter, err := parser.ExtractFrontmatterFromContent(string(source))
			require.NoError(t, err)
			steps, ok := frontmatter.Frontmatter[tt.section].([]any)
			require.True(t, ok)
			found := false
			for _, entry := range steps {
				step, ok := entry.(map[string]any)
				require.True(t, ok)
				if step["name"] != tt.stepName {
					continue
				}
				found = true
				run, ok := step["run"].(string)
				require.True(t, ok)
				require.Contains(t, run, "npm ci")
				require.NotContains(t, run, "npm install")
			}
			require.True(t, found, "project install step must exist")
			for _, manifest := range []string{"package.json", "package-lock.json"} {
				_, err := os.Stat(filepath.Join("../..", tt.project, manifest))
				require.NoError(t, err, "npm ci requires committed project manifests")
			}

			lock, err := os.ReadFile(filepath.Join(workflowDir, tt.workflow+".lock.yml"))
			require.NoError(t, err)
			var compiled struct {
				Jobs map[string]struct {
					Steps []struct {
						Name string
						Run  string
					}
				}
			}
			require.NoError(t, yaml.Unmarshal(lock, &compiled))
			found = false
			for _, job := range compiled.Jobs {
				for _, step := range job.Steps {
					if step.Name == tt.stepName {
						found = true
						require.Contains(t, step.Run, "npm ci")
						require.NotContains(t, step.Run, "npm install")
					}
				}
			}
			require.True(t, found, "compiled project install step must exist")
		})
	}
}

func TestCompilerPreservesInstallsWithoutProjectLockfiles(t *testing.T) {
	dir := testutil.TempDir(t, "install-without-lockfile")
	path := filepath.Join(dir, "install.md")
	source := `---
on: workflow_dispatch
engine: copilot
strict: false
permissions:
  contents: read
steps:
  - name: Install standalone package
    run: npm install --ignore-scripts --prefix /tmp/standalone example-package@1.2.3
  - name: Install local Python project
    run: uv pip install /tmp/local-project
---
Install dependencies.
`
	require.NoError(t, os.WriteFile(path, []byte(source), 0600))
	require.NoError(t, NewCompiler().CompileWorkflow(path))
	lock, err := os.ReadFile(strings.TrimSuffix(path, ".md") + ".lock.yml")
	require.NoError(t, err)
	require.Contains(t, string(lock), "npm install --ignore-scripts --prefix /tmp/standalone example-package@1.2.3")
	require.Contains(t, string(lock), "uv pip install /tmp/local-project")
	require.NotContains(t, string(lock), "npm ci")
	require.NotContains(t, string(lock), "uv pip install --frozen")
}
