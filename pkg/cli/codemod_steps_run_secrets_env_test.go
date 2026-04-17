//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepsRunSecretsToEnvCodemod(t *testing.T) {
	codemod := getStepsRunSecretsToEnvCodemod()

	t.Run("moves inline run secret to env binding", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: Clone runtime
    run: git clone https://x:${{ secrets.RUNTIME_TRIAGE_TOKEN }}@github.com/org/repo.git
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name": "Clone runtime",
					"run":  "git clone https://x:${{ secrets.RUNTIME_TRIAGE_TOKEN }}@github.com/org/repo.git",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "run: git clone https://x:$RUNTIME_TRIAGE_TOKEN@github.com/org/repo.git", "run should use env var")
		assert.Contains(t, result, "env:", "step env block should be added")
		assert.Contains(t, result, "RUNTIME_TRIAGE_TOKEN: ${{ secrets.RUNTIME_TRIAGE_TOKEN }}", "secret should be bound in env")
	})

	t.Run("appends missing binding to existing env block", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: Run checks
    env:
      FOO: bar
    run: echo ${{ secrets.TEST_TOKEN }}
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name": "Run checks",
					"env": map[string]any{
						"FOO": "bar",
					},
					"run": "echo ${{ secrets.TEST_TOKEN }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "FOO: bar", "existing env entries should be preserved")
		assert.Contains(t, result, "TEST_TOKEN: ${{ secrets.TEST_TOKEN }}", "missing env binding should be added")
		assert.Contains(t, result, "run: echo $TEST_TOKEN", "run should use env var")
	})

	t.Run("supports pre-steps section", func(t *testing.T) {
		content := `---
on: pull_request
pre-steps:
  - name: Pre check
    run: npm config set //registry.npmjs.org/:_authToken=${{ secrets.NPM_TOKEN }}
---
`
		frontmatter := map[string]any{
			"on": "pull_request",
			"pre-steps": []any{
				map[string]any{
					"name": "Pre check",
					"run":  "npm config set //registry.npmjs.org/:_authToken=${{ secrets.NPM_TOKEN }}",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err, "codemod should apply cleanly")
		assert.True(t, applied, "codemod should apply")
		assert.Contains(t, result, "_authToken=$NPM_TOKEN", "secret should be replaced with shell env reference")
		assert.Contains(t, result, "NPM_TOKEN: ${{ secrets.NPM_TOKEN }}", "env binding should be added")
	})

	t.Run("no-op when no inline run secrets are present", func(t *testing.T) {
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
}
