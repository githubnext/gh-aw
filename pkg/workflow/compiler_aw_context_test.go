//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ── injectAwContextIntoOnYAML ─────────────────────────────────────────────

func TestInjectAwContextIntoOnYAML_BasicInjection(t *testing.T) {
	input := `on:
  workflow_dispatch:`
	result := injectAwContextIntoOnYAML(input)
	assert.Contains(t, result, AwContextInputName+":", "aw_context input should be injected")
	assert.Contains(t, result, "inputs:", "inputs block should be added")
	assert.Contains(t, result, "type: string", "type should be set")
}

func TestInjectAwContextIntoOnYAML_NoWorkflowDispatch(t *testing.T) {
	input := `on:
  push:
    branches: [main]`
	result := injectAwContextIntoOnYAML(input)
	assert.Equal(t, input, result, "should be unchanged when no workflow_dispatch")
}

func TestInjectAwContextIntoOnYAML_WorkflowCallUnchanged(t *testing.T) {
	// workflow_call should NOT get aw_context injected via this function
	input := `on:
  workflow_call:`
	result := injectAwContextIntoOnYAML(input)
	assert.NotContains(t, result, AwContextInputName+":",
		"workflow_call should not get aw_context injected by injectAwContextIntoOnYAML")
}
