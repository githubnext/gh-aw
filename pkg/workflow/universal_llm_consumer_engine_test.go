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
