package workflow

import (
	"errors"
	"fmt"
	"maps"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var universalLLMConsumerLog = logger.New("workflow:universal_llm_consumer_engine")

type UniversalLLMBackend string

const (
	UniversalLLMBackendCopilot   UniversalLLMBackend = "copilot"
	UniversalLLMBackendAnthropic UniversalLLMBackend = "anthropic"
	UniversalLLMBackendCodex     UniversalLLMBackend = "codex"
)

type UniversalLLMConsumerEngine struct {
	BaseEngine
}

type universalLLMBackendProfile struct {
	coreSecretNames []string
	env             map[string]string
	baseURLEnvName  string
	gatewayPort     int
}

func resolveUniversalLLMBackendFromModel(model string) (UniversalLLMBackend, error) {
	universalLLMConsumerLog.Printf("Resolving LLM backend from model: %q", model)
	model = strings.TrimSpace(model)
	if model == "" {
		return "", errors.New("for universal consumer engines (OpenCode/Crush), engine.model is required and must use provider/model format (supported providers: copilot, anthropic, openai, codex)")
	}

	parts := strings.SplitN(model, "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", errors.New("for universal consumer engines (OpenCode/Crush), engine.model must use provider/model format (for example: copilot/gpt-5, anthropic/claude-sonnet-4, openai/gpt-4.1)")
	}

	switch strings.ToLower(strings.TrimSpace(parts[0])) {
	case "copilot":
		universalLLMConsumerLog.Printf("Resolved backend: copilot (model=%s)", parts[1])
		return UniversalLLMBackendCopilot, nil
	case "anthropic":
		universalLLMConsumerLog.Printf("Resolved backend: anthropic (model=%s)", parts[1])
		return UniversalLLMBackendAnthropic, nil
	case "openai", "codex":
		universalLLMConsumerLog.Printf("Resolved backend: codex/openai (model=%s)", parts[1])
		return UniversalLLMBackendCodex, nil
	default:
		return "", fmt.Errorf("unsupported provider %q in engine.model; supported providers: copilot, anthropic, openai, codex", parts[0])
	}
}

func getUniversalLLMBackendProfile(backend UniversalLLMBackend, useCopilotRequests bool) universalLLMBackendProfile {
	switch backend {
	case UniversalLLMBackendAnthropic:
		return universalLLMBackendProfile{
			coreSecretNames: []string{"ANTHROPIC_API_KEY"},
			env: map[string]string{
				"ANTHROPIC_API_KEY": "${{ secrets.ANTHROPIC_API_KEY }}",
			},
			baseURLEnvName: "ANTHROPIC_BASE_URL",
			gatewayPort:    constants.ClaudeLLMGatewayPort,
		}
	case UniversalLLMBackendCodex:
		return universalLLMBackendProfile{
			coreSecretNames: []string{"CODEX_API_KEY", "OPENAI_API_KEY"},
			env: map[string]string{
				"CODEX_API_KEY":  "${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}",
				"OPENAI_API_KEY": "${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}",
			},
			baseURLEnvName: "OPENAI_BASE_URL",
			gatewayPort:    constants.CodexLLMGatewayPort,
		}
	default:
		copilotToken := "${{ secrets.COPILOT_GITHUB_TOKEN }}"
		coreSecrets := []string{"COPILOT_GITHUB_TOKEN"}
		if useCopilotRequests {
			copilotToken = "${{ github.token }}"
			coreSecrets = []string{}
		}
		return universalLLMBackendProfile{
			coreSecretNames: coreSecrets,
			env: map[string]string{
				"COPILOT_GITHUB_TOKEN": copilotToken,
				"OPENAI_API_KEY":       copilotToken,
			},
			baseURLEnvName: "GITHUB_COPILOT_BASE_URL",
			gatewayPort:    constants.CopilotLLMGatewayPort,
		}
	}
}

func (e *UniversalLLMConsumerEngine) resolveBackend(workflowData *WorkflowData) UniversalLLMBackend {
	model := ""
	if workflowData != nil && workflowData.EngineConfig != nil {
		model = workflowData.EngineConfig.Model
	}
	backend, err := resolveUniversalLLMBackendFromModel(model)
	if err != nil {
		universalLLMConsumerLog.Printf("Falling back to copilot backend while resolving model %q: %v", model, err)
		return UniversalLLMBackendCopilot
	}
	return backend
}

func (e *UniversalLLMConsumerEngine) GetUniversalRequiredSecretNames(workflowData *WorkflowData) []string {
	backend := e.resolveBackend(workflowData)
	universalLLMConsumerLog.Printf("Collecting required secret names for backend: %s", backend)
	profile := getUniversalLLMBackendProfile(backend, isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData))
	secrets := append([]string{}, profile.coreSecretNames...)

	if workflowData != nil && workflowData.EngineConfig != nil && len(workflowData.EngineConfig.Env) > 0 {
		for key := range workflowData.EngineConfig.Env {
			if strings.HasSuffix(key, "_API_KEY") || strings.HasSuffix(key, "_KEY") {
				secrets = append(secrets, key)
			}
		}
	}

	if workflowData != nil {
		secrets = append(secrets, collectCommonMCPSecrets(workflowData)...)
	}

	parsedTools, tools := extractToolsConfig(workflowData)

	if hasGitHubTool(parsedTools) {
		secrets = append(secrets, "GITHUB_MCP_SERVER_TOKEN")
	}

	headerSecrets := collectHTTPMCPHeaderSecrets(tools)
	for varName := range headerSecrets {
		secrets = append(secrets, varName)
	}

	universalLLMConsumerLog.Printf("Resolved %d required secret names for backend %s", len(secrets), backend)
	return secrets
}

func extractToolsConfig(workflowData *WorkflowData) (*ToolsConfig, map[string]any) {
	if workflowData == nil {
		return nil, map[string]any{}
	}
	if workflowData.Tools == nil {
		return workflowData.ParsedTools, map[string]any{}
	}
	return workflowData.ParsedTools, workflowData.Tools
}

func (e *UniversalLLMConsumerEngine) GetUniversalSecretValidationStep(workflowData *WorkflowData, engineName, docsURL string) GitHubActionStep {
	backend := e.resolveBackend(workflowData)
	profile := getUniversalLLMBackendProfile(backend, isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData))
	if len(profile.coreSecretNames) == 0 {
		return GitHubActionStep{}
	}
	return BuildDefaultSecretValidationStep(workflowData, profile.coreSecretNames, engineName, docsURL)
}

func (e *UniversalLLMConsumerEngine) ApplyUniversalProviderEnv(env map[string]string, workflowData *WorkflowData, firewallEnabled bool) {
	backend := e.resolveBackend(workflowData)
	universalLLMConsumerLog.Printf("Applying provider env for backend=%s, firewallEnabled=%t", backend, firewallEnabled)
	profile := getUniversalLLMBackendProfile(backend, isFeatureEnabled(constants.CopilotRequestsFeatureFlag, workflowData))
	maps.Copy(env, profile.env)
	if firewallEnabled {
		universalLLMConsumerLog.Printf("Setting %s to gateway port %d", profile.baseURLEnvName, profile.gatewayPort)
		env[profile.baseURLEnvName] = fmt.Sprintf("http://host.docker.internal:%d", profile.gatewayPort)
	}
}
