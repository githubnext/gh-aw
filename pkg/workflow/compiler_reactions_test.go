package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestAIReactionWorkflow(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "reaction-test")

	// Create a test markdown file with reaction
	testContent := `---
on:
  issues:
    types: [opened]
  reaction: eyes
permissions:
  contents: read
  issues: write
  pull-requests: write
strict: false
tools:
  github:
    toolsets: [issues]
timeout-minutes: 5
---

# AI Reaction Test

Test workflow with reaction.
`

	testFile := filepath.Join(tmpDir, "test-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
compiler.SetActionMode(ActionModeRelease) // Use release mode for inline JavaScript

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify reaction field is parsed correctly
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains reaction jobs
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for reaction-specific content in generated YAML
	expectedStrings := []string{
		"GH_AW_REACTION: \"eyes\"",
		"uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected string: %s", expected)
		}
	}

	// Verify three jobs are created (pre_activation, activation, agent) - reaction step is now in activation job
	// Count jobs by checking for job names (more reliable than counting runs-on)
	jobCount := 0
	if strings.Contains(yamlContent, "pre_activation:") {
		jobCount++
	}
	if strings.Contains(yamlContent, "activation:") {
		jobCount++
	}
	if strings.Contains(yamlContent, "agent:") {
		jobCount++
	}
	if jobCount != 3 {
		t.Errorf("Expected 3 jobs (pre_activation, activation, agent), found %d", jobCount)
	}

	// Verify reaction step is in activation job, not a separate job
	if strings.Contains(yamlContent, "add_reaction:") {
		t.Error("Generated YAML should not contain separate add_reaction job")
	}

	// Verify reaction step is in activation job
	activationJobSection := extractJobSection(yamlContent, "activation")
	if !strings.Contains(activationJobSection, "Add eyes reaction to the triggering item") {
		t.Error("Activation job should contain the reaction step")
	}
}

// TestAIReactionWorkflowWithoutReaction tests that workflows without explicit reaction do not create reaction actions
func TestAIReactionWorkflowWithoutReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "no-reaction-test")

	// Create a test markdown file without explicit reaction (should not create reaction action)
	testContent := `---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  issues: write
  pull-requests: read
strict: false
tools:
  github:
    toolsets: [issues]
timeout-minutes: 5
---

# No Reaction Test

Test workflow without explicit reaction (should not create reaction action).
`

	testFile := filepath.Join(tmpDir, "test-no-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
compiler.SetActionMode(ActionModeRelease) // Use release mode for inline JavaScript

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify reaction field is empty (not defaulted)
	if workflowData.AIReaction != "" {
		t.Errorf("Expected AIReaction to be empty, got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it does NOT contain reaction jobs
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check that reaction-specific content is NOT in generated YAML
	unexpectedStrings := []string{
		"GH_AW_REACTION:",
		"Add eyes reaction to the triggering item",
	}

	for _, unexpected := range unexpectedStrings {
		if strings.Contains(yamlContent, unexpected) {
			t.Errorf("Generated YAML should NOT contain: %s", unexpected)
		}
	}

	// Verify three jobs are created (pre_activation, activation, agent) - no separate add_reaction job
	// Count jobs by checking for job names (more reliable than counting runs-on)
	jobCount := 0
	if strings.Contains(yamlContent, "pre_activation:") {
		jobCount++
	}
	if strings.Contains(yamlContent, "activation:") {
		jobCount++
	}
	if strings.Contains(yamlContent, "agent:") {
		jobCount++
	}
	if jobCount != 3 {
		t.Errorf("Expected 3 jobs (pre_activation, activation, agent), found %d", jobCount)
	}
}

// TestAIReactionWithCommentEditFunctionality tests that the enhanced reaction script includes comment creation
func TestAIReactionWithCommentEditFunctionality(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "reaction-edit-test")

	// Create a test markdown file with reaction
	testContent := `---
on:
  issue_comment:
    types: [created]
  reaction: eyes
permissions:
  contents: read
  issues: write
  pull-requests: write
strict: false
tools:
  github:
    allowed: [issue_read]
---

# AI Reaction with Comment Creation Test

Test workflow with reaction and comment creation.
`

	testFile := filepath.Join(tmpDir, "test-reaction-edit.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
compiler.SetActionMode(ActionModeRelease) // Use release mode for inline JavaScript

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify reaction field is parsed correctly
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains the enhanced reaction script
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for enhanced reaction functionality in generated YAML
	expectedStrings := []string{
		"GH_AW_REACTION: \"eyes\"",
		"uses: actions/github-script@ed597411d8f924073f98dfc5c65a23a2325f34cd",
		"addCommentWithWorkflowLink", // This should be in the new script
		"runUrl =",                   // This should be in the new script for workflow run URL
		"Comment endpoint",           // This should be logged in the new script
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected string: %s", expected)
		}
	}

	// Verify that the script includes comment creation logic for all workflows (not just command workflows)
	if !strings.Contains(yamlContent, "shouldCreateComment") {
		t.Error("Generated YAML should contain shouldCreateComment logic")
	}

	// Verify the script handles different event types appropriately
	if !strings.Contains(yamlContent, "issue_comment") {
		t.Error("Generated YAML should reference issue_comment event handling")
	}

	// Verify reaction step is in activation job, not a separate job
	if strings.Contains(yamlContent, "add_reaction:") {
		t.Error("Generated YAML should not contain separate add_reaction job")
	}

	// Verify reaction step is in activation job
	activationJobSection := extractJobSection(yamlContent, "activation")
	if !strings.Contains(activationJobSection, "Add eyes reaction to the triggering item") {
		t.Error("Activation job should contain the reaction step")
	}
}

// TestCommandReactionWithCommentEdit tests command workflows with reaction and comment editing
func TestCommandReactionWithCommentEdit(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "command-reaction-edit-test")

	// Create a test markdown file with command and reaction
	testContent := `---
on:
  command:
    name: test-bot
  reaction: eyes
permissions:
  contents: read
  issues: write
  pull-requests: write
strict: false
tools:
  github:
    allowed: [issue_read]
---

# Command Bot with Reaction Test

Test command workflow with reaction and comment editing.
`

	testFile := filepath.Join(tmpDir, "test-command-bot.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
compiler.SetActionMode(ActionModeRelease) // Use release mode for inline JavaScript

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify command and reaction fields are parsed correctly
	if workflowData.Command != "test-bot" {
		t.Errorf("Expected Command to be 'test-bot', got '%s'", workflowData.Command)
	}
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains both alias and reaction environment variables
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for both environment variables in the generated YAML
	expectedEnvVars := []string{
		"GH_AW_REACTION: \"eyes\"",
		"GH_AW_COMMAND: test-bot",
	}

	for _, expected := range expectedEnvVars {
		if !strings.Contains(yamlContent, expected) {
			t.Errorf("Generated YAML does not contain expected environment variable: %s", expected)
		}
	}

	// Verify the script contains comment creation logic (now always enabled, not just for command workflows)
	if !strings.Contains(yamlContent, "shouldCreateComment = true") {
		t.Error("Generated YAML should contain comment creation logic")
	}
}

// TestCommandTriggerDefaultReaction tests that command triggers automatically enable "eyes" reaction
func TestCommandTriggerDefaultReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "command-default-reaction-test")

	// Create a test markdown file with command but NO explicit reaction
	testContent := `---
on:
  command:
    name: auto-bot
permissions:
  contents: read
  issues: write
  pull-requests: write
strict: false
tools:
  github:
    allowed: [issue_read]
---

# Command Bot with Auto Reaction

Test command workflow that should automatically get "eyes" reaction.
`

	testFile := filepath.Join(tmpDir, "test-auto-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
compiler.SetActionMode(ActionModeRelease) // Use release mode for inline JavaScript

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify command is parsed correctly
	if workflowData.Command != "auto-bot" {
		t.Errorf("Expected Command to be 'auto-bot', got '%s'", workflowData.Command)
	}

	// Verify reaction is automatically set to "eyes"
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to be auto-set to 'eyes', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains reaction
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for reaction environment variable in the generated YAML
	if !strings.Contains(yamlContent, "GH_AW_REACTION: \"eyes\"") {
		t.Error("Generated YAML should contain default 'eyes' reaction for command workflow")
	}

	// Check for command environment variable
	if !strings.Contains(yamlContent, "GH_AW_COMMAND: auto-bot") {
		t.Error("Generated YAML should contain command environment variable")
	}

	// Verify reaction step is in activation job, not a separate job
	if strings.Contains(yamlContent, "add_reaction:") {
		t.Error("Generated YAML should not contain separate add_reaction job")
	}

	// Verify reaction step is in activation job
	activationJobSection := extractJobSection(yamlContent, "activation")
	if !strings.Contains(activationJobSection, "Add eyes reaction to the triggering item") {
		t.Error("Activation job should contain the reaction step")
	}
}

// TestCommandTriggerCustomReaction tests that command triggers allow custom reaction override
func TestCommandTriggerCustomReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "command-custom-reaction-test")

	// Create a test markdown file with command and custom reaction
	testContent := `---
on:
  command:
    name: custom-bot
  reaction: rocket
permissions:
  contents: read
  issues: write
  pull-requests: write
strict: false
tools:
  github:
    allowed: [issue_read]
---

# Command Bot with Custom Reaction

Test command workflow with custom reaction override.
`

	testFile := filepath.Join(tmpDir, "test-custom-reaction.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")
compiler.SetActionMode(ActionModeRelease) // Use release mode for inline JavaScript

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify command is parsed correctly
	if workflowData.Command != "custom-bot" {
		t.Errorf("Expected Command to be 'custom-bot', got '%s'", workflowData.Command)
	}

	// Verify custom reaction overrides the default
	if workflowData.AIReaction != "rocket" {
		t.Errorf("Expected AIReaction to be 'rocket', got '%s'", workflowData.AIReaction)
	}

	// Generate YAML and verify it contains custom reaction
	yamlContent, err := compiler.generateYAML(workflowData, testFile)
	if err != nil {
		t.Fatalf("Failed to generate YAML: %v", err)
	}

	// Check for custom reaction in the generated YAML
	if !strings.Contains(yamlContent, "GH_AW_REACTION: \"rocket\"") {
		t.Error("Generated YAML should contain custom 'rocket' reaction")
	}

	// Verify it doesn't contain default "eyes"
	if strings.Contains(yamlContent, "GH_AW_REACTION: \"eyes\"") {
		t.Error("Generated YAML should not contain default 'eyes' when custom reaction is specified")
	}
}

// TestInvalidReactionValue tests that invalid reaction values are rejected
