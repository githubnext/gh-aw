import type { SafeOutputItems, UpdateIssueItem } from "./types/safe-outputs";

interface UpdatedIssue {
  number: number;
  title: string;
  html_url: string;
}

async function updateIssueMain(): Promise<UpdatedIssue[]> {
  // Check if we're in staged mode
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return [];
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return [];
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput: SafeOutputItems;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return [];
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return [];
  }

  // Find all update-issue items
  const updateItems = validatedOutput.items.filter(item => item.type === "update-issue") as UpdateIssueItem[];
  if (updateItems.length === 0) {
    core.info("No update-issue items found in agent output");
    return [];
  }

  core.info(`Found ${updateItems.length} update-issue item(s)`);

  // If in staged mode, emit step summary instead of updating issues
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Update Issues Preview\n\n";
    summaryContent += "The following issue updates would be applied if staged mode was disabled:\n\n";

    for (let i = 0; i < updateItems.length; i++) {
      const item = updateItems[i];
      summaryContent += `### Issue Update ${i + 1}\n`;
      if ((item as any).issue_number) {
        summaryContent += `**Target Issue:** #${(item as any).issue_number}\n\n`;
      } else {
        summaryContent += `**Target:** Current issue/PR\n\n`;
      }

      if (item.title) {
        summaryContent += `**New Title:** ${item.title}\n\n`;
      }
      if (item.body) {
        summaryContent += `**New Body:**\n${item.body}\n\n`;
      }
      if ((item as any).state) {
        summaryContent += `**New State:** ${(item as any).state}\n\n`;
      }
      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Issue update preview written to step summary");
    return [];
  }

  // Get the target configuration from environment variable
  const updateTarget = process.env.GITHUB_AW_UPDATE_TARGET || "triggering";
  core.info(`Update target configuration: ${updateTarget}`);

  // Check if we're in an issue or pull request context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment";

  // Validate context based on target configuration
  if (updateTarget === "triggering" && !isIssueContext && !isPRContext) {
    core.info('Target is "triggering" but not running in issue or pull request context, skipping issue updates');
    return [];
  }

  const updatedIssues: UpdatedIssue[] = [];

  // Process each update item
  for (let i = 0; i < updateItems.length; i++) {
    const updateItem = updateItems[i];
    core.info(`Processing update-issue item ${i + 1}/${updateItems.length}`);

    // Determine the issue number for this update
    let issueNumber: number | undefined;

    if (updateTarget === "*") {
      // For target "*", we need an explicit issue number from the update item
      if ((updateItem as any).issue_number) {
        issueNumber = parseInt((updateItem as any).issue_number, 10);
        if (isNaN(issueNumber) || issueNumber <= 0) {
          core.info(`Invalid issue number specified: ${(updateItem as any).issue_number}`);
          continue;
        }
      } else {
        core.info('Target is "*" but no issue_number specified in update item');
        continue;
      }
    } else if (updateTarget && updateTarget !== "triggering") {
      // Explicit issue number specified in target
      issueNumber = parseInt(updateTarget, 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        core.info(`Invalid issue number in target configuration: ${updateTarget}`);
        continue;
      }
    } else {
      // Default behavior: use triggering issue/PR
      if (isIssueContext) {
        if (context.payload.issue) {
          issueNumber = context.payload.issue.number;
        } else {
          core.info("Issue context detected but no issue found in payload");
          continue;
        }
      } else if (isPRContext) {
        if (context.payload.pull_request) {
          issueNumber = context.payload.pull_request.number;
        } else {
          core.info("Pull request context detected but no pull request found in payload");
          continue;
        }
      }
    }

    if (!issueNumber) {
      core.info("Could not determine issue or pull request number");
      continue;
    }

    // Build the update payload
    const updatePayload: any = {
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: issueNumber,
    };

    // Add fields from the update item
    if (updateItem.title) {
      updatePayload.title = updateItem.title.trim();
    }
    if (updateItem.body) {
      updatePayload.body = updateItem.body.trim();
    }
    if ((updateItem as any).state) {
      updatePayload.state = (updateItem as any).state;
    }

    core.info(`Updating issue #${issueNumber}`);
    if (updatePayload.title) {
      core.info(`New title: ${updatePayload.title}`);
    }
    if (updatePayload.body) {
      core.info(`New body length: ${updatePayload.body.length}`);
    }
    if (updatePayload.state) {
      core.info(`New state: ${updatePayload.state}`);
    }

    try {
      // Update the issue using GitHub API
      const { data: issue } = await github.rest.issues.update(updatePayload);

      core.info("Updated issue #" + issue.number + ": " + issue.html_url);
      updatedIssues.push(issue);

      // Set output for the last updated issue (for backward compatibility)
      if (i === updateItems.length - 1) {
        core.setOutput("issue_number", issue.number);
        core.setOutput("issue_url", issue.html_url);
      }
    } catch (error) {
      core.error(`✗ Failed to update issue #${issueNumber}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Write summary for all updated issues
  if (updatedIssues.length > 0) {
    let summaryContent = "\n\n## GitHub Issues Updated\n";
    for (const issue of updatedIssues) {
      summaryContent += `- Issue #${issue.number}: [${issue.title}](${issue.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully updated ${updatedIssues.length} issue(s)`);
  return updatedIssues;
}

(async () => {
  await updateIssueMain();
})();