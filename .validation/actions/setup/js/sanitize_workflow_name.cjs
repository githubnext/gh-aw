// @ts-check
/**
 * Sanitizes a workflow name for use in file paths
 * @param {string} name - Workflow name to sanitize
 * @returns {string} Sanitized name
 */
function sanitizeWorkflowName(name) {
  return name
    .toLowerCase()
    .replace(/[:\\/\s]/g, "-")
    .replace(/[^a-z0-9._-]/g, "-");
}

module.exports = { sanitizeWorkflowName };
