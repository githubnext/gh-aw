// This file implements the ImportsProvider interface for the Copilot engine.
//
// The Copilot engine supports native marketplace registration and plugin installation
// via the `copilot` CLI before the main agent execution step.  The GitHub Actions
// token is passed as GITHUB_TOKEN so the CLI can authenticate with GitHub-hosted
// registries.
//
// CLI commands emitted:
//   - Marketplace: copilot plugin marketplace add <url>
//   - Plugin:      copilot plugin install <name>
//
// Built-in marketplaces (pre-registered by the Copilot CLI, not re-registered):
//   - https://github.com/github/copilot-plugins

package workflow

// copilotBuiltinMarketplaces lists the marketplace URLs that the Copilot CLI
// already registers by default.  Attempting to add these again with
// "copilot plugin marketplace add" results in an error, so they are filtered
// out before emitting marketplace setup steps.
var copilotBuiltinMarketplaces = []string{
	"https://github.com/github/copilot-plugins",
}

// GetBuiltinMarketplaces returns the marketplace URLs that are already
// pre-registered in the Copilot CLI and must not be re-registered.
func (e *CopilotEngine) GetBuiltinMarketplaces() []string {
	return copilotBuiltinMarketplaces
}

// GetMarketplaceSetupSteps returns GitHub Actions steps that register marketplace
// URLs with the Copilot CLI before agent execution.
func (e *CopilotEngine) GetMarketplaceSetupSteps(marketplaces []string, workflowData *WorkflowData) []GitHubActionStep {
	if len(marketplaces) == 0 {
		return nil
	}

	var steps []GitHubActionStep
	for _, url := range marketplaces {
		step := GitHubActionStep{
			`      - name: "Register Copilot marketplace: ` + url + `"`,
			"        env:",
			"          GITHUB_TOKEN: ${{ github.token }}",
			"        run: copilot plugin marketplace add " + url,
		}
		steps = append(steps, step)
	}
	return steps
}

// GetPluginInstallSteps returns GitHub Actions steps that install Copilot plugins
// via the `copilot plugin install` command before agent execution.
func (e *CopilotEngine) GetPluginInstallSteps(plugins []string, workflowData *WorkflowData) []GitHubActionStep {
	if len(plugins) == 0 {
		return nil
	}

	var steps []GitHubActionStep
	for _, name := range plugins {
		step := GitHubActionStep{
			`      - name: "Install Copilot plugin: ` + name + `"`,
			"        env:",
			"          GITHUB_TOKEN: ${{ github.token }}",
			"        run: copilot plugin install " + name,
		}
		steps = append(steps, step)
	}
	return steps
}
