//go:build !integration

package cli

import (
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInferToDisableModelInvocation_AWAST(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name    string
		input   string
		want    any
		applied bool
	}{
		{"quoted false", "'infer': false # preserve\nnested: {infer: false}", true, true},
		{"flow true", `{"infer": true, nested: {infer: false}} # preserve`, false, true},
		{"authoritative null", `{'infer': {child: true}, disable-model-invocation: null, nested: {infer: false}} # preserve`, nil, true},
		{"nonboolean", `{'infer': "false", nested: {infer: false}} # preserve`, nil, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			body := "\n---\n\n# Body\ninfer: false\n"
			content := "---\n" + tt.input + body
			result, applied, err := getInferToDisableModelInvocationCodemod().Apply(content, nil)
			require.NoError(t, err)
			require.Equal(t, tt.applied, applied)
			if !applied {
				assert.Equal(t, content, result)
				return
			}
			document, err := aw.Parse(result)
			require.NoError(t, err)
			assert.NotContains(t, document.Frontmatter, "infer")
			assert.Equal(t, tt.want, document.Frontmatter["disable-model-invocation"])
			assert.Equal(t, map[string]any{"infer": false}, document.Frontmatter["nested"])
			assert.Contains(t, result, "# preserve")
			assert.Contains(t, result, body)
			repeated, changed, err := getInferToDisableModelInvocationCodemod().Apply(result, nil)
			require.NoError(t, err)
			assert.False(t, changed)
			assert.Equal(t, result, repeated)
		})
	}
}

func TestGetInferToDisableModelInvocationCodemod(t *testing.T) {
	t.Parallel()
	codemod := getInferToDisableModelInvocationCodemod()

	assert.Equal(t, "infer-to-disable-model-invocation", codemod.ID)
	assert.Equal(t, "Migrate 'infer' to 'disable-model-invocation'", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestInferToDisableModelInvocation_InferFalse(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
