//go:build !integration

package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skillLine returns a ## skill: `name` start marker line for use in test fixtures.
func skillLine(name string) string {
	return fmt.Sprintf("## skill: `%s`", name)
}

func TestExtractInlineSkills_NoSeparators(t *testing.T) {
	markdown := "# Hello\n\nThis is a workflow."
	mainMarkdown, skills, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "no separators should not produce an error")
	assert.Equal(t, markdown, mainMarkdown, "markdown should be unchanged when no separators present")
	assert.Nil(t, skills, "skills should be nil when no separators found")
}

func TestExtractInlineSkills_EmptyMarkdown(t *testing.T) {
	mainMarkdown, skills, err := ExtractInlineSkills("")

	require.NoError(t, err, "empty markdown should not produce an error")
	assert.Empty(t, mainMarkdown, "empty markdown should return empty main")
	assert.Nil(t, skills, "skills should be nil for empty markdown")
}

func TestExtractInlineSkills_SingleSkill(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		"Handle the issue.",
		"",
		skillLine("planner"),
		"---",
		"engine: copilot",
		"---",
		"You are a planning assistant.",
	}, "\n")

	mainMarkdown, skills, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "single skill should parse without error")
	assert.Equal(t, "# Main workflow\n\nHandle the issue.", mainMarkdown, "main markdown should exclude skill section")
	require.Len(t, skills, 1, "should extract one skill")
	assert.Equal(t, "planner", skills[0].Name, "skill name should be 'planner'")
	assert.Equal(t, "---\nengine: copilot\n---\nYou are a planning assistant.", skills[0].Content, "skill content should be trimmed")
}

func TestExtractInlineSkills_MultipleSkills(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		"Main prompt.",
		"",
		skillLine("planner"),
		"---",
		"engine: copilot",
		"---",
		"You are a planner.",
		"",
		skillLine("executor"),
		"---",
		"engine: copilot",
		"---",
		"You are an executor.",
	}, "\n")

	mainMarkdown, skills, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "multiple skills should parse without error")
	assert.Equal(t, "# Main workflow\n\nMain prompt.", mainMarkdown, "main markdown should exclude skill sections")
	require.Len(t, skills, 2, "should extract two skills")

	assert.Equal(t, "planner", skills[0].Name, "first skill name should be 'planner'")
	assert.Contains(t, skills[0].Content, "You are a planner.", "first skill content should contain prompt")

	assert.Equal(t, "executor", skills[1].Name, "second skill name should be 'executor'")
	assert.Contains(t, skills[1].Content, "You are an executor.", "second skill content should contain prompt")
}

func TestExtractInlineSkills_SkillAtStartOfFile(t *testing.T) {
	markdown := skillLine("only-skill") + "\n---\nengine: copilot\n---\nSkill prompt."

	mainMarkdown, skills, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "skill at start of file should parse without error")
	assert.Empty(t, mainMarkdown, "main markdown should be empty when skill is first")
	require.Len(t, skills, 1, "should extract one skill")
	assert.Equal(t, "only-skill", skills[0].Name, "skill name should be 'only-skill'")
}

func TestExtractInlineSkills_SkillWithoutFrontmatter(t *testing.T) {
	markdown := "Main workflow.\n\n" + skillLine("simple") + "\nJust a prompt, no frontmatter."

	_, skills, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "skill without frontmatter should parse without error")
	require.Len(t, skills, 1, "should extract one skill")
	assert.Equal(t, "simple", skills[0].Name, "skill name should be 'simple'")
	assert.Equal(t, "Just a prompt, no frontmatter.", skills[0].Content, "skill content should be the prompt")
}

func TestExtractInlineSkills_SeparatorWithTrailingWhitespace(t *testing.T) {
	// Trailing whitespace after the closing backtick should be tolerated
	markdown := "Main.\n\n" + skillLine("padded") + "   \nSkill content."

	_, skills, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "separator with trailing whitespace should be recognized")
	require.Len(t, skills, 1, "should extract one skill")
	assert.Equal(t, "padded", skills[0].Name, "skill name should be 'padded'")
}

func TestExtractInlineSkills_InvalidNameNotRecognized(t *testing.T) {
	tests := []struct {
		name      string
		separator string
	}{
		{
			name:      "name starts with digit",
			separator: "## skill: `1skill`",
		},
		{
			name:      "name contains spaces",
			separator: "## skill: `my skill`",
		},
		{
			name:      "name contains slash",
			separator: "## skill: `my/skill`",
		},
		{
			name:      "missing name",
			separator: "## skill:",
		},
		{
			name:      "name not in backticks",
			separator: "## skill: myskill",
		},
		{
			name:      "name uppercase",
			separator: "## skill: `MySkill`",
		},
		{
			name:      "wrong heading level (H1)",
			separator: "# skill: `myskill`",
		},
		{
			name:      "wrong heading level (H3)",
			separator: "### skill: `myskill`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markdown := "Main content.\n\n" + tt.separator + "\nSkill content."
			mainMarkdown, skills, err := ExtractInlineSkills(markdown)

			require.NoError(t, err, "invalid separator should be treated as regular text")
			assert.Equal(t, markdown, mainMarkdown, "invalid separator should not consume main markdown")
			assert.Nil(t, skills, "invalid separator should not produce skills")
		})
	}
}

func TestExtractInlineSkills_DuplicateNameError(t *testing.T) {
	markdown := "Main.\n\n" + skillLine("planner") + "\nContent 1.\n\n" + skillLine("planner") + "\nContent 2."

	_, _, err := ExtractInlineSkills(markdown)

	require.Error(t, err, "duplicate skill name should produce an error")
	assert.Contains(t, err.Error(), "duplicate", "error should mention duplicate")
	assert.Contains(t, err.Error(), "planner", "error should include the duplicate name")
}

func TestExtractInlineSkills_NameVariants(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
	}{
		{"with hyphens", "my-skill"},
		{"with underscores", "my_skill"},
		{"with digits", "skill1"},
		{"single letter", "a"},
		{"mixed pattern", "planner-v2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markdown := "Main.\n\n" + skillLine(tt.skillName) + "\nContent."
			_, skills, err := ExtractInlineSkills(markdown)

			require.NoError(t, err, "valid skill name %q should parse without error", tt.skillName)
			require.Len(t, skills, 1, "should extract one skill")
			assert.Equal(t, tt.skillName, skills[0].Name, "skill name should match")
		})
	}
}

func TestExtractInlineSkills_ContentTrimmed(t *testing.T) {
	// Content after the separator should have leading/trailing whitespace trimmed
	markdown := "Main.\n\n" + skillLine("trim-test") + "\n\n\n  Skill content here.  \n\n"

	_, skills, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "content trimming should not produce an error")
	require.Len(t, skills, 1, "should extract one skill")
	assert.Equal(t, "Skill content here.", skills[0].Content, "skill content should be trimmed")
}

func TestExtractInlineSkills_SkillEndsAtNextH2(t *testing.T) {
	// An skill block must end at the next H2 heading (any ##), not just ## skill:.
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		"Main prompt.",
		"",
		skillLine("planner"),
		"---",
		"engine: copilot",
		"---",
		"You are a planner.",
		"",
		"## Summary",
		"This content is outside the skill block.",
	}, "\n")

	mainMarkdown, skills, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "H2 ending should parse without error")
	assert.Equal(t, "# Main workflow\n\nMain prompt.", mainMarkdown, "main markdown should exclude skill section")
	require.Len(t, skills, 1, "should extract one skill")
	assert.Equal(t, "planner", skills[0].Name)
	assert.Contains(t, skills[0].Content, "You are a planner.", "skill content should contain prompt")
	assert.NotContains(t, skills[0].Content, "Summary", "content after H2 must not appear in skill")
	assert.NotContains(t, skills[0].Content, "outside the skill block", "content after H2 must not appear in skill")
}

func TestExtractInlineSkills_SkillEndsAtNextSkillH2(t *testing.T) {
	// A new ## skill: `name` marker (which is itself an H2) also ends the previous skill.
	markdown := strings.Join([]string{
		"Main.",
		"",
		skillLine("planner"),
		"Planner prompt.",
		"",
		skillLine("executor"),
		"Executor prompt.",
	}, "\n")

	_, skills, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "multiple skills should parse without error")
	require.Len(t, skills, 2, "should extract two skills")
	assert.Equal(t, "planner", skills[0].Name)
	assert.Equal(t, "Planner prompt.", skills[0].Content, "planner content must stop at next skill marker")
	assert.Equal(t, "executor", skills[1].Name)
	assert.Equal(t, "Executor prompt.", skills[1].Content)
}

func TestExtractInlineSkills_MainMarkdownTrailingNewlinesStripped(t *testing.T) {
	markdown := "Line 1.\nLine 2.\n\n\n" + skillLine("a") + "\nContent."

	mainMarkdown, _, err := ExtractInlineSkills(markdown)

	require.NoError(t, err, "should parse without error")
	assert.Equal(t, "Line 1.\nLine 2.", mainMarkdown, "trailing newlines should be stripped from main markdown")
}
