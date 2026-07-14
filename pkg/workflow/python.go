package workflow

import (
	"fmt"

	"github.com/github/gh-aw/pkg/logger"
)

var pythonLog = logger.New("workflow:python")

// GenerateUvSetupStep creates a GitHub Actions step for setting up uv.
// Returns a step that installs uv using the pinned action version.
func GenerateUvSetupStep() GitHubActionStep {
	pin := getActionPin("astral-sh/setup-uv")
	if pin == "" {
		pin = "astral-sh/setup-uv@v8"
	}
	return GitHubActionStep{
		"      - name: Setup uv",
		"        uses: " + pin,
	}
}

// GenerateUvToolInstallSteps generates uv tool install steps for a package.
//
// Parameters:
//   - packageName: The package name (e.g., "pydantic-ai[cli]")
//   - version: The package version to install
//   - stepName: The name to display for the install step
//   - binaryName: The binary name installed by the package
//   - verifyCommand: Optional command to verify installation (empty to skip)
//   - verifyStepName: The name for the verify step
//
// Returns steps for installing the package via uv tool install.
func GenerateUvToolInstallSteps(packageName, version, stepName, binaryName, verifyCommand, verifyStepName string) []GitHubActionStep {
	pythonLog.Printf("Generating uv tool install steps: package=%s, version=%s", packageName, version)

	var steps []GitHubActionStep

	// Add uv setup step
	steps = append(steps, GenerateUvSetupStep())

	// Build install command
	var installCmd string
	if version != "" {
		if ExpressionPattern.MatchString(version) {
			// Version is a GitHub Actions expression — use env var for injection safety
			installCmd = fmt.Sprintf(`uv tool install '%s==${{ env.ENGINE_VERSION }}'`, packageName)
			steps = append(steps, GitHubActionStep{
				"      - name: " + stepName,
				"        run: " + installCmd,
				"        env:",
				"          ENGINE_VERSION: " + version,
			})
		} else {
			installCmd = fmt.Sprintf(`uv tool install '%s==%s'`, packageName, version)
			steps = append(steps, GitHubActionStep{
				"      - name: " + stepName,
				"        run: " + installCmd,
			})
		}
	} else {
		installCmd = fmt.Sprintf(`uv tool install '%s'`, packageName)
		steps = append(steps, GitHubActionStep{
			"      - name: " + stepName,
			"        run: " + installCmd,
		})
	}

	// Add verify step if provided
	if verifyCommand != "" && verifyStepName != "" {
		steps = append(steps, GitHubActionStep{
			"      - name: " + verifyStepName,
			"        run: " + verifyCommand,
		})
	}

	return steps
}

// BuildUvEngineInstallStepsWithAWF injects an AWF installation step between the uv
// setup step and the package install steps when the firewall is enabled.
//
// The expected layout of uvSteps is:
//   - uvSteps[0] – uv setup step
//   - uvSteps[1:] – package installation step(s)
//
// Parameters:
//   - uvSteps: Pre-computed uv installation steps (from GenerateUvToolInstallSteps)
//   - workflowData: The workflow data (used to determine firewall configuration)
//
// Returns:
//   - []GitHubActionStep: Steps in order: uv setup, AWF (if enabled), package install
func BuildUvEngineInstallStepsWithAWF(uvSteps []GitHubActionStep, workflowData *WorkflowData) []GitHubActionStep {
	var steps []GitHubActionStep

	if len(uvSteps) > 0 {
		steps = append(steps, uvSteps[0]) // uv setup step
	}

	// Inject AWF installation after uv setup but before the package install steps
	if isFirewallEnabled(workflowData) {
		firewallConfig := getFirewallConfig(workflowData)
		agentConfig := getAgentConfig(workflowData)
		var awfVersion string
		if firewallConfig != nil {
			awfVersion = firewallConfig.Version
		}
		awfInstall := generateAWFInstallationStep(awfVersion, agentConfig)
		if len(awfInstall) > 0 {
			steps = append(steps, awfInstall)
		}
	}

	if len(uvSteps) > 1 {
		steps = append(steps, uvSteps[1:]...)
	}

	return steps
}
