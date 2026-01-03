package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBeadsToolConfig(t *testing.T) {
	t.Run("empty object enables beads with defaults", func(t *testing.T) {
		toolsMap := map[string]any{
			"beads": map[string]any{},
		}

		tools := NewTools(toolsMap)
		require.NotNil(t, tools.Beads, "Beads tool should be configured")
		assert.Empty(t, tools.Beads.Version, "Version should be empty (use latest)")
		assert.Empty(t, tools.Beads.Commands, "Commands should be empty (allow all)")
		assert.False(t, tools.Beads.ReadOnly, "ReadOnly should be false")
	})

	t.Run("nil value enables beads with defaults", func(t *testing.T) {
		toolsMap := map[string]any{
			"beads": nil,
		}

		tools := NewTools(toolsMap)
		require.NotNil(t, tools.Beads, "Beads tool should be configured")
		assert.Empty(t, tools.Beads.Version, "Version should be empty (use latest)")
		assert.Empty(t, tools.Beads.Commands, "Commands should be empty (allow all)")
		assert.False(t, tools.Beads.ReadOnly, "ReadOnly should be false")
	})

	t.Run("version string", func(t *testing.T) {
		toolsMap := map[string]any{
			"beads": "v1.0.0",
		}

		tools := NewTools(toolsMap)
		require.NotNil(t, tools.Beads, "Beads tool should be configured")
		assert.Equal(t, "v1.0.0", tools.Beads.Version, "Version should be set")
		assert.Empty(t, tools.Beads.Commands, "Commands should be empty (allow all)")
		assert.False(t, tools.Beads.ReadOnly, "ReadOnly should be false")
	})

	t.Run("full configuration with commands", func(t *testing.T) {
		toolsMap := map[string]any{
			"beads": map[string]any{
				"version": "v1.2.3",
				"commands": []any{
					"bd ready",
					"bd create",
					"bd status",
					"bd close",
				},
			},
		}

		tools := NewTools(toolsMap)
		require.NotNil(t, tools.Beads, "Beads tool should be configured")
		assert.Equal(t, "v1.2.3", tools.Beads.Version, "Version should be set")
		assert.Len(t, tools.Beads.Commands, 4, "Should have 4 allowed commands")
		assert.Contains(t, tools.Beads.Commands, "bd ready")
		assert.Contains(t, tools.Beads.Commands, "bd create")
		assert.Contains(t, tools.Beads.Commands, "bd status")
		assert.Contains(t, tools.Beads.Commands, "bd close")
		assert.False(t, tools.Beads.ReadOnly, "ReadOnly should be false")
	})

	t.Run("read-only configuration", func(t *testing.T) {
		toolsMap := map[string]any{
			"beads": map[string]any{
				"read-only": true,
				"commands": []any{
					"bd ready",
					"bd status",
					"bd list",
				},
			},
		}

		tools := NewTools(toolsMap)
		require.NotNil(t, tools.Beads, "Beads tool should be configured")
		assert.True(t, tools.Beads.ReadOnly, "ReadOnly should be true")
		assert.Len(t, tools.Beads.Commands, 3, "Should have 3 allowed commands")
	})
}

func TestBeadsToolHasTool(t *testing.T) {
	toolsMap := map[string]any{
		"beads": map[string]any{
			"version": "v1.0.0",
		},
	}

	tools := NewTools(toolsMap)
	assert.True(t, tools.HasTool("beads"), "Should have beads tool")
	assert.False(t, tools.HasTool("nonexistent"), "Should not have nonexistent tool")
}

func TestBeadsToolGetToolNames(t *testing.T) {
	toolsMap := map[string]any{
		"beads": map[string]any{},
		"bash":  []any{"echo"},
	}

	tools := NewTools(toolsMap)
	names := tools.GetToolNames()
	assert.Contains(t, names, "beads", "Tool names should include beads")
	assert.Contains(t, names, "bash", "Tool names should include bash")
}

func TestBeadsToolToMap(t *testing.T) {
	t.Run("converts beads config back to map", func(t *testing.T) {
		toolsMap := map[string]any{
			"beads": map[string]any{
				"version": "v1.0.0",
				"commands": []any{
					"bd ready",
					"bd create",
				},
				"read-only": true,
			},
		}

		tools := NewTools(toolsMap)
		resultMap := tools.ToMap()

		beads, ok := resultMap["beads"]
		require.True(t, ok, "Result map should contain beads")
		require.NotNil(t, beads, "Beads should not be nil")

		// The ToMap function returns the raw map since it was preserved
		beadsMap, ok := beads.(map[string]any)
		require.True(t, ok, "Beads should be map[string]any from raw map")
		assert.Equal(t, "v1.0.0", beadsMap["version"])

		commands, ok := beadsMap["commands"].([]any)
		require.True(t, ok, "Commands should be []any")
		assert.Len(t, commands, 2)
		assert.True(t, beadsMap["read-only"].(bool))
	})
}

func TestBeadsToolWithOtherTools(t *testing.T) {
	t.Run("beads works alongside other tools", func(t *testing.T) {
		toolsMap := map[string]any{
			"beads": map[string]any{
				"version": "latest",
			},
			"bash": []any{
				"echo",
				"git status",
			},
			"github": map[string]any{
				"allowed": []any{"issue_read", "pr_read"},
			},
		}

		tools := NewTools(toolsMap)
		require.NotNil(t, tools.Beads, "Should have beads tool")
		require.NotNil(t, tools.Bash, "Should have bash tool")
		require.NotNil(t, tools.GitHub, "Should have github tool")

		assert.Equal(t, "latest", tools.Beads.Version)
		assert.Len(t, tools.Bash.AllowedCommands, 2)
		assert.Len(t, tools.GitHub.Allowed, 2)
	})
}
