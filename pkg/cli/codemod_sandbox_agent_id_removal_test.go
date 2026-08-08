//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetSandboxAgentIDRemovalCodemod(t *testing.T) {
	codemod := getSandboxAgentIDRemovalCodemod()

	assert.Equal(t, "sandbox-agent-id-awf-removal", codemod.ID)
	assert.Equal(t, "Remove redundant sandbox.agent.id: awf field", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.28.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestSandboxAgentIDRemoval_RemovesIDAwf(t *testing.T) {
	codemod := getSandboxAgentIDRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent:
    id: awf
    runtime: docker-sbx
    sudo: true
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": map[string]any{
				"id":      "awf",
				"runtime": "docker-sbx",
				"sudo":    true,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "id: awf")
	assert.Contains(t, result, "runtime: docker-sbx")
	assert.Contains(t, result, "sudo: true")
}

func TestSandboxAgentIDRemoval_RemovesIDAwfOnly(t *testing.T) {
	codemod := getSandboxAgentIDRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent:
    id: awf
    version: v0.25.0
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": map[string]any{
				"id":      "awf",
				"version": "v0.25.0",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "id: awf")
	assert.Contains(t, result, "version: v0.25.0")
}

func TestSandboxAgentIDRemoval_NoSandboxField(t *testing.T) {
	codemod := getSandboxAgentIDRemovalCodemod()

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

func TestSandboxAgentIDRemoval_AgentString(t *testing.T) {
	codemod := getSandboxAgentIDRemovalCodemod()

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

func TestSandboxAgentIDRemoval_AgentFalse(t *testing.T) {
	codemod := getSandboxAgentIDRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent: false
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": false,
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestSandboxAgentIDRemoval_NoIDField(t *testing.T) {
	codemod := getSandboxAgentIDRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent:
    runtime: docker-sbx
    sudo: true
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": map[string]any{
				"runtime": "docker-sbx",
				"sudo":    true,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied)
	assert.Equal(t, content, result)
}

func TestSandboxAgentIDRemoval_PreservesMarkdown(t *testing.T) {
	codemod := getSandboxAgentIDRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  agent:
    id: awf
    runtime: docker-sbx
---

# My Workflow

This workflow does things.`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"agent": map[string]any{
				"id":      "awf",
				"runtime": "docker-sbx",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "# My Workflow")
	assert.Contains(t, result, "This workflow does things.")
	assert.NotContains(t, result, "id: awf")
}
