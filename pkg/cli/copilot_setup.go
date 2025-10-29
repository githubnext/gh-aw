package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

		contentStr := string(content)

		// Check if the extension install step is already present
		if strings.Contains(contentStr, "gh extension install githubnext/gh-aw") ||
			strings.Contains(contentStr, "Install gh-aw extension") {
			copilotSetupLog.Print("Extension install step already exists, skipping update")
			if verbose {
				fmt.Fprintf(os.Stderr, "Skipping %s (already has gh-aw extension install step)\n", setupStepsPath)
			}
			return nil
		}

		// Inject the extension install step
		copilotSetupLog.Print("Injecting extension install step into existing file")
		updatedContent := injectExtensionInstallStep(contentStr)

		if err := os.WriteFile(setupStepsPath, []byte(updatedContent), 0644); err != nil {
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
func injectExtensionInstallStep(content string) string {
	// Define the extension install step to inject
	extensionStep := `      - name: Install gh-aw extension
        run: gh extension install githubnext/gh-aw
        env:
          GH_TOKEN: ${{ github.token }}`

	// Try to inject after "Set up Go" step
	lines := strings.Split(content, "\n")
	var result []string
	injected := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		result = append(result, line)

		// If we find "Set up Go" and haven't injected yet
		if !injected && strings.Contains(line, "- name: Set up Go") {
			// Find the end of this step (next "- name:" at the same or less indentation)
			stepIndent := len(line) - len(strings.TrimLeft(line, " "))

			j := i + 1
			for j < len(lines) {
				nextLine := lines[j]
				if strings.TrimSpace(nextLine) == "" {
					result = append(result, nextLine)
					j++
					continue
				}

				nextIndent := len(nextLine) - len(strings.TrimLeft(nextLine, " "))
				if strings.HasPrefix(strings.TrimSpace(nextLine), "- name:") && nextIndent <= stepIndent {
					// Found the next step at same level, inject before it
					result = append(result, "")
					result = append(result, extensionStep)
					injected = true
					i = j - 1 // Will be incremented in the loop
					break
				}
				result = append(result, nextLine)
				j++
			}

			// If we reached the end without finding another step
			if j >= len(lines) && !injected {
				result = append(result, "")
				result = append(result, extensionStep)
				injected = true
				break
			}
		}
	}

	if !injected {
		// Fallback: try to inject after checkout step
		result = []string{}
		for i := 0; i < len(lines); i++ {
			line := lines[i]
			result = append(result, line)

			if strings.Contains(line, "- name: Checkout code") || strings.Contains(line, "actions/checkout@") {
				// Find the end of checkout step
				stepIndent := len(line) - len(strings.TrimLeft(line, " "))

				j := i + 1
				for j < len(lines) {
					nextLine := lines[j]
					if strings.TrimSpace(nextLine) == "" {
						result = append(result, nextLine)
						j++
						continue
					}

					nextIndent := len(nextLine) - len(strings.TrimLeft(nextLine, " "))
					if strings.HasPrefix(strings.TrimSpace(nextLine), "- name:") && nextIndent <= stepIndent {
						// Found the next step, inject before it
						result = append(result, "")
						result = append(result, extensionStep)
						injected = true
						i = j - 1
						break
					}
					result = append(result, nextLine)
					j++
				}

				if j >= len(lines) && !injected {
					result = append(result, "")
					result = append(result, extensionStep)
					injected = true
				}
				break
			}
		}
	}

	// If still not injected, append at the end
	if !injected {
		result = append(result, "")
		result = append(result, extensionStep)
	}

	return strings.Join(result, "\n")
}
