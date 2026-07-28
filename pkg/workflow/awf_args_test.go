//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
)

// TestBuildAWFArgsAuditDir tests that audit-dir and proxy-logs-dir are emitted in config,
// not CLI flags, for both standard and ARC/DinD workflows.
func TestBuildAWFArgsAuditDir(t *testing.T) {
	t.Run("non-arc-dind omits audit-dir and proxy-logs-dir from CLI flags", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "copilot",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		// Non-ARC/DinD: these should be in config, not CLI flags
		assert.NotContains(t, argsStr, "--audit-dir", "audit-dir should be in config for non-arc-dind")
		assert.NotContains(t, argsStr, "--proxy-logs-dir", "proxy-logs-dir should be in config for non-arc-dind")
	})

	t.Run("arc-dind also omits audit-dir and proxy-logs-dir from CLI flags", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
		}

		config := AWFCommandConfig{
			EngineName:     "copilot",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--audit-dir", "arc-dind audit-dir should be emitted via config JSON")
		assert.NotContains(t, argsStr, "--proxy-logs-dir", "arc-dind proxy-logs-dir should be emitted via config JSON")
	})
}

// TestBuildAWFArgsAllowHostPorts tests that BuildAWFArgs includes --allow-host-ports
// with port 80, 443, and the MCP gateway port so the AWF agent container can reach
// the gateway through the firewall's iptables rules.
func TestBuildAWFArgsAllowHostPorts(t *testing.T) {
	t.Run("includes default MCP gateway port 8080", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName: "copilot",
			WorkflowData: &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{LegacySecurity: true},
				},
			},
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--allow-host-ports", "Should include --allow-host-ports flag")
		assert.Contains(t, argsStr, "80,443,8080", "Should allow default gateway port 8080 alongside 80 and 443")
	})

	t.Run("uses custom MCP gateway port from sandbox config", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName: "copilot",
			WorkflowData: &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{LegacySecurity: true},
					MCP:   &MCPGatewayRuntimeConfig{Port: 9090},
				},
			},
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--allow-host-ports", "Should include --allow-host-ports flag")
		assert.Contains(t, argsStr, "80,443,9090", "Should use custom gateway port from sandbox config")
		assert.NotContains(t, argsStr, "8080", "Should not include default port when custom port is set")
	})

	t.Run("handles nil SandboxConfig gracefully — strict mode skips host-access", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName: "copilot",
			WorkflowData: &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
			},
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--allow-host-ports", "Strict mode (default) should not emit --allow-host-ports")
		assert.NotContains(t, argsStr, "--enable-host-access", "Strict mode (default) should not emit --enable-host-access")
	})

	t.Run("skips --allow-host-ports when AWF version is too old", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName: "copilot",
			WorkflowData: &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Enabled: true,
						Version: "v0.25.23",
					},
				},
			},
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--allow-host-ports", "Should skip --allow-host-ports for AWF versions below minimum support")
	})

	t.Run("skips host-access flags when network isolation is enabled", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName: "copilot",
			WorkflowData: &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: &EngineConfig{ID: "copilot"},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true},
				},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						Type:             SandboxTypeAWF,
						NetworkIsolation: true,
					},
				},
			},
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--enable-host-access", "Should skip --enable-host-access in network isolation mode")
		assert.NotContains(t, argsStr, "--allow-host-ports", "Should skip --allow-host-ports in network isolation mode")
	})
}

// TestBuildAWFArgsDiagnosticLogs tests that BuildAWFArgs includes --diagnostic-logs
// only when features.awf-diagnostic-logs is enabled.
func TestBuildAWFArgsDiagnosticLogs(t *testing.T) {
	baseWorkflow := func(features map[string]any) *WorkflowData {
		return &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
			Features: features,
		}
	}

	t.Run("does not include --diagnostic-logs when feature flag is absent", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			WorkflowData:   baseWorkflow(nil),
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--diagnostic-logs", "Should not include --diagnostic-logs when feature flag is absent")
	})

	t.Run("includes --diagnostic-logs when awf-diagnostic-logs is enabled", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName: "copilot",
			WorkflowData: baseWorkflow(map[string]any{
				string(constants.AwfDiagnosticLogsFeatureFlag): true,
			}),
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--diagnostic-logs", "Should include --diagnostic-logs when feature flag is enabled")
	})
}

// TestBuildAWFArgsMemoryLimit tests that BuildAWFArgs passes --memory-limit
// when sandbox.agent.memory is configured in the workflow frontmatter
func TestBuildAWFArgsMemoryLimit(t *testing.T) {
	t.Run("includes --memory-limit flag when memory is configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					Memory: "6g",
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "copilot",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--memory-limit", "Should include --memory-limit flag")
		assert.Contains(t, argsStr, "6g", "Should include the memory value")
	})

	t.Run("does not include --memory-limit flag when memory is not configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "copilot",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--memory-limit", "Should not include --memory-limit when memory is not configured")
	})

	t.Run("includes correct memory value when multiple sizes configured", func(t *testing.T) {
		for _, memory := range []string{"512m", "4g", "8g"} {
			t.Run(memory, func(t *testing.T) {
				workflowData := &WorkflowData{
					Name: "test-workflow",
					EngineConfig: &EngineConfig{
						ID: "copilot",
					},
					SandboxConfig: &SandboxConfig{
						Agent: &AgentSandboxConfig{
							Memory: memory,
						},
					},
				}

				config := AWFCommandConfig{
					EngineName:     "copilot",
					WorkflowData:   workflowData,
					AllowedDomains: "github.com",
				}

				args := BuildAWFArgs(config)
				argsStr := strings.Join(args, " ")

				assert.Contains(t, argsStr, "--memory-limit", "Should include --memory-limit flag")
				assert.Contains(t, argsStr, memory, "Should include the correct memory value")
			})
		}
	})
}

// TestGetAWFCommandPrefixNetworkIsolation tests that GetAWFCommandPrefix returns the correct
// command based on security mode: strict (default, no sudo) or legacy (sudo -E awf).
func TestGetAWFCommandPrefixNetworkIsolation(t *testing.T) {
	t.Run("returns awf (no sudo) when sudo is false (network isolation mode)", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:         "test-workflow",
			EngineConfig: &EngineConfig{ID: "copilot"},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:               "awf",
					NetworkIsolation: true,
				},
			},
		}
		cmd := GetAWFCommandPrefix(workflowData)
		assert.Equal(t, "awf", cmd, "Should return rootless 'awf' when sudo is false (network isolation mode)")
		assert.NotContains(t, cmd, "sudo", "Should not contain sudo when sudo is false (network isolation mode)")
	})

	t.Run("returns awf (no sudo) by default in strict security mode", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:         "test-workflow",
			EngineConfig: &EngineConfig{ID: "copilot"},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:               "awf",
					NetworkIsolation: false,
				},
			},
		}
		cmd := GetAWFCommandPrefix(workflowData)
		assert.Equal(t, "awf", cmd, "Should return 'awf' (no sudo) in strict security mode even with sudo: true")
	})

	t.Run("returns awf (no sudo) when no sandbox config is set", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:         "test-workflow",
			EngineConfig: &EngineConfig{ID: "copilot"},
		}
		cmd := GetAWFCommandPrefix(workflowData)
		assert.Equal(t, "awf", cmd, "Should return 'awf' (no sudo) when there is no sandbox config")
	})

	t.Run("returns sudo -E awf when legacy-security is enabled", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:         "test-workflow",
			EngineConfig: &EngineConfig{ID: "copilot"},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:             "awf",
					LegacySecurity: true,
				},
			},
		}
		cmd := GetAWFCommandPrefix(workflowData)
		assert.Equal(t, "sudo -E awf", cmd, "Should return 'sudo -E awf' when legacy-security is enabled")
	})

	t.Run("custom command takes precedence over sudo setting", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name:         "test-workflow",
			EngineConfig: &EngineConfig{ID: "copilot"},
			SandboxConfig: &SandboxConfig{
				Agent: &AgentSandboxConfig{
					ID:               "awf",
					NetworkIsolation: true,
					Command:          "custom-awf",
				},
			},
		}
		cmd := GetAWFCommandPrefix(workflowData)
		assert.Equal(t, "custom-awf", cmd, "Custom command should take precedence over sudo rootless mode")
	})
}

func TestBuildAWFArgs_LegacySecurityVersionGuard(t *testing.T) {
	t.Run("emits --legacy-security when AWF version supports it", func(t *testing.T) {
		config := AWFCommandConfig{
			WorkflowData: &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: &EngineConfig{ID: "copilot"},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID:             "awf",
						LegacySecurity: true,
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Version: "0.27.32",
					},
				},
			},
			EngineName: "copilot",
		}
		args := BuildAWFArgs(config)
		assert.Contains(t, args, "--legacy-security", "Should emit --legacy-security for AWF >= v0.27.32")
	})

	t.Run("skips --legacy-security when AWF version is too old", func(t *testing.T) {
		config := AWFCommandConfig{
			WorkflowData: &WorkflowData{
				Name:         "test-workflow",
				EngineConfig: &EngineConfig{ID: "copilot"},
				SandboxConfig: &SandboxConfig{
					Agent: &AgentSandboxConfig{
						ID:             "awf",
						LegacySecurity: true,
					},
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{
						Version: "0.27.30",
					},
				},
			},
			EngineName: "copilot",
		}
		args := BuildAWFArgs(config)
		assert.NotContains(t, args, "--legacy-security", "Should NOT emit --legacy-security for AWF < v0.27.32")
		// But should still emit --enable-host-access for backward compat
		assert.Contains(t, args, "--enable-host-access", "Should still emit --enable-host-access for legacy mode")
	})
}
