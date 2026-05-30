//go:build integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
)

// TestMCPGatewayMCPScriptsEnvVarsConditional tests that MCP Scripts environment variables
// are only included in the MCP gateway step when MCP Scripts is actually configured
func TestMCPGatewayMCPScriptsEnvVarsConditional(t *testing.T) {
	tests := []struct {
		name                    string
		workflowContent         string
		expectMCPScriptsEnvVars bool
		description             string
	}{
		{
			name: "No MCP Scripts - Should Not Include Env Vars",
			workflowContent: `---
on: workflow_dispatch
engine: copilot
tools:
  github:
    toolsets: [default]
---

Test workflow without mcp-scripts
`,
			expectMCPScriptsEnvVars: false,
			description:             "When mcp-scripts is not configured, the MCP gateway should not reference MCP Scripts env vars",
		},
		{
			name: "With MCP Scripts - Should Include Env Vars",
			workflowContent: `---
on: workflow_dispatch
engine: copilot
tools:
  github:
    toolsets: [default]
mcp-scripts:
  test-tool:
    description: Test tool
    script: |
      return { result: "test" };
---

Test workflow with mcp-scripts
`,
			expectMCPScriptsEnvVars: true,
			description:             "When mcp-scripts is configured, the MCP gateway should reference MCP Scripts env vars",
		},
		{
			name: "Claude Engine Without MCP Scripts",
			workflowContent: `---
on: workflow_dispatch
engine: claude
tools:
  github:
    toolsets: [default]
---

Test Claude workflow without mcp-scripts
`,
			expectMCPScriptsEnvVars: false,
			description:             "Claude engine without mcp-scripts should not include MCP Scripts env vars",
		},
		{
			name: "Codex Engine With MCP Scripts",
			workflowContent: `---
on: workflow_dispatch
engine: codex
mcp-scripts:
  my-tool:
    description: My custom tool
    script: |
      return { status: "ok" };
---

Test Codex workflow with mcp-scripts
`,
			expectMCPScriptsEnvVars: true,
			description:             "Codex engine with mcp-scripts should include MCP Scripts env vars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			workflowPath := filepath.Join(tempDir, "test-workflow.md")

			err := os.WriteFile(workflowPath, []byte(tt.workflowContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write workflow file: %v", err)
			}

			compiler := NewCompiler()
			err = compiler.CompileWorkflow(workflowPath)
			if err != nil {
				t.Fatalf("Failed to compile workflow: %v", err)
			}

			lockPath := stringutil.MarkdownToLockFile(workflowPath)
			lockContent, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatalf("Failed to read lock file: %v", err)
			}

			yamlStr := string(lockContent)

			// Check for "Start MCP Gateway" step
			if !strings.Contains(yamlStr, "Start MCP Gateway") {
				t.Skip("No MCP gateway step generated (sandbox might be disabled)")
			}

			// Extract the MCP gateway step section
			startIdx := strings.Index(yamlStr, "Start MCP Gateway")
			if startIdx == -1 {
				t.Fatal("Start MCP Gateway step not found")
			}

			// Find the next step or end of steps section
			nextStepIdx := strings.Index(yamlStr[startIdx+1:], "- name:")
			var mcpGatewaySection string
			if nextStepIdx != -1 {
				mcpGatewaySection = yamlStr[startIdx : startIdx+1+nextStepIdx]
			} else {
				mcpGatewaySection = yamlStr[startIdx:]
			}

			// Check for MCP Scripts env vars in the env section
			hasEnvVarsInEnvSection := strings.Contains(mcpGatewaySection, "GH_AW_MCP_SCRIPTS_PORT:") &&
				strings.Contains(mcpGatewaySection, "GH_AW_MCP_SCRIPTS_API_KEY:")

			// Check for MCP Scripts env vars in the Docker command
			hasEnvVarsInDockerCmd := strings.Contains(mcpGatewaySection, "-e GH_AW_MCP_SCRIPTS_PORT") &&
				strings.Contains(mcpGatewaySection, "-e GH_AW_MCP_SCRIPTS_API_KEY")

			if tt.expectMCPScriptsEnvVars {
				if !hasEnvVarsInEnvSection {
					t.Errorf("%s: Expected GH_AW_MCP_SCRIPTS_PORT and GH_AW_MCP_SCRIPTS_API_KEY in env section but not found", tt.description)
				}
				if !hasEnvVarsInDockerCmd {
					t.Errorf("%s: Expected GH_AW_MCP_SCRIPTS_PORT and GH_AW_MCP_SCRIPTS_API_KEY in Docker command but not found", tt.description)
				}
			} else {
				if hasEnvVarsInEnvSection {
					t.Errorf("%s: Did not expect GH_AW_MCP_SCRIPTS_PORT or GH_AW_MCP_SCRIPTS_API_KEY in env section but found them", tt.description)
				}
				if hasEnvVarsInDockerCmd {
					t.Errorf("%s: Did not expect GH_AW_MCP_SCRIPTS_PORT or GH_AW_MCP_SCRIPTS_API_KEY in Docker command but found them", tt.description)
				}
			}
		})
	}
}

// TestMCPGatewayMCPScriptsValidation tests that the workflow fails validation
// if MCP Scripts env vars are referenced but the mcp-scripts-start step doesn't exist
func TestMCPGatewayMCPScriptsValidation(t *testing.T) {
	t.Skip("This test is for future validation logic - not implemented yet")
	// This would test that if someone manually adds MCP Scripts to MCP tools list
	// without actually configuring mcp-scripts, validation catches the error
}
