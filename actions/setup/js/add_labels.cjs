// @ts-check
/// <reference types="@actions/github-script" />

const { processSafeOutput } = require("./safe_output_processor.cjs");
const { validateLabels } = require("./safe_output_validator.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

async function main(handlerConfig = {}) {
  // Use shared processor for common steps
  const result = await processSafeOutput(
    {
      itemType: "add_labels",
      configKey: "add_labels",
      displayName: "Labels",
      itemTypeName: "label addition",
      supportsPR: true,
      supportsIssue: true,
      envVars: {
        // Config values now passed via config object, not env vars
        allowed: undefined,
        maxCount: undefined,
        target: undefined,
      },
    },
    {
      title: "Add Labels",
      description: "The following labels would be added if staged mode was disabled:",
      renderItem: item => {
        let content = "";
        if (item.item_number) {
          content += `**Target Issue:** #${item.item_number}\n\n`;
        } else {
          content += `**Target:** Current issue/PR\n\n`;
        }
        if (item.labels && item.labels.length > 0) {
          content += `**Labels to add:** ${item.labels.join(", ")}\n\n`;
        }
        return content;
      },
    },
    handlerConfig // Pass handler config as third parameter
  );

  if (!result.success) {
    return;
  }

  const { item: labelsItem, config, targetResult } = result;
  if (!config || !targetResult || targetResult.number === undefined) {
    core.setFailed("Internal error: config, targetResult, or targetResult.number is undefined");
    return;
  }
  const { allowed: allowedLabels, maxCount } = config;
  const itemNumber = targetResult.number;
  const { contextType } = targetResult;

  const requestedLabels = labelsItem.labels ?? [];
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
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.error(`Failed to add labels: ${errorMessage}`);
    core.setFailed(`Failed to add labels: ${errorMessage}`);
  }
}

module.exports = { main };
