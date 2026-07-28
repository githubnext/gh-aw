//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	assert.Contains(t, stepContent, `\"copilot\"`, "Should include copilot target in config JSON")
	assert.Contains(t, stepContent, "copilot-api.contoso-aw.ghe.com", "Should include custom Copilot hostname in config JSON")
	assert.NotContains(t, stepContent, "--copilot-api-target", "Should not emit --copilot-api-target as CLI flag")
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
	assert.Contains(t, stepContent, `\"gemini\"`, "Should include gemini target in config JSON")
	assert.Contains(t, stepContent, "generativelanguage.googleapis.com", "Should include default Gemini API hostname")
	assert.NotContains(t, stepContent, "--gemini-api-target", "Should not emit --gemini-api-target as CLI flag")
}
