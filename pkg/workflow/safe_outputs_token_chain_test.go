//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestSafeOutputsTokenChain verifies that safe outputs use the correct token precedence chain:
// per-output config > safe-outputs global > workflow-level > GITHUB_TOKEN
func TestSafeOutputsTokenChain(t *testing.T) {
	tests := []struct {
		name                string
		markdown            string
		expectedTokenInYAML string
	}{
		{
			name: "no custom tokens - uses GITHUB_TOKEN",
			markdown: `---
name: Test Safe Outputs Token Chain
on: issue_comment
engine: copilot

safe-outputs:
  create-issue:
---

Test workflow`,
			expectedTokenInYAML: "github-token: ${{ secrets.GITHUB_TOKEN }}",
		},
		{
			name: "workflow-level token - uses workflow token",
			markdown: `---
name: Test Safe Outputs Token Chain
on: issue_comment
engine: copilot

tools:
  github:
    github-token: ${{ secrets.CUSTOM_WORKFLOW_TOKEN }}

safe-outputs:
  create-issue:
---

Test workflow`,
			expectedTokenInYAML: "github-token: ${{ secrets.CUSTOM_WORKFLOW_TOKEN }}",
		},
		{
			name: "safe-outputs global token - overrides workflow token",
			markdown: `---
name: Test Safe Outputs Token Chain
on: issue_comment
engine: copilot

tools:
  github:
    github-token: ${{ secrets.CUSTOM_WORKFLOW_TOKEN }}

safe-outputs:
  github-token: ${{ secrets.SAFE_OUTPUTS_GLOBAL_TOKEN }}
  create-issue:
---

Test workflow`,
			expectedTokenInYAML: "github-token: ${{ secrets.SAFE_OUTPUTS_GLOBAL_TOKEN }}",
		},
		{
			name: "per-output token - highest precedence",
			markdown: `---
name: Test Safe Outputs Token Chain
on: issue_comment
engine: copilot

tools:
  github:
    github-token: ${{ secrets.CUSTOM_WORKFLOW_TOKEN }}

safe-outputs:
  github-token: ${{ secrets.SAFE_OUTPUTS_GLOBAL_TOKEN }}
  create-issue:
    github-token: ${{ secrets.PER_OUTPUT_TOKEN }}
---

Test workflow`,
			expectedTokenInYAML: "github-token: ${{ secrets.PER_OUTPUT_TOKEN }}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory and file
			tmpDir := testutil.TempDir(t, "token-chain-test")
			testFile := filepath.Join(tmpDir, "test-workflow.md")
			
			if err := os.WriteFile(testFile, []byte(tt.markdown), 0644); err != nil {
				t.Fatal(err)
			}

			compiler := NewCompiler(false, "", "")
			err := compiler.CompileWorkflow(testFile)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			// Read the compiled lock file
			lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
			compiledBytes, err := os.ReadFile(lockFile)
			if err != nil {
				t.Fatalf("Failed to read compiled lock file: %v", err)
			}
			compiled := string(compiledBytes)

			// Check that the expected token is in the compiled YAML
			if !strings.Contains(compiled, tt.expectedTokenInYAML) {
				t.Errorf("Expected compiled YAML to contain %q but it was not found.\nCompiled YAML:\n%s",
					tt.expectedTokenInYAML, compiled)
			}

			// Ensure the MCP token chain is NOT used in safe outputs
			// (the old chain was: GH_AW_GITHUB_MCP_SERVER_TOKEN || GH_AW_GITHUB_TOKEN || GITHUB_TOKEN)
			if strings.Contains(compiled, "GH_AW_GITHUB_MCP_SERVER_TOKEN") && strings.Contains(compiled, "create-issue") {
				t.Error("Safe outputs should not use GH_AW_GITHUB_MCP_SERVER_TOKEN in the token chain")
			}
			if strings.Contains(compiled, "GH_AW_GITHUB_TOKEN") && strings.Contains(compiled, "create-issue") && !strings.Contains(tt.markdown, "GH_AW_GITHUB_TOKEN") {
				t.Error("Safe outputs should not use GH_AW_GITHUB_TOKEN in the token chain unless explicitly configured")
			}
		})
	}
}
