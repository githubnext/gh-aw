// @ts-check

/**
 * daily_aic_cache_helpers.cjs
 *
 * Shared helpers for reading and pruning the per-workflow AIC usage cache JSONL file.
 * Both the daily-AIC guardrail check (check_daily_aic_workflow_guardrail.cjs) and the
 * cache write step (write_daily_aic_usage_cache.cjs) use the same retention policy;
 * this module is the single source of truth for that logic.
 */

/** Path where the per-workflow usage cache lives on the runner. */
const AIC_USAGE_CACHE_FILE_PATH = "/tmp/gh-aw/agentic-workflow-usage-cache.jsonl";

/** Cache entries older than this threshold (in ms) are pruned when reading or writing. */
const CACHE_RETENTION_MS = 48 * 60 * 60 * 1000;

/**
 * Splits raw JSONL file content into lines, pruning entries whose `timestamp` field
 * is older than `cutoffMs`.
 *
 * - Empty lines are discarded.
 * - Lines that do not look like JSON objects (do not start with "{") are discarded.
 * - Object-like lines that cannot be parsed as JSON are preserved (defensive: avoids data loss).
 * - Lines that have no `timestamp` field are preserved (backward compatibility with
 *   entries written by older versions of the write script).
 *
 * @param {string} content  Raw JSONL file content.
 * @param {number} cutoffMs Entries with a `timestamp` that parses to a value strictly
 *                          less than `cutoffMs` are pruned.
 * @returns {{ keptLines: string[], prunedCount: number, totalCount: number }}
 */
function pruneStaleJSONLCacheLines(content, cutoffMs) {
  /** @type {string[]} */
  const keptLines = [];
  let prunedCount = 0;
  let totalCount = 0;
  for (const rawLine of content.split("\n")) {
    const line = rawLine.trim();
    if (!line) continue;
    totalCount++;
    if (!line.startsWith("{")) {
      prunedCount++;
      continue;
    }
    try {
      const entry = JSON.parse(line);
      if (typeof entry?.timestamp === "string") {
        const ts = Date.parse(entry.timestamp);
        if (Number.isFinite(ts) && ts < cutoffMs) {
          prunedCount++;
          continue;
        }
      }
      keptLines.push(line);
    } catch {
      // Preserve object-like lines that cannot be parsed (defensive: avoids data loss).
      keptLines.push(line);
    }
  }
  return { keptLines, prunedCount, totalCount };
}

module.exports = {
  AIC_USAGE_CACHE_FILE_PATH,
  CACHE_RETENTION_MS,
  pruneStaleJSONLCacheLines,
};
