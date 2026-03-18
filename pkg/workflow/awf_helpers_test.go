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
