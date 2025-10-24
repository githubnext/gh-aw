package workflow

import (
	"fmt"
	"strings"
)

// generatePRContextPromptStep generates a separate step for PR context instructions
func (c *Compiler) generatePRContextPromptStep(yaml *strings.Builder, data *WorkflowData) {
	// Check if any of the workflow's event triggers are comment-related events
	hasCommentTriggers := c.hasCommentRelatedTriggers(data)

	if !hasCommentTriggers {
		return // No comment-related triggers, skip PR context instructions
	}

	// Also check if checkout step will be added - only show prompt if checkout happens
	needsCheckout := c.shouldAddCheckoutStep(data)
	if !needsCheckout {
		return // No checkout, so no PR branch checkout will happen
	}

	// Check that permissions allow contents read access
	permParser := NewPermissionsParser(data.Permissions)
	if !permParser.HasContentsReadAccess() {
		return // No contents read access, cannot checkout
	}

	// Build the condition string
	condition := BuildPRCommentCondition()

	// Use shared helper but we need to render condition manually since it requires RenderConditionAsIf
	// which is more complex than a simple if: string
	yaml.WriteString("      - name: Append PR context instructions to prompt\n")
	RenderConditionAsIf(yaml, condition, "          ")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_AW_PROMPT: /tmp/gh-aw/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	WritePromptTextToYAML(yaml, prContextPromptText, "          ")
}

// hasCommentRelatedTriggers checks if the workflow has any comment-related event triggers
func (c *Compiler) hasCommentRelatedTriggers(data *WorkflowData) bool {
	// Check for command trigger (which expands to comment events)
	if data.Command != "" {
		return true
	}

	if data.On == "" {
		return false
	}

	// Check for comment-related event types in the "on" configuration
	commentEvents := []string{"issue_comment", "pull_request_review_comment", "pull_request_review"}
	for _, event := range commentEvents {
		if strings.Contains(data.On, event) {
			return true
		}
	}

	return false
}

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
	yaml.WriteString(fmt.Sprintf("        uses: %s\n", GetActionPin("actions/github-script", "v8")))
	yaml.WriteString("        with:\n")
	yaml.WriteString("          script: |\n")

	// Add the JavaScript for checking out the PR branch
	WriteJavaScriptToYAML(yaml, checkoutPRBranchScript)
}
