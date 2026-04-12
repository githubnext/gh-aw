package workflow

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
	"github.com/github/gh-aw/pkg/logger"
)

var engineFirewallSupportLog = logger.New("workflow:engine_firewall_support")

// hasNetworkRestrictions checks if the workflow has network restrictions defined
// Network restrictions exist if:
// - network.allowed has domains specified (non-empty list) AND it's not just "defaults"
// - network.blocked has domains specified (non-empty list)
func hasNetworkRestrictions(networkPermissions *NetworkPermissions) bool {
	if networkPermissions == nil {
		return false
	}

	// If allowed domains are specified and it's not just the defaults ecosystem, we have restrictions
	if len(networkPermissions.Allowed) > 0 {
		// Check if it's ONLY "defaults" (which means use default ecosystem, not a restriction)
		if len(networkPermissions.Allowed) == 1 && networkPermissions.Allowed[0] == "defaults" {
			return false
		}
		return true
	}

	// Empty allowed list [] means deny-all, which is a restriction
	if networkPermissions.ExplicitlyDefined && len(networkPermissions.Allowed) == 0 {
		return true
	}

	// If blocked domains are specified, we have restrictions
	if len(networkPermissions.Blocked) > 0 {
		return true
	}

	return false
}

// checkNetworkSupport validates that the selected engine supports network restrictions
// when network restrictions are defined in the workflow
func (c *Compiler) checkNetworkSupport(engine CodingAgentEngine, networkPermissions *NetworkPermissions) error {
	engineFirewallSupportLog.Printf("Checking network support: engine=%s, strict_mode=%t", engine.GetID(), c.strictMode)

	// First, check for explicit firewall disable
	if err := c.checkFirewallDisable(networkPermissions); err != nil {
		return err
	}

	// Check if network restrictions exist
	if !hasNetworkRestrictions(networkPermissions) {
		engineFirewallSupportLog.Print("No network restrictions defined, skipping validation")
		// No restrictions, no validation needed
		return nil
	}

	engineFirewallSupportLog.Printf("Engine supports firewall: %s", engine.GetID())
	return nil
}

// checkFirewallDisable validates firewall: "disable" configuration
// - Warning if allowed != * (unrestricted)
// - Error in strict mode if allowed is not *
func (c *Compiler) checkFirewallDisable(networkPermissions *NetworkPermissions) error {
	if networkPermissions == nil || networkPermissions.Firewall == nil {
		return nil
	}

	// Check if firewall is explicitly disabled
	if !networkPermissions.Firewall.Enabled {
		// Check if network has restrictions (allowed list specified with domains)
		hasRestrictions := len(networkPermissions.Allowed) > 0

		if hasRestrictions {
			message := "Firewall is disabled but network restrictions are specified (network.allowed). Network may not be properly sandboxed."

			if c.strictMode {
				// In strict mode, this is an error
				return errors.New("strict mode: cannot disable firewall when network restrictions (network.allowed) are set")
			}

			// In non-strict mode, emit a warning
			fmt.Fprintln(os.Stderr, console.FormatWarningMessage(message))
			c.IncrementWarningCount()
		}
	}

	return nil
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

// defaultGetSquidLogsSteps returns the steps for uploading and parsing Squid logs after
// secret redaction. It is shared across engines (Claude, Codex, Copilot) whose
// GetSquidLogsSteps implementations are otherwise identical save for the logger used.
func defaultGetSquidLogsSteps(workflowData *WorkflowData, log *logger.Logger) []GitHubActionStep {
	var steps []GitHubActionStep

	// Only add upload and parsing steps if firewall is enabled
	if isFirewallEnabled(workflowData) {
		log.Printf("Adding Squid logs upload and parsing steps for workflow: %s", workflowData.Name)

		squidLogsUpload := generateSquidLogsUploadStep(workflowData.Name)
		steps = append(steps, squidLogsUpload)

		// Add firewall log parsing step to create step summary
		firewallLogParsing := generateFirewallLogParsingStep(workflowData.Name)
		steps = append(steps, firewallLogParsing)
	} else {
		log.Print("Firewall disabled, skipping Squid logs upload")
	}

	return steps
}
