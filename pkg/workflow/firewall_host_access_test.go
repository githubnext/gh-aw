package workflow

import (
	"strings"
	"testing"
)

// TestFirewallEnableHostAccessWithMCPServers tests that --enable-host-access is added
// to AWF command when MCP servers are configured (mcpg is enabled)
func TestFirewallEnableHostAccessWithMCPServers(t *testing.T) {
	tests := []struct {
		name             string
		engineID         string
		tools            map[string]any
		expectHostAccess bool
	}{
		{
			name:     "copilot with github tool enables host access",
			engineID: "copilot",
			tools: map[string]any{
				"github": true,
			},
			expectHostAccess: true,
		},
		{
			name:             "copilot without tools does not enable host access",
			engineID:         "copilot",
			tools:            map[string]any{},
			expectHostAccess: false,
		},
		{
			name:     "copilot with playwright tool enables host access",
			engineID: "copilot",
			tools: map[string]any{
				"playwright": true,
			},
			expectHostAccess: true,
		},
		{
			name:     "copilot with disabled github tool does not enable host access",
			engineID: "copilot",
			tools: map[string]any{
				"github": false,
			},
			expectHostAccess: false,
		},
		{
			name:     "claude with github tool enables host access",
			engineID: "claude",
			tools: map[string]any{
				"github": true,
			},
			expectHostAccess: true,
		},
		{
			name:             "claude without tools does not enable host access",
			engineID:         "claude",
			tools:            map[string]any{},
			expectHostAccess: false,
		},
		{
			name:     "codex with github tool enables host access",
			engineID: "codex",
			tools: map[string]any{
				"github": true,
			},
			expectHostAccess: true,
		},
		{
			name:             "codex without tools does not enable host access",
			engineID:         "codex",
			tools:            map[string]any{},
			expectHostAccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: tt.engineID,
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
					},
				},
				Tools: tt.tools,
			}

			var steps []GitHubActionStep
			switch tt.engineID {
			case "copilot":
				engine := NewCopilotEngine()
				steps = engine.GetExecutionSteps(workflowData, "test.log")
			case "claude":
				engine := NewClaudeEngine()
				steps = engine.GetExecutionSteps(workflowData, "test.log")
			case "codex":
				engine := NewCodexEngine()
				steps = engine.GetExecutionSteps(workflowData, "test.log")
			}

			if len(steps) == 0 {
				t.Fatal("Expected at least one execution step")
			}

			stepContent := strings.Join(steps[0], "\n")

			hasHostAccess := strings.Contains(stepContent, "--enable-host-access")
			if hasHostAccess != tt.expectHostAccess {
				if tt.expectHostAccess {
					t.Errorf("Expected AWF command to contain '--enable-host-access' when MCP servers are configured")
				} else {
					t.Errorf("Expected AWF command to NOT contain '--enable-host-access' when no MCP servers are configured")
				}
			}
		})
	}
}
