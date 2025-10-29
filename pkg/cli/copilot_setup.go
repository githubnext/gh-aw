package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/githubnext/gh-aw/pkg/logger"
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

      - name: Install gh CLI
        run: |
          if ! command -v gh &> /dev/null; then
            echo "Installing GitHub CLI..."
            type -p curl >/dev/null || (sudo apt update && sudo apt install curl -y)
            curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg \
            && sudo chmod go+r /usr/share/keyrings/githubcli-archive-keyring.gpg \
            && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null \
            && sudo apt update \
            && sudo apt install gh -y
          else
            echo "GitHub CLI already installed"
          fi

      - name: Install gh-aw extension
        run: |
          # Install or update the gh-aw extension
          if gh extension list | grep -q githubnext/gh-aw; then
            echo "Upgrading gh-aw extension..."
            gh extension upgrade githubnext/gh-aw
          else
            echo "Installing gh-aw extension..."
            gh extension install githubnext/gh-aw
          fi
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
		// File exists, skip creation to avoid overwriting user customizations
		if verbose {
			fmt.Fprintf(os.Stderr, "Skipping %s (already exists)\n", setupStepsPath)
		}
		return nil
	}

	if err := os.WriteFile(setupStepsPath, []byte(copilotSetupStepsYAML), 0644); err != nil {
		return fmt.Errorf("failed to write copilot-setup-steps.yml: %w", err)
	}
	copilotSetupLog.Printf("Created file: %s", setupStepsPath)

	return nil
}
