// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get campaign from environment variable, log it, and optionally format it
 * @param {string} [format] - Output format: "markdown" for HTML comment, "text" for plain text, or undefined for raw value
 * @returns {string} Campaign in requested format or empty string
 */
function getCampaign(format) {
  const campaign = process.env.GH_AW_CAMPAIGN || "";
  if (campaign) {
    core.info(`Campaign: ${campaign}`);
    return format === "markdown" ? `\n\n<!-- campaign: ${campaign} -->` : campaign;
  }
  return "";
}

module.exports = {
  getCampaign,
};
