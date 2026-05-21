//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInferToDisableModelInvocationCodemod(t *testing.T) {
	codemod := getInferToDisableModelInvocationCodemod()

	assert.Equal(t, "infer-to-disable-model-invocation", codemod.ID)
	assert.Equal(t, "Migrate 'infer' to 'disable-model-invocation'", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestInferToDisableModelInvocation_InferFalse(t *testing.T) {
	codemod := getInferToDisableModelInvocationCodemod()

	content := `---
on: workflow_dispatch
infer: false
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on":    "workflow_dispatch",
		"infer": false,
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "infer:")
	assert.Contains(t, result, "disable-model-invocation: true")
}

func TestInferToDisableModelInvocation_InferTrue(t *testing.T) {
	codemod := getInferToDisableModelInvocationCodemod()

	content := `---
on: workflow_dispatch
infer: true
---

# Test`

	frontmatter := map[string]any{
		"on":    "workflow_dispatch",
		"infer": true,
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "infer:")
	assert.Contains(t, result, "disable-model-invocation: false")
}

func TestInferToDisableModelInvocation_NoInferField(t *testing.T) {
	codemod := getInferToDisableModelInvocationCodemod()

	content := `---
on: workflow_dispatch
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestInferToDisableModelInvocation_BothFieldsPresent(t *testing.T) {
	codemod := getInferToDisableModelInvocationCodemod()

	content := `---
on: workflow_dispatch
infer: false
disable-model-invocation: true
---

# Test`

	frontmatter := map[string]any{
		"on":                       "workflow_dispatch",
		"infer":                    false,
		"disable-model-invocation": true,
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "infer:")
	assert.Contains(t, result, "disable-model-invocation: true")
}

func TestInferToDisableModelInvocation_PreservesOtherFields(t *testing.T) {
	codemod := getInferToDisableModelInvocationCodemod()

	content := `---
on: workflow_dispatch
engine: copilot
infer: false
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on":     "workflow_dispatch",
		"engine": "copilot",
		"infer":  false,
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "infer:")
	assert.Contains(t, result, "disable-model-invocation: true")
	assert.Contains(t, result, "engine: copilot")
	assert.Contains(t, result, "contents: read")
}
