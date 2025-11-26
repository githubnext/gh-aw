package workflow

import (
	"fmt"

	"github.com/githubnext/gh-aw/pkg/constants"
	"github.com/githubnext/gh-aw/pkg/logger"
)

var playwrightBrowserInstallLog = logger.New("workflow:playwright_browser_install")

// HasPlaywrightMCPServer checks if the workflow uses the Playwright MCP server.
// This is used to determine if browser pre-installation is needed.
func HasPlaywrightMCPServer(workflowData *WorkflowData) bool {
	if workflowData == nil {
		return false
	}

	// Check parsed tools first (strongly-typed)
	if workflowData.ParsedTools != nil && workflowData.ParsedTools.Playwright != nil {
		playwrightBrowserInstallLog.Print("Playwright tool found in ParsedTools")
		return true
	}

	// Fallback to raw tools map
	if workflowData.Tools != nil {
		if _, exists := workflowData.Tools["playwright"]; exists {
			playwrightBrowserInstallLog.Print("Playwright tool found in Tools map")
			return true
		}
	}

	return false
}

// ShouldPreinstallPlaywrightBrowsers checks if Playwright browsers should be pre-installed.
// Returns true if:
// - Playwright MCP server is configured in the workflow
// - Firewall is enabled (where browser install would fail due to network restrictions)
func ShouldPreinstallPlaywrightBrowsers(workflowData *WorkflowData) bool {
	if !HasPlaywrightMCPServer(workflowData) {
		playwrightBrowserInstallLog.Print("Playwright not configured, skipping pre-installation check")
		return false
	}

	if !isFirewallEnabled(workflowData) {
		playwrightBrowserInstallLog.Print("Firewall not enabled, skipping pre-installation (install will happen at runtime)")
		return false
	}

	playwrightBrowserInstallLog.Print("Playwright pre-installation needed (firewall enabled)")
	return true
}

// GeneratePlaywrightBrowserInstallStep creates a GitHub Actions step to pre-install
// Playwright browsers before the firewall starts. This prevents timeout issues
// when the Playwright MCP server tries to install browsers through the firewall.
//
// The step installs only Chromium with --with-deps to minimize installation time
// while ensuring system dependencies are also installed.
func GeneratePlaywrightBrowserInstallStep() GitHubActionStep {
	// Use the default Playwright MCP version for compatibility
	playwrightVersion := string(constants.DefaultPlaywrightMCPVersion)

	playwrightBrowserInstallLog.Printf("Generating Playwright browser install step with version %s", playwrightVersion)

	stepLines := []string{
		"      - name: Pre-install Playwright browsers",
		"        run: |",
		fmt.Sprintf("          echo \"Installing Playwright browsers (version %s)...\"", playwrightVersion),
		"          # Install Playwright browsers with system dependencies before firewall starts",
		"          # This prevents timeout issues when Playwright tries to install through the firewall",
		fmt.Sprintf("          npx playwright@%s install --with-deps chromium", playwrightVersion),
		"          echo \"Playwright browsers installed successfully\"",
	}

	return GitHubActionStep(stepLines)
}
