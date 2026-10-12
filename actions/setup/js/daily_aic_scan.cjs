// @ts-check

const fs = require("fs");
const path = require("path");
const { AIC_SCAN_CACHE_FILE_PATH, readScanCache, matchesCompletedRun, scanCacheEntry } = require("./daily_aic_cache_helpers.cjs");
const { getErrorStatus, isQuotaError, isTransientError, safeResponseHeaders } = require("./daily_aic_api_budget.cjs");

const WINDOW_MS = 24 * 60 * 60 * 1000;
const MAX_PAGES = 25;

/**
 * The run listing is authoritative for window membership and attempt identity.
 * Publish only individually resolved observations, including nonzero usage.
 * Missing entries and concurrent snapshots are safe misses, never zero usage.
 */
async function scanDailyAIC({ github, context, budget, artifactClient, getRunAIC, listPage, token, fallbackAIC = 0, cachePath = AIC_SCAN_CACHE_FILE_PATH, now = Date.now() }) {
  const { owner, repo } = context.repo;
  const repository = `${owner}/${repo}`;
  const currentResponse = await budget.request(() => github.rest.actions.getWorkflowRun({ owner, repo, run_id: context.runId }));
  const current = currentResponse.data;
  if (!Number.isSafeInteger(current?.workflow_id) || current.workflow_id <= 0) throw new Error("Cannot resolve the daily AIC workflow");
  core.info(
    `[daily-workflow-aic] Started authoritative AIC scan: ${JSON.stringify({
      repository,
      currentRunId: context.runId,
      workflowId: current.workflow_id,
      fallbackAIC: Number.isFinite(fallbackAIC) && fallbackAIC > 0 ? fallbackAIC : null,
    })}`
  );
  const entries = fs.existsSync(cachePath) ? readScanCache(fs.readFileSync(cachePath, "utf8"), repository, current.workflow_id, now) : new Map();
  core.info(`[daily-workflow-aic] Loaded scan observations: ${JSON.stringify({ workflowId: current.workflow_id, cachedEntries: entries.size })}`);
  const candidates = new Map();
  let complete = false;
  let lookupMode = "workflow_id";
  const created = `>=${new Date(now - WINDOW_MS).toISOString()}`;
  for (let page = 1; page <= MAX_PAGES; page++) {
    const result = await listPage(
      github,
      {
        owner,
        repo,
        workflowId: current.workflow_id,
        created,
        page,
        perPage: 100,
        lookupMode,
      },
      budget
    );
    lookupMode = result.lookupMode;
    const pageRuns = result.response.data?.workflow_runs;
    if (!Array.isArray(pageRuns)) throw new Error("Incomplete daily AIC workflow history");
    core.info(
      `[daily-workflow-aic] Listed workflow run candidates: ${JSON.stringify({
        page,
        lookupMode,
        sourceRunCount: result.sourceRunCount,
        returnedRunCount: pageRuns.length,
      })}`
    );
    for (const run of pageRuns) {
      if (!Number.isSafeInteger(run?.id) || run.id <= 0) throw new Error("A workflow run has an invalid identity");
      if (run.id === context.runId) continue;
      const created = Date.parse(run.created_at);
      if (!Number.isFinite(created)) throw new Error("A workflow run has an unknown creation time");
      if (created < now - WINDOW_MS) {
        complete = true;
        break;
      }
      if (run.status !== "completed" || !Number.isSafeInteger(run.run_attempt) || run.run_attempt < 1 || !Number.isFinite(Date.parse(run.updated_at))) {
        throw new Error("A workflow run has incomplete attempt metadata");
      }
      candidates.set(run.id, run);
    }
    if (result.sourceRunCount < 100) complete = true;
    if (result.oldestUnfilteredCreatedAt != null) {
      const oldest = Date.parse(result.oldestUnfilteredCreatedAt);
      if (!Number.isFinite(oldest)) throw new Error("Workflow history has an unknown creation time");
      if (oldest < now - WINDOW_MS) complete = true;
    }
    if (complete) break;
  }
  if (!complete) throw new Error("Daily AIC workflow history exceeds the complete pagination window");

  const countedRuns = [];
  let cacheHits = 0;
  try {
    for (const run of candidates.values()) {
      const cached = entries.get(run.id);
      const hit = matchesCompletedRun(cached, run) && cached.source === "recorded";
      core.info(
        `[daily-workflow-aic] Resolving run AIC: ${JSON.stringify({
          runId: run.id,
          runAttempt: run.run_attempt,
          conclusion: run.conclusion || "unknown",
          cacheHit: hit,
        })}`
      );
      let aic;
      let reason;
      let source;
      if (hit) {
        aic = cached.aic;
        reason = "scan_cache";
        source = "recorded";
      } else {
        try {
          aic = await getRunAIC(artifactClient, run.id, token, owner, repo, run, { github, budget });
          reason = "artifact_accounting";
          source = "recorded";
        } catch (error) {
          const status = getErrorStatus(error);
          const canUseFallback = Number.isFinite(fallbackAIC) && fallbackAIC > 0 && !isQuotaError(error) && safeResponseHeaders(error?.response?.headers)["retry-after"] == null && (!status || isTransientError(error));
          core.info(
            `[daily-workflow-aic] Run AIC resolution failed: ${JSON.stringify({
              runId: run.id,
              runAttempt: run.run_attempt,
              status: status || null,
              error: error instanceof Error ? error.message : String(error),
              fallbackAvailable: canUseFallback,
            })}`
          );
          if (!canUseFallback) throw error;
          aic = fallbackAIC;
          reason = "max_ai_credits_fallback";
          source = "estimated";
          core.warning(
            `[daily-workflow-aic] Assuming max AI Credits after all accounting sources failed: ${JSON.stringify({
              runId: run.id,
              runAttempt: run.run_attempt,
              assumedAIC: aic,
            })}`
          );
        }
      }
      entries.set(run.id, scanCacheEntry(run, aic, repository, current.workflow_id, now, source));
      if (hit) {
        cacheHits++;
      }
      core.info(`[daily-workflow-aic] Computed run AIC: ${JSON.stringify({ runId: run.id, aic, reason })}`);
      countedRuns.push({ ...run, aic, source });
    }
  } finally {
    // This runs even after an API error. Partial snapshots accelerate recovery but
    // cannot authorize a later run without a new complete authoritative listing.
    fs.mkdirSync(path.dirname(cachePath), { recursive: true });
    fs.writeFileSync(cachePath, [...entries.values()].map(entry => JSON.stringify(entry)).join("\n") + "\n", "utf8");
    core.info(`[daily-workflow-aic] Persisted scan observations: ${JSON.stringify({ cachePath, entries: entries.size })}`);
  }
  core.info(`[daily-workflow-aic] Completed authoritative AIC scan: ${JSON.stringify({ candidateRuns: candidates.size, countedRuns: countedRuns.length, cacheHits })}`);
  return { countedRuns, candidateRunsCount: candidates.size, cacheHits, current };
}

module.exports = { scanDailyAIC };
