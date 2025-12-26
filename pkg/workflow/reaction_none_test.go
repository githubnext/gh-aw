package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestCommandWorkflowWithReactionNone(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "reaction-none-test")

	// Create a test markdown file with command and reaction: none
	testContent := `---
on:
  command:
    name: test-bot
  reaction: none
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
safe-outputs:
  add-comment:
---

# Command Bot with Reaction None

Test command workflow with reaction explicitly disabled.
`

	testFile := filepath.Join(tmpDir, "test-command-bot.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify command and reaction fields are parsed correctly
	if workflowData.Command != "test-bot" {
		t.Errorf("Expected Command to be 'test-bot', got '%s'", workflowData.Command)
	}

	if workflowData.AIReaction != "none" {
		t.Errorf("Expected AIReaction to be 'none', got '%s'", workflowData.AIReaction)
	}

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-command-bot.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that activation job does NOT have reaction step
	if strings.Contains(compiled, "Add none reaction to the triggering item") {
		t.Error("Activation job should not have reaction step when reaction is 'none'")
	}

	// Verify that activation job does NOT have reaction permissions
	activationJobSection := extractJobSection(compiled, string(constants.ActivationJobName))
	if strings.Contains(activationJobSection, "issues: write") {
		t.Error("Activation job should not have 'issues: write' permission when reaction is 'none'")
	}
	if strings.Contains(activationJobSection, "pull-requests: write") {
		t.Error("Activation job should not have 'pull-requests: write' permission when reaction is 'none'")
	}
	if strings.Contains(activationJobSection, "discussions: write") {
		t.Error("Activation job should not have 'discussions: write' permission when reaction is 'none'")
	}

	// Verify that activation job DOES have contents: read permission for checkout
	if !strings.Contains(activationJobSection, "contents: read") {
		t.Error("Activation job should have 'contents: read' permission for checkout step")
	}

	// Verify that conclusion job IS created (to handle noop messages)
	if !strings.Contains(compiled, "conclusion:") {
		t.Error("conclusion job should be created when safe-outputs exist (to handle noop)")
	}
}

func TestCommandWorkflowDefaultReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "reaction-default-test")

	// Create a test markdown file with command but no explicit reaction
	testContent := `---
on:
  command:
    name: test-bot
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
safe-outputs:
  add-comment:
---

# Command Bot with Default Reaction

Test command workflow with default (eyes) reaction.
`

	testFile := filepath.Join(tmpDir, "test-command-bot-default.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify command is parsed correctly
	if workflowData.Command != "test-bot" {
		t.Errorf("Expected Command to be 'test-bot', got '%s'", workflowData.Command)
	}

	// Verify AIReaction defaults to "eyes" for command workflows
	if workflowData.AIReaction != "eyes" {
		t.Errorf("Expected AIReaction to default to 'eyes' for command workflows, got '%s'", workflowData.AIReaction)
	}

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-command-bot-default.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that activation job HAS reaction step
	if !strings.Contains(compiled, "Add eyes reaction to the triggering item") {
		t.Error("Activation job should have reaction step when reaction defaults to 'eyes'")
	}

	// Verify that activation job HAS reaction permissions
	activationJobSection := extractJobSection(compiled, string(constants.ActivationJobName))
	if !strings.Contains(activationJobSection, "issues: write") {
		t.Error("Activation job should have 'issues: write' permission when reaction is enabled")
	}
	if !strings.Contains(activationJobSection, "pull-requests: write") {
		t.Error("Activation job should have 'pull-requests: write' permission when reaction is enabled")
	}
	if !strings.Contains(activationJobSection, "discussions: write") {
		t.Error("Activation job should have 'discussions: write' permission when reaction is enabled")
	}

	// Verify that activation job also has contents: read permission for checkout
	if !strings.Contains(activationJobSection, "contents: read") {
		t.Error("Activation job should have 'contents: read' permission for checkout step")
	}

	// Verify that conclusion job IS created
	if !strings.Contains(compiled, "conclusion:") {
		t.Error("conclusion job should be created when reaction is enabled and add-comment is configured")
	}
}

func TestCommandWorkflowExplicitReaction(t *testing.T) {
	// Create temporary directory for test files
	tmpDir := testutil.TempDir(t, "reaction-explicit-test")

	// Create a test markdown file with command and explicit reaction
	testContent := `---
on:
  command:
    name: test-bot
  reaction: rocket
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
strict: false
safe-outputs:
  add-comment:
---

# Command Bot with Rocket Reaction

Test command workflow with explicit rocket reaction.
`

	testFile := filepath.Join(tmpDir, "test-command-bot-rocket.md")
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatal(err)
	}

	compiler := NewCompiler(false, "", "test")

	// Parse the workflow
	workflowData, err := compiler.ParseWorkflowFile(testFile)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify AIReaction is set to "rocket"
	if workflowData.AIReaction != "rocket" {
		t.Errorf("Expected AIReaction to be 'rocket', got '%s'", workflowData.AIReaction)
	}

	// Compile the workflow
	if err := compiler.CompileWorkflow(testFile); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled workflow
	lockFile := filepath.Join(tmpDir, "test-command-bot-rocket.lock.yml")
	compiledBytes, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}
	compiled := string(compiledBytes)

	// Verify that activation job HAS rocket reaction step
	if !strings.Contains(compiled, "Add rocket reaction to the triggering item") {
		t.Error("Activation job should have rocket reaction step")
	}

	// Verify that conclusion job IS created
	if !strings.Contains(compiled, "conclusion:") {
		t.Error("conclusion job should be created when reaction is enabled and add-comment is configured")
	}
}
