// @ts-check

const { sanitizeTitle } = require("./sanitize_title.cjs");

const TRIVIAL_PROBE_VALUES = new Set(["test", "testing", "test no base", "probe", "placeholder", "dummy", "temp", "temporary", "todo", "tbd", "wip", "example"]);

/**
 * @param {unknown} value
 * @returns {string}
 */
function normalizeProbeValue(value) {
  if (typeof value !== "string") return "";
  return value.trim().toLowerCase().replace(/\s+/g, " ");
}

/**
 * @param {unknown} value
 * @param {string} fieldName
 * @returns {boolean}
 */
function isTrivialProbeValue(value, fieldName) {
  const normalized = normalizeProbeValue(value);
  return TRIVIAL_PROBE_VALUES.has(normalized) || normalized === `test ${fieldName}`;
}

/**
 * Detects noop messages that look like a schema probe (e.g. an agent testing
 * tool availability) rather than a genuine completion signal. Probing noop
 * calls are ignored entirely - they don't consume the noop call budget and
 * aren't surfaced in summaries or tracking issues - so a real noop call is
 * never crowded out by an earlier probe.
 * @param {unknown} message
 * @returns {boolean}
 */
function isProbingNoopMessage(message) {
  return isTrivialProbeValue(message, "message");
}

/**
 * @param {unknown} value
 * @returns {boolean}
 */
function looksLikeExploratoryBranch(value) {
  const branch = normalizeProbeValue(value);
  const hasProbeBranchMarker = /(^|[-/])probe([-/]|$)/.test(branch);
  return branch.includes("test-from-main") || isTrivialProbeValue(branch, "branch") || hasProbeBranchMarker;
}

/**
 * @param {{title?: unknown, body?: unknown}} entry
 * @returns {string}
 */
function resolveIssueTitleForValidation(entry) {
  if (typeof entry.title === "string" && entry.title.trim()) {
    return entry.title;
  }
  if (typeof entry.body === "string" && entry.body.trim()) {
    // Mirror create_issue's fallback: use the first non-empty line of the body.
    const firstBodyLine = entry.body
      .split("\n")
      .map(l => l.replace(/^#+\s*/, "").trim())
      .find(l => l.length > 0);
    if (firstBodyLine) return firstBodyLine;
  }
  // Mirror create_issue's own fallback so probe validation sees the same
  // effective title the handler would ultimately use when both fields are absent.
  return "Agent Output";
}

/**
 * Detects obviously exploratory PR payloads so the agent can fail fast
 * instead of recording a stray real-world PR intent.
 * @param {{title?: unknown, body?: unknown, branch?: unknown}} entry
 * @returns {string|null}
 */
function validateCreatePullRequestIntent(entry) {
  const title = normalizeProbeValue(entry.title);
  const body = normalizeProbeValue(entry.body);
  const branch = normalizeProbeValue(entry.branch);

  const looksLikeProbeTitle = isTrivialProbeValue(title, "title");
  const looksLikeProbeBody = isTrivialProbeValue(body, "body");
  const looksLikeTestFromMainBranch = branch.includes("test-from-main");
  const looksLikeProbeBranch = looksLikeExploratoryBranch(branch);

  if (looksLikeTestFromMainBranch || (looksLikeProbeTitle && looksLikeProbeBody) || (looksLikeProbeBranch && (looksLikeProbeTitle || looksLikeProbeBody))) {
    return (
      "Refusing to record an exploratory pull request. " +
      "create_pull_request is for a real intended PR only and successful calls can lead to an externally visible pull request. " +
      "Do not use placeholder values like 'test' or probe branches. " +
      "If you are not ready to open the real PR, use noop or report_incomplete instead."
    );
  }

  return null;
}

/**
 * Detects obviously exploratory issue payloads so the agent can fail fast
 * instead of recording a stray real-world issue intent.
 * @param {{title?: unknown, body?: unknown}} entry
 * @param {string} [titlePrefix]
 * @returns {string|null}
 */
function validateCreateIssueIntent(entry, titlePrefix = "") {
  const title = normalizeProbeValue(sanitizeTitle(resolveIssueTitleForValidation(entry), titlePrefix));
  const titleField = typeof entry.title === "string" && entry.title.trim() ? "title" : "body";
  const looksLikeProbeTitle = isTrivialProbeValue(title, titleField) || [...TRIVIAL_PROBE_VALUES].some(word => title === `${word} title`);

  if (looksLikeProbeTitle) {
    return (
      "Refusing to record an exploratory issue. " +
      "create_issue is for a real intended issue only and successful calls can lead to an externally visible issue. " +
      "Replace the placeholder title with a descriptive title for the real issue before retrying. " +
      "If you are not ready to open the real issue, use noop or report_incomplete instead."
    );
  }

  return null;
}

/**
 * Detects obviously exploratory comment payloads so the agent can fail fast
 * instead of recording a stray real-world comment intent.
 * @param {{body?: unknown}} entry
 * @returns {string|null}
 */
function validateAddCommentIntent(entry) {
  if (isTrivialProbeValue(entry.body, "body")) {
    return (
      "Refusing to record an exploratory comment. " +
      "add_comment is for a real intended comment only and successful calls can lead to an externally visible comment. " +
      "Do not use placeholder bodies like 'test'. " +
      "If you are not ready to post the real comment, use noop or report_incomplete instead."
    );
  }

  return null;
}

/**
 * Detects obviously exploratory PR-branch push payloads so the agent can fail fast
 * instead of recording a stray real-world branch update intent.
 * @param {{branch?: unknown, message?: unknown}} entry
 * @returns {string|null}
 */
function validatePushToPullRequestBranchIntent(entry) {
  const branch = normalizeProbeValue(entry.branch);
  const looksLikeTestFromMainBranch = branch.includes("test-from-main");
  const looksLikeProbeBranch = looksLikeExploratoryBranch(branch);
  const looksLikeProbeMessage = isTrivialProbeValue(entry.message, "message");

  if (looksLikeTestFromMainBranch || isTrivialProbeValue(branch, "branch") || (looksLikeProbeBranch && looksLikeProbeMessage)) {
    return (
      "Refusing to record an exploratory pull request branch update. " +
      "push_to_pull_request_branch is for a real intended PR update only and successful calls can lead to externally visible branch changes. " +
      "Do not use probe branches, '*-test-from-main-*' branches, or placeholder commit messages like 'test'. " +
      "If you are not ready to push the real update, use noop or report_incomplete instead."
    );
  }

  return null;
}

module.exports = {
  isProbingNoopMessage,
  looksLikeExploratoryBranch,
  normalizeProbeValue,
  resolveIssueTitleForValidation,
  validateAddCommentIntent,
  validateCreateIssueIntent,
  validateCreatePullRequestIntent,
  validatePushToPullRequestBranchIntent,
};
