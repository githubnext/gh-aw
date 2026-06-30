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

	t.Run("rejects 39-char sha", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de",
			},
		})
		require.Error(t, err)
	})

	t.Run("rejects uppercase sha chars", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"githubnext/skills@1F181B37D3FE5862AB590648F25A292E345B5DE6",
			},
		})
		require.Error(t, err)
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

	t.Run("accepts empty skills array", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{},
		})
		require.NoError(t, err)
	})

	t.Run("rejects non-string items", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{42},
		})
		require.Error(t, err)
	})

}

func TestIsRepositorySkillSpec(t *testing.T) {
	require.True(t, isRepositorySkillSpec("githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6"), "owner/repo@sha should be treated as a repository skill spec")
	require.False(t, isRepositorySkillSpec("githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6"), "owner/repo/skill/path@sha should be treated as a path-scoped skill spec")
	require.True(t, isRepositorySkillSpec("githubnext/skills@${{ github.sha }}"), "owner/repo@expression should still be treated as a repository skill spec")
	require.False(t, isRepositorySkillSpec("${{ inputs.skill_ref }}"), "whole-expression specs should defer repository/path detection to runtime")
}
