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

/**
 * Generate a quoted footer with optional expiration line
 * @param {Object} options - Footer generation options
 * @param {string} options.footerText - The main footer text (already formatted with ">")
 * @param {number} [options.expiresHours] - Hours until expiration (0 or undefined means no expiration)
 * @param {string} [options.entityType] - Type of entity for logging (e.g., "Issue", "Discussion", "Pull Request")
 * @param {string} [options.suffix] - Optional suffix to append after the footer (e.g., XML marker, type marker)
 * @returns {string} Complete footer with expiration in quoted section
 */
function generateFooterWithExpiration(options) {
  const { footerText, expiresHours, entityType, suffix } = options;
  let footer = footerText;

  // Add expiration line inside the quoted section if configured
  if (expiresHours && expiresHours > 0) {
    const expirationDate = new Date();
    expirationDate.setHours(expirationDate.getHours() + expiresHours);
    const expirationLine = createExpirationLine(expirationDate);
    footer = `${footer}\n>\n> ${expirationLine}`;

    if (entityType) {
      core.info(`${entityType} will expire on ${expirationDate.toISOString()} (${expiresHours} hours)`);
    }
  }

  // Add suffix if provided (e.g., XML marker, type marker)
  if (suffix) {
    footer = `${footer}${suffix}`;
  }

  return footer;
}

/**
 * Add expiration to an existing footer that may contain an XML marker
 * Inserts the expiration line before the XML marker to keep it in the quoted section
 * @param {string} footer - Existing footer text
 * @param {number} [expiresHours] - Hours until expiration (0 or undefined means no expiration)
 * @param {string} [entityType] - Type of entity for logging
 * @returns {string} Footer with expiration inserted before XML marker
 */
function addExpirationToFooter(footer, expiresHours, entityType) {
  if (!expiresHours || expiresHours <= 0) {
    return footer;
  }

  const expirationDate = new Date();
  expirationDate.setHours(expirationDate.getHours() + expiresHours);
  const expirationLine = createExpirationLine(expirationDate);

  // Look for XML marker at the end of footer
  const xmlMarkerMatch = footer.match(/\n\n<!--.*?-->\n?$/s);
  if (xmlMarkerMatch) {
    // Insert expiration before XML marker
    const xmlMarker = xmlMarkerMatch[0];
    const footerWithoutXml = footer.substring(0, footer.length - xmlMarker.length);
    footer = `${footerWithoutXml}\n>\n> ${expirationLine}${xmlMarker}`;
  } else {
    // No XML marker, just append to footer
    footer = `${footer}\n>\n> ${expirationLine}`;
  }

  if (entityType) {
    core.info(`${entityType} will expire on ${expirationDate.toISOString()} (${expiresHours} hours)`);
  }

  return footer;
}

module.exports = {
  addExpirationComment,
  generateFooterWithExpiration,
  addExpirationToFooter,
};
