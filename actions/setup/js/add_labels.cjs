// @ts-check
/// <reference types="@actions/github-script" />

const { validateLabels } = require("./safe_output_validator.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Factory function for creating label addition handler
 * @param {Object} config - Handler configuration
 * @param {string[]|undefined} config.allowed - Allowed labels list
 * @param {number} [config.max] - Maximum number of labels (default: 1)
 * @param {string} [config.target] - Target configuration
 * @returns {Function} Handler function that processes individual messages
 */
async function main(config = {}) {
  const { allowed: allowedLabels, max: maxCount = 1, target = "triggering" } = config;

  /**
   * Process a single add_labels message
   * @param {Object} outputItem - The safe output item
   * @param {Object} resolvedTemporaryIds - Map of temporary IDs to actual IDs
   * @returns {Promise<{repo: string, number: number}|undefined>} Result with repo and number, or undefined if skipped
   */
  return async function (outputItem, resolvedTemporaryIds) {
    // Extract labels from the output item
    const requestedLabels = outputItem.labels ?? [];
    core.info(`Requested labels: ${JSON.stringify(requestedLabels)}`);

    // Use validation helper to sanitize and validate labels
    const labelsResult = validateLabels(requestedLabels, allowedLabels, maxCount);
    if (!labelsResult.valid) {
      // If no valid labels, log info and return gracefully instead of failing
      if (labelsResult.error?.includes("No valid labels")) {
        core.info("No labels to add");
        core.setOutput("labels_added", "");
        await core.summary
          .addRaw(
            `
## Label Addition

No labels were added (no valid labels found in agent output).
`
          )
          .write();
        return;
      }
      // For other validation errors, fail the workflow
      core.setFailed(labelsResult.error ?? "Invalid labels");
      return;
    }

    const uniqueLabels = labelsResult.value ?? [];

    if (uniqueLabels.length === 0) {
      core.info("No labels to add");
      core.setOutput("labels_added", "");
      await core.summary
        .addRaw(
          `
## Label Addition

No labels were added (no valid labels found in agent output).
`
        )
        .write();
      return;
    }

    // Determine the target issue/PR number
    let itemNumber;
    let contextType = "issue";

    if (target === "*") {
      // Use item_number from the output item
      itemNumber = outputItem.item_number || outputItem.issue_number || outputItem.pull_request_number;
      if (!itemNumber) {
        core.setFailed('Target is "*" but no item_number/issue_number specified in label addition item');
        return;
      }
      itemNumber = parseInt(itemNumber, 10);
      if (isNaN(itemNumber) || itemNumber <= 0) {
        core.setFailed(`Invalid item_number/issue_number/pull_request_number specified: ${outputItem.item_number}`);
        return;
      }
    } else if (target !== "triggering") {
      // Use explicit issue number from target configuration
      itemNumber = parseInt(target, 10);
      if (isNaN(itemNumber) || itemNumber <= 0) {
        core.setFailed(`Invalid issue number in target configuration: ${target}`);
        return;
      }
    } else {
      // Use triggering issue/PR
      const isPRContext =
        context.eventName === "pull_request" ||
        context.eventName === "pull_request_review" ||
        context.eventName === "pull_request_review_comment";
      const isIssueContext =
        context.eventName === "issues" || context.eventName === "issue_comment";

      if (isPRContext) {
        itemNumber = context.payload.pull_request?.number;
        contextType = "pull request";
        if (!itemNumber) {
          core.setFailed("Pull request context detected but no pull request found in payload");
          return;
        }
      } else if (isIssueContext) {
        itemNumber = context.payload.issue?.number;
        contextType = "issue";
        if (!itemNumber) {
          core.setFailed("Issue context detected but no issue found in payload");
          return;
        }
      } else {
        core.info('Target is "triggering" but not running in issue or pull request context, skipping label addition');
        return;
      }
    }

    core.info(`Adding ${uniqueLabels.length} labels to ${contextType} #${itemNumber}: ${JSON.stringify(uniqueLabels)}`);
    try {
      await github.rest.issues.addLabels({
        owner: context.repo.owner,
        repo: context.repo.repo,
        issue_number: itemNumber,
        labels: uniqueLabels,
      });
      core.info(`Successfully added ${uniqueLabels.length} labels to ${contextType} #${itemNumber}`);
      core.setOutput("labels_added", uniqueLabels.join("\n"));
      const labelsListMarkdown = uniqueLabels.map(label => `- \`${label}\``).join("\n");
      await core.summary
        .addRaw(
          `
## Label Addition

Successfully added ${uniqueLabels.length} label(s) to ${contextType} #${itemNumber}:

${labelsListMarkdown}
`
        )
        .write();

      return {
        repo: `${context.repo.owner}/${context.repo.repo}`,
        number: itemNumber,
      };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to add labels: ${errorMessage}`);
      core.setFailed(`Failed to add labels: ${errorMessage}`);
      return;
    }
  };
}

module.exports = { main };
