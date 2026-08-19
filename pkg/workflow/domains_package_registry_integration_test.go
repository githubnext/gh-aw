//go:build integration

package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// TestCompiledWorkflowGatesPackageRegistries compiles real workflows and asserts that the
// firewall allow-list in the generated lock file only contains package registry domains when
// the workflow explicitly opts into the corresponding ecosystem. Selecting an engine alone
// must never make npm or PyPI reachable from the sandbox.
func TestCompiledWorkflowGatesPackageRegistries(t *testing.T) {
	registries := []string{"registry.npmjs.org", "pypi.org", "files.pythonhosted.org"}

	tests := []struct {
		name              string
		engine            string
		networkBlock      string
		expectContains    []string
		expectNotContains []string
	}{
		{
			name:              "copilot deny-all network",
			engine:            "copilot",
			networkBlock:      "network: {}",
			expectNotContains: registries,
		},
		{
			name:              "copilot defaults and github",
			engine:            "copilot",
			networkBlock:      "network:\n  allowed:\n    - defaults\n    - github",
			expectNotContains: registries,
		},
		{
			name:              "claude deny-all network",
			engine:            "claude",
			networkBlock:      "network: {}",
			expectNotContains: registries,
		},
		{
			name:              "claude defaults and github",
			engine:            "claude",
			networkBlock:      "network:\n  allowed:\n    - defaults\n    - github",
			expectNotContains: registries,
		},
		{
			name:              "copilot explicit node opt-in",
			engine:            "copilot",
			networkBlock:      "network:\n  allowed:\n    - defaults\n    - node",
			expectContains:    []string{"registry.npmjs.org"},
			expectNotContains: []string{"pypi.org", "files.pythonhosted.org"},
		},
		{
			name:              "copilot explicit python opt-in",
			engine:            "copilot",
			networkBlock:      "network:\n  allowed:\n    - defaults\n    - python",
			expectContains:    []string{"pypi.org", "files.pythonhosted.org"},
			expectNotContains: []string{"registry.npmjs.org"},
		},
		{
			name:           "copilot node runtime declaration",
			engine:         "copilot",
			networkBlock:   "network:\n  allowed:\n    - defaults\nruntimes:\n  node:\n    version: \"24\"",
			expectContains: []string{"registry.npmjs.org"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := testutil.TempDir(t, "registry-gate-*")
			workflowPath := filepath.Join(tempDir, "test-workflow.md")
			content := "---\non: issues\npermissions:\n  contents: read\nengine: " + tt.engine +
				"\nstrict: false\n" + tt.networkBlock + "\n---\n\n# Registry Gate Test\n\nDo nothing.\n"
			if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write workflow: %v", err)
			}

			compiler := workflow.NewCompiler()
			if err := compiler.CompileWorkflow(workflowPath); err != nil {
				t.Fatalf("CompileWorkflow failed: %v", err)
			}

			lockContent, err := os.ReadFile(stringutil.MarkdownToLockFile(workflowPath))
			if err != nil {
				t.Fatalf("failed to read lock file: %v", err)
			}
			lock := string(lockContent)

			for _, expected := range tt.expectContains {
				if !strings.Contains(lock, expected) {
					t.Errorf("expected compiled workflow to allow %q", expected)
				}
			}
			for _, unexpected := range tt.expectNotContains {
				if strings.Contains(lock, unexpected) {
					t.Errorf("compiled workflow must not allow %q without explicit ecosystem opt-in", unexpected)
				}
			}
		})
	}
}
