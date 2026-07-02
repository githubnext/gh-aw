package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var llmProviderLog = logger.New("workflow:llm_provider")

const (
	LLMProviderGitHub    = "github"
	LLMProviderAnthropic = "anthropic"
	LLMProviderOpenAI    = "openai"
)

type llmProviderProfile struct {
	id          string
	gatewayPort int
}

func normalizeLLMProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case LLMProviderGitHub:
		return LLMProviderGitHub
	case LLMProviderOpenAI:
		return LLMProviderOpenAI
	default:
		return LLMProviderAnthropic
	}
}

func resolveEngineLLMProvider(workflowData *WorkflowData, defaultProvider string) string {
	if workflowData == nil || workflowData.EngineConfig == nil || workflowData.EngineConfig.LLMProvider == "" {
		provider := normalizeLLMProvider(defaultProvider)
		llmProviderLog.Printf("Resolved LLM provider from default: %s", provider)
		return provider
	}
	provider := normalizeLLMProvider(workflowData.EngineConfig.LLMProvider)
	llmProviderLog.Printf("Resolved LLM provider from engine config: %s", provider)
	return provider
}

func llmProviderProfileFor(provider string) llmProviderProfile {
	switch normalizeLLMProvider(provider) {
	case LLMProviderGitHub:
		return llmProviderProfile{
			id:          LLMProviderGitHub,
			gatewayPort: constants.CopilotLLMGatewayPort,
		}
	case LLMProviderOpenAI:
		return llmProviderProfile{
			id:          LLMProviderOpenAI,
			gatewayPort: constants.CodexLLMGatewayPort,
		}
	default:
		return llmProviderProfile{
			id:          LLMProviderAnthropic,
			gatewayPort: constants.ClaudeLLMGatewayPort,
		}
	}
}

func llmProviderSecretNames(provider string) []string {
	switch normalizeLLMProvider(provider) {
	case LLMProviderGitHub:
		return []string{"COPILOT_GITHUB_TOKEN"}
	case LLMProviderOpenAI:
		return []string{"CODEX_API_KEY", "OPENAI_API_KEY"}
	default:
		return []string{"ANTHROPIC_API_KEY"}
	}
}

func llmProviderSecretExpression(provider string, workflowData *WorkflowData) string {
	switch normalizeLLMProvider(provider) {
	case LLMProviderGitHub:
		if hasCopilotRequestsWritePermission(workflowData) {
			llmProviderLog.Print("Using github.token for GitHub Copilot (copilot-requests write permission present)")
			return "${{ github.token }}"
		}
		llmProviderLog.Print("Using COPILOT_GITHUB_TOKEN secret for GitHub Copilot")
		return "${{ secrets.COPILOT_GITHUB_TOKEN }}"
	case LLMProviderOpenAI:
		return "${{ secrets.CODEX_API_KEY || secrets.OPENAI_API_KEY }}"
	default:
		return "${{ secrets.ANTHROPIC_API_KEY }}"
	}
}

func llmProviderGatewayBaseURL(provider string) string {
	profile := llmProviderProfileFor(provider)
	return fmt.Sprintf("http://host.docker.internal:%d", profile.gatewayPort)
}

func llmProviderDocsURL(provider string) string {
	switch normalizeLLMProvider(provider) {
	case LLMProviderGitHub:
		return "https://github.github.com/gh-aw/reference/engines/#github-copilot-default"
	case LLMProviderOpenAI:
		return "https://github.github.com/gh-aw/reference/engines/#openai-codex"
	default:
		return "https://github.github.com/gh-aw/reference/engines/#anthropic-claude-code"
	}
}
