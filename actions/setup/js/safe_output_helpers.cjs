// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared helper functions for safe-output scripts
 * Provides common validation and target resolution logic
 */

/**
 * Parse a comma-separated list of allowed items from environment variable
 * @param {string|undefined} envValue - Environment variable value
 * @returns {string[]|undefined} Array of allowed items, or undefined if no restrictions
 */
function parseAllowedItems(envValue) {
  const trimmed = envValue?.trim();
  if (!trimmed) {
    return undefined;
  }
  return trimmed
    .split(",")
    .map(item => item.trim())
    .filter(item => item);
}

/**
 * Parse and validate max count from environment variable
 * @param {string|undefined} envValue - Environment variable value
 * @param {number} defaultValue - Default value if not specified
 * @returns {{valid: true, value: number} | {valid: false, error: string}} Validation result
 */
function parseMaxCount(envValue, defaultValue = 3) {
  if (!envValue) {
    return { valid: true, value: defaultValue };
  }

  const parsed = parseInt(envValue, 10);
  if (isNaN(parsed) || parsed < 1) {
    return {
      valid: false,
      error: `Invalid max value: ${envValue}. Must be a positive integer`,
    };
  }

  return { valid: true, value: parsed };
}

/**
 * Resolve the target number (issue/PR) based on configuration and context
 *
 * This function determines which issue or pull request to target based on:
 * - The handler's support for issues vs PRs (supportsPR, supportsIssue)
 * - The target configuration ("triggering", "*", or explicit number)
 * - The workflow context (issue event, PR event, etc.)
 * - Fields in the safe output item (issue_number, pull_request_number, item_number)
 *
 * @param {Object} params - Resolution parameters
 * @param {string} params.targetConfig - Target configuration ("triggering", "*", or explicit number)
 * @param {any} params.item - Safe output item with optional item_number, issue_number, or pull_request_number
 * @param {any} params.context - GitHub Actions context
 * @param {string} params.itemType - Type of item being processed (for error messages)
 * @param {boolean} params.supportsPR - When true, handler supports BOTH issues and PRs (e.g., add_labels)
 *                                       When false, handler supports PRs ONLY (e.g., add_reviewers)
 * @param {boolean} params.supportsIssue - When true, handler supports issues ONLY (e.g., update_issue)
 *                                          Mutually exclusive with supportsPR=false
 * @returns {{success: true, number: number, contextType: string} | {success: false, error: string, shouldFail: boolean}} Resolution result
 */
function resolveTarget(params) {
  const { targetConfig, item, context, itemType, supportsPR = false, supportsIssue = false } = params;

  // Check context type
  const isIssueContext = context.eventName === "issues" || context.eventName === "issue_comment";
  const isPRContext = context.eventName === "pull_request" || context.eventName === "pull_request_review" || context.eventName === "pull_request_review_comment";

  // Default target is "triggering"
  const target = targetConfig || "triggering";

  // Validate context for triggering mode
  if (target === "triggering") {
    if (supportsPR) {
      // Supports both issues and PRs
      if (!isIssueContext && !isPRContext) {
        return {
          success: false,
          error: `Target is "triggering" but not running in issue or pull request context, skipping ${itemType}`,
          shouldFail: false, // Just skip, don't fail the workflow
        };
      }
    } else if (supportsIssue) {
      // Supports issues only
      if (!isIssueContext) {
        return {
          success: false,
          error: `Target is "triggering" but not running in issue context, skipping ${itemType}`,
          shouldFail: false, // Just skip, don't fail the workflow
        };
      }
    } else {
      // Supports PRs only
      if (!isPRContext) {
        return {
          success: false,
          error: `Target is "triggering" but not running in pull request context, skipping ${itemType}`,
          shouldFail: false, // Just skip, don't fail the workflow
        };
      }
    }
  }

  // Resolve target number
  let itemNumber;
  let contextType;

  if (target === "*") {
    // Use item_number, issue_number, or pull_request_number from item
    let numberField;
    if (supportsPR) {
      // Supports both issues and PRs: check all fields
      numberField = item.item_number || item.issue_number || item.pull_request_number;
    } else if (supportsIssue) {
      // Supports issues only: check issue-related fields
      numberField = item.item_number || item.issue_number;
    } else {
      // Supports PRs only: check PR field
      numberField = item.pull_request_number;
    }

    if (numberField) {
      itemNumber = typeof numberField === "number" ? numberField : parseInt(String(numberField), 10);
      if (isNaN(itemNumber) || itemNumber <= 0) {
        const fieldNames = supportsPR ? "item_number/issue_number/pull_request_number" : supportsIssue ? "item_number/issue_number" : "pull_request_number";
        return {
          success: false,
          error: `Invalid ${fieldNames} specified: ${numberField}`,
          shouldFail: true,
        };
      }
      if (supportsPR || supportsIssue) {
        contextType = item.item_number || item.issue_number ? "issue" : "pull request";
      } else {
        contextType = "pull request";
      }
    } else {
      const fieldNames = supportsPR ? "item_number/issue_number" : supportsIssue ? "item_number/issue_number" : "pull_request_number";
      return {
        success: false,
        error: `Target is "*" but no ${fieldNames} specified in ${itemType} item`,
        shouldFail: true,
      };
    }
  } else if (target !== "triggering") {
    // Explicit number
    itemNumber = parseInt(target, 10);
    if (isNaN(itemNumber) || itemNumber <= 0) {
      const itemTypeName = supportsPR || supportsIssue ? "issue" : "pull request";
      return {
        success: false,
        error: `Invalid ${itemTypeName} number in target configuration: ${target}`,
        shouldFail: true,
      };
    }
    contextType = supportsPR || supportsIssue ? "issue" : "pull request";
  } else {
    // Use triggering context
    if (isIssueContext) {
      if (context.payload.issue) {
        itemNumber = context.payload.issue.number;
        contextType = "issue";
      } else {
        return {
          success: false,
          error: "Issue context detected but no issue found in payload",
          shouldFail: true,
        };
      }
    } else if (isPRContext) {
      if (context.payload.pull_request) {
        itemNumber = context.payload.pull_request.number;
        contextType = "pull request";
      } else {
        return {
          success: false,
          error: "Pull request context detected but no pull request found in payload",
          shouldFail: true,
        };
      }
    }
  }

  if (!itemNumber) {
    const itemTypeName = supportsPR ? "issue or pull request" : supportsIssue ? "issue" : "pull request";
    return {
      success: false,
      error: `Could not determine ${itemTypeName} number`,
      shouldFail: true,
    };
  }

  return {
    success: true,
    number: itemNumber,
    contextType: contextType || (supportsPR || supportsIssue ? "issue" : "pull request"),
  };
}

module.exports = {
  parseAllowedItems,
  parseMaxCount,
  resolveTarget,
};
