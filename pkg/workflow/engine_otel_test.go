//go:build !integration

package workflow

import (
	"strconv"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngineOTELCapabilities(t *testing.T) {
	assert.True(t, NewCopilotEngine().GetCapabilities().OTEL)
	assert.True(t, NewClaudeEngine().GetCapabilities().OTEL)
	assert.False(t, NewCodexEngine().GetCapabilities().OTEL)
	assert.False(t, NewGeminiEngine().GetCapabilities().OTEL)
	assert.False(t, NewPiEngine().GetCapabilities().OTEL)
	assert.False(t, (EngineCapabilitiesDefinition{}).ToRuntimeCapabilities().OTEL)

	for _, enabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			var capabilities EngineCapabilitiesDefinition
			require.NoError(t, yaml.Unmarshal([]byte("otel: "+strconv.FormatBool(enabled)), &capabilities))
			assert.Equal(t, enabled, capabilities.ToRuntimeCapabilities().OTEL)
			def := newHarnessEngineDefinition()
			def.Behaviors.Capabilities = capabilities
			engine, err := NewBehaviorDefinedEngine(def)
			require.NoError(t, err)
			assert.Equal(t, enabled, engine.GetCapabilities().OTEL)
		})
	}
}
