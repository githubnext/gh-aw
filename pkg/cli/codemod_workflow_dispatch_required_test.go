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

func TestTopLevelEnvSecretsGuidedErrorCodemod(t *testing.T) {
	codemod := getTopLevelEnvSecretsGuidedErrorCodemod()

	t.Run("returns guided error when top-level env contains a secret", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
			},
		}

		_, applied, err := codemod.Apply(content, frontmatter)
		require.Error(t, err, "should return an error for top-level env secrets")
		assert.False(t, applied, "should not modify the file")
		assert.Contains(t, err.Error(), "top-level env: contains secrets")
		assert.Contains(t, err.Error(), "${{ secrets.GITHUB_TOKEN }}")
		assert.Contains(t, err.Error(), "Manual fix required")
		assert.Contains(t, err.Error(), "https://github.github.com/gh-aw/reference/engines/")
	})

	t.Run("returns guided error with multiple secret references", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  PAT: ${{ secrets.GITHUB_PERSONAL_ACCESS_TOKEN || secrets.GITHUB_TOKEN }}
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"PAT": "${{ secrets.GITHUB_PERSONAL_ACCESS_TOKEN || secrets.GITHUB_TOKEN }}",
			},
		}

		_, applied, err := codemod.Apply(content, frontmatter)
		require.Error(t, err)
		assert.False(t, applied)
		assert.Contains(t, err.Error(), "top-level env: contains secrets")
	})

	t.Run("no-op when top-level env has no secrets", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  LOG_LEVEL: debug
  NODE_ENV: production
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"LOG_LEVEL": "debug",
				"NODE_ENV":  "production",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "should not error when env has no secrets")
		assert.False(t, applied, "should not apply when no secrets in env")
		assert.Equal(t, content, result)
	})

	t.Run("no-op when there is no top-level env section", func(t *testing.T) {
		content := `---
on: workflow_dispatch
engine:
  id: copilot
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"engine": map[string]any{
				"id": "copilot",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("no-op when env only uses vars not secrets", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  API_URL: ${{ vars.API_URL }}
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"API_URL": "${{ vars.API_URL }}",
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "vars references are not secrets")
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("does not modify content even when secret found", func(t *testing.T) {
		content := `---
on: workflow_dispatch
env:
  TOKEN: ${{ secrets.MY_TOKEN }}
---

# Agent
`
		frontmatter := map[string]any{
			"on": "workflow_dispatch",
			"env": map[string]any{
				"TOKEN": "${{ secrets.MY_TOKEN }}",
			},
		}

		result, _, _ := codemod.Apply(content, frontmatter)
		assert.Equal(t, content, result, "content must remain unchanged (guided error, not auto-fix)")
	})
}
