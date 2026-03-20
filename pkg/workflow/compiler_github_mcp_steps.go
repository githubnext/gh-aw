package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/constants"
)

// generateGitHubMCPLockdownDetectionStep generates a step to determine automatic guard policy
// for GitHub MCP server based on repository visibility.
// This step is added when:
//   - GitHub tool is enabled AND
//   - guard policy (repos/min-integrity) is not fully configured in the workflow
//
// For public repositories, the step automatically sets min-integrity to "approved" and
// repos to "all" if they are not already configured.
// This applies regardless of whether a GitHub App token is configured, because repo-scoping
// is not a substitute for author-integrity filtering inside a repository.
func (c *Compiler) generateGitHubMCPLockdownDetectionStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if GitHub tool is present
	githubTool, hasGitHub := data.Tools["github"]
	if !hasGitHub || githubTool == false {
		return
	}

	// Skip when guard policy is already fully configured in the workflow.
	// The step is only needed to auto-configure guard policies for public repos.
	if len(getGitHubGuardPolicies(githubTool)) > 0 {
		githubConfigLog.Print("Guard policy already configured in workflow, skipping automatic guard policy determination")
		return
	}

	githubConfigLog.Print("Generating automatic guard policy determination step for GitHub MCP server")

	// Resolve the latest version of actions/github-script
	actionRepo := "actions/github-script"
	actionVersion := string(constants.DefaultGitHubScriptVersion)
	pinnedAction, err := GetActionPinWithData(actionRepo, actionVersion, data)
	if err != nil {
		githubConfigLog.Printf("Failed to resolve %s@%s: %v", actionRepo, actionVersion, err)
		// In strict mode, this error would have been returned by GetActionPinWithData
		// In normal mode, we fall back to using the version tag without pinning
		pinnedAction = fmt.Sprintf("%s@%s", actionRepo, actionVersion)
	}

	// Extract current guard policy configuration to pass as env vars so the step can
	// detect whether each field is already configured and avoid overriding it.
	configuredMinIntegrity := ""
	configuredRepos := ""
	if toolConfig, ok := githubTool.(map[string]any); ok {
		if v, exists := toolConfig["min-integrity"]; exists {
			configuredMinIntegrity = fmt.Sprintf("%v", v)
		}
		if v, exists := toolConfig["repos"]; exists {
			configuredRepos = fmt.Sprintf("%v", v)
		}
	}

	// Generate the step using the determine_automatic_lockdown.cjs action
	yaml.WriteString("      - name: Determine automatic lockdown mode for GitHub MCP Server\n")
	yaml.WriteString("        id: determine-automatic-lockdown\n")
	fmt.Fprintf(yaml, "        uses: %s\n", pinnedAction)
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_GITHUB_TOKEN: ${{ secrets.GH_AW_GITHUB_TOKEN }}\n")
	yaml.WriteString("          GH_AW_GITHUB_MCP_SERVER_TOKEN: ${{ secrets.GH_AW_GITHUB_MCP_SERVER_TOKEN }}\n")
	if configuredMinIntegrity != "" {
		fmt.Fprintf(yaml, "          GH_AW_GITHUB_MIN_INTEGRITY: %s\n", configuredMinIntegrity)
	}
	if configuredRepos != "" {
		fmt.Fprintf(yaml, "          GH_AW_GITHUB_REPOS: %s\n", configuredRepos)
	}
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")
	yaml.WriteString("            const determineAutomaticLockdown = require('${{ runner.temp }}/gh-aw/actions/determine_automatic_lockdown.cjs');\n")
	yaml.WriteString("            await determineAutomaticLockdown(github, context, core);\n")
}

// generateGitHubMCPAppTokenMintingStep generates a step to mint a GitHub App token for GitHub MCP server
// This step is added when:
// - GitHub tool is enabled with app configuration
// The step mints an installation access token with permissions matching the agent job permissions
func (c *Compiler) generateGitHubMCPAppTokenMintingStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if GitHub tool has app configuration
	if data.ParsedTools == nil || data.ParsedTools.GitHub == nil || data.ParsedTools.GitHub.GitHubApp == nil {
		return
	}

	app := data.ParsedTools.GitHub.GitHubApp
	githubConfigLog.Printf("Generating GitHub App token minting step for GitHub MCP server: app-id=%s", app.AppID)

	// Get permissions from the agent job - parse from YAML string
	var permissions *Permissions
	if data.Permissions != "" {
		parser := NewPermissionsParser(data.Permissions)
		permissions = parser.ToPermissions()
	} else {
		githubConfigLog.Print("No permissions specified, using empty permissions")
		permissions = NewPermissions()
	}

	// Generate the token minting step using the existing helper from safe_outputs_app.go
	steps := c.buildGitHubAppTokenMintStep(app, permissions, "")

	// Modify the step ID to differentiate from safe-outputs app token
	// Replace "safe-outputs-app-token" with "github-mcp-app-token"
	for _, step := range steps {
		modifiedStep := strings.ReplaceAll(step, "id: safe-outputs-app-token", "id: github-mcp-app-token")
		yaml.WriteString(modifiedStep)
	}
}

// generateGitHubMCPAppTokenInvalidationStep generates a step to invalidate the GitHub App token for GitHub MCP server
// This step always runs (even on failure) to ensure tokens are properly cleaned up
func (c *Compiler) generateGitHubMCPAppTokenInvalidationStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if GitHub tool has app configuration
	if data.ParsedTools == nil || data.ParsedTools.GitHub == nil || data.ParsedTools.GitHub.GitHubApp == nil {
		return
	}

	githubConfigLog.Print("Generating GitHub App token invalidation step for GitHub MCP server")

	// Generate the token invalidation step using the existing helper from safe_outputs_app.go
	steps := c.buildGitHubAppTokenInvalidationStep()

	// Modify the step references to use github-mcp-app-token instead of safe-outputs-app-token
	for _, step := range steps {
		modifiedStep := strings.ReplaceAll(step, "steps.safe-outputs-app-token.outputs.token", "steps.github-mcp-app-token.outputs.token")
		yaml.WriteString(modifiedStep)
	}
}
