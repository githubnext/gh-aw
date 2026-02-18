// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Validate Context Variables Script
 *
 * Validates that context variables that should be numeric (like github.event.issue.number)
 * are either empty or contain only integers. This prevents malicious payloads from hiding
 * special text or code in numeric fields.
 *
 * Context variables validated:
 * - github.event.issue.number
 * - github.event.pull_request.number
 * - github.event.discussion.number
 * - github.event.milestone.number
 * - github.event.check_run.number
 * - github.event.check_suite.number
 * - github.event.workflow_run.number
 * - github.event.check_run.id
 * - github.event.check_suite.id
 * - github.event.comment.id
 * - github.event.deployment.id
 * - github.event.deployment_status.id
 * - github.event.head_commit.id
 * - github.event.installation.id
 * - github.event.workflow_job.run_id
 * - github.event.label.id
 * - github.event.milestone.id
 * - github.event.organization.id
 * - github.event.page.id
 * - github.event.project.id
 * - github.event.project_card.id
 * - github.event.project_column.id
 * - github.event.release.id
 * - github.event.repository.id
 * - github.event.review.id
 * - github.event.review_comment.id
 * - github.event.sender.id
 * - github.event.workflow_run.id
 * - github.event.workflow_job.id
 * - github.run_id
 * - github.run_number
 */

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * List of numeric context variable paths to validate
 * Each entry contains the path within the context object and a display name
 */
const NUMERIC_CONTEXT_PATHS = [
  { path: ["payload", "issue", "number"], name: "github.event.issue.number" },
  { path: ["payload", "pull_request", "number"], name: "github.event.pull_request.number" },
  { path: ["payload", "discussion", "number"], name: "github.event.discussion.number" },
  { path: ["payload", "milestone", "number"], name: "github.event.milestone.number" },
  { path: ["payload", "check_run", "number"], name: "github.event.check_run.number" },
  { path: ["payload", "check_suite", "number"], name: "github.event.check_suite.number" },
  { path: ["payload", "workflow_run", "number"], name: "github.event.workflow_run.number" },
  { path: ["payload", "check_run", "id"], name: "github.event.check_run.id" },
  { path: ["payload", "check_suite", "id"], name: "github.event.check_suite.id" },
  { path: ["payload", "comment", "id"], name: "github.event.comment.id" },
  { path: ["payload", "deployment", "id"], name: "github.event.deployment.id" },
  { path: ["payload", "deployment_status", "id"], name: "github.event.deployment_status.id" },
  { path: ["payload", "head_commit", "id"], name: "github.event.head_commit.id" },
  { path: ["payload", "installation", "id"], name: "github.event.installation.id" },
  { path: ["payload", "workflow_job", "run_id"], name: "github.event.workflow_job.run_id" },
  { path: ["payload", "label", "id"], name: "github.event.label.id" },
  { path: ["payload", "milestone", "id"], name: "github.event.milestone.id" },
  { path: ["payload", "organization", "id"], name: "github.event.organization.id" },
  { path: ["payload", "page", "id"], name: "github.event.page.id" },
  { path: ["payload", "project", "id"], name: "github.event.project.id" },
  { path: ["payload", "project_card", "id"], name: "github.event.project_card.id" },
  { path: ["payload", "project_column", "id"], name: "github.event.project_column.id" },
  { path: ["payload", "release", "id"], name: "github.event.release.id" },
  { path: ["payload", "repository", "id"], name: "github.event.repository.id" },
  { path: ["payload", "review", "id"], name: "github.event.review.id" },
  { path: ["payload", "review_comment", "id"], name: "github.event.review_comment.id" },
  { path: ["payload", "sender", "id"], name: "github.event.sender.id" },
  { path: ["payload", "workflow_run", "id"], name: "github.event.workflow_run.id" },
  { path: ["payload", "workflow_job", "id"], name: "github.event.workflow_job.id" },
  { path: ["run_id"], name: "github.run_id" },
  { path: ["run_number"], name: "github.run_number" },
];

/**
 * Gets a value from a nested object using a path array
 * @param {any} obj - The object to traverse
 * @param {string[]} path - Array of keys to traverse
 * @returns {any} The value at the path, or undefined if not found
 */
function getNestedValue(obj, path) {
  let current = obj;
  for (const key of path) {
    if (current == null || typeof current !== "object") {
      return undefined;
    }
    current = current[key];
  }
  return current;
}

/**
 * Validates that a value is either empty or a valid integer
 * @param {any} value - The value to validate
 * @param {string} varName - The variable name for error reporting
 * @returns {{valid: boolean, message: string}} Validation result
 */
function validateNumericValue(value, varName) {
  // Empty, null, or undefined is valid (field not present)
  if (value == null || value === "") {
    return { valid: true, message: `${varName} is empty (valid)` };
  }

  // Convert to string for validation
  const valueStr = String(value);

  // Check if the value is a valid integer (positive or negative)
  // Allow only digits, optionally preceded by a minus sign
  const isValidInteger = /^-?\d+$/.test(valueStr.trim());

  if (!isValidInteger) {
    return {
      valid: false,
      message: `${varName} contains non-numeric characters: "${valueStr}"`,
    };
  }

  // Additional check: ensure it's within JavaScript's safe integer range
  const numValue = parseInt(valueStr.trim(), 10);
  if (!Number.isSafeInteger(numValue)) {
    return {
      valid: false,
      message: `${varName} is outside safe integer range: ${valueStr}`,
    };
  }

  return { valid: true, message: `${varName} is valid: ${valueStr}` };
}

/**
 * Main validation function
 */
async function main() {
  try {
    core.info("Starting context variable validation...");

    const failures = [];
    let checkedCount = 0;

    // Validate each numeric context variable by reading directly from context
    for (const { path, name } of NUMERIC_CONTEXT_PATHS) {
      const value = getNestedValue(context, path);

      // Only validate if the value exists
      if (value !== undefined) {
        checkedCount++;
        const result = validateNumericValue(value, name);

        if (result.valid) {
          core.info(`✓ ${result.message}`);
        } else {
          core.error(`✗ ${result.message}`);
          failures.push({
            name,
            value,
            message: result.message,
          });
        }
      }
    }

    core.info(`Validated ${checkedCount} context variables`);

    // If there are any failures, fail the workflow
    if (failures.length > 0) {
      const errorMessage =
        `Context variable validation failed!\n\n` +
        `Found ${failures.length} malicious or invalid numeric field(s):\n\n` +
        failures.map(f => `  - ${f.name}: "${f.value}"\n    ${f.message}`).join("\n\n") +
        "\n\n" +
        "Numeric context variables (like github.event.issue.number) must be either empty or valid integers.\n" +
        "This validation prevents injection attacks where special text or code is hidden in numeric fields.\n\n" +
        "If you believe this is a false positive, please report it at:\n" +
        "https://github.com/github/gh-aw/issues";

      core.setFailed(errorMessage);
      throw new Error(errorMessage);
    }

    core.info("✅ All context variables validated successfully");
  } catch (error) {
    const errorMessage = getErrorMessage(error);
    core.setFailed(`Context variable validation failed: ${errorMessage}`);
    throw error;
  }
}

module.exports = {
  main,
  validateNumericValue,
  getNestedValue,
  NUMERIC_CONTEXT_PATHS,
};
