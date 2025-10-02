package workflow

import "strings"

// generatePRBranchCheckout generates a step to checkout the PR branch if the event is a comment on a PR
func (c *Compiler) generatePRBranchCheckout(yaml *strings.Builder, data *WorkflowData) {
	// Check if any of the workflow's event triggers are comment-related events
	hasCommentTriggers := c.hasCommentRelatedTriggers(data)

	if !hasCommentTriggers {
		return // No comment-related triggers, skip PR branch checkout
	}

	// Add a conditional step that checks out the PR branch if applicable
	yaml.WriteString("      - name: Checkout PR branch if applicable\n")
	yaml.WriteString("        if: |\n")

	// Use the helper function to generate the condition expression
	condition := BuildPRCommentCondition()
	conditionStr := condition.Render()

	// Format the condition with proper indentation
	// The condition should be indented with "          " (10 spaces)
	lines := strings.Split(conditionStr, "\n")
	for i, line := range lines {
		if i == 0 {
			yaml.WriteString("          " + line + "\n")
		} else {
			yaml.WriteString("          " + line + "\n")
		}
	}

	yaml.WriteString("        run: |\n")
	yaml.WriteString("          set -e\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          # Determine PR number based on event type\n")
	yaml.WriteString("          if [ \"${{ github.event_name }}\" = \"issue_comment\" ]; then\n")
	yaml.WriteString("            PR_NUMBER=\"${{ github.event.issue.number }}\"\n")
	yaml.WriteString("          elif [ \"${{ github.event_name }}\" = \"pull_request_review_comment\" ]; then\n")
	yaml.WriteString("            PR_NUMBER=\"${{ github.event.pull_request.number }}\"\n")
	yaml.WriteString("          elif [ \"${{ github.event_name }}\" = \"pull_request_review\" ]; then\n")
	yaml.WriteString("            PR_NUMBER=\"${{ github.event.pull_request.number }}\"\n")
	yaml.WriteString("          fi\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          echo \"Fetching PR #$PR_NUMBER...\"\n")
	yaml.WriteString("          gh pr checkout \"$PR_NUMBER\"\n")
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

	yaml.WriteString("      - name: Append PR context instructions to prompt\n")
	yaml.WriteString("        if: |\n")

	// Use the helper function to generate the condition expression
	condition := BuildPRCommentCondition()
	conditionStr := condition.Render()

	// Format the condition with proper indentation
	// The condition should be indented with "          " (10 spaces)
	lines := strings.Split(conditionStr, "\n")
	for i, line := range lines {
		if i == 0 {
			yaml.WriteString("          " + line + "\n")
		} else {
			yaml.WriteString("          " + line + "\n")
		}
	}

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
