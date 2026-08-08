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

	t.Run("accepts local path references", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"skills/rig",
				".github/skills/my-skill",
				"./skills/my-skill",
			},
		})
		require.NoError(t, err)
	})

	t.Run("accepts object form with local path skill", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				map[string]any{
					"skill": "skills/rig",
				},
			},
		})
		require.NoError(t, err)
	})

	t.Run("rejects local path traversal segments", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"skills/../rig",
				"../skills/rig",
			},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "without '..' traversal segments")
	})

	t.Run("accepts non-sha refs (branch/tag)", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"githubnext/skills@main",
				"githubnext/skills/review/security@v1.2.3",
				"githubnext/skills@release/1.0",
			},
		})
		require.NoError(t, err)
	})

	t.Run("accepts remote spec with no ref specified", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"githubnext/skills@",
				"githubnext/skills/review/security@",
			},
		})
		require.NoError(t, err)
	})

	t.Run("rejects invalid remote spec shape", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"owner@main",
			},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "owner/repo@<ref>")
	})

	t.Run("rejects ref with unsafe characters", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				"githubnext/skills@main; rm -rf /",
			},
		})
		require.Error(t, err)
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
		require.ErrorContains(t, err, "does not support expressions")
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

	t.Run("rejects object form with steps output github-token", func(t *testing.T) {
		err := validateFrontmatterSkills(map[string]any{
			"skills": []any{
				map[string]any{
					"skill":        "githubnext/skills@1f181b37d3fe5862ab590648f25a292e345b5de6",
					"github-token": "${{ steps.fetch_token.outputs.token }}",
				},
			},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "skills[0].github-token must be a valid GitHub token expression")
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
		require.ErrorContains(t, err, "skills[0].github-token must be a valid GitHub token expression")
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
		require.ErrorContains(t, err, "skills[0].skill")
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
		require.ErrorContains(t, err, "skills[0].github-app")
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
		require.ErrorContains(t, err, "skills[0].token is not supported")
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
		require.ErrorContains(t, err, "mutually exclusive")
	})

}

func TestParseSkillRefSpec(t *testing.T) {
	const sha = "1f181b37d3fe5862ab590648f25a292e345b5de6"
	tests := []struct {
		name                               string
		spec                               string
		local, expression, remote, fullSHA bool
		repoPath, ref                      string
	}{
		{name: "local path", spec: "./skills/my-skill", local: true},
		{name: "bare expression", spec: "${{ github.sha }}", expression: true},
		{name: "expression", spec: "githubnext/skills@${{ github.sha }}", expression: true},
		{name: "malformed remote", spec: "githubnext@main"},
		{name: "unpinned remote", spec: " githubnext/skills@ ", remote: true, repoPath: "githubnext/skills"},
		{name: "SHA-pinned remote", spec: "githubnext/skills@" + sha, remote: true, fullSHA: true, repoPath: "githubnext/skills", ref: sha},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := parseSkillRefSpec(tt.spec)
			require.Equal(t, tt.local, parsed.isLocal)
			require.Equal(t, tt.expression, parsed.isExpression)
			require.Equal(t, tt.remote, parsed.isRemote)
			require.Equal(t, tt.fullSHA, parsed.isFullSHA)
			require.Equal(t, tt.repoPath, parsed.repoPath)
			require.Equal(t, tt.ref, parsed.ref)
		})
	}
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
