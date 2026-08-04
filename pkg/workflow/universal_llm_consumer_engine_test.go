//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUniversalLLMConsumerEngine_GetUniversalRequiredSecretNames_NilWorkflowData(t *testing.T) {
	engine := &UniversalLLMConsumerEngine{}

	assert.NotPanics(t, func() {
		secrets := engine.GetUniversalRequiredSecretNames(nil)
		assert.ElementsMatch(t, []string{"COPILOT_GITHUB_TOKEN"}, secrets, "Nil workflow data should safely fall back to only the copilot backend secret profile")
	}, "GetUniversalRequiredSecretNames should handle nil workflowData safely")
}

func TestUniversalLLMConsumerEngine_ApplyUniversalProviderEnv_SetsProvider(t *testing.T) {
	engine := &UniversalLLMConsumerEngine{}
	tests := []struct {
		model    string
		provider string
	}{
		{model: "copilot/gpt-5", provider: "github"},
		{model: "anthropic/claude-sonnet-4.6", provider: "anthropic"},
		{model: "openai/gpt-5", provider: "openai"},
		{model: "codex/gpt-5", provider: "openai"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			env := map[string]string{}
			engine.ApplyUniversalProviderEnv(env, &WorkflowData{
				Model:        tt.model,
				EngineConfig: &EngineConfig{},
			}, true)
			assert.Equal(t, tt.provider, env["GH_AW_LLM_PROVIDER"])
		})
	}
}
