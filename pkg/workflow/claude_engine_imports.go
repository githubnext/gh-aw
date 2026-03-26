// This file implements the ImportsProvider interface for the Claude engine.
//
// The Claude engine supports native plugin installation via the `claude` CLI
// before the main agent execution step.  The GitHub Actions token is passed as
// GITHUB_TOKEN so the CLI can authenticate with GitHub-hosted registries.
//
// CLI command emitted:
//   - Plugin: claude plugin install <name>

package workflow

// GetPluginInstallSteps returns GitHub Actions steps that install Claude plugins
// via the `claude plugin install` command before agent execution.
func (e *ClaudeEngine) GetPluginInstallSteps(plugins []string, workflowData *WorkflowData) []GitHubActionStep {
	if len(plugins) == 0 {
		return nil
	}

	var steps []GitHubActionStep
	for _, name := range plugins {
		step := GitHubActionStep{
			`      - name: "Install Claude plugin: ` + name + `"`,
			"        env:",
			"          GITHUB_TOKEN: ${{ github.token }}",
			"        run: claude plugin install " + name,
		}
		steps = append(steps, step)
	}
	return steps
}
