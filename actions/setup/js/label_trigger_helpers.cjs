// @ts-check
/// <reference types="@actions/github-script" />

// @safe-outputs-exempt SEC-004 — this helper only reads the raw issue body from the GitHub event
// payload and returns it as a plain string for callers to parse; it never writes body content to
// any GitHub API, so no sanitization is required here.

const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");

/**
 * Ensures a label exists in the repository, creating it if necessary.
 * A 422 response means the label already exists (expected on most runs).
 * Other errors are non-fatal and logged as warnings.
 *
 * @param {string} owner
 * @param {string} repo
 * @param {string} name - Label name
 * @param {string} color - Hex color without '#' (e.g. "8250df")
 * @param {string} description - Short label description
 * @returns {Promise<void>}
 */
async function ensureLabelExists(owner, repo, name, color, description) {
  try {
    await github.rest.issues.createLabel({ owner, repo, name, color, description });
    core.info(`✅ Created label '${name}'`);
  } catch (err) {
    if (err !== null && typeof err === "object" && /** @type {any} */ err.status === 422) {
      core.info(`ℹ️  Label '${name}' already exists`);
    } else {
      core.warning(`Failed to ensure label '${name}' exists: ${getErrorMessage(err)}`);
    }
  }
}

/**
 * Validates that the current GitHub Actions event is an 'issues: labeled' event
 * matching the given label. Resolves the owner/repo and reads the issue from the payload.
 *
 * Returns { owner, repo, issueNumber, body } on success, or null if the event
 * should be silently skipped (wrong event type, missing payload, or wrong label).
 *
 * @param {string} expectedLabel - The label name to match
 * @returns {{ owner: string, repo: string, issueNumber: number, body: string } | null}
 */
function validateLabeledIssueEvent(expectedLabel) {
  const eventName = context.eventName;
  if (eventName !== "issues") {
    core.info(`Skipping: unexpected event type '${eventName}' (expected 'issues')`);
    return null;
  }

  const item = context.payload.issue;
  if (!item) {
    core.warning("No issue found in event payload");
    return null;
  }

  const labelName = context.payload.label?.name;
  if (labelName !== expectedLabel) {
    core.info(`Skipping: label '${labelName}' is not '${expectedLabel}'`);
    return null;
  }

  const { owner, repo } = resolveExecutionOwnerRepo();
  return {
    owner,
    repo,
    issueNumber: item.number,
    body: item.body || "",
  };
}

/**
 * Removes a label from an issue. Non-fatal: logs a warning on failure.
 *
 * @param {string} owner
 * @param {string} repo
 * @param {number} issueNumber
 * @param {string} labelName
 * @returns {Promise<void>}
 */
async function removeLabelSafely(owner, repo, issueNumber, labelName) {
  try {
    await github.rest.issues.removeLabel({ owner, repo, issue_number: issueNumber, name: labelName });
    core.info(`Removed label '${labelName}' from issue #${issueNumber}`);
  } catch (err) {
    core.warning(`Failed to remove label '${labelName}': ${getErrorMessage(err)}`);
  }
}

module.exports = { ensureLabelExists, validateLabeledIssueEvent, removeLabelSafely };
