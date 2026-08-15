// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "approve_workflow_run";

/**
 * @param {unknown} value
 * @returns {number | undefined}
 */
function parseRunId(value) {
  if (typeof value !== "number" && typeof value !== "string") return undefined;
  const normalized = typeof value === "string" ? value.trim() : value;
  if (normalized === "") return undefined;
  const runId = Number(normalized);
  if (!Number.isSafeInteger(runId) || runId <= 0) return undefined;
  return runId;
}

/**
 * Main handler factory for approve_workflow_run.
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const maxCount = config.max || 1;
  const isStaged = isStagedMode(config);
  const githubToken = config["github-token"];
  let processedCount = 0;

  core.info(`Approve workflow run configuration: max=${maxCount}`);

  if (!isStaged && !githubToken) {
    const error = "approve_workflow_run requires an external github-token or GitHub App token";
    core.error(error);
    return async () => ({ success: false, error });
  }

  const githubClient = isStaged ? null : await createAuthenticatedGitHubClient(config);

  return async function handleApproveWorkflowRun(message) {
    const runId = parseRunId(message.run_id);
    if (!runId) {
      const error = "run_id must be a positive integer";
      core.warning(error);
      return { success: false, error };
    }

    if (isStaged) {
      logStagedPreviewInfo(`Would approve workflow run ${runId}`);
      return { success: true, staged: true, run_id: runId };
    }

    if (processedCount >= maxCount) {
      core.warning(`Skipping ${HANDLER_TYPE}: max count of ${maxCount} reached`);
      return { success: false, error: `Max count of ${maxCount} reached` };
    }

    try {
      const { data: run } = await githubClient.rest.actions.getWorkflowRun({
        owner: context.repo.owner,
        repo: context.repo.repo,
        run_id: runId,
      });

      if (run.event !== "pull_request" || !Array.isArray(run.pull_requests) || run.pull_requests.length === 0) {
        const error = `Workflow run ${runId} is not associated with a pull request`;
        core.warning(error);
        return { success: false, error };
      }

      if (run.status !== "waiting") {
        const error = `Workflow run ${runId} is not awaiting approval (status: ${run.status || "none"})`;
        core.warning(error);
        return { success: false, error };
      }

      processedCount++;

      await githubClient.rest.actions.approveWorkflowRun({
        owner: context.repo.owner,
        repo: context.repo.repo,
        run_id: runId,
      });

      core.info(`Approved workflow run ${runId}: ${run.html_url}`);
      return { success: true, run_id: runId, url: run.html_url };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to approve workflow run ${runId}: ${errorMessage}`);
      return { success: false, error: errorMessage };
    }
  };
}

module.exports = { main, parseRunId };
