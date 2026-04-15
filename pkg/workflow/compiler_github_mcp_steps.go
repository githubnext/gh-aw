package workflow

import (
	"fmt"
	"os"
	"strings"

	"github.com/github/gh-aw/pkg/console"
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
	pinnedAction, err := getActionPinWithData(actionRepo, actionVersion, data)
	if err != nil {
		githubConfigLog.Printf("Failed to resolve %s@%s: %v", actionRepo, actionVersion, err)
		// In strict mode, this error would have been returned by getActionPinWithData
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
		// Support both 'allowed-repos' (preferred) and deprecated 'repos'
		if v, exists := toolConfig["allowed-repos"]; exists {
			configuredRepos = fmt.Sprintf("%v", v)
		} else if v, exists := toolConfig["repos"]; exists {
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

// generateGitHubMCPAppTokenMintingSteps returns the YAML steps to mint a GitHub App token
// for the GitHub MCP server. The steps are generated with id: github-mcp-app-token and
// permissions derived from the agent job's declared permissions plus any extra permissions
// configured under tools.github.github-app.permissions.
//
// The returned steps are added directly to the agent job so that the minted token is
// available as steps.github-mcp-app-token.outputs.token within that job.
// Minting happens inside the agent job (not the activation job) because
// actions/create-github-app-token calls ::add-mask:: on the produced token, and the
// GitHub Actions runner silently drops masked values when used as job outputs (runner v2.308+).
func (c *Compiler) generateGitHubMCPAppTokenMintingSteps(data *WorkflowData) []string {
	// Check if GitHub tool has app configuration
	if data.ParsedTools == nil || data.ParsedTools.GitHub == nil || data.ParsedTools.GitHub.GitHubApp == nil {
		return nil
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

	// Apply extra permissions from github-app.permissions (nested wins over job-level)
	if len(app.Permissions) > 0 {
		githubConfigLog.Printf("Applying %d extra permissions from github-app.permissions", len(app.Permissions))
		for key, val := range app.Permissions {
			scope := convertStringToPermissionScope(key)
			if scope == "" {
				msg := fmt.Sprintf("Unknown permission scope %q in tools.github.github-app.permissions. Valid scopes include: members, organization-administration, team-discussions, organization-members, administration, etc.", key)
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(msg))
				continue
			}
			level := strings.ToLower(strings.TrimSpace(val))
			if level != string(PermissionRead) && level != string(PermissionNone) {
				msg := fmt.Sprintf("Unknown permission level %q for scope %q in tools.github.github-app.permissions. Valid levels are: read, none.", val, key)
				fmt.Fprintln(os.Stderr, console.FormatWarningMessage(msg))
				continue
			}
			permissions.Set(scope, PermissionLevel(level))
		}
	}

	// Generate the token minting step using the existing helper from safe_outputs_app.go
	rawSteps := c.buildGitHubAppTokenMintStep(app, permissions, "")

	// Replace the default step ID with github-mcp-app-token to differentiate it from
	// the safe-outputs app token.
	var steps []string
	for _, step := range rawSteps {
		steps = append(steps, strings.ReplaceAll(step, "id: safe-outputs-app-token", "id: github-mcp-app-token"))
	}
	return steps
}

// generateGitHubMCPAppTokenInvalidationStep generates a step to invalidate the GitHub App token for GitHub MCP server
// This step always runs (even on failure) to ensure tokens are properly cleaned up.
// The token was minted in the agent job and is referenced via steps.github-mcp-app-token.outputs.token.
func (c *Compiler) generateGitHubMCPAppTokenInvalidationStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if GitHub tool has app configuration
	if data.ParsedTools == nil || data.ParsedTools.GitHub == nil || data.ParsedTools.GitHub.GitHubApp == nil {
		return
	}

	githubConfigLog.Print("Generating GitHub App token invalidation step for GitHub MCP server")

	// The token was minted in the agent job; reference it via steps output.
	const tokenExpr = "steps.github-mcp-app-token.outputs.token"

	yaml.WriteString("      - name: Invalidate GitHub App token\n")
	fmt.Fprintf(yaml, "        if: always() && %s != ''\n", tokenExpr)
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          TOKEN: ${{ %s }}\n", tokenExpr)
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          echo \"Revoking GitHub App installation token...\"\n")
	yaml.WriteString("          # GitHub CLI will auth with the token being revoked.\n")
	yaml.WriteString("          gh api \\\n")
	yaml.WriteString("            --method DELETE \\\n")
	yaml.WriteString("            -H \"Authorization: token $TOKEN\" \\\n")
	yaml.WriteString("            /installation/token || echo \"Token revoke may already be expired.\"\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          echo \"Token invalidation step complete.\"\n")
}

// generateParseGuardVarsStep generates a step that parses the blocked-users, trusted-users, and
// approval-labels variables at runtime into proper JSON arrays.
//
// The step is only emitted when explicit guard policies are configured (min-integrity or
// allowed-repos set), because only then does the guard-policies block reference
// `steps.parse-guard-vars.outputs.*`.
//
// The step runs parse_guard_list.sh which:
//   - Accepts GH_AW_BLOCKED_USERS_EXTRA / GH_AW_TRUSTED_USERS_EXTRA / GH_AW_APPROVAL_LABELS_EXTRA
//     for compile-time static items or user-provided expressions.
//   - Accepts GH_AW_BLOCKED_USERS_VAR / GH_AW_TRUSTED_USERS_VAR / GH_AW_APPROVAL_LABELS_VAR for
//     the GH_AW_GITHUB_* org/repo variable fallbacks.
//   - Splits all inputs on commas and newlines, trims whitespace, removes empty entries.
//   - Outputs `blocked_users`, `trusted_users`, and `approval_labels` as JSON arrays via $GITHUB_OUTPUT.
//   - Fails the step if any item is invalid.
func (c *Compiler) generateParseGuardVarsStep(yaml *strings.Builder, data *WorkflowData) {
	githubTool, hasGitHub := data.Tools["github"]
	if !hasGitHub || githubTool == false {
		return
	}

	// Only generate the step when guard policies are configured.
	if len(getGitHubGuardPolicies(githubTool)) == 0 {
		return
	}

	githubConfigLog.Print("Generating parse-guard-vars step for blocked-users, trusted-users and approval-labels")

	// Determine the compile-time static values (or user expression) for each field.
	// These come from the parsed tools config so we don't lose data from the raw map.
	var blockedUsersExtra, trustedUsersExtra, approvalLabelsExtra string

	if data.ParsedTools != nil && data.ParsedTools.GitHub != nil {
		gh := data.ParsedTools.GitHub
		switch {
		case len(gh.BlockedUsers) > 0:
			// Static list from frontmatter — join as comma-separated for the env var.
			blockedUsersExtra = strings.Join(gh.BlockedUsers, ",")
		case gh.BlockedUsersExpr != "":
			// User-provided GitHub Actions expression — passed verbatim; GHA evaluates it.
			blockedUsersExtra = gh.BlockedUsersExpr
		}
		switch {
		case len(gh.TrustedUsers) > 0:
			trustedUsersExtra = strings.Join(gh.TrustedUsers, ",")
		case gh.TrustedUsersExpr != "":
			trustedUsersExtra = gh.TrustedUsersExpr
		}
		switch {
		case len(gh.ApprovalLabels) > 0:
			approvalLabelsExtra = strings.Join(gh.ApprovalLabels, ",")
		case gh.ApprovalLabelsExpr != "":
			approvalLabelsExtra = gh.ApprovalLabelsExpr
		}
	}

	yaml.WriteString("      - name: Parse integrity filter lists\n")
	yaml.WriteString("        id: parse-guard-vars\n")
	yaml.WriteString("        env:\n")

	if blockedUsersExtra != "" {
		fmt.Fprintf(yaml, "          GH_AW_BLOCKED_USERS_EXTRA: %s\n", blockedUsersExtra)
	}
	fmt.Fprintf(yaml, "          GH_AW_BLOCKED_USERS_VAR: ${{ vars.%s || '' }}\n", constants.EnvVarGitHubBlockedUsers)

	if trustedUsersExtra != "" {
		fmt.Fprintf(yaml, "          GH_AW_TRUSTED_USERS_EXTRA: %s\n", trustedUsersExtra)
	}
	fmt.Fprintf(yaml, "          GH_AW_TRUSTED_USERS_VAR: ${{ vars.%s || '' }}\n", constants.EnvVarGitHubTrustedUsers)

	if approvalLabelsExtra != "" {
		fmt.Fprintf(yaml, "          GH_AW_APPROVAL_LABELS_EXTRA: %s\n", approvalLabelsExtra)
	}
	fmt.Fprintf(yaml, "          GH_AW_APPROVAL_LABELS_VAR: ${{ vars.%s || '' }}\n", constants.EnvVarGitHubApprovalLabels)

	yaml.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/parse_guard_list.sh\"\n")
}
