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

func TestGetActiveAgentManifestFoldersAndFiles(t *testing.T) {
	t.Run("keeps existing built-in engine manifest union", func(t *testing.T) {
		c := NewCompiler(WithVersion("dev"))
		folders, files := c.getActiveAgentManifestFoldersAndFiles(&WorkflowData{
			EngineConfig: &EngineConfig{ID: "claude"},
		})

		assert.Contains(t, folders, ".agents")
		assert.Contains(t, folders, ".claude")
		assert.Contains(t, folders, ".codex")
		assert.Contains(t, folders, ".gemini")
		assert.Contains(t, files, "AGENTS.md")
		assert.Contains(t, files, "CLAUDE.md")
		assert.Contains(t, files, "GEMINI.md")
	})

	t.Run("uses declarative behavior manifest", func(t *testing.T) {
		c := NewCompiler(WithVersion("dev"))
		def := &EngineDefinition{
			ID:          "opencode-test",
			DisplayName: "OpenCode Test",
			Behaviors: &EngineBehaviorDefinition{
				Manifest: &EngineManifestDefinition{
					Files:        []string{"opencode.jsonc", "AGENTS.md"},
					PathPrefixes: []string{".opencode/"},
				},
			},
		}
		engine, err := NewBehaviorDefinedEngine(def)
		require.NoError(t, err)
		require.NoError(t, c.engineRegistry.Register(engine))
		def.RuntimeID = engine.GetID()
		c.engineCatalog.Register(def)

		folders, files := c.getActiveAgentManifestFoldersAndFiles(&WorkflowData{
			EngineConfig: &EngineConfig{ID: "opencode-test"},
		})

		assert.Equal(t, []string{".agents", ".opencode"}, folders)
		assert.ElementsMatch(t, []string{"AGENTS.md", "opencode.jsonc"}, files)
		assert.NotContains(t, folders, ".claude")
		assert.NotContains(t, folders, ".codex")
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
