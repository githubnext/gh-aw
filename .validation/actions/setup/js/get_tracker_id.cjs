// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get tracker-id from environment variable, log it, and optionally format it
 * @param {string} [format] - Output format: "markdown" for HTML comment, "text" for plain text, or undefined for raw value
 * @returns {string} Tracker ID in requested format or empty string
 */
function getTrackerID(format) {
  const trackerID = process.env.GH_AW_TRACKER_ID || "";
  if (trackerID) {
    core.info(`Tracker ID: ${trackerID}`);
    return format === "markdown" ? `\n\n<!-- gh-aw-tracker-id: ${trackerID} -->` : trackerID;
  }
  return "";
}

module.exports = {
  getTrackerID,
};
