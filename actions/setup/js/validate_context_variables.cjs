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
 * List of numeric context variable names
 * These correspond to the environment variables passed from the Go compiler
 */
const NUMERIC_CONTEXT_VARS = [
  "ISSUE_NUMBER",
  "PULL_REQUEST_NUMBER",
  "DISCUSSION_NUMBER",
  "MILESTONE_NUMBER",
  "CHECK_RUN_NUMBER",
  "CHECK_SUITE_NUMBER",
  "WORKFLOW_RUN_NUMBER",
  "CHECK_RUN_ID",
  "CHECK_SUITE_ID",
  "COMMENT_ID",
  "DEPLOYMENT_ID",
  "DEPLOYMENT_STATUS_ID",
  "HEAD_COMMIT_ID",
  "INSTALLATION_ID",
  "WORKFLOW_JOB_RUN_ID",
  "LABEL_ID",
  "MILESTONE_ID",
  "ORGANIZATION_ID",
  "PAGE_ID",
  "PROJECT_ID",
  "PROJECT_CARD_ID",
  "PROJECT_COLUMN_ID",
  "RELEASE_ID",
  "REPOSITORY_ID",
  "REVIEW_ID",
  "REVIEW_COMMENT_ID",
  "SENDER_ID",
  "WORKFLOW_RUN_ID",
  "WORKFLOW_JOB_ID",
  "RUN_ID",
  "RUN_NUMBER",
];

/**
 * Validates that a value is either empty or a valid integer
 * @param {string | undefined} value - The value to validate
 * @param {string} varName - The variable name for error reporting
 * @returns {{valid: boolean, message: string}} Validation result
 */
function validateNumericValue(value, varName) {
  // Empty or undefined is valid (field not present)
  if (!value || value.trim() === "") {
    return { valid: true, message: `${varName} is empty (valid)` };
  }

  // Check if the value is a valid integer (positive or negative)
  // Allow only digits, optionally preceded by a minus sign
  const isValidInteger = /^-?\d+$/.test(value.trim());

  if (!isValidInteger) {
    return {
      valid: false,
      message: `${varName} contains non-numeric characters: "${value}"`,
    };
  }

  // Additional check: ensure it's within JavaScript's safe integer range
  const numValue = parseInt(value.trim(), 10);
  if (!Number.isSafeInteger(numValue)) {
    return {
      valid: false,
      message: `${varName} is outside safe integer range: ${value}`,
    };
  }

  return { valid: true, message: `${varName} is valid: ${value}` };
}

/**
 * Main validation function
 */
async function main() {
  try {
    core.info("Starting context variable validation...");

    const failures = [];
    const warnings = [];
    let checkedCount = 0;

    // Validate each numeric context variable
    for (const varName of NUMERIC_CONTEXT_VARS) {
      const envVarName = `GH_AW_CONTEXT_${varName}`;
      const value = process.env[envVarName];

      // Only validate if the variable is set
      if (value !== undefined) {
        checkedCount++;
        const result = validateNumericValue(value, varName);

        if (result.valid) {
          core.info(`✓ ${result.message}`);
        } else {
          core.error(`✗ ${result.message}`);
          failures.push({
            varName,
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
        failures.map(f => `  - ${f.varName}: "${f.value}"\n    ${f.message}`).join("\n\n") +
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
  NUMERIC_CONTEXT_VARS,
};
