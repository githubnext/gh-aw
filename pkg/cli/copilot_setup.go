package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/githubnext/gh-aw/pkg/workflow"
	"github.com/goccy/go-yaml"
)

var copilotSetupLog = logger.New("cli:copilot_setup")

// generateCopilotSetupStepsYAML generates the copilot-setup-steps.yml content based on action mode
func generateCopilotSetupStepsYAML(actionMode workflow.ActionMode) string {
	if actionMode.IsRelease() {
		// Use the actions/setup-cli action in release mode
		return `name: "Copilot Setup Steps"

# This workflow configures the environment for GitHub Copilot Agent with gh-aw MCP server
on:
  workflow_dispatch:
  push:
    paths:
      - .github/workflows/copilot-setup-steps.yml

jobs:
  # The job MUST be called 'copilot-setup-steps' to be recognized by GitHub Copilot Agent
  copilot-setup-steps:
    runs-on: ubuntu-latest

    # Set minimal permissions for setup steps
    # Copilot Agent receives its own token with appropriate permissions
    permissions:
      contents: read

    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
      - name: Install gh-aw extension
        uses: githubnext/gh-aw/actions/setup-cli@main
        with:
          version: v0.37.18
      - name: Verify gh-aw installation
        run: gh aw version
`
	}

	// Default (dev/script mode): use curl to download install script
	return `name: "Copilot Setup Steps"

# This workflow configures the environment for GitHub Copilot Agent with gh-aw MCP server
on:
  workflow_dispatch:
  push:
    paths:
      - .github/workflows/copilot-setup-steps.yml

jobs:
  # The job MUST be called 'copilot-setup-steps' to be recognized by GitHub Copilot Agent
  copilot-setup-steps:
    runs-on: ubuntu-latest

    # Set minimal permissions for setup steps
    # Copilot Agent receives its own token with appropriate permissions
    permissions:
      contents: read

    steps:
      - name: Install gh-aw extension
        run: |
          curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install-gh-aw.sh | bash
      - name: Verify gh-aw installation
        run: gh aw version
`
}

const copilotSetupStepsYAML = `name: "Copilot Setup Steps"

# This workflow configures the environment for GitHub Copilot Agent with gh-aw MCP server
on:
  workflow_dispatch:
  push:
    paths:
      - .github/workflows/copilot-setup-steps.yml

jobs:
  # The job MUST be called 'copilot-setup-steps' to be recognized by GitHub Copilot Agent
  copilot-setup-steps:
    runs-on: ubuntu-latest

    # Set minimal permissions for setup steps
    # Copilot Agent receives its own token with appropriate permissions
    permissions:
      contents: read

    steps:
      - name: Install gh-aw extension
        run: |
          curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install-gh-aw.sh | bash
      - name: Verify gh-aw installation
        run: gh aw version
`

// CopilotWorkflowStep represents a GitHub Actions workflow step for Copilot setup scaffolding
type CopilotWorkflowStep struct {
	Name string         `yaml:"name,omitempty"`
	Uses string         `yaml:"uses,omitempty"`
	Run  string         `yaml:"run,omitempty"`
	With map[string]any `yaml:"with,omitempty"`
	Env  map[string]any `yaml:"env,omitempty"`
}

// WorkflowJob represents a GitHub Actions workflow job
type WorkflowJob struct {
	RunsOn      any                   `yaml:"runs-on,omitempty"`
	Permissions map[string]any        `yaml:"permissions,omitempty"`
	Steps       []CopilotWorkflowStep `yaml:"steps,omitempty"`
}

// Workflow represents a GitHub Actions workflow file
type Workflow struct {
	Name string                 `yaml:"name,omitempty"`
	On   any                    `yaml:"on,omitempty"`
	Jobs map[string]WorkflowJob `yaml:"jobs,omitempty"`
}

// ensureCopilotSetupSteps creates or updates .github/workflows/copilot-setup-steps.yml
func ensureCopilotSetupSteps(verbose bool, actionMode workflow.ActionMode) error {
	copilotSetupLog.Printf("Creating copilot-setup-steps.yml with action mode: %s", actionMode)

	// Create .github/workflows directory if it doesn't exist
	workflowsDir := filepath.Join(".github", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}
	copilotSetupLog.Printf("Ensured directory exists: %s", workflowsDir)

	// Write copilot-setup-steps.yml
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")

	// Check if file already exists
	if _, err := os.Stat(setupStepsPath); err == nil {
		copilotSetupLog.Printf("File already exists: %s", setupStepsPath)

		// Read existing file to check if extension install step exists
		content, err := os.ReadFile(setupStepsPath)
		if err != nil {
			return fmt.Errorf("failed to read existing copilot-setup-steps.yml: %w", err)
		}

		// Check if the extension install step is already present (check for both modes)
		contentStr := string(content)
		hasLegacyInstall := strings.Contains(contentStr, "install-gh-aw.sh") ||
			(strings.Contains(contentStr, "Install gh-aw extension") && strings.Contains(contentStr, "curl -fsSL"))
		hasActionInstall := strings.Contains(contentStr, "actions/setup-cli")

		if hasLegacyInstall || hasActionInstall {
			copilotSetupLog.Print("Extension install step already exists, skipping update")
			if verbose {
				fmt.Fprintf(os.Stderr, "Skipping %s (already has gh-aw extension install step)\n", setupStepsPath)
			}
			return nil
		}

		// Parse existing workflow
		var workflow Workflow
		if err := yaml.Unmarshal(content, &workflow); err != nil {
			return fmt.Errorf("failed to parse existing copilot-setup-steps.yml: %w", err)
		}

		// Inject the extension install step
		copilotSetupLog.Print("Injecting extension install step into existing file")
		if err := injectExtensionInstallStep(&workflow, actionMode); err != nil {
			return fmt.Errorf("failed to inject extension install step: %w", err)
		}

		// Marshal back to YAML
		updatedContent, err := yaml.Marshal(&workflow)
		if err != nil {
			return fmt.Errorf("failed to marshal updated workflow: %w", err)
		}

		if err := os.WriteFile(setupStepsPath, updatedContent, 0600); err != nil {
			return fmt.Errorf("failed to update copilot-setup-steps.yml: %w", err)
		}
		copilotSetupLog.Printf("Updated file with extension install step: %s", setupStepsPath)

		if verbose {
			fmt.Fprintf(os.Stderr, "Updated %s with gh-aw extension install step\n", setupStepsPath)
		}
		return nil
	}

	if err := os.WriteFile(setupStepsPath, []byte(generateCopilotSetupStepsYAML(actionMode)), 0600); err != nil {
		return fmt.Errorf("failed to write copilot-setup-steps.yml: %w", err)
	}
	copilotSetupLog.Printf("Created file: %s", setupStepsPath)

	return nil
}

// injectExtensionInstallStep injects the gh-aw extension install and verification steps into an existing workflow
func injectExtensionInstallStep(workflow *Workflow, actionMode workflow.ActionMode) error {
	var installStep, checkoutStep CopilotWorkflowStep

	if actionMode.IsRelease() {
		// In release mode, use the actions/setup-cli action
		checkoutStep = CopilotWorkflowStep{
			Name: "Checkout repository",
			Uses: "actions/checkout@v4",
		}
		installStep = CopilotWorkflowStep{
			Name: "Install gh-aw extension",
			Uses: "githubnext/gh-aw/actions/setup-cli@main",
			With: map[string]any{
				"version": "v0.37.18",
			},
		}
	} else {
		// In dev/script mode, use curl to download install script
		installStep = CopilotWorkflowStep{
			Name: "Install gh-aw extension",
			Run:  "curl -fsSL https://raw.githubusercontent.com/githubnext/gh-aw/refs/heads/main/install-gh-aw.sh | bash",
		}
	}

	verifyStep := CopilotWorkflowStep{
		Name: "Verify gh-aw installation",
		Run:  "gh aw version",
	}

	// Find the copilot-setup-steps job
	job, exists := workflow.Jobs["copilot-setup-steps"]
	if !exists {
		return fmt.Errorf("copilot-setup-steps job not found in workflow")
	}

	// Insert the extension install and verify steps at the beginning
	insertPosition := 0

	// Prepare steps to insert based on mode
	var stepsToInsert []CopilotWorkflowStep
	if actionMode.IsRelease() {
		stepsToInsert = []CopilotWorkflowStep{checkoutStep, installStep, verifyStep}
	} else {
		stepsToInsert = []CopilotWorkflowStep{installStep, verifyStep}
	}

	// Insert steps at the determined position
	newSteps := make([]CopilotWorkflowStep, 0, len(job.Steps)+len(stepsToInsert))
	newSteps = append(newSteps, job.Steps[:insertPosition]...)
	newSteps = append(newSteps, stepsToInsert...)
	newSteps = append(newSteps, job.Steps[insertPosition:]...)

	job.Steps = newSteps
	workflow.Jobs["copilot-setup-steps"] = job

	return nil
}
