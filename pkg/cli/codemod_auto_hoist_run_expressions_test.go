//go:build !integration

package cli

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoHoistRunExpressionsCodemod(t *testing.T) {
	codemod := getAutoHoistRunExpressionsCodemod()

	t.Run("hoists github.token from block run to env binding", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: Capture token
    run: |
      echo "GH_TOKEN=${{ github.token }}" >> "$GITHUB_OUTPUT"
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name": "Capture token",
					"run":  "echo \"GH_TOKEN=${{ github.token }}\" >> \"$GITHUB_OUTPUT\"",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "EXPR_GITHUB_TOKEN: ${{ github.token }}", "github.token should be bound in env")
		assert.Contains(t, result, `echo "GH_TOKEN=$EXPR_GITHUB_TOKEN"`, "run should use hoisted env var")
		assert.NotContains(t, result, `echo "GH_TOKEN=${{ github.token }}"`, "inline expression should be removed from run block")
	})

	t.Run("hoists github.repository from inline run to env binding", func(t *testing.T) {
		content := `---
on: push
steps:
  - run: echo ${{ github.repository }}
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"run": "echo ${{ github.repository }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "EXPR_GITHUB_REPOSITORY: ${{ github.repository }}", "github.repository should be bound in env")
		assert.Contains(t, result, "run: echo $EXPR_GITHUB_REPOSITORY", "run should use hoisted env var")
	})

	t.Run("hoists inputs expression from run to env binding", func(t *testing.T) {
		content := `---
on: push
steps:
  - run: echo ${{ inputs.my-input }}
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"run": "echo ${{ inputs.my-input }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "EXPR_INPUTS_MY_INPUT: ${{ inputs.my-input }}", "inputs.my-input should be bound in env")
		assert.Contains(t, result, "run: echo $EXPR_INPUTS_MY_INPUT", "run should use hoisted env var")
	})

	t.Run("hoists steps output expression from run to env binding", func(t *testing.T) {
		content := `---
on: push
steps:
  - run: echo ${{ steps.my-step.outputs.result }}
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"run": "echo ${{ steps.my-step.outputs.result }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "EXPR_STEPS_MY_STEP_OUTPUTS_RESULT: ${{ steps.my-step.outputs.result }}", "steps output should be bound in env")
		assert.Contains(t, result, "run: echo $EXPR_STEPS_MY_STEP_OUTPUTS_RESULT", "run should use hoisted env var")
	})

	t.Run("uses hash-based name for complex expressions", func(t *testing.T) {
		content := `---
on: push
steps:
  - run: echo "${{ github.token || secrets.PAT }}"
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"run": `echo "${{ github.token || secrets.PAT }}"`,
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "${{ github.token || secrets.PAT }}", "complex expression should be preserved in env binding")
		envBindings := regexp.MustCompile(`EXPR_[0-9a-f]{8}:`).FindAllString(result, -1)
		assert.Len(t, envBindings, 1, "one hash-based binding should be created for the complex expression")
	})

	t.Run("deduplicates identical expressions in one step", func(t *testing.T) {
		content := `---
on: push
steps:
  - run: |
      echo "${{ github.token }}"
      echo "${{ github.token }}"
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"run": "echo \"${{ github.token }}\"\necho \"${{ github.token }}\"",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Equal(t, 1, strings.Count(result, "EXPR_GITHUB_TOKEN: ${{ github.token }}"), "binding should appear exactly once")
		assert.Equal(t, 2, strings.Count(result, "$EXPR_GITHUB_TOKEN"), "both occurrences in run should be replaced")
	})

	t.Run("handles multiple distinct expressions in one step", func(t *testing.T) {
		content := `---
on: push
steps:
  - run: |
      echo "${{ github.token }}"
      echo "${{ github.repository }}"
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"run": "echo \"${{ github.token }}\"\necho \"${{ github.repository }}\"",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "EXPR_GITHUB_TOKEN: ${{ github.token }}", "github.token binding should be added")
		assert.Contains(t, result, "EXPR_GITHUB_REPOSITORY: ${{ github.repository }}", "github.repository binding should be added")
		assert.Contains(t, result, `echo "$EXPR_GITHUB_TOKEN"`, "first run line should use hoisted token var")
		assert.Contains(t, result, `echo "$EXPR_GITHUB_REPOSITORY"`, "second run line should use hoisted repo var")
	})

	t.Run("appends missing bindings to existing env block", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: Run checks
    env:
      FOO: bar
    run: echo ${{ github.actor }}
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name": "Run checks",
					"env":  map[string]any{"FOO": "bar"},
					"run":  "echo ${{ github.actor }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "FOO: bar", "existing env entry should be preserved")
		assert.Contains(t, result, "EXPR_GITHUB_ACTOR: ${{ github.actor }}", "new binding should be appended")
		assert.Contains(t, result, "run: echo $EXPR_GITHUB_ACTOR", "run should reference the new env var")
	})

	t.Run("does not duplicate pre-existing EXPR_ bindings", func(t *testing.T) {
		content := `---
on: push
steps:
  - env:
      EXPR_GITHUB_TOKEN: ${{ github.token }}
    run: echo "${{ github.token }}"
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"env": map[string]any{
						"EXPR_GITHUB_TOKEN": "${{ github.token }}",
					},
					"run": `echo "${{ github.token }}"`,
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should still rewrite run expression reference")
		assert.Equal(t, 1, strings.Count(result, "EXPR_GITHUB_TOKEN: ${{ github.token }}"), "existing binding should not be duplicated")
		assert.Contains(t, result, `run: echo "$EXPR_GITHUB_TOKEN"`, "run should use the existing binding")
	})

	t.Run("uses $env:VARNAME for PowerShell steps", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: PS step
    shell: pwsh
    run: |
      Write-Output "${{ github.token }}"
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name":  "PS step",
					"shell": "pwsh",
					"run":   `Write-Output "${{ github.token }}"`,
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "EXPR_GITHUB_TOKEN: ${{ github.token }}", "env binding should be added")
		assert.Contains(t, result, `Write-Output "$env:EXPR_GITHUB_TOKEN"`, "PowerShell step should use $env:VARNAME syntax")
	})

	t.Run("uses $env:VARNAME for powershell shell variant", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: PS step
    shell: powershell
    run: Write-Output ${{ github.actor }}
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name":  "PS step",
					"shell": "powershell",
					"run":   "Write-Output ${{ github.actor }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "EXPR_GITHUB_ACTOR: ${{ github.actor }}", "env binding should be added")
		assert.Contains(t, result, "run: Write-Output $env:EXPR_GITHUB_ACTOR", "powershell step should use $env:VARNAME syntax")
	})

	t.Run("bash step uses $VARNAME not $env:VARNAME", func(t *testing.T) {
		content := `---
on: push
steps:
  - shell: bash
    run: echo ${{ github.actor }}
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"shell": "bash",
					"run":   "echo ${{ github.actor }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "run: echo $EXPR_GITHUB_ACTOR", "bash step should use $VARNAME syntax")
		assert.NotContains(t, result, "$env:EXPR_GITHUB_ACTOR", "bash step should not use $env: syntax")
	})

	t.Run("supports pre-steps section", func(t *testing.T) {
		content := `---
on: pull_request
pre-steps:
  - name: Pre check
    run: echo ${{ github.event.pull_request.number }}
---
`
		frontmatter := map[string]any{
			"on": "pull_request",
			"pre-steps": []any{
				map[string]any{
					"name": "Pre check",
					"run":  "echo ${{ github.event.pull_request.number }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "run: echo $EXPR_GITHUB_EVENT_PULL_REQUEST_NUMBER", "pre-steps run should be rewritten")
		assert.Contains(t, result, "EXPR_GITHUB_EVENT_PULL_REQUEST_NUMBER: ${{ github.event.pull_request.number }}", "binding should be added")
	})

	t.Run("supports post-steps section", func(t *testing.T) {
		content := `---
on: push
post-steps:
  - name: Post
    run: echo ${{ github.run_number }}
---
`
		frontmatter := map[string]any{
			"on": "push",
			"post-steps": []any{
				map[string]any{
					"name": "Post",
					"run":  "echo ${{ github.run_number }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "EXPR_GITHUB_RUN_NUMBER: ${{ github.run_number }}", "post-steps binding should be added")
	})

	t.Run("no-op when no inline run expressions are present", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: Safe
    run: echo "hello"
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name": "Safe",
					"run":  `echo "hello"`,
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not error in no-op case")
		assert.False(t, applied, "codemod should not apply")
		assert.Equal(t, content, result, "content should be unchanged")
	})

	t.Run("no-op when workflow has no steps sections", func(t *testing.T) {
		content := `---
on: push
---
`
		frontmatter := map[string]any{
			"on": "push",
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should not error with no steps sections")
		assert.False(t, applied, "codemod should not apply")
		assert.Equal(t, content, result, "content should be unchanged")
	})

	t.Run("two distinct complex expressions get different hash names", func(t *testing.T) {
		content := `---
on: push
steps:
  - run: echo "${{ github.token || '' }} ${{ github.token || 'fallback' }}"
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"run": `echo "${{ github.token || '' }} ${{ github.token || 'fallback' }}"`,
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		envBindings := regexp.MustCompile(`EXPR_[0-9a-f]{8}:`).FindAllString(result, -1)
		assert.Len(t, envBindings, 2, "two distinct complex expressions should produce two distinct hashed bindings")
		assert.NotEqual(t, envBindings[0], envBindings[1], "hashed names should differ for different expressions")
	})

	t.Run("list-item-inline run key is hoisted", func(t *testing.T) {
		content := `---
on: push
steps:
  - run: echo ${{ github.sha }}
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"run": "echo ${{ github.sha }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "run: echo $EXPR_GITHUB_SHA", "inline run should be rewritten")
		assert.Contains(t, result, "EXPR_GITHUB_SHA: ${{ github.sha }}", "inline run should get env binding")
	})
}
