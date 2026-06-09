//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowDispatchRequiredFalseCodemod(t *testing.T) {
	codemod := getWorkflowDispatchRequiredFalseCodemod()

	t.Run("rewrites required: true to required: false for slash_command trigger", func(t *testing.T) {
		content := `---
on:
  slash_command:
    name: evaluate-tests
    strategy: centralized
  workflow_dispatch:
    inputs:
      pr_number:
        description: "PR number to evaluate"
        required: true
        type: number
tools:
  github:
    allowed: [list_issues]
---

# Evaluate Tests
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"slash_command": map[string]any{
					"name":     "evaluate-tests",
					"strategy": "centralized",
				},
				"workflow_dispatch": map[string]any{
					"inputs": map[string]any{
						"pr_number": map[string]any{
							"description": "PR number to evaluate",
							"required":    true,
							"type":        "number",
						},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "required: false")
		assert.NotContains(t, result, "required: true")
	})

	t.Run("rewrites required: true to required: false for label_command trigger", func(t *testing.T) {
		content := `---
on:
  label_command:
    name: triage
  workflow_dispatch:
    inputs:
      issue_number:
        description: "Issue number"
        required: true
        type: number
tools:
  github:
    allowed: [list_issues]
---

# Triage
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"label_command": map[string]any{
					"name": "triage",
				},
				"workflow_dispatch": map[string]any{
					"inputs": map[string]any{
						"issue_number": map[string]any{
							"description": "Issue number",
							"required":    true,
							"type":        "number",
						},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "required: false")
		assert.NotContains(t, result, "required: true")
	})

	t.Run("rewrites multiple required: true inputs", func(t *testing.T) {
		content := `---
on:
  slash_command:
    name: run
  workflow_dispatch:
    inputs:
      pr_number:
        description: "PR number"
        required: true
        type: number
      branch:
        description: "Branch name"
        required: true
        type: string
---

# Run
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"slash_command": map[string]any{"name": "run"},
				"workflow_dispatch": map[string]any{
					"inputs": map[string]any{
						"pr_number": map[string]any{
							"description": "PR number",
							"required":    true,
							"type":        "number",
						},
						"branch": map[string]any{
							"description": "Branch name",
							"required":    true,
							"type":        "string",
						},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.NotContains(t, result, "required: true")
	})

	t.Run("rewrites only required: true inputs and preserves required: false", func(t *testing.T) {
		content := `---
on:
  slash_command:
    name: run
  workflow_dispatch:
    inputs:
      pr_number:
        description: "PR number"
        required: true
        type: number
      dry_run:
        description: "Dry run flag"
        required: false
        type: boolean
---

# Run
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"slash_command": map[string]any{"name": "run"},
				"workflow_dispatch": map[string]any{
					"inputs": map[string]any{
						"pr_number": map[string]any{"required": true},
						"dry_run":   map[string]any{"required": false},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "pr_number:")
		assert.Contains(t, result, "required: false")
		assert.NotContains(t, result, "required: true")
	})

	t.Run("rewrites when inputs line has trailing comment", func(t *testing.T) {
		content := `---
on:
  slash_command:
    name: run
  workflow_dispatch:
    inputs: # workflow inputs
      pr_number:
        description: "PR number"
        required: true
        type: number
---

# Run
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"slash_command": map[string]any{"name": "run"},
				"workflow_dispatch": map[string]any{
					"inputs": map[string]any{
						"pr_number": map[string]any{"required": true},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "inputs: # workflow inputs")
		assert.Contains(t, result, "required: false")
		assert.NotContains(t, result, "required: true")
	})

	t.Run("rewrites inline inputs mapping", func(t *testing.T) {
		content := `---
on:
  slash_command:
    name: run
  workflow_dispatch:
    inputs: { pr_number: { required: true, type: number }, dry_run: { required: false, type: boolean } }
---

# Run
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"slash_command": map[string]any{"name": "run"},
				"workflow_dispatch": map[string]any{
					"inputs": map[string]any{
						"pr_number": map[string]any{"required": true, "type": "number"},
						"dry_run":   map[string]any{"required": false, "type": "boolean"},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "required: false, type: number")
		assert.NotContains(t, result, "required: true")
	})

	t.Run("no-op when required is already false", func(t *testing.T) {
		content := `---
on:
  slash_command:
    name: scout
  workflow_dispatch:
    inputs:
      topic:
        description: "Topic"
        required: false
        type: string
---

# Scout
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"slash_command": map[string]any{"name": "scout"},
				"workflow_dispatch": map[string]any{
					"inputs": map[string]any{
						"topic": map[string]any{
							"description": "Topic",
							"required":    false,
							"type":        "string",
						},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("no-op when no slash_command or label_command trigger", func(t *testing.T) {
		content := `---
on:
  workflow_dispatch:
    inputs:
      topic:
        description: "Topic"
        required: true
        type: string
---

# Manual
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"workflow_dispatch": map[string]any{
					"inputs": map[string]any{
						"topic": map[string]any{
							"description": "Topic",
							"required":    true,
							"type":        "string",
						},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied, "should not apply when no slash/label command trigger")
		assert.Equal(t, content, result)
	})

	t.Run("no-op when no workflow_dispatch trigger", func(t *testing.T) {
		content := `---
on:
  slash_command:
    name: scout
---

# Scout
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"slash_command": map[string]any{"name": "scout"},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("no-op when workflow_dispatch has no inputs", func(t *testing.T) {
		content := `---
on:
  slash_command:
    name: scout
  workflow_dispatch:
---

# Scout
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"slash_command":     map[string]any{"name": "scout"},
				"workflow_dispatch": map[string]any{},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("preserves other input fields unchanged", func(t *testing.T) {
		content := `---
on:
  slash_command:
    name: scout
  workflow_dispatch:
    inputs:
      pr_number:
        description: "PR number"
        required: true
        type: number
        default: "0"
---

# Scout
`
		frontmatter := map[string]any{
			"on": map[string]any{
				"slash_command": map[string]any{"name": "scout"},
				"workflow_dispatch": map[string]any{
					"inputs": map[string]any{
						"pr_number": map[string]any{
							"description": "PR number",
							"required":    true,
							"type":        "number",
							"default":     "0",
						},
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, `description: "PR number"`)
		assert.Contains(t, result, "type: number")
		assert.Contains(t, result, `default: "0"`)
		assert.Contains(t, result, "required: false")
		assert.NotContains(t, result, "required: true")
	})
}
