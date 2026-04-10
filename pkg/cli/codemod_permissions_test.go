//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPermissionsReadCodemod(t *testing.T) {
	codemod := getExpandPermissionsShorthandCodemod()

	assert.Equal(t, "permissions-read-to-read-all", codemod.ID)
	assert.Equal(t, "Convert invalid permissions shorthand", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.5.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestPermissionsReadCodemod_Read(t *testing.T) {
	codemod := getExpandPermissionsShorthandCodemod()

	content := `---
on: workflow_dispatch
permissions: read
---

# Test`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"permissions": "read",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "permissions: read-all")
	assert.NotContains(t, result, "permissions: read\n")
}

func TestPermissionsReadCodemod_Write(t *testing.T) {
	codemod := getExpandPermissionsShorthandCodemod()

	content := `---
on: workflow_dispatch
permissions: write
---

# Test`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"permissions": "write",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "permissions: write-all")
	assert.NotContains(t, result, "permissions: write\n")
}

func TestPermissionsReadCodemod_NoChange_ReadAll(t *testing.T) {
	codemod := getExpandPermissionsShorthandCodemod()

	content := `---
on: workflow_dispatch
permissions: read-all
---

# Test`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"permissions": "read-all",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestPermissionsReadCodemod_NoChange_WriteAll(t *testing.T) {
	codemod := getExpandPermissionsShorthandCodemod()

	content := `---
on: workflow_dispatch
permissions: write-all
---

# Test`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"permissions": "write-all",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestPermissionsReadCodemod_NoChange_MapFormat(t *testing.T) {
	codemod := getExpandPermissionsShorthandCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: read
  issues: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "read",
			"issues":   "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestPermissionsReadCodemod_NoPermissions(t *testing.T) {
	codemod := getExpandPermissionsShorthandCodemod()

	content := `---
on: workflow_dispatch
timeout-minutes: 30
---

# Test`

	frontmatter := map[string]any{
		"on":              "workflow_dispatch",
		"timeout-minutes": 30,
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestPermissionsReadCodemod_PreservesMarkdown(t *testing.T) {
	codemod := getExpandPermissionsShorthandCodemod()

	content := `---
on: workflow_dispatch
permissions: read
---

# Test Workflow

This workflow needs permissions.`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"permissions": "read",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "# Test Workflow")
	assert.Contains(t, result, "This workflow needs permissions.")
}

func TestGetWritePermissionsCodemod(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	assert.Equal(t, "write-permissions-to-read-migration", codemod.ID)
	assert.Equal(t, "Convert write permissions to read", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.4.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestWritePermissionsCodemod_ShorthandWriteAll(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions: write-all
---

# Test`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"permissions": "write-all",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "permissions: read-all")
	assert.NotContains(t, result, "write-all")
}

func TestWritePermissionsCodemod_ShorthandWrite(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions: write
---

# Test`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"permissions": "write",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "permissions: read")
	assert.NotContains(t, result, "permissions: write")
}

func TestWritePermissionsCodemod_MapFormat(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: write
  issues: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "write",
			"issues":   "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "contents: read")
	assert.Contains(t, result, "issues: read")
	assert.NotContains(t, result, "contents: write")
}

func TestWritePermissionsCodemod_MultipleWritePermissions(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: write
  pull-requests: write
  issues: write
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents":      "write",
			"pull-requests": "write",
			"issues":        "write",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "contents: read")
	assert.Contains(t, result, "pull-requests: read")
	assert.Contains(t, result, "issues: read")
}

func TestWritePermissionsCodemod_NoPermissionsField(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
timeout-minutes: 30
---

# Test`

	frontmatter := map[string]any{
		"on":              "workflow_dispatch",
		"timeout-minutes": 30,
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestWritePermissionsCodemod_OnlyReadPermissions(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: read
  issues: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "read",
			"issues":   "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestWritePermissionsCodemod_PreservesIndentation(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: write
  issues: write
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "write",
			"issues":   "write",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "  contents: read")
	assert.Contains(t, result, "  issues: read")
}

func TestWritePermissionsCodemod_PreservesComments(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: write  # Write access for commits
  issues: read  # Read-only for issues
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "write",
			"issues":   "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "contents: read  # Write access for commits")
	assert.Contains(t, result, "issues: read  # Read-only for issues")
}

func TestWritePermissionsCodemod_PreservesMarkdown(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions: write-all
---

# Test Workflow

This workflow needs permissions.`

	frontmatter := map[string]any{
		"on":          "workflow_dispatch",
		"permissions": "write-all",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "# Test Workflow")
	assert.Contains(t, result, "This workflow needs permissions.")
}

func TestWritePermissionsCodemod_SkipsIdToken(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: read
  id-token: write
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "read",
			"id-token": "write",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "codemod should not return an error")
	assert.False(t, applied, "Should not be applied when only write-only permissions have write")
	assert.Equal(t, content, result, "codemod result should be unchanged when only write-only permissions have write")
	assert.Contains(t, result, "id-token: write", "id-token permission should remain write after codemod")
	assert.NotContains(t, result, "id-token: read", "id-token should never be converted to read")
}

func TestWritePermissionsCodemod_SkipsCopilotRequests(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: read
  copilot-requests: write
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents":         "read",
			"copilot-requests": "write",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "codemod should not return an error")
	assert.False(t, applied, "Should not be applied when only write-only permissions have write")
	assert.Equal(t, content, result, "codemod result should be unchanged when only write-only permissions have write")
	assert.Contains(t, result, "copilot-requests: write", "copilot-requests permission should remain write after codemod")
	assert.NotContains(t, result, "copilot-requests: read", "copilot-requests should never be converted to read")
}

func TestWritePermissionsCodemod_MixedWithIdToken(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: write
  issues: write
  id-token: write
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "write",
			"issues":   "write",
			"id-token": "write",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "codemod should not return an error")
	assert.True(t, applied, "codemod should be applied when non-write-only permissions have write")
	assert.Contains(t, result, "contents: read", "contents permission should be downgraded from write to read")
	assert.Contains(t, result, "issues: read", "issues permission should be downgraded from write to read")
	// id-token must remain write — "read" is not a valid value for it
	assert.Contains(t, result, "id-token: write", "id-token permission should remain write after codemod")
	assert.NotContains(t, result, "id-token: read", "id-token should never be converted to read")
}

func TestWritePermissionsCodemod_MixedWithCopilotRequests(t *testing.T) {
	codemod := getMigrateWritePermissionsToReadCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: write
  copilot-requests: write
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents":         "write",
			"copilot-requests": "write",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err, "codemod should not return an error")
	assert.True(t, applied, "codemod should be applied when non-write-only permissions have write")
	assert.Contains(t, result, "contents: read", "contents permission should be downgraded from write to read")
	// copilot-requests must remain write — "read" is not a valid value for it
	assert.Contains(t, result, "copilot-requests: write", "copilot-requests permission should remain write after codemod")
	assert.NotContains(t, result, "copilot-requests: read", "copilot-requests should never be converted to read")
}
