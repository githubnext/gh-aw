package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/fileutil"
	"github.com/github/gh-aw/pkg/gitutil"
	"github.com/github/gh-aw/pkg/logger"
	"github.com/github/gh-aw/pkg/workflow"
)

var copilotSetupLog = logger.New("cli:copilot_setup")

// installScriptTempPath is the temporary file path used for the downloaded gh-aw install script.
const installScriptTempPath = "/tmp/gh-aw/install-gh-aw.sh"

// copilotSetupStepsStaticSHA is the pinned commit SHA of install-gh-aw.sh used in the static
// YAML test template and as the fallback when ResolveGhAwRef is unavailable.
// Run scripts/update-install-script-hashes.sh to refresh both this value and copilotSetupStepsStaticSHA256.
const copilotSetupStepsStaticSHA = "21a6827c430f89d3b7443074cfc8bd25b84d278f"

// copilotSetupStepsStaticSHA256 is the SHA256 hex digest of install-gh-aw.sh at copilotSetupStepsStaticSHA.
// Run scripts/update-install-script-hashes.sh to refresh both this value and copilotSetupStepsStaticSHA.
const copilotSetupStepsStaticSHA256 = "248ccebcb998c6a506548156e1bf9f02429cbbaec407d5adbdfd316ab0f866a0"

// sha256HexRegex matches a valid lowercase SHA256 hex digest (exactly 64 hex chars).
var sha256HexRegex = regexp.MustCompile(`^[0-9a-f]{64}$`)

// resolveInstallScriptSHA256 fetches install-gh-aw.sh at the given immutable commit SHA
// and returns its SHA256 hex digest for use in a sha256sum integrity check.
// Returns an empty string and logs a warning if the fetch or computation fails.
func resolveInstallScriptSHA256(ctx context.Context, commitSHA string) string {
	if !gitutil.IsValidFullSHA(commitSHA) {
		copilotSetupLog.Printf("resolveInstallScriptSHA256: commitSHA %q is not a valid full SHA, skipping", commitSHA)
		return ""
	}
	scriptURL := fmt.Sprintf("https://raw.githubusercontent.com/github/gh-aw/%s/install-gh-aw.sh", commitSHA)
	res, err := FetchImportURL(ctx, scriptURL, FetchOptions{})
	if err != nil {
		copilotSetupLog.Printf("Could not fetch install-gh-aw.sh from %s to compute SHA256: %v", scriptURL, err)
		return ""
	}
	h := sha256.Sum256(res.Body)
	return hex.EncodeToString(h[:])
}

// sha256CheckLine returns a YAML-indented shell command (with trailing newline) that verifies
// digest against path using sha256sum. Returns an empty string if either parameter is invalid.
// digest must be a 64-char lowercase hex string (sha256HexRegex); path must contain only
// safe filesystem characters (no spaces, quotes, or shell metacharacters).
func sha256CheckLine(digest, path string) string {
	if !sha256HexRegex.MatchString(digest) {
		return ""
	}
	if path == "" || strings.ContainsAny(path, " \t\n\"'\\$`;&|<>(){}*?") {
		copilotSetupLog.Printf("sha256CheckLine: unsafe path %q, skipping", path)
		return ""
	}
	return fmt.Sprintf(`          echo "%s  %s" | sha256sum -c -`+"\n", digest, path)
}

// If a resolver is provided and mode is release or action, attempts to resolve the SHA for a SHA-pinned reference.
// Falls back to a version tag reference if SHA resolution fails or resolver is nil.
func getActionRef(ctx context.Context, actionMode workflow.ActionMode, version string, resolver workflow.SHAResolver) string {
	if actionMode.IsRelease() && version != "" && version != "dev" {
		if resolver != nil {
			sha, err := resolver.ResolveSHA(ctx, "github/gh-aw-actions/setup-cli", version)
			if err == nil && sha != "" {
				return fmt.Sprintf("@%s # %s", sha, version)
			}
			copilotSetupLog.Printf("Failed to resolve SHA for setup-cli@%s: %v, falling back to version tag", version, err)
		}
		return "@" + version
	}
	if actionMode.IsAction() && version != "" && version != "dev" {
		if resolver != nil {
			sha, err := resolver.ResolveSHA(ctx, "github/gh-aw-actions/setup-cli", version)
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
func generateCopilotSetupStepsYAML(ctx context.Context, actionMode workflow.ActionMode, version string, resolver workflow.SHAResolver) string {
	// Determine the action reference - use SHA-pinned or version tag in release/action mode, @main in dev mode
	actionRef := getActionRef(ctx, actionMode, version, resolver)

	if actionMode.IsRelease() || actionMode.IsAction() {
		// Determine the action repo based on mode
		actionRepo := "github/gh-aw-actions/setup-cli"
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

	// Default (dev/script mode): try to resolve the main branch to a pinned SHA so the
	// downloaded script is immutable; fall back to the mutable branch ref if unavailable.
	installRef := "refs/heads/main"
	installSHA256 := ""
	if sha, err := workflow.ResolveGhAwRef(ctx, "main"); err == nil && sha != "" {
		installRef = sha
		// Fetch the script to compute an explicit SHA256 integrity check line.
		installSHA256 = resolveInstallScriptSHA256(ctx, sha)
	} else {
		copilotSetupLog.Printf("Could not resolve github/gh-aw main SHA for dev-mode template, falling back to mutable ref: %v", err)
	}
	sha256Cmd := sha256CheckLine(installSHA256, installScriptTempPath)
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
      - name: Install gh-aw extension
        run: |
          mkdir -p /tmp/gh-aw
          curl -fsSL https://raw.githubusercontent.com/github/gh-aw/%s/install-gh-aw.sh -o %s
%s          bash %s
`, installRef, installScriptTempPath, sha256Cmd, installScriptTempPath)
}

// copilotSetupStepsYAML is a static dev-mode template used only for YAML validity tests.
// It is built from copilotSetupStepsStaticSHA and copilotSetupStepsStaticSHA256 so that
// scripts/update-install-script-hashes.sh can refresh both values in a single place.
// The runtime function generateCopilotSetupStepsYAML resolves the ref dynamically via ResolveGhAwRef.
var copilotSetupStepsYAML = fmt.Sprintf(`name: "Copilot Setup Steps"

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
          mkdir -p /tmp/gh-aw
          curl -fsSL https://raw.githubusercontent.com/github/gh-aw/%s/install-gh-aw.sh -o %s
          echo "%s  %s" | sha256sum -c -
          bash %s
`, copilotSetupStepsStaticSHA, installScriptTempPath, copilotSetupStepsStaticSHA256, installScriptTempPath, installScriptTempPath)

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
func ensureCopilotSetupSteps(ctx context.Context, verbose bool, actionMode workflow.ActionMode, version string) error {
	return ensureCopilotSetupStepsWithUpgrade(ctx, verbose, actionMode, version, false)
}

// upgradeCopilotSetupSteps upgrades the version in existing copilot-setup-steps.yml
func upgradeCopilotSetupSteps(ctx context.Context, verbose bool, actionMode workflow.ActionMode, version string) error {
	return ensureCopilotSetupStepsWithUpgrade(ctx, verbose, actionMode, version, true)
}

// ensureCopilotSetupStepsWithUpgrade creates .github/workflows/copilot-setup-steps.yml
// If the file already exists, it renders console instructions instead of editing
// When upgradeVersion is true and called from upgrade command, this is a special case
func ensureCopilotSetupStepsWithUpgrade(ctx context.Context, verbose bool, actionMode workflow.ActionMode, version string, upgradeVersion bool) error {
	copilotSetupLog.Printf("Creating copilot-setup-steps.yml with action mode: %s, version: %s, upgradeVersion: %v", actionMode, version, upgradeVersion)

	// Create a SHA resolver for release/action mode to enable SHA-pinned action references
	var resolver workflow.SHAResolver
	if actionMode.IsRelease() || actionMode.IsAction() {
		cache := workflow.NewActionCache(".")
		_ = cache.Load() // Ignore errors if cache doesn't exist yet
		resolver = workflow.NewActionResolver(cache)
	}

	workflowsDir := constants.GetWorkflowDir()
	setupStepsPath := filepath.Join(workflowsDir, "copilot-setup-steps.yml")
	if err := fileutil.EnsureParentDir(setupStepsPath, constants.DirPermPublic); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}
	copilotSetupLog.Printf("Ensured directory exists: %s", workflowsDir)

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

			upgraded, updatedContent, err := upgradeSetupCliVersionInContent(ctx, content, actionMode, version, resolver)
			if err != nil {
				return fmt.Errorf("failed to upgrade setup-cli version: %w", err)
			}

			if !upgraded {
				copilotSetupLog.Print("No version upgrade needed")
				if verbose {
					fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr("No version upgrade needed for "+setupStepsPath))
				}
				return nil
			}

			if err := os.WriteFile(setupStepsPath, updatedContent, constants.FilePermSensitive); err != nil {
				return fmt.Errorf("failed to update copilot-setup-steps.yml: %w", err)
			}
			copilotSetupLog.Printf("Upgraded version in file: %s", setupStepsPath)

			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatSuccessMessageStderr(fmt.Sprintf("Updated %s with new version %s", setupStepsPath, version)))
			}
			return nil
		}

		// File exists - render instructions instead of editing
		if hasLegacyInstall || hasActionInstall {
			copilotSetupLog.Print("Extension install step already exists, file is up to date")
			if verbose {
				fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr(fmt.Sprintf("Skipping %s (already has gh-aw extension install step)", setupStepsPath)))
			}
			return nil
		}

		// File exists but needs update - render instructions
		copilotSetupLog.Print("File exists without install step, rendering update instructions instead of editing")
		renderCopilotSetupUpdateInstructions(ctx, setupStepsPath, actionMode, version, resolver)
		return nil
	}

	// File doesn't exist - create it
	if err := os.WriteFile(setupStepsPath, []byte(generateCopilotSetupStepsYAML(ctx, actionMode, version, resolver)), constants.FilePermSensitive); err != nil {
		return fmt.Errorf("failed to write copilot-setup-steps.yml: %w", err)
	}
	copilotSetupLog.Printf("Created file: %s", setupStepsPath)

	return nil
}

// renderCopilotSetupUpdateInstructions renders console instructions for updating copilot-setup-steps.yml
func renderCopilotSetupUpdateInstructions(ctx context.Context, filePath string, actionMode workflow.ActionMode, version string, resolver workflow.SHAResolver) {
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, console.FormatInfoMessageStderr("Existing file detected: "+filePath))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "To enable GitHub Copilot Agent integration, please add the following steps")
	fmt.Fprintln(os.Stderr, "to the 'copilot-setup-steps' job in your .github/workflows/copilot-setup-steps.yml file:")
	fmt.Fprintln(os.Stderr)

	// Determine the action reference
	actionRef := getActionRef(ctx, actionMode, version, resolver)

	if actionMode.IsRelease() || actionMode.IsAction() {
		actionRepo := "github/gh-aw-actions/setup-cli"
		fmt.Fprintln(os.Stderr, "      - name: Checkout repository")
		fmt.Fprintln(os.Stderr, "        uses: actions/checkout@v6")
		fmt.Fprintln(os.Stderr, "      - name: Install gh-aw extension")
		fmt.Fprintln(os.Stderr, "        uses: "+actionRepo+actionRef)
		fmt.Fprintln(os.Stderr, "        with:")
		fmt.Fprintln(os.Stderr, "          version: "+version)
	} else {
		// Dev/script mode: try to resolve main to a pinned SHA so the instructions emit an
		// immutable URL; fall back to the mutable branch ref if resolution is unavailable.
		installRef := "refs/heads/main"
		installSHA256 := ""
		if sha, err := workflow.ResolveGhAwRef(ctx, "main"); err == nil && sha != "" {
			installRef = sha
			installSHA256 = resolveInstallScriptSHA256(ctx, sha)
		}
		fmt.Fprintln(os.Stderr, "      - name: Install gh-aw extension")
		fmt.Fprintln(os.Stderr, "        run: |")
		fmt.Fprintln(os.Stderr, "          mkdir -p /tmp/gh-aw")
		fmt.Fprintln(os.Stderr, "          curl -fsSL https://raw.githubusercontent.com/github/gh-aw/"+installRef+"/install-gh-aw.sh -o "+installScriptTempPath)
		if line := sha256CheckLine(installSHA256, installScriptTempPath); line != "" {
			fmt.Fprint(os.Stderr, line) // sha256CheckLine includes trailing newline
		}
		fmt.Fprintln(os.Stderr, "          bash "+installScriptTempPath)
	}
	fmt.Fprintln(os.Stderr)
}

// setupCliUsesPattern matches the uses: line for either github/gh-aw/actions/setup-cli
// or github/gh-aw-actions/setup-cli.
// It handles unquoted version-tag refs, unquoted SHA-pinned refs (with trailing comment),
// and quoted refs produced by some YAML marshalers (e.g. "...@sha # vX.Y.Z").
var setupCliUsesPattern = regexp.MustCompile(
	`(?m)^(\s+uses:[ \t]*)"?(github/gh-aw(?:-actions)?/(?:actions/)?setup-cli@[^"\n]*)"?([ \t]*)$`)

// versionInWithPattern matches the version: parameter in the with: block that immediately
// follows any setup-cli uses: line (any ref format: version tag, SHA-pinned, or quoted).
// It is anchored to the same action repos as setupCliUsesPattern so that it only updates
// the version belonging to the setup-cli step, but is independent of the exact ref value.
// This allows it to correct pre-existing drift where the uses: comment and with: version:
// were already out of sync before the upgrade was run.
//
// Pattern breakdown:
//
//	[ \t]+uses:[ \t]*         — indented uses: key with optional surrounding spaces
//	"?github/gh-aw(?:-actions)?/(?:actions/)?setup-cli@[^"\n]*"?
//	                           — any setup-cli ref (version tag, SHA+comment, or quoted)
//	[^\n]*\n                   — rest of the uses: line (e.g. trailing spaces)
//	(?:[^\n]*\n)*?             — zero or more lines between uses: and with: (non-greedy)
//	[ \t]+with:[ \t]*\n        — indented with: key
//	(?:[^\n]*\n)*?             — zero or more lines between with: and version: (non-greedy)
//	[ \t]+version:[ \t]*       — indented version: key (final part of the prefix captured as group 1)
//	(\S+)                      — the version value (captured as group 2)
//	([ \t]*(?:\n|$))           — trailing whitespace and line terminator (captured as group 3)
//
// Note: In the full pattern, group 1 wraps the entire prefix from the setup-cli `uses:` line
// through the `version:` key (and following spaces), group 2 is just the version value, and
// group 3 is the trailing whitespace plus the line terminator.
var versionInWithPattern = regexp.MustCompile(
	`(?s)([ \t]+uses:[ \t]*"?github/gh-aw(?:-actions)?/(?:actions/)?setup-cli@[^"\n]*"?[^\n]*\n(?:[^\n]*\n)*?[ \t]+with:[ \t]*\n(?:[^\n]*\n)*?[ \t]+version:[ \t]*)(\S+)([ \t]*(?:\n|$))`)

// upgradeSetupCliVersionInContent replaces the setup-cli action reference and the
// associated version: parameter in the raw YAML content using targeted regex
// substitutions, preserving all other formatting in the file.
//
// Returns (upgraded, updatedContent, error).  upgraded is false when no change
// was required (e.g. already at the target version, or file has no setup-cli step).
func upgradeSetupCliVersionInContent(ctx context.Context, content []byte, actionMode workflow.ActionMode, version string, resolver workflow.SHAResolver) (bool, []byte, error) {
	if !actionMode.IsRelease() && !actionMode.IsAction() {
		return false, content, nil
	}

	if !setupCliUsesPattern.Match(content) {
		return false, content, nil
	}

	actionRef := getActionRef(ctx, actionMode, version, resolver)
	actionRepo := "github/gh-aw-actions/setup-cli"
	newUses := actionRepo + actionRef

	// Replace the uses: line, stripping any surrounding quotes in the process.
	updated := setupCliUsesPattern.ReplaceAll(content, []byte("${1}"+newUses+"${3}"))

	// Replace the version: value in the with: block immediately following the
	// setup-cli uses: line.  versionInWithPattern matches any valid setup-cli
	// reference so it succeeds even when there was pre-existing drift between
	// the uses: comment and the version: parameter before the upgrade was run.
	updated = versionInWithPattern.ReplaceAll(updated, []byte("${1}"+version+"${3}"))

	if bytes.Equal(content, updated) {
		return false, content, nil
	}
	return true, updated, nil
}
