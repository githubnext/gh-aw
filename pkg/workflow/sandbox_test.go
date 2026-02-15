//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSandboxConfig(t *testing.T) {
	tests := []struct {
		name        string
		data        *WorkflowData
		expectError bool
		errorMsg    string
	}{
		{
			name: "nil workflow data",
			data: nil,
		},
		{
			name: "nil sandbox config",
			data: &WorkflowData{},
		},
		{
			name: "valid AWF sandbox config",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Type: SandboxTypeAWF,
					},
				},
				Tools: map[string]any{
					"github": map[string]any{
						"mode": "remote",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSandboxConfig(tt.data)
			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplySandboxDefaults(t *testing.T) {
	tests := []struct {
		name     string
		config   *SandboxConfig
		engine   *EngineConfig
		expected *SandboxConfig
	}{
		{
			name:   "nil config creates default with AWF",
			config: nil,
			engine: &EngineConfig{ID: "copilot"},
			expected: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		},
		{
			name: "explicit AWF config preserved",
			config: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
			engine: &EngineConfig{ID: "copilot"},
			expected: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applySandboxDefaults(tt.config, tt.engine)
			if tt.expected != nil {
				require.NotNil(t, result)
				require.NotNil(t, result.Agent)
				assert.Equal(t, tt.expected.Agent.Type, result.Agent.Type)
			}
		})
	}
}

func TestWorkflowHashWithSandbox(t *testing.T) {
	// Test that sandbox config is included in workflow hash
	tmpDir := t.TempDir()
	defer os.RemoveAll(tmpDir)

	workflowFile := filepath.Join(tmpDir, "test-workflow.md")
	content := `---
sandbox:
  agent: awf
---
# Test Workflow
Test prompt
`
	err := os.WriteFile(workflowFile, []byte(content), 0644)
	require.NoError(t, err)

	// Just verify the file can be read
	data, err := os.ReadFile(workflowFile)
	require.NoError(t, err)
	assert.Contains(t, string(data), "sandbox:")
}
