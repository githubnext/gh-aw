package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/logger"
)

var copilotInstallerLog = logger.New("workflow:copilot_installer")

// GenerateCopilotInstallerSteps creates GitHub Actions steps to install the Copilot CLI using the official installer.
func GenerateCopilotInstallerSteps(version, stepName string) []GitHubActionStep {
	// If no version is specified, use the default version from constants
	// This prevents the installer from defaulting to "latest"
	if version == "" {
		version = string(constants.DefaultCopilotVersion)
		copilotInstallerLog.Printf("No version specified, using default: %s", version)
	}

	copilotInstallerLog.Printf("Generating Copilot installer steps using install_copilot_cli.sh: version=%s", version)

	// Use the install_copilot_cli.sh script from actions/setup/sh
	// This script includes retry logic for robustness against transient network failures
	stepLines := []string{
		"      - name: " + stepName,
		"        run: /opt/gh-aw/actions/install_copilot_cli.sh " + version,
	}

	return []GitHubActionStep{GitHubActionStep(stepLines)}
}

// generateSquidLogsUploadStep creates a GitHub Actions step to upload Squid logs as artifact.
func generateSquidLogsUploadStep(workflowName string) GitHubActionStep {
	sanitizedName := strings.ToLower(SanitizeWorkflowName(workflowName))
	artifactName := "firewall-logs-" + sanitizedName
	// Firewall logs are now at a known location in the sandbox folder structure
	firewallLogsDir := "/tmp/gh-aw/sandbox/firewall/logs/"

	stepLines := []string{
		"      - name: Upload Firewall Logs",
		"        if: always()",
		"        continue-on-error: true",
		"        uses: " + GetActionPin("actions/upload-artifact"),
		"        with:",
		"          name: " + artifactName,
		"          path: " + firewallLogsDir,
		"          if-no-files-found: ignore",
	}

	return GitHubActionStep(stepLines)
}

// generateFirewallLogParsingStep creates a GitHub Actions step to parse firewall logs and create step summary.
func generateFirewallLogParsingStep(workflowName string) GitHubActionStep {
	// Firewall logs are at a known location in the sandbox folder structure
	firewallLogsDir := "/tmp/gh-aw/sandbox/firewall/logs"

	stepLines := []string{
		"      - name: Print firewall logs",
		"        if: always()",
		"        continue-on-error: true",
		"        env:",
		"          AWF_LOGS_DIR: " + firewallLogsDir,
		"        run: |",
		"          # Fix permissions on firewall logs so they can be uploaded as artifacts",
		"          # AWF runs with sudo, creating files owned by root",
		fmt.Sprintf("          sudo chmod -R a+r %s 2>/dev/null || true", firewallLogsDir),
		"          # Only run awf logs summary if awf command exists (it may not be installed if workflow failed before install step)",
		"          if command -v awf &> /dev/null; then",
		"            awf logs summary | tee -a \"$GITHUB_STEP_SUMMARY\"",
		"          else",
		"            echo 'AWF binary not installed, skipping firewall log summary'",
		"          fi",
	}

	return GitHubActionStep(stepLines)
}
