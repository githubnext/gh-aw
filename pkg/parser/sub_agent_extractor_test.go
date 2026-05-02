//go:build !integration

package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractInlineSubAgents_NoSeparators(t *testing.T) {
	markdown := "# Hello\n\nThis is a workflow."
	mainMarkdown, agents, err := ExtractInlineSubAgents(markdown)

	require.NoError(t, err, "no separators should not produce an error")
	assert.Equal(t, markdown, mainMarkdown, "markdown should be unchanged when no separators present")
	assert.Nil(t, agents, "agents should be nil when no separators found")
}

func TestExtractInlineSubAgents_EmptyMarkdown(t *testing.T) {
	mainMarkdown, agents, err := ExtractInlineSubAgents("")

	require.NoError(t, err, "empty markdown should not produce an error")
	assert.Empty(t, mainMarkdown, "empty markdown should return empty main")
	assert.Nil(t, agents, "agents should be nil for empty markdown")
}

func TestExtractInlineSubAgents_SingleAgent(t *testing.T) {
	markdown := `# Main workflow

Handle the issue.

## @agent: planner
---
engine: copilot
---
You are a planning assistant.`

	mainMarkdown, agents, err := ExtractInlineSubAgents(markdown)

	require.NoError(t, err, "single sub-agent should parse without error")
	assert.Equal(t, "# Main workflow\n\nHandle the issue.", mainMarkdown, "main markdown should exclude agent section")
	require.Len(t, agents, 1, "should extract one sub-agent")
	assert.Equal(t, "planner", agents[0].Name, "agent name should be 'planner'")
	assert.Equal(t, "---\nengine: copilot\n---\nYou are a planning assistant.", agents[0].Content, "agent content should be trimmed")
}

func TestExtractInlineSubAgents_MultipleAgents(t *testing.T) {
	markdown := `# Main workflow

Main prompt.

## @agent: planner
---
engine: copilot
---
You are a planner.

## @agent: executor
---
engine: copilot
---
You are an executor.`

	mainMarkdown, agents, err := ExtractInlineSubAgents(markdown)

	require.NoError(t, err, "multiple sub-agents should parse without error")
	assert.Equal(t, "# Main workflow\n\nMain prompt.", mainMarkdown, "main markdown should exclude agent sections")
	require.Len(t, agents, 2, "should extract two sub-agents")

	assert.Equal(t, "planner", agents[0].Name, "first agent name should be 'planner'")
	assert.Contains(t, agents[0].Content, "You are a planner.", "first agent content should contain prompt")

	assert.Equal(t, "executor", agents[1].Name, "second agent name should be 'executor'")
	assert.Contains(t, agents[1].Content, "You are an executor.", "second agent content should contain prompt")
}

func TestExtractInlineSubAgents_AgentAtStartOfFile(t *testing.T) {
	markdown := `## @agent: only-agent
---
engine: copilot
---
Agent prompt.`

	mainMarkdown, agents, err := ExtractInlineSubAgents(markdown)

	require.NoError(t, err, "agent at start of file should parse without error")
	assert.Empty(t, mainMarkdown, "main markdown should be empty when agent is first")
	require.Len(t, agents, 1, "should extract one sub-agent")
	assert.Equal(t, "only-agent", agents[0].Name, "agent name should be 'only-agent'")
}

func TestExtractInlineSubAgents_AgentWithoutFrontmatter(t *testing.T) {
	markdown := `Main workflow.

## @agent: simple
Just a prompt, no frontmatter.`

	_, agents, err := ExtractInlineSubAgents(markdown)

	require.NoError(t, err, "agent without frontmatter should parse without error")
	require.Len(t, agents, 1, "should extract one sub-agent")
	assert.Equal(t, "simple", agents[0].Name, "agent name should be 'simple'")
	assert.Equal(t, "Just a prompt, no frontmatter.", agents[0].Content, "agent content should be the prompt")
}

func TestExtractInlineSubAgents_SeparatorWithTrailingWhitespace(t *testing.T) {
	// Trailing whitespace after the name should be tolerated
	markdown := "Main.\n\n## @agent: padded   \nAgent content."

	_, agents, err := ExtractInlineSubAgents(markdown)

	require.NoError(t, err, "separator with trailing whitespace should be recognized")
	require.Len(t, agents, 1, "should extract one sub-agent")
	assert.Equal(t, "padded", agents[0].Name, "agent name should be 'padded'")
}

func TestExtractInlineSubAgents_InvalidNameNotRecognized(t *testing.T) {
	tests := []struct {
		name      string
		separator string
	}{
		{
			name:      "name starts with digit",
			separator: "## @agent: 1agent",
		},
		{
			name:      "name contains spaces",
			separator: "## @agent: my agent",
		},
		{
			name:      "name contains slash",
			separator: "## @agent: my/agent",
		},
		{
			name:      "missing name",
			separator: "## @agent:",
		},
		{
			name:      "wrong heading level (H1)",
			separator: "# @agent: myagent",
		},
		{
			name:      "wrong heading level (H3)",
			separator: "### @agent: myagent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markdown := "Main content.\n\n" + tt.separator + "\nAgent content."
			mainMarkdown, agents, err := ExtractInlineSubAgents(markdown)

			require.NoError(t, err, "invalid separator should be treated as regular text")
			assert.Equal(t, markdown, mainMarkdown, "invalid separator should not consume main markdown")
			assert.Nil(t, agents, "invalid separator should not produce agents")
		})
	}
}

func TestExtractInlineSubAgents_DuplicateNameError(t *testing.T) {
	markdown := `Main.

## @agent: planner
Content 1.

## @agent: planner
Content 2.`

	_, _, err := ExtractInlineSubAgents(markdown)

	require.Error(t, err, "duplicate agent name should produce an error")
	assert.Contains(t, err.Error(), "duplicate", "error should mention duplicate")
	assert.Contains(t, err.Error(), "planner", "error should include the duplicate name")
}

func TestExtractInlineSubAgents_NameVariants(t *testing.T) {
	tests := []struct {
		name      string
		separator string
		agentName string
	}{
		{"with hyphens", "## @agent: my-agent", "my-agent"},
		{"with underscores", "## @agent: my_agent", "my_agent"},
		{"with digits", "## @agent: agent1", "agent1"},
		{"mixed case", "## @agent: MyAgent", "MyAgent"},
		{"single letter", "## @agent: a", "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markdown := "Main.\n\n" + tt.separator + "\nContent."
			_, agents, err := ExtractInlineSubAgents(markdown)

			require.NoError(t, err, "valid agent name %q should parse without error", tt.agentName)
			require.Len(t, agents, 1, "should extract one sub-agent")
			assert.Equal(t, tt.agentName, agents[0].Name, "agent name should match")
		})
	}
}

func TestExtractInlineSubAgents_ContentTrimmed(t *testing.T) {
	// Content after the separator should have leading/trailing whitespace trimmed
	markdown := "Main.\n\n## @agent: trim-test\n\n\n  Agent content here.  \n\n"

	_, agents, err := ExtractInlineSubAgents(markdown)

	require.NoError(t, err, "content trimming should not produce an error")
	require.Len(t, agents, 1, "should extract one sub-agent")
	assert.Equal(t, "Agent content here.", agents[0].Content, "agent content should be trimmed")
}

func TestExtractInlineSubAgents_MainMarkdownTrailingNewlinesStripped(t *testing.T) {
	markdown := "Line 1.\nLine 2.\n\n\n## @agent: a\nContent."

	mainMarkdown, _, err := ExtractInlineSubAgents(markdown)

	require.NoError(t, err, "should parse without error")
	assert.Equal(t, "Line 1.\nLine 2.", mainMarkdown, "trailing newlines should be stripped from main markdown")
}
