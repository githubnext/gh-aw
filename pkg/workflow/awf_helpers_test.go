//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractAPITargetHost tests the extractAPITargetHost function that extracts
// hostnames from custom API base URLs in engine.env
func TestExtractAPITargetHost(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		envVar       string
		expected     string
	}{
		{
			name: "extracts hostname from HTTPS URL with path",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "https://llm-router.internal.example.com/v1",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "llm-router.internal.example.com",
		},
		{
			name: "extracts hostname from HTTP URL with port and path",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"ANTHROPIC_BASE_URL": "http://localhost:8080/v1",
					},
				},
			},
			envVar:   "ANTHROPIC_BASE_URL",
			expected: "localhost:8080",
		},
		{
			name: "handles hostname without protocol or path",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "api.openai.com",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "api.openai.com",
		},
		{
			name: "handles hostname with port but no protocol",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "localhost:8000",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "localhost:8000",
		},
		{
			name: "returns empty string when env var not set",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OTHER_VAR": "value",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "",
		},
		{
			name: "returns empty string when engine config is nil",
			workflowData: &WorkflowData{
				EngineConfig: nil,
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "",
		},
		{
			name:         "returns empty string when workflow data is nil",
			workflowData: nil,
			envVar:       "OPENAI_BASE_URL",
			expected:     "",
		},
		{
			name: "returns empty string for empty URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "",
		},
		{
			name: "extracts Azure OpenAI endpoint hostname",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": "https://my-resource.openai.azure.com/openai/deployments/gpt-4",
					},
				},
			},
			envVar:   "OPENAI_BASE_URL",
			expected: "my-resource.openai.azure.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractAPITargetHost(tt.workflowData, tt.envVar)
			assert.Equal(t, tt.expected, result, "Extracted hostname should match expected value")
		})
	}
}

// TestAWFCustomAPITargetFlags tests that BuildAWFConfigJSON includes custom API targets
// when OPENAI_BASE_URL or ANTHROPIC_BASE_URL are configured in engine.env.
// With config file support (default AWF version), API targets move to the JSON config
// rather than being emitted as --*-api-target CLI flags.
func TestAWFCustomAPITargetFlags(t *testing.T) {
	t.Run("includes openai target in config JSON when OPENAI_BASE_URL is configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
				Env: map[string]string{
					"OPENAI_BASE_URL": "https://llm-router.internal.example.com/v1",
					"OPENAI_API_KEY":  "${{ secrets.LLM_ROUTER_KEY }}",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "codex",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		// API targets are in the JSON config file, not in CLI args
		awfConfigJSON, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should succeed")
		assert.Contains(t, awfConfigJSON, `"openai"`, "Should include openai target in config JSON")
		assert.Contains(t, awfConfigJSON, "llm-router.internal.example.com", "Should include custom hostname in config JSON")

		// --openai-api-target should NOT appear as a CLI flag
		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")
		assert.NotContains(t, argsStr, "--openai-api-target", "Should not emit --openai-api-target as CLI flag when config file is used")
	})

	t.Run("includes anthropic target in config JSON when ANTHROPIC_BASE_URL is configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
				Env: map[string]string{
					"ANTHROPIC_BASE_URL": "https://claude-proxy.internal.company.com",
					"ANTHROPIC_API_KEY":  "${{ secrets.CLAUDE_PROXY_KEY }}",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "claude",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		awfConfigJSON, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should succeed")
		assert.Contains(t, awfConfigJSON, `"anthropic"`, "Should include anthropic target in config JSON")
		assert.Contains(t, awfConfigJSON, "claude-proxy.internal.company.com", "Should include custom hostname in config JSON")

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")
		assert.NotContains(t, argsStr, "--anthropic-api-target", "Should not emit --anthropic-api-target as CLI flag when config file is used")
	})

	t.Run("does not include api targets in config JSON when using default URLs", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
				// No custom OPENAI_BASE_URL
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "codex",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		awfConfigJSON, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should succeed")
		assert.NotContains(t, awfConfigJSON, `"openai"`, "Should not include openai target when not configured")
		assert.NotContains(t, awfConfigJSON, `"anthropic"`, "Should not include anthropic target when not configured")

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")
		assert.NotContains(t, argsStr, "--openai-api-target", "Should not include --openai-api-target when not configured")
		assert.NotContains(t, argsStr, "--anthropic-api-target", "Should not include --anthropic-api-target when not configured")
	})

	t.Run("includes both api targets in config JSON when both are configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "custom",
				Env: map[string]string{
					"OPENAI_BASE_URL":    "https://openai-proxy.company.com/v1",
					"ANTHROPIC_BASE_URL": "https://anthropic-proxy.company.com",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "custom",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		awfConfigJSON, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should succeed")
		assert.Contains(t, awfConfigJSON, `"openai"`, "Should include openai target")
		assert.Contains(t, awfConfigJSON, "openai-proxy.company.com", "Should include OpenAI custom hostname")
		assert.Contains(t, awfConfigJSON, `"anthropic"`, "Should include anthropic target")
		assert.Contains(t, awfConfigJSON, "anthropic-proxy.company.com", "Should include Anthropic custom hostname")

		// API targets should not appear as CLI flags
		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")
		assert.NotContains(t, argsStr, "--openai-api-target", "Should not emit --openai-api-target as CLI flag")
		assert.NotContains(t, argsStr, "--anthropic-api-target", "Should not emit --anthropic-api-target as CLI flag")
	})
}

// TestExtractAPIBasePath tests the extractAPIBasePath function that extracts
// path components from custom API base URLs in engine.env
func TestExtractAPIBasePath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{"databricks serving endpoint", "https://host.com/serving-endpoints", "/serving-endpoints"},
		{"azure openai deployment", "https://host.com/openai/deployments/gpt-4", "/openai/deployments/gpt-4"},
		{"simple path", "https://host.com/v1", "/v1"},
		{"trailing slash stripped", "https://host.com/api/", "/api"},
		{"multiple trailing slashes stripped", "https://host.com/api///", "/api"},
		{"no path", "https://host.com", ""},
		{"bare hostname", "host.com", ""},
		{"root path only", "https://host.com/", ""},
		{"query string stripped", "https://host.com/api?param=value", "/api"},
		{"fragment stripped", "https://host.com/api#section", "/api"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workflowData := &WorkflowData{
				EngineConfig: &EngineConfig{
					Env: map[string]string{
						"OPENAI_BASE_URL": tt.url,
					},
				},
			}
			result := extractAPIBasePath(workflowData, "OPENAI_BASE_URL")
			assert.Equal(t, tt.expected, result, "Extracted base path should match expected value")
		})
	}

	t.Run("returns empty string when workflow data is nil", func(t *testing.T) {
		result := extractAPIBasePath(nil, "OPENAI_BASE_URL")
		assert.Empty(t, result, "Should return empty string for nil workflow data")
	})

	t.Run("returns empty string when engine config is nil", func(t *testing.T) {
		workflowData := &WorkflowData{EngineConfig: nil}
		result := extractAPIBasePath(workflowData, "OPENAI_BASE_URL")
		assert.Empty(t, result, "Should return empty string when engine config is nil")
	})

	t.Run("returns empty string when env var not set", func(t *testing.T) {
		workflowData := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{"OTHER_VAR": "value"},
			},
		}
		result := extractAPIBasePath(workflowData, "OPENAI_BASE_URL")
		assert.Empty(t, result, "Should return empty string when env var not set")
	})
}

// TestAWFBasePathFlags tests that BuildAWFArgs includes --openai-api-base-path and
// --anthropic-api-base-path when the configured URLs contain a path component.
// Note: API targets (hosts) move to the JSON config file, while base paths remain
// as CLI flags — they are not yet represented in the AWF config file schema.
func TestAWFBasePathFlags(t *testing.T) {
	t.Run("includes openai-api-base-path when OPENAI_BASE_URL has path component", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
				Env: map[string]string{
					"OPENAI_BASE_URL": "https://stone-dataplatform.cloud.databricks.com/serving-endpoints",
					"OPENAI_API_KEY":  "${{ secrets.DATABRICKS_KEY }}",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "codex",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		// Base path is still a CLI flag (not in config file schema yet)
		assert.Contains(t, argsStr, "--openai-api-base-path", "Should include --openai-api-base-path flag")
		assert.Contains(t, argsStr, "/serving-endpoints", "Should include the path component")

		// API target (host) is now in the config JSON
		assert.NotContains(t, argsStr, "--openai-api-target", "API target should be in config JSON, not CLI args")
		awfConfigJSON, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should succeed")
		assert.Contains(t, awfConfigJSON, "stone-dataplatform.cloud.databricks.com", "Target host should be in config JSON")
	})

	t.Run("includes anthropic-api-base-path when ANTHROPIC_BASE_URL has path component", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
				Env: map[string]string{
					"ANTHROPIC_BASE_URL": "https://proxy.company.com/anthropic/v1",
					"ANTHROPIC_API_KEY":  "${{ secrets.ANTHROPIC_KEY }}",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "claude",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		// Base path is still a CLI flag
		assert.Contains(t, argsStr, "--anthropic-api-base-path", "Should include --anthropic-api-base-path flag")
		assert.Contains(t, argsStr, "/anthropic/v1", "Should include the path component")

		// API target (host) is now in the config JSON
		assert.NotContains(t, argsStr, "--anthropic-api-target", "API target should be in config JSON, not CLI args")
		awfConfigJSON, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should succeed")
		assert.Contains(t, awfConfigJSON, "proxy.company.com", "Target host should be in config JSON")
	})

	t.Run("does not include base-path flags when URLs have no path", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
				Env: map[string]string{
					"OPENAI_BASE_URL":    "https://openai-proxy.company.com",
					"ANTHROPIC_BASE_URL": "https://anthropic-proxy.company.com",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "codex",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--openai-api-base-path", "Should not include --openai-api-base-path when no path in URL")
		assert.NotContains(t, argsStr, "--anthropic-api-base-path", "Should not include --anthropic-api-base-path when no path in URL")
	})
}

// TestBuildAWFArgsAuditDir tests that BuildAWFArgs always includes --audit-dir
// pointing to the AWF audit directory for policy-manifest.json and other audit files
func TestBuildAWFArgsAuditDir(t *testing.T) {
	t.Run("includes --audit-dir flag with correct path", func(t *testing.T) {
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

		assert.Contains(t, argsStr, "--audit-dir", "Should include --audit-dir flag")
		assert.Contains(t, argsStr, "/tmp/gh-aw/sandbox/firewall/audit", "Should include the audit directory path")
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
					MCP: &MCPGatewayRuntimeConfig{Port: 9090},
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

	t.Run("handles nil SandboxConfig gracefully", func(t *testing.T) {
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

		assert.Contains(t, argsStr, "80,443,8080", "Should fall back to default port with nil SandboxConfig")
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

// TestEngineExecutionWithCustomAPITarget tests that engine execution steps include
// custom API targets when configured in engine.env.
// With config file support (default AWF version), API targets are in the JSON config.
func TestEngineExecutionWithCustomAPITarget(t *testing.T) {
	t.Run("Codex engine includes openai target in config JSON when OPENAI_BASE_URL is configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "codex",
				Env: map[string]string{
					"OPENAI_BASE_URL": "https://llm-router.internal.example.com/v1",
					"OPENAI_API_KEY":  "${{ secrets.LLM_ROUTER_KEY }}",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewCodexEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		assert.NotEmpty(t, steps, "Should generate execution steps")

		stepContent := strings.Join(steps[0], "\n")

		// API target is in the JSON config (in the printf command), not as a CLI flag
		assert.Contains(t, stepContent, `"openai"`, "Should include openai target in config JSON")
		assert.Contains(t, stepContent, "llm-router.internal.example.com", "Should include custom hostname in config JSON")
		assert.NotContains(t, stepContent, "--openai-api-target", "Should not emit --openai-api-target as CLI flag")
	})

	t.Run("Claude engine includes anthropic target in config JSON when ANTHROPIC_BASE_URL is configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
				Env: map[string]string{
					"ANTHROPIC_BASE_URL": "https://claude-proxy.internal.company.com",
					"ANTHROPIC_API_KEY":  "${{ secrets.CLAUDE_PROXY_KEY }}",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		engine := NewClaudeEngine()
		steps := engine.GetExecutionSteps(workflowData, "test.log")

		assert.NotEmpty(t, steps, "Should generate execution steps")

		stepContent := strings.Join(steps[0], "\n")

		// API target is in the JSON config (in the printf command), not as a CLI flag
		assert.Contains(t, stepContent, `"anthropic"`, "Should include anthropic target in config JSON")
		assert.Contains(t, stepContent, "claude-proxy.internal.company.com", "Should include custom hostname in config JSON")
		assert.NotContains(t, stepContent, "--anthropic-api-target", "Should not emit --anthropic-api-target as CLI flag")
	})
}

// TestGetCopilotAPITarget tests the GetCopilotAPITarget helper that resolves the effective
// Copilot API target from either engine.api-target or GITHUB_COPILOT_BASE_URL in engine.env.
func TestGetCopilotAPITarget(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		expected     string
	}{
		{
			name: "engine.api-target takes precedence over GITHUB_COPILOT_BASE_URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID:        "copilot",
					APITarget: "api.acme.ghe.com",
					Env: map[string]string{
						"GITHUB_COPILOT_BASE_URL": "https://other.endpoint.com",
					},
				},
			},
			expected: "api.acme.ghe.com",
		},
		{
			name: "GITHUB_COPILOT_BASE_URL used as fallback when api-target not set",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
					Env: map[string]string{
						"GITHUB_COPILOT_BASE_URL": "https://copilot-api.contoso-aw.ghe.com",
					},
				},
			},
			expected: "copilot-api.contoso-aw.ghe.com",
		},
		{
			name: "GITHUB_COPILOT_BASE_URL with path extracts hostname only",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
					Env: map[string]string{
						"GITHUB_COPILOT_BASE_URL": "https://copilot-proxy.corp.example.com/v1",
					},
				},
			},
			expected: "copilot-proxy.corp.example.com",
		},
		{
			name: "empty when neither api-target nor GITHUB_COPILOT_BASE_URL is set",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
			},
			expected: "",
		},
		{
			name:         "empty when workflowData is nil",
			workflowData: nil,
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetCopilotAPITarget(tt.workflowData)
			assert.Equal(t, tt.expected, result, "GetCopilotAPITarget should return expected hostname")
		})
	}
}

// TestCopilotEngineIncludesCopilotAPITargetFromEnvVar tests that the Copilot engine execution
// step includes the copilot API target in the JSON config when GITHUB_COPILOT_BASE_URL is
// configured in engine.env.
func TestCopilotEngineIncludesCopilotAPITargetFromEnvVar(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "copilot",
			Env: map[string]string{
				"GITHUB_COPILOT_BASE_URL": "https://copilot-api.contoso-aw.ghe.com",
			},
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		},
	}

	engine := NewCopilotEngine()
	steps := engine.GetExecutionSteps(workflowData, "test.log")

	assert.NotEmpty(t, steps, "Should generate execution steps")

	stepContent := strings.Join(steps[0], "\n")

	// With config file support, Copilot API target is in the JSON config (not as CLI flag)
	assert.Contains(t, stepContent, `"copilot"`, "Should include copilot target in config JSON")
	assert.Contains(t, stepContent, "copilot-api.contoso-aw.ghe.com", "Should include custom Copilot hostname in config JSON")
	assert.NotContains(t, stepContent, "--copilot-api-target", "Should not emit --copilot-api-target as CLI flag")
}

// TestAWFSupportsExcludeEnv verifies that --exclude-env is only enabled for AWF v0.25.3+.
func TestAWFSupportsExcludeEnv(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config (default version) supports --exclude-env",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version (default) supports --exclude-env",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "v0.25.3 supports --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.25.3"},
			want:           true,
		},
		{
			name:           "v0.26.0 supports --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.26.0"},
			want:           true,
		},
		{
			name:           "v0.27.0 supports --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.27.0"},
			want:           true,
		},
		{
			name:           "latest supports --exclude-env",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.0 does not support --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.25.0"},
			want:           false,
		},
		{
			name:           "v0.1.0 does not support --exclude-env",
			firewallConfig: &FirewallConfig{Version: "v0.1.0"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsExcludeEnv(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsExcludeEnv result")
		})
	}
}

// TestBuildAWFArgsCliProxy tests that BuildAWFArgs correctly injects --difc-proxy-host
// and --difc-proxy-ca-cert based on the cli-proxy feature flag.
func TestBuildAWFArgsCliProxy(t *testing.T) {
	baseWorkflow := func(features map[string]any, tools map[string]any) *WorkflowData {
		return &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true},
			},
			Features: features,
			Tools:    tools,
		}
	}

	t.Run("does not include cli-proxy flags when feature flag is absent", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName:     "copilot",
			WorkflowData:   baseWorkflow(nil, nil),
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--difc-proxy-host", "Should not include --difc-proxy-host when feature flag is absent")
		assert.NotContains(t, argsStr, "--difc-proxy-ca-cert", "Should not include --difc-proxy-ca-cert when feature flag is absent")
		assert.NotContains(t, argsStr, "--enable-cli-proxy", "Should not include deprecated --enable-cli-proxy")
		assert.NotContains(t, argsStr, "--cli-proxy-policy", "Should not include deprecated --cli-proxy-policy")
	})

	t.Run("includes --difc-proxy-host and --difc-proxy-ca-cert when cli-proxy is enabled", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName: "copilot",
			WorkflowData: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true, Version: "v0.26.0"},
				},
				Features: map[string]any{"cli-proxy": true},
			},
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--difc-proxy-host", "Should include --difc-proxy-host when cli-proxy is enabled")
		assert.Contains(t, argsStr, "host.docker.internal:18443", "Should use host.docker.internal:18443 as proxy host")
		assert.Contains(t, argsStr, "--difc-proxy-ca-cert", "Should include --difc-proxy-ca-cert")
		assert.Contains(t, argsStr, "/tmp/gh-aw/difc-proxy-tls/ca.crt", "Should use the correct CA cert path")
		assert.NotContains(t, argsStr, "--enable-cli-proxy", "Should not include deprecated --enable-cli-proxy")
		assert.NotContains(t, argsStr, "--cli-proxy-policy", "Should not include deprecated --cli-proxy-policy")
	})

	t.Run("does not include cli-proxy flags for copilot by default", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName: "copilot",
			WorkflowData: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true, Version: "v0.26.0"},
				},
				Features: map[string]any{},
			},
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--difc-proxy-host", "Should not include --difc-proxy-host for copilot by default")
		assert.NotContains(t, argsStr, "--difc-proxy-ca-cert", "Should not include --difc-proxy-ca-cert for copilot by default")
	})

	t.Run("does not include deprecated flags even with guard policy configured", func(t *testing.T) {
		config := AWFCommandConfig{
			EngineName: "copilot",
			WorkflowData: &WorkflowData{
				Name: "test-workflow",
				EngineConfig: &EngineConfig{
					ID: "copilot",
				},
				NetworkPermissions: &NetworkPermissions{
					Firewall: &FirewallConfig{Enabled: true, Version: "v0.26.0"},
				},
				Features: map[string]any{"cli-proxy": true},
				Tools: map[string]any{
					"github": map[string]any{
						"min-integrity": "approved",
					},
				},
			},
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--difc-proxy-host", "Should include --difc-proxy-host")
		assert.Contains(t, argsStr, "--difc-proxy-ca-cert", "Should include --difc-proxy-ca-cert")
		assert.NotContains(t, argsStr, "--enable-cli-proxy", "Should not include deprecated --enable-cli-proxy")
		assert.NotContains(t, argsStr, "--cli-proxy-policy", "Should not include deprecated --cli-proxy-policy")
	})

	t.Run("skips all cli-proxy flags when AWF version is too old", func(t *testing.T) {
		// Simulate a workflow that pins an AWF version older than AWFCliProxyMinVersion
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "copilot",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
					Version: "v0.25.16", // older than AWFCliProxyMinVersion v0.25.17
				},
			},
			Features: map[string]any{
				"cli-proxy": true,
			},
			Tools: map[string]any{
				"github": map[string]any{
					"min-integrity": "approved",
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

		assert.NotContains(t, argsStr, "--difc-proxy-host", "Should not include --difc-proxy-host for old AWF")
		assert.NotContains(t, argsStr, "--difc-proxy-ca-cert", "Should not include --difc-proxy-ca-cert for old AWF")
		assert.NotContains(t, argsStr, "--enable-cli-proxy", "Should not include deprecated --enable-cli-proxy")
	})
}

// TestAWFSupportsCliProxy tests the awfSupportsCliProxy version gate function.
func TestAWFSupportsCliProxy(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns true (uses default version)",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version returns true (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.17 supports CLI proxy flags (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.25.17"},
			want:           true,
		},
		{
			name:           "v0.26.0 supports CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.26.0"},
			want:           true,
		},
		{
			name:           "v0.27.0 supports CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.27.0"},
			want:           true,
		},
		{
			name:           "v0.25.16 does not support CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.25.16"},
			want:           false,
		},
		{
			name:           "v0.25.14 does not support CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.25.14"},
			want:           false,
		},
		{
			name:           "v0.1.0 does not support CLI proxy flags",
			firewallConfig: &FirewallConfig{Version: "v0.1.0"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsCliProxy(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsCliProxy result")
		})
	}
}

// TestAWFSupportsAllowHostPorts tests the awfSupportsAllowHostPorts version gate function.
func TestAWFSupportsAllowHostPorts(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns true (uses default version)",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version returns true (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.24 supports --allow-host-ports (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.25.24"},
			want:           true,
		},
		{
			name:           "v0.26.0 supports --allow-host-ports",
			firewallConfig: &FirewallConfig{Version: "v0.26.0"},
			want:           true,
		},
		{
			name:           "v0.25.23 does not support --allow-host-ports",
			firewallConfig: &FirewallConfig{Version: "v0.25.23"},
			want:           false,
		},
		{
			name:           "v0.1.0 does not support --allow-host-ports",
			firewallConfig: &FirewallConfig{Version: "v0.1.0"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsAllowHostPorts(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsAllowHostPorts result")
		})
	}
}

// TestAWFSupportsDockerHostPathPrefix tests the awfSupportsDockerHostPathPrefix version gate.
func TestAWFSupportsDockerHostPathPrefix(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns true (uses default version)",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version returns true (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.43 supports --docker-host-path-prefix (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.25.43"},
			want:           true,
		},
		{
			name:           "v0.25.42 does not support --docker-host-path-prefix",
			firewallConfig: &FirewallConfig{Version: "v0.25.42"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsDockerHostPathPrefix(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsDockerHostPathPrefix result")
		})
	}
}

// TestAWFSupportsTokenSteering tests the awfSupportsTokenSteering version gate.
func TestAWFSupportsTokenSteering(t *testing.T) {
	tests := []struct {
		name           string
		firewallConfig *FirewallConfig
		want           bool
	}{
		{
			name:           "nil firewall config returns true (uses default version)",
			firewallConfig: nil,
			want:           true,
		},
		{
			name:           "empty version returns true (uses default version)",
			firewallConfig: &FirewallConfig{},
			want:           true,
		},
		{
			name:           "latest returns true",
			firewallConfig: &FirewallConfig{Version: "latest"},
			want:           true,
		},
		{
			name:           "v0.25.44 supports token steering (exact minimum version)",
			firewallConfig: &FirewallConfig{Version: "v0.25.44"},
			want:           true,
		},
		{
			name:           "v0.25.43 does not support token steering",
			firewallConfig: &FirewallConfig{Version: "v0.25.43"},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awfSupportsTokenSteering(tt.firewallConfig)
			assert.Equal(t, tt.want, got, "awfSupportsTokenSteering result")
		})
	}
}

// TestGetGeminiAPITarget tests the GetGeminiAPITarget helper that resolves the effective
// Gemini API target from GEMINI_API_BASE_URL in engine.env or the default endpoint.
func TestGetGeminiAPITarget(t *testing.T) {
	tests := []struct {
		name         string
		workflowData *WorkflowData
		engineName   string
		expected     string
	}{
		{
			name: "returns default target for gemini engine with no custom URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "gemini",
				},
			},
			engineName: "gemini",
			expected:   "generativelanguage.googleapis.com",
		},
		{
			name: "custom GEMINI_API_BASE_URL takes precedence over default",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "gemini",
					Env: map[string]string{
						"GEMINI_API_BASE_URL": "https://gemini-proxy.internal.company.com/v1",
					},
				},
			},
			engineName: "gemini",
			expected:   "gemini-proxy.internal.company.com",
		},
		{
			name: "returns empty for non-gemini engine without custom URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "claude",
				},
			},
			engineName: "claude",
			expected:   "",
		},
		{
			name:         "returns empty when workflowData is nil",
			workflowData: nil,
			engineName:   "gemini",
			expected:     "generativelanguage.googleapis.com",
		},
		{
			name: "returns custom target for non-gemini engine with GEMINI_API_BASE_URL",
			workflowData: &WorkflowData{
				EngineConfig: &EngineConfig{
					ID: "custom",
					Env: map[string]string{
						"GEMINI_API_BASE_URL": "https://custom-proxy.example.com",
					},
				},
			},
			engineName: "custom",
			expected:   "custom-proxy.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetGeminiAPITarget(tt.workflowData, tt.engineName)
			assert.Equal(t, tt.expected, result, "GetGeminiAPITarget should return expected hostname")
		})
	}
}

// TestAWFGeminiAPITargetFlags tests that BuildAWFConfigJSON includes --gemini target
// for the Gemini engine with default and custom endpoints, while base paths remain CLI flags.
func TestAWFGeminiAPITargetFlags(t *testing.T) {
	t.Run("includes default gemini target in config JSON for gemini engine", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "gemini",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "gemini",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		// Gemini target is in the JSON config, not in CLI args
		awfConfigJSON, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should succeed")
		assert.Contains(t, awfConfigJSON, `"gemini"`, "Should include gemini target in config JSON")
		assert.Contains(t, awfConfigJSON, "generativelanguage.googleapis.com", "Should include default Gemini API hostname")

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")
		assert.NotContains(t, argsStr, "--gemini-api-target", "Should not emit --gemini-api-target as CLI flag")
	})

	t.Run("includes custom gemini target in config JSON when GEMINI_API_BASE_URL is configured", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "gemini",
				Env: map[string]string{
					"GEMINI_API_BASE_URL": "https://gemini-proxy.internal.company.com/v1",
					"GEMINI_API_KEY":      "${{ secrets.GEMINI_PROXY_KEY }}",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "gemini",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		awfConfigJSON, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should succeed")
		assert.Contains(t, awfConfigJSON, `"gemini"`, "Should include gemini target in config JSON")
		assert.Contains(t, awfConfigJSON, "gemini-proxy.internal.company.com", "Should include custom Gemini hostname")

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")
		assert.NotContains(t, argsStr, "--gemini-api-target", "Should not emit --gemini-api-target as CLI flag")
	})

	t.Run("does not include gemini target for non-gemini engine without custom URL", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "claude",
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "claude",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		awfConfigJSON, err := BuildAWFConfigJSON(config)
		require.NoError(t, err, "BuildAWFConfigJSON should succeed")
		assert.NotContains(t, awfConfigJSON, `"gemini"`, "Should not include gemini target for non-gemini engine")

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")
		assert.NotContains(t, argsStr, "--gemini-api-target", "Should not include --gemini-api-target for non-gemini engine")
	})

	t.Run("includes gemini-api-base-path when custom URL has path component", func(t *testing.T) {
		workflowData := &WorkflowData{
			Name: "test-workflow",
			EngineConfig: &EngineConfig{
				ID: "gemini",
				Env: map[string]string{
					"GEMINI_API_BASE_URL": "https://gemini-proxy.company.com/serving-endpoints",
					"GEMINI_API_KEY":      "${{ secrets.GEMINI_PROXY_KEY }}",
				},
			},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{
					Enabled: true,
				},
			},
		}

		config := AWFCommandConfig{
			EngineName:     "gemini",
			WorkflowData:   workflowData,
			AllowedDomains: "github.com",
		}

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		// Base path remains as a CLI flag (not in config file schema yet)
		assert.Contains(t, argsStr, "--gemini-api-base-path", "Should include --gemini-api-base-path flag")
		assert.Contains(t, argsStr, "/serving-endpoints", "Should include the path component")
	})
}

// TestGeminiEngineIncludesGeminiAPITarget tests that the Gemini engine execution
// step includes the gemini API target in the JSON config when firewall is enabled.
func TestGeminiEngineIncludesGeminiAPITarget(t *testing.T) {
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID: "gemini",
		},
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{
				Enabled: true,
			},
		},
	}

	engine := NewGeminiEngine()
	steps := engine.GetExecutionSteps(workflowData, "test.log")

	if len(steps) < 2 {
		t.Fatal("Expected at least two execution steps (settings + execution)")
	}

	// steps[0] = Write Gemini Config, steps[1] = Execute Gemini CLI
	stepContent := strings.Join(steps[1], "\n")

	// With config file support, Gemini target is in the JSON config (not as CLI flag)
	assert.Contains(t, stepContent, `"gemini"`, "Should include gemini target in config JSON")
	assert.Contains(t, stepContent, "generativelanguage.googleapis.com", "Should include default Gemini API hostname")
	assert.NotContains(t, stepContent, "--gemini-api-target", "Should not emit --gemini-api-target as CLI flag")
}

func TestBuildAWFImageTagWithDigests(t *testing.T) {
	t.Run("includes digest metadata for known firewall images", func(t *testing.T) {
		tag := buildAWFImageTagWithDigests("0.25.28", nil)

		assert.Contains(t, tag, "0.25.28", "should keep original AWF tag")
		assert.Contains(t, tag, "squid=sha256:", "should include squid digest metadata")
		assert.Contains(t, tag, "agent=sha256:", "should include agent digest metadata")
		assert.Contains(t, tag, "api-proxy=sha256:", "should include api-proxy digest metadata")
		assert.Contains(t, tag, "cli-proxy=sha256:", "should include cli-proxy digest metadata")
	})

	t.Run("leaves tag unchanged when digests are unavailable", func(t *testing.T) {
		tag := buildAWFImageTagWithDigests("0.0.1", nil)
		assert.Equal(t, "0.0.1", tag, "should not append digest metadata when no pins are available")
	})
}

func TestBuildAWFArgs_ImageTagIncludesDigests(t *testing.T) {
	// Use a version that has embedded container pins so we can verify digest metadata
	// is included in the AWF config JSON. Version 0.25.29 has full embedded pins.
	config := AWFCommandConfig{
		EngineName:     "copilot",
		AllowedDomains: "github.com",
		WorkflowData: &WorkflowData{
			EngineConfig: &EngineConfig{ID: "copilot"},
			NetworkPermissions: &NetworkPermissions{
				Firewall: &FirewallConfig{Enabled: true, Version: "0.25.29"},
			},
		},
	}

	// When the AWF version supports --config (default), --image-tag moves to the JSON config file.
	// Verify the config file JSON contains the image tag with digest metadata.
	awfConfigJSON, err := BuildAWFConfigJSON(config)
	require.NoError(t, err, "BuildAWFConfigJSON should not error")
	assert.Contains(t, awfConfigJSON, "imageTag", "expected imageTag in AWF config JSON")
	assert.Contains(t, awfConfigJSON, "squid=sha256:", "expected squid digest metadata in AWF config JSON")
	assert.Contains(t, awfConfigJSON, "agent=sha256:", "expected agent digest metadata in AWF config JSON")
	assert.Contains(t, awfConfigJSON, "api-proxy=sha256:", "expected api-proxy digest metadata in AWF config JSON")

	// --image-tag should NOT appear in the CLI args (it's in the config file).
	args := BuildAWFArgs(config)
	argsStr := strings.Join(args, " ")
	assert.NotContains(t, argsStr, "--image-tag", "expected --image-tag to be absent from CLI args when config file is used")
}
