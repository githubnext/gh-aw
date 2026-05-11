//go:build !integration

package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckoutPersistCredentialsFalseCodemod(t *testing.T) {
	codemod := getCheckoutPersistCredentialsFalseCodemod()
	assert.Equal(t, "1.0.44", codemod.IntroducedIn)

	t.Run("adds with block when checkout step has none", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: Checkout repository
    uses: actions/checkout@v5
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name": "Checkout repository",
					"uses": "actions/checkout@v5",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "uses: actions/checkout@v5\n    with:\n      persist-credentials: false")
	})

	t.Run("adds persist-credentials under existing with block", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: Checkout repository
    uses: actions/checkout@v5
    with:
      fetch-depth: 0
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name": "Checkout repository",
					"uses": "actions/checkout@v5",
					"with": map[string]any{
						"fetch-depth": 0,
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "fetch-depth: 0\n      persist-credentials: false")
	})

	t.Run("does not mutate explicit persist-credentials true", func(t *testing.T) {
		content := `---
on: push
steps:
  - name: Checkout repository
    uses: actions/checkout@v5
    with:
      persist-credentials: true
---
`
		frontmatter := map[string]any{
			"on": "push",
			"steps": []any{
				map[string]any{
					"name": "Checkout repository",
					"uses": "actions/checkout@v5",
					"with": map[string]any{
						"persist-credentials": true,
					},
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, result)
	})

	t.Run("supports pre-steps and post-steps sections", func(t *testing.T) {
		content := `---
on: pull_request
pre-steps:
  - uses: actions/checkout@v5
post-steps:
  - name: Checkout repo post
    uses: actions/checkout@v5
---
`
		frontmatter := map[string]any{
			"on": "pull_request",
			"pre-steps": []any{
				map[string]any{"uses": "actions/checkout@v5"},
			},
			"post-steps": []any{
				map[string]any{
					"name": "Checkout repo post",
					"uses": "actions/checkout@v5",
				},
			},
		}

		result, applied, err := codemod.Apply(content, frontmatter)
		require.NoError(t, err)
		assert.True(t, applied)
		assert.Contains(t, result, "pre-steps:\n  - uses: actions/checkout@v5\n    with:\n      persist-credentials: false")
		assert.Contains(t, result, "uses: actions/checkout@v5\n    with:\n      persist-credentials: false")
	})
}
