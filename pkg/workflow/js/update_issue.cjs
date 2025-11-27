// @ts-check
/// <reference types="@actions/github-script" />

const { runSafeOutput } = require("./safe_output_runner.cjs");

/**
 * Render function for staged preview
 * @param {any} item - The update_issue item
 * @param {number} index - Index of the item
 * @returns {string} Markdown content for the preview
 */
function renderUpdateIssuePreview(item, index) {
  let content = `### Issue Update ${index + 1}\n`;
  if (item.issue_number) {
    content += `**Target Issue:** #${item.issue_number}\n\n`;
  } else {
    content += `**Target:** Current issue\n\n`;
  }

  if (item.title !== undefined) {
    content += `**New Title:** ${item.title}\n\n`;
  }
  if (item.body !== undefined) {
    content += `**New Body:**\n${item.body}\n\n`;
  }
  if (item.status !== undefined) {
    content += `**New Status:** ${item.status}\n\n`;
  }
  return content;
}

/**
 * Process update_issue items
 * @param {any[]} updateItems - The update_issue items to process
 */
async function processUpdateIssueItems(updateItems) {
  // Get the configuration from environment variables
  const updateTarget = process.env.GH_AW_UPDATE_TARGET || "triggering";
  const canUpdateStatus = process.env.GH_AW_UPDATE_STATUS === "true";
  const canUpdateTitle = process.env.GH_AW_UPDATE_TITLE === "true";
  const canUpdateBody = process.env.GH_AW_UPDATE_BODY === "true";

  core.info(`Update target configuration: ${updateTarget}`);
  core.info(`Can update status: ${canUpdateStatus}, title: ${canUpdateTitle}, body: ${canUpdateBody}`);

  // Check if we're in an issue context
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";

  // Validate context based on target configuration
  if (updateTarget === "triggering" && !isIssueContext) {
    core.info('Target is "triggering" but not running in issue context, skipping issue update');
    return;
  }

  const updatedIssues = [];

  // Process each update item
  for (let i = 0; i < updateItems.length; i++) {
    const updateItem = updateItems[i];
    core.info(`Processing update-issue item ${i + 1}/${updateItems.length}`);

    // Determine the issue number for this update
    let issueNumber;

    if (updateTarget === "*") {
      // For target "*", we need an explicit issue number from the update item
      if (updateItem.issue_number) {
        issueNumber = parseInt(updateItem.issue_number, 10);
        if (isNaN(issueNumber) || issueNumber <= 0) {
          core.info(`Invalid issue number specified: ${updateItem.issue_number}`);
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
      // Default behavior: use triggering issue
      if (isIssueContext) {
        if (context.payload.issue) {
          issueNumber = context.payload.issue.number;
        } else {
          core.info("Issue context detected but no issue found in payload");
          continue;
        }
      } else {
        core.info("Could not determine issue number");
        continue;
      }
    }

    if (!issueNumber) {
      core.info("Could not determine issue number");
      continue;
    }

    core.info(`Updating issue #${issueNumber}`);

    // Build the update object based on allowed fields and provided values
    /** @type {any} */
    const updateData = {};
    let hasUpdates = false;

    if (canUpdateStatus && updateItem.status !== undefined) {
      // Validate status value
      if (updateItem.status === "open" || updateItem.status === "closed") {
        updateData.state = updateItem.status;
        hasUpdates = true;
        core.info(`Will update status to: ${updateItem.status}`);
      } else {
        core.info(`Invalid status value: ${updateItem.status}. Must be 'open' or 'closed'`);
      }
    }

    if (canUpdateTitle && updateItem.title !== undefined) {
      if (typeof updateItem.title === "string" && updateItem.title.trim().length > 0) {
        updateData.title = updateItem.title.trim();
        hasUpdates = true;
        core.info(`Will update title to: ${updateItem.title.trim()}`);
      } else {
        core.info("Invalid title value: must be a non-empty string");
      }
    }

    if (canUpdateBody && updateItem.body !== undefined) {
      if (typeof updateItem.body === "string") {
        updateData.body = updateItem.body;
        hasUpdates = true;
        core.info(`Will update body (length: ${updateItem.body.length})`);
      } else {
        core.info("Invalid body value: must be a string");
      }
    }

    if (!hasUpdates) {
      core.info("No valid updates to apply for this item");
      continue;
    }

    try {
      // Update the issue using GitHub API
      const { data: issue } = await github.rest.issues.update({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: issueNumber,
        ...updateData,
      });

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
    let summaryContent = "\n\n## Updated Issues\n";
    for (const issue of updatedIssues) {
      summaryContent += `- Issue #${issue.number}: [${issue.title}](${issue.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully updated ${updatedIssues.length} issue(s)`);
  return updatedIssues;
}

async function main() {
  await runSafeOutput({
    itemType: "update_issue",
    itemTypePlural: "update-issue",
    stagedTitle: "Update Issues",
    stagedDescription: "The following issue updates would be applied if staged mode was disabled:",
    renderStagedItem: renderUpdateIssuePreview,
    processItems: processUpdateIssueItems,
  });
}

await main();
