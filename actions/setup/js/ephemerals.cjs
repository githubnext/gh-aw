// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Regex pattern to match expiration marker with checked checkbox
 * Allows flexible whitespace: - [x] expires <!-- gh-aw-expires: DATE --> on ...
 * Pattern is more resilient to spacing variations
 */
const EXPIRATION_PATTERN = /^-\s*\[x\]\s+expires\s*<!--\s*gh-aw-expires:\s*([^>]+)\s*-->/m;

/**
 * Format a Date object to human-readable string in UTC
 * @param {Date} date - Date to format
 * @returns {string} Human-readable date string (e.g., "Jan 25, 2026, 1:53 PM")
 */
function formatExpirationDate(date) {
  return date.toLocaleString("en-US", {
    dateStyle: "medium",
    timeStyle: "short",
    timeZone: "UTC",
  });
}

/**
 * Create expiration marker line with checkbox, XML comment, and human-readable date
 * @param {Date} expirationDate - Date when the item expires
 * @returns {string} Formatted expiration line
 */
function createExpirationLine(expirationDate) {
  const expirationISO = expirationDate.toISOString();
  const humanReadableDate = formatExpirationDate(expirationDate);
  return `- [x] expires <!-- gh-aw-expires: ${expirationISO} --> on ${humanReadableDate} UTC`;
}

/**
 * Extract expiration date from text body
 * @param {string} body - Text body containing expiration marker
 * @returns {Date|null} Expiration date or null if not found/invalid
 */
function extractExpirationDate(body) {
  const match = body.match(EXPIRATION_PATTERN);

  if (!match) {
    return null;
  }

  const expirationISO = match[1].trim();
  const expirationDate = new Date(expirationISO);

  // Validate the date
  if (isNaN(expirationDate.getTime())) {
    return null;
  }

  return expirationDate;
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
    footer = `${footer}\n> ${expirationLine}`;

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
    footer = `${footerWithoutXml}\n> ${expirationLine}${xmlMarker}`;
  } else {
    // No XML marker, just append to footer
    footer = `${footer}\n> ${expirationLine}`;
  }

  if (entityType) {
    core.info(`${entityType} will expire on ${expirationDate.toISOString()} (${expiresHours} hours)`);
  }

  return footer;
}

module.exports = {
  EXPIRATION_PATTERN,
  formatExpirationDate,
  createExpirationLine,
  extractExpirationDate,
  generateFooterWithExpiration,
  addExpirationToFooter,
};
