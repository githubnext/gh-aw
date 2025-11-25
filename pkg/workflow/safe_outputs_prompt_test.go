package workflow

import (
	"strings"
	"testing"
)

func TestGenerateSafeOutputsPromptStep_IncludesWhenEnabled(t *testing.T) {
	compiler := &Compiler{}
	var yaml strings.Builder

	compiler.generateSafeOutputsPromptStep(&yaml, true)

	output := yaml.String()
	if !strings.Contains(output, "Append safe outputs instructions to prompt") {
		t.Error("Expected safe outputs prompt step to be generated when enabled")
	}
	if !strings.Contains(output, "safeoutputs MCP server") {
		t.Error("Expected prompt to mention safeoutputs MCP server")
	}
	if !strings.Contains(output, "gh (GitHub CLI) command is NOT authenticated") {
		t.Error("Expected prompt to warn about gh CLI not being authenticated")
	}
}

func TestGenerateSafeOutputsPromptStep_SkippedWhenDisabled(t *testing.T) {
	compiler := &Compiler{}
	var yaml strings.Builder

	compiler.generateSafeOutputsPromptStep(&yaml, false)

	output := yaml.String()
	if strings.Contains(output, "safe outputs") {
		t.Error("Expected safe outputs prompt step to NOT be generated when disabled")
	}
}

func TestGenerateSafeOutputsPromptStep_ListsAvailableTools(t *testing.T) {
	compiler := &Compiler{}
	var yaml strings.Builder

	compiler.generateSafeOutputsPromptStep(&yaml, true)

	output := yaml.String()
	expectedTools := []string{
		"create_issue",
		"create_discussion",
		"add_comment",
		"create_pull_request",
		"add_labels",
		"update_issue",
		"noop",
		"missing_tool",
	}

	for _, tool := range expectedTools {
		if !strings.Contains(output, tool) {
			t.Errorf("Expected prompt to mention tool: %s", tool)
		}
	}
}

func TestSafeOutputsPromptText_FollowsXMLFormat(t *testing.T) {
	if !strings.Contains(safeOutputsPromptText, "<safe-outputs>") {
		t.Error("Expected prompt to start with <safe-outputs> XML tag")
	}
	if !strings.Contains(safeOutputsPromptText, "</safe-outputs>") {
		t.Error("Expected prompt to end with </safe-outputs> XML tag")
	}
	if !strings.Contains(safeOutputsPromptText, "<important>") {
		t.Error("Expected prompt to contain <important> section")
	}
	if !strings.Contains(safeOutputsPromptText, "<available-tools>") {
		t.Error("Expected prompt to contain <available-tools> section")
	}
}
