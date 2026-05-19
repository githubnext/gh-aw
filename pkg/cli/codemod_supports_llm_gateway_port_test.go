//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupportsLLMGatewayToLLMGatewayPortCodemod_Metadata(t *testing.T) {
	codemod := getSupportsLLMGatewayToLLMGatewayPortCodemod()

	assert.Equal(t, "supports-llm-gateway-to-llm-gateway-port", codemod.ID)
	assert.Equal(t, "Replace supportsLLMGateway with llmGatewayPort", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.15.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestSupportsLLMGatewayToLLMGatewayPortCodemod_MigratesTrue(t *testing.T) {
	codemod := getSupportsLLMGatewayToLLMGatewayPortCodemod()

	content := `---
engine:
  id: custom-engine
  supportsLLMGateway: true
---
`
	frontmatter := map[string]any{
		"engine": map[string]any{
			"id":                 "custom-engine",
			"supportsLLMGateway": true,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "llmGatewayPort: 8080")
	assert.NotContains(t, result, "supportsLLMGateway:")
}

func TestSupportsLLMGatewayToLLMGatewayPortCodemod_NoOpWhenFalse(t *testing.T) {
	codemod := getSupportsLLMGatewayToLLMGatewayPortCodemod()

	content := `---
engine:
  id: custom-engine
  supportsLLMGateway: false
---
`
	frontmatter := map[string]any{
		"engine": map[string]any{
			"id":                 "custom-engine",
			"supportsLLMGateway": false,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)
	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}
