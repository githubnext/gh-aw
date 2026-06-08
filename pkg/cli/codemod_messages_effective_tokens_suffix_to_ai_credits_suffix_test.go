//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod(t *testing.T) {
	codemod := getMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod()

	assert.Equal(t, "messages-effective-tokens-suffix-to-ai-credits-suffix", codemod.ID)
	assert.Equal(t, "Migrate safe-outputs messages ET suffix placeholder to AI credits suffix", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.48", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod_MigratesMessagesTemplates(t *testing.T) {
	codemod := getMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod()

	content := `---
safe-outputs:
  messages:
    footer: "> Run {effective_tokens_suffix}"
    run-failure: "Failed {effective_tokens_suffix}"
---

# Workflow`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"messages": map[string]any{
				"footer":      "> Run {effective_tokens_suffix}",
				"run-failure": "Failed {effective_tokens_suffix}",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, `footer: "> Run {ai_credits_suffix}"`)
	assert.Contains(t, result, `run-failure: "Failed {ai_credits_suffix}"`)
	assert.NotContains(t, result, "{effective_tokens_suffix}")
	assert.Contains(t, result, "\n# Workflow")
}

func TestMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod_NoOpWhenPlaceholderMissing(t *testing.T) {
	codemod := getMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod()

	content := `---
safe-outputs:
  messages:
    footer: "> Run {ai_credits_suffix}"
---`
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"messages": map[string]any{
				"footer": "> Run {ai_credits_suffix}",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod_IdempotentAfterMigration(t *testing.T) {
	codemod := getMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod()

	content := `---
safe-outputs:
  messages:
    footer: "> Run {ai_credits_suffix}"
---`
	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"messages": map[string]any{
				"footer": "> Run {ai_credits_suffix}",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestMessagesETSuffixCodemod_PreservesFormattingAndBlockScalars(t *testing.T) {
	codemod := getMessagesEffectiveTokensSuffixToAICreditsSuffixCodemod()

	content := `---
safe-outputs:
  messages:
    # keep this comment
    footer: "> Run {effective_tokens_suffix}" # keep inline comment
    run-failure: |
      Failed {effective_tokens_suffix}
      Retry {effective_tokens_suffix}
on: workflow_dispatch
---

# Workflow Body`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"messages": map[string]any{
				"footer":      "> Run {effective_tokens_suffix}",
				"run-failure": "Failed {effective_tokens_suffix}\nRetry {effective_tokens_suffix}",
			},
		},
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, `    # keep this comment`)
	assert.Contains(t, result, `footer: "> Run {ai_credits_suffix}" # keep inline comment`)
	assert.Contains(t, result, "      Failed {ai_credits_suffix}")
	assert.Contains(t, result, "      Retry {ai_credits_suffix}")
	assert.Contains(t, result, "\non: workflow_dispatch\n")
	assert.Contains(t, result, "\n# Workflow Body")
}
