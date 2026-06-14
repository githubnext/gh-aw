// @ts-check
/// <reference types="@actions/github-script" />

/**
 * restore_aic_usage_cache_fallback.cjs
 *
 * Called from the activation job when actions/cache/restore has a cache miss.
 * Downloads the most recent `aic-usage-cache` artifact from the same workflow's
 * recent runs to populate the local cache file without requiring the artifact to
 * have been saved on the current branch.
 *
 * Background: GitHub Actions `actions/cache` is branch-scoped.  Workflows that
 * trigger on `pull_request` events run on a unique per-PR branch, so caches saved
 * by the conclusion job of one PR run are not visible to the activation job of a
 * different PR run.  This script compensates by falling back to a named artifact
 * (`aic-usage-cache`) that the conclusion job uploads after writing the cache file.
 * Artifacts are accessible cross-branch via the GitHub REST API.
 *
 * Requires setupGlobals() to have been called first (sets global.core, global.github,
 * global.context).
 */

"use strict";

const fs = require("fs");
const path = require("path");
const os = require("os");
const { DefaultArtifactClient } = require("./artifact_client.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Retrieve the bearer token from the builtin `github` Octokit instance.
 * Returns an empty string if the auth plugin does not support the "token" type.
 *
 * @returns {Promise<string>}
 */
async function getTokenFromGithub() {
  try {
    const auth = await github.auth({ type: "token" });
    return (auth && typeof auth.token === "string" && auth.token) || "";
  } catch {
    return "";
  }
}

/** Path where the activation job expects the usage cache to be restored. */
const CACHE_FILE_PATH = "/tmp/gh-aw/agentic-workflow-usage-cache.jsonl";

/** Name of the artifact uploaded by the conclusion job that holds the aggregated cache. */
const AIC_USAGE_CACHE_ARTIFACT_NAME = "aic-usage-cache";

/** Maximum number of recent workflow runs to search for a usable cache artifact. */
const MAX_RUNS_TO_SEARCH = 5;

/**
 * @param {string} message
 * @param {Record<string, unknown>} [details]
 */
function logFallback(message, details) {
  const suffix =
    details && Object.keys(details).length > 0
      ? ": " +
        (() => {
          try {
            return JSON.stringify(details);
          } catch {
            return "{}";
          }
        })()
      : "";
  core.info(`[daily-aic-cache-fallback] ${message}${suffix}`);
}

/**
 * Downloads the most recent `aic-usage-cache` artifact from the same workflow's
 * recent runs and writes it to {@link CACHE_FILE_PATH}.
 *
 * @param {string} [cacheFilePath] Override for the target cache file path (used in tests).
 * @param {{ createArtifactClient?: () => import("./artifact_client.cjs").DefaultArtifactClient }} [options]
 *   Optional overrides for testing (e.g. inject a mock artifact client factory).
 * @returns {Promise<void>}
 */
async function mainWithPaths(cacheFilePath, options = {}) {
  const cachePath = cacheFilePath || CACHE_FILE_PATH;
  const createArtifactClient = options.createArtifactClient || (() => new DefaultArtifactClient());

  // If the file already exists (e.g., restored via actions/cache restore-keys prefix
  // match or a prior step), skip the artifact-based fallback entirely.
  if (fs.existsSync(cachePath)) {
    logFallback("Cache file already exists; skipping artifact fallback", { path: cachePath });
    return;
  }

  const { owner, repo } = context.repo;

  try {
    // Resolve the numeric GitHub workflow ID so we can list runs for this specific workflow.
    const currentRunData = await github.rest.actions.getWorkflowRun({
      owner,
      repo,
      run_id: context.runId,
    });
    const workflowNumericId = currentRunData.data.workflow_id;
    if (!workflowNumericId) {
      logFallback("Could not determine numeric workflow ID; skipping artifact fallback");
      return;
    }

    logFallback("Searching for aic-usage-cache artifact from recent runs", {
      workflowId: workflowNumericId,
      currentRunId: context.runId,
      maxRunsToSearch: MAX_RUNS_TO_SEARCH,
    });

    const { data: runsData } = await github.rest.actions.listWorkflowRuns({
      owner,
      repo,
      workflow_id: workflowNumericId,
      status: "completed",
      per_page: MAX_RUNS_TO_SEARCH,
    });

    for (const run of runsData.workflow_runs) {
      // Skip the current run — its conclusion job hasn't written the artifact yet.
      if (run.id === context.runId) {
        continue;
      }
      try {
        // Use the builtin github instance (already authenticated) to list artifacts.
        const { data: artifactsData } = await github.rest.actions.listWorkflowRunArtifacts({
          owner,
          repo,
          run_id: run.id,
        });

        const cacheArtifact = artifactsData.artifacts.find(a => a.name === AIC_USAGE_CACHE_ARTIFACT_NAME && !a.expired);
        if (!cacheArtifact) {
          logFallback("No aic-usage-cache artifact in run", { runId: run.id });
          continue;
        }

        logFallback("Found aic-usage-cache artifact; downloading", {
          runId: run.id,
          artifactId: cacheArtifact.id,
        });

        // Get the token from the builtin github instance for the download step.
        const token = await getTokenFromGithub();

        const downloadRoot = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-aic-cache-fallback-"));
        const artifactClient = createArtifactClient();
        const download = await artifactClient.downloadArtifact(cacheArtifact.id, {
          path: downloadRoot,
          findBy: {
            token,
            workflowRunId: run.id,
            repositoryOwner: owner,
            repositoryName: repo,
          },
        });

        const downloadPath = download.downloadPath || downloadRoot;

        // Locate the JSONL file inside the extracted artifact directory.
        const files = fs.readdirSync(downloadPath);
        const jsonlFile = files.find(f => f.endsWith(".jsonl"));
        if (!jsonlFile) {
          logFallback("No JSONL file in downloaded artifact; trying next run", {
            downloadPath,
            files,
          });
          continue;
        }

        const srcPath = path.join(downloadPath, jsonlFile);
        const dir = path.dirname(cachePath);
        fs.mkdirSync(dir, { recursive: true });
        fs.copyFileSync(srcPath, cachePath);

        logFallback("Restored cache from artifact", {
          runId: run.id,
          artifactId: cacheArtifact.id,
          path: cachePath,
        });
        return;
      } catch (runErr) {
        logFallback("Error processing run; trying next", {
          runId: run.id,
          error: getErrorMessage(runErr),
        });
      }
    }

    logFallback("No aic-usage-cache artifact found in recent runs; proceeding without cache", { runsSearched: runsData.workflow_runs.length });
  } catch (error) {
    // Non-fatal: a failure here should never block the activation job.
    logFallback("Failed to restore cache from artifact fallback; proceeding without cache", {
      error: getErrorMessage(error),
    });
  }
}

/**
 * Entry point called from the GitHub Actions step.
 *
 * @returns {Promise<void>}
 */
async function main() {
  return mainWithPaths();
}

module.exports = { main, mainWithPaths };
