package cli

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var commandsLog = logger.New("cli:commands")

// Package-level version information
var (
	version = "dev"
)

//go:embed templates/github-agentic-workflows.md
var copilotInstructionsTemplate string

//go:embed templates/agentic-workflows.agent.md
var agenticWorkflowsDispatcherTemplate string

//go:embed templates/create-agentic-workflow.md
var createWorkflowPromptTemplate string

//go:embed templates/update-agentic-workflow.md
var updateWorkflowPromptTemplate string

//go:embed templates/create-shared-agentic-workflow.md
var createSharedAgenticWorkflowPromptTemplate string

//go:embed templates/debug-agentic-workflow.md
var debugWorkflowPromptTemplate string

//go:embed templates/upgrade-agentic-workflows.md
var upgradeAgenticWorkflowsPromptTemplate string

//go:embed templates/generate-agentic-campaign.md
var campaignGeneratorInstructionsTemplate string

//go:embed templates/orchestrate-agentic-campaign.md
var campaignOrchestratorInstructionsTemplate string

//go:embed templates/update-agentic-campaign-project.md
var campaignProjectUpdateInstructionsTemplate string

//go:embed templates/execute-agentic-campaign-workflow.md
var campaignWorkflowExecutionTemplate string

//go:embed templates/close-agentic-campaign.md
var campaignClosingInstructionsTemplate string

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
	return resolveWorkflowFileInDir(fileOrWorkflowName, verbose, "")
}

func resolveWorkflowFileInDir(fileOrWorkflowName string, verbose bool, workflowDir string) (string, error) {
	// First, try to use it as a direct file path
	if _, err := os.Stat(fileOrWorkflowName); err == nil {
		commandsLog.Printf("Found workflow file at path: %s", fileOrWorkflowName)
		console.LogVerbose(verbose, fmt.Sprintf("Found workflow file at path: %s", fileOrWorkflowName))
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

	// Use provided directory or default
	workflowsDir := workflowDir
	if workflowsDir == "" {
		workflowsDir = getWorkflowsDir()
	}

	// Try to find the workflow in local sources only (not packages)
	_, path, err := readWorkflowFile(workflowPath, workflowsDir)
	if err != nil {
		suggestions := []string{
			fmt.Sprintf("Run '%s status' to see all available workflows", string(constants.CLIExtensionPrefix)),
			fmt.Sprintf("Create a new workflow with '%s new %s'", string(constants.CLIExtensionPrefix), fileOrWorkflowName),
			"Check for typos in the workflow name",
		}

		// Add fuzzy match suggestions
		similarNames := suggestWorkflowNames(fileOrWorkflowName)
		if len(similarNames) > 0 {
			suggestions = append([]string{fmt.Sprintf("Did you mean: %s?", strings.Join(similarNames, ", "))}, suggestions...)
		}

		return "", errors.New(console.FormatErrorWithSuggestions(
			fmt.Sprintf("workflow '%s' not found in local .github/workflows", fileOrWorkflowName),
			suggestions,
		))
	}

	commandsLog.Print("Found workflow in local .github/workflows")

	// Return absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return path, nil // fallback to original path
	}
	return absPath, nil
}

// NewWorkflow creates a new workflow with a two-file structure:
// - .github/workflows/<workflow-name>.md: frontmatter + runtime-import
// - .github/agentics/<workflow-name>.md: markdown body only
func NewWorkflow(workflowName string, verbose bool, force bool) error {
	commandsLog.Printf("Creating new workflow: name=%s, force=%v", workflowName, force)

	// Normalize the workflow name by removing .md extension if present
	// This ensures consistent behavior whether user provides "my-workflow" or "my-workflow.md"
	workflowName = strings.TrimSuffix(workflowName, ".md")
	commandsLog.Printf("Normalized workflow name: %s", workflowName)

	console.LogVerbose(verbose, fmt.Sprintf("Creating new workflow: %s", workflowName))

	// Get current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		commandsLog.Printf("Failed to get working directory: %v", err)
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Create .github/workflows directory if it doesn't exist
	githubWorkflowsDir := filepath.Join(workingDir, constants.GetWorkflowDir())
	commandsLog.Printf("Creating workflows directory: %s", githubWorkflowsDir)

	if err := os.MkdirAll(githubWorkflowsDir, 0755); err != nil {
		commandsLog.Printf("Failed to create workflows directory: %v", err)
		return fmt.Errorf("failed to create .github/workflows directory: %w", err)
	}

	// Create .github/agentics directory if it doesn't exist
	githubAgenticsDir := filepath.Join(workingDir, ".github", "agentics")
	commandsLog.Printf("Creating agentics directory: %s", githubAgenticsDir)

	if err := os.MkdirAll(githubAgenticsDir, 0755); err != nil {
		commandsLog.Printf("Failed to create agentics directory: %v", err)
		return fmt.Errorf("failed to create .github/agentics directory: %w", err)
	}

	// Construct the destination file paths
	workflowFile := filepath.Join(githubWorkflowsDir, workflowName+".md")
	agenticsFile := filepath.Join(githubAgenticsDir, workflowName+".md")
	commandsLog.Printf("Workflow file: %s", workflowFile)
	commandsLog.Printf("Agentics file: %s", agenticsFile)

	// Check if destination files already exist
	if !force {
		if _, err := os.Stat(workflowFile); err == nil {
			commandsLog.Printf("Workflow file already exists and force=false: %s", workflowFile)
			return fmt.Errorf("workflow file '%s' already exists. Use --force to overwrite", workflowFile)
		}
		if _, err := os.Stat(agenticsFile); err == nil {
			commandsLog.Printf("Agentics file already exists and force=false: %s", agenticsFile)
			return fmt.Errorf("agentics file '%s' already exists. Use --force to overwrite", agenticsFile)
		}
	}

	// Create the template content for both files
	workflowContent := createWorkflowConfigTemplate(workflowName)
	agenticsContent := createAgenticsBodyTemplate(workflowName)

	// Write the workflow config file with restrictive permissions (owner-only)
	if err := os.WriteFile(workflowFile, []byte(workflowContent), 0600); err != nil {
		return fmt.Errorf("failed to write workflow file '%s': %w", workflowFile, err)
	}

	// Write the agentics body file with restrictive permissions (owner-only)
	if err := os.WriteFile(agenticsFile, []byte(agenticsContent), 0600); err != nil {
		return fmt.Errorf("failed to write agentics file '%s': %w", agenticsFile, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created new workflow files:")))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Configuration: %s", workflowFile)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  Instructions:  %s", agenticsFile)))
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Next steps:")))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  1. Edit %s to customize agent behavior (no recompile needed)", agenticsFile)))
	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("  2. Run '%s compile %s' to generate the GitHub Actions workflow", string(constants.CLIExtensionPrefix), workflowName)))

	return nil
}

// createWorkflowConfigTemplate generates the workflow configuration file with frontmatter + runtime-import
func createWorkflowConfigTemplate(workflowName string) string {
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
#   schedule: daily  # Fuzzy daily schedule (scattered execution time)
#   # schedule: weekly on monday  # Fuzzy weekly schedule

# Permissions - what can this workflow access?
permissions:
  contents: read

# Outputs - what APIs and tools can the AI use?
safe-outputs:
  create-issue:          # Creates issues (default max: 1)
    max: 5               # Optional: specify maximum number
  # create-agent-session:   # Creates GitHub Copilot agent sessions (max: 1)
  # create-pull-request: # Creates exactly one pull request
  # add-comment:   # Adds comments (default max: 1)
  #   max: 2             # Optional: specify maximum number
  # add-labels:

---

<!-- Edit the file linked below to modify the agent without recompilation. Feel free to move the entire markdown body to that file. -->
{{#runtime-import agentics/` + workflowName + `.md}}
`
}

// createAgenticsBodyTemplate generates the agentics file with markdown body only (no frontmatter)
func createAgenticsBodyTemplate(workflowName string) string {
	return `<!-- This prompt will be imported in the agentic workflow .github/workflows/` + workflowName + `.md at runtime. -->
<!-- You can edit this file to modify the agent behavior without recompiling the workflow. -->

# ` + workflowName + `

Describe what you want the AI to do when this workflow runs.

## Instructions

Replace this section with specific instructions for the AI. For example:

1. Read the issue description and comments
2. Analyze the request and gather relevant information
3. Provide a helpful response or take appropriate action

Be clear and specific about what the AI should accomplish.

## Notes

- Changes to this file take effect immediately on the next workflow run (no recompile needed)
- To change configuration (triggers, permissions, tools), edit .github/workflows/` + workflowName + `.md and recompile
- See https://githubnext.github.io/gh-aw/ for complete configuration options and tools documentation
`
}
