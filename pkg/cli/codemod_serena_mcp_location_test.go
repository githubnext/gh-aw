//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSerenaMCPContainerLocationCodemod(t *testing.T) {
	codemod := getSerenaMCPContainerLocationCodemod()

	t.Run("updates legacy Serena MCP server container and entrypoint", func(t *testing.T) {
		content := `---
mcp-servers:
  serena:
    container: "ghcr.io/github/serena-mcp-server:sha-891c160"
    args:
      - "--network"
      - "host"
    entrypoint: "serena"
    entrypointArgs:
      - "start-mcp-server"
      - "--context"
      - "codex"
      - "--project"
      - ${GITHUB_WORKSPACE}
---

# Test Workflow
`
		frontmatter := map[string]any{
			"mcp-servers": map[string]any{
				"serena": map[string]any{
					"container":  "ghcr.io/github/serena-mcp-server:sha-891c160",
					"entrypoint": "serena",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied for legacy Serena MCP server settings")
		assert.Contains(t, result, "container: \"ghcr.io/oraios/serena:latest\"", "Codemod should update the container location to the project-maintained image")
		assert.Contains(t, result, "entrypoint: \"/workspaces/serena/.venv/bin/serena\"", "Codemod should update the Serena entrypoint to the project-provided venv path")
	})

	t.Run("skips already migrated Serena MCP server settings", func(t *testing.T) {
		content := `---
mcp-servers:
  serena:
    container: "ghcr.io/oraios/serena:1.8.0"
    entrypoint: "/workspaces/serena/.venv/bin/serena"
---

# Test Workflow
`
		frontmatter := map[string]any{
			"mcp-servers": map[string]any{
				"serena": map[string]any{
					"container":  "ghcr.io/oraios/serena:1.8.0",
					"entrypoint": "/workspaces/serena/.venv/bin/serena",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.False(t, applied, "Codemod should not modify already migrated Serena MCP settings")
		assert.Equal(t, content, result, "Content should remain unchanged when the container is already migrated")
	})

	t.Run("migrates legacy path-style Serena entrypoints", func(t *testing.T) {
		content := `---
mcp-servers:
  serena:
    container: "ghcr.io/github/serena-mcp-server:sha-891c160"
    entrypoint: "/usr/local/bin/serena"
---

# Test Workflow
`
		frontmatter := map[string]any{
			"mcp-servers": map[string]any{
				"serena": map[string]any{
					"container":  "ghcr.io/github/serena-mcp-server:sha-891c160",
					"entrypoint": "/usr/local/bin/serena",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied for legacy path-style Serena entrypoints")
		assert.Contains(t, result, "entrypoint: \"/workspaces/serena/.venv/bin/serena\"", "Codemod should rewrite legacy Serena entrypoints to the venv path")
	})

	t.Run("does not rewrite entrypoint when container is not legacy, regardless of field order", func(t *testing.T) {
		content := `---
mcp-servers:
  serena:
    entrypoint: "serena"
    container: "ghcr.io/oraios/serena:1.8.0"
---

# Test Workflow
`
		frontmatter := map[string]any{
			"mcp-servers": map[string]any{
				"serena": map[string]any{
					"entrypoint": "serena",
					"container":  "ghcr.io/oraios/serena:1.8.0",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.False(t, applied, "Codemod should not modify the entrypoint when the container is already the maintained image")
		assert.Equal(t, content, result, "Content should remain unchanged when the container is not legacy")
	})

	t.Run("rewrites entrypoint declared before the legacy container", func(t *testing.T) {
		content := `---
mcp-servers:
  serena:
    entrypoint: "serena"
    container: "ghcr.io/github/serena-mcp-server:sha-891c160"
---

# Test Workflow
`
		frontmatter := map[string]any{
			"mcp-servers": map[string]any{
				"serena": map[string]any{
					"entrypoint": "serena",
					"container":  "ghcr.io/github/serena-mcp-server:sha-891c160",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "Codemod should not return an error")
		assert.True(t, applied, "Codemod should be applied when the legacy container appears after the entrypoint")
		assert.Contains(t, result, "entrypoint: \"/workspaces/serena/.venv/bin/serena\"", "Codemod should rewrite the entrypoint when the same block uses the legacy container")
		assert.Contains(t, result, "container: \"ghcr.io/oraios/serena:latest\"", "Codemod should rewrite the legacy container")
	})
}
