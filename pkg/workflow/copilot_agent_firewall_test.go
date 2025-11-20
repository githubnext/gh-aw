package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCopilotAgentFileWithFirewall tests that when firewall is enabled and a custom agent is specified,
// the agent file is copied to the location where Copilot CLI expects to find it.
func TestCopilotAgentFileWithFirewall(t *testing.T) {
	tests := []struct {
		name             string
		firewallEnabled  bool
		agentFile        string
		expectAgentCopy  bool
		expectedIdentifier string
	}{
		{
			name:             "firewall enabled with custom agent",
			firewallEnabled:  true,
			agentFile:        ".github/agents/test-agent.md",
			expectAgentCopy:  true,
			expectedIdentifier: "test-agent",
		},
		{
			name:             "firewall enabled without custom agent",
			firewallEnabled:  true,
			agentFile:        "",
			expectAgentCopy:  false,
			expectedIdentifier: "",
		},
		{
			name:             "firewall disabled with custom agent",
			firewallEnabled:  false,
			agentFile:        ".github/agents/test-agent.md",
			expectAgentCopy:  false,
			expectedIdentifier: "test-agent",
		},
		{
			name:             "firewall disabled without custom agent",
			firewallEnabled:  false,
			agentFile:        "",
			expectAgentCopy:  false,
			expectedIdentifier: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create workflow data with firewall and agent configuration
			workflowData := &WorkflowData{
				Name:      "test-workflow",
				AgentFile: tt.agentFile,
				Tools:     map[string]any{},
			}

			// Configure firewall if enabled
			if tt.firewallEnabled {
				workflowData.NetworkPermissions = &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
					},
				}
			}

			// Create copilot engine and render MCP config
			engine := NewCopilotEngine()
			var yaml strings.Builder
			engine.RenderMCPConfig(&yaml, map[string]any{}, []string{}, workflowData)

			yamlContent := yaml.String()

			// Check for agent file copy commands
			if tt.expectAgentCopy {
				// Should create agents directory
				assert.Contains(t, yamlContent, "mkdir -p /home/runner/.copilot/agents",
					"Should create agents directory when firewall enabled with custom agent")

				// Should copy agent file from host filesystem (/host prefix)
				expectedCopy := "cp \"/host${GITHUB_WORKSPACE}/" + tt.agentFile + "\" \"/home/runner/.copilot/agents/" + tt.expectedIdentifier + ".md\""
				assert.Contains(t, yamlContent, expectedCopy,
					"Should copy agent file from /host${GITHUB_WORKSPACE} to /home/runner/.copilot/agents/")

				// Should echo confirmation message
				expectedEcho := "echo \"Copied agent file to /home/runner/.copilot/agents/" + tt.expectedIdentifier + ".md\""
				assert.Contains(t, yamlContent, expectedEcho,
					"Should echo agent file copy confirmation")
			} else {
				// Should NOT have agent-specific directory creation
				if tt.agentFile != "" && !tt.firewallEnabled {
					// When firewall is disabled, agent copy should not occur
					assert.NotContains(t, yamlContent, "mkdir -p /home/runner/.copilot/agents",
						"Should not create agents directory when firewall disabled")
				}

				if tt.agentFile == "" {
					// When no agent file is specified, no copy should occur
					assert.NotContains(t, yamlContent, "cp \"/host${GITHUB_WORKSPACE}/",
						"Should not copy agent file when no agent specified")
				}
			}
		})
	}
}

// TestCopilotAgentFilePathInFirewall tests that the agent file path uses /host prefix for AWF mounted filesystem
func TestCopilotAgentFilePathInFirewall(t *testing.T) {
	workflowData := &WorkflowData{
		Name:      "test-workflow",
		AgentFile: ".github/agents/my-custom-agent.md",
		Tools:     map[string]any{},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		},
	}

	engine := NewCopilotEngine()
	var yaml strings.Builder
	engine.RenderMCPConfig(&yaml, map[string]any{}, []string{}, workflowData)

	yamlContent := yaml.String()

	// Verify the path uses /host prefix (AWF mounts host filesystem to /host)
	assert.Contains(t, yamlContent, "/host${GITHUB_WORKSPACE}/.github/agents/my-custom-agent.md",
		"Agent file path should use /host prefix for AWF mounted filesystem")

	// Verify the target path uses the correct identifier
	assert.Contains(t, yamlContent, "/home/runner/.copilot/agents/my-custom-agent.md",
		"Agent file should be copied to /home/runner/.copilot/agents/ with correct identifier")
}


