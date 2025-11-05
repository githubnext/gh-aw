// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generate a staged mode preview summary and write it to the step summary.
 *
 * @param {Object} options - Configuration options for the preview
 * @param {string} options.title - The main title for the preview (e.g., "Create Issues")
 * @param {string} options.description - Description of what would happen if staged mode was disabled
 * @param {Array<any>} options.items - Array of items to preview
 * @param {(item: any, index: number) => string} options.renderItem - Function to render each item as markdown
 * @returns {Promise<void>}
 */
async function generateStagedPreview(options) {
  const { title, description, items, renderItem } = options;

  let summaryContent = `## 🎭 Staged Mode: ${title} Preview\n\n`;
  summaryContent += `${description}\n\n`;

  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    summaryContent += renderItem(item, i);
    summaryContent += "---\n\n";
  }

  try {
    await core.summary.addRaw(summaryContent).write();
    core.info(summaryContent);
    core.info(`📝 ${title} preview written to step summary`);
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
}

module.exports = { generateStagedPreview };
