//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePiEngineRequirements(t *testing.T) {
	compiler := NewCompiler()

	t.Run("non pi engine skips validation", func(t *testing.T) {
		err := compiler.validatePiEngineRequirements(NewTools(map[string]any{}), NewCopilotEngine())
		assert.NoError(t, err)
	})

	t.Run("pi requires github gh-proxy mode", func(t *testing.T) {
		err := compiler.validatePiEngineRequirements(NewTools(map[string]any{
			"github": true,
		}), NewPiEngine())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tools.github.mode: gh-proxy")
	})

	t.Run("pi requires cli-proxy", func(t *testing.T) {
		err := compiler.validatePiEngineRequirements(NewTools(map[string]any{
			"github": map[string]any{"mode": "gh-proxy"},
		}), NewPiEngine())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tools.cli-proxy: true")
	})

	t.Run("valid pi tool config passes", func(t *testing.T) {
		err := compiler.validatePiEngineRequirements(NewTools(map[string]any{
			"github":    map[string]any{"mode": "gh-proxy"},
			"cli-proxy": true,
		}), NewPiEngine())
		assert.NoError(t, err)
	})
}
