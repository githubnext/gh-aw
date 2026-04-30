// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_NOT_FOUND } = require("./error_codes.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");

const APPLY_SAFE_OUTPUTS_LABEL = "agentic-workflows:apply-safe-outputs";
const APPLY_SAFE_OUTPUTS_LABEL_COLOR = "8250df"; // GitHub purple
const APPLY_SAFE_OUTPUTS_LABEL_DESCRIPTION = "Re-apply the safe outputs from the agentic workflow run referenced in this issue";

/**
 * Extract a workflow run URL or numeric run ID from an issue body.
 *
 * Looks for (in priority order):
 * 1. Combined marker run field: <!-- gh-aw-agentic-workflow: ..., run: https://..., ... -->
 * 2. Combined marker id field:  <!-- gh-aw-agentic-workflow: ..., id: 12345, ... -->
 * 3. Standalone run-url marker: <!-- gh-aw-run-url: https://... -->
 *
 * @param {string|null|undefined} body - Issue body
 * @returns {string|null} Run URL or numeric run ID, or null if not found
 */
function extractRunUrl(body) {
  if (!body) return null;

  // 1. Combined marker — extract the run: field (full URL)
  const runUrlMatch = body.match(/<!--\s*gh-aw-agentic-workflow:[^>]*?\brun:\s*(https?:\/\/[^\s,>]+)/s);
  if (runUrlMatch) {
    return runUrlMatch[1].trim();
  }

  // 2. Combined marker — extract the id: field (numeric run ID)
  const idMatch = body.match(/<!--\s*gh-aw-agentic-workflow:[^>]*?\bid:\s*(\d+)/s);
  if (idMatch) {
    return idMatch[1].trim();
  }

  // 3. Standalone marker: <!-- gh-aw-run-url: https://... -->
  const standaloneMatch = body.match(/<!--\s*gh-aw-run-url:\s*(https?:\/\/[^\s>]+|[0-9]+)\s*-->/);
  if (standaloneMatch) {
    return standaloneMatch[1].trim();
  }

  return null;
}

/**
 * Ensure the "agentic-workflows:apply-safe-outputs" label exists in the repository.
 * Creates it with the standard purple color if it is missing.
 * This is a no-op (and non-fatal) when the label already exists.
 *
 * @param {string} owner
 * @param {string} repo
 * @returns {Promise<void>}
 */
async function ensureApplySafeOutputsLabelExists(owner, repo) {
  try {
    await github.rest.issues.createLabel({
      owner,
      repo,
      name: APPLY_SAFE_OUTPUTS_LABEL,
      color: APPLY_SAFE_OUTPUTS_LABEL_COLOR,
      description: APPLY_SAFE_OUTPUTS_LABEL_DESCRIPTION,
    });
    core.info(`✅ Created label '${APPLY_SAFE_OUTPUTS_LABEL}'`);
  } catch (err) {
    // 422 means the label already exists — expected on most runs
    if (err !== null && typeof err === "object" && /** @type {any} */ err.status === 422) {
      core.info(`ℹ️  Label '${APPLY_SAFE_OUTPUTS_LABEL}' already exists`);
    } else {
      // Non-fatal: log a warning but continue — the label may already be present
      core.warning(`Failed to ensure label '${APPLY_SAFE_OUTPUTS_LABEL}' exists: ${getErrorMessage(err)}`);
    }
  }
}

/**
 * Re-apply safe outputs from a previous workflow run when the
 * "agentic-workflows:apply-safe-outputs" label is applied to an issue.
 *
 * Reads the labeled issue body to extract a workflow run URL or run ID from XML comment
 * markers, re-applies the safe outputs from that run, posts a success comment, and
 * removes the label.
 *
 * @returns {Promise<void>}
 */
async function main() {
  const eventName = context.eventName;
  if (eventName !== "issues") {
    core.info(`Skipping: unexpected event type '${eventName}' (expected 'issues')`);
    return;
  }

  const { owner, repo } = resolveExecutionOwnerRepo();

  // Ensure the label exists so it is available for future use
  await ensureApplySafeOutputsLabelExists(owner, repo);

  // Get the issue from the payload
  const item = context.payload.issue;
  if (!item) {
    core.warning("No issue found in event payload");
    return;
  }

  const issueNumber = item.number;
  const labelName = context.payload.label?.name;

  if (labelName !== APPLY_SAFE_OUTPUTS_LABEL) {
    core.info(`Skipping: label '${labelName}' is not '${APPLY_SAFE_OUTPUTS_LABEL}'`);
    return;
  }

  core.info(`Processing issue #${issueNumber} labeled with '${labelName}'`);

  // Extract run URL from body XML comment markers
  const body = item.body || "";
  const runUrl = extractRunUrl(body);

  if (!runUrl) {
    core.warning(`Could not find run URL in issue #${issueNumber} body.`);
    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: issueNumber,
      body:
        `> [!WARNING]\n` +
        `> **Could not apply safe outputs**\n>\n` +
        `> No workflow run reference was found in this issue's body. ` +
        `The \`${APPLY_SAFE_OUTPUTS_LABEL}\` label can only be used on issues that were created by an agentic workflow ` +
        `(they contain a \`<!-- gh-aw-agentic-workflow: ..., run: https://... -->\` marker with a run URL).\n>\n` +
        `> To apply safe outputs manually, use the maintenance workflow with the \`safe_outputs\` operation and supply the run URL.`,
    });
    core.setFailed(`${ERR_NOT_FOUND}: No run URL marker found in issue #${issueNumber}`);
    return;
  }

  core.info(`Found run reference: ${runUrl}`);

  // Set GH_AW_RUN_URL so apply_safe_outputs_replay.cjs can consume it
  process.env.GH_AW_RUN_URL = runUrl;

  // Delegate to the existing replay driver
  const { main: replayMain } = require("./apply_safe_outputs_replay.cjs");
  try {
    await replayMain();
  } catch (err) {
    const msg = getErrorMessage(err);
    core.error(`Failed to apply safe outputs: ${msg}`);
    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: issueNumber,
      body:
        `> [!WARNING]\n` +
        `> **Failed to apply safe outputs from run \`${runUrl}\`**\n>\n` +
        `> ${msg}\n>\n` +
        `> Please check the [workflow run logs](${process.env.GITHUB_SERVER_URL || "https://github.com"}/${owner}/${repo}/actions/runs/${process.env.GITHUB_RUN_ID || ""}) for details.`,
    });
    core.setFailed(`Failed to apply safe outputs: ${msg}`);
    return;
  }

  core.info(`Successfully applied safe outputs from run ${runUrl}`);

  // Post a success comment on the issue
  await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
    body: `✅ Safe outputs from [run \`${runUrl}\`](${runUrl}) have been applied.\n\n` + `<!-- gh-aw-comment-type: safe-outputs-applied -->`,
  });

  core.info(`Posted success comment on issue #${issueNumber}`);

  // Remove the label now that the action is complete
  try {
    await github.rest.issues.removeLabel({
      owner,
      repo,
      issue_number: issueNumber,
      name: APPLY_SAFE_OUTPUTS_LABEL,
    });
    core.info(`Removed label '${APPLY_SAFE_OUTPUTS_LABEL}' from issue #${issueNumber}`);
  } catch (err) {
    // Non-fatal: the apply already succeeded, just log a warning
    core.warning(`Failed to remove label '${APPLY_SAFE_OUTPUTS_LABEL}': ${getErrorMessage(err)}`);
  }
}

module.exports = { main, extractRunUrl, ensureApplySafeOutputsLabelExists };
