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
		assert.Contains(t, secrets, "COPILOT_GITHUB_TOKEN", "Nil workflow data should safely fall back to copilot backend profile")
	}, "GetUniversalRequiredSecretNames should handle nil workflowData safely")
}
