/**
 * Missing Messages Helper
 *
 * This module provides access to collected missing_tool, missing_data,
 * and report_incomplete messages for use by safe output handlers.
 */

const { generateMissingInfoSections } = require("./missing_info_formatter.cjs");

/**
 * Global storage for collected missing messages
 * @type {{missingTools: Array<any>, missingData: Array<any>, noopMessages?: Array<any>, reportIncomplete: Array<any>} | null}
 */
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

module.exports = {
  setCollectedMissings,
  getCollectedMissings,
  getMissingInfoSections,
};
