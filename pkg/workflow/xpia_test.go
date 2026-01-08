package workflow

import (
	"strings"
	"testing"
)

// TestXPIAPromptDefaultEnabled tests that XPIA prompt is enabled by default
func TestXPIAPromptDefaultEnabled(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Empty tools map should default to safety prompt enabled
	emptyTools := map[string]any{}
	result := compiler.extractSafetyPromptSetting(emptyTools)

	if !result {
		t.Error("Expected safety-prompt to be enabled by default when tools is empty")
	}

	// Nil tools should also default to enabled
	result = compiler.extractSafetyPromptSetting(nil)

	if !result {
		t.Error("Expected safety-prompt to be enabled by default when tools is nil")
	}
}

// TestXPIAPromptExplicitlyDisabled tests that XPIA prompt can be disabled
func TestXPIAPromptExplicitlyDisabled(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	toolsWithDisabledPrompt := map[string]any{
		"safety-prompt": false,
		"github": map[string]any{
			"allowed": []string{"issue_read"},
		},
	}

	result := compiler.extractSafetyPromptSetting(toolsWithDisabledPrompt)

	if result {
		t.Error("Expected safety-prompt to be disabled when explicitly set to false")
	}
}

// TestXPIAPromptExplicitlyEnabled tests that XPIA prompt can be explicitly enabled
func TestXPIAPromptExplicitlyEnabled(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	toolsWithEnabledPrompt := map[string]any{
		"safety-prompt": true,
		"github": map[string]any{
			"allowed": []string{"issue_read"},
		},
	}

	result := compiler.extractSafetyPromptSetting(toolsWithEnabledPrompt)

	if !result {
		t.Error("Expected safety-prompt to be enabled when explicitly set to true")
	}
}

// TestXPIAPromptInWorkflow tests that XPIA prompt appears in compiled workflow
func TestXPIAPromptInWorkflow(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Create a simple workflow data
	data := &WorkflowData{
		MarkdownContent: "Test workflow content",
		SafetyPrompt:    true,
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that XPIA prompt step is included
	if !strings.Contains(output, "Append XPIA security instructions to prompt") {
		t.Error("Expected XPIA security instructions step in workflow")
	}

	// Check that the cat command to the XPIA prompt file is included
	if !strings.Contains(output, "cat \"/opt/gh-aw/prompts/xpia_prompt.md\" >> \"$GH_AW_PROMPT\"") {
		t.Error("Expected cat command for XPIA prompt file")
	}
}

// TestXPIAPromptNotInWorkflowWhenDisabled tests that XPIA prompt is NOT included when disabled
func TestXPIAPromptNotInWorkflowWhenDisabled(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	// Create a simple workflow data with safety prompt disabled
	data := &WorkflowData{
		MarkdownContent: "Test workflow content",
		SafetyPrompt:    false,
	}

	var yaml strings.Builder
	compiler.generatePrompt(&yaml, data)

	output := yaml.String()

	// Check that XPIA prompt step is NOT included
	if strings.Contains(output, "Append XPIA security instructions to prompt") {
		t.Error("Did not expect XPIA security instructions step when disabled")
	}

	// Check that security notice is NOT included
	if strings.Contains(output, "IMPORTANT SECURITY NOTICE") {
		t.Error("Did not expect security notice when XPIA prompt is disabled")
	}
}
