package workflow

import (
	"fmt"
	"strings"

	"github.com/githubnext/gh-aw/pkg/logger"
)

var prLog = logger.New("workflow:pr")

// generatePRReadyForReviewCheckout generates a step to checkout the PR branch when PR context is available
func (c *Compiler) generatePRReadyForReviewCheckout(yaml *strings.Builder, data *WorkflowData) {
	prLog.Print("Generating PR checkout step")
	// Check that permissions allow contents read access
	permParser := NewPermissionsParser(data.Permissions)
	if !permParser.HasContentsReadAccess() {
		prLog.Print("Skipping PR checkout step: no contents read access")
		return // No contents read access, cannot checkout
	}

	// Always add the step with a condition that checks if PR context is available
	yaml.WriteString("      - name: Checkout PR branch\n")

	// Build condition that checks if github.event.pull_request exists
	// This will be true for pull_request events and comment events on PRs
	condition := BuildPropertyAccess("github.event.pull_request")
	RenderConditionAsIf(yaml, condition, "          ")

	// Use actions/github-script instead of shell script
	fmt.Fprintf(yaml, "        uses: %s\n", GetActionPin("actions/github-script"))

	// Add env section with GH_TOKEN for gh CLI
	// Use safe-outputs github-token if available, otherwise top-level token
	safeOutputsToken := ""
	if data.SafeOutputs != nil && data.SafeOutputs.GitHubToken != "" {
		safeOutputsToken = data.SafeOutputs.GitHubToken
	}
	effectiveToken := getEffectiveGitHubToken(safeOutputsToken, data.GitHubToken)
	prLog.Print("PR checkout step configured with GitHub token")
	yaml.WriteString("        env:\n")
	fmt.Fprintf(yaml, "          GH_TOKEN: %s\n", effectiveToken)

	yaml.WriteString("        with:\n")

	// Add github-token to make it available to the GitHub API client
	fmt.Fprintf(yaml, "          github-token: %s\n", effectiveToken)

	yaml.WriteString("          script: |\n")

	// Add the JavaScript for checking out the PR branch
	WriteJavaScriptToYAML(yaml, checkoutPRBranchScript)
}
