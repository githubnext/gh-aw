package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeImportedSandboxMounts(t *testing.T) {
	compiler := NewCompiler()

	t.Run("merges imported and main mounts with dedup", func(t *testing.T) {
		mainSandbox := &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Mounts: []string{
					"/tmp/b:/tmp/b:ro",
					"/tmp/c:/tmp/c:ro",
				},
			},
		}

		mergedSandbox := `{"agent":{"mounts":["/tmp/a:/tmp/a:ro","/tmp/b:/tmp/b:ro"]}}
{"agent":{"mounts":["/tmp/b:/tmp/b:ro"]}}`

		result := compiler.mergeImportedSandboxMounts(mainSandbox, mergedSandbox)
		require.NotNil(t, result)
		require.NotNil(t, result.Agent)
		assert.ElementsMatch(t,
			[]string{"/tmp/a:/tmp/a:ro", "/tmp/b:/tmp/b:ro", "/tmp/c:/tmp/c:ro"},
			result.Agent.Mounts,
		)
		assert.Len(t, result.Agent.Mounts, 3)
	})

	t.Run("creates sandbox config when only imports define mounts", func(t *testing.T) {
		mergedSandbox := `{"agent":{"mounts":["/tmp/a:/tmp/a:ro"]}}`
		result := compiler.mergeImportedSandboxMounts(nil, mergedSandbox)
		require.NotNil(t, result)
		require.NotNil(t, result.Agent)
		assert.Equal(t, []string{"/tmp/a:/tmp/a:ro"}, result.Agent.Mounts)
	})

	t.Run("preserves explicit sandbox disable", func(t *testing.T) {
		mainSandbox := &SandboxConfig{
			Agent: &AgentSandboxConfig{
				Disabled: true,
				Mounts:   []string{"/tmp/main:/tmp/main:ro"},
			},
		}
		mergedSandbox := `{"agent":{"mounts":["/tmp/import:/tmp/import:ro"]}}`

		result := compiler.mergeImportedSandboxMounts(mainSandbox, mergedSandbox)
		require.NotNil(t, result)
		require.NotNil(t, result.Agent)
		assert.True(t, result.Agent.Disabled)
		assert.Equal(t, []string{"/tmp/main:/tmp/main:ro"}, result.Agent.Mounts)
	})
}

