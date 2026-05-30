// @ts-check
/// <reference types="@actions/github-script" />

const { ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * Parse allowed issue field names from config.
 * @param {string[]|string|undefined} value
 * @returns {string[]}
 */
function parseAllowedIssueFields(value) {
  if (value == null || value === "") {
    return [];
  }
  const raw = Array.isArray(value) ? value : String(value).split(",");
  const uniqueFields = new Set();
  for (const item of raw) {
    const normalized = String(item).trim();
    if (normalized) {
      uniqueFields.add(normalized);
    }
  }
  return [...uniqueFields];
}

/**
 * Validate one issue field name against configured allowed-fields.
 * @param {string} fieldName
 * @param {string[]} allowedFields
 * @returns {void}
 */
function validateAllowedIssueFieldName(fieldName, allowedFields) {
  if (!fieldName) {
    return;
  }
  if (!Array.isArray(allowedFields) || allowedFields.length === 0 || allowedFields.includes("*")) {
    return;
  }
  const allowedFieldSet = new Set(allowedFields.map(field => field.toLowerCase()));
  if (!allowedFieldSet.has(fieldName.toLowerCase())) {
    throw new Error(`${ERR_VALIDATION}: issue field "${fieldName}" is not in the allowed-fields list: ${allowedFields.join(", ")}`);
  }
}

/**
 * Validate requested issue fields against configured allowed-fields.
 * @param {Array<{name: string, value: string|number}>} issueFields
 * @param {string[]} allowedFields
 * @returns {void}
 */
function validateAllowedIssueFields(issueFields, allowedFields) {
  if (!Array.isArray(issueFields) || issueFields.length === 0) {
    return;
  }
  for (const field of issueFields) {
    validateAllowedIssueFieldName(field.name, allowedFields);
  }
}

module.exports = {
  parseAllowedIssueFields,
  validateAllowedIssueFieldName,
  validateAllowedIssueFields,
};
