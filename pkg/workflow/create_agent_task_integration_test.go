package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/constants"
)

func TestAgentTaskWorkflowCompilation(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow file
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
safe-outputs:
  create-agent-task:
    base: main
---

# Test Agent Task Creation

Create a GitHub Copilot agent task to improve the code quality.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow() error = %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockYAML := string(lockContent)

	// Verify the create_agent_task job was created
	if !strings.Contains(lockYAML, "create_agent_task:") {
		t.Error("Generated workflow does not contain create_agent_task job")
	}

	// Verify the job has the correct needs
	if !strings.Contains(lockYAML, "needs:") {
		t.Error("create_agent_task job missing needs field")
	}

	// Verify checkout step for gh CLI
	if !strings.Contains(lockYAML, "Checkout repository for gh CLI") {
		t.Error("create_agent_task job missing checkout step")
	}

	// Verify the job has correct outputs
	if !strings.Contains(lockYAML, "task_number:") {
		t.Error("create_agent_task job missing task_number output")
	}
	if !strings.Contains(lockYAML, "task_url:") {
		t.Error("create_agent_task job missing task_url output")
	}

	// Verify environment variables
	if !strings.Contains(lockYAML, "GITHUB_AW_AGENT_TASK_BASE") {
		t.Error("create_agent_task job missing GITHUB_AW_AGENT_TASK_BASE env var")
	}

	// Verify permissions
	if !strings.Contains(lockYAML, "contents: write") {
		t.Error("create_agent_task job missing contents: write permission")
	}
	if !strings.Contains(lockYAML, "issues: write") {
		t.Error("create_agent_task job missing issues: write permission")
	}
	if !strings.Contains(lockYAML, "pull-requests: write") {
		t.Error("create_agent_task job missing pull-requests: write permission")
	}

	// Verify timeout
	if !strings.Contains(lockYAML, "timeout-minutes: 10") {
		t.Error("create_agent_task job missing or incorrect timeout-minutes")
	}

	// Verify the JavaScript script is embedded
	if !strings.Contains(lockYAML, "create_agent_task") && !strings.Contains(lockYAML, "createAgentTaskItems") {
		t.Error("create_agent_task job missing JavaScript implementation")
	}
}

func TestAgentTaskWorkflowWithTargetRepo(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test workflow file with target-repo
	workflowContent := `---
on: workflow_dispatch
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
safe-outputs:
  create-agent-task:
    base: develop
    target-repo: org/other-repo
---

# Test Agent Task with Target Repo

Create a GitHub Copilot agent task in another repository.
`

	workflowPath := filepath.Join(tmpDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write test workflow: %v", err)
	}

	// Compile the workflow
	compiler := NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow() error = %v", err)
	}

	// Read the generated lock file
	lockFile := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockYAML := string(lockContent)

	// Verify the create_agent_task job was created
	if !strings.Contains(lockYAML, "create_agent_task:") {
		t.Error("Generated workflow does not contain create_agent_task job")
	}

	// Verify target repo configuration
	if !strings.Contains(lockYAML, "GITHUB_AW_TARGET_REPO") {
		t.Error("create_agent_task job missing GITHUB_AW_TARGET_REPO env var")
	}

	// Verify base branch configuration
	if !strings.Contains(lockYAML, "GITHUB_AW_AGENT_TASK_BASE") {
		t.Error("create_agent_task job missing GITHUB_AW_AGENT_TASK_BASE env var")
	}
}

func TestAgentTaskPromptSection(t *testing.T) {
	config := &SafeOutputsConfig{
		CreateAgentTasks: &CreateAgentTaskConfig{},
	}

	var builder strings.Builder
	generateSafeOutputsPromptSection(&builder, config)
	prompt := builder.String()

	// Verify the prompt includes agent task instructions
	if !strings.Contains(prompt, "Creating an Agent Task") {
		t.Error("Prompt section missing 'Creating an Agent Task' header")
	}

	if !strings.Contains(prompt, "create-agent-task") {
		t.Error("Prompt section missing create-agent-task tool reference")
	}

	if !strings.Contains(prompt, constants.SafeOutputsMCPServerID) {
		t.Error("Prompt section missing safeoutputs MCP reference")
	}
}
