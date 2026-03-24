//go:build !integration

package workflow

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// combineStepLines joins a GitHubActionStep slice into a single string for assertion.
func combineStepLines(step []string) string {
	var sb strings.Builder
	for _, line := range step {
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

func TestExtractAPMDependenciesFromFrontmatter(t *testing.T) {
	tests := []struct {
		name             string
		frontmatter      map[string]any
		expectedDeps     []string
		expectedIsolated bool
	}{
		{
			name: "No dependencies field",
			frontmatter: map[string]any{
				"engine": "copilot",
			},
			expectedDeps: nil,
		},
		{
			name: "Single dependency (array format)",
			frontmatter: map[string]any{
				"dependencies": []any{"microsoft/apm-sample-package"},
			},
			expectedDeps: []string{"microsoft/apm-sample-package"},
		},
		{
			name: "Multiple dependencies (array format)",
			frontmatter: map[string]any{
				"dependencies": []any{
					"microsoft/apm-sample-package",
					"github/awesome-copilot/skills/review-and-refactor",
					"anthropics/skills/skills/frontend-design",
				},
			},
			expectedDeps: []string{
				"microsoft/apm-sample-package",
				"github/awesome-copilot/skills/review-and-refactor",
				"anthropics/skills/skills/frontend-design",
			},
		},
		{
			name: "Empty array",
			frontmatter: map[string]any{
				"dependencies": []any{},
			},
			expectedDeps: nil,
		},
		{
			name: "Non-array, non-object value is ignored",
			frontmatter: map[string]any{
				"dependencies": "microsoft/apm-sample-package",
			},
			expectedDeps: nil,
		},
		{
			name: "Empty string items are skipped",
			frontmatter: map[string]any{
				"dependencies": []any{"microsoft/apm-sample-package", "", "github/awesome-copilot"},
			},
			expectedDeps: []string{"microsoft/apm-sample-package", "github/awesome-copilot"},
		},
		{
			name: "Object format with packages only",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{
						"microsoft/apm-sample-package",
						"github/awesome-copilot",
					},
				},
			},
			expectedDeps:     []string{"microsoft/apm-sample-package", "github/awesome-copilot"},
			expectedIsolated: false,
		},
		{
			name: "Object format with isolated true",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{"microsoft/apm-sample-package"},
					"isolated": true,
				},
			},
			expectedDeps:     []string{"microsoft/apm-sample-package"},
			expectedIsolated: true,
		},
		{
			name: "Object format with isolated false",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{"microsoft/apm-sample-package"},
					"isolated": false,
				},
			},
			expectedDeps:     []string{"microsoft/apm-sample-package"},
			expectedIsolated: false,
		},
		{
			name: "Object format with empty packages",
			frontmatter: map[string]any{
				"dependencies": map[string]any{
					"packages": []any{},
				},
			},
			expectedDeps: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractAPMDependenciesFromFrontmatter(tt.frontmatter)
			require.NoError(t, err, "Should not return an error for valid frontmatter")
			if tt.expectedDeps == nil {
				assert.Nil(t, result, "Should return nil for no dependencies")
			} else {
				require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
				assert.Equal(t, tt.expectedDeps, result.Packages, "Extracted packages should match expected")
				assert.Equal(t, tt.expectedIsolated, result.Isolated, "Isolated flag should match expected")
			}
		})
	}
}

func TestExtractAPMDependenciesGitHubApp(t *testing.T) {
	t.Run("Object format with github-app", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"acme-org/acme-skills/plugins/dev-tools"},
				"github-app": map[string]any{
					"app-id":      "${{ vars.APP_ID }}",
					"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
				},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Equal(t, []string{"acme-org/acme-skills/plugins/dev-tools"}, result.Packages)
		require.NotNil(t, result.GitHubApp, "GitHubApp should be set")
		assert.Equal(t, "${{ vars.APP_ID }}", result.GitHubApp.AppID)
		assert.Equal(t, "${{ secrets.APP_PRIVATE_KEY }}", result.GitHubApp.PrivateKey)
	})

	t.Run("Object format with github-app including owner and repositories", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"acme-org/acme-skills/plugins/dev-tools"},
				"github-app": map[string]any{
					"app-id":       "${{ vars.APP_ID }}",
					"private-key":  "${{ secrets.APP_PRIVATE_KEY }}",
					"owner":        "acme-org",
					"repositories": []any{"acme-skills"},
				},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		require.NotNil(t, result.GitHubApp, "GitHubApp should be set")
		assert.Equal(t, "acme-org", result.GitHubApp.Owner)
		assert.Equal(t, []string{"acme-skills"}, result.GitHubApp.Repositories)
	})

	t.Run("Object format with github-app missing app-id is ignored", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"acme-org/acme-skills"},
				"github-app": map[string]any{
					"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
				},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Packages should still be extracted")
		assert.Nil(t, result.GitHubApp, "GitHubApp should be nil when app-id is missing")
	})

	t.Run("Array format does not support github-app", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": []any{"microsoft/apm-sample-package"},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Nil(t, result.GitHubApp, "GitHubApp should be nil for array format")
	})
}

func TestExtractAPMDependenciesVersion(t *testing.T) {
	t.Run("Object format with version field", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
				"version":  "v1.0.0",
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error for valid version")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Equal(t, "v1.0.0", result.Version, "Version should be extracted from object format")
	})

	t.Run("Array format has no version field", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": []any{"microsoft/apm-sample-package"},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Empty(t, result.Version, "Version should be empty for array format")
	})

	t.Run("Object format without version uses empty string", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Empty(t, result.Version, "Version should be empty when not specified")
	})

	t.Run("Invalid version with trailing quote produces error", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
				"version":  `v0.8.0"`,
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.Error(t, err, "Should return an error for invalid version tag")
		assert.Nil(t, result, "Should return nil result on error")
		assert.Contains(t, err.Error(), "dependencies.version", "Error should mention the field name")
	})

	t.Run("Invalid version without v prefix produces error", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
				"version":  "1.2.3",
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.Error(t, err, "Should return an error for version missing v prefix")
		assert.Nil(t, result, "Should return nil result on error")
		assert.Contains(t, err.Error(), "vX.Y.Z", "Error should describe expected format")
	})

	t.Run("Invalid version string 'latest' produces error", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
				"version":  "latest",
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.Error(t, err, "Should return an error for non-semver version string")
		assert.Nil(t, result, "Should return nil result on error")
	})

	t.Run("Valid partial version v1 compiles without error", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
				"version":  "v1",
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error for valid partial version")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Equal(t, "v1", result.Version, "Version should be extracted")
	})

	t.Run("Valid partial version v1.2 compiles without error", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
				"version":  "v1.2",
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error for valid partial version")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Equal(t, "v1.2", result.Version, "Version should be extracted")
	})
}

func TestEngineGetAPMTarget(t *testing.T) {
	tests := []struct {
		name     string
		engine   CodingAgentEngine
		expected string
	}{
		{name: "copilot engine returns copilot", engine: NewCopilotEngine(), expected: "copilot"},
		{name: "claude engine returns claude", engine: NewClaudeEngine(), expected: "claude"},
		{name: "codex engine returns all", engine: NewCodexEngine(), expected: "all"},
		{name: "gemini engine returns all", engine: NewGeminiEngine(), expected: "all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.engine.GetAPMTarget()
			assert.Equal(t, tt.expected, result, "APM target should match for engine %s", tt.engine.GetID())
		})
	}
}

func TestGenerateAPMPackStep(t *testing.T) {
	tests := []struct {
		name             string
		apmDeps          *APMDependenciesInfo
		target           string
		expectedContains []string
		expectedEmpty    bool
	}{
		{
			name:          "Nil deps returns empty step",
			apmDeps:       nil,
			target:        "copilot",
			expectedEmpty: true,
		},
		{
			name:          "Empty packages returns empty step",
			apmDeps:       &APMDependenciesInfo{Packages: []string{}},
			target:        "copilot",
			expectedEmpty: true,
		},
		{
			name:    "Single dependency with copilot target",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}},
			target:  "copilot",
			expectedContains: []string{
				"Install and pack APM dependencies",
				"id: apm_pack",
				"microsoft/apm-action",
				"dependencies: |",
				"- microsoft/apm-sample-package",
				"isolated: 'true'",
				"pack: 'true'",
				"archive: 'true'",
				"target: copilot",
				"working-directory: /tmp/gh-aw/apm-workspace",
				"apm-version: ${{ env.GH_AW_INFO_APM_VERSION }}",
			},
		},
		{
			name:    "Multiple dependencies with claude target",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package", "github/skills/review"}},
			target:  "claude",
			expectedContains: []string{
				"Install and pack APM dependencies",
				"id: apm_pack",
				"microsoft/apm-action",
				"- microsoft/apm-sample-package",
				"- github/skills/review",
				"target: claude",
				"apm-version: ${{ env.GH_AW_INFO_APM_VERSION }}",
			},
		},
		{
			name:    "All target for non-copilot/claude engine",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}},
			target:  "all",
			expectedContains: []string{
				"target: all",
				"apm-version: ${{ env.GH_AW_INFO_APM_VERSION }}",
			},
		},
		{
			name:    "Custom APM version still uses env var reference in step",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Version: "v1.0.0"},
			target:  "copilot",
			expectedContains: []string{
				"apm-version: ${{ env.GH_AW_INFO_APM_VERSION }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{Name: "test-workflow"}
			step := GenerateAPMPackStep(tt.apmDeps, tt.target, data)

			if tt.expectedEmpty {
				assert.Empty(t, step, "Step should be empty for empty/nil dependencies")
				return
			}

			require.NotEmpty(t, step, "Step should not be empty")
			combined := combineStepLines(step)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, combined, expected, "Step should contain: %s", expected)
			}
		})
	}
}

func TestGenerateAPMRestoreStep(t *testing.T) {
	tests := []struct {
		name                string
		apmDeps             *APMDependenciesInfo
		expectedContains    []string
		expectedNotContains []string
		expectedEmpty       bool
	}{
		{
			name:          "Nil deps returns empty step",
			apmDeps:       nil,
			expectedEmpty: true,
		},
		{
			name:          "Empty packages returns empty step",
			apmDeps:       &APMDependenciesInfo{Packages: []string{}},
			expectedEmpty: true,
		},
		{
			name:    "Non-isolated restore step",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Isolated: false},
			expectedContains: []string{
				"Restore APM dependencies",
				"microsoft/apm-action",
				"bundle: /tmp/gh-aw/apm-bundle/*.tar.gz",
				"apm-version: ${{ env.GH_AW_INFO_APM_VERSION }}",
			},
			expectedNotContains: []string{"isolated"},
		},
		{
			name:    "Isolated restore step",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Isolated: true},
			expectedContains: []string{
				"Restore APM dependencies",
				"microsoft/apm-action",
				"bundle: /tmp/gh-aw/apm-bundle/*.tar.gz",
				"isolated: 'true'",
				"apm-version: ${{ env.GH_AW_INFO_APM_VERSION }}",
			},
		},
		{
			name:    "Custom APM version still uses env var reference in step",
			apmDeps: &APMDependenciesInfo{Packages: []string{"microsoft/apm-sample-package"}, Version: "v1.0.0"},
			expectedContains: []string{
				"apm-version: ${{ env.GH_AW_INFO_APM_VERSION }}",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := &WorkflowData{Name: "test-workflow"}
			step := GenerateAPMRestoreStep(tt.apmDeps, data)

			if tt.expectedEmpty {
				assert.Empty(t, step, "Step should be empty for empty/nil dependencies")
				return
			}

			require.NotEmpty(t, step, "Step should not be empty")
			combined := combineStepLines(step)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, combined, expected, "Step should contain: %s", expected)
			}
			for _, notExpected := range tt.expectedNotContains {
				assert.NotContains(t, combined, notExpected, "Step should not contain: %s", notExpected)
			}
		})
	}
}

func TestGenerateAPMPackStepWithGitHubApp(t *testing.T) {
	t.Run("Pack step includes GITHUB_TOKEN env when github-app is configured", func(t *testing.T) {
		apmDeps := &APMDependenciesInfo{
			Packages: []string{"acme-org/acme-skills/plugins/dev-tools"},
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}
		data := &WorkflowData{Name: "test-workflow"}
		step := GenerateAPMPackStep(apmDeps, "claude", data)

		require.NotEmpty(t, step, "Step should not be empty")
		combined := combineStepLines(step)

		assert.Contains(t, combined, "GITHUB_TOKEN: ${{ steps.apm-app-token.outputs.token }}", "Should inject app token as GITHUB_TOKEN")
		assert.Contains(t, combined, "env:", "Should have env section")
		assert.Contains(t, combined, "- acme-org/acme-skills/plugins/dev-tools", "Should list dependency")
	})

	t.Run("Pack step uses cascading fallback GITHUB_TOKEN without github-app", func(t *testing.T) {
		apmDeps := &APMDependenciesInfo{
			Packages: []string{"microsoft/apm-sample-package"},
		}
		data := &WorkflowData{Name: "test-workflow"}
		step := GenerateAPMPackStep(apmDeps, "copilot", data)

		require.NotEmpty(t, step, "Step should not be empty")
		combined := combineStepLines(step)

		assert.Contains(t, combined, "GITHUB_TOKEN:", "Should have GITHUB_TOKEN with cascading fallback")
		assert.Contains(t, combined, "GH_AW_PLUGINS_TOKEN", "Should reference cascading token")
		assert.Contains(t, combined, "GH_AW_GITHUB_TOKEN", "Should reference cascading token")
		assert.NotContains(t, combined, "apm-app-token", "Should not reference app token without github-app")
	})
}

func TestBuildAPMAppTokenMintStep(t *testing.T) {
	t.Run("Basic app token mint step", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:      "${{ vars.APP_ID }}",
			PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
		}
		steps := buildAPMAppTokenMintStep(app, "")

		combined := strings.Join(steps, "")
		assert.Contains(t, combined, "Generate GitHub App token for APM dependencies", "Should have descriptive step name")
		assert.Contains(t, combined, "id: apm-app-token", "Should use apm-app-token step ID")
		assert.Contains(t, combined, "actions/create-github-app-token", "Should use create-github-app-token action")
		assert.Contains(t, combined, "app-id: ${{ vars.APP_ID }}", "Should include app-id")
		assert.Contains(t, combined, "private-key: ${{ secrets.APP_PRIVATE_KEY }}", "Should include private-key")
		assert.Contains(t, combined, "github-api-url: ${{ github.api_url }}", "Should include github-api-url")
	})

	t.Run("App token mint step with explicit owner", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:      "${{ vars.APP_ID }}",
			PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			Owner:      "acme-org",
		}
		steps := buildAPMAppTokenMintStep(app, "")

		combined := strings.Join(steps, "")
		assert.Contains(t, combined, "owner: acme-org", "Should include explicit owner")
	})

	t.Run("App token mint step defaults to github.repository_owner when owner not set", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:      "${{ vars.APP_ID }}",
			PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
		}
		steps := buildAPMAppTokenMintStep(app, "")

		combined := strings.Join(steps, "")
		assert.Contains(t, combined, "owner: ${{ github.repository_owner }}", "Should default owner to github.repository_owner")
	})

	t.Run("App token mint step with wildcard repositories omits repositories field", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:        "${{ vars.APP_ID }}",
			PrivateKey:   "${{ secrets.APP_PRIVATE_KEY }}",
			Repositories: []string{"*"},
		}
		steps := buildAPMAppTokenMintStep(app, "")

		combined := strings.Join(steps, "")
		assert.NotContains(t, combined, "repositories:", "Should omit repositories field for org-wide access")
	})

	t.Run("App token mint step with explicit repositories", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:        "${{ vars.APP_ID }}",
			PrivateKey:   "${{ secrets.APP_PRIVATE_KEY }}",
			Repositories: []string{"acme-skills"},
		}
		steps := buildAPMAppTokenMintStep(app, "")

		combined := strings.Join(steps, "")
		assert.Contains(t, combined, "repositories: acme-skills", "Should include explicit repository")
	})

	t.Run("App token mint step uses fallbackRepoExpr when no repositories configured", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:      "${{ vars.APP_ID }}",
			PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
		}
		steps := buildAPMAppTokenMintStep(app, "${{ steps.resolve-host-repo.outputs.target_repo_name }}")

		combined := strings.Join(steps, "")
		assert.Contains(t, combined, "repositories: ${{ steps.resolve-host-repo.outputs.target_repo_name }}", "Should use fallback repo expr for workflow_call relay")
		assert.NotContains(t, combined, "github.event.repository.name", "Should not fall back to event repository")
	})

	t.Run("App token mint step defaults to github.event.repository.name when no fallback and no repositories", func(t *testing.T) {
		app := &GitHubAppConfig{
			AppID:      "${{ vars.APP_ID }}",
			PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
		}
		steps := buildAPMAppTokenMintStep(app, "")

		combined := strings.Join(steps, "")
		assert.Contains(t, combined, "repositories: ${{ github.event.repository.name }}", "Should default to event repository name")
	})
}

func TestBuildAPMAppTokenInvalidationStep(t *testing.T) {
	t.Run("Invalidation step targets apm-app-token step ID", func(t *testing.T) {
		steps := buildAPMAppTokenInvalidationStep()

		combined := strings.Join(steps, "")
		assert.Contains(t, combined, "Invalidate GitHub App token for APM", "Should have descriptive step name")
		assert.Contains(t, combined, fmt.Sprintf("if: always() && steps.%s.outputs.token != ''", apmAppTokenStepID), "Should run always and check token exists")
		assert.Contains(t, combined, fmt.Sprintf("TOKEN: ${{ steps.%s.outputs.token }}", apmAppTokenStepID), "Should reference apm-app-token step output")
		assert.Contains(t, combined, "gh api", "Should call GitHub API to revoke token")
		assert.Contains(t, combined, "--method DELETE", "Should use DELETE method to revoke token")
		assert.Contains(t, combined, "/installation/token", "Should target installation token endpoint")
	})

	t.Run("Invalidation step uses always() condition for cleanup even on failure", func(t *testing.T) {
		steps := buildAPMAppTokenInvalidationStep()

		combined := strings.Join(steps, "")
		assert.Contains(t, combined, "always()", "Must run even if prior steps fail to ensure token cleanup")
	})
}

func TestExtractAPMDependenciesEnv(t *testing.T) {
	t.Run("Object format with env field extracts env vars", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
				"env": map[string]any{
					"MY_TOKEN": "${{ secrets.MY_TOKEN }}",
					"REGISTRY": "https://registry.example.com",
				},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		require.NotNil(t, result.Env, "Env map should be set")
		assert.Equal(t, "${{ secrets.MY_TOKEN }}", result.Env["MY_TOKEN"], "MY_TOKEN should be extracted")
		assert.Equal(t, "https://registry.example.com", result.Env["REGISTRY"], "REGISTRY should be extracted")
	})

	t.Run("Object format without env field has nil Env", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Nil(t, result.Env, "Env should be nil when not specified")
	})

	t.Run("Array format has nil Env", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": []any{"microsoft/apm-sample-package"},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Nil(t, result.Env, "Env should be nil for array format")
	})

	t.Run("Non-string env values are skipped", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
				"env": map[string]any{
					"VALID":   "value",
					"INVALID": 42,
				},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		require.NotNil(t, result.Env, "Env map should be set")
		assert.Equal(t, "value", result.Env["VALID"], "String value should be extracted")
		assert.NotContains(t, result.Env, "INVALID", "Non-string value should be skipped")
	})
}

func TestGenerateAPMPackStepWithEnv(t *testing.T) {
	t.Run("Pack step includes user env vars and cascading GITHUB_TOKEN", func(t *testing.T) {
		apmDeps := &APMDependenciesInfo{
			Packages: []string{"microsoft/apm-sample-package"},
			Env: map[string]string{
				"MY_TOKEN": "${{ secrets.MY_TOKEN }}",
				"REGISTRY": "https://registry.example.com",
			},
		}
		data := &WorkflowData{Name: "test-workflow"}
		step := GenerateAPMPackStep(apmDeps, "copilot", data)

		require.NotEmpty(t, step, "Step should not be empty")
		combined := combineStepLines(step)

		assert.Contains(t, combined, "env:", "Should have env section")
		assert.Contains(t, combined, "MY_TOKEN: ${{ secrets.MY_TOKEN }}", "Should include MY_TOKEN env var")
		assert.Contains(t, combined, "REGISTRY: https://registry.example.com", "Should include REGISTRY env var")
		assert.Contains(t, combined, "GITHUB_TOKEN:", "Should have GITHUB_TOKEN with cascading fallback")
		assert.Contains(t, combined, "GH_AW_PLUGINS_TOKEN", "Cascading fallback should include GH_AW_PLUGINS_TOKEN")
	})

	t.Run("Pack step with env vars and github-app includes both", func(t *testing.T) {
		apmDeps := &APMDependenciesInfo{
			Packages: []string{"acme-org/acme-skills"},
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
			Env: map[string]string{
				"EXTRA": "value",
			},
		}
		data := &WorkflowData{Name: "test-workflow"}
		step := GenerateAPMPackStep(apmDeps, "copilot", data)

		require.NotEmpty(t, step, "Step should not be empty")
		combined := combineStepLines(step)

		assert.Contains(t, combined, "GITHUB_TOKEN: ${{ steps.apm-app-token.outputs.token }}", "Should have GITHUB_TOKEN from app")
		assert.Contains(t, combined, "EXTRA: value", "Should include user env var")
	})

	t.Run("Env vars are output in sorted order for determinism", func(t *testing.T) {
		apmDeps := &APMDependenciesInfo{
			Packages: []string{"microsoft/apm-sample-package"},
			Env: map[string]string{
				"Z_VAR": "z",
				"A_VAR": "a",
				"M_VAR": "m",
			},
		}
		data := &WorkflowData{Name: "test-workflow"}
		step := GenerateAPMPackStep(apmDeps, "copilot", data)

		require.NotEmpty(t, step, "Step should not be empty")
		combined := combineStepLines(step)

		aPos := strings.Index(combined, "A_VAR:")
		mPos := strings.Index(combined, "M_VAR:")
		zPos := strings.Index(combined, "Z_VAR:")
		require.NotEqual(t, -1, aPos, "A_VAR should be present in output")
		require.NotEqual(t, -1, mPos, "M_VAR should be present in output")
		require.NotEqual(t, -1, zPos, "Z_VAR should be present in output")
		assert.True(t, aPos < mPos && mPos < zPos, "Env vars should be sorted alphabetically")
	})

	t.Run("GITHUB_TOKEN in user env is skipped when github-app is configured", func(t *testing.T) {
		apmDeps := &APMDependenciesInfo{
			Packages: []string{"acme-org/acme-skills"},
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
			Env: map[string]string{
				"GITHUB_TOKEN": "should-be-skipped",
				"OTHER_VAR":    "kept",
			},
		}
		data := &WorkflowData{Name: "test-workflow"}
		step := GenerateAPMPackStep(apmDeps, "copilot", data)

		require.NotEmpty(t, step, "Step should not be empty")
		combined := combineStepLines(step)

		assert.Contains(t, combined, "GITHUB_TOKEN: ${{ steps.apm-app-token.outputs.token }}", "Should have GITHUB_TOKEN from app token, not user env")
		assert.NotContains(t, combined, "should-be-skipped", "User-supplied GITHUB_TOKEN value should be absent")
		assert.Contains(t, combined, "OTHER_VAR: kept", "Other user env vars should be present")
		count := strings.Count(combined, "GITHUB_TOKEN:")
		assert.Equal(t, 1, count, "GITHUB_TOKEN should appear exactly once")
	})
}

func TestExtractAPMDependenciesGitHubToken(t *testing.T) {
	t.Run("Object format with github-token extracts token", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages":     []any{"microsoft/apm-sample-package"},
				"github-token": "${{ secrets.MY_TOKEN }}",
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Equal(t, "${{ secrets.MY_TOKEN }}", result.GitHubToken, "Should extract github-token")
	})

	t.Run("Object format without github-token has empty GitHubToken", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": map[string]any{
				"packages": []any{"microsoft/apm-sample-package"},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Empty(t, result.GitHubToken, "GitHubToken should be empty when not specified")
	})

	t.Run("Array format has empty GitHubToken", func(t *testing.T) {
		frontmatter := map[string]any{
			"dependencies": []any{"microsoft/apm-sample-package"},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Empty(t, result.GitHubToken, "GitHubToken should be empty for array format")
	})
}

func TestGetEffectiveAPMGitHubToken(t *testing.T) {
	t.Run("Custom token is used as-is", func(t *testing.T) {
		result := getEffectiveAPMGitHubToken("${{ secrets.MY_CUSTOM_TOKEN }}")
		assert.Equal(t, "${{ secrets.MY_CUSTOM_TOKEN }}", result, "Custom token should be returned unchanged")
	})

	t.Run("Empty token returns cascading fallback", func(t *testing.T) {
		result := getEffectiveAPMGitHubToken("")
		assert.Contains(t, result, "GH_AW_PLUGINS_TOKEN", "Fallback should include GH_AW_PLUGINS_TOKEN")
		assert.Contains(t, result, "GH_AW_GITHUB_TOKEN", "Fallback should include GH_AW_GITHUB_TOKEN")
		assert.Contains(t, result, "GITHUB_TOKEN", "Fallback should include GITHUB_TOKEN")
	})
}

func TestGenerateAPMPackStepWithGitHubToken(t *testing.T) {
	t.Run("Pack step uses custom github-token when specified", func(t *testing.T) {
		apmDeps := &APMDependenciesInfo{
			Packages:    []string{"microsoft/apm-sample-package"},
			GitHubToken: "${{ secrets.MY_TOKEN }}",
		}
		data := &WorkflowData{Name: "test-workflow"}
		step := GenerateAPMPackStep(apmDeps, "copilot", data)

		require.NotEmpty(t, step, "Step should not be empty")
		combined := combineStepLines(step)

		assert.Contains(t, combined, "GITHUB_TOKEN: ${{ secrets.MY_TOKEN }}", "Should use custom token directly")
		assert.NotContains(t, combined, "apm-app-token", "Should not reference app token")
	})

	t.Run("Pack step uses cascading fallback when no github-token specified", func(t *testing.T) {
		apmDeps := &APMDependenciesInfo{
			Packages: []string{"microsoft/apm-sample-package"},
		}
		data := &WorkflowData{Name: "test-workflow"}
		step := GenerateAPMPackStep(apmDeps, "copilot", data)

		require.NotEmpty(t, step, "Step should not be empty")
		combined := combineStepLines(step)

		assert.Contains(t, combined, "GITHUB_TOKEN:", "Should have GITHUB_TOKEN")
		assert.Contains(t, combined, "GH_AW_PLUGINS_TOKEN", "Should include GH_AW_PLUGINS_TOKEN in cascade")
	})

	t.Run("github-app takes priority over github-token", func(t *testing.T) {
		apmDeps := &APMDependenciesInfo{
			Packages:    []string{"microsoft/apm-sample-package"},
			GitHubToken: "${{ secrets.MY_TOKEN }}",
			GitHubApp: &GitHubAppConfig{
				AppID:      "${{ vars.APP_ID }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}
		data := &WorkflowData{Name: "test-workflow"}
		step := GenerateAPMPackStep(apmDeps, "copilot", data)

		require.NotEmpty(t, step, "Step should not be empty")
		combined := combineStepLines(step)

		assert.Contains(t, combined, "GITHUB_TOKEN: ${{ steps.apm-app-token.outputs.token }}", "github-app token should take priority")
		assert.NotContains(t, combined, "secrets.MY_TOKEN", "Custom github-token should not appear when github-app is configured")
	})
}

func TestExtractAPMDependenciesFromImportsAPMPackages(t *testing.T) {
	t.Run("Simple array form under imports.apm-packages", func(t *testing.T) {
		frontmatter := map[string]any{
			"imports": map[string]any{
				"apm-packages": []any{
					"microsoft/apm-sample-package",
					"github/awesome-copilot/skills/review-and-refactor",
				},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Equal(t, []string{
			"microsoft/apm-sample-package",
			"github/awesome-copilot/skills/review-and-refactor",
		}, result.Packages, "Should extract packages from imports.apm-packages")
	})

	t.Run("Object form under imports.apm-packages with isolated", func(t *testing.T) {
		frontmatter := map[string]any{
			"imports": map[string]any{
				"apm-packages": map[string]any{
					"packages": []any{"microsoft/apm-sample-package"},
					"isolated": true,
				},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Equal(t, []string{"microsoft/apm-sample-package"}, result.Packages, "Should extract packages")
		assert.True(t, result.Isolated, "Should extract isolated flag")
	})

	t.Run("Object form under imports.apm-packages with github-app", func(t *testing.T) {
		frontmatter := map[string]any{
			"imports": map[string]any{
				"apm-packages": map[string]any{
					"packages": []any{"acme-org/acme-skills/plugins/dev-tools"},
					"github-app": map[string]any{
						"app-id":      "${{ vars.APP_ID }}",
						"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
					},
				},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Equal(t, []string{"acme-org/acme-skills/plugins/dev-tools"}, result.Packages)
		require.NotNil(t, result.GitHubApp, "GitHubApp should be set")
		assert.Equal(t, "${{ vars.APP_ID }}", result.GitHubApp.AppID)
		assert.Equal(t, "${{ secrets.APP_PRIVATE_KEY }}", result.GitHubApp.PrivateKey)
	})

	t.Run("mixing imports.apm-packages and top-level dependencies is an error", func(t *testing.T) {
		frontmatter := map[string]any{
			"imports": map[string]any{
				"apm-packages": []any{"microsoft/from-imports"},
			},
			"dependencies": []any{"microsoft/from-dependencies"},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.Error(t, err, "Should return an error when both imports.apm-packages and dependencies are present")
		assert.Nil(t, result, "Should return nil result on error")
		assert.Contains(t, err.Error(), "imports.apm-packages", "Error should mention imports.apm-packages")
		assert.Contains(t, err.Error(), "dependencies", "Error should mention dependencies")
	})

	t.Run("imports.apm-packages and aw subfield coexist", func(t *testing.T) {
		frontmatter := map[string]any{
			"imports": map[string]any{
				"aw":           []any{"shared/common-tools.md"},
				"apm-packages": []any{"microsoft/apm-sample-package"},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		require.NotNil(t, result, "Should return non-nil APMDependenciesInfo")
		assert.Equal(t, []string{"microsoft/apm-sample-package"}, result.Packages)
	})

	t.Run("No imports.apm-packages and no dependencies returns nil", func(t *testing.T) {
		frontmatter := map[string]any{
			"imports": map[string]any{
				"aw": []any{"shared/common-tools.md"},
			},
		}
		result, err := extractAPMDependenciesFromFrontmatter(frontmatter)
		require.NoError(t, err, "Should not return an error")
		assert.Nil(t, result, "Should return nil when neither imports.apm-packages nor dependencies is present")
	})
}
