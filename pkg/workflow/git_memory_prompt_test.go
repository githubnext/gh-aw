package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGenerateGitMemoryPromptSection tests git memory prompt generation
func TestGenerateGitMemoryPromptSection(t *testing.T) {
	tests := []struct {
		name          string
		config        *GitMemoryConfig
		expectedEmpty bool
		contains      []string
		notContains   []string
	}{
		{
			name:          "nil config",
			config:        nil,
			expectedEmpty: true,
		},
		{
			name: "empty branches",
			config: &GitMemoryConfig{
				Branches: []GitMemoryEntry{},
			},
			expectedEmpty: true,
		},
		{
			name: "single default branch",
			config: &GitMemoryConfig{
				Branches: []GitMemoryEntry{
					{
						ID:     "default",
						Branch: "memory/default",
					},
				},
			},
			contains: []string{
				"## Git Memory Branch Available",
				"memory/default",
				"Git Storage",
				"Read/Write Access",
				"Orphaned Branch",
				"Fast-Forward Merge",
				"Automatic Commit",
				"notes.txt",
				"preferences.json",
			},
			notContains: []string{
				"## Git Memory Branches Available", // plural form
			},
		},
		{
			name: "single branch with description",
			config: &GitMemoryConfig{
				Branches: []GitMemoryEntry{
					{
						ID:          "audit",
						Branch:      "memory/audit",
						Description: "Stores audit workflow state",
					},
				},
			},
			contains: []string{
				"## Git Memory Branches Available", // uses plural form for non-default ID
				"memory/audit",
				"Stores audit workflow state",
				"**audit**",
			},
		},
		{
			name: "multiple branches",
			config: &GitMemoryConfig{
				Branches: []GitMemoryEntry{
					{
						ID:     "default",
						Branch: "memory/default",
					},
					{
						ID:          "session",
						Branch:      "memory/session",
						Description: "Session state",
					},
				},
			},
			contains: []string{
				"## Git Memory Branches Available", // plural form
				"memory/default",
				"memory/session",
				"Session state",
				"**default**",
				"**session**",
			},
			notContains: []string{
				"## Git Memory Branch Available", // singular form
			},
		},
		{
			name: "non-default single branch uses plural form",
			config: &GitMemoryConfig{
				Branches: []GitMemoryEntry{
					{
						ID:     "custom",
						Branch: "memory/custom",
					},
				},
			},
			contains: []string{
				"## Git Memory Branches Available", // plural form for non-default
				"memory/custom",
				"**custom**",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			generateGitMemoryPromptSection(&builder, tt.config)

			result := builder.String()

			if tt.expectedEmpty {
				assert.Empty(t, result)
				return
			}

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected, "should contain: %s", expected)
			}

			for _, notExpected := range tt.notContains {
				assert.NotContains(t, result, notExpected, "should not contain: %s", notExpected)
			}
		})
	}
}

// TestGenerateGitMemoryPromptStep tests the compiler method for prompt step generation
func TestGenerateGitMemoryPromptStep(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	tests := []struct {
		name          string
		config        *GitMemoryConfig
		expectedEmpty bool
		contains      []string
	}{
		{
			name:          "nil config generates no step",
			config:        nil,
			expectedEmpty: true,
		},
		{
			name: "valid config generates step",
			config: &GitMemoryConfig{
				Branches: []GitMemoryEntry{
					{
						ID:     "default",
						Branch: "memory/default",
					},
				},
			},
			contains: []string{
				"name: Append git memory instructions to prompt",
				"memory/default",
				"Git Memory Branch Available",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var builder strings.Builder
			compiler.generateGitMemoryPromptStep(&builder, tt.config)

			result := builder.String()

			if tt.expectedEmpty {
				assert.Empty(t, result)
				return
			}

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

// TestGitMemoryPromptSectionFormatting tests the formatting of prompt sections
func TestGitMemoryPromptSectionFormatting(t *testing.T) {
	config := &GitMemoryConfig{
		Branches: []GitMemoryEntry{
			{
				ID:     "default",
				Branch: "memory/default",
			},
		},
	}

	var builder strings.Builder
	generateGitMemoryPromptSection(&builder, config)

	result := builder.String()

	// Verify proper markdown formatting
	assert.Contains(t, result, "---\n") // Section separator
	assert.Contains(t, result, "## Git Memory") // Header
	assert.Contains(t, result, "- **") // Bullet points with bold
	assert.Contains(t, result, "Examples of what you can store:") // Examples section
	assert.Contains(t, result, "`memory/default`") // Code formatting for branch name
	assert.Contains(t, result, "`notes.txt`") // Code formatting for example files
}
