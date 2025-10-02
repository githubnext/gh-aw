package workflow

import "strings"

// renderConditionAsIf renders a ConditionNode as an 'if' condition with proper YAML indentation
func renderConditionAsIf(yaml *strings.Builder, condition ConditionNode, indent string) {
	yaml.WriteString("        if: |\n")
	conditionStr := condition.Render()

	// Format the condition with proper indentation
	lines := strings.Split(conditionStr, "\n")
	for _, line := range lines {
		yaml.WriteString(indent + line + "\n")
	}
}

// generatePRBranchCheckout generates a step to checkout the PR branch if the event is a comment on a PR
func (c *Compiler) generatePRBranchCheckout(yaml *strings.Builder, data *WorkflowData) {
	// Check if any of the workflow's event triggers are comment-related events
	hasCommentTriggers := c.hasCommentRelatedTriggers(data)

	if !hasCommentTriggers {
		return // No comment-related triggers, skip PR branch checkout
	}

	// Check that permissions allow contents read access
	permParser := NewPermissionsParser(data.Permissions)
	if !permParser.HasContentsReadAccess() {
		return // No contents read access, cannot checkout
	}

	// Add a conditional step that checks out the PR branch if applicable
	yaml.WriteString("      - name: Checkout PR branch if applicable\n")

	// Use the helper function to render the condition
	condition := BuildPRCommentCondition()
	renderConditionAsIf(yaml, condition, "          ")

	yaml.WriteString("        run: |\n")
	WriteShellScriptToYAML(yaml, checkoutPRScript, "          ")
	yaml.WriteString("        env:\n")
	yaml.WriteString("          GH_TOKEN: ${{ github.token }}\n")
}

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

	yaml.WriteString("      - name: Append PR context instructions to prompt\n")

	// Use the helper function to render the condition
	condition := BuildPRCommentCondition()
	renderConditionAsIf(yaml, condition, "          ")

	yaml.WriteString("        env:\n")
	yaml.WriteString("          GITHUB_AW_PROMPT: /tmp/aw-prompts/prompt.txt\n")
	yaml.WriteString("        run: |\n")
	yaml.WriteString("          cat >> $GITHUB_AW_PROMPT << 'EOF'\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          ---\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          ## Current Branch Context\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          **IMPORTANT**: This workflow was triggered by a comment on a pull request. The repository has been automatically checked out to the PR's branch, not the default branch.\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          ### What This Means\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          - The current working directory contains the code from the pull request branch\n")
	yaml.WriteString("          - Any file operations you perform will be on the PR branch code\n")
	yaml.WriteString("          - You can inspect, analyze, and work with the PR changes directly\n")
	yaml.WriteString("          - The PR branch has been checked out using `gh pr checkout`\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          ### Available Actions\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          You can:\n")
	yaml.WriteString("          - Review the changes in the PR by examining files\n")
	yaml.WriteString("          - Run tests or linters on the PR code\n")
	yaml.WriteString("          - Make additional changes to the PR branch if needed\n")
	yaml.WriteString("          - Create commits on the PR branch (they will appear in the PR)\n")
	yaml.WriteString("          - Switch to other branches using `git checkout` if needed\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          ### Current Branch Information\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          To see which branch you're currently on, you can run:\n")
	yaml.WriteString("          ```bash\n")
	yaml.WriteString("          git branch --show-current\n")
	yaml.WriteString("          git log -1 --oneline\n")
	yaml.WriteString("          ```\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          EOF\n")
}

// hasCommentRelatedTriggers checks if the workflow has any comment-related event triggers
func (c *Compiler) hasCommentRelatedTriggers(data *WorkflowData) bool {
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
