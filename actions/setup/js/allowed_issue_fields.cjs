// @ts-check
/// <reference types="@actions/github-script" />

const { ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * Parse allowed issue field names from config.
 * @param {string[]|string|undefined} value
 * @returns {string[]}
 */
function parseAllowedIssueFields(value) {
  if (value == null || value === "") return [];
  const raw = Array.isArray(value) ? value : String(value).split(",");
  return [...new Set(raw.map(item => String(item).trim()).filter(Boolean))];
}

/**
 * Build a lowercased Set from allowedFields.
 * Returns null when no restriction applies (empty list, non-array, or wildcard "*").
 * @param {string[]} allowedFields
 * @returns {Set<string>|null}
 */
function buildAllowedFieldSet(allowedFields) {
  if (!Array.isArray(allowedFields) || allowedFields.length === 0 || allowedFields.includes("*")) return null;
  return new Set(allowedFields.map(f => f.toLowerCase()));
}

/**
 * Validate one issue field name against configured allowed-fields.
 * @param {string} fieldName
 * @param {string[]} allowedFields
 * @returns {void}
 */
function validateAllowedIssueFieldName(fieldName, allowedFields) {
  if (!fieldName) return;
  const fieldSet = buildAllowedFieldSet(allowedFields);
  if (!fieldSet) return;
  if (!fieldSet.has(fieldName.toLowerCase())) {
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
  if (!Array.isArray(issueFields) || issueFields.length === 0) return;
  const fieldSet = buildAllowedFieldSet(allowedFields);
  if (!fieldSet) return;
  for (const field of issueFields) {
    if (!fieldSet.has(field.name.toLowerCase())) {
      throw new Error(`${ERR_VALIDATION}: issue field "${field.name}" is not in the allowed-fields list: ${allowedFields.join(", ")}`);
    }
  }
}

module.exports = {
  parseAllowedIssueFields,
  validateAllowedIssueFieldName,
  validateAllowedIssueFields,
};
