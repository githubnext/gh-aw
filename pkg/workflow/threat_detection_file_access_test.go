package workflow

import (
	"os"
	"strings"
	"testing"
)

// readFile is a helper to read file contents
func readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

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

	// Verify that the setup script requires the setup_threat_detection.cjs file
	if !strings.Contains(stepsString, "setup_threat_detection.cjs") {
		t.Error("Expected threat detection to require setup_threat_detection.cjs file")
	}

	// Verify that we call main without parameters (template is read from file)
	if !strings.Contains(stepsString, "await main()") {
		t.Error("Expected to call main function without parameters")
	}

	// Verify that the template content is NOT passed inline anymore
	if strings.Contains(stepsString, "const templateContent = `# Threat Detection Analysis") {
		t.Error("Template content should not be passed inline - it should be read from file")
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
	// Read the threat detection template from the file
	templatePath := "../../actions/setup/md/threat_detection.md"
	templateContent, err := readFile(templatePath)
	if err != nil {
		t.Fatalf("Failed to read threat detection template: %v", err)
	}

	// Check that the template uses file path reference
	if !strings.Contains(templateContent, "Agent Output File") {
		t.Error("Expected template to have 'Agent Output File' section")
	}

	if !strings.Contains(templateContent, "{AGENT_OUTPUT_FILE}") {
		t.Error("Expected template to use {AGENT_OUTPUT_FILE} placeholder")
	}

	if !strings.Contains(templateContent, "Read and analyze this file") {
		t.Error("Expected template to instruct agent to read the file")
	}

	// Verify the old inline approach is removed
	if strings.Contains(templateContent, "{AGENT_OUTPUT}") {
		t.Error("Template should not use {AGENT_OUTPUT} placeholder anymore")
	}

	if strings.Contains(templateContent, "<agent-output>") {
		t.Error("Template should not have <agent-output> tag for inline content")
	}
}
