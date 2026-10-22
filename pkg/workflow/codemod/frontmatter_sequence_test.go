package codemod

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeSequence(t *testing.T) {
	for _, source := range []struct {
		name string
		yaml string
	}{
		{"flow", "bots: [dependabot, renovate, renovate, other]\n"},
		{"block", "bots:\n  - dependabot\n  - renovate\n  - renovate\n  - other\n"},
	} {
		for _, destination := range []struct {
			name string
			yaml string
		}{
			{"flow", "on:\n  bots: [dependabot]\n"},
			{"block", "on:\n  bots:\n    - dependabot\n"},
		} {
			t.Run(source.name+" to "+destination.name, func(t *testing.T) {
				body := "\n# Body  \nBots: [untouched]\n"
				content := "---\n" + source.yaml + destination.yaml + "---\n" + body
				updated, applied, err := Apply(content, func(f *Frontmatter) error {
					return f.MergeSequence([]string{"bots"}, []string{"on", "bots"})
				})
				require.NoError(t, err)
				assert.True(t, applied)
				document, err := aw.Parse(updated)
				require.NoError(t, err)
				assert.NotContains(t, document.Frontmatter, "bots")
				assert.Equal(t, []any{"dependabot", "renovate", "other"}, document.Frontmatter["on"].(map[string]any)["bots"])
				assert.Equal(t, body, document.Markdown)
			})
		}
	}
}

func TestMergeSequenceNoOp(t *testing.T) {
	for _, content := range []string{
		"---\nbots: [dependabot, renovate] # keep\n---\n# Body\n",
		"---\nlabels: &labels [renovate]\nbots: *labels\n---\n# Body\n",
	} {
		updated, applied, err := Apply(content, func(f *Frontmatter) error {
			return f.MergeSequence([]string{"bots"}, []string{"bots"})
		})
		require.NoError(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, updated)
	}
}

func TestMergeSequenceOnlyDuplicates(t *testing.T) {
	destination := "on:\n  bots: ['dependabot', renovate] # authoritative\n"
	content := "---\nbots: [renovate, dependabot, renovate]\n" + destination + "---\n# Body\n"
	updated, applied, err := Apply(content, func(f *Frontmatter) error {
		return f.MergeSequence([]string{"bots"}, []string{"on", "bots"})
	})
	require.NoError(t, err)
	assert.True(t, applied)
	assert.Equal(t, "---\n"+destination+"---\n# Body\n", updated)
}

func TestMergeSequenceBlockComments(t *testing.T) {
	for _, source := range []string{
		"  - dependabot # duplicate\n  # Unique bot\n  # Keep this explanation\n  - renovate # source inline\n  # Another unique bot\n  - other\n",
		"  # Unique bot\n  # Keep this explanation\n  - renovate # source inline\n  - dependabot # duplicate\n  # Another unique bot\n  - other\n",
	} {
		content := "---\nbots:\n" + source + "on:\n  bots:\n    # Destination explanation\n    - dependabot # destination inline\n---\n\n# Body\n"
		updated, applied, err := Apply(content, func(f *Frontmatter) error {
			return f.MergeSequence([]string{"bots"}, []string{"on", "bots"})
		})
		require.NoError(t, err)
		assert.True(t, applied)
		for _, comment := range []string{"# Unique bot", "# Keep this explanation", "# source inline", "# Another unique bot", "# Destination explanation", "# destination inline"} {
			assert.Equal(t, 1, strings.Count(updated, comment), "comment should be preserved once: %s", comment)
		}
		document, err := aw.Parse(updated)
		require.NoError(t, err)
		assert.Equal(t, []any{"dependabot", "renovate", "other"}, document.Frontmatter["on"].(map[string]any)["bots"])
		assert.Equal(t, "\n# Body\n", document.Markdown)
	}
}

func TestMergeSequenceAliases(t *testing.T) {
	for _, source := range []string{
		"labels: &labels [renovate]\nbots: *labels\non:\n  bots:\n    - dependabot\n",
		"bots: &labels\n  - renovate\nlabels: *labels\non:\n  bots:\n    - dependabot\n",
		"labels: &labels [dependabot]\nbots:\n  - renovate\non:\n  bots: *labels\n",
		"on: &trigger\n  bots:\n    - dependabot\nother: *trigger\nbots:\n  - renovate\n",
	} {
		t.Run(source, func(t *testing.T) {
			body := "\n# Body  \n"
			updated, applied, err := Apply("---\n"+source+"---\n"+body, func(f *Frontmatter) error {
				return f.MergeSequence([]string{"bots"}, []string{"on", "bots"})
			})
			require.NoError(t, err)
			assert.True(t, applied)
			document, err := aw.Parse(updated)
			require.NoError(t, err)
			assert.NotContains(t, document.Frontmatter, "bots")
			assert.Equal(t, []any{"dependabot", "renovate"}, document.Frontmatter["on"].(map[string]any)["bots"])
			if strings.HasPrefix(source, "labels: &labels [dependabot]") {
				assert.Equal(t, []any{"dependabot"}, document.Frontmatter["labels"])
			} else if strings.Contains(source, "labels:") {
				assert.Equal(t, []any{"renovate"}, document.Frontmatter["labels"])
			} else {
				assert.Equal(t, map[string]any{"bots": []any{"dependabot"}}, document.Frontmatter["other"])
			}
			assert.Equal(t, body, document.Markdown)
		})
	}
}

func TestMergeSequenceInvalidSequences(t *testing.T) {
	for _, source := range []string{
		"on: {bots: []}\n",
		"bots: null\non: {bots: []}\n",
		"bots: []\n",
		"bots: []\non: {bots: null}\n",
	} {
		content := "---\n" + source + "---\n# Body\n"
		updated, applied, err := Apply(content, func(f *Frontmatter) error {
			return f.MergeSequence([]string{"bots"}, []string{"on", "bots"})
		})
		require.Error(t, err)
		assert.False(t, applied)
		assert.Equal(t, content, updated)
	}
}
