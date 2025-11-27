// @ts-check
/// <reference types="@actions/github-script" />

const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  // Find all update-pull-request items
  const updateItems = result.items.filter(/** @param {any} item */ item => item.type === "update_pull_request");
  if (updateItems.length === 0) {
    core.info("No update-pull-request items found in agent output");
    return;
  }

  core.info(`Found ${updateItems.length} update-pull-request item(s)`);

  // If in staged mode, emit step summary instead of updating pull requests
  if (isStaged) {
    await generateStagedPreview({
      title: "Update Pull Requests",
      description: "The following pull request updates would be applied if staged mode was disabled:",
      items: updateItems,
      renderItem: (item, index) => {
        let content = `### Pull Request Update ${index + 1}\n`;
        if (item.pull_request_number) {
          content += `**Target PR:** #${item.pull_request_number}\n\n`;
        } else {
          content += `**Target:** Current pull request\n\n`;
        }

        if (item.title !== undefined) {
          content += `**New Title:** ${item.title}\n\n`;
        }
        if (item.body !== undefined) {
          const operation = item.operation || "replace";
          content += `**Operation:** ${operation}\n`;
          content += `**Body Content:**\n${item.body}\n\n`;
        }
        return content;
      },
    });
    return;
  }

  // Get the configuration from environment variables
  const updateTarget = process.env.GH_AW_UPDATE_TARGET || "triggering";
  const canUpdateTitle = process.env.GH_AW_UPDATE_TITLE === "true";
  const canUpdateBody = process.env.GH_AW_UPDATE_BODY === "true";

  core.info(`Update target configuration: ${updateTarget}`);
  core.info(`Can update title: ${canUpdateTitle}, body: ${canUpdateBody}`);

  // Check if we're in a pull request context
  const isPRContext =
    context.eventName === "pull_request" ||
    context.eventName === "pull_request_review" ||
    context.eventName === "pull_request_review_comment" ||
    context.eventName === "pull_request_target";

  // Also check for issue_comment on a PR (issue_comment events can be on PRs too)
  const isIssueCommentOnPR = context.eventName === "issue_comment" && context.payload.issue && context.payload.issue.pull_request;

  const hasPRContext = isPRContext || isIssueCommentOnPR;

  // Validate context based on target configuration
  if (updateTarget === "triggering" && !hasPRContext) {
    core.info('Target is "triggering" but not running in pull request context, skipping pull request update');
    return;
  }

  const updatedPRs = [];

  // Process each update item
  for (let i = 0; i < updateItems.length; i++) {
    const updateItem = updateItems[i];
    core.info(`Processing update-pull-request item ${i + 1}/${updateItems.length}`);

    // Determine the pull request number for this update
    let prNumber;

    if (updateTarget === "*") {
      // For target "*", we need an explicit PR number from the update item
      if (updateItem.pull_request_number) {
        prNumber = parseInt(updateItem.pull_request_number, 10);
        if (isNaN(prNumber) || prNumber <= 0) {
          core.info(`Invalid PR number specified: ${updateItem.pull_request_number}`);
          continue;
        }
      } else {
        core.info('Target is "*" but no pull_request_number specified in update item');
        continue;
      }
    } else if (updateTarget && updateTarget !== "triggering") {
      // Explicit PR number specified in target
      prNumber = parseInt(updateTarget, 10);
      if (isNaN(prNumber) || prNumber <= 0) {
        core.info(`Invalid PR number in target configuration: ${updateTarget}`);
        continue;
      }
    } else {
      // Default behavior: use triggering PR
      if (context.payload.pull_request) {
        prNumber = context.payload.pull_request.number;
      } else if (context.payload.issue && context.payload.issue.pull_request) {
        // For issue_comment events on PRs, the PR number is in issue.number
        prNumber = context.payload.issue.number;
      } else {
        core.info("Could not determine pull request number");
        continue;
      }
    }

    if (!prNumber) {
      core.info("Could not determine pull request number");
      continue;
    }

    core.info(`Updating pull request #${prNumber}`);

    // Determine operation (default to 'replace')
    const operation = updateItem.operation || "replace";
    core.info(`Body operation: ${operation}`);

    // Get workflow run URL for AI attribution
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "GitHub Agentic Workflow";
    const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;

    // Build the update object based on allowed fields and provided values
    /** @type {any} */
    const updateData = {};
    let hasUpdates = false;

    if (canUpdateTitle && updateItem.title !== undefined) {
      const trimmedTitle = typeof updateItem.title === "string" ? updateItem.title.trim() : "";
      if (trimmedTitle.length > 0) {
        updateData.title = trimmedTitle;
        hasUpdates = true;
        core.info(`Will update title to: ${trimmedTitle}`);
      } else {
        core.info("Invalid title value: must be a non-empty string");
      }
    }

    if (canUpdateBody && updateItem.body !== undefined) {
      if (typeof updateItem.body === "string") {
        // For append/prepend operations, we need to fetch the current PR body first
        if (operation === "append" || operation === "prepend") {
          try {
            const { data: currentPR } = await github.rest.pulls.get({
              owner: context.repo.owner,
              repo: context.repo.repo,
              pull_number: prNumber,
            });
            const currentBody = currentPR.body || "";

            if (operation === "prepend") {
              // Prepend: add content, AI footer, and horizontal line at the start
              const aiFooter = `\n\n> AI generated by [${workflowName}](${runUrl})`;
              const prependSection = `${updateItem.body}${aiFooter}\n\n---\n\n`;
              updateData.body = prependSection + currentBody;
              core.info("Operation: prepend (add to start with separator)");
            } else {
              // Append: add horizontal line, content, and AI footer at the end
              const aiFooter = `\n\n> AI generated by [${workflowName}](${runUrl})`;
              const appendSection = `\n\n---\n\n${updateItem.body}${aiFooter}`;
              updateData.body = currentBody + appendSection;
              core.info("Operation: append (add to end with separator)");
            }
          } catch (error) {
            core.error(`Failed to fetch current PR for ${operation} operation: ${error instanceof Error ? error.message : String(error)}`);
            throw error;
          }
        } else {
          // Replace: just use the new content
          updateData.body = updateItem.body;
          core.info("Operation: replace (full body replacement)");
        }
        hasUpdates = true;
        core.info(`Will update body (length: ${updateData.body.length})`);
      } else {
        core.info("Invalid body value: must be a string");
      }
    }

    if (!hasUpdates) {
      core.info("No valid updates to apply for this item");
      continue;
    }

    try {
      // Update the pull request using GitHub API
      const { data: pr } = await github.rest.pulls.update({
        owner: context.repo.owner,
        repo: context.repo.repo,
        pull_number: prNumber,
        ...updateData,
      });

      core.info("Updated pull request #" + pr.number + ": " + pr.html_url);
      updatedPRs.push(pr);

      // Set output for the last updated PR (for backward compatibility)
      if (i === updateItems.length - 1) {
        core.setOutput("pull_request_number", pr.number);
        core.setOutput("pull_request_url", pr.html_url);
      }
    } catch (error) {
      core.error(`✗ Failed to update pull request #${prNumber}: ${error instanceof Error ? error.message : String(error)}`);
      throw error;
    }
  }

  // Write summary for all updated pull requests
  if (updatedPRs.length > 0) {
    let summaryContent = "\n\n## Updated Pull Requests\n";
    for (const pr of updatedPRs) {
      summaryContent += `- PR #${pr.number}: [${pr.title}](${pr.html_url})\n`;
    }
    await core.summary.addRaw(summaryContent).write();
  }

  core.info(`Successfully updated ${updatedPRs.length} pull request(s)`);
  return updatedPRs;
}
await main();
