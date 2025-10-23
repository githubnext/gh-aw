// @ts-check
/// <reference types="@actions/github-script" />

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Read the validated output content from environment variable
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  if (!agentOutputFile) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  // Read agent output from file
  let outputContent;
  try {
    outputContent = require("fs").readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    core.setFailed(`Error reading agent output file: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    return;
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find all update-issue items
  const updateItems = validatedOutput.items.filter(/** @param {any} item */ item => item.type === "update_issue");
  if (updateItems.length === 0) {
    core.info("No update-issue items found in agent output");
    return;
  }

  core.info(`Found ${updateItems.length} update-issue item(s)`);

  // If in staged mode, emit step summary instead of updating issues
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode Preview\n\n";
    summaryContent += "_Would update the following issues:_\n\n";

    for (let i = 0; i < updateItems.length; i++) {
      const item = updateItems[i];
      summaryContent += `### Issue Update ${i + 1}\n\n`;
      if (item.issue_number) {
        summaryContent += `_Target: #${item.issue_number}_\n\n`;
      } else {
        summaryContent += `_Target: Current issue_\n\n`;
      }

      if (item.title !== undefined) {
        summaryContent += `_New title:_ ${item.title}\n\n`;
      }
      if (item.body !== undefined) {
        summaryContent += `${item.body}\n\n`;
      }
      if (item.status !== undefined) {
        summaryContent += `_New status: ${item.status}_\n\n`;
      }
      summaryContent += "---\n\n";
    }

    // Write to step summary
    await core.summary.addRaw(summaryContent).write();
    core.info("📝 Issue update preview written to step summary");
    return;
  }

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
      summaryContent += `- [${issue.title} #${issue.number}](${issue.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully updated ${updatedIssues.length} issue(s)`);
  return updatedIssues;
}
await main();
