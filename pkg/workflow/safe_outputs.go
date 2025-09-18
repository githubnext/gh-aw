package workflow

import "strings"

// HasSafeOutputsEnabled checks if any safe-outputs are enabled
func HasSafeOutputsEnabled(safeOutputs *SafeOutputsConfig) bool {
	return safeOutputs.CreateIssues != nil ||
		safeOutputs.CreateDiscussions != nil ||
		safeOutputs.AddComments != nil ||
		safeOutputs.CreatePullRequests != nil ||
		safeOutputs.CreatePullRequestReviewComments != nil ||
		safeOutputs.CreateCodeScanningAlerts != nil ||
		safeOutputs.AddLabels != nil ||
		safeOutputs.UpdateIssues != nil ||
		safeOutputs.PushToPullRequestBranch != nil ||
		safeOutputs.MissingTool != nil
}

// generateSafeOutputsPromptSection generates the safe-outputs instruction section for prompts
// when safe-outputs are configured, informing the agent about available output capabilities
func generateSafeOutputsPromptSection(yaml *strings.Builder, safeOutputs *SafeOutputsConfig) {
	if safeOutputs == nil {
		return
	}

	// Add output instructions for all engines (GITHUB_AW_SAFE_OUTPUTS functionality)
	yaml.WriteString("          \n")
	yaml.WriteString("          ---\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          ## ")
	written := false
	if safeOutputs.AddComments != nil {
		yaml.WriteString("Adding a Comment to an Issue or Pull Request")
		written = true
	}
	if safeOutputs.CreateIssues != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Creating an Issue")
	}
	if safeOutputs.CreatePullRequests != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Creating a Pull Request")
	}

	if safeOutputs.AddLabels != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Adding Labels to Issues or Pull Requests")
		written = true
	}

	if safeOutputs.UpdateIssues != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Updating Issues")
		written = true
	}

	if safeOutputs.PushToPullRequestBranch != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Pushing Changes to Branch")
		written = true
	}

	if safeOutputs.CreateCodeScanningAlerts != nil {
		if written {
			yaml.WriteString(", ")
		}
		yaml.WriteString("Creating Code Scanning Alert")
		written = true
	}

	// Missing-tool is always available
	if written {
		yaml.WriteString(", ")
	}
	yaml.WriteString("Reporting Missing Tools or Functionality")

	yaml.WriteString("\n")
	yaml.WriteString("          \n")
	yaml.WriteString("          **IMPORTANT**: To do the actions mentioned in the header of this section, use the **safe-outputs** tools, do NOT attempt to use `gh`, do NOT attempt to use the GitHub API. You don't have write access to the GitHub repo.\n")
	yaml.WriteString("          \n")

	if safeOutputs.AddComments != nil {
		yaml.WriteString("          **Adding a Comment to an Issue or Pull Request**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To add a comment to an issue or pull request, use the add-comments tool from the safe-outputs MCP\n")
		yaml.WriteString("          \n")
	}

	if safeOutputs.CreateIssues != nil {
		yaml.WriteString("          **Creating an Issue**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To create an issue, use the create-issue tool from the safe-outputs MCP\n")
		yaml.WriteString("          \n")
	}

	if safeOutputs.CreatePullRequests != nil {
		yaml.WriteString("          **Creating a Pull Request**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To create a pull request:\n")
		yaml.WriteString("          1. Make any file changes directly in the working directory\n")
		yaml.WriteString("          2. If you haven't done so already, create a local branch using an appropriate unique name\n")
		yaml.WriteString("          3. Add and commit your changes to the branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Check you haven't deleted or changed any files you didn't intend to.\n")
		yaml.WriteString("          4. Do not push your changes. That will be done by the tool.\n")
		yaml.WriteString("          5. Create the pull request with the create-pull-request tool from the safe-outputs MCP\n")
		yaml.WriteString("          \n")
	}

	if safeOutputs.AddLabels != nil {
		yaml.WriteString("          **Adding Labels to Issues or Pull Requests**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To add labels to an issue or a pull request, use the add-labels tool from the safe-outputs MCP\n")
		yaml.WriteString("          \n")
	}

	if safeOutputs.UpdateIssues != nil {
		yaml.WriteString("          **Updating an Issue**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To udpate an issue, use the update-issue tool from the safe-outputs MCP\n")
		yaml.WriteString("          \n")
	}

	if safeOutputs.PushToPullRequestBranch != nil {
		yaml.WriteString("          **Pushing Changes to Pull Request Branch**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To push changes to the branch of a pull request:\n")
		yaml.WriteString("          1. Make any file changes directly in the working directory\n")
		yaml.WriteString("          2. Add and commit your changes to the local copy of the pull request branch. Be careful to add exactly the files you intend, and check there are no extra files left un-added. Check you haven't deleted or changed any files you didn't intend to.\n")
		yaml.WriteString("          3. Push the branch to the repo by using the push-to-pr-branch tool from the safe-outputs MCP\n")
		yaml.WriteString("          \n")
	}

	if safeOutputs.CreateCodeScanningAlerts != nil {
		yaml.WriteString("          **Creating Code Scanning Alert**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To create code scanning alert use the create-code-scanning-alert tool from the safe-outputs MCP\n")
		yaml.WriteString("          \n")
	}

	// Missing-tool instructions are only included when configured
	if safeOutputs.MissingTool != nil {
		yaml.WriteString("          **Reporting Missing Tools or Functionality**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To report a missing tool use the missing-tool tool from the safe-outputs MCP.\n")
		yaml.WriteString("          \n")
	}

	if safeOutputs.CreatePullRequestReviewComments != nil {
		yaml.WriteString("          **Creating a Pull Request Review Comment**\n")
		yaml.WriteString("          \n")
		yaml.WriteString("          To create a pull request review comment, use the create-pull-request-review-comment tool from the safe-outputs MCP\n")
		yaml.WriteString("          \n")
	}
}
