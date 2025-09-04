async function main() {
  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    console.log("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    return;
  }

  if (outputContent.trim() === "") {
    console.log("Agent output content is empty");
    return;
  }

  console.log("Agent output content length:", outputContent.length);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    console.log(
      "Error parsing agent output JSON:",
      error instanceof Error ? error.message : String(error)
    );
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    console.log("No valid items found in agent output");
    return;
  }

  // Find all update-issue items
  const updateItems = validatedOutput.items.filter(
    /** @param {any} item */ item => item.type === "update-issue"
  );
  if (updateItems.length === 0) {
    console.log("No update-issue items found in agent output");
    return;
  }

  console.log(`Found ${updateItems.length} update-issue item(s)`);

  // Get the configuration from environment variables
  const updateTarget = process.env.GITHUB_AW_UPDATE_TARGET || "triggering";
  const canUpdateStatus = process.env.GITHUB_AW_UPDATE_STATUS === "true";
  const canUpdateTitle = process.env.GITHUB_AW_UPDATE_TITLE === "true";
  const canUpdateBody = process.env.GITHUB_AW_UPDATE_BODY === "true";

  console.log(`Update target configuration: ${updateTarget}`);
  console.log(
    `Can update status: ${canUpdateStatus}, title: ${canUpdateTitle}, body: ${canUpdateBody}`
  );

  // Check if we're in an issue context
  const isIssueContext =
    context.eventName === "issues" || context.eventName === "issue_comment";

  // Validate context based on target configuration
  if (updateTarget === "triggering" && !isIssueContext) {
    console.log(
      'Target is "triggering" but not running in issue context, skipping issue update'
    );
    return;
  }

  const updatedIssues = [];

  // Process each update item
  for (let i = 0; i < updateItems.length; i++) {
    const updateItem = updateItems[i];
    console.log(`Processing update-issue item ${i + 1}/${updateItems.length}`);

    // Determine the issue number for this update
    let issueNumber;

    if (updateTarget === "*") {
      // For target "*", we need an explicit issue number from the update item
      if (updateItem.issue_number) {
        issueNumber = parseInt(updateItem.issue_number, 10);
        if (isNaN(issueNumber) || issueNumber <= 0) {
          console.log(
            `Invalid issue number specified: ${updateItem.issue_number}`
          );
          continue;
        }
      } else {
        console.log(
          'Target is "*" but no issue_number specified in update item'
        );
        continue;
      }
    } else if (updateTarget && updateTarget !== "triggering") {
      // Explicit issue number specified in target
      issueNumber = parseInt(updateTarget, 10);
      if (isNaN(issueNumber) || issueNumber <= 0) {
        console.log(
          `Invalid issue number in target configuration: ${updateTarget}`
        );
        continue;
      }
    } else {
      // Default behavior: use triggering issue
      if (isIssueContext) {
        if (context.payload.issue) {
          issueNumber = context.payload.issue.number;
        } else {
          console.log("Issue context detected but no issue found in payload");
          continue;
        }
      } else {
        console.log("Could not determine issue number");
        continue;
      }
    }

    if (!issueNumber) {
      console.log("Could not determine issue number");
      continue;
    }

    console.log(`Updating issue #${issueNumber}`);

    // Build the update object based on allowed fields and provided values
    const updateData = {};
    let hasUpdates = false;

    if (canUpdateStatus && updateItem.status !== undefined) {
      // Validate status value
      if (updateItem.status === "open" || updateItem.status === "closed") {
        updateData.state = updateItem.status;
        hasUpdates = true;
        console.log(`Will update status to: ${updateItem.status}`);
      } else {
        console.log(
          `Invalid status value: ${updateItem.status}. Must be 'open' or 'closed'`
        );
      }
    }

    if (canUpdateTitle && updateItem.title !== undefined) {
      if (
        typeof updateItem.title === "string" &&
        updateItem.title.trim().length > 0
      ) {
        updateData.title = updateItem.title.trim();
        hasUpdates = true;
        console.log(`Will update title to: ${updateItem.title.trim()}`);
      } else {
        console.log("Invalid title value: must be a non-empty string");
      }
    }

    if (canUpdateBody && updateItem.body !== undefined) {
      if (typeof updateItem.body === "string") {
        updateData.body = updateItem.body;
        hasUpdates = true;
        console.log(`Will update body (length: ${updateItem.body.length})`);
      } else {
        console.log("Invalid body value: must be a string");
      }
    }

    if (!hasUpdates) {
      console.log("No valid updates to apply for this item");
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

      console.log("Updated issue #" + issue.number + ": " + issue.html_url);
      updatedIssues.push(issue);

      // Set output for the last updated issue (for backward compatibility)
      if (i === updateItems.length - 1) {
        core.setOutput("issue_number", issue.number);
        core.setOutput("issue_url", issue.html_url);
      }
    } catch (error) {
      console.error(
        `✗ Failed to update issue #${issueNumber}:`,
        error instanceof Error ? error.message : String(error)
      );
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

  console.log(`Successfully updated ${updatedIssues.length} issue(s)`);
  return updatedIssues;
}
await main();
