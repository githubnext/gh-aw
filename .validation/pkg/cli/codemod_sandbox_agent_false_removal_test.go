//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSandboxAgentFalseRemovalCodemod(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	assert.Equal(t, "sandbox-agent-false-removal", codemod.ID)
	assert.Equal(t, "Remove deprecated sandbox.agent: false field", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.26.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestSandboxAgentFalseRemoval_RemovesAgentFalse(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent: false
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": false,
		},
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "agent: false")
	// Empty sandbox block must be removed to avoid "got null, want object" compile error
	assert.NotContains(t, result, "sandbox:")
}

func TestSandboxAgentFalseRemoval_PreservesOtherSandboxKeys(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent: false
  mcp:
    port: 8080
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": false,
			"mcp": map[string]any{
				"port": 8080,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "agent: false")
	assert.Contains(t, result, "mcp:")
	assert.Contains(t, result, "port: 8080")
}

func TestSandboxAgentFalseRemoval_NoSandboxKey(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	content := `---
on: workflow_dispatch
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestSandboxAgentFalseRemoval_AgentNotFalse(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent: awf
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": "awf",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestSandboxAgentFalseRemoval_AgentObject(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent:
    id: awf
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": map[string]any{
				"id": "awf",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestSandboxAgentFalseRemoval_AgentTrue(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent: true
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": true,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestSandboxAgentFalseRemoval_SkipsWhenStrictFalse(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	content := `---
on: workflow_dispatch
strict: false
sandbox:
  agent: false
---

# Test`

	frontmatter := map[string]any{
		"on":     "workflow_dispatch",
		"strict": false,
		"sandbox": map[string]any{
			"agent": false,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "should not apply when strict: false is set")
	assert.Equal(t, content, result)
}

func TestSandboxAgentFalseRemoval_PreservesMarkdown(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent: false
---

# Workflow Title

This workflow was using the nosandbox escape hatch.`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": false,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "# Workflow Title")
	assert.NotContains(t, result, "agent: false")
}

func TestSandboxAgentFalseRemoval_NoAgentKey(t *testing.T) {
	codemod := getSandboxAgentFalseRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  mcp:
    port: 8080
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"port": 8080,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}
