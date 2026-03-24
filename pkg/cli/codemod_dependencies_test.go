//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDependenciesToImportsAPMPackagesCodemod(t *testing.T) {
	codemod := getDependenciesToImportsAPMPackagesCodemod()

	assert.Equal(t, "dependencies-to-imports-apm-packages", codemod.ID)
	assert.Equal(t, "Migrate dependencies to imports.apm-packages", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "1.18.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestDependenciesToImportsAPMPackagesCodemod_NoDependencies(t *testing.T) {
	codemod := getDependenciesToImportsAPMPackagesCodemod()

	content := `---
on: workflow_dispatch
engine: copilot
---

# No dependencies`

	frontmatter := map[string]any{
		"on":     "workflow_dispatch",
		"engine": "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "Codemod should not be applied when dependencies is absent")
	assert.Equal(t, content, result, "Content should not be modified")
}

func TestDependenciesToImportsAPMPackagesCodemod_SimpleArray_NoImports(t *testing.T) {
	codemod := getDependenciesToImportsAPMPackagesCodemod()

	content := `---
on:
  issues:
    types: [opened]
engine: copilot
dependencies:
  - microsoft/apm-sample-package
  - github/awesome-copilot
---

# Test workflow`

	frontmatter := map[string]any{
		"on":           map[string]any{"issues": map[string]any{"types": []any{"opened"}}},
		"engine":       "copilot",
		"dependencies": []any{"microsoft/apm-sample-package", "github/awesome-copilot"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.NotContains(t, result, "dependencies:", "dependencies key should be removed")
	assert.Contains(t, result, "imports:", "imports key should be present")
	assert.Contains(t, result, "apm-packages:", "apm-packages key should be present")
	assert.Contains(t, result, "- microsoft/apm-sample-package", "first package should be present")
	assert.Contains(t, result, "- github/awesome-copilot", "second package should be present")
}

func TestDependenciesToImportsAPMPackagesCodemod_ObjectFormat_NoImports(t *testing.T) {
	codemod := getDependenciesToImportsAPMPackagesCodemod()

	content := `---
on:
  issues:
    types: [opened]
engine: copilot
dependencies:
  packages:
    - acme-org/acme-skills
  github-app:
    app-id: ${{ vars.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
---

# Test workflow`

	frontmatter := map[string]any{
		"on":     map[string]any{"issues": map[string]any{"types": []any{"opened"}}},
		"engine": "copilot",
		"dependencies": map[string]any{
			"packages": []any{"acme-org/acme-skills"},
			"github-app": map[string]any{
				"app-id":      "${{ vars.APP_ID }}",
				"private-key": "${{ secrets.APP_PRIVATE_KEY }}",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.NotContains(t, result, "dependencies:", "dependencies key should be removed")
	assert.Contains(t, result, "imports:", "imports key should be present")
	assert.Contains(t, result, "apm-packages:", "apm-packages key should be present")
	assert.Contains(t, result, "packages:", "packages sub-key should be preserved")
	assert.Contains(t, result, "github-app:", "github-app sub-key should be preserved")
	assert.Contains(t, result, "acme-org/acme-skills", "package should be preserved")
}

func TestDependenciesToImportsAPMPackagesCodemod_WithExistingArrayImports(t *testing.T) {
	codemod := getDependenciesToImportsAPMPackagesCodemod()

	content := `---
on: workflow_dispatch
imports:
  - shared/common.md
  - shared/tools.md
dependencies:
  - microsoft/apm-sample-package
---

# Test workflow`

	frontmatter := map[string]any{
		"on":           "workflow_dispatch",
		"imports":      []any{"shared/common.md", "shared/tools.md"},
		"dependencies": []any{"microsoft/apm-sample-package"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.NotContains(t, result, "dependencies:", "dependencies key should be removed")
	assert.Contains(t, result, "imports:", "imports key should be present")
	assert.Contains(t, result, "aw:", "aw subfield should be present")
	assert.Contains(t, result, "apm-packages:", "apm-packages key should be present")
	assert.Contains(t, result, "shared/common.md", "existing import should be preserved")
	assert.Contains(t, result, "shared/tools.md", "existing import should be preserved")
	assert.Contains(t, result, "microsoft/apm-sample-package", "package should be present")
}

func TestDependenciesToImportsAPMPackagesCodemod_WithExistingObjectImports(t *testing.T) {
	codemod := getDependenciesToImportsAPMPackagesCodemod()

	content := `---
on: workflow_dispatch
imports:
  aw:
    - shared/common.md
dependencies:
  - microsoft/apm-sample-package
---

# Test workflow`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"imports": map[string]any{
			"aw": []any{"shared/common.md"},
		},
		"dependencies": []any{"microsoft/apm-sample-package"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.NotContains(t, result, "dependencies:", "dependencies key should be removed")
	assert.Contains(t, result, "imports:", "imports key should be present")
	assert.Contains(t, result, "apm-packages:", "apm-packages key should be added")
	assert.Contains(t, result, "shared/common.md", "existing aw import should be preserved")
	assert.Contains(t, result, "microsoft/apm-sample-package", "package should be present")
}

func TestDependenciesToImportsAPMPackagesCodemod_SkipsWhenAPMPackagesExist(t *testing.T) {
	codemod := getDependenciesToImportsAPMPackagesCodemod()

	content := `---
on: workflow_dispatch
imports:
  apm-packages:
    - existing/package
dependencies:
  - microsoft/apm-sample-package
---`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"imports": map[string]any{
			"apm-packages": []any{"existing/package"},
		},
		"dependencies": []any{"microsoft/apm-sample-package"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "Codemod should be skipped when imports.apm-packages already exists")
	assert.Equal(t, content, result, "Content should not be modified")
}

func TestDependenciesToImportsAPMPackagesCodemod_PreservesMarkdownBody(t *testing.T) {
	codemod := getDependenciesToImportsAPMPackagesCodemod()

	content := `---
engine: copilot
dependencies:
  - microsoft/apm-sample-package
---

# My workflow

Use the skills provided.`

	frontmatter := map[string]any{
		"engine":       "copilot",
		"dependencies": []any{"microsoft/apm-sample-package"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied, "Codemod should have been applied")
	assert.Contains(t, result, "# My workflow", "Markdown body should be preserved")
	assert.Contains(t, result, "Use the skills provided.", "Markdown body should be preserved")
}
