// @ts-check
"use strict";

/**
 * Read a GitHub Actions input value, handling both the standard underscore form
 * (INPUT_<NAME>) and the hyphen form (INPUT_<NAM-E>) preserved by some runner versions.
 *
 * GitHub Actions converts input names to INPUT_<UPPER_UNDERSCORE> by default, but
 * some runner versions preserve the original hyphen from the input name. Checking
 * both forms ensures the value is resolved regardless of the runner version.
 *
 * @param {string} name - Input name in UPPER_UNDERSCORE form (e.g. "JOB_NAME")
 * @returns {string} Trimmed input value, or "" if not set.
 */
function getActionInput(name) {
  const hyphenName = name.replace(/_/g, "-");
  return (process.env[`INPUT_${name}`] || process.env[`INPUT_${hyphenName}`] || "").trim();
}

module.exports = { getActionInput };
