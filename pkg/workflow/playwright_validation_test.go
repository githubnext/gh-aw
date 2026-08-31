//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidatePlaywrightMode tests the CLI-only Playwright mode validation.
func TestValidatePlaywrightMode(t *testing.T) {
	tests := []struct {
		name        string
		tools       map[string]any
		expectError bool
		errorSubstr string
	}{
		{
			name:  "playwright not configured",
			tools: map[string]any{},
		},
		{
			name:  "playwright set to false",
			tools: map[string]any{"playwright": false},
		},
		{
			name:  "playwright nil defaults to CLI",
			tools: map[string]any{"playwright": nil},
		},
		{
			name:  "playwright empty map defaults to CLI",
			tools: map[string]any{"playwright": map[string]any{}},
		},
		{
			name:        "playwright explicit MCP mode",
			tools:       map[string]any{"playwright": map[string]any{"mode": "mcp"}},
			expectError: true,
			errorSubstr: "built-in Playwright MCP support has been removed",
		},
		{
			name:  "playwright CLI mode",
			tools: map[string]any{"playwright": map[string]any{"mode": "cli"}},
		},
		{
			name:  "playwright CLI mode uppercase",
			tools: map[string]any{"playwright": map[string]any{"mode": "CLI"}},
		},
		{
			name:        "playwright mode expression is rejected",
			tools:       map[string]any{"playwright": map[string]any{"mode": "${{ inputs.playwright-mode }}"}},
			expectError: true,
			errorSubstr: "mode must be a literal value; expressions are not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiler := NewCompiler()

			workflowData := &WorkflowData{
				Tools: tt.tools,
			}

			err := compiler.validatePlaywrightMode(workflowData)

			if tt.expectError {
				require.Error(t, err, "expected an error but got none")
				require.ErrorContains(t, err, tt.errorSubstr,
					"error %q should contain %q", err.Error(), tt.errorSubstr)
			} else {
				assert.NoError(t, err, "expected no error")
			}
		})
	}
}

func TestCompileWorkflowRejectsPlaywrightMCPModeWithMigrationGuidance(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test-workflow.md")
	content := `---
on: push
engine: claude
tools:
  playwright:
    mode: mcp
---

# Test Workflow
`
	require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

	err := NewCompiler().CompileWorkflow(mdPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "built-in Playwright MCP support has been removed")
	assert.Contains(t, err.Error(), "playwright-cli <command>")
	assert.Contains(t, err.Error(), "mcp-servers")
}

// TestCompileWorkflowRejectsPlaywrightModeExpression ensures that a full compile
// of a workflow with an expression-valued tools.playwright.mode surfaces the
// field-specific error from validatePlaywrightMode, rather than the generic
// JSON schema enum error. This guards against the schema's enum constraint
// preempting the dedicated validator.
func TestCompileWorkflowRejectsPlaywrightModeExpression(t *testing.T) {
	tmpDir := t.TempDir()
	mdPath := filepath.Join(tmpDir, "test-workflow.md")
	content := `---
on: push
engine: claude
tools:
  playwright:
    mode: ${{ inputs.playwright-mode }}
---

# Test Workflow

Test playwright mode expression rejection.
`
	require.NoError(t, os.WriteFile(mdPath, []byte(content), 0644))

	compiler := NewCompiler()
	err := compiler.CompileWorkflow(mdPath)

	require.Error(t, err, "expected compilation to fail for expression-valued playwright mode")
	assert.Contains(t, err.Error(), "tools.playwright.mode")
	assert.Contains(t, err.Error(), "mode must be a literal value; expressions are not allowed")
}

// TestValidatePlaywrightModeNilWorkflow ensures no panic on nil/empty input.
func TestValidatePlaywrightModeNilWorkflow(t *testing.T) {
	compiler := NewCompiler()

	err := compiler.validatePlaywrightMode(nil)
	require.NoError(t, err, "nil workflowData should not return error")

	err = compiler.validatePlaywrightMode(&WorkflowData{Tools: nil})
	require.NoError(t, err, "nil tools should not return error")
}
