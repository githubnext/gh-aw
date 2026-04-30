// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_NOT_FOUND } = require("./error_codes.cjs");
const { resolveExecutionOwnerRepo } = require("./repo_helpers.cjs");

const DISABLE_LABEL = "agentic-workflows:disable";

/**
 * Extract the workflow_id from an issue or pull request body using XML comment markers.
 *
 * Looks for:
 * 1. Standalone marker: <!-- gh-aw-workflow-id: my-workflow -->
 * 2. Combined marker: <!-- gh-aw-agentic-workflow: ..., workflow_id: my-workflow, ... -->
 *
 * @param {string|null|undefined} body - Issue or PR body
 * @returns {string|null} Workflow ID or null if not found
 */
function extractWorkflowId(body) {
  if (!body) return null;

  // Try standalone marker: <!-- gh-aw-workflow-id: my-workflow -->
  const standaloneMatch = body.match(/<!--\s*gh-aw-workflow-id:\s*([\w.-]+)\s*-->/);
  if (standaloneMatch) {
    return standaloneMatch[1].trim();
  }

  // Try combined marker: <!-- gh-aw-agentic-workflow: ..., workflow_id: my-workflow, ... -->
  // Match workflow_id: value followed by comma, space, or end of comment
  const combinedMatch = body.match(/workflow_id:\s*([\w.-]+)(?:[,\s]|-->)/);
  if (combinedMatch) {
    return combinedMatch[1].trim();
  }

  return null;
}

/**
 * Disable an agentic workflow when the "agentic-workflows:disable" label is applied to an issue or PR.
 *
 * Reads the labeled issue/PR body to extract the workflow_id from XML comment markers,
 * disables the corresponding agentic workflow using `gh aw disable`, and posts a comment
 * confirming the action.
 *
 * @returns {Promise<void>}
 */
async function main() {
  const eventName = context.eventName;
  if (eventName !== "issues" && eventName !== "pull_request") {
    core.info(`Skipping: unexpected event type '${eventName}'`);
    return;
  }

  const { owner, repo } = resolveExecutionOwnerRepo();

  // Get the item (issue or PR) from the payload
  const item = context.payload.issue || context.payload.pull_request;
  if (!item) {
    core.warning("No issue or pull_request found in event payload");
    return;
  }

  const itemNumber = item.number;
  const labelName = context.payload.label?.name;

  if (labelName !== DISABLE_LABEL) {
    core.info(`Skipping: label '${labelName}' is not '${DISABLE_LABEL}'`);
    return;
  }

  const itemType = eventName === "issues" ? "issue" : "pull request";
  core.info(`Processing ${itemType} #${itemNumber} labeled with '${labelName}'`);

  // Extract workflow ID from body XML comment markers
  const body = item.body || "";
  const workflowId = extractWorkflowId(body);

  if (!workflowId) {
    core.warning(`Could not find workflow ID in ${itemType} #${itemNumber} body. ` + `Expected a <!-- gh-aw-workflow-id: ... --> marker.`);
    await github.rest.issues.createComment({
      owner,
      repo,
      issue_number: itemNumber,
      body:
        `> [!WARNING]\n` +
        `> **Could not disable agentic workflow**\n>\n` +
        `> No workflow ID marker was found in this ${itemType}'s body. ` +
        `The \`${DISABLE_LABEL}\` label can only be used on issues and pull requests that were created by an agentic workflow ` +
        `(they contain a \`<!-- gh-aw-workflow-id: ... -->\` marker).\n>\n` +
        `> To disable a workflow manually, use:\n` +
        `> \`\`\`\n` +
        `> gh aw disable <workflow-id>\n` +
        `> \`\`\``,
    });
    core.setFailed(`${ERR_NOT_FOUND}: No workflow ID marker found in ${itemType} #${itemNumber}`);
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
      env: { ...process.env, GH_TOKEN: process.env.GH_TOKEN || process.env.GITHUB_TOKEN || "" },
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

  // Post a success comment on the issue/PR
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

  core.info(`Posted disable confirmation comment on ${itemType} #${itemNumber}`);
}

module.exports = { main, extractWorkflowId };
