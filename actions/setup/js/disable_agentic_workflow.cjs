// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_NOT_FOUND } = require("./error_codes.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");

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
 * Ensure the "agentic-workflows:disable" label exists in the repository.
 * Creates it with the standard purple color if it is missing.
 * This is a no-op (and non-fatal) when the label already exists.
 *
 * @param {string} owner
 * @param {string} repo
 * @returns {Promise<void>}
 */
async function ensureDisableLabelExists(owner, repo) {
  try {
    await github.rest.issues.createLabel({
      owner,
      repo,
      name: DISABLE_LABEL,
      color: DISABLE_LABEL_COLOR,
      description: DISABLE_LABEL_DESCRIPTION,
    });
    core.info(`✅ Created label '${DISABLE_LABEL}'`);
  } catch (err) {
    // 422 means the label already exists — expected on most runs
    if (err !== null && typeof err === "object" && /** @type {any} */ err.status === 422) {
      core.info(`ℹ️  Label '${DISABLE_LABEL}' already exists`);
    } else {
      // Non-fatal: log a warning but continue — the label may already be present
      core.warning(`Failed to ensure label '${DISABLE_LABEL}' exists: ${getErrorMessage(err)}`);
    }
  }
}

/**
 * Disable an agentic workflow when the "agentic-workflows:disable" label is applied to an issue.
 *
 * Reads the labeled issue body to extract the workflow_id from XML comment markers,
 * disables the corresponding agentic workflow using `gh aw disable`, and posts a comment
 * confirming the action.
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

  // Ensure the disable label exists so it is available for future use
  await ensureDisableLabelExists(owner, repo);

  // Get the issue from the payload
  const item = context.payload.issue;
  if (!item) {
    core.warning("No issue found in event payload");
    return;
  }

  const itemNumber = item.number;
  const labelName = context.payload.label?.name;

  if (labelName !== DISABLE_LABEL) {
    core.info(`Skipping: label '${labelName}' is not '${DISABLE_LABEL}'`);
    return;
  }

  core.info(`Processing issue #${itemNumber} labeled with '${labelName}'`);

  // Extract workflow ID from body XML comment markers
  const body = item.body || "";
  const workflowId = extractWorkflowId(body);

  if (!workflowId) {
    core.warning(`Could not find workflow ID in issue #${itemNumber} body. Expected a <!-- gh-aw-workflow-id: ... --> marker.`);
    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: itemNumber,
      body:
        `> [!WARNING]\n` +
        `> **Could not disable agentic workflow**\n>\n` +
        `> No workflow ID marker was found in this issue's body. ` +
        `The \`${DISABLE_LABEL}\` label can only be used on issues that were created by an agentic workflow ` +
        `(they contain a \`<!-- gh-aw-workflow-id: ... -->\` marker).\n>\n` +
        `> To disable a workflow manually, use:\n` +
        `> \`\`\`\n` +
        `> gh aw disable <workflow-id>\n` +
        `> \`\`\``,
    });
    core.setFailed(`${ERR_NOT_FOUND}: No workflow ID marker found in issue #${itemNumber}`);
    return;
  }

  core.info(`Found workflow ID: ${workflowId}`);

  // Disable the workflow using gh aw disable <workflow_id>
  const cmdPrefixStr = process.env.GH_AW_CMD_PREFIX || "gh aw";
  const [bin, ...prefixArgs] = cmdPrefixStr.split(" ").filter(Boolean);

  core.info(`Disabling agentic workflow '${workflowId}'...`);
  let exitCode;
  try {
    exitCode = await exec.exec(bin, [...prefixArgs, "disable", workflowId], {
      env: {
        HOME: process.env.HOME || "",
        PATH: process.env.PATH || "",
        GH_TOKEN: process.env.GH_TOKEN || process.env.GITHUB_TOKEN || "",
        GITHUB_TOKEN: process.env.GITHUB_TOKEN || "",
        GITHUB_REPOSITORY: process.env.GITHUB_REPOSITORY || "",
        GITHUB_SERVER_URL: process.env.GITHUB_SERVER_URL || "https://github.com",
        GH_AW_CMD_PREFIX: cmdPrefixStr,
      },
      ignoreReturnCode: true,
    });
  } catch (err) {
    const msg = getErrorMessage(err);
    core.error(`Failed to run disable command: ${msg}`);
    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: itemNumber,
      body:
        `> [!WARNING]\n` +
        `> **Failed to disable agentic workflow \`${workflowId}\`**\n>\n` +
        `> The disable command encountered an error: ${msg}\n>\n` +
        `> Please check the [workflow run logs](${process.env.GITHUB_SERVER_URL || "https://github.com"}/${owner}/${repo}/actions/runs/${process.env.GITHUB_RUN_ID || ""}) for details.`,
    });
    core.setFailed(`Failed to disable workflow '${workflowId}': ${msg}`);
    return;
  }

  if (exitCode !== 0) {
    const msg = `Command exited with code ${exitCode}`;
    core.error(msg);
    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: itemNumber,
      body:
        `> [!WARNING]\n` +
        `> **Failed to disable agentic workflow \`${workflowId}\`**\n>\n` +
        `> The \`gh aw disable ${workflowId}\` command failed (exit code ${exitCode}).\n>\n` +
        `> Please check the [workflow run logs](${process.env.GITHUB_SERVER_URL || "https://github.com"}/${owner}/${repo}/actions/runs/${process.env.GITHUB_RUN_ID || ""}) for details.`,
    });
    core.setFailed(`gh aw disable '${workflowId}' failed with exit code ${exitCode}`);
    return;
  }

  core.info(`Successfully disabled workflow '${workflowId}'`);

  // Post a success comment on the issue
  await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: itemNumber,
    body:
      `The agentic workflow \`${workflowId}\` has been disabled.\n\n` +
      `To re-enable it, use:\n` +
      `\`\`\`\n` +
      `gh aw enable ${workflowId}\n` +
      `\`\`\`\n\n` +
      `Or trigger the maintenance workflow with the \`enable\` operation.\n\n` +
      `<!-- gh-aw-comment-type: workflow-disabled -->`,
  });

  core.info(`Posted disable confirmation comment on issue #${itemNumber}`);

  // Remove the disable label now that the action is complete
  try {
    await github.rest.issues.removeLabel({
      owner,
      repo,
      issue_number: itemNumber,
      name: DISABLE_LABEL,
    });
    core.info(`Removed label '${DISABLE_LABEL}' from issue #${itemNumber}`);
  } catch (err) {
    // Non-fatal: the disable already succeeded, just log a warning
    core.warning(`Failed to remove label '${DISABLE_LABEL}': ${getErrorMessage(err)}`);
  }
}

module.exports = { main, extractWorkflowId, isValidWorkflowId, ensureDisableLabelExists };
