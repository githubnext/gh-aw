//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

// TestAWFCustomAPITargetFlags tests that BuildAWFArgs includes custom API target flags
// when OPENAI_BASE_URL or ANTHROPIC_BASE_URL are configured in engine.env
func TestAWFCustomAPITargetFlags(t *testing.T) {
	t.Run("includes openai-api-target flag when OPENAI_BASE_URL is configured", func(t *testing.T) {
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

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--openai-api-target", "Should include --openai-api-target flag")
		assert.Contains(t, argsStr, "llm-router.internal.example.com", "Should include custom hostname")
	})

	t.Run("includes anthropic-api-target flag when ANTHROPIC_BASE_URL is configured", func(t *testing.T) {
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

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--anthropic-api-target", "Should include --anthropic-api-target flag")
		assert.Contains(t, argsStr, "claude-proxy.internal.company.com", "Should include custom hostname")
	})

	t.Run("does not include api-target flags when using default URLs", func(t *testing.T) {
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

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.NotContains(t, argsStr, "--openai-api-target", "Should not include --openai-api-target when not configured")
		assert.NotContains(t, argsStr, "--anthropic-api-target", "Should not include --anthropic-api-target when not configured")
	})

	t.Run("includes both api-target flags when both are configured", func(t *testing.T) {
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

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--openai-api-target", "Should include --openai-api-target flag")
		assert.Contains(t, argsStr, "openai-proxy.company.com", "Should include OpenAI custom hostname")
		assert.Contains(t, argsStr, "--anthropic-api-target", "Should include --anthropic-api-target flag")
		assert.Contains(t, argsStr, "anthropic-proxy.company.com", "Should include Anthropic custom hostname")
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
// --anthropic-api-base-path when the configured URLs contain a path component
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

		assert.Contains(t, argsStr, "--openai-api-target", "Should include --openai-api-target flag")
		assert.Contains(t, argsStr, "--openai-api-base-path", "Should include --openai-api-base-path flag")
		assert.Contains(t, argsStr, "/serving-endpoints", "Should include the path component")
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

		assert.Contains(t, argsStr, "--anthropic-api-target", "Should include --anthropic-api-target flag")
		assert.Contains(t, argsStr, "--anthropic-api-base-path", "Should include --anthropic-api-base-path flag")
		assert.Contains(t, argsStr, "/anthropic/v1", "Should include the path component")
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
// custom API target flags when configured in engine.env
func TestEngineExecutionWithCustomAPITarget(t *testing.T) {
	t.Run("Codex engine includes openai-api-target flag when OPENAI_BASE_URL is configured", func(t *testing.T) {
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

		assert.Contains(t, stepContent, "--openai-api-target", "Should include --openai-api-target flag")
		assert.Contains(t, stepContent, "llm-router.internal.example.com", "Should include custom hostname")
	})

	t.Run("Claude engine includes anthropic-api-target flag when ANTHROPIC_BASE_URL is configured", func(t *testing.T) {
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

		assert.Contains(t, stepContent, "--anthropic-api-target", "Should include --anthropic-api-target flag")
		assert.Contains(t, stepContent, "claude-proxy.internal.company.com", "Should include custom hostname")
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
// step includes --copilot-api-target when GITHUB_COPILOT_BASE_URL is configured in engine.env.
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

	assert.Contains(t, stepContent, "--copilot-api-target", "Should include --copilot-api-target flag")
	assert.Contains(t, stepContent, "copilot-api.contoso-aw.ghe.com", "Should include custom Copilot hostname")
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

// TestAWFGeminiAPITargetFlags tests that BuildAWFArgs includes --gemini-api-target flag
// for the Gemini engine with default and custom endpoints.
func TestAWFGeminiAPITargetFlags(t *testing.T) {
	t.Run("includes default gemini-api-target flag for gemini engine", func(t *testing.T) {
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

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--gemini-api-target", "Should include --gemini-api-target flag")
		assert.Contains(t, argsStr, "generativelanguage.googleapis.com", "Should include default Gemini API hostname")
	})

	t.Run("includes custom gemini-api-target flag when GEMINI_API_BASE_URL is configured", func(t *testing.T) {
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

		args := BuildAWFArgs(config)
		argsStr := strings.Join(args, " ")

		assert.Contains(t, argsStr, "--gemini-api-target", "Should include --gemini-api-target flag")
		assert.Contains(t, argsStr, "gemini-proxy.internal.company.com", "Should include custom Gemini hostname")
	})

	t.Run("does not include gemini-api-target for non-gemini engine without custom URL", func(t *testing.T) {
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

		assert.Contains(t, argsStr, "--gemini-api-base-path", "Should include --gemini-api-base-path flag")
		assert.Contains(t, argsStr, "/serving-endpoints", "Should include the path component")
	})
}

// TestGeminiEngineIncludesGeminiAPITarget tests that the Gemini engine execution
// step includes --gemini-api-target when firewall is enabled.
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

	// steps[0] = Write Gemini Settings, steps[1] = Execute Gemini CLI
	stepContent := strings.Join(steps[1], "\n")

	assert.Contains(t, stepContent, "--gemini-api-target", "Should include --gemini-api-target flag")
	assert.Contains(t, stepContent, "generativelanguage.googleapis.com", "Should include default Gemini API hostname")
}
