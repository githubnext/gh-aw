// @ts-check
/// <reference types="@actions/github-script" />

const { runSafeOutput } = require("./safe_output_runner.cjs");

/**
 * Render function for staged preview
 * @param {any} item - The noop item
 * @param {number} index - Index of the item
 * @returns {string} Markdown content for the preview
 */
function renderNoopPreview(item, index) {
  let content = `### Message ${index + 1}\n`;
  content += `${item.message}\n\n`;
  return content;
}

/**
 * Process noop items - just log the messages for transparency
 * @param {any[]} noopItems - The noop items to process
 */
async function processNoopItems(noopItems) {
  // Process each noop item - just log the messages for transparency
  let summaryContent = "\n\n## No-Op Messages\n\n";
  summaryContent += "The following messages were logged for transparency:\n\n";

  for (let i = 0; i < noopItems.length; i++) {
    const item = noopItems[i];
    core.info(`No-op message ${i + 1}: ${item.message}`);
    summaryContent += `- ${item.message}\n`;
  }

  // Write summary for all noop messages
  await core.summary.addRaw(summaryContent).write();

  // Export the first noop message for use in add-comment default reporting
  if (noopItems.length > 0) {
    core.setOutput("noop_message", noopItems[0].message);
    core.exportVariable("GH_AW_NOOP_MESSAGE", noopItems[0].message);
  }

  core.info(`Successfully processed ${noopItems.length} noop message(s)`);
}

/**
 * Main function to handle noop safe output
 * No-op is a fallback output type that logs messages for transparency
 * without taking any GitHub API actions
 */
async function main() {
  await runSafeOutput({
    itemType: "noop",
    itemTypePlural: "noop",
    stagedTitle: "No-Op Messages",
    stagedDescription: "The following messages would be logged if staged mode was disabled:",
    renderStagedItem: renderNoopPreview,
    processItems: processNoopItems,
  });
}

await main();
