// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Read the validated output content from environment variable
  const outputContent = process.env.GH_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
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
    core.setFailed(`Error parsing agent output JSON: ${getErrorMessage(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    return;
  }

  // Find all pull_request_ready_for_review items
  const items = validatedOutput.items.filter(/** @param {any} item */ item => item.type === "pull_request_ready_for_review");
  if (items.length === 0) {
    core.info("No pull_request_ready_for_review items found in agent output");
    return;
  }

  core.info(`Found ${items.length} pull_request_ready_for_review item(s)`);

  // If in staged mode, emit step summary instead of performing actions
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Pull Request Ready for Review Preview\n\n";
    summaryContent += "The following actions would be performed if staged mode was disabled:\n\n";

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      const prNumber = item.pull_request_number || context.payload.pull_request?.number;
      summaryContent += `### Action ${i + 1}\n`;
      summaryContent += `**Pull Request**: #${prNumber}\n`;
      summaryContent += `**Reason**: ${item.reason}\n`;
      summaryContent += `**Action**: Mark PR as ready for review (set draft: false)\n`;
      summaryContent += `**Comment**: Will post reason as comment on PR\n`;
      summaryContent += "\n";
    }

    core.summary.addRaw(summaryContent).write();
    return;
  }

  // Process each item
  for (const item of items) {
    try {
      // Determine PR number
      let prNumber = item.pull_request_number;

      if (!prNumber) {
        // Use the triggering PR if available
        if (context.payload.pull_request?.number) {
          prNumber = context.payload.pull_request.number;
        } else {
          core.error("No pull request number provided and no triggering PR found");
          core.setFailed("No pull request number available");
          return;
        }
      }

      // Convert to number if it's a string
      if (typeof prNumber === "string") {
        prNumber = parseInt(prNumber, 10);
        if (isNaN(prNumber)) {
          core.error(`Invalid pull request number: ${item.pull_request_number}`);
          core.setFailed(`Invalid pull request number: ${item.pull_request_number}`);
          return;
        }
      }

      core.info(`Processing pull_request_ready_for_review for PR #${prNumber}`);

      // First, check if the PR is actually a draft
      const { data: pr } = await github.rest.pulls.get({
        owner: context.repo.owner,
        repo: context.repo.repo,
        pull_number: prNumber,
      });

      if (!pr.draft) {
        core.info(`PR #${prNumber} is already marked as ready for review (not a draft)`);
        continue;
      }

      // Mark the PR as ready for review by setting draft to false
      await github.rest.pulls.update({
        owner: context.repo.owner,
        repo: context.repo.repo,
        pull_number: prNumber,
        draft: false,
      });

      core.info(`✓ Successfully marked PR #${prNumber} as ready for review`);

      // Post the reason as a comment
      await github.rest.issues.createComment({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: prNumber,
        body: item.reason,
      });

      core.info(`✓ Posted reason comment on PR #${prNumber}`);

      // Set outputs
      core.setOutput("pull_request_number", prNumber);
      core.setOutput("pull_request_url", pr.html_url);
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to mark PR as ready for review: ${errorMessage}`);
      core.setFailed(`Failed to mark PR as ready for review: ${errorMessage}`);
      return;
    }
  }
}

// Call the main function
await main();
