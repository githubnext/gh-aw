// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Add expiration checkbox with XML comment to body lines if expires is set
 * @param {string[]} bodyLines - Array of body lines to append to
 * @param {string} envVarName - Name of the environment variable containing expires hours (e.g., "GH_AW_DISCUSSION_EXPIRES")
 * @param {string} entityType - Type of entity for logging (e.g., "Discussion", "Issue", "Pull Request")
 * @returns {void}
 */
function addExpirationComment(bodyLines, envVarName, entityType) {
  const expiresEnv = process.env[envVarName];
  if (expiresEnv) {
    const expiresHours = parseInt(expiresEnv, 10);
    if (!isNaN(expiresHours) && expiresHours > 0) {
      const expirationDate = new Date();
      expirationDate.setHours(expirationDate.getHours() + expiresHours);
      const expirationISO = expirationDate.toISOString();
      // Format: - [x] expires <!-- gh-aw-expires: ISO_DATE --> on DATETIME
      const humanReadableDate = expirationDate.toLocaleString("en-US", {
        dateStyle: "medium",
        timeStyle: "short",
        timeZone: "UTC",
      });
      bodyLines.push(`- [x] expires <!-- gh-aw-expires: ${expirationISO} --> on ${humanReadableDate} UTC`);
      core.info(`${entityType} will expire on ${expirationISO} (${expiresHours} hours)`);
    }
  }
}

module.exports = {
  addExpirationComment,
};
