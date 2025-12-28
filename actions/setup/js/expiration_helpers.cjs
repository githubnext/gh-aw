// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Add expiration XML comment to body lines if expires is set
 * @param {string[]} bodyLines - Array of body lines to append to
 * @param {string} envVarName - Name of the environment variable containing expires days (e.g., "GH_AW_DISCUSSION_EXPIRES")
 * @param {string} entityType - Type of entity for logging (e.g., "Discussion", "Issue", "Pull Request")
 * @returns {void}
 */
function addExpirationComment(bodyLines, envVarName, entityType) {
  const expiresEnv = process.env[envVarName];
  if (expiresEnv) {
    const expiresDays = parseInt(expiresEnv, 10);
    if (!isNaN(expiresDays) && expiresDays > 0) {
      const expirationDate = new Date();
      expirationDate.setDate(expirationDate.getDate() + expiresDays);
      const expirationISO = expirationDate.toISOString();
      bodyLines.push(`<!-- gh-aw-expires: ${expirationISO} -->`);
      core.info(`${entityType} will expire on ${expirationISO} (${expiresDays} days)`);
    }
  }
}

module.exports = {
  addExpirationComment,
};
