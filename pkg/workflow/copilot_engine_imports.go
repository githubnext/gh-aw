// This file implements the ImportsProvider interface for the Copilot engine.
//
// The Copilot engine supports native plugin installation via the `copilot` CLI
// before the main agent execution step.  The GitHub Actions token is passed as
// GITHUB_TOKEN so the CLI can authenticate with GitHub-hosted registries.
//
// CLI command emitted:
//   - Plugin: copilot plugin install <spec>
//
// Supported plugin spec formats (see https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-plugin-reference):
//   - plugin@marketplace      — plugin from a registered marketplace
//   - OWNER/REPO              — root of a GitHub repository
//   - OWNER/REPO:PATH/TO/PLUGIN — subdirectory in a repository
//   - https://github.com/o/r.git — any Git URL

package workflow

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
