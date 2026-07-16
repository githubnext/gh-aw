// @ts-check
/// <reference types="@actions/github-script" />

/** @type {typeof import("crypto")} */
const crypto = require("crypto");
const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { isTransientError } = require("./error_recovery.cjs");
const { tryEnforceArrayLimit } = require("./limit_enforcement_helpers.cjs");
const { MAX_ASSIGNEES } = require("./constants.cjs");
const { encodePathSegments, renderTemplateFromFile, getPromptPath } = require("./messages_core.cjs");

/** @type {string} Label always added to fallback issues so the triage system can find them */
const MANAGED_FALLBACK_ISSUE_LABEL = "agentic-workflows";

/** @type {number} Number of retry attempts for label operations */
const LABEL_MAX_RETRIES = 5;
/** @type {number} Base delay in ms used to calculate label retry backoff (3 seconds) */
const LABEL_INITIAL_DELAY_MS = 3000;
/** @type {number} Maximum delay in ms between label retries (30 seconds) */
const LABEL_MAX_DELAY_MS = 30000;

/**
 * Summarize a list for log output to avoid excessively long lines.
 * @param {string[]} values
 * @param {number} limit
 * @returns {string}
 */
function summarizeListForLog(values, limit = 10) {
  if (!Array.isArray(values) || values.length === 0) {
    return "(none)";
  }
  const preview = values.slice(0, limit).join(", ");
  return values.length > limit ? `${preview} ... and ${values.length - limit} more` : preview;
}

/**
 * Creates a temporary refs/bundles ref for applying create_pull_request bundles.
 * Branch names are sanitized for ref compatibility, and a short crypto-random
 * suffix avoids collisions between branches that sanitize to the same value.
 *
 * @param {string} branchName - Target branch name
 * @returns {string} Temporary bundle ref name
 */
function createBundleTempRef(branchName) {
  const suffix = crypto.randomBytes(4).toString("hex");
  return `refs/bundles/create-pr-${branchName.replace(/[^a-zA-Z0-9-]/g, "-")}-${suffix}`;
}

/**
 * Determines if a label API error is transient and worth retrying.
 * Returns true for:
 *  - The GitHub race condition where a newly-created PR's node ID is not immediately
 *    resolvable via the REST/GraphQL bridge (unprocessable validation error).
 *  - Any standard transient error matched by {@link isTransientError} (network issues,
 *    rate limits, 5xx gateway errors, etc.).
 * @param {any} error - The error to check
 * @returns {boolean} True if the error is transient and should be retried
 */
function isLabelTransientError(error) {
  const msg = getErrorMessage(error);
  if (msg.includes("Could not resolve to a node with the global id")) {
    return true;
  }
  return isTransientError(error);
}

/**
 * Parse allowed base branch patterns from config value (array or comma-separated string)
 * @param {string[]|string|undefined} allowedBaseBranchesValue
 * @returns {Set<string>}
 */
function parseAllowedBaseBranches(allowedBaseBranchesValue) {
  const set = new Set();
  if (Array.isArray(allowedBaseBranchesValue)) {
    allowedBaseBranchesValue
      .map(branch => String(branch).trim())
      .filter(Boolean)
      .forEach(branch => set.add(branch));
  } else if (typeof allowedBaseBranchesValue === "string") {
    allowedBaseBranchesValue
      .split(",")
      .map(branch => branch.trim())
      .filter(Boolean)
      .forEach(branch => set.add(branch));
  }
  return set;
}

/**
 * Check if a base branch matches an allowed pattern.
 * Supports exact matches and "*" glob patterns (e.g. "release/*").
 * @param {string} baseBranch
 * @param {Set<string>} allowedBaseBranches
 * @returns {boolean}
 */
function isBaseBranchAllowed(baseBranch, allowedBaseBranches) {
  if (allowedBaseBranches.has(baseBranch)) {
    return true;
  }
  for (const pattern of allowedBaseBranches) {
    if (pattern === "*") {
      return true;
    }
    if (pattern.includes("*") && globPatternToRegex(pattern, { pathMode: true, caseSensitive: true }).test(baseBranch)) {
      return true;
    }
  }
  return false;
}

/**
 * Parse config values that may be arrays or comma-separated strings.
 * @param {string[]|string|undefined} value
 * @returns {string[]}
 */
function parseStringListConfig(value) {
  if (!value) {
    return [];
  }
  const raw = Array.isArray(value) ? value : String(value).split(",");
  return raw.map(item => String(item).trim()).filter(Boolean);
}

/**
 * Merges the required fallback label with any workflow-configured labels,
 * deduplicating and filtering empty values.
 * @param {string[]} [labels]
 * @returns {string[]}
 */
function mergeFallbackIssueLabels(labels = []) {
  const normalizedLabels = labels
    .filter(label => !!label)
    .map(label => String(label).trim())
    .filter(label => label);
  return [...new Set([MANAGED_FALLBACK_ISSUE_LABEL, ...normalizedLabels])];
}

/**
 * Sanitizes configured assignees for fallback issue creation.
 * Filters invalid values, removes the special "copilot" username (not a valid GitHub user
 * for issue assignment), and enforces the MAX_ASSIGNEES limit.
 * Returns null (no assignees field) if the sanitized list is empty.
 * @param {string[]} assignees - Raw assignees from config
 * @returns {string[] | null} Sanitized assignees or null if none remain
 */
function sanitizeFallbackAssignees(assignees) {
  if (!assignees || assignees.length === 0) {
    return null;
  }
  const sanitized = assignees
    .filter(a => typeof a === "string")
    .map(a => a.trim())
    .filter(a => a.length > 0 && a.toLowerCase() !== "copilot");

  if (sanitized.length === 0) {
    return null;
  }

  const limitResult = tryEnforceArrayLimit(sanitized, MAX_ASSIGNEES, "assignees");
  if (!limitResult.success) {
    core.warning(`Assignees limit exceeded for fallback issue: ${limitResult.error}. Using first ${MAX_ASSIGNEES}.`);
    return sanitized.slice(0, MAX_ASSIGNEES);
  }

  return sanitized;
}

/**
 * Neutralizes issue-closing keywords in body text to avoid unintended cross-issue closure
 * when PR content is reused in fallback issue bodies.
 *
 * Example: "Closes #123" -> "Closes \\#123"
 *
 * @param {string} content
 * @returns {string}
 */
function neutralizeClosingKeywordsForIssueBody(content) {
  if (!content) {
    return content;
  }
  const closingKeywordPattern = /\b(fix|fixes|fixed|close|closes|closed|resolve|resolves|resolved)\s+((?:[a-z0-9_.-]+\/[a-z0-9_.-]+)?#\d+)\b/gi;
  const escapeIssueRef = (_match, keyword, issueRef) => `${keyword} ${String(issueRef).replace("#", "\\#")}`;
  return String(content).replace(closingKeywordPattern, escapeIssueRef);
}

/**
 * Generate a patch preview with max 500 lines and 2000 chars for issue body
 * @param {string} patchContent - The full patch content
 * @returns {string} Formatted patch preview
 */
function generatePatchPreview(patchContent) {
  if (!patchContent || !patchContent.trim()) {
    return "";
  }

  const lines = patchContent.split("\n");
  const maxLines = 500;
  const maxChars = 2000;

  // Apply line limit first
  let preview = lines.length <= maxLines ? patchContent : lines.slice(0, maxLines).join("\n");
  const lineTruncated = lines.length > maxLines;

  // Apply character limit
  const charTruncated = preview.length > maxChars;
  if (charTruncated) {
    preview = preview.slice(0, maxChars);
  }

  const truncated = lineTruncated || charTruncated;
  const summary = truncated ? `Show patch preview (${Math.min(maxLines, lines.length)} of ${lines.length} lines)` : `Show patch (${lines.length} lines)`;

  return `\n\n<details><summary>${summary}</summary>\n\n\`\`\`diff\n${preview}${truncated ? "\n... (truncated)" : ""}\n\`\`\`\n\n</details>`;
}

/**
 * Builds a compare URL used in protected-files fallback issue bodies.
 * Optionally appends a prefilled PR body that closes the fallback issue.
 * @param {string} githubServer
 * @param {{owner: string, repo: string}} repoParts
 * @param {string} baseBranch
 * @param {string} branchName
 * @param {string} title
 * @param {number} [fallbackIssueNumber]
 * @param {string} [headRef]
 * @returns {string}
 */
function buildManifestProtectionCreatePrUrl(githubServer, repoParts, baseBranch, branchName, title, fallbackIssueNumber, headRef) {
  const encodedBase = encodePathSegments(baseBranch);
  const encodedHead = encodePathSegments(headRef || branchName);
  let createPrUrl = `${githubServer}/${repoParts.owner}/${repoParts.repo}/compare/${encodedBase}...${encodedHead}?expand=1&title=${encodeURIComponent(title)}`;
  if (typeof fallbackIssueNumber === "number") {
    createPrUrl += `&body=${encodeURIComponent(`Closes #${fallbackIssueNumber}`)}`;
  }
  return createPrUrl;
}

/**
 * Renders protected-files fallback issue body with a prefilled compare URL.
 * @param {string} mainBodyContent
 * @param {string} footerContent
 * @param {string} fileList
 * @param {string} createPrUrl
 * @returns {string}
 */
function renderManifestProtectionFallbackBody(mainBodyContent, footerContent, fileList, createPrUrl) {
  const templatePath = getPromptPath("manifest_protection_create_pr_fallback.md");
  return renderTemplateFromFile(templatePath, {
    main_body: mainBodyContent,
    footer: footerContent,
    files: fileList,
    create_pr_url: createPrUrl,
  });
}

module.exports = {
  MANAGED_FALLBACK_ISSUE_LABEL,
  LABEL_MAX_RETRIES,
  LABEL_INITIAL_DELAY_MS,
  LABEL_MAX_DELAY_MS,
  summarizeListForLog,
  createBundleTempRef,
  isLabelTransientError,
  parseAllowedBaseBranches,
  isBaseBranchAllowed,
  parseStringListConfig,
  mergeFallbackIssueLabels,
  sanitizeFallbackAssignees,
  neutralizeClosingKeywordsForIssueBody,
  generatePatchPreview,
  buildManifestProtectionCreatePrUrl,
  renderManifestProtectionFallbackBody,
};
