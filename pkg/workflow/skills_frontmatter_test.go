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

	t.Run("rejects github actions expressions", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"${{ inputs.skill_ref }}",
				"githubnext/skills@${{ github.sha }}",
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "40-char-sha")
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

	t.Run("accepts object form with github-token", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				map[string]any{
					"skill":        "githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
					"github-token": "${{ secrets.SOME_TOKEN }}",
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("rejects object form with github-token literal", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				map[string]any{
					"skill":        "githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
					"github-token": "ghp_literal",
				},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "skills[0].github-token must be a valid GitHub token expression")
	})

	t.Run("accepts object form with github-app", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				map[string]any{
					"skill": "githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6",
					"github-app": map[string]any{
						"client-id":   "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("rejects object form without skill", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				map[string]any{
					"github-token": "${{ secrets.SOME_TOKEN }}",
				},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "skills[0].skill")
	})

	t.Run("rejects object form github-app without private-key", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				map[string]any{
					"skill": "githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
					"github-app": map[string]any{
						"client-id": "${{ vars.APP_ID }}",
					},
				},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "skills[0].github-app")
	})

	t.Run("rejects object form with unknown fields", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				map[string]any{
					"skill":        "githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
					"github-token": "${{ secrets.SOME_TOKEN }}",
					"token":        "${{ secrets.OTHER_TOKEN }}",
				},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "skills[0].token is not supported")
	})

	t.Run("rejects object form that sets both github-token and github-app", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				map[string]any{
					"skill":        "githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
					"github-token": "${{ secrets.SOME_TOKEN }}",
					"github-app": map[string]any{
						"client-id":   "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "mutually exclusive")
	})

}

func TestIsRepositorySkillSpec(t *testing.T) {
	require.True(t, isRepositorySkillSpec("githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6"), "owner/repo@sha should be treated as a repository skill spec")
	require.False(t, isRepositorySkillSpec("githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6"), "owner/repo/skill/path@sha should be treated as a path-scoped skill spec")
}

func TestParseRawSkillReferences_ParsesGitHubApp(t *testing.T) {
	refs := parseRawSkillReferences([]any{
		map[string]any{
			"skill": "githubnext/skills/review/security@1f181b37d3fe5862ab590648f25a292e345b5de6",
			"github-app": map[string]any{
				"client-id":   "${{ vars.APP_ID }}",
				"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
			},
		},
	})

	require.Len(t, refs, 1)
	require.NotNil(t, refs[0].GitHubApp)
	require.Equal(t, "${{ vars.APP_ID }}", refs[0].GitHubApp.AppID)
	require.Equal(t, "${{ secrets.APP_PRIVATE_KEY }}", refs[0].GitHubApp.PrivateKey)
}
