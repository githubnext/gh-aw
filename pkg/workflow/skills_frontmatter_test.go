//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFrontmatterSkills(t *testing.T) {
	t.Run("accepts pinned repository and path specs", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
				"githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6",
			},
		})
		require.NoError(t, err)
	})

	t.Run("rejects non-sha refs", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"githubnext/skills@main",
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "40-char-sha")
	})

	t.Run("accepts github actions expressions", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"${{ inputs.skill_ref }}",
				"githubnext/skills@${{ github.sha }}",
			},
		})
		require.NoError(t, err)
	})
}

func TestIsRepositorySkillSpec(t *testing.T) {
	require.True(t, isRepositorySkillSpec("githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6"))
	require.False(t, isRepositorySkillSpec("githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6"))
}
