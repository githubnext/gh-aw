package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
	"github.com/goccy/go-yaml"
)

var copilotSetupLog = logger.New("cli:copilot_setup")

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
      - name: Checkout code
        uses: actions/checkout@v5

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true

      - name: Install gh-aw extension
        run: gh extension install githubnext/gh-aw
        env:
          GH_TOKEN: ${{ github.token }}

      - name: Build gh-aw from source
        run: |
          echo "Building gh-aw from source for latest features..."
          make build
        continue-on-error: true

      - name: Verify gh-aw installation
        run: |
          gh aw version || ./gh-aw version
`

// WorkflowStep represents a GitHub Actions workflow step
type WorkflowStep struct {
	Name string         `yaml:"name,omitempty"`
	Uses string         `yaml:"uses,omitempty"`
	Run  string         `yaml:"run,omitempty"`
	With map[string]any `yaml:"with,omitempty"`
	Env  map[string]any `yaml:"env,omitempty"`
}

// WorkflowJob represents a GitHub Actions workflow job
type WorkflowJob struct {
	RunsOn      any            `yaml:"runs-on,omitempty"`
	Permissions map[string]any `yaml:"permissions,omitempty"`
	Steps       []WorkflowStep `yaml:"steps,omitempty"`
}

// Workflow represents a GitHub Actions workflow file
type Workflow struct {
	Name string                 `yaml:"name,omitempty"`
	On   any                    `yaml:"on,omitempty"`
	Jobs map[string]WorkflowJob `yaml:"jobs,omitempty"`
}

// ensureCopilotSetupSteps creates or updates .github/workflows/copilot-setup-steps.yml
func ensureCopilotSetupSteps(verbose bool) error {
	copilotSetupLog.Print("Creating copilot-setup-steps.yml")

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

		// Check if the extension install step is already present (quick check)
		contentStr := string(content)
		if strings.Contains(contentStr, "gh extension install githubnext/gh-aw") ||
			strings.Contains(contentStr, "Install gh-aw extension") {
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
		if err := injectExtensionInstallStep(&workflow); err != nil {
			return fmt.Errorf("failed to inject extension install step: %w", err)
		}

		// Marshal back to YAML
		updatedContent, err := yaml.Marshal(&workflow)
		if err != nil {
			return fmt.Errorf("failed to marshal updated workflow: %w", err)
		}

		if err := os.WriteFile(setupStepsPath, updatedContent, 0644); err != nil {
			return fmt.Errorf("failed to update copilot-setup-steps.yml: %w", err)
		}
		copilotSetupLog.Printf("Updated file with extension install step: %s", setupStepsPath)

		if verbose {
			fmt.Fprintf(os.Stderr, "Updated %s with gh-aw extension install step\n", setupStepsPath)
		}
		return nil
	}

	if err := os.WriteFile(setupStepsPath, []byte(copilotSetupStepsYAML), 0644); err != nil {
		return fmt.Errorf("failed to write copilot-setup-steps.yml: %w", err)
	}
	copilotSetupLog.Printf("Created file: %s", setupStepsPath)

	return nil
}

// injectExtensionInstallStep injects the gh-aw extension install step into an existing workflow
func injectExtensionInstallStep(workflow *Workflow) error {
	// Define the extension install step to inject
	extensionStep := WorkflowStep{
		Name: "Install gh-aw extension",
		Run:  "gh extension install githubnext/gh-aw",
		Env: map[string]any{
			"GH_TOKEN": "${{ github.token }}",
		},
	}

	// Find the copilot-setup-steps job
	job, exists := workflow.Jobs["copilot-setup-steps"]
	if !exists {
		return fmt.Errorf("copilot-setup-steps job not found in workflow")
	}

	// Find the position to insert the step (after "Set up Go" or after "Checkout code")
	insertPosition := -1
	for i, step := range job.Steps {
		if strings.Contains(step.Name, "Set up Go") {
			insertPosition = i + 1
			break
		}
	}

	// If Set up Go not found, try after Checkout
	if insertPosition == -1 {
		for i, step := range job.Steps {
			if strings.Contains(step.Name, "Checkout") || strings.Contains(step.Uses, "checkout@") {
				insertPosition = i + 1
				break
			}
		}
	}

	// If still not found, append at the end
	if insertPosition == -1 {
		insertPosition = len(job.Steps)
	}

	// Insert the step at the determined position
	newSteps := make([]WorkflowStep, 0, len(job.Steps)+1)
	newSteps = append(newSteps, job.Steps[:insertPosition]...)
	newSteps = append(newSteps, extensionStep)
	newSteps = append(newSteps, job.Steps[insertPosition:]...)

	job.Steps = newSteps
	workflow.Jobs["copilot-setup-steps"] = job

	return nil
}
