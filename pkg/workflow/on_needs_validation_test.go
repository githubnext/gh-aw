//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOnNeedsTargets(t *testing.T) {
	t.Run("valid on.needs target", func(t *testing.T) {
		data := &WorkflowData{
			Jobs: map[string]any{
				"secrets_fetcher": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
			OnNeeds: []string{"secrets_fetcher"},
		}

		require.NoError(t, validateOnNeedsTargets(data), "expected on.needs validation to pass")
	})

	t.Run("built-in target rejected", func(t *testing.T) {
		data := &WorkflowData{
			Jobs:    map[string]any{"secrets_fetcher": map[string]any{}},
			OnNeeds: []string{"activation"},
		}

		err := validateOnNeedsTargets(data)
		require.Error(t, err, "expected on.needs validation error")
		assert.Contains(t, err.Error(), `built-in job "activation"`, "error should explain invalid built-in target")
	})

	t.Run("target depending on activation rejected", func(t *testing.T) {
		data := &WorkflowData{
			Jobs: map[string]any{
				"secrets_fetcher": map[string]any{
					"needs": "activation",
				},
			},
			OnNeeds: []string{"secrets_fetcher"},
		}

		err := validateOnNeedsTargets(data)
		require.Error(t, err, "expected on.needs validation error")
		assert.Contains(t, err.Error(), "cannot depend on activation/pre_activation", "error should explain cyclic dependency risk")
	})
}

func TestValidateOnNeedsDependencyChains(t *testing.T) {
	c := NewCompiler()

	t.Run("rejects chain where transitive dependency may get implicit activation need", func(t *testing.T) {
		data := &WorkflowData{
			Jobs: map[string]any{
				"secrets_fetcher": map[string]any{
					"needs": []any{"bootstrap"},
				},
				"bootstrap": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
			OnNeeds: []string{"secrets_fetcher"},
		}

		err := c.validateOnNeeds(data)
		require.Error(t, err, "expected transitive chain validation error")
		assert.Contains(t, err.Error(), `depends on "bootstrap"`, "error should identify problematic transitive dependency")
		assert.Contains(t, err.Error(), "implicit needs: activation", "error should explain cycle-prone implicit activation dependency")
	})

	t.Run("allows chain when transitive dependency is explicitly in on.needs", func(t *testing.T) {
		data := &WorkflowData{
			Jobs: map[string]any{
				"secrets_fetcher": map[string]any{
					"needs": []any{"bootstrap"},
				},
				"bootstrap": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
			OnNeeds: []string{"secrets_fetcher", "bootstrap"},
		}

		require.NoError(t, c.validateOnNeeds(data), "expected transitive chain to be valid when all dependencies are in on.needs")
	})
}

func TestValidateOnGitHubAppNeedsExpressions(t *testing.T) {
	c := NewCompiler()

	t.Run("allows on.needs expression in on.github-app", func(t *testing.T) {
		data := &WorkflowData{
			Jobs: map[string]any{
				"secrets_fetcher": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
			OnNeeds: []string{"secrets_fetcher"},
			ActivationGitHubApp: &GitHubAppConfig{
				AppID:      "${{ needs.secrets_fetcher.outputs.app_id }}",
				PrivateKey: "${{ needs.secrets_fetcher.outputs.private_key }}",
			},
		}

		require.NoError(t, c.validateOnNeeds(data), "expected on.github-app needs expression to validate")
	})

	t.Run("rejects unknown needs expression in on.github-app", func(t *testing.T) {
		data := &WorkflowData{
			Jobs: map[string]any{
				"secrets_fetcher": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
			ActivationGitHubApp: &GitHubAppConfig{
				AppID:      "${{ needs.missing_job.outputs.app_id }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}

		err := c.validateOnNeeds(data)
		require.Error(t, err, "expected on.github-app validation error")
		assert.Contains(t, err.Error(), `unknown job "missing_job"`, "error should identify unknown needs job")
	})

	t.Run("error field label uses client-id", func(t *testing.T) {
		data := &WorkflowData{
			Jobs: map[string]any{
				"secrets_fetcher": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
			ActivationGitHubApp: &GitHubAppConfig{
				AppID:      "${{ needs.secrets_fetcher.outputs.app_id }}",
				PrivateKey: "${{ secrets.APP_PRIVATE_KEY }}",
			},
		}

		err := c.validateOnNeeds(data)
		require.Error(t, err, "expected on.github-app validation error")
		assert.Contains(t, err.Error(), "on.github-app.client-id", "error field should use yaml key client-id")
	})
}
