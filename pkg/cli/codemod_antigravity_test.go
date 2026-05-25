package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiToAntigravityCodemod_Metadata(t *testing.T) {
	codemod := getGeminiToAntigravityCodemod()

	assert.Equal(t, "gemini-to-antigravity", codemod.ID)
	assert.Equal(t, "Migrate Gemini engine to Antigravity", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	require.NotNil(t, codemod.Apply)
}

func TestGeminiToAntigravityCodemod_Apply(t *testing.T) {
	codemod := getGeminiToAntigravityCodemod()
	input := `---
name: Smoke Gemini
description: Smoke test workflow that validates Gemini engine functionality
engine:
  id: gemini
  model: gemini-2.5-pro
env:
  GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
---

# Smoke Test: Gemini Engine Validation

run: gemini --prompt "$(cat prompt.txt)"
cache-dir: .gemini/cache
footer: Powered by Gemini CLI from Google Gemini
`

	result, applied, err := codemod.Apply(input, map[string]any{
		"engine": map[string]any{
			"id":    "gemini",
			"model": "gemini-2.5-pro",
		},
	})
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "name: Smoke Antigravity")
	assert.Contains(t, result, "description: Smoke test workflow that validates Antigravity engine functionality")
	assert.Contains(t, result, "id: antigravity")
	assert.Contains(t, result, "# MANUAL MIGRATION: review former Gemini model mapping for Antigravity.")
	assert.Contains(t, result, "  model: gemini-2.5-pro")
	assert.Contains(t, result, "ANTIGRAVITY_API_KEY: ${{ secrets.ANTIGRAVITY_API_KEY }}")
	assert.Contains(t, result, "run: agy --prompt")
	assert.Contains(t, result, "cache-dir: .antigravity/cache")
	assert.Contains(t, result, "footer: Powered by Antigravity CLI from Antigravity")
	assert.NotContains(t, result, "Smoke Gemini")
	assert.NotContains(t, result, "GEMINI_API_KEY")
}

func TestGeminiToAntigravityCodemod_Idempotent(t *testing.T) {
	codemod := getGeminiToAntigravityCodemod()
	input := `---
name: Smoke Antigravity
engine:
  id: antigravity
---

run: agy --prompt "$(cat prompt.txt)"
`

	result, applied, err := codemod.Apply(input, map[string]any{
		"engine": map[string]any{"id": "antigravity"},
	})
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, input, result)
}
