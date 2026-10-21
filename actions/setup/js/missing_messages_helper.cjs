// @ts-check
/**
 * Missing Messages Helper
 *
 * This module provides access to collected report_missing_tool, report_missing_data,
 * and report_incomplete messages for use by safe output handlers.
 */

const { generateMissingInfoSections } = require("./missing_info_formatter.cjs");
const { parseReportMarkdown } = require("./report_markdown.cjs");

/**
 * Global storage for collected missing messages
 * @type {{missingTools: Array<any>, missingData: Array<any>, noopMessages?: Array<any>, reportIncomplete: Array<any>} | null}
 */
/** @type {any} */
let collectedMissings = null;

/**
 * Set the collected missing messages
 * @param {{missingTools: Array<any>, missingData: Array<any>, noopMessages?: Array<any>, reportIncomplete: Array<any>}} missings - Collected missing messages
 */
function setCollectedMissings(missings) {
  collectedMissings = missings;
}

/**
 * Get the collected missing messages
 * @returns {{missingTools: Array<any>, missingData: Array<any>, noopMessages?: Array<any>, reportIncomplete: Array<any>} | null} Collected missing messages
 */
function getCollectedMissings() {
  return collectedMissings;
}

/**
 * Generate missing info sections for appending to safe output footers
 * @returns {string} HTML details sections for missing tools, data, and incomplete signals
 */
function getMissingInfoSections() {
  if (!collectedMissings) {
    return "";
  }
  return generateMissingInfoSections(collectedMissings);
}

/**
 * Build diagnostic nodes from a trusted template, never from agent-authored markup.
 * Scalar slots are inserted after parsing and cannot create framework wrappers.
 * @param {{missingTools?: Array<any>, missingData?: Array<any>, noopMessages?: Array<any>, reportIncomplete?: Array<any>} | null} missings
 * @returns {import("./report_ast.cjs").ReportBlock[]}
 */
function generateMissingInfoReportSections(missings) {
  if (!missings) return [];
  /** @type {Record<string, string>} */
  const context = {};
  let nextSlot = 0;
  /** @param {Array<any> | undefined} messages */
  const slots = messages =>
    messages?.map(item =>
      Object.fromEntries(
        Object.entries(item).map(([field, value]) => {
          if (!value) return [field, value];
          const key = `diagnostic${nextSlot++}`;
          context[key] = String(value);
          return [field, `{${key}}`];
        })
      )
    );
  const template = generateMissingInfoSections({
    missingTools: slots(missings.missingTools),
    missingData: slots(missings.missingData),
    noopMessages: slots(missings.noopMessages),
    reportIncomplete: slots(missings.reportIncomplete),
  });
  return parseReportMarkdown(template, { context, minimumHeadingLevel: 2, agentTainted: true }).map(node => (node.type === "details" ? { ...node, agentTainted: false } : node));
}

module.exports = {
  setCollectedMissings,
  getCollectedMissings,
  getMissingInfoSections,
  generateMissingInfoReportSections,
};
