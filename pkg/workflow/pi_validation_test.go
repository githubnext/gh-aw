//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePiEngineRequirements_NoPiEngine(t *testing.T) {
	c := NewCompiler()
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{ID: "copilot"},
	}
	err := c.validatePiEngineRequirements(workflowData)
	assert.NoError(t, err, "Non-Pi engine should not trigger Pi validation")
}

func TestValidatePiEngineRequirements_NilEngineConfig(t *testing.T) {
	c := NewCompiler()
	workflowData := &WorkflowData{AI: "copilot"}
	err := c.validatePiEngineRequirements(workflowData)
	assert.NoError(t, err, "Nil EngineConfig should not trigger Pi validation")
}

func TestValidatePiEngineRequirements_BothMissing(t *testing.T) {
	c := NewCompiler()
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{ID: "pi"},
		Tools:        map[string]any{},
		ParsedTools:  NewTools(map[string]any{}),
	}
	err := c.validatePiEngineRequirements(workflowData)
	require.Error(t, err, "Pi engine without gh-proxy and cli-proxy should fail")
	assert.Contains(t, err.Error(), "gh-proxy")
	assert.Contains(t, err.Error(), "CLI proxy")
}

func TestValidatePiEngineRequirements_OnlyGhProxy(t *testing.T) {
	c := NewCompiler()
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{ID: "pi"},
		Tools: map[string]any{
			"github": map[string]any{"mode": "gh-proxy"},
		},
		ParsedTools: NewTools(map[string]any{
			"github":    map[string]any{"mode": "gh-proxy"},
			"cli-proxy": false,
		}),
	}
	err := c.validatePiEngineRequirements(workflowData)
	require.Error(t, err, "Pi engine without cli-proxy should fail")
	assert.Contains(t, err.Error(), "CLI proxy")
	assert.NotContains(t, err.Error(), "gh-proxy required")
}

func TestValidatePiEngineRequirements_OnlyCliProxy(t *testing.T) {
	c := NewCompiler()
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{ID: "pi"},
		Tools:        map[string]any{},
		ParsedTools:  NewTools(map[string]any{"cli-proxy": true}),
	}
	err := c.validatePiEngineRequirements(workflowData)
	require.Error(t, err, "Pi engine without gh-proxy should fail")
	assert.Contains(t, err.Error(), "gh-proxy")
}

func TestValidatePiEngineRequirements_BothEnabled(t *testing.T) {
	c := NewCompiler()
	toolsRaw := map[string]any{
		"github":    map[string]any{"mode": "gh-proxy"},
		"cli-proxy": true,
	}
	workflowData := &WorkflowData{
		EngineConfig: &EngineConfig{ID: "pi"},
		Tools:        toolsRaw,
		ParsedTools:  NewTools(toolsRaw),
	}
	err := c.validatePiEngineRequirements(workflowData)
	assert.NoError(t, err, "Pi engine with both gh-proxy and cli-proxy should be valid")
}
