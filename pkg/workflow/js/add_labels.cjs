// @ts-check
/// <reference types="@actions/github-script" />

const { executeListAction, renderMarkdownList } = require("./safe_output_list_action.cjs");
const { validateLabels } = require("./safe_output_validator.cjs");
const { parseAllowedItems } = require("./safe_output_helpers.cjs");
const { getSafeOutputConfig, validateMaxCount } = require("./safe_output_validator.cjs");

await executeListAction({
  itemType: "add_labels",
  singularNoun: "label",
  pluralNoun: "labels",
  itemsField: "labels",
  configKey: "add_labels",
  configAllowedField: "allowed",
  envAllowedVar: "GH_AW_LABELS_ALLOWED",
  envMaxCountVar: "GH_AW_LABELS_MAX_COUNT",
  envTargetVar: "GH_AW_LABELS_TARGET",
  targetNumberField: "item_number",
  supportsPR: true,
  stagedPreviewTitle: "Add Labels",
  stagedPreviewDescription: "The following labels would be added if staged mode was disabled:",
  renderStagedItem: item => {
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
  validateItems: requestedLabels => {
    // Get configuration for custom validation
    const config = getSafeOutputConfig("add_labels");
    const allowedLabels = parseAllowedItems(process.env.GH_AW_LABELS_ALLOWED) || config.allowed;
    const maxCountResult = validateMaxCount(process.env.GH_AW_LABELS_MAX_COUNT, config.max);
    const maxCount = maxCountResult.valid ? maxCountResult.value : 1;
    return validateLabels(requestedLabels, allowedLabels, maxCount);
  },
  applyAction: async (labels, contextType, itemNumber) => {
    await github.rest.issues.addLabels({
      owner: context.repo.owner,
      repo: context.repo.repo,
      issue_number: itemNumber,
      labels,
    });
  },
  outputField: "labels_added",
  summaryTitle: "Label Addition",
  renderSuccessSummary: (labels, contextType, itemNumber) => {
    const labelsListMarkdown = renderMarkdownList(labels);
    return `
## Label Addition

Successfully added ${labels.length} label(s) to ${contextType} #${itemNumber}:

${labelsListMarkdown}
`;
  },
});
