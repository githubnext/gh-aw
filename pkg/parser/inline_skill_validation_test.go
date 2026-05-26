//go:build !integration

package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateInlineSkillsFrontmatter_NoSkills verifies that a file with no
// inline skill markers produces no warnings.
func TestValidateInlineSkillsFrontmatter_NoSkills(t *testing.T) {
	markdown := `---
engine: copilot
on:
  workflow_dispatch:
---
# Main workflow
Do some work.
`
	warnings := ValidateInlineSkillsFrontmatter(markdown)
	assert.Empty(t, warnings, "no skills should produce no warnings")
}

// TestValidateInlineSkillsFrontmatter_ValidFields verifies that known fields
// (description) produce no warnings.
func TestValidateInlineSkillsFrontmatter_ValidFields(t *testing.T) {
	markdown := strings.Join([]string{
		"---",
		"engine: copilot",
		"on:",
		"  workflow_dispatch:",
		"---",
		"# Main workflow",
		"",
		skillLine("helper"),
		"---",
		"description: A helpful skill",
		"---",
		"You are a helpful assistant.",
	}, "\n")

	warnings := ValidateInlineSkillsFrontmatter(markdown)
	assert.Empty(t, warnings, "only valid fields should produce no warnings")
}

// TestValidateInlineSkillsFrontmatter_UnknownField verifies that an unknown
// frontmatter field in a skill block produces a warning.
func TestValidateInlineSkillsFrontmatter_UnknownField(t *testing.T) {
	markdown := strings.Join([]string{
		"---",
		"engine: copilot",
		"on:",
		"  workflow_dispatch:",
		"---",
		"# Main workflow",
		"",
		skillLine("helper"),
		"---",
		"description: A helpful skill",
		"engine: copilot",
		"---",
		"You are a helpful assistant.",
	}, "\n")

	warnings := ValidateInlineSkillsFrontmatter(markdown)
	assert.Len(t, warnings, 1, "one unknown field should produce one warning")
	assert.Contains(t, warnings[0], `skill "helper"`, "warning should include skill name")
	assert.Contains(t, warnings[0], "engine", "warning should name the unknown field")
	assert.Contains(t, warnings[0], "description", "warning should list valid fields")
}

// TestValidateInlineSkillsFrontmatter_MultipleUnknownFields verifies that
// multiple unknown fields in the same skill are reported in a single warning.
func TestValidateInlineSkillsFrontmatter_MultipleUnknownFields(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		skillLine("worker"),
		"---",
		"engine: copilot",
		"on:",
		"  workflow_dispatch:",
		"---",
		"Do work.",
	}, "\n")

	warnings := ValidateInlineSkillsFrontmatter(markdown)
	assert.Len(t, warnings, 1, "multiple unknown fields should produce one warning per skill")
	assert.Contains(t, warnings[0], `skill "worker"`, "warning should include skill name")
	assert.Contains(t, warnings[0], "engine", "warning should mention engine field")
	assert.Contains(t, warnings[0], "on", "warning should mention on field")
}

// TestValidateInlineSkillsFrontmatter_MultipleSkills verifies that each
// skill with issues produces its own warning.
func TestValidateInlineSkillsFrontmatter_MultipleSkills(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		skillLine("planner"),
		"---",
		"description: The planner",
		"bad-field: value",
		"---",
		"Plan things.",
		"",
		skillLine("executor"),
		"---",
		"description: The executor",
		"also-bad: yes",
		"---",
		"Execute things.",
	}, "\n")

	warnings := ValidateInlineSkillsFrontmatter(markdown)
	assert.Len(t, warnings, 2, "each skill with issues should produce one warning")

	combined := strings.Join(warnings, " ")
	assert.Contains(t, combined, "planner", "should warn about planner skill")
	assert.Contains(t, combined, "executor", "should warn about executor skill")
}

// TestValidateInlineSkillsFrontmatter_NoFrontmatter verifies that a skill
// without a frontmatter block produces no warning.
func TestValidateInlineSkillsFrontmatter_NoFrontmatter(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		skillLine("helper"),
		"You are a helpful assistant with no frontmatter.",
	}, "\n")

	warnings := ValidateInlineSkillsFrontmatter(markdown)
	assert.Empty(t, warnings, "skill without frontmatter should produce no warnings")
}

// TestValidateInlineSkillsFrontmatter_EmptyContent verifies that empty input
// produces no warnings.
func TestValidateInlineSkillsFrontmatter_EmptyContent(t *testing.T) {
	warnings := ValidateInlineSkillsFrontmatter("")
	assert.Empty(t, warnings, "empty content should produce no warnings")
}

// TestValidateInlineSkillsFrontmatter_TopLevelFrontmatterNotValidated verifies
// that fields in the top-level file frontmatter are not reported as unknown
// (only skill frontmatter is checked).
func TestValidateInlineSkillsFrontmatter_TopLevelFrontmatterNotValidated(t *testing.T) {
	markdown := strings.Join([]string{
		"---",
		"engine: copilot",
		"permissions:",
		"  contents: read",
		"on:",
		"  workflow_dispatch:",
		"---",
		"# Main workflow",
		"",
		skillLine("helper"),
		"---",
		"description: Helper",
		"---",
		"Help out.",
	}, "\n")

	warnings := ValidateInlineSkillsFrontmatter(markdown)
	assert.Empty(t, warnings, "top-level frontmatter fields must not trigger skill warnings")
}

// TestValidateInlineSkillsFrontmatter_DuplicateSkillNames verifies that when
// ExtractInlineSkills fails (e.g. duplicate skill names), a warning is returned
// instead of silently returning nil.
func TestValidateInlineSkillsFrontmatter_DuplicateSkillNames(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		skillLine("helper"),
		"---",
		"description: First helper",
		"---",
		"First helper content.",
		"",
		skillLine("helper"),
		"---",
		"description: Duplicate name",
		"---",
		"Second helper content.",
	}, "\n")

	warnings := ValidateInlineSkillsFrontmatter(markdown)
	assert.NotEmpty(t, warnings, "duplicate skill names should produce a warning")
	assert.Contains(t, warnings[0], "helper", "warning should mention the duplicate skill name")
}

// TestValidateInlineSkillsFrontmatter_FieldFormat verifies that unknown fields are
// formatted with comma separation rather than Go slice notation.
func TestValidateInlineSkillsFrontmatter_FieldFormat(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		skillLine("worker"),
		"---",
		"engine: copilot",
		"on:",
		"  workflow_dispatch:",
		"---",
		"Do work.",
	}, "\n")

	warnings := ValidateInlineSkillsFrontmatter(markdown)
	assert.Len(t, warnings, 1, "should produce one warning")
	// Fields should be comma-separated, not formatted as a Go slice [engine on]
	assert.NotContains(t, warnings[0], "[", "warning should not use Go slice notation")
	assert.NotContains(t, warnings[0], "]", "warning should not use Go slice notation")
}
