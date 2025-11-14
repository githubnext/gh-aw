package workflow

import (
	"fmt"
	"strings"
)

// generatePRReadyForReviewCheckout generates a step to checkout the PR branch when PR context is available
func (c *Compiler) generatePRReadyForReviewCheckout(yaml *strings.Builder, data *WorkflowData) {
	// Check that permissions allow contents read access
	permParser := NewPermissionsParser(data.Permissions)
	if !permParser.HasContentsReadAccess() {
		return // No contents read access, cannot checkout
	}

	// Always add the step with a condition that checks if PR context is available
	yaml.WriteString("      - name: Checkout PR branch\n")

	// Build condition that checks if github.event.pull_request exists
	// This will be true for pull_request events and comment events on PRs
	condition := BuildPropertyAccess("github.event.pull_request")
	RenderConditionAsIf(yaml, condition, "          ")

	// Use actions/github-script instead of shell script
	yaml.WriteString(fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script", data)))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Add the JavaScript for checking out the PR branch
	WriteJavaScriptToYAML(yaml, checkoutPRBranchScript)
}
