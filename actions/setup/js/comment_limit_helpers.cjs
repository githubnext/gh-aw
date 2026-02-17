// @ts-check

/**
 * Comment Limit Helpers
 *
 * This module provides validation functions for enforcing comment constraints
 * across both MCP server (Phase 4 - tool invocation) and safe output processor
 * (Phase 6 - API execution) per Safe Outputs Specification MCE1 and MCE4.
 *
 * The limits defined here must match the constraints documented in:
 * - pkg/workflow/js/safe_outputs_tools.json (add_comment tool description)
 * - docs/src/content/docs/reference/safe-outputs-specification.md
 *
 * This ensures constraint consistency per Requirement MCE5.
 */

/**
 * Maximum limits for comment parameters to prevent resource exhaustion.
 * These limits align with GitHub's API constraints and security best practices.
 */

/** @type {number} Maximum comment body length (GitHub's limit) */
const MAX_COMMENT_LENGTH = 65536;

/** @type {number} Maximum number of mentions allowed per comment */
const MAX_MENTIONS = 10;

/** @type {number} Maximum number of links allowed per comment */
const MAX_LINKS = 50;

/**
 * Enforces maximum limits on comment parameters to prevent resource exhaustion attacks.
 * Per Safe Outputs specification requirement MR3, limits must be enforced before API calls.
 *
 * @param {string} body - Comment body to validate
 * @throws {Error} When any limit is exceeded, with error code and details
 */
function enforceCommentLimits(body) {
  // Check body length - max limit exceeded check
  if (body.length > MAX_COMMENT_LENGTH) {
    throw new Error(`E006: Comment body exceeds maximum length of ${MAX_COMMENT_LENGTH} characters (got ${body.length})`);
  }

  // Count mentions (@username pattern) - max limit exceeded check
  const mentions = (body.match(/@\w+/g) || []).length;
  if (mentions > MAX_MENTIONS) {
    throw new Error(`E007: Comment contains ${mentions} mentions, maximum is ${MAX_MENTIONS}`);
  }

  // Count links (http:// and https:// URLs) - max limit exceeded check
  const links = (body.match(/https?:\/\/[^\s]+/g) || []).length;
  if (links > MAX_LINKS) {
    throw new Error(`E008: Comment contains ${links} links, maximum is ${MAX_LINKS}`);
  }
}

module.exports = {
  MAX_COMMENT_LENGTH,
  MAX_MENTIONS,
  MAX_LINKS,
  enforceCommentLimits,
};
