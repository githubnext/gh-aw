// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeLabelContent } = require("./sanitize_label_content.cjs");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { generateStagedPreview } = require("./staged_preview.cjs");
const { parseAllowedItems, parseMaxCount, resolveTarget } = require("./safe_output_helpers.cjs");

async function main() {
  const result = loadAgentOutput();
  if (!result.success) {
    return;
  }

  const labelsItem = result.items.find(item => item.type === "add_labels");
  if (!labelsItem) {
    core.warning("No add-labels item found in agent output");
    return;
  }
  core.info(`Found add-labels item with ${labelsItem.labels.length} labels`);
  if (process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true") {
    await generateStagedPreview({
      title: "Add Labels",
      description: "The following labels would be added if staged mode was disabled:",
      items: [labelsItem],
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
    });
    return;
  }

  // Parse allowed labels
  const allowedLabels = parseAllowedItems(process.env.GH_AW_LABELS_ALLOWED);
  if (allowedLabels) {
    core.info(`Allowed labels: ${JSON.stringify(allowedLabels)}`);
  } else {
    core.info("No label restrictions - any labels are allowed");
  }

  // Parse max count
  const maxCountResult = parseMaxCount(process.env.GH_AW_LABELS_MAX_COUNT, 3);
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
  for (const label of requestedLabels) {
    if (label && typeof label === "string" && label.startsWith("-")) {
      core.setFailed(`Label removal is not permitted. Found line starting with '-': ${label}`);
      return;
    }
  }
  let validLabels;
  if (allowedLabels) {
    validLabels = requestedLabels.filter(label => allowedLabels.includes(label));
  } else {
    validLabels = requestedLabels;
  }
  let uniqueLabels = validLabels
    .filter(label => label != null && label !== false && label !== 0)
    .map(label => String(label).trim())
    .filter(label => label)
    .map(label => sanitizeLabelContent(label))
    .filter(label => label)
    .map(label => (label.length > 64 ? label.substring(0, 64) : label))
    .filter((label, index, arr) => arr.indexOf(label) === index);
  if (uniqueLabels.length > maxCount) {
    core.info(`too many labels, keep ${maxCount}`);
    uniqueLabels = uniqueLabels.slice(0, maxCount);
  }
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
await main();
