package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRunnerConfig(t *testing.T) {
	t.Run("nil when runner not present", func(t *testing.T) {
		fm := map[string]any{"name": "test"}
		assert.Nil(t, extractRunnerConfig(fm))
	})

	t.Run("nil when runner is not an object", func(t *testing.T) {
		fm := map[string]any{"runner": "arc-dind"}
		assert.Nil(t, extractRunnerConfig(fm))
	})

	t.Run("nil when topology is empty", func(t *testing.T) {
		fm := map[string]any{"runner": map[string]any{}}
		assert.Nil(t, extractRunnerConfig(fm))
	})

	t.Run("extracts arc-dind topology", func(t *testing.T) {
		fm := map[string]any{
			"runner": map[string]any{
				"topology": "arc-dind",
			},
		}
		cfg := extractRunnerConfig(fm)
		require.NotNil(t, cfg)
		assert.Equal(t, RunnerTopologyArcDind, cfg.Topology)
	})
}

func TestValidateRunnerConfig(t *testing.T) {
	t.Run("nil config is valid", func(t *testing.T) {
		assert.NoError(t, validateRunnerConfig(nil))
	})

	t.Run("arc-dind is valid", func(t *testing.T) {
		assert.NoError(t, validateRunnerConfig(&RunnerConfig{Topology: RunnerTopologyArcDind}))
	})

	t.Run("empty topology is valid", func(t *testing.T) {
		assert.NoError(t, validateRunnerConfig(&RunnerConfig{}))
	})

	t.Run("unsupported topology returns error", func(t *testing.T) {
		err := validateRunnerConfig(&RunnerConfig{Topology: "unknown"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported runner.topology")
		assert.Contains(t, err.Error(), "unknown")
	})
}

func TestIsArcDindTopology(t *testing.T) {
	t.Run("false when nil workflow data", func(t *testing.T) {
		assert.False(t, isArcDindTopology(nil))
	})

	t.Run("false when nil runner config", func(t *testing.T) {
		assert.False(t, isArcDindTopology(&WorkflowData{}))
	})

	t.Run("false when topology is empty", func(t *testing.T) {
		assert.False(t, isArcDindTopology(&WorkflowData{
			RunnerConfig: &RunnerConfig{},
		}))
	})

	t.Run("true when topology is arc-dind", func(t *testing.T) {
		assert.True(t, isArcDindTopology(&WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: RunnerTopologyArcDind},
		}))
	})
}

func TestGetRunnerTopology(t *testing.T) {
	t.Run("empty when nil workflow data", func(t *testing.T) {
		assert.Empty(t, getRunnerTopology(nil))
	})

	t.Run("returns topology when set", func(t *testing.T) {
		wd := &WorkflowData{
			RunnerConfig: &RunnerConfig{Topology: "arc-dind"},
		}
		assert.Equal(t, "arc-dind", getRunnerTopology(wd))
	})
}
