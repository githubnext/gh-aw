// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const os = require("os");
const path = require("path");
const { MAX_RESTORE_RUNS, MAX_RESTORE_PAGES, SCAN_WINDOW_MS } = require("./constants.cjs");
const { DefaultArtifactClient } = require("./artifact_client.cjs");
const { AIC_SCAN_CACHE_FILE_PATH, AIC_SCAN_CACHE_ARTIFACT_NAME, readScanCache } = require("./daily_aic_cache_helpers.cjs");
const { API_READ_OPERATIONS, createAPIBudget, retryNotBefore, isQuotaError, safeResponseHeaders, getAPIReadFailure } = require("./daily_aic_api_budget.cjs");
const { getGitHubAPIDiagnostic } = require("./github_api_diagnostics.cjs");

function isTrustedProducer(run, current, repository, defaultBranch) {
  if (run.workflow_id !== current.workflow_id || run.path !== current.path || run.repository?.full_name !== repository) return false;
  // pull_request executes the contributor's workflow. Never trust its summary,
  // even if it has the same workflow ID or the artifact has the expected name.
  return (
    run.event === "pull_request_target" || (["push", "schedule", "workflow_dispatch", "issues", "issue_comment"].includes(run.event) && !!defaultBranch && run.head_branch === defaultBranch && run.head_repository?.full_name === repository)
  );
}

async function listRequiredWorkflowRuns(owner, repo, current, repository, defaultBranch, budget) {
  const runs = [];
  const created = `>=${new Date(Date.now() - SCAN_WINDOW_MS).toISOString()}`;
  for (let page = 1; page <= MAX_RESTORE_PAGES && runs.length < MAX_RESTORE_RUNS; page++) {
    const response = await budget.request(() => github.rest.actions.listWorkflowRunsForRepo({ owner, repo, status: "completed", created, per_page: 100, page }), true, API_READ_OPERATIONS.REPOSITORY_HISTORY);
    const pageRuns = response.data.workflow_runs || [];
    const trustedRuns = pageRuns.filter(run => run.id !== context.runId && isTrustedProducer(run, current, repository, defaultBranch));
    core.info(`[daily-aic-cache] Evaluated repository history page: ${JSON.stringify({ page, listedRuns: pageRuns.length, trustedCandidates: trustedRuns.length })}`);
    runs.push(...trustedRuns.slice(0, MAX_RESTORE_RUNS - runs.length));
    if (pageRuns.length < 100) break;
  }
  return runs;
}

async function listRecentWorkflowRuns(owner, repo, current, repository, defaultBranch, budget) {
  let response;
  core.info(`[daily-aic-cache] Querying workflow-specific run history: ${JSON.stringify({ workflowId: current.workflow_id })}`);
  try {
    response = await budget.request(() => github.rest.actions.listWorkflowRuns({ owner, repo, workflow_id: current.workflow_id, per_page: 10 }), true, API_READ_OPERATIONS.WORKFLOW_HISTORY);
  } catch (error) {
    if (error?.status !== 404 || isQuotaError(error) || safeResponseHeaders(error?.response?.headers)["retry-after"] != null) throw error;
    core.info("[daily-aic-cache] Workflow-specific history unavailable; searching repository history for required workflow runs.");
    return listRequiredWorkflowRuns(owner, repo, current, repository, defaultBranch, budget);
  }
  core.info(`[daily-aic-cache] Workflow-specific run history returned candidates: ${JSON.stringify({ workflowId: current.workflow_id, runs: response.data.workflow_runs?.length ?? 0 })}`);
  return response.data.workflow_runs;
}

async function restoreWithBudget(cachePath, options, budget) {
  const client = options.createArtifactClient?.() || new DefaultArtifactClient({ request: budget.request });
  const { owner, repo } = context.repo;
  const repository = `${owner}/${repo}`;
  const currentResponse = await budget.request(() => github.rest.actions.getWorkflowRun({ owner, repo, run_id: context.runId }), true, API_READ_OPERATIONS.CURRENT_RUN);
  const current = currentResponse.data;
  if (!Number.isSafeInteger(current?.workflow_id) || current.workflow_id <= 0) throw new Error("Cannot resolve workflow for daily AIC snapshot restore");
  const defaultBranch = context.payload.repository?.default_branch;
  const runs = await listRecentWorkflowRuns(owner, repo, current, repository, defaultBranch, budget);
  const auth = await github.auth({ type: "token" });
  if (!auth || typeof auth !== "object" || !("token" in auth) || typeof auth.token !== "string" || !auth.token) {
    throw new Error("No token available to restore daily AIC observations");
  }
  const token = auth.token;
  for (const run of runs) {
    const isCurrentRun = run.id === context.runId;
    const trusted = isTrustedProducer(run, current, repository, defaultBranch);
    core.info(
      `[daily-aic-cache] Evaluated snapshot producer: ${JSON.stringify({
        producerRunId: run.id,
        isCurrentRun,
        trusted,
        workflowMatches: run.workflow_id === current.workflow_id,
        pathMatches: run.path === current.path,
        repositoryMatches: run.repository?.full_name === repository,
        defaultBranchMatches: !!defaultBranch && run.head_branch === defaultBranch,
        headRepositoryMatches: run.head_repository?.full_name === repository,
      })}`
    );
    if (isCurrentRun || !trusted) continue;
    const response = await budget.request(() => github.rest.actions.listWorkflowRunArtifacts({ owner, repo, run_id: run.id, per_page: 100 }), true, API_READ_OPERATIONS.ARTIFACT_LIST);
    const artifact = response.data.artifacts.find(item => item.name === AIC_SCAN_CACHE_ARTIFACT_NAME && !item.expired);
    if (!artifact) {
      core.info(`[daily-aic-cache] Skipped snapshot producer: ${JSON.stringify({ producerRunId: run.id, reason: "no_unexpired_scan_artifact" })}`);
      continue;
    }
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "aic-scan-restore-"));
    try {
      const download = await client.downloadArtifact(artifact.id, {
        path: directory,
        findBy: { token, workflowRunId: run.id, repositoryOwner: owner, repositoryName: repo },
      });
      const file = path.join(download.downloadPath || directory, path.basename(AIC_SCAN_CACHE_FILE_PATH));
      if (!fs.existsSync(file)) {
        core.info(`[daily-aic-cache] Skipped scan artifact: ${JSON.stringify({ producerRunId: run.id, artifactId: artifact.id, reason: "missing_snapshot_file" })}`);
        continue;
      }
      const entries = readScanCache(fs.readFileSync(file, "utf8"), repository, current.workflow_id);
      if (entries.size === 0) {
        core.info(`[daily-aic-cache] Skipped scan artifact: ${JSON.stringify({ producerRunId: run.id, artifactId: artifact.id, reason: "no_valid_observations" })}`);
        continue;
      }
      fs.mkdirSync(path.dirname(cachePath), { recursive: true });
      fs.writeFileSync(cachePath, [...entries.values()].map(entry => JSON.stringify(entry)).join("\n") + "\n", "utf8");
      core.info(`[daily-aic-cache] Restored verified scan observations: ${JSON.stringify({ producerRunId: run.id, artifactId: artifact.id, entries: entries.size })}`);
      return;
    } finally {
      fs.rmSync(directory, { recursive: true, force: true });
    }
  }
  core.info("[daily-aic-cache] No trusted scan snapshot found; resolve the complete window from usage artifacts.");
}

async function mainWithPaths(cachePath = AIC_SCAN_CACHE_FILE_PATH, options = {}) {
  const budget = createAPIBudget();
  try {
    await restoreWithBudget(cachePath, options, budget);
  } catch (error) {
    const diagnostic = getGitHubAPIDiagnostic(error);
    const failure = getAPIReadFailure(error);
    if (diagnostic && failure) {
      const message = `Daily AIC snapshot restore stopped at ${failure.operation} after ${failure.attempts} attempt(s): ${diagnostic.message}. Accounting remains unverified; agent startup is blocked.`;
      core.info(`[daily-aic-cache] ${message}`);
      if (core.summary) {
        try {
          await core.summary.addRaw(`### Daily AIC snapshot restore blocked\n\n${message}\n`).write();
        } catch {
          core.warning("Failed to write daily AIC snapshot restore failure summary.");
        }
      }
    }
    const retryAt = retryNotBefore(error?.response?.headers, Date.now(), isQuotaError(error));
    if (retryAt) core.info(`[daily-aic-cache] No further API requests before ${retryAt}`);
    throw error;
  } finally {
    core.info(`[daily-aic-cache] GitHub API rate limit level: ${JSON.stringify(budget.snapshot())}`);
  }
}

async function main() {
  const { shouldSkipDailyAICGuardrail } = require("./check_daily_aic_workflow_guardrail.cjs");
  if (shouldSkipDailyAICGuardrail()) return;
  await mainWithPaths();
}

module.exports = { main, mainWithPaths, isTrustedProducer };
