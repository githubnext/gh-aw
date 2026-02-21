//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAssignToAgentDefaultAgentCodemod(t *testing.T) {
	codemod := getAssignToAgentDefaultAgentCodemod()

	assert.Equal(t, "assign-to-agent-default-agent-to-name", codemod.ID)
	assert.Equal(t, "Migrate assign-to-agent default-agent to name", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.NotEmpty(t, codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestAssignToAgentCodemod_BasicMigration(t *testing.T) {
	codemod := getAssignToAgentDefaultAgentCodemod()

	content := `---
on: issues
engine: copilot
safe-outputs:
  assign-to-agent:
    default-agent: copilot
---

# Test Workflow`

	frontmatter := map[string]any{
		"on":     "issues",
		"engine": "copilot",
		"safe-outputs": map[string]any{
			"assign-to-agent": map[string]any{
				"default-agent": "copilot",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "name: copilot")
	assert.NotContains(t, result, "default-agent:")
}

func TestAssignToAgentCodemod_PreservesIndentation(t *testing.T) {
	codemod := getAssignToAgentDefaultAgentCodemod()

	content := `---
on: issues
safe-outputs:
  assign-to-agent:
    default-agent: copilot
    max: 2
---

# Test`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"assign-to-agent": map[string]any{
				"default-agent": "copilot",
				"max":           2,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "    name: copilot")
	assert.Contains(t, result, "    max: 2")
	assert.NotContains(t, result, "default-agent:")
}

func TestAssignToAgentCodemod_PreservesComment(t *testing.T) {
	codemod := getAssignToAgentDefaultAgentCodemod()

	content := `---
on: issues
safe-outputs:
  assign-to-agent:
    default-agent: copilot  # the default agent
---

# Test`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"assign-to-agent": map[string]any{
				"default-agent": "copilot",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "name: copilot  # the default agent")
}

func TestAssignToAgentCodemod_NoSafeOutputs(t *testing.T) {
	codemod := getAssignToAgentDefaultAgentCodemod()

	content := `---
on: issues
engine: copilot
---

# Test`

	frontmatter := map[string]any{
		"on":     "issues",
		"engine": "copilot",
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestAssignToAgentCodemod_NoAssignToAgent(t *testing.T) {
	codemod := getAssignToAgentDefaultAgentCodemod()

	content := `---
on: issues
safe-outputs:
  create-issue:
    title: Bug
---

# Test`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"title": "Bug",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestAssignToAgentCodemod_NoDefaultAgent(t *testing.T) {
	codemod := getAssignToAgentDefaultAgentCodemod()

	content := `---
on: issues
safe-outputs:
  assign-to-agent:
    name: copilot
---

# Test`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"assign-to-agent": map[string]any{
				"name": "copilot",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestAssignToAgentCodemod_SkipsWhenNameAlreadyExists(t *testing.T) {
	codemod := getAssignToAgentDefaultAgentCodemod()

	content := `---
on: issues
safe-outputs:
  assign-to-agent:
    name: copilot
    default-agent: other-agent
---

# Test`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"assign-to-agent": map[string]any{
				"name":          "copilot",
				"default-agent": "other-agent",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "Should not apply when 'name' already exists")
	assert.Equal(t, content, result)
}

func TestAssignToAgentCodemod_PreservesOtherSafeOutputs(t *testing.T) {
	codemod := getAssignToAgentDefaultAgentCodemod()

	content := `---
on: issues
safe-outputs:
  create-issue:
    title: Bug
  assign-to-agent:
    default-agent: copilot
---

# Test`

	frontmatter := map[string]any{
		"safe-outputs": map[string]any{
			"create-issue": map[string]any{
				"title": "Bug",
			},
			"assign-to-agent": map[string]any{
				"default-agent": "copilot",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "create-issue:")
	assert.Contains(t, result, "name: copilot")
	assert.NotContains(t, result, "default-agent:")
}

func TestAssignToAgentCodemod_RegisteredInAllCodemods(t *testing.T) {
	codemods := GetAllCodemods()
	var found bool
	for _, c := range codemods {
		if c.ID == "assign-to-agent-default-agent-to-name" {
			found = true
			break
		}
	}
	assert.True(t, found, "assign-to-agent-default-agent-to-name codemod should be registered in GetAllCodemods")
}
