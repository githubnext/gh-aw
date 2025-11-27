// @ts-check
/// <reference types="@actions/github-script" />

const { runSingleItemSafeOutput } = require("./safe_output_runner.cjs");
const { parseAllowedItems, resolveTarget } = require("./safe_output_helpers.cjs");
const { getSafeOutputConfig, validateLabels, validateMaxCount } = require("./safe_output_validator.cjs");

/**
 * Render function for staged preview
 * @param {any} item - The add_labels item
 * @returns {string} Markdown content for the preview
 */
function renderLabelsPreview(item) {
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
}

/**
 * Process a single add_labels item
 * @param {any} labelsItem - The add_labels item to process
 */
async function processAddLabels(labelsItem) {
  // Get configuration from config.json
  const config = getSafeOutputConfig("add_labels");

  // Parse allowed labels (from env or config)
  const allowedLabels = parseAllowedItems(process.env.GH_AW_LABELS_ALLOWED) || config.allowed;
  if (allowedLabels) {
    core.info(`Allowed labels: ${JSON.stringify(allowedLabels)}`);
  } else {
    core.info("No label restrictions - any labels are allowed");
  }

  // Parse max count (env takes priority, then config)
  const maxCountResult = validateMaxCount(process.env.GH_AW_LABELS_MAX_COUNT, config.max);
  if (!maxCountResult.valid) {
    core.setFailed(maxCountResult.error);
    return;
  }
  const maxCount = maxCountResult.value;
  core.info(`Max count: ${maxCount}`);

  // Resolve target
  const labelsTarget = process.env.GH_AW_LABELS_TARGET || "triggering";
  core.info(`Labels target configuration: ${labelsTarget}`);

  const targetResult = resolveTarget({
    targetConfig: labelsTarget,
    item: labelsItem,
    context,
    itemType: "label addition",
    supportsPR: true,
  });

  if (!targetResult.success) {
    if (targetResult.shouldFail) {
      core.setFailed(targetResult.error);
    } else {
      core.info(targetResult.error);
    }
    return;
  }

  const itemNumber = targetResult.number;
  const contextType = targetResult.contextType;
  const requestedLabels = labelsItem.labels || [];
  core.info(`Requested labels: ${JSON.stringify(requestedLabels)}`);

  // Use validation helper to sanitize and validate labels
  const labelsResult = validateLabels(requestedLabels, allowedLabels, maxCount);
  if (!labelsResult.valid) {
    // If no valid labels, log info and return gracefully instead of failing
    if (labelsResult.error && labelsResult.error.includes("No valid labels")) {
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
    core.setFailed(labelsResult.error || "Invalid labels");
    return;
  }

  const uniqueLabels = labelsResult.value || [];

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
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.error(`Failed to add labels: ${errorMessage}`);
    core.setFailed(`Failed to add labels: ${errorMessage}`);
  }
}

async function main() {
  await runSingleItemSafeOutput({
    itemType: "add_labels",
    itemTypePlural: "add-labels",
    stagedTitle: "Add Labels",
    stagedDescription: "The following labels would be added if staged mode was disabled:",
    renderStagedItem: renderLabelsPreview,
    processSingleItem: processAddLabels,
  });
}

await main();
