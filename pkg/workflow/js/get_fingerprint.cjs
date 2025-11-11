// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get fingerprint from environment variable, log it, and optionally format it
 * @param {string} [format] - Output format: "markdown" for HTML comment, "text" for plain text, or undefined for raw value
 * @returns {string} Fingerprint in requested format or empty string
 */
function getFingerprint(format) {
  const fingerprint = process.env.GH_AW_FINGERPRINT || "";
  if (fingerprint) {
    core.info(`Fingerprint: ${fingerprint}`);
    return format === "markdown" ? `\n\n<!-- fingerprint: ${fingerprint} -->` : fingerprint;
  }
  return "";
}

module.exports = {
  getFingerprint,
};
