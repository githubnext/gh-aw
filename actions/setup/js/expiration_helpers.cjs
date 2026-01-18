// @ts-check
/// <reference types="@actions/github-script" />

const { createExpirationLine } = require("./ephemerals.cjs");

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
      bodyLines.push(createExpirationLine(expirationDate));
      core.info(`${entityType} will expire on ${expirationDate.toISOString()} (${expiresHours} hours)`);
    }
  }
}

module.exports = {
  addExpirationComment,
};
