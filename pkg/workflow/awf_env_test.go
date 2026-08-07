//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInjectMaxAICreditsExpressionWithoutMaxRunsLeavesJSONUnchanged(t *testing.T) {
	configJSON := `{"apiProxy":{"enabled":true}}`

	got := injectMaxAICreditsExpression(configJSON, "${GH_AW_MAX_AI_CREDITS}")

	if got != configJSON {
		t.Fatalf("expected config JSON to be unchanged, got %q", got)
	}
}

func TestApplyDefaultMaxAICreditsEnvToMapHandlesNilMap(t *testing.T) {
	assert.NotPanics(t, func() {
		applyDefaultMaxAICreditsEnvToMap(nil, nil)
	})
}
