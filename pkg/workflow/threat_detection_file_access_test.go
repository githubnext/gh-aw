package workflow

import (
	"strings"
	"testing"
)

// TestThreatDetectionUsesFilePathNotInline verifies that the threat detection job
// references the agent output file path instead of inlining the full content
func TestThreatDetectionUsesFilePathNotInline(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		Name:            "Test Workflow",
		Description:     "Test Description",
		MarkdownContent: "Test markdown content",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildThreatDetectionSteps(data, "agent")
	stepsString := strings.Join(steps, "")

	// Verify that the setup script checks for agent output file existence
	if !strings.Contains(stepsString, "agent_output.json") {
		t.Error("Expected threat detection to reference agent_output.json file")
	}

	// Verify that we use file path info instead of inline content
	if !strings.Contains(stepsString, "agentOutputFileInfo") {
		t.Error("Expected threat detection to use agentOutputFileInfo variable")
	}

	// Verify the prompt template references file path
	if !strings.Contains(stepsString, "{AGENT_OUTPUT_FILE}") {
		t.Error("Expected threat detection prompt to use {AGENT_OUTPUT_FILE} placeholder")
	}

	// Verify we replace with file info, not content
	if !strings.Contains(stepsString, ".replace(/{AGENT_OUTPUT_FILE}/g, agentOutputFileInfo)") {
		t.Error("Expected prompt to replace {AGENT_OUTPUT_FILE} with agentOutputFileInfo")
	}

	// Verify we DON'T inline the agent output content via environment variable
	if strings.Contains(stepsString, "AGENT_OUTPUT: ${{ needs.agent.outputs.output }}") {
		t.Error("Threat detection should not pass agent output via environment variable to avoid CLI overflow")
	}

	// Verify we DON'T use the old AGENT_OUTPUT replacement
	if strings.Contains(stepsString, ".replace(/{AGENT_OUTPUT}/g, process.env.AGENT_OUTPUT") {
		t.Error("Threat detection should not replace {AGENT_OUTPUT} with environment variable content")
	}
}

// TestThreatDetectionHasBashReadTools verifies that bash read tools are configured
func TestThreatDetectionHasBashReadTools(t *testing.T) {
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{},
		},
	}

	steps := compiler.buildThreatDetectionSteps(data, "agent")
	stepsString := strings.Join(steps, "")

	// Verify bash tools are configured - check for the comments in the execution step
	expectedBashTools := []string{
		"Bash(cat)",
		"Bash(head)",
		"Bash(tail)",
		"Bash(wc)",
		"Bash(grep)",
		"Bash(ls)",
		"Bash(jq)",
	}

	for _, tool := range expectedBashTools {
		if !strings.Contains(stepsString, tool) {
			t.Errorf("Expected threat detection to have bash tool: %s", tool)
		}
	}
}

// TestThreatDetectionTemplateUsesFilePath verifies the template markdown is updated
func TestThreatDetectionTemplateUsesFilePath(t *testing.T) {
	// Check that the embedded template uses file path reference
	if !strings.Contains(defaultThreatDetectionPrompt, "Agent Output File") {
		t.Error("Expected template to have 'Agent Output File' section")
	}

	if !strings.Contains(defaultThreatDetectionPrompt, "{AGENT_OUTPUT_FILE}") {
		t.Error("Expected template to use {AGENT_OUTPUT_FILE} placeholder")
	}

	if !strings.Contains(defaultThreatDetectionPrompt, "Read and analyze this file") {
		t.Error("Expected template to instruct agent to read the file")
	}

	// Verify the old inline approach is removed
	if strings.Contains(defaultThreatDetectionPrompt, "{AGENT_OUTPUT}") {
		t.Error("Template should not use {AGENT_OUTPUT} placeholder anymore")
	}

	if strings.Contains(defaultThreatDetectionPrompt, "<agent-output>") {
		t.Error("Template should not have <agent-output> tag for inline content")
	}
}
