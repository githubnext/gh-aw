//go:build !integration

package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRolesToOnRolesCodemod(t *testing.T) {
	codemod := getRolesToOnRolesCodemod()

	assert.Equal(t, "roles-to-on-roles", codemod.ID)
	assert.Equal(t, "Move roles to on.roles", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.10.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestRolesToOnRolesCodemod_SingleLineArray(t *testing.T) {
	codemod := getRolesToOnRolesCodemod()

	content := `---
on:
  issues:
    types: [opened]
roles: [admin, maintainer, write]
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"roles": []any{"admin", "maintainer", "write"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "on:")
	assert.Contains(t, result, "roles: [admin, maintainer, write]")
	assert.NotContains(t, result, "\nroles: [admin, maintainer, write]")
	// Ensure roles is nested under on:
	lines := strings.Split(result, "\n")
	foundOn := false
	foundRoles := false
	for _, line := range lines {
		if line == "on:" {
			foundOn = true
		}
		if foundOn && strings.Contains(line, "roles:") {
			foundRoles = true
			// Check that roles line has indentation (nested under on:)
			assert.Greater(t, len(line), len(strings.TrimSpace(line)), "roles should be indented under on:")
			break
		}
	}
	assert.True(t, foundRoles, "Should find roles nested under on:")
}

func TestRolesToOnRolesCodemod_MultiLineArray(t *testing.T) {
	codemod := getRolesToOnRolesCodemod()

	content := `---
on:
  issues:
    types: [opened]
roles:
  - admin
  - maintainer
  - write
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"roles": []any{"admin", "maintainer", "write"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "on:")
	assert.Contains(t, result, "roles:")
	assert.Contains(t, result, "- admin")
	assert.Contains(t, result, "- maintainer")
	assert.Contains(t, result, "- write")
}

func TestRolesToOnRolesCodemod_AllValue(t *testing.T) {
	codemod := getRolesToOnRolesCodemod()

	content := `---
on:
  issues:
    types: [opened]
roles: all
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"roles": "all",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "roles: all")
	// Ensure roles is nested under on:
	lines := strings.Split(result, "\n")
	foundOn := false
	foundRoles := false
	for _, line := range lines {
		if line == "on:" {
			foundOn = true
		}
		if foundOn && strings.Contains(line, "roles:") {
			foundRoles = true
			// Check that roles line has indentation
			assert.Greater(t, len(line), len(strings.TrimSpace(line)), "roles should be indented under on:")
			break
		}
	}
	assert.True(t, foundRoles, "Should find roles nested under on:")
}

func TestRolesToOnRolesCodemod_NoOnBlock(t *testing.T) {
	codemod := getRolesToOnRolesCodemod()

	content := `---
roles: [admin, write]
engine: copilot
---

# Test workflow`

	frontmatter := map[string]any{
		"roles":  []any{"admin", "write"},
		"engine": "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "on:")
	assert.Contains(t, result, "roles: [admin, write]")
	// Ensure on: block is created
	lines := strings.Split(result, "\n")
	foundOn := slices.Contains(lines, "on:")
	assert.True(t, foundOn, "Should create new on: block")
}

func TestRolesToOnRolesCodemod_NoChange_NoRoles(t *testing.T) {
	codemod := getRolesToOnRolesCodemod()

	content := `---
on:
  issues:
    types: [opened]
engine: copilot
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
		},
		"engine": "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestRolesToOnRolesCodemod_NoChange_OnRolesExists(t *testing.T) {
	codemod := getRolesToOnRolesCodemod()

	content := `---
on:
  issues:
    types: [opened]
  roles: [admin, write]
roles: [admin, maintainer, write]
---

# Test workflow`

	frontmatter := map[string]any{
		"on": map[string]any{
			"issues": map[string]any{
				"types": []any{"opened"},
			},
			"roles": []any{"admin", "write"},
		},
		"roles": []any{"admin", "maintainer", "write"},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}
