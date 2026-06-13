// @ts-check
/// <reference types="@actions/github-script" />

/**
 * write_daily_aic_usage_cache.cjs
 *
 * Called from the conclusion job to record this run's AI Credits consumption in the
 * per-workflow usage cache. The cache is later restored in the activation job so the
 * daily-AIC guardrail can look up prior run costs without re-downloading artifacts.
 *
 * Requires setupGlobals() to have been called first (sets global.core).
 */

const fs = require("fs");
const path = require("path");

const { findJSONLFiles, sumAICFromUsageJSONLFiles } = require("./daily_aic_workflow_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/** Path where the restored (and updated) usage cache lives on the runner. */
const CACHE_FILE_PATH = "/tmp/gh-aw/agentic-workflow-usage-cache.jsonl";

/**
 * Directory prepared by the "Collect usage artifact files" step in the conclusion job.
 * Contains agent_usage.jsonl and agent/token_usage.jsonl which mirror the contents of
 * the "usage" artifact that getRunAIC() downloads during the daily-AIC guardrail check.
 */
const USAGE_DIR = "/tmp/gh-aw/usage";

/**
 * @param {string} message
 * @param {Record<string, unknown>} [details]
 */
function logCache(message, details) {
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
  core.info(`[daily-aic-cache] ${message}${suffix}`);
}

/**
 * Appends a `{run_id, aic}` JSONL entry to the cache file, preserving any existing entries
 * that were restored from the previous cache snapshot.
 *
 * @returns {Promise<void>}
 */
async function main() {
  try {
    const runId = Number(process.env.GITHUB_RUN_ID || 0);
    if (!runId) {
      core.warning("[daily-aic-cache] GITHUB_RUN_ID not set; skipping cache write.");
      return;
    }

    // Compute AIC from the usage JSONL files prepared by buildUsageArtifactUploadSteps.
    const usageFiles = findJSONLFiles(USAGE_DIR);
    logCache("Scanning usage JSONL files", { dir: USAGE_DIR, count: usageFiles.length, files: usageFiles });
    const aic = sumAICFromUsageJSONLFiles(usageFiles);
    logCache("Computed AIC for current run", { runId, aic });

    // Skip writing a non-finite or negative AIC: those values indicate an unexpected computation
    // error.  A zero AIC is written intentionally — it records runs where the agent was blocked
    // (e.g. daily-AIC guardrail exceeded) so that subsequent activations do not waste an
    // artifact-download round-trip trying to re-fetch usage data that does not exist.
    if (!Number.isFinite(aic) || aic < 0) {
      core.warning(`[daily-aic-cache] Computed AIC is ${aic} (negative or non-finite); skipping cache write.`);
      return;
    }

    // Read existing cache content (restored from the previous run's cache snapshot, if any).
    let existingLines = "";
    try {
      if (fs.existsSync(CACHE_FILE_PATH)) {
        existingLines = fs.readFileSync(CACHE_FILE_PATH, "utf8").trimEnd();
        const lineCount = existingLines ? existingLines.split("\n").length : 0;
        logCache("Loaded existing cache entries", { path: CACHE_FILE_PATH, lineCount });
      } else {
        logCache("No existing cache file found; starting fresh", { path: CACHE_FILE_PATH });
      }
    } catch (readErr) {
      core.warning(`[daily-aic-cache] Could not read existing cache file: ${getErrorMessage(readErr)}`);
    }

    // Build the updated JSONL content.
    const newEntry = JSON.stringify({ run_id: runId, aic });
    const updatedContent = existingLines ? `${existingLines}\n${newEntry}\n` : `${newEntry}\n`;

    // Ensure the directory exists and write the updated file.
    const dir = path.dirname(CACHE_FILE_PATH);
    fs.mkdirSync(dir, { recursive: true });
    fs.writeFileSync(CACHE_FILE_PATH, updatedContent, "utf8");
    logCache("Wrote cache entry", { runId, aic, path: CACHE_FILE_PATH });
  } catch (error) {
    // Non-fatal: a cache write failure should never block the conclusion job.
    core.warning(`[daily-aic-cache] Failed to write usage cache: ${getErrorMessage(error)}`);
  }
}

module.exports = { main };
