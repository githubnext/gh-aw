//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		assert.Contains(t, stepContent, `\"openai\"`, "Should include openai target in config JSON")
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
		assert.Contains(t, stepContent, `\"anthropic\"`, "Should include anthropic target in config JSON")
		assert.Contains(t, stepContent, "claude-proxy.internal.company.com", "Should include custom hostname in config JSON")
		assert.NotContains(t, stepContent, "--anthropic-api-target", "Should not emit --anthropic-api-target as CLI flag")
	})
}
