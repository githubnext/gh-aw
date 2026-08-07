//go:build !integration

package workflow

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoMemoryTemplate_DoesNotContainMarkdownHeader(t *testing.T) {
	data, err := os.ReadFile("../../actions/setup/md/repo_memory_prompt.md")
	require.NoError(t, err, "should read repo memory template file")

	templateContent := string(data)
	assert.NotContains(t, templateContent, "## Repo Memory Available", "template should not include a markdown header in <repo-memory> section")
}

func TestRepoMemorySubfolderConstraint(t *testing.T) {
	t.Run("slashless pattern adds required layout note", func(t *testing.T) {
		assert.Contains(t, repoMemorySubfolderConstraint([]string{"*.md", "*.json"}), "**Required Layout**")
	})

	t.Run("patterns with explicit paths add no note", func(t *testing.T) {
		assert.Empty(t, repoMemorySubfolderConstraint([]string{"metrics/**", "data/*.json"}))
	})

	t.Run("no glob adds no note", func(t *testing.T) {
		assert.Empty(t, repoMemorySubfolderConstraint(nil))
	})
}

func TestRepoMemoryPromptConstraintsIncludeSubfolderRequirement(t *testing.T) {
	config := &RepoMemoryConfig{
		Memories: []RepoMemoryEntry{
			{
				ID:         "default",
				BranchName: "memory/deep-report",
				FileGlob:   []string{"*.md"},
			},
		},
	}

	section := buildRepoMemoryPromptSection(config)
	require.NotNil(t, section)
	assert.Contains(t, section.EnvVars["GH_AW_MEMORY_CONSTRAINTS"], "**Required Layout**")
}
