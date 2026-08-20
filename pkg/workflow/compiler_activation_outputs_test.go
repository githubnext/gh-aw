//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestActivationArtifactNameIsValid(t *testing.T) {
	data := &WorkflowData{On: `"on":
  push:`}
	ctx := &activationJobBuildContext{data: data}
	compiler := NewCompiler()

	compiler.addActivationArtifactUploadStep(ctx)

	assert.Contains(t, strings.Join(ctx.steps, ""), "name: activation-artifact")
}

// TestConfigureActivationNeedsAndCondition_EngineEnvJobReferences tests that custom jobs
// referenced via needs.<job>.outputs.* in engine.env values are added to the activation
// job's needs list.
func TestConfigureActivationNeedsAndCondition_EngineEnvJobReferences(t *testing.T) {
	t.Run("engine.env reference adds custom job to activation needs without pre_activation", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"COPILOT_GITHUB_TOKEN": "${{ needs.custom_token.outputs.token }}",
				},
			},
			Jobs: map[string]any{
				"custom_token": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
		}
		ctx := &activationJobBuildContext{
			data:             data,
			preActivationJob: false,
		}

		c.configureActivationNeedsAndCondition(ctx)

		assert.Contains(t, ctx.activationNeeds, "custom_token",
			"activation job must depend on custom_token referenced in engine.env")
		assert.Contains(t, ctx.customJobsBeforeActivation, "custom_token",
			"custom_token must be in customJobsBeforeActivation")
	})

	t.Run("engine.env reference with case() expression adds custom job to activation needs", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"COPILOT_GITHUB_TOKEN": "${{ case(needs.custom_token.outputs.use_custom == true, secrets.CUSTOM_TOKEN, secrets.COPILOT_GITHUB_TOKEN) }}",
				},
			},
			Jobs: map[string]any{
				"custom_token": map[string]any{
					"runs-on": "ubuntu-latest",
					"outputs": map[string]any{
						"use_custom": "${{ steps.check.outputs.use_custom }}",
					},
				},
			},
		}
		ctx := &activationJobBuildContext{
			data:             data,
			preActivationJob: false,
		}

		c.configureActivationNeedsAndCondition(ctx)

		assert.Contains(t, ctx.activationNeeds, "custom_token",
			"activation job must depend on custom_token referenced via case() in engine.env")
	})

	t.Run("engine.env reference adds custom job to activation needs with pre_activation", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"COPILOT_GITHUB_TOKEN": "${{ needs.token_provider.outputs.token }}",
				},
			},
			Jobs: map[string]any{
				"token_provider": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
		}
		ctx := &activationJobBuildContext{
			data:             data,
			preActivationJob: true,
		}

		c.configureActivationNeedsAndCondition(ctx)

		assert.Contains(t, ctx.activationNeeds, "token_provider",
			"activation job must depend on token_provider referenced in engine.env (with pre_activation)")
	})

	t.Run("engine.env reference to custom job with explicit needs does not add activation dependency", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"COPILOT_GITHUB_TOKEN": "${{ needs.custom_token.outputs.token }}",
				},
			},
			Jobs: map[string]any{
				"custom_token": map[string]any{
					"runs-on": "ubuntu-latest",
					"needs":   "build_context",
				},
				"build_context": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
		}
		ctx := &activationJobBuildContext{
			data:             data,
			preActivationJob: false,
		}

		c.configureActivationNeedsAndCondition(ctx)

		assert.NotContains(t, ctx.activationNeeds, "custom_token",
			"custom_token with explicit needs must not be re-added to activation dependencies")
		assert.NotContains(t, ctx.customJobsBeforeActivation, "custom_token",
			"custom_token with explicit needs must not be added to customJobsBeforeActivation via engine.env scanning")
	})

	t.Run("engine.env with no needs references does not add extra deps", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"COPILOT_GITHUB_TOKEN": "${{ secrets.COPILOT_GITHUB_TOKEN }}",
				},
			},
			Jobs: map[string]any{
				"some_job": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
		}
		ctx := &activationJobBuildContext{
			data:             data,
			preActivationJob: false,
		}

		c.configureActivationNeedsAndCondition(ctx)

		assert.NotContains(t, ctx.activationNeeds, "some_job",
			"some_job must NOT be in activation needs when not referenced in engine.env")
	})

	t.Run("engine.env references outside activation-rendered keys do not add activation deps", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"FOO":                  "${{ needs.prepare.outputs.value }}",
					"COPILOT_GITHUB_TOKEN": "${{ secrets.COPILOT_GITHUB_TOKEN }}",
				},
			},
			Jobs: map[string]any{
				"prepare": map[string]any{
					"runs-on": "ubuntu-latest",
				},
			},
		}
		ctx := &activationJobBuildContext{
			data:             data,
			preActivationJob: false,
		}

		c.configureActivationNeedsAndCondition(ctx)

		assert.NotContains(t, ctx.activationNeeds, "prepare",
			"agent-only engine.env overrides should not force prepare before activation")
		assert.NotContains(t, ctx.customJobsBeforeActivation, "prepare",
			"agent-only engine.env overrides should not be treated as activation dependencies")
	})

	t.Run("engine.env reference to non-custom (built-in) job is not added to activation needs", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"SOME_VAR": "${{ needs.activation.outputs.model }}",
				},
			},
			Jobs: map[string]any{},
		}
		ctx := &activationJobBuildContext{
			data:             data,
			preActivationJob: false,
		}

		c.configureActivationNeedsAndCondition(ctx)

		assert.Empty(t, ctx.activationNeeds,
			"built-in job references in engine.env should not be added to activation needs")
	})

	t.Run("nil engine.env does not panic", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			Jobs: map[string]any{
				"some_job": map[string]any{"runs-on": "ubuntu-latest"},
			},
		}
		ctx := &activationJobBuildContext{
			data:             data,
			preActivationJob: false,
		}

		assert.NotPanics(t, func() {
			c.configureActivationNeedsAndCondition(ctx)
		})
	})

	t.Run("job already in customJobsBeforeActivation is not duplicated", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"COPILOT_GITHUB_TOKEN": "${{ needs.custom_token.outputs.token }}",
				},
			},
			// custom_token depends on pre_activation so it's already added via getCustomJobsDependingOnPreActivation
			Jobs: map[string]any{
				"custom_token": map[string]any{
					"runs-on": "ubuntu-latest",
					"needs":   "pre_activation",
				},
			},
		}
		ctx := &activationJobBuildContext{
			data:             data,
			preActivationJob: true,
		}

		c.configureActivationNeedsAndCondition(ctx)

		count := 0
		for _, n := range ctx.activationNeeds {
			if n == "custom_token" {
				count++
			}
		}
		assert.Equal(t, 1, count, "custom_token should appear exactly once in activationNeeds")
	})

	t.Run("engine.env helper returns referenced custom jobs in sorted order", func(t *testing.T) {
		c := NewCompiler()
		data := &WorkflowData{
			EngineConfig: &EngineConfig{
				Env: map[string]string{
					"COPILOT_GITHUB_TOKEN": "${{ case(needs.z_job.outputs.value != '', needs.z_job.outputs.value, needs.a_job.outputs.value) }}",
				},
			},
			Jobs: map[string]any{
				"z_job": map[string]any{"runs-on": "ubuntu-latest"},
				"a_job": map[string]any{"runs-on": "ubuntu-latest"},
			},
		}

		referenced := c.getEngineEnvReferencedCustomJobsWithNoExplicitNeeds(data)
		assert.Equal(t, []string{"a_job", "z_job"}, referenced)
	})
}
