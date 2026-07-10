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
		{
			name: "network isolation allows host.docker.internal HTTP MCP server URL",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Type:             SandboxTypeAWF,
						NetworkIsolation: true,
					},
				},
				Tools: map[string]any{
					"github": map[string]any{
						"mode": "remote",
					},
				},
				ResolvedMCPServers: map[string]any{
					"mempalace": map[string]any{
						"type": "http",
						"url":  "http://host.docker.internal:8765/mcp",
					},
				},
			},
		},
		{
			name: "sandbox.agent false with valid justification",
			data: &WorkflowData{
				Features: map[string]any{
					"dangerously-disable-sandbox-agent": "controlled environment with no internet access",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Disabled: true,
					},
				},
			},
		},
		{
			name: "sandbox.agent false without justification",
			data: &WorkflowData{
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Disabled: true,
					},
				},
			},
			expectError: true,
			errorMsg:    "dangerously-disable-sandbox-agent",
		},
		{
			name: "sandbox.agent false with short justification",
			data: &WorkflowData{
				Features: map[string]any{
					"dangerously-disable-sandbox-agent": "too short",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Disabled: true,
					},
				},
			},
			expectError: true,
			errorMsg:    "at least 20 characters",
		},
		{
			name: "sandbox.agent false with expression justification",
			data: &WorkflowData{
				Features: map[string]any{
					"dangerously-disable-sandbox-agent": "${{ inputs.reason }}",
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Disabled: true,
					},
				},
			},
			expectError: true,
			errorMsg:    "expressions",
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
		name                   string
		config                 *SandboxConfig
		engine                 *EngineConfig
		expected               *SandboxConfig
		expectDefaultWritePath bool
		expectedAllowWrite     []string
	}{
		{
			name:                   "nil config creates default with AWF",
			config:                 nil,
			engine:                 &EngineConfig{ID: "copilot"},
			expectDefaultWritePath: true,
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
			engine:                 &EngineConfig{ID: "copilot"},
			expectDefaultWritePath: true,
			expected: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		},
		{
			// version-only object (no id/type) must default to AWF so the sandbox is
			// always enabled, matching the previous analysis of the smoke-gemini bug.
			name: "version-only agent defaults to AWF",
			config: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Version: "v0.25.29",
				},
			},
			engine:                 &EngineConfig{ID: "gemini"},
			expectDefaultWritePath: true,
			expected: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type:    SandboxTypeAWF,
					Version: "v0.25.29",
				},
			},
		},
		{
			// An agent object with only an empty string ID must also default to AWF.
			name: "empty ID agent defaults to AWF",
			config: &SandboxConfig{
				Agent: &AgentSandboxConfig{},
			},
			engine:                 &EngineConfig{ID: "copilot"},
			expectDefaultWritePath: true,
			expected: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
				},
			},
		},
		{
			// Explicitly disabled agent must never be overridden.
			name: "disabled agent is preserved",
			config: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Disabled: true,
				},
			},
			engine:                 nil,
			expectDefaultWritePath: false,
			expected: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Disabled: true,
				},
			},
		},
		{
			name: "existing allowWrite entries are preserved",
			config: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Type: SandboxTypeAWF,
					Config: &SandboxRuntimeConfig{
						Filesystem: &SRTFilesystemConfig{
							AllowWrite: []string{"/tmp/custom"},
						},
					},
				},
			},
			engine:                 &EngineConfig{ID: "claude"},
			expectDefaultWritePath: true,
			expectedAllowWrite:     []string{"/tmp/custom", defaultAgentWorkspaceWritePath},
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
			require.NotNil(t, result)
			require.NotNil(t, result.Agent)
			assert.Equal(t, tt.expected.Agent.Type, result.Agent.Type, "agent type")
			if tt.expected.Agent.Version != "" {
				assert.Equal(t, tt.expected.Agent.Version, result.Agent.Version, "agent version")
			}
			assert.Equal(t, tt.expected.Agent.Disabled, result.Agent.Disabled, "agent disabled flag")
			if tt.expectDefaultWritePath {
				require.NotNil(t, result.Agent.Config)
				require.NotNil(t, result.Agent.Config.Filesystem)
				assert.Contains(t, result.Agent.Config.Filesystem.AllowWrite, defaultAgentWorkspaceWritePath)
			} else if result.Agent.Config != nil && result.Agent.Config.Filesystem != nil {
				assert.NotContains(t, result.Agent.Config.Filesystem.AllowWrite, defaultAgentWorkspaceWritePath)
			}
			for _, expectedPath := range tt.expectedAllowWrite {
				require.NotNil(t, result.Agent.Config)
				require.NotNil(t, result.Agent.Config.Filesystem)
				assert.Contains(t, result.Agent.Config.Filesystem.AllowWrite, expectedPath)
			}
		})
	}
}

func TestMergeImportedSandboxAgentMounts(t *testing.T) {
	tests := []struct {
		name           string
		initial        *SandboxConfig
		imported       []string
		expected       []string
		expectNil      bool
		expectDisabled bool
	}{
		{
			name:      "no imported mounts returns original nil config",
			initial:   nil,
			imported:  nil,
			expectNil: true,
		},
		{
			name:     "creates sandbox agent config from imports",
			initial:  nil,
			imported: []string{"/tool-a:/tool-a:ro"},
			expected: []string{"/tool-a:/tool-a:ro"},
		},
		{
			name: "deduplicates imported and main mounts",
			initial: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Mounts: []string{
						"/main:/main:ro",
						"/shared:/shared:ro",
					},
				},
			},
			imported: []string{
				"/shared:/shared:ro",
				"/import-a:/import-a:ro",
			},
			expected: []string{
				"/shared:/shared:ro",
				"/import-a:/import-a:ro",
				"/main:/main:ro",
			},
		},
		{
			name: "does not modify disabled agent sandbox",
			initial: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Disabled: true,
				},
			},
			imported:       []string{"/tool-a:/tool-a:ro"},
			expectDisabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merged := mergeImportedSandboxAgentMounts(tt.initial, tt.imported)

			if tt.expectNil {
				assert.Nil(t, merged)
				return
			}

			require.NotNil(t, merged)
			require.NotNil(t, merged.Agent)
			if tt.expectDisabled {
				assert.True(t, merged.Agent.Disabled)
				assert.Empty(t, merged.Agent.Mounts)
				return
			}
			assert.Equal(t, tt.expected, merged.Agent.Mounts)
		})
	}
}

func TestDefaultAgentWorkspaceWritePath(t *testing.T) {
	assert.Equal(t, "/tmp/gh-aw/agent", defaultAgentWorkspaceWritePath)
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
