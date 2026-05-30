//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- getSandboxMCPContainerRemovalCodemod tests -----

func TestGetSandboxMCPContainerRemovalCodemod(t *testing.T) {
	codemod := getSandboxMCPContainerRemovalCodemod()

	assert.Equal(t, "sandbox-mcp-container-removal", codemod.ID)
	assert.Equal(t, "Remove deprecated sandbox.mcp.container field", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.26.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestSandboxMCPContainerRemoval_RemovesContainer(t *testing.T) {
	codemod := getSandboxMCPContainerRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  mcp:
    container: ghcr.io/example/gateway
    port: 8080
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"container": "ghcr.io/example/gateway",
				"port":      8080,
			},
		},
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "container:")
	assert.Contains(t, result, "mcp:")
	assert.Contains(t, result, "port: 8080")
}

func TestSandboxMCPContainerRemoval_NoSandboxKey(t *testing.T) {
	codemod := getSandboxMCPContainerRemovalCodemod()

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

func TestSandboxMCPContainerRemoval_NoMCPKey(t *testing.T) {
	codemod := getSandboxMCPContainerRemovalCodemod()

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

func TestSandboxMCPContainerRemoval_NoContainerField(t *testing.T) {
	codemod := getSandboxMCPContainerRemovalCodemod()

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

func TestSandboxMCPContainerRemoval_PreservesMarkdown(t *testing.T) {
	codemod := getSandboxMCPContainerRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  mcp:
    container: github/gh-aw-mcpg
---

# Workflow Title

This is a test workflow.`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"container": "github/gh-aw-mcpg",
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.Contains(t, result, "# Workflow Title")
	assert.Contains(t, result, "This is a test workflow.")
	assert.NotContains(t, result, "container:")
}

// ----- getSandboxMCPVersionRemovalCodemod tests -----

func TestGetSandboxMCPVersionRemovalCodemod(t *testing.T) {
	codemod := getSandboxMCPVersionRemovalCodemod()

	assert.Equal(t, "sandbox-mcp-version-removal", codemod.ID)
	assert.Equal(t, "Remove deprecated sandbox.mcp.version field", codemod.Name)
	assert.NotEmpty(t, codemod.Description)
	assert.Equal(t, "0.26.0", codemod.IntroducedIn)
	require.NotNil(t, codemod.Apply)
}

func TestSandboxMCPVersionRemoval_RemovesVersion(t *testing.T) {
	codemod := getSandboxMCPVersionRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  mcp:
    version: v1.0.0
    port: 8080
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"version": "v1.0.0",
				"port":    8080,
			},
		},
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "version: v1.0.0")
	assert.Contains(t, result, "mcp:")
	assert.Contains(t, result, "port: 8080")
}

func TestSandboxMCPVersionRemoval_NoVersionField(t *testing.T) {
	codemod := getSandboxMCPVersionRemovalCodemod()

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

func TestSandboxMCPVersionRemoval_NoSandboxKey(t *testing.T) {
	codemod := getSandboxMCPVersionRemovalCodemod()

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

func TestSandboxMCPContainerRemoval_SkipsWhenStrictFalse(t *testing.T) {
	codemod := getSandboxMCPContainerRemovalCodemod()

	content := `---
on: workflow_dispatch
strict: false
sandbox:
  mcp:
    container: github/gh-aw-mcpg
    port: 8080
---

# Test`

	frontmatter := map[string]any{
		"on":     "workflow_dispatch",
		"strict": false,
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"container": "github/gh-aw-mcpg",
				"port":      8080,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "should not apply when strict: false is set")
	assert.Equal(t, content, result)
}

func TestSandboxMCPVersionRemoval_SkipsWhenStrictFalse(t *testing.T) {
	codemod := getSandboxMCPVersionRemovalCodemod()

	content := `---
on: workflow_dispatch
strict: false
sandbox:
  mcp:
    version: v1.0.0
    port: 8080
---

# Test`

	frontmatter := map[string]any{
		"on":     "workflow_dispatch",
		"strict": false,
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"version": "v1.0.0",
				"port":    8080,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.False(t, applied, "should not apply when strict: false is set")
	assert.Equal(t, content, result)
}

func TestSandboxMCPVersionRemoval_BothContainerAndVersion(t *testing.T) {
	// Verify that version removal does not affect the container key.
	codemod := getSandboxMCPVersionRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  mcp:
    container: github/gh-aw-mcpg
    version: v0.0.12
    port: 8080
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"container": "github/gh-aw-mcpg",
				"version":   "v0.0.12",
				"port":      8080,
			},
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "version: v0.0.12")
	assert.Contains(t, result, "container: github/gh-aw-mcpg")
	assert.Contains(t, result, "port: 8080")
}

func TestSandboxMCPContainerRemoval_RemovesEmptySandboxGrandparent(t *testing.T) {
	// When container: is the only field under mcp:, and mcp: is the only field
	// under sandbox:, the codemod must remove all three levels to avoid a
	// dangling "sandbox:" key that YAML parses as null.
	codemod := getSandboxMCPContainerRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  mcp:
    container: ghcr.io/example/gateway
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"container": "ghcr.io/example/gateway",
			},
		},
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "container:", "container field should be removed")
	assert.NotContains(t, result, "mcp:", "empty mcp block should be removed")
	assert.NotContains(t, result, "sandbox:", "empty sandbox block should be removed")
	assert.Contains(t, result, "permissions:", "permissions block should be preserved")
}

func TestSandboxMCPVersionRemoval_RemovesEmptySandboxGrandparent(t *testing.T) {
	// When version: is the only field under mcp:, and mcp: is the only field
	// under sandbox:, the codemod must remove all three levels to avoid a
	// dangling "sandbox:" key that YAML parses as null.
	codemod := getSandboxMCPVersionRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  mcp:
    version: v0.1.0
permissions:
  contents: read
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"version": "v0.1.0",
			},
		},
		"permissions": map[string]any{
			"contents": "read",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "version:", "version field should be removed")
	assert.NotContains(t, result, "mcp:", "empty mcp block should be removed")
	assert.NotContains(t, result, "sandbox:", "empty sandbox block should be removed")
	assert.Contains(t, result, "permissions:", "permissions block should be preserved")
}

func TestSandboxMCPContainerRemoval_KeepsSandboxWhenOtherChildrenRemain(t *testing.T) {
	// When sandbox: has other children besides mcp:, it must be preserved even
	// after mcp: becomes empty and is removed.
	codemod := getSandboxMCPContainerRemovalCodemod()

	content := `---
on: workflow_dispatch
sandbox:
  mcp:
    container: ghcr.io/example/gateway
  agent: awf
---

# Test`

	frontmatter := map[string]any{
		"on": "workflow_dispatch",
		"sandbox": map[string]any{
			"mcp": map[string]any{
				"container": "ghcr.io/example/gateway",
			},
			"agent": "awf",
		},
	}

	result, applied, err := codemod.Apply(content, frontmatter)

	require.NoError(t, err)
	assert.True(t, applied)
	assert.NotContains(t, result, "container:", "container field should be removed")
	assert.NotContains(t, result, "mcp:", "empty mcp block should be removed")
	assert.Contains(t, result, "sandbox:", "sandbox block should be kept (still has agent:)")
	assert.Contains(t, result, "agent: awf", "other sandbox children should be preserved")
}
