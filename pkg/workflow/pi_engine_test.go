//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPiEngine(t *testing.T) {
	engine := NewPiEngine()
	require.NotNil(t, engine, "NewPiEngine should return a non-nil engine")
	assert.Equal(t, "pi", engine.GetID(), "Engine ID should be 'pi'")
	assert.Equal(t, "Pi", engine.GetDisplayName(), "Display name should be 'Pi'")
	assert.True(t, engine.IsExperimental(), "Pi engine should be experimental")
	capabilities := engine.GetCapabilities()
	assert.True(t, capabilities.ToolsAllowlist, "Pi should support tools allowlist (needed for gh-proxy/cli-proxy settings)")
	assert.False(t, capabilities.MaxTurns, "Pi should not support max turns")
}

func TestPiEngine_GetModelEnvVarName(t *testing.T) {
	engine := NewPiEngine()
	assert.Equal(t, "PI_MODEL", engine.GetModelEnvVarName(), "Model env var should be PI_MODEL")
}

func TestPiEngine_GetRequiredSecretNames(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{Name: "test-workflow"}
	secrets := engine.GetRequiredSecretNames(workflowData)
	assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN", "Required secrets should include COPILOT_GITHUB_TOKEN")
	assert.NotContains(t, secrets, "PI_API_KEY", "Required secrets should not include PI_API_KEY")
}

func TestPiEngine_GetRequiredSecretNames_CopilotProvider(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "copilot/claude-sonnet-4-20250514"},
	}
	secrets := engine.GetRequiredSecretNames(workflowData)
	assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN", "copilot/ prefix should require COPILOT_GITHUB_TOKEN")
}

func TestPiEngine_GetRequiredSecretNames_AnthropicProvider(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "anthropic/claude-sonnet-4-20250514"},
	}
	secrets := engine.GetRequiredSecretNames(workflowData)
	assert.Contains(t, secrets, "ANTHROPIC_API_KEY", "anthropic/ prefix should require ANTHROPIC_API_KEY")
	assert.NotContains(t, secrets, "COPILOT_GITHUB_TOKEN", "anthropic/ prefix should not require COPILOT_GITHUB_TOKEN")
}

func TestPiEngine_GetRequiredSecretNames_CodexProvider(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "codex/gpt-4o"},
	}
	secrets := engine.GetRequiredSecretNames(workflowData)
	assert.Contains(t, secrets, "CODEX_API_KEY", "codex/ prefix should require CODEX_API_KEY")
	assert.Contains(t, secrets, "OPENAI_API_KEY", "codex/ prefix should also require OPENAI_API_KEY (from Codex backend profile)")
}

func TestPiEngine_GetRequiredSecretNames_NoPrefix(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "claude-sonnet-4-20250514"},
	}
	secrets := engine.GetRequiredSecretNames(workflowData)
	assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN", "bare model (no prefix) should default to COPILOT_GITHUB_TOKEN")
}

func TestPiEngine_GetLogParserScriptId(t *testing.T) {
	engine := NewPiEngine()
	assert.Equal(t, "parse_pi_log", engine.GetLogParserScriptId(), "Log parser script ID should be parse_pi_log")
}

func TestPiEngine_GetLogFileForParsing(t *testing.T) {
	engine := NewPiEngine()
	assert.Equal(t, PiStreamingLogFile, engine.GetLogFileForParsing(), "Log file for parsing should be PiStreamingLogFile")
}

func TestPiEngine_GetAgentManifestFiles(t *testing.T) {
	engine := NewPiEngine()
	files := engine.GetAgentManifestFiles()
	assert.Contains(t, files, "PI.md", "Manifest files should include PI.md")
	assert.Contains(t, files, "AGENTS.md", "Manifest files should include AGENTS.md")
}

func TestPiEngine_GetAgentManifestPathPrefixes(t *testing.T) {
	engine := NewPiEngine()
	prefixes := engine.GetAgentManifestPathPrefixes()
	assert.Contains(t, prefixes, ".pi/", "Path prefixes should include .pi/")
}

func TestPiEngine_GetDeclaredOutputFiles(t *testing.T) {
	engine := NewPiEngine()
	files := engine.GetDeclaredOutputFiles()
	assert.Contains(t, files, PiStreamingLogFile, "Declared output files should include the streaming log")
}

func TestPiEngine_GetInstallationSteps_NoCustomCommand(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi"},
	}
	steps := engine.GetInstallationSteps(workflowData)
	assert.NotEmpty(t, steps, "Installation steps should not be empty")

	// The steps should reference @earendil-works/pi-coding-agent
	found := false
	for _, step := range steps {
		for _, line := range step {
			if strings.Contains(line, "@earendil-works/pi-coding-agent") {
				found = true
				break
			}
		}
	}
	assert.True(t, found, "Installation steps should install @earendil-works/pi-coding-agent")
}

func TestPiEngine_GetInstallationSteps_WithCustomCommand(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Command: "/custom/pi"},
	}
	steps := engine.GetInstallationSteps(workflowData)
	assert.Empty(t, steps, "Installation steps should be skipped when custom command is set")
}

func TestPiEngine_GetInstallationSteps_WithExtensions(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
		EngineConfig: &EngineConfig{
			ID:         "pi",
			Extensions: []string{"@pi/web-search", "@pi/file-browser"},
		},
	}
	steps := engine.GetInstallationSteps(workflowData)
	require.NotEmpty(t, steps, "Steps should not be empty with extensions")

	// Find extension install steps
	var extensionSteps []GitHubActionStep
	for _, step := range steps {
		for _, line := range step {
			if strings.Contains(line, "Install Pi extension") {
				extensionSteps = append(extensionSteps, step)
				break
			}
		}
	}
	assert.Len(t, extensionSteps, 2, "Should have 2 extension install steps")
}

func TestPiEngine_GetExecutionSteps_Basic(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi"},
		ParsedTools:  NewTools(map[string]any{}),
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
	require.Len(t, steps, 1, "Should produce exactly one execution step")

	stepText := strings.Join(steps[0], "\n")
	assert.Contains(t, stepText, "Execute Pi CLI", "Step should be named 'Execute Pi CLI'")
	assert.Contains(t, stepText, "--print", "Step should use --print flag (non-interactive mode)")
	assert.Contains(t, stepText, "--mode json", "Step should use --mode json for structured JSONL output")
	assert.NotContains(t, stepText, "pi run", "Step should not use the removed 'pi run' subcommand")
	assert.NotContains(t, stepText, "--json-log", "Step should not use the removed --json-log flag")
	assert.Contains(t, stepText, "agentic_execution", "Step should have agentic_execution id")
	assert.Contains(t, stepText, "pi_provider.cjs", "Step should load the provider extension")
	assert.Contains(t, stepText, "pi_steering_extension.cjs", "Step should automatically load the steering extension")
}

func TestPiEngine_GetExecutionSteps_WithModel(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "copilot/claude-sonnet-4"},
		ParsedTools:  NewTools(map[string]any{}),
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
	require.NotEmpty(t, steps, "Steps should not be empty")

	stepText := strings.Join(steps[0], "\n")
	// When firewall is not enabled, Pi is invoked with the --model flag using the
	// native github-copilot provider (Pi's built-in provider for GitHub Copilot).
	assert.Contains(t, stepText, "--model", "Step should pass --model flag to Pi CLI")
	assert.Contains(t, stepText, "github-copilot", "Non-firewall copilot model should use github-copilot/ provider prefix")
	assert.Contains(t, stepText, "claude-sonnet-4", "Step should include the model ID portion")
	assert.Contains(t, stepText, "GH_AW_PI_MODEL", "Step should expose the original workflow model to Pi extensions")
	assert.NotContains(t, stepText, "\n          PI_MODEL:", "Step should not set PI_MODEL in the environment when the CLI model is passed via --model")
}

func TestPiEngine_GetExecutionSteps_ProviderPrefixCopilot(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "copilot/claude-sonnet-4-20250514"},
		ParsedTools:  NewTools(map[string]any{}),
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
	require.Len(t, steps, 1, "Should produce exactly one execution step")

	stepText := strings.Join(steps[0], "\n")
	assert.Contains(t, stepText, "COPILOT_GITHUB_TOKEN", "copilot/ prefix should inject COPILOT_GITHUB_TOKEN")
	// OPENAI_API_KEY must not be injected: Pi reads it and routes to api.openai.com,
	// bypassing the github-copilot provider and the AWF firewall.
	assert.NotContains(t, stepText, "OPENAI_API_KEY", "copilot/ prefix must not inject OPENAI_API_KEY (causes Pi to use OpenAI instead of github-copilot)")
	assert.Contains(t, stepText, "pi_provider.cjs", "Step should load the provider extension")
	assert.Contains(t, stepText, "--model", "Step should pass --model flag to Pi CLI")
}

func TestPiEngine_GetExecutionSteps_ProviderPrefixAnthropic(t *testing.T) {
	engine := NewPiEngine()
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "anthropic/claude-sonnet-4-20250514"},
		ParsedTools:  NewTools(map[string]any{}),
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
	require.Len(t, steps, 1, "Should produce exactly one execution step")

	stepText := strings.Join(steps[0], "\n")
	assert.Contains(t, stepText, "ANTHROPIC_API_KEY", "anthropic/ prefix should inject ANTHROPIC_API_KEY")
	assert.NotContains(t, stepText, "COPILOT_GITHUB_TOKEN", "anthropic/ prefix should not inject COPILOT_GITHUB_TOKEN")
}

func TestPiEngine_ImplementsCodingAgentEngine(t *testing.T) {
	var _ CodingAgentEngine = NewPiEngine()
}

func TestPiEngine_GetExecutionSteps_FirewallCopilotProvider(t *testing.T) {
	engine := NewPiEngine()
	toolsRaw := map[string]any{
		"github":    map[string]any{"mode": "gh-proxy"},
		"cli-proxy": true,
	}
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "copilot/claude-sonnet-4-20250514"},
		Tools:        toolsRaw,
		ParsedTools:  NewTools(toolsRaw),
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
	require.Len(t, steps, 1, "Should produce exactly one execution step")

	stepText := strings.Join(steps[0], "\n")
	// When firewall is enabled, Pi uses models.json to route through the api-proxy gateway.
	assert.Contains(t, stepText, "PI_CODING_AGENT_DIR", "Firewall mode should set PI_CODING_AGENT_DIR for models.json config")
	assert.Contains(t, stepText, "PI_CODING_AGENT_DIR: /tmp/gh-aw/pi-agent-dir", "PI_CODING_AGENT_DIR should point to the models.json directory")
	assert.Contains(t, stepText, "models.json", "Firewall mode should write a models.json gateway config")
	assert.Contains(t, stepText, "aw-gateway", "Firewall mode should register the aw-gateway provider in models.json")
	assert.Contains(t, stepText, "claude-sonnet-4-20250514", "Step should include the model ID in models.json")
	// AWF config JSON embedded in step must enable the api-proxy sidecar.
	assert.Contains(t, stepText, `"enabled":true`, "Firewall mode should enable the api-proxy in AWF config JSON")
	// The models.json is embedded in the step as a printf argument. Verify the correct
	// Copilot gateway port is present by re-building the expected JSON.
	// models.json must use the "api-proxy" Docker service hostname, not host.docker.internal.
	// host.docker.internal resolves to the runner host, NOT the api-proxy sidecar container.
	expectedModelsJSON := buildPiModelsJSON(constants.CopilotLLMGatewayPort, "COPILOT_GITHUB_TOKEN", "claude-sonnet-4-20250514")
	assert.Contains(t, expectedModelsJSON, "api-proxy:", "models.json baseUrl must use the api-proxy Docker hostname within the AWF network")
	assert.NotContains(t, expectedModelsJSON, "host.docker.internal", "models.json baseUrl must not use host.docker.internal (not the api-proxy)")
	assert.Contains(t, stepText, expectedModelsJSON, "Copilot provider should route through CopilotLLMGatewayPort via models.json")
}

func TestPiEngine_GetExecutionSteps_FirewallAnthropicProvider(t *testing.T) {
	engine := NewPiEngine()
	toolsRaw := map[string]any{
		"github":    map[string]any{"mode": "gh-proxy"},
		"cli-proxy": true,
	}
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "anthropic/claude-opus-4-20251101"},
		Tools:        toolsRaw,
		ParsedTools:  NewTools(toolsRaw),
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
	require.Len(t, steps, 1, "Should produce exactly one execution step")

	stepText := strings.Join(steps[0], "\n")
	assert.Contains(t, stepText, "PI_CODING_AGENT_DIR", "Firewall mode should set PI_CODING_AGENT_DIR for models.json config")
	assert.Contains(t, stepText, "aw-gateway", "Firewall mode should register the aw-gateway provider in models.json")
	assert.Contains(t, stepText, "claude-opus-4-20251101", "Step should include the model ID in models.json")
	assert.Contains(t, stepText, `"enabled":true`, "Firewall mode should enable the api-proxy in AWF config JSON")
	// Anthropic provider routes through the Claude LLM gateway port.
	// models.json must use the "api-proxy" Docker service hostname, not host.docker.internal.
	expectedModelsJSON := buildPiModelsJSON(constants.ClaudeLLMGatewayPort, "ANTHROPIC_API_KEY", "claude-opus-4-20251101")
	assert.Contains(t, expectedModelsJSON, "api-proxy:", "models.json baseUrl must use the api-proxy Docker hostname within the AWF network")
	assert.NotContains(t, expectedModelsJSON, "host.docker.internal", "models.json baseUrl must not use host.docker.internal (not the api-proxy)")
	assert.Contains(t, stepText, expectedModelsJSON, "Anthropic provider should route through ClaudeLLMGatewayPort via models.json")
}

func TestPiEngine_GetExecutionSteps_FirewallCodexProvider(t *testing.T) {
	engine := NewPiEngine()
	toolsRaw := map[string]any{
		"github":    map[string]any{"mode": "gh-proxy"},
		"cli-proxy": true,
	}
	workflowData := &WorkflowData{
		Name:         "test-workflow",
		EngineConfig: &EngineConfig{ID: "pi", Model: "codex/gpt-4.1"},
		Tools:        toolsRaw,
		ParsedTools:  NewTools(toolsRaw),
		NetworkPermissions: &NetworkPermissions{
			Firewall: &FirewallConfig{Enabled: true},
		},
	}
	steps := engine.GetExecutionSteps(workflowData, "/tmp/gh-aw/agent-stdio.log")
	require.Len(t, steps, 1, "Should produce exactly one execution step")

	stepText := strings.Join(steps[0], "\n")
	assert.Contains(t, stepText, "PI_CODING_AGENT_DIR", "Firewall mode should set PI_CODING_AGENT_DIR for models.json config")
	assert.Contains(t, stepText, "aw-gateway", "Firewall mode should register the aw-gateway provider in models.json")
	assert.Contains(t, stepText, "gpt-4.1", "Step should include the model ID in models.json")
	assert.Contains(t, stepText, `"enabled":true`, "Firewall mode should enable the api-proxy in AWF config JSON")
	// Codex/OpenAI provider routes through the Codex LLM gateway port.
	// models.json must use the "api-proxy" Docker service hostname, not host.docker.internal.
	expectedModelsJSON := buildPiModelsJSON(constants.CodexLLMGatewayPort, "CODEX_API_KEY", "gpt-4.1")
	assert.Contains(t, expectedModelsJSON, "api-proxy:", "models.json baseUrl must use the api-proxy Docker hostname within the AWF network")
	assert.NotContains(t, expectedModelsJSON, "host.docker.internal", "models.json baseUrl must not use host.docker.internal (not the api-proxy)")
	assert.Contains(t, stepText, expectedModelsJSON, "Codex provider should route through CodexLLMGatewayPort via models.json")
}
