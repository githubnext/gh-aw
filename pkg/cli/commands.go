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

// NewWorkflow creates a new workflow markdown file with template content
// If quick is true, creates only the workflow file; otherwise creates workflow + comprehensive README
func NewWorkflow(workflowName string, verbose bool, force bool, quick bool) error {
	commandsLog.Printf("Creating new workflow: name=%s, force=%v, quick=%v", workflowName, force, quick)

	// Normalize the workflow name by removing .md extension if present
	// This ensures consistent behavior whether user provides "my-workflow" or "my-workflow.md"
	workflowName = strings.TrimSuffix(workflowName, ".md")
	commandsLog.Printf("Normalized workflow name: %s", workflowName)

	console.LogVerbose(verbose, fmt.Sprintf("Creating new workflow: %s", workflowName))

	// Get current working directory for .github/workflows
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

	// Construct the destination file path
	destFile := filepath.Join(githubWorkflowsDir, workflowName+".md")
	commandsLog.Printf("Destination file: %s", destFile)

	// Check if destination file already exists
	if _, err := os.Stat(destFile); err == nil && !force {
		commandsLog.Printf("Workflow file already exists and force=false: %s", destFile)
		return fmt.Errorf("workflow file '%s' already exists. Use --force to overwrite", destFile)
	}

	// Create the template content
	template := createWorkflowTemplate(workflowName)

	// Write the template to file with restrictive permissions (owner-only)
	if err := os.WriteFile(destFile, []byte(template), 0600); err != nil {
		return fmt.Errorf("failed to write workflow file '%s': %w", destFile, err)
	}

	fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created new workflow: %s", destFile)))

	// Generate documentation based on mode
	if !quick {
		// Comprehensive mode: Create detailed README with examples
		readmePath := filepath.Join(githubWorkflowsDir, workflowName+"-README.md")
		commandsLog.Printf("Creating comprehensive README: %s", readmePath)
		
		readmeContent := createComprehensiveReadme(workflowName)
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0600); err != nil {
			// Non-fatal: log warning but continue
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create README: %v", err)))
			commandsLog.Printf("Failed to create README: %v", err)
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created documentation: %s", readmePath)))
		}
	} else {
		// Quick mode: Create minimal README
		readmePath := filepath.Join(githubWorkflowsDir, workflowName+"-README.md")
		commandsLog.Printf("Creating quick README: %s", readmePath)
		
		readmeContent := createQuickReadme(workflowName)
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0600); err != nil {
			// Non-fatal: log warning but continue
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(fmt.Sprintf("Failed to create README: %v", err)))
			commandsLog.Printf("Failed to create README: %v", err)
		} else {
			fmt.Fprintln(os.Stderr, console.FormatSuccessMessage(fmt.Sprintf("Created minimal README: %s", readmePath)))
		}
	}

	fmt.Fprintln(os.Stderr, console.FormatInfoMessage(fmt.Sprintf("Edit the file to customize your workflow, then run '%s compile' to generate the GitHub Actions workflow", string(constants.CLIExtensionPrefix))))

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
#   schedule: daily  # Fuzzy daily schedule (scattered execution time)
#   # schedule: weekly on monday  # Fuzzy weekly schedule

# Permissions - what can this workflow access?
permissions:
  contents: read
  issues: write
  pull-requests: write

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

# ` + workflowName + `

Describe what you want the AI to do when this workflow runs.

## Instructions

Replace this section with specific instructions for the AI. For example:

1. Read the issue description and comments
2. Analyze the request and gather relevant information
3. Provide a helpful response or take appropriate action

Be clear and specific about what the AI should accomplish.

## Notes

- Run ` + "`" + string(constants.CLIExtensionPrefix) + " compile`" + ` to generate the GitHub Actions workflow
- See https://githubnext.github.io/gh-aw/ for complete configuration options and tools documentation
`
}

// createQuickReadme generates a minimal README for the workflow
func createQuickReadme(workflowName string) string {
	cliPrefix := string(constants.CLIExtensionPrefix)
	backtick := "`"
	return fmt.Sprintf(`# %s

## Quick Start

1. Edit %s.github/workflows/%s.md%s to customize the workflow
2. Run %s%s compile%s to generate the GitHub Actions workflow
3. Commit and push the changes

## What This Workflow Does

[Describe the purpose of this workflow]

## Triggering the Workflow

This workflow can be triggered:
- Manually via workflow_dispatch
- [Add other triggers as configured]

## Documentation

For complete documentation, see https://githubnext.github.io/gh-aw/
`, workflowName, backtick, workflowName, backtick, backtick, cliPrefix, backtick)
}

// createComprehensiveReadme generates a detailed README for the workflow
func createComprehensiveReadme(workflowName string) string {
	cliPrefix := string(constants.CLIExtensionPrefix)
	backtick := "`"
	tripleBacktick := "```"
	return fmt.Sprintf(`# %s

## Overview

[Provide a comprehensive description of what this workflow does, its purpose, and when it should be used]

## Features

- [Feature 1]
- [Feature 2]
- [Feature 3]

## Setup

### Prerequisites

- Repository must be initialized for agentic workflows (run %s%s init%s)
- Required permissions configured in workflow frontmatter
- Any necessary secrets or environment variables set

### Installation

1. Edit the workflow file at %s.github/workflows/%s.md%s
2. Configure the triggers, permissions, and safe-outputs as needed
3. Compile the workflow:
   %sbash
   %s compile %s
   %s
4. Commit and push the changes

## Configuration

### Triggers

The workflow can be configured with various triggers in the frontmatter:

- **workflow_dispatch**: Manual trigger (default)
- **issues**: Trigger on issue events (opened, edited, etc.)
- **pull_request**: Trigger on PR events (opened, synchronize, etc.)
- **schedule**: Trigger on a schedule (daily, weekly, etc.)

Example:
%syaml
on:
  issues:
    types: [opened, edited]
  workflow_dispatch:
%s

### Permissions

Configure the minimum required permissions for your workflow:

%syaml
permissions:
  contents: read
  issues: write
  pull-requests: write
%s

### Safe Outputs

Define what actions the AI can take:

%syaml
safe-outputs:
  create-issue:
    max: 5
  add-comment:
    max: 2
%s

## Usage

### Manual Trigger

Navigate to Actions → %s → Run workflow

### Automatic Trigger

The workflow will run automatically based on the configured triggers.

## Workflow Behavior

[Describe in detail what the workflow does when triggered, what inputs it processes, and what outputs it produces]

## Examples

### Example 1: Basic Usage

[Provide a concrete example of how to use this workflow]

### Example 2: Advanced Configuration

[Provide an advanced example with additional configuration]

## Troubleshooting

### Common Issues

**Issue**: Workflow fails to compile
- **Solution**: Check the frontmatter syntax and ensure all required fields are present

**Issue**: Workflow runs but doesn't produce expected output
- **Solution**: Check the workflow logs with %s%s logs %s%s

### Debugging

Use the audit command to investigate workflow runs:

%sbash
%s logs %s
%s audit <run-id>
%s

## Best Practices

1. **Start with minimal permissions**: Only grant what the workflow needs
2. **Use safe-outputs**: Prefer structured outputs over raw permissions
3. **Test thoroughly**: Run the workflow manually before enabling automatic triggers
4. **Monitor usage**: Keep an eye on workflow runs and costs
5. **Document behavior**: Keep this README updated with workflow changes

## Additional Resources

- [GitHub Agentic Workflows Documentation](https://githubnext.github.io/gh-aw/)
- [Workflow Examples](https://github.com/githubnext/gh-aw/tree/main/.github/workflows)
- [Security Best Practices](https://githubnext.github.io/gh-aw/security/)

## Contributing

To modify this workflow:

1. Edit %s.github/workflows/%s.md%s
2. Update this README if behavior changes
3. Recompile: %s%s compile %s%s
4. Test the changes
5. Commit and create a pull request

## License

[Your license information]
`, workflowName, backtick, cliPrefix, backtick, backtick, workflowName, backtick, tripleBacktick, cliPrefix, workflowName, tripleBacktick,
		tripleBacktick, tripleBacktick, tripleBacktick, tripleBacktick, tripleBacktick, tripleBacktick, workflowName,
		backtick, cliPrefix, workflowName, backtick, tripleBacktick, cliPrefix, workflowName, cliPrefix, tripleBacktick,
		backtick, workflowName, backtick, backtick, cliPrefix, workflowName, backtick)
}
