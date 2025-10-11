package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestThreatDetectionWorkflowMarkdownFileIntegration(t *testing.T) {
	// Create a workflow with substantial markdown content
	workflowMarkdown := `---
on:
  issues:
    types: [opened]
engine: claude
safe-outputs:
  create-issue:
    title-prefix: "[test] "
---

# Large Workflow for Testing

This is a test workflow with substantial markdown content to verify that the workflow markdown
is written to a file instead of being embedded as an environment variable in the YAML.

## Background
The goal is to avoid bloating the generated YAML and hitting the maximum expression size limit
when workflows have large markdown content.

## Implementation Details
Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.

### Section 1
Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur.
Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.

### Section 2
With 'single quotes', "double quotes", and $special characters that need proper handling.

## Conclusion
This should all be safely base64 encoded and written to a file.`

	// Create temp directory and file
	tmpDir := t.TempDir()
	testFile := tmpDir + "/test-large-workflow.md"

	if err := os.WriteFile(testFile, []byte(workflowMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(testFile, ".md") + ".lock.yml"
	content, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContent := string(content)

	// Verify the file-writing step is present
	if !strings.Contains(lockContent, "Write workflow markdown to file") {
		t.Error("Expected lock file to contain 'Write workflow markdown to file' step")
	}

	if !strings.Contains(lockContent, "mkdir -p /tmp/gh-aw/templates") {
		t.Error("Expected lock file to create /tmp/gh-aw/templates directory")
	}

	if !strings.Contains(lockContent, "base64 -d /tmp/gh-aw/templates/workflow.b64 > /tmp/gh-aw/templates/workflow.md") {
		t.Error("Expected lock file to use base64 decoding from .b64 file to write workflow.md")
	}

	// Verify WORKFLOW_MARKDOWN is NOT in environment variables
	// Find the "Setup threat detection" step and check its env section
	setupStepIndex := strings.Index(lockContent, "name: Setup threat detection")
	if setupStepIndex == -1 {
		t.Fatal("Could not find 'Setup threat detection' step in lock file")
	}

	// Extract the section from setup step to the next step
	nextStepIndex := strings.Index(lockContent[setupStepIndex+100:], "- name:")
	if nextStepIndex == -1 {
		nextStepIndex = len(lockContent) - setupStepIndex
	} else {
		nextStepIndex += setupStepIndex + 100
	}
	setupSection := lockContent[setupStepIndex:nextStepIndex]

	if strings.Contains(setupSection, "WORKFLOW_MARKDOWN:") {
		t.Error("Expected WORKFLOW_MARKDOWN to NOT be in environment variables of setup step")
	}

	// Verify that WORKFLOW_NAME and WORKFLOW_DESCRIPTION are still present
	if !strings.Contains(setupSection, "WORKFLOW_NAME:") {
		t.Error("Expected WORKFLOW_NAME to be in environment variables")
	}

	if !strings.Contains(setupSection, "WORKFLOW_DESCRIPTION:") {
		t.Error("Expected WORKFLOW_DESCRIPTION to be in environment variables")
	}

	// Verify the JavaScript reads from file
	if !strings.Contains(lockContent, "const workflowMarkdownPath = '/tmp/gh-aw/templates/workflow.md'") {
		t.Error("Expected JavaScript to define workflowMarkdownPath")
	}

	if !strings.Contains(lockContent, "workflowMarkdown = fs.readFileSync(workflowMarkdownPath, 'utf8')") {
		t.Error("Expected JavaScript to read workflow markdown from file")
	}

	// Verify that the actual markdown content is NOT in the environment variables section
	// but IS in the base64-encoded shell step
	if strings.Contains(setupSection, "Lorem ipsum dolor sit amet") {
		t.Error("Expected markdown content to NOT be in plain text in environment variables")
	}

	// Find the file-writing step and verify base64 content exists there
	writeStepIndex := strings.Index(lockContent, "Write workflow markdown to file")
	if writeStepIndex == -1 {
		t.Fatal("Could not find 'Write workflow markdown to file' step")
	}

	writeStepEnd := strings.Index(lockContent[writeStepIndex:], "- name:")
	if writeStepEnd == -1 {
		writeStepEnd = len(lockContent) - writeStepIndex
	}
	writeStepSection := lockContent[writeStepIndex : writeStepIndex+writeStepEnd]

	// The base64-encoded content should be written to a .b64 file first
	if !strings.Contains(writeStepSection, "echo '") {
		t.Error("Expected base64-encoded content in write step")
	}

	if !strings.Contains(writeStepSection, "/tmp/gh-aw/templates/workflow.b64") {
		t.Error("Expected base64 content to be written to workflow.b64 file")
	}

	if !strings.Contains(writeStepSection, "base64 -d /tmp/gh-aw/templates/workflow.b64") {
		t.Error("Expected base64 decoding from workflow.b64 file")
	}
}

func TestThreatDetectionStepsOrder(t *testing.T) {
	// Verify the order of steps in threat detection job
	compiler := NewCompiler(false, "", "test")

	data := &WorkflowData{
		Name:            "Test Workflow",
		Description:     "Test Description",
		MarkdownContent: "# Test Content",
		SafeOutputs: &SafeOutputsConfig{
			ThreatDetection: &ThreatDetectionConfig{
				Enabled: true,
			},
		},
	}

	steps := compiler.buildThreatDetectionSteps(data, "agent")
	stepsString := strings.Join(steps, "")

	// Find the indices of key steps
	downloadIndex := strings.Index(stepsString, "Download agent output artifact")
	echoIndex := strings.Index(stepsString, "Echo agent outputs")
	writeIndex := strings.Index(stepsString, "Write workflow markdown to file")
	setupIndex := strings.Index(stepsString, "Setup threat detection")
	uploadIndex := strings.Index(stepsString, "Upload threat detection log")

	// Verify all steps exist
	if downloadIndex == -1 {
		t.Error("Expected to find 'Download agent output artifact' step")
	}
	if echoIndex == -1 {
		t.Error("Expected to find 'Echo agent outputs' step")
	}
	if writeIndex == -1 {
		t.Error("Expected to find 'Write workflow markdown to file' step")
	}
	if setupIndex == -1 {
		t.Error("Expected to find 'Setup threat detection' step")
	}
	if uploadIndex == -1 {
		t.Error("Expected to find 'Upload threat detection log' step")
	}

	// Verify the order: download -> echo -> write -> setup -> ... -> upload
	if downloadIndex >= echoIndex {
		t.Error("Expected download step to come before echo step")
	}
	if echoIndex >= writeIndex {
		t.Error("Expected echo step to come before write workflow markdown step")
	}
	if writeIndex >= setupIndex {
		t.Error("Expected write workflow markdown step to come before setup step")
	}
	// Upload should be last
	if uploadIndex < setupIndex {
		t.Error("Expected upload step to come after setup step")
	}
}
