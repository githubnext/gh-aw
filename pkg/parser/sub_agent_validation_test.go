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

// TestValidateInlineSubAgentsFrontmatter_PreservedFields verifies that authored
// sub-agent frontmatter is accepted without warnings.
func TestValidateInlineSubAgentsFrontmatter_PreservedFields(t *testing.T) {
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
		"tools:",
		"  github:",
		"    toolsets: [issues]",
		"model: claude-haiku-4.5",
		"engine: copilot",
		"---",
		"You are a helpful assistant.",
	}, "\n")

	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.Empty(t, warnings, "preserved sub-agent frontmatter should not produce warnings")
}

// TestValidateInlineSubAgentsFrontmatter_ArbitraryFields verifies that arbitrary
// frontmatter fields in a sub-agent block do not produce warnings.
func TestValidateInlineSubAgentsFrontmatter_ArbitraryFields(t *testing.T) {
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
	assert.Empty(t, warnings, "arbitrary sub-agent frontmatter should not produce warnings")
}

// TestValidateInlineSubAgentsFrontmatter_MultipleArbitraryFields verifies that
// multiple arbitrary fields in the same sub-agent are accepted.
func TestValidateInlineSubAgentsFrontmatter_MultipleArbitraryFields(t *testing.T) {
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
	assert.Empty(t, warnings, "multiple arbitrary fields should not produce warnings")
}

// TestValidateInlineSubAgentsFrontmatter_MultipleAgents verifies that multiple
// sub-agents with arbitrary fields are accepted.
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
	assert.Empty(t, warnings, "multiple agents with arbitrary frontmatter should not produce warnings")
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

// TestValidateInlineSubAgentsFrontmatter_DuplicateAgentNames verifies that when
// ExtractInlineSubAgents fails (e.g. duplicate agent names), a warning is returned
// instead of silently returning nil.
func TestValidateInlineSubAgentsFrontmatter_DuplicateAgentNames(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		agentLine("helper"),
		"---",
		"description: First helper",
		"---",
		"First helper content.",
		"",
		agentLine("helper"),
		"---",
		"description: Duplicate name",
		"---",
		"Second helper content.",
	}, "\n")

	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.NotEmpty(t, warnings, "duplicate agent names should produce a warning")
	assert.Contains(t, warnings[0], "helper", "warning should mention the duplicate agent name")
}

// TestValidateInlineSubAgentsFrontmatter_ParseError verifies that malformed
// sub-agent frontmatter still surfaces a warning.
func TestValidateInlineSubAgentsFrontmatter_ParseError(t *testing.T) {
	markdown := strings.Join([]string{
		"# Main workflow",
		"",
		agentLine("worker"),
		"---",
		`description: "unterminated string`,
		"model: claude-haiku-4.5",
		"---",
		"Do work.",
	}, "\n")

	warnings := ValidateInlineSubAgentsFrontmatter(markdown)
	assert.Len(t, warnings, 1, "should produce one warning")
	assert.Contains(t, warnings[0], `sub-agent "worker"`, "warning should include agent name")
	assert.Contains(t, warnings[0], "could not parse frontmatter", "warning should mention the parse failure")
}
