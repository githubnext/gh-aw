// @ts-check
/// <reference types="@actions/github-script" />

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { matchesSimpleGlob } = require("./glob_pattern_helpers.cjs");
const { SAFE_OUTPUT_E007 } = require("./error_codes.cjs");
const path = require("node:path");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { logStagedPreviewInfo } = require("./staged_preview.cjs");
const { checkFileProtectionPostApply } = require("./manifest_file_helpers.cjs");

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "approve_workflow_run";

/**
 * @param {unknown} value
 * @returns {number | undefined}
 */
function parsePositiveInt(value) {
  if (typeof value !== "number" && typeof value !== "string") return undefined;
  const normalized = typeof value === "string" ? value.trim() : value;
  if (normalized === "") return undefined;
  const runId = Number(normalized);
  if (!Number.isSafeInteger(runId) || runId <= 0) return undefined;
  return runId;
}

/**
 * @param {unknown} value
 * @returns {Set<number>}
 */
function parseAllowedPullRequests(value) {
  let normalized = value;
  if (typeof normalized === "string") {
    const trimmed = normalized.trim();
    if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
      try {
        normalized = JSON.parse(trimmed);
      } catch {
        normalized = value;
      }
    }
  }
  const parsed = (Array.isArray(normalized) ? normalized : [normalized]).map(parsePositiveInt).filter(candidate => candidate !== undefined);
  return new Set(parsed);
}

/**
 * @param {unknown} value
 * @returns {string | undefined}
 */
function normalizeWorkflowFilename(value) {
  if (typeof value !== "string" || value === "") return undefined;
  return path.posix.basename(value).replace(/\.yaml$/i, ".yml");
}

/**
 * @param {unknown} workflowPath
 * @param {unknown} allowedWorkflows
 * @returns {boolean}
 */
function isAllowedWorkflow(workflowPath, allowedWorkflows) {
  const filename = normalizeWorkflowFilename(workflowPath);
  if (!filename || !Array.isArray(allowedWorkflows) || allowedWorkflows.length === 0) return false;

  return allowedWorkflows.some(pattern => {
    if (typeof pattern !== "string" || path.posix.basename(pattern) !== pattern) return false;
    const normalizedPattern = normalizeWorkflowFilename(pattern);
    return normalizedPattern !== undefined && matchesSimpleGlob(filename, normalizedPattern);
  });
}

/**
 * @returns {number | undefined}
 */
function getCurrentPullRequestNumber() {
  const payload = context.payload || {};
  return parsePositiveInt(payload.pull_request?.number || (payload.issue?.pull_request ? payload.issue.number : undefined));
}

/**
 * @param {any} githubClient
 * @param {number} pullRequestNumber
 * @returns {Promise<string[]>}
 */
async function getModifiedPullRequestFiles(githubClient, pullRequestNumber) {
  const files = await githubClient.paginate(githubClient.rest.pulls.listFiles, {
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: pullRequestNumber,
    per_page: 100,
  });
  if (!Array.isArray(files) || files.some(file => typeof file?.filename !== "string")) {
    throw new Error(`${SAFE_OUTPUT_E007}: Unable to verify modified files for pull request #${pullRequestNumber}`);
  }
  return files.map(file => file.filename);
}

/**
 * @param {any} githubClient
 * @param {number} pullRequestNumber
 * @returns {Promise<boolean>}
 */
async function isForkPullRequest(githubClient, pullRequestNumber) {
  const { data: pullRequest } = await githubClient.rest.pulls.get({
    owner: context.repo.owner,
    repo: context.repo.repo,
    pull_number: pullRequestNumber,
  });
  if (pullRequest?.head?.repo === null) {
    throw new Error(`${SAFE_OUTPUT_E007}: Cannot approve pull request #${pullRequestNumber}: its fork repository is unavailable`);
  }
  if (typeof pullRequest?.head?.repo?.fork !== "boolean") {
    throw new Error(`${SAFE_OUTPUT_E007}: Unable to verify fork status for pull request #${pullRequestNumber}`);
  }
  return pullRequest.head.repo.fork;
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
    const runId = parsePositiveInt(message.run_id);
    if (!runId) {
      const error = "run_id must be a positive integer";
      core.warning(error);
      return { success: false, error };
    }

    if (context.eventName === "pull_request_target") {
      const error = "approve_workflow_run cannot run on pull_request_target events; use pull_request instead";
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

      const workflowId = parsePositiveInt(run.workflow_id);
      if (!workflowId) {
        const error = `Workflow run ${runId} has invalid workflow data`;
        core.warning(error);
        return { success: false, error };
      }
      const { data: workflow } = await githubClient.rest.actions.getWorkflow({
        owner: context.repo.owner,
        repo: context.repo.repo,
        workflow_id: workflowId,
      });
      if (!isAllowedWorkflow(workflow?.path, config.allowed_workflows)) {
        const error = `Workflow run ${runId} does not match an allowed workflow`;
        core.warning(error);
        return { success: false, error };
      }

      const allowedPullRequests = parseAllowedPullRequests(config.allowed_pull_requests);
      const currentPullRequest = getCurrentPullRequestNumber();
      const isAuthorized = run.pull_requests.every(pullRequest => {
        const pullRequestNumber = parsePositiveInt(pullRequest.number);
        return pullRequestNumber !== undefined && ((currentPullRequest !== undefined && pullRequestNumber === currentPullRequest) || allowedPullRequests.has(pullRequestNumber));
      });
      if (!isAuthorized) {
        const error = `Workflow run ${runId} is not associated exclusively with the triggering pull request or explicitly allowed pull requests`;
        core.warning(error);
        return { success: false, error };
      }

      for (const pullRequest of run.pull_requests) {
        const pullRequestNumber = parsePositiveInt(pullRequest.number);
        if (pullRequestNumber === undefined) {
          const error = `Workflow run ${runId} has invalid pull request data`;
          core.warning(error);
          return { success: false, error };
        }
        if ((await isForkPullRequest(githubClient, pullRequestNumber)) && config.fork !== true) {
          const error = `Workflow run ${runId} cannot be approved because pull request #${pullRequestNumber} is from a fork; set fork: true to allow fork pull requests`;
          core.warning(error);
          return { success: false, error };
        }
        const files = await getModifiedPullRequestFiles(githubClient, pullRequestNumber);
        const protection = checkFileProtectionPostApply(files, {
          ...config,
          protected_files_policy: "blocked",
        });
        if (protection.action !== "allow") {
          const error = `Workflow run ${runId} cannot be approved because pull request #${pullRequestNumber} modifies protected files (${protection.files.join(", ")})`;
          core.warning(error);
          return { success: false, error };
        }
      }

      processedCount++;
      try {
        await githubClient.rest.actions.approveWorkflowRun({
          owner: context.repo.owner,
          repo: context.repo.repo,
          run_id: runId,
        });
      } catch (error) {
        processedCount--;
        throw error;
      }

      core.info(`Approved workflow run ${runId}: ${run.html_url}`);
      return { success: true, run_id: runId, url: run.html_url };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.error(`Failed to approve workflow run ${runId}: ${errorMessage}`);
      return { success: false, error: errorMessage };
    }
  };
}

module.exports = {
  main,
  parseAllowedPullRequests,
  parsePositiveInt,
  normalizeWorkflowFilename,
  isAllowedWorkflow,
  getModifiedPullRequestFiles,
  isForkPullRequest,
};
