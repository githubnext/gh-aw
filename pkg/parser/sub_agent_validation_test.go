//go:build !integration

package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateInlineSubAgentsFrontmatter_NoAgents verifies that a file with no
// inline sub-agent markers produces no warnings.
func TestValidateInlineSubAgentsFrontmatter_NoAgents(t *testing.T) {
	markdown := `---
engine: copilot
on:
  workflow_dispatch:
---
# Main workflow
Do some work.
`
	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.Empty(t, warnings, "no sub-agents should produce no warnings")
}

// TestValidateInlineSubAgentsFrontmatter_ValidFields verifies that known fields
// (description and model) produce no warnings.
func TestValidateInlineSubAgentsFrontmatter_ValidFields(t *testing.T) {
	markdown := strings.Join([]string{
		"---",
		"engine: copilot",
		"on:",
		"  workflow_dispatch:",
		"---",
		"# Main workflow",
		"",
		agentLine("helper"),
		"---",
		"description: A helpful sub-agent",
		"model: claude-haiku-4.5",
		"---",
		"You are a helpful assistant.",
	}, "\n")

	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.Empty(t, warnings, "only valid fields should produce no warnings")
}

// TestValidateInlineSubAgentsFrontmatter_UnknownField verifies that an unknown
// frontmatter field in a sub-agent block produces a warning.
func TestValidateInlineSubAgentsFrontmatter_UnknownField(t *testing.T) {
	markdown := strings.Join([]string{
		"---",
		"engine: copilot",
		"on:",
		"  workflow_dispatch:",
		"---",
		"# Main workflow",
		"",
		agentLine("helper"),
		"---",
		"description: A helpful sub-agent",
		"engine: copilot",
		"---",
		"You are a helpful assistant.",
	}, "\n")

	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.Len(t, warnings, 1, "one unknown field should produce one warning")
	assert.Contains(t, warnings[0], `sub-agent "helper"`, "warning should include agent name")
	assert.Contains(t, warnings[0], "engine", "warning should name the unknown field")
	assert.Contains(t, warnings[0], "description, model", "warning should list valid fields")
}

// TestValidateInlineSubAgentsFrontmatter_MultipleUnknownFields verifies that
// multiple unknown fields in the same sub-agent are reported in a single warning.
func TestValidateInlineSubAgentsFrontmatter_MultipleUnknownFields(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		agentLine("worker"),
		"---",
		"engine: copilot",
		"on:",
		"  workflow_dispatch:",
		"---",
		"Do work.",
	}, "\n")

	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.Len(t, warnings, 1, "multiple unknown fields should produce one warning per agent")
	assert.Contains(t, warnings[0], `sub-agent "worker"`, "warning should include agent name")
	assert.Contains(t, warnings[0], "engine", "warning should mention engine field")
	assert.Contains(t, warnings[0], "on", "warning should mention on field")
}

// TestValidateInlineSubAgentsFrontmatter_MultipleAgents verifies that each
// sub-agent with issues produces its own warning.
func TestValidateInlineSubAgentsFrontmatter_MultipleAgents(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		agentLine("planner"),
		"---",
		"description: The planner",
		"bad-field: value",
		"---",
		"Plan things.",
		"",
		agentLine("executor"),
		"---",
		"model: claude-haiku-4.5",
		"also-bad: yes",
		"---",
		"Execute things.",
	}, "\n")

	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.Len(t, warnings, 2, "each agent with issues should produce one warning")

	combined := strings.Join(warnings, " ")
	assert.Contains(t, combined, "planner", "should warn about planner agent")
	assert.Contains(t, combined, "executor", "should warn about executor agent")
}

// TestValidateInlineSubAgentsFrontmatter_NoFrontmatter verifies that a sub-agent
// without a frontmatter block produces no warning.
func TestValidateInlineSubAgentsFrontmatter_NoFrontmatter(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		agentLine("helper"),
		"You are a helpful assistant with no frontmatter.",
	}, "\n")

	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.Empty(t, warnings, "sub-agent without frontmatter should produce no warnings")
}

// TestValidateInlineSubAgentsFrontmatter_EmptyContent verifies that empty input
// produces no warnings.
func TestValidateInlineSubAgentsFrontmatter_EmptyContent(t *testing.T) {
	warnings := ValidateInlineSubAgentsFrontmatter("")
	assert.Empty(t, warnings, "empty content should produce no warnings")
}

// TestValidateInlineSubAgentsFrontmatter_TopLevelFrontmatterNotValidated verifies
// that fields in the top-level file frontmatter are not reported as unknown
// (only sub-agent frontmatter is checked).
func TestValidateInlineSubAgentsFrontmatter_TopLevelFrontmatterNotValidated(t *testing.T) {
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
		agentLine("helper"),
		"---",
		"description: Helper",
		"---",
		"Help out.",
	}, "\n")

	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.Empty(t, warnings, "top-level frontmatter fields must not trigger sub-agent warnings")
}
