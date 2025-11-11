// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Get fingerprint from environment variable and log it if present
 * @returns {string} Fingerprint value or empty string
 */
function getFingerprint() {
  const fingerprint = process.env.GH_AW_FINGERPRINT || "";
  if (fingerprint) {
    core.info(`Fingerprint: ${fingerprint}`);
  }
  return fingerprint;
}

module.exports = {
  getFingerprint,
};
