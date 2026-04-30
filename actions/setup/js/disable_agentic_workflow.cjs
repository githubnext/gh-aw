// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_NOT_FOUND } = require("./error_codes.cjs");
const { ensureLabelExists, validateLabeledIssueEvent, removeLabelSafely } = require("./label_trigger_helpers.cjs");

const DISABLE_LABEL = "agentic-workflows:disable";
const DISABLE_LABEL_COLOR = "8250df"; // GitHub purple
const DISABLE_LABEL_DESCRIPTION = "Disable the agentic workflow that created this issue or pull request";

/**
 * Validate that an extracted workflow ID has a safe, expected format.
 * Workflow IDs are file basenames (without .md) and must not contain
 * path traversal sequences or other shell-unsafe characters.
 *
 * @param {string} id - Candidate workflow ID
 * @returns {boolean} True if the ID is safe to use as a CLI argument
 */
function isValidWorkflowId(id) {
  // Allow alphanumeric characters, hyphens, underscores, and dots.
  // Reject anything else, as well as path traversal sequences like "..".
  return id.length > 0 && id.length <= 100 && /^[\w.-]+$/.test(id) && !id.includes("..");
}

/**
 * Extract the workflow_id from an issue or pull request body using XML comment markers.
 *
 * Looks for (in priority order):
 * 1. Standalone marker: <!-- gh-aw-workflow-id: my-workflow -->
 * 2. Combined marker: <!-- gh-aw-agentic-workflow: ..., workflow_id: my-workflow, ... -->
 * 3. Workflow-call-id marker: <!-- gh-aw-workflow-call-id: owner/repo/my-workflow -->
 *    (extracts the last path segment to get the workflow ID)
 *
 * The combined and call-id markers are only searched within actual HTML comment blocks
 * to prevent unintended matches in user-provided content.
 *
 * @param {string|null|undefined} body - Issue or PR body
 * @returns {string|null} Workflow ID or null if not found or invalid
 */
function extractWorkflowId(body) {
  if (!body) return null;

  // Try standalone marker: <!-- gh-aw-workflow-id: my-workflow -->
  const standaloneMatch = body.match(/<!--\s*gh-aw-workflow-id:\s*([\w.-]+)\s*-->/);
  if (standaloneMatch) {
    const id = standaloneMatch[1].trim();
    return isValidWorkflowId(id) ? id : null;
  }

  // Try combined marker, but only within HTML comment blocks that contain
  // gh-aw-agentic-workflow: to avoid matching user content.
  const commentMatch = body.match(/<!--\s*gh-aw-agentic-workflow:[^>]*?workflow_id:\s*([\w.-]+)[\s,>]/s);
  if (commentMatch) {
    const id = commentMatch[1].trim();
    return isValidWorkflowId(id) ? id : null;
  }

  // Try workflow-call-id marker (handles workflow_dispatch): <!-- gh-aw-workflow-call-id: owner/repo/my-workflow -->
  // The call-id has the form "owner/repo/workflow-id"; extract the last non-empty path segment.
  const callIdMatch = body.match(/<!--\s*gh-aw-workflow-call-id:\s*([^\s>][^>]*?)\s*-->/);
  if (callIdMatch) {
    const segments = callIdMatch[1].trim().split("/");
    const id = segments[segments.length - 1].trim();
    if (id.length === 0) return null;
    return isValidWorkflowId(id) ? id : null;
  }

  return null;
}

/**
 * Disable an agentic workflow when the "agentic-workflows:disable" label is applied to an issue.
 *
 * Reads the labeled issue body to extract the workflow_id from XML comment markers,
 * disables the corresponding agentic workflow via the GitHub REST API, and posts a comment
 * confirming the action.
 *
 * @returns {Promise<void>}
 */
async function main() {
  const ctx = validateLabeledIssueEvent(DISABLE_LABEL);
  if (!ctx) return;

  const { owner, repo, issueNumber, body } = ctx;

  // Ensure the disable label exists so it is available for future use
  await ensureLabelExists(owner, repo, DISABLE_LABEL, DISABLE_LABEL_COLOR, DISABLE_LABEL_DESCRIPTION);

  core.info(`Processing issue #${issueNumber} labeled with '${DISABLE_LABEL}'`);

  // Extract workflow ID from body XML comment markers
  const workflowId = extractWorkflowId(body);

  if (!workflowId) {
    core.warning(`Could not find workflow ID in issue #${issueNumber} body. Expected a <!-- gh-aw-workflow-id: ... --> marker.`);
    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: issueNumber,
      body:
        `> [!WARNING]\n` +
        `> **Could not disable agentic workflow**\n>\n` +
        `> No workflow ID marker was found in this issue's body. ` +
        `The \`${DISABLE_LABEL}\` label can only be used on issues that were created by an agentic workflow ` +
        `(they contain a \`<!-- gh-aw-workflow-id: ... -->\` marker).\n>\n` +
        `> To disable a workflow manually, trigger the maintenance workflow with the \`disable\` operation.`,
    });
    core.setFailed(`${ERR_NOT_FOUND}: No workflow ID marker found in issue #${issueNumber}`);
    return;
  }

  core.info(`Found workflow ID: ${workflowId}`);
  core.info(`Disabling agentic workflow '${workflowId}'...`);

  // Disable the workflow via the GitHub REST API using its compiled lock file name
  const lockFileName = `${workflowId}.lock.yml`;
  try {
    await github.rest.actions.disableWorkflow({ owner, repo, workflow_id: lockFileName });
  } catch (err) {
    const msg = getErrorMessage(err);
    core.error(`Failed to disable workflow '${workflowId}': ${msg}`);
    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: issueNumber,
      body:
        `> [!WARNING]\n` +
        `> **Failed to disable agentic workflow \`${workflowId}\`**\n>\n` +
        `> ${msg}\n>\n` +
        `> Please check the [workflow run logs](${process.env.GITHUB_SERVER_URL || "https://github.com"}/${owner}/${repo}/actions/runs/${process.env.GITHUB_RUN_ID || ""}) for details.`,
    });
    core.setFailed(`Failed to disable workflow '${workflowId}': ${msg}`);
    return;
  }

  core.info(`Successfully disabled workflow '${workflowId}'`);

  // Post a success comment on the issue
  await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
    body: `The agentic workflow \`${workflowId}\` has been disabled.\n\n` + `To re-enable it, trigger the maintenance workflow with the \`enable\` operation.\n\n` + `<!-- gh-aw-comment-type: workflow-disabled -->`,
  });

  core.info(`Posted disable confirmation comment on issue #${issueNumber}`);

  // Remove the disable label now that the action is complete
  await removeLabelSafely(owner, repo, issueNumber, DISABLE_LABEL);
}

module.exports = { main, extractWorkflowId, isValidWorkflowId };
