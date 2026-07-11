//go:build !integration

package workflow

import (
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveActivationEngineID(t *testing.T) {
	t.Run("defaults when empty", func(t *testing.T) {
		assert.Equal(t, string(constants.DefaultEngine), resolveActivationEngineID(&WorkflowData{}))
	})

	t.Run("trims configured value", func(t *testing.T) {
		data := &WorkflowData{EngineConfig: &EngineConfig{ID: "  copilot  "}}
		assert.Equal(t, "copilot", resolveActivationEngineID(data))
	})
}

func TestBuildDailyAICActivationJobEnv(t *testing.T) {
	data := &WorkflowData{
		MaxDailyAICredits: strPtr("25"),
		RawFrontmatter:    map[string]any{},
	}
	env := buildDailyAICActivationJobEnv(data)
	require.NotNil(t, env)
	assert.Equal(t, `"25"`, env[maxDailyAICreditsEnvVar])

	data.MaxDailyAICredits = strPtr("${{ vars.MAX_DAILY_AICREDITS }}")
	env = buildDailyAICActivationJobEnv(data)
	require.NotNil(t, env)
	assert.Equal(t, "${{ vars.MAX_DAILY_AICREDITS }}", env[maxDailyAICreditsEnvVar])
}

func TestBuildActivationTextOutputEnvLines(t *testing.T) {
	data := &WorkflowData{Bots: []string{"dependabot[bot]", "copilot[bot]"}}
	lines := buildActivationTextOutputEnvLines(data, "api.github.com")
	require.Len(t, lines, 2)
	assert.Contains(t, lines[0], "GH_AW_ALLOWED_BOTS")
	assert.Contains(t, lines[1], "GH_AW_ALLOWED_DOMAINS")
}

func TestEnsureActivationCommentOutputs(t *testing.T) {
	ctx := &activationJobBuildContext{
		outputs: map[string]string{
			"comment_id": "existing",
		},
	}
	ensureActivationCommentOutputs(ctx)

	assert.Equal(t, "existing", ctx.outputs["comment_id"])
	assert.Equal(t, `""`, ctx.outputs["comment_repo"])
}
