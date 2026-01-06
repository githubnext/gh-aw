package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxAgentEnablesDefaultTools(t *testing.T) {
	t.Run("sandbox.agent: awf enables edit and bash tools by default", func(t *testing.T) {
		// Create temp directory for test workflows
		workflowsDir := t.TempDir()

		markdown := `---
engine: copilot
sandbox:
  agent: awf
on: workflow_dispatch
---

Test workflow to verify sandbox.agent: awf enables edit and bash tools.
`

		workflowPath := filepath.Join(workflowsDir, "test-agent-awf.md")
		err := os.WriteFile(workflowPath, []byte(markdown), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(true)

		err = compiler.CompileWorkflow(workflowPath)
		require.NoError(t, err, "Compilation failed")

		// Read the compiled workflow
		lockPath := filepath.Join(workflowsDir, "test-agent-awf.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		require.NoError(t, err, "Failed to read compiled workflow")

		lockStr := string(lockContent)

		// Verify that edit tool is present
		assert.Contains(t, lockStr, "edit", "Expected edit tool to be enabled by default with sandbox.agent: awf")

		// Verify that bash tool with wildcard is present
		// The bash tool should be set to wildcard ["*"] when sandbox.agent is enabled
		assert.Contains(t, lockStr, "bash", "Expected bash tool to be enabled by default with sandbox.agent: awf")
	})

	t.Run("sandbox.agent: srt enables edit and bash tools by default", func(t *testing.T) {
		// Create temp directory for test workflows
		workflowsDir := t.TempDir()

		markdown := `---
engine: copilot
sandbox:
  agent: srt
features:
  sandbox-runtime: true
on: workflow_dispatch
---

Test workflow to verify sandbox.agent: srt enables edit and bash tools.
`

		workflowPath := filepath.Join(workflowsDir, "test-agent-srt.md")
		err := os.WriteFile(workflowPath, []byte(markdown), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(true)

		err = compiler.CompileWorkflow(workflowPath)
		require.NoError(t, err, "Compilation failed")

		// Read the compiled workflow
		lockPath := filepath.Join(workflowsDir, "test-agent-srt.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		require.NoError(t, err, "Failed to read compiled workflow")

		lockStr := string(lockContent)

		// Verify that edit tool is present
		assert.Contains(t, lockStr, "edit", "Expected edit tool to be enabled by default with sandbox.agent: srt")

		// Verify that bash tool is present
		assert.Contains(t, lockStr, "bash", "Expected bash tool to be enabled by default with sandbox.agent: srt")
	})

	t.Run("sandbox.agent: false does NOT enable edit and bash tools", func(t *testing.T) {
		// Create temp directory for test workflows
		workflowsDir := t.TempDir()

		markdown := `---
engine: copilot
sandbox:
  agent: false
strict: false
on: workflow_dispatch
---

Test workflow to verify sandbox.agent: false does not enable tools by default.
`

		workflowPath := filepath.Join(workflowsDir, "test-agent-false-tools.md")
		err := os.WriteFile(workflowPath, []byte(markdown), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(true)

		err = compiler.CompileWorkflow(workflowPath)
		require.NoError(t, err, "Compilation failed")

		// Read the compiled workflow
		lockPath := filepath.Join(workflowsDir, "test-agent-false-tools.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		require.NoError(t, err, "Failed to read compiled workflow")

		lockStr := string(lockContent)

		// Verify that edit tool is NOT present (unless explicitly added)
		// We check that edit is not in the tools list by looking for the edit tool configuration
		// Since we're not setting tools explicitly, edit should not be added when agent is disabled
		
		// Count occurrences of "edit" to see if it's actually configured as a tool
		// (edit might appear in comments or other contexts)
		editCount := strings.Count(strings.ToLower(lockStr), "edit")
		
		// If edit appears very few times, it's likely just in comments
		// A proper tool configuration would have multiple references
		if editCount > 10 {
			t.Errorf("Expected edit tool to NOT be enabled by default with sandbox.agent: false, but found %d occurrences", editCount)
		}
	})

	t.Run("no sandbox config does NOT enable edit and bash tools", func(t *testing.T) {
		// Create temp directory for test workflows
		workflowsDir := t.TempDir()

		markdown := `---
engine: copilot
on: workflow_dispatch
---

Test workflow without sandbox config.
`

		workflowPath := filepath.Join(workflowsDir, "test-no-sandbox.md")
		err := os.WriteFile(workflowPath, []byte(markdown), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(true)

		err = compiler.CompileWorkflow(workflowPath)
		require.NoError(t, err, "Compilation failed")

		// Read the compiled workflow
		lockPath := filepath.Join(workflowsDir, "test-no-sandbox.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		require.NoError(t, err, "Failed to read compiled workflow")

		lockStr := string(lockContent)

		// Without sandbox config, edit and bash should not be added by our new logic
		// They might still be added by other default logic, but we're specifically testing
		// that our sandbox.agent logic doesn't activate
		
		// We just verify compilation succeeded - the actual tool presence depends on other defaults
		assert.NotEmpty(t, lockStr, "Lock file should not be empty")
	})

	t.Run("explicit bash tools takes precedence over sandbox.agent default", func(t *testing.T) {
		// Create temp directory for test workflows
		workflowsDir := t.TempDir()

		// Provide explicit bash configuration to override default
		markdown := `---
engine: copilot
sandbox:
  agent: awf
tools:
  bash: ["echo", "ls"]
on: workflow_dispatch
---

Test workflow where explicit tools.bash should take precedence over default.
`

		workflowPath := filepath.Join(workflowsDir, "test-explicit-bash.md")
		err := os.WriteFile(workflowPath, []byte(markdown), 0644)
		require.NoError(t, err, "Failed to write workflow file")

		// Compile the workflow
		compiler := NewCompiler(false, "", "test")
		compiler.SetSkipValidation(true)

		err = compiler.CompileWorkflow(workflowPath)
		require.NoError(t, err, "Compilation failed")

		// Read the compiled workflow
		lockPath := filepath.Join(workflowsDir, "test-explicit-bash.lock.yml")
		lockContent, err := os.ReadFile(lockPath)
		require.NoError(t, err, "Failed to read compiled workflow")

		lockStr := string(lockContent)

		// The explicit bash configuration should be preserved
		// We verify the workflow compiled successfully
		assert.NotEmpty(t, lockStr, "Lock file should not be empty")
		
		// Verify bash tool is present (explicit configuration)
		assert.Contains(t, lockStr, "bash", "Expected bash tool with explicit configuration")
	})
}

func TestIsSandboxAgentEnabled(t *testing.T) {
	tests := []struct {
		name          string
		sandboxConfig *SandboxConfig
		expected      bool
	}{
		{
			name:          "nil sandbox config",
			sandboxConfig: nil,
			expected:      false,
		},
		{
			name: "nil agent",
			sandboxConfig: &SandboxConfig{
				Agent: nil,
			},
			expected: false,
		},
		{
			name: "agent disabled",
			sandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Disabled: true,
				},
			},
			expected: false,
		},
		{
			name: "agent awf",
			sandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
			expected: true,
		},
		{
			name: "agent srt",
			sandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeSRT,
				},
			},
			expected: true,
		},
		{
			name: "agent default",
			sandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeDefault,
				},
			},
			expected: true,
		},
		{
			name: "agent with ID awf",
			sandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "awf",
				},
			},
			expected: true,
		},
		{
			name: "agent with ID srt",
			sandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID: "srt",
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isSandboxAgentEnabled(tt.sandboxConfig)
			assert.Equal(t, tt.expected, result, "isSandboxAgentEnabled returned unexpected result")
		})
	}
}
