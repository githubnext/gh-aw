package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var commandsLog = logger.New("cli:commands")

// Package-level version information
var (
	version = "dev"
)

// Template content loaded from .github/ directory at runtime
// TODO: These should be embedded from pkg/cli/templates/ but files don't exist there yet
var copilotInstructionsTemplate string
var agenticWorkflowPromptTemplate string
var sharedAgenticWorkflowPromptTemplate string
var gettingStartedPromptTemplate string

func init() {
	// Load templates from .github/ directory at runtime
	// This is a temporary solution until files are moved to pkg/cli/templates/
	gitRoot, err := findGitRoot()
	if err != nil {
		// If not in a git repo, templates will remain empty
		// This is expected during tests
		return
	}

	// Load each template file, ignoring errors (files might not exist in all contexts)
	if content, err := os.ReadFile(filepath.Join(gitRoot, ".github/instructions/github-agentic-workflows.instructions.md")); err == nil {
		copilotInstructionsTemplate = string(content)
	}
	if content, err := os.ReadFile(filepath.Join(gitRoot, ".github/prompts/create-agentic-workflow.prompt.md")); err == nil {
		agenticWorkflowPromptTemplate = string(content)
	}
	if content, err := os.ReadFile(filepath.Join(gitRoot, ".github/prompts/create-shared-agentic-workflow.prompt.md")); err == nil {
		sharedAgenticWorkflowPromptTemplate = string(content)
	}
	if content, err := os.ReadFile(filepath.Join(gitRoot, ".github/prompts/setup-agentic-workflows.prompt.md")); err == nil {
		gettingStartedPromptTemplate = string(content)
	}
}

// SetVersionInfo sets the version information for the CLI
func SetVersionInfo(v string) {
	version = v
}

// GetVersion returns the current version
func GetVersion() string {
	return version
}

func isGHCLIAvailable() bool {
	cmd := exec.Command("gh", "--version")
	return cmd.Run() == nil
}

// resolveWorkflowFile resolves a file or workflow name to an actual file path
// Note: This function only looks for local workflows, not packages
func resolveWorkflowFile(fileOrWorkflowName string, verbose bool) (string, error) {
	// First, try to use it as a direct file path
	if _, err := os.Stat(fileOrWorkflowName); err == nil {
		commandsLog.Printf("Found workflow file at path: %s", fileOrWorkflowName)
		if verbose {
			fmt.Printf("Found workflow file at path: %s\n", fileOrWorkflowName)
		}
		// Return absolute path
		absPath, err := filepath.Abs(fileOrWorkflowName)
		if err != nil {
			return fileOrWorkflowName, nil // fallback to original path
		}
		return absPath, nil
	}

	// If it's not a direct file path, try to resolve it as a workflow name
	commandsLog.Printf("File not found at %s, trying to resolve as workflow name", fileOrWorkflowName)

	// Add .md extension if not present
	workflowPath := fileOrWorkflowName
	if !strings.HasSuffix(workflowPath, ".md") {
		workflowPath += ".md"
	}

	commandsLog.Printf("Looking for workflow file: %s", workflowPath)

	workflowsDir := getWorkflowsDir()

	// Try to find the workflow in local sources only (not packages)
	_, path, err := readWorkflowFile(workflowPath, workflowsDir)
	if err != nil {
		return "", fmt.Errorf("workflow '%s' not found in local .github/workflows or components", fileOrWorkflowName)
	}

	commandsLog.Print("Found workflow in local components")

	// Return absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path, nil // fallback to original path
	}
	return absPath, nil
}

// NewWorkflow creates a new workflow markdown file with template content
func NewWorkflow(workflowName string, verbose bool, force bool) error {
	if verbose {
		fmt.Printf("Creating new workflow: %s\n", workflowName)
	}

	// Get current working directory for .github/workflows
	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create .github/workflows directory if it doesn't exist
	githubWorkflowsDir := filepath.Join(workingDir, constants.GetWorkflowDir())
	if err := os.MkdirAll(githubWorkflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	// Construct the destination file path
	destFile := filepath.Join(githubWorkflowsDir, workflowName+".md")

	// Check if destination file already exists
	if _, err := os.Stat(destFile); err == nil && !force {
		return fmt.Errorf("workflow file '%s' already exists. Use --force to overwrite", destFile)
	}

	// Create the template content
	template := createWorkflowTemplate(workflowName)

	// Write the template to file
	if err := os.WriteFile(destFile, []byte(template), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file '%s': %w", destFile, err)
	}

	fmt.Printf("Created new workflow: %s\n", destFile)
	fmt.Printf("Edit the file to customize your workflow, then run '" + constants.CLIExtensionPrefix + " compile' to generate the GitHub Actions workflow.\n")

	return nil
}

// createWorkflowTemplate generates a concise workflow template with essential options
func createWorkflowTemplate(workflowName string) string {
	return `---
# Trigger - when should this workflow run?
on:
  workflow_dispatch:  # Manual trigger

# Alternative triggers (uncomment to use):
# on:
#   issues:
#     types: [opened, reopened]
#   pull_request:
#     types: [opened, synchronize]
#   schedule:
#     - cron: "0 9 * * 1"  # Every Monday at 9 AM UTC

# Permissions - what can this workflow access?
permissions:
  contents: read
  issues: write
  pull-requests: write

# Outputs - what APIs and tools can the AI use?
safe-outputs:
  create-issue:          # Creates issues (default max: 1)
    max: 5               # Optional: specify maximum number
  # create-agent-task:   # Creates GitHub Copilot agent tasks (max: 1)
  # create-pull-request: # Creates exactly one pull request
  # add-comment:   # Adds comments (default max: 1)
  #   max: 2             # Optional: specify maximum number
  # add-labels:

---

# ` + workflowName + `

Describe what you want the AI to do when this workflow runs.

## Instructions

Replace this section with specific instructions for the AI. For example:

1. Read the issue description and comments
2. Analyze the request and gather relevant information
3. Provide a helpful response or take appropriate action

Be clear and specific about what the AI should accomplish.

## Notes

- Run ` + "`" + constants.CLIExtensionPrefix + " compile`" + ` to generate the GitHub Actions workflow
- See https://githubnext.github.io/gh-aw/ for complete configuration options and tools documentation
`
}
