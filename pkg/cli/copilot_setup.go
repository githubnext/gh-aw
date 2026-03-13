package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var copilotSetupLog = logger.New("cli:copilot_setup")

// getActionRef returns the action reference string based on action mode and version.
// If a resolver is provided and mode is release or action, attempts to resolve the SHA for a SHA-pinned reference.
// Falls back to a version tag reference if SHA resolution fails or resolver is nil.
func getActionRef(actionMode workflow.ActionMode, version string, resolver workflow.ActionSHAResolver) string {
	if actionMode.IsRelease() && version != "" && version != "dev" {
		if resolver != nil {
			sha, err := resolver.ResolveSHA("github/gh-aw/actions/setup-cli", version)
			if err == nil && sha != "" {
				return fmt.Sprintf("@%s # %s", sha, version)
			}
			copilotSetupLog.Printf("Failed to resolve SHA for setup-cli@%s: %v, falling back to version tag", version, err)
		}
		return "@" + version
	}
	if actionMode.IsAction() && version != "" && version != "dev" {
		if resolver != nil {
			sha, err := resolver.ResolveSHA("github/gh-aw-actions/setup-cli", version)
			if err == nil && sha != "" {
				return fmt.Sprintf("@%s # %s", sha, version)
			}
			copilotSetupLog.Printf("Failed to resolve SHA for gh-aw-actions/setup-cli@%s: %v, falling back to version tag", version, err)
		}
		return "@" + version
	}
	return "@main"
}

// generateCopilotSetupStepsYAML generates the copilot-setup-steps.yml content based on action mode
func generateCopilotSetupStepsYAML(actionMode workflow.ActionMode, version string, resolver workflow.ActionSHAResolver) string {
	// Determine the action reference - use SHA-pinned or version tag in release/action mode, @main in dev mode
	actionRef := getActionRef(actionMode, version, resolver)

	if actionMode.IsRelease() || actionMode.IsAction() {
		// Determine the action repo based on mode
		actionRepo := "github/gh-aw/actions/setup-cli"
		if actionMode.IsAction() {
			actionRepo = "github/gh-aw-actions/setup-cli"
		}
		return fmt.Sprintf(`name: "Copilot Setup Steps"

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
        uses: actions/checkout@v6
      - name: Install gh-aw extension
        uses: %s%s
        with:
          version: %s
`, actionRepo, actionRef, version)
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
          curl -fsSL https://raw.githubusercontent.com/github/gh-aw/refs/heads/main/install-gh-aw.sh | bash
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
          curl -fsSL https://raw.githubusercontent.com/github/gh-aw/refs/heads/main/install-gh-aw.sh | bash
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
func ensureCopilotSetupSteps(verbose bool, actionMode workflow.ActionMode, version string) error {
	return ensureCopilotSetupStepsWithUpgrade(verbose, actionMode, version, false)
}

// upgradeCopilotSetupSteps upgrades the version in existing copilot-setup-steps.yml
func upgradeCopilotSetupSteps(verbose bool, actionMode workflow.ActionMode, version string) error {
	return ensureCopilotSetupStepsWithUpgrade(verbose, actionMode, version, true)
}

// ensureCopilotSetupStepsWithUpgrade creates .github/workflows/copilot-setup-steps.yml
// If the file already exists, it renders console instructions instead of editing
// When upgradeVersion is true and called from upgrade command, this is a special case
func ensureCopilotSetupStepsWithUpgrade(verbose bool, actionMode workflow.ActionMode, version string, upgradeVersion bool) error {
	copilotSetupLog.Printf("Creating copilot-setup-steps.yml with action mode: %s, version: %s, upgradeVersion: %v", actionMode, version, upgradeVersion)

	// Create a SHA resolver for release/action mode to enable SHA-pinned action references
	var resolver workflow.ActionSHAResolver
	if actionMode.IsRelease() || actionMode.IsAction() {
		cache := workflow.NewActionCache(".")
		_ = cache.Load() // Ignore errors if cache doesn't exist yet
		resolver = workflow.NewActionResolver(cache)
	}

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

		// If we have an install step and upgradeVersion is true, this is from upgrade command
		// In this case, we still update the file for backward compatibility
		if (hasLegacyInstall || hasActionInstall) && upgradeVersion {
			copilotSetupLog.Print("Extension install step exists, attempting version upgrade (upgrade command)")

			upgraded, updatedContent, err := upgradeSetupCliVersionInContent(content, actionMode, version, resolver)
			if err != nil {
				return fmt.Errorf("failed to upgrade setup-cli version: %w", err)
			}

			if !upgraded {
				copilotSetupLog.Print("No version upgrade needed")
				if verbose {
					fmt.Fprintf(os.Stderr, "No version upgrade needed for %s\n", setupStepsPath)
				}
				return nil
			}

			if err := os.WriteFile(setupStepsPath, updatedContent, 0600); err != nil {
				return fmt.Errorf("failed to update copilot-setup-steps.yml: %w", err)
			}
			copilotSetupLog.Printf("Upgraded version in file: %s", setupStepsPath)

			if verbose {
				fmt.Fprintf(os.Stderr, "Updated %s with new version %s\n", setupStepsPath, version)
			}
			return nil
		}

		// File exists - render instructions instead of editing
		if hasLegacyInstall || hasActionInstall {
			copilotSetupLog.Print("Extension install step already exists, file is up to date")
			if verbose {
				fmt.Fprintf(os.Stderr, "Skipping %s (already has gh-aw extension install step)\n", setupStepsPath)
			}
			return nil
		}

		// File exists but needs update - render instructions
		copilotSetupLog.Print("File exists without install step, rendering update instructions instead of editing")
		renderCopilotSetupUpdateInstructions(setupStepsPath, actionMode, version, resolver)
		return nil
	}

	// File doesn't exist - create it
	if err := os.WriteFile(setupStepsPath, []byte(generateCopilotSetupStepsYAML(actionMode, version, resolver)), 0600); err != nil {
		return fmt.Errorf("failed to write copilot-setup-steps.yml: %w", err)
	}
	copilotSetupLog.Printf("Created file: %s", setupStepsPath)

	return nil
}

// renderCopilotSetupUpdateInstructions renders console instructions for updating copilot-setup-steps.yml
func renderCopilotSetupUpdateInstructions(filePath string, actionMode workflow.ActionMode, version string, resolver workflow.ActionSHAResolver) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "%s %s\n",
		"ℹ",
		"Existing file detected: "+filePath)
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "To enable GitHub Copilot Agent integration, please add the following steps")
	fmt.Fprintln(os.Stderr, "to the 'copilot-setup-steps' job in your .github/workflows/copilot-setup-steps.yml file:")
	fmt.Fprintln(os.Stderr)

	// Determine the action reference
	actionRef := getActionRef(actionMode, version, resolver)

	if actionMode.IsRelease() || actionMode.IsAction() {
		actionRepo := "github/gh-aw/actions/setup-cli"
		if actionMode.IsAction() {
			actionRepo = "github/gh-aw-actions/setup-cli"
		}
		fmt.Fprintln(os.Stderr, "      - name: Checkout repository")
		fmt.Fprintln(os.Stderr, "        uses: actions/checkout@v6")
		fmt.Fprintf(os.Stderr, "      - name: Install gh-aw extension\n")
		fmt.Fprintf(os.Stderr, "        uses: %s%s\n", actionRepo, actionRef)
		fmt.Fprintln(os.Stderr, "        with:")
		fmt.Fprintf(os.Stderr, "          version: %s\n", version)
	} else {
		fmt.Fprintln(os.Stderr, "      - name: Install gh-aw extension")
		fmt.Fprintln(os.Stderr, "        run: |")
		fmt.Fprintln(os.Stderr, "          curl -fsSL https://raw.githubusercontent.com/github/gh-aw/refs/heads/main/install-gh-aw.sh | bash")
	}
	fmt.Fprintln(os.Stderr)
}

// setupCliUsesPattern matches the uses: line for either github/gh-aw/actions/setup-cli
// or github/gh-aw-actions/setup-cli.
// It handles unquoted version-tag refs, unquoted SHA-pinned refs (with trailing comment),
// and quoted refs produced by some YAML marshalers (e.g. "...@sha # vX.Y.Z").
var setupCliUsesPattern = regexp.MustCompile(
	`(?m)^(\s+uses:[ \t]*)"?(github/gh-aw(?:-actions)?/(?:actions/)?setup-cli@[^"\n]*)"?([ \t]*)$`)

// upgradeSetupCliVersionInContent replaces the setup-cli action reference and the
// associated version: parameter in the raw YAML content using targeted regex
// substitutions, preserving all other formatting in the file.
//
// Returns (upgraded, updatedContent, error).  upgraded is false when no change
// was required (e.g. already at the target version, or file has no setup-cli step).
func upgradeSetupCliVersionInContent(content []byte, actionMode workflow.ActionMode, version string, resolver workflow.ActionSHAResolver) (bool, []byte, error) {
	if !actionMode.IsRelease() && !actionMode.IsAction() {
		return false, content, nil
	}

	if !setupCliUsesPattern.Match(content) {
		return false, content, nil
	}

	actionRef := getActionRef(actionMode, version, resolver)
	actionRepo := "github/gh-aw/actions/setup-cli"
	if actionMode.IsAction() {
		actionRepo = "github/gh-aw-actions/setup-cli"
	}
	newUses := actionRepo + actionRef

	// Replace the uses: line, stripping any surrounding quotes in the process.
	updated := setupCliUsesPattern.ReplaceAll(content, []byte("${1}"+newUses+"${3}"))

	// Replace the version: value in the with: block immediately following the
	// setup-cli uses: line.  A combined multiline match is used so that only the
	// version: parameter belonging to this specific step is updated.
	// This pattern cannot be pre-compiled at package level because it embeds
	// the runtime value newUses (which varies with version and resolver output).
	escapedNewUses := regexp.QuoteMeta(newUses)
	versionInWithPattern := regexp.MustCompile(
		`(?s)(uses:[ \t]*` + escapedNewUses + `[^\n]*\n(?:[^\n]*\n)*?[ \t]+with:[ \t]*\n(?:[^\n]*\n)*?[ \t]+version:[ \t]*)(\S+)([ \t]*(?:\n|$))`)
	updated = versionInWithPattern.ReplaceAll(updated, []byte("${1}"+version+"${3}"))

	if bytes.Equal(content, updated) {
		return false, content, nil
	}
	return true, updated, nil
}
