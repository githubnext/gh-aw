//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPluginsToDependenciesCodemod(t *testing.T) {
	codemod := getPluginsToDependenciesCodemod()

	assert.Equal(t, "plugins-to-dependencies", codemod.ID)
	assert.Equal(t, "Migrate plugins to dependencies", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.0.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestPluginsToDependenciesCodemod_NoPlugins(t *testing.T) {
	codemod := getPluginsToDependenciesCodemod()

	content := `---
on: workflow_dispatch
engine: copilot
---

# No plugins`

	frontmatter := map[string]any{
		"on":     "workflow_dispatch",
		"engine": "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "Codemod should not be applied when plugins is absent")
	assert.Equal(t, content, result, "Content should not be modified")
}

func TestPluginsToDependenciesCodemod_ArrayFormat(t *testing.T) {
	codemod := getPluginsToDependenciesCodemod()

	content := `---
on:
  issues:
    types: [opened]
engine: copilot
plugins:
  - github/test-plugin
  - acme/custom-tools
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{"types": []any{"opened"}},
		},
		"engine":  "copilot",
		"plugins": []any{"github/test-plugin", "acme/custom-tools"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.NotContains(t, result, "plugins:", "plugins key should be removed")
	assert.Contains(t, result, "dependencies:", "dependencies key should be present")
	assert.Contains(t, result, "- github/test-plugin", "first plugin should be preserved")
	assert.Contains(t, result, "- acme/custom-tools", "second plugin should be preserved")
}

func TestPluginsToDependenciesCodemod_ObjectFormat(t *testing.T) {
	codemod := getPluginsToDependenciesCodemod()

	content := `---
on:
  issues:
    types: [opened]
engine: copilot
plugins:
  repos:
    - github/test-plugin
    - acme/custom-tools
  github-token: ${{ secrets.MY_TOKEN }}
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{"types": []any{"opened"}},
		},
		"engine": "copilot",
		"plugins": map[string]any{
			"repos":        []any{"github/test-plugin", "acme/custom-tools"},
			"github-token": "${{ secrets.MY_TOKEN }}",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.NotContains(t, result, "plugins:", "plugins key should be removed")
	assert.Contains(t, result, "dependencies:", "dependencies key should be present")
	assert.Contains(t, result, "github/test-plugin", "first plugin should be preserved")
	assert.Contains(t, result, "acme/custom-tools", "second plugin should be preserved")
	assert.Contains(t, result, "github-token:", "github-token should be preserved")
	assert.Contains(t, result, "packages:", "repos should be renamed to packages")
	assert.NotContains(t, result, "repos:", "repos sub-key should be renamed")
}

func TestPluginsToDependenciesCodemod_SkipsWhenDepsExist(t *testing.T) {
	codemod := getPluginsToDependenciesCodemod()

	content := `---
on: workflow_dispatch
plugins:
  - github/test-plugin
dependencies:
  - microsoft/apm-sample-package
---`

	frontmatter := map[string]any{
		"on":           "workflow_dispatch",
		"plugins":      []any{"github/test-plugin"},
		"dependencies": []any{"microsoft/apm-sample-package"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "Codemod should be skipped when dependencies already exist")
	assert.Equal(t, content, result, "Content should not be modified")
}

func TestPluginsToDependenciesCodemod_PreservesMarkdownBody(t *testing.T) {
	codemod := getPluginsToDependenciesCodemod()

	content := `---
engine: copilot
plugins:
  - github/test-plugin
---

# My workflow

Install the plugin and do work.`

	frontmatter := map[string]any{
		"engine":  "copilot",
		"plugins": []any{"github/test-plugin"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.Contains(t, result, "# My workflow", "Markdown body should be preserved")
	assert.Contains(t, result, "Install the plugin and do work.", "Markdown body should be preserved")
}
