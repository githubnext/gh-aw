package workflow

import (
	"fmt"
	"strings"

	"github.com/github/gh-aw/pkg/logger"
)

var prLog = logger.New("workflow:pr")

// ShouldGeneratePRCheckoutStep returns true if the checkout-pr step should be generated
// based on the workflow permissions. The step requires contents read access.
func ShouldGeneratePRCheckoutStep(data *WorkflowData) bool {
	permParser := NewPermissionsParser(data.Permissions)
	return permParser.HasContentsReadAccess()
}

// generateSaveBaseGitHubFoldersStep generates step strings (for the activation job) that
// snapshot agent config folders and root instruction files from the workspace into
// /tmp/gh-aw/base/.  The folder and file lists are computed from the engine registry so
// that engine implementations are the single source of truth — no hardcoding in the script.
//
// folders: the agent config directories to snapshot (e.g. ".agents", ".claude", ".github")
// files:   the root instruction files to snapshot (e.g. "AGENTS.md", "CLAUDE.md")
func generateSaveBaseGitHubFoldersStep(folders, files []string) []string {
	var lines []string
	lines = append(lines, "      - name: Save agent config folders for base branch restoration\n")
	lines = append(lines, "        env:\n")
	lines = append(lines, fmt.Sprintf("          GH_AW_AGENT_FOLDERS: \"%s\"\n", strings.Join(folders, " ")))
	lines = append(lines, fmt.Sprintf("          GH_AW_AGENT_FILES: \"%s\"\n", strings.Join(files, " ")))
	lines = append(lines, "        # poutine:ignore untrusted_checkout_exec\n")
	lines = append(lines, "        run: bash \"${RUNNER_TEMP}/gh-aw/actions/save_base_github_folders.sh\"\n")
	return lines
}

// generateRestoreBaseGitHubFoldersStep generates a step (for the agent job) that restores
// agent config from the activation artifact after checkout_pr_branch.cjs has run.
// This prevents fork PRs from injecting malicious skill or instruction files.
// The step also removes .mcp.json and only runs when the PR checkout step succeeded.
//
// folders: the agent config directories to restore (must match save step)
// files:   the root instruction files to restore (must match save step)
func generateRestoreBaseGitHubFoldersStep(yaml *strings.Builder, folders, files []string) {
	prLog.Print("Generating step to restore agent config folders from base branch")
	yaml.WriteString("      - name: Restore agent config folders from base branch\n")
	yaml.WriteString("        if: steps.checkout-pr.outcome == 'success'\n")
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_AW_AGENT_FOLDERS: \"%s\"\n", strings.Join(folders, " "))
	fmt.Fprintf(yaml, "          GH_AW_AGENT_FILES: \"%s\"\n", strings.Join(files, " "))
	yaml.WriteString("        run: bash \"${RUNNER_TEMP}/gh-aw/actions/restore_base_github_folders.sh\"\n")
}

// generatePRReadyForReviewCheckout generates a step to checkout the PR branch when PR context is available
func (c *Compiler) generatePRReadyForReviewCheckout(yaml *strings.Builder, data *WorkflowData) {
	prLog.Print("Generating PR checkout step")
	// Check that permissions allow contents read access
	if !ShouldGeneratePRCheckoutStep(data) {
		prLog.Print("Skipping PR checkout step: no contents read access")
		return // No contents read access, cannot checkout
	}

	// Determine script loading method based on action mode
	setupActionRef := c.resolveActionReference("./actions/setup", data)
	useRequire := setupActionRef != ""

	// Always add the step with a condition that checks if PR context is available
	yaml.WriteString("      - name: Checkout PR branch\n")
	yaml.WriteString("        id: checkout-pr\n")

	// Build condition that checks if github.event.pull_request exists (for pull_request events)
	// OR github.event.issue.pull_request exists (for issue_comment events on PRs).
	// Note: issue_comment events on PRs do NOT set github.event.pull_request; instead
	// github.event.issue.pull_request is set to indicate the issue is a PR.
	condition := BuildOr(
		BuildPropertyAccess("github.event.pull_request"),
		BuildPropertyAccess("github.event.issue.pull_request"),
	)
	RenderConditionAsIf(yaml, condition, "          ")

	// Use actions/github-script instead of shell script
	fmt.Fprintf(yaml, "        uses: %s\n", getCachedActionPin("actions/github-script", data))

	// Add env section with GH_TOKEN for gh CLI
	// Use safe-outputs github-token if available, otherwise default token
	safeOutputsToken := ""
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubToken != "" {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}
	effectiveToken := getEffectiveGitHubToken(safeOutputsToken)
	prLog.Print("PR checkout step configured with GitHub token")
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_TOKEN: %s\n", effectiveToken)

	yaml.WriteString("        with:\n")

	// Add github-token to make it available to the GitHub API client
	fmt.Fprintf(yaml, "          github-token: %s\n", effectiveToken)

	yaml.WriteString("          script: |\n")

	if useRequire {
		// Use require() to load script from copied files using setup_globals helper
		yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
		yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")
		yaml.WriteString("            const { main } = require('" + SetupActionDestination + "/checkout_pr_branch.cjs');\n")
		yaml.WriteString("            await main();\n")
	} else {
		// Inline JavaScript: Attach GitHub Actions builtin objects to global scope before script execution
		yaml.WriteString("            const { setupGlobals } = require('" + SetupActionDestination + "/setup_globals.cjs');\n")
		yaml.WriteString("            setupGlobals(core, github, context, exec, io, getOctokit);\n")

		// Add the JavaScript for checking out the PR branch
		WriteJavaScriptToYAML(yaml, "const { main } = require('${{ runner.temp }}/gh-aw/actions/checkout_pr_branch.cjs'); await main();")
	}
}
