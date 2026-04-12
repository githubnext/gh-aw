// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { delay } = require("./expired_entity_cleanup_helpers.cjs");
const { checkRateLimit, MIN_RATE_LIMIT_REMAINING } = require("./rate_limit_helpers.cjs");
const { fetchAndLogRateLimit } = require("./github_rate_limit_logger.cjs");

/**
 * Default delay in ms between delete operations to avoid throttling.
 */
const DELETE_DELAY_MS = 250;

/**
 * Default delay in ms between list pages to avoid throttling.
 */
const LIST_DELAY_MS = 100;

/**
 * Maximum number of pages to fetch when listing caches.
 * At 100 caches per page this allows up to 5000 caches.
 */
const MAX_LIST_PAGES = 50;

/**
 * Parse a cache key to extract the run ID and group key in a single pass.
 * Cache keys follow the pattern: memory-{parts}-{runID}
 * where runID is the last purely numeric segment.
 *
 * @param {string} key - Cache key string
 * @returns {{ runId: number | null, groupKey: string }}
 */
function parseCacheKey(key) {
  const parts = key.split("-");
  for (let i = parts.length - 1; i >= 0; i--) {
    if (/^\d+$/.test(parts[i])) {
      return {
        runId: parseInt(parts[i], 10),
        groupKey: parts.slice(0, i).join("-"),
      };
    }
  }
  return { runId: null, groupKey: key };
}

/**
 * @typedef {Object} CacheEntry
 * @property {number} id - Cache ID for deletion
 * @property {string} key - Full cache key
 * @property {number | null} runId - Extracted run ID
 * @property {string} groupKey - Group key (key without run ID)
 */

/**
 * List all caches starting with "memory-" prefix, handling pagination.
 * Results are sorted newest-first by last_accessed_at from the API.
 *
 * @param {any} github - GitHub REST client
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} [listDelayMs] - Delay between list pages in ms
 * @returns {Promise<CacheEntry[]>} List of cache entries
 */
async function listMemoryCaches(github, owner, repo, listDelayMs = LIST_DELAY_MS) {
  /** @type {CacheEntry[]} */
  const caches = [];
  let page = 1;
  const perPage = 100;

  while (page <= MAX_LIST_PAGES) {
    core.info(`   Fetching cache list page ${page}...`);
    const response = await github.rest.actions.getActionsCacheList({
      owner,
      repo,
      key: "memory-",
      per_page: perPage,
      page,
      sort: "last_accessed_at",
      direction: "desc",
    });

    const actionsCaches = response.data.actions_caches;
    if (!actionsCaches || actionsCaches.length === 0) {
      break;
    }

    for (const cache of actionsCaches) {
      if (!cache.key || !cache.key.startsWith("memory-")) {
        continue;
      }
      const { runId, groupKey } = parseCacheKey(cache.key);
      caches.push({ id: cache.id, key: cache.key, runId, groupKey });
    }

    core.info(`   Page ${page}: ${actionsCaches.length} cache(s) fetched (${caches.length} total)`);

    if (actionsCaches.length < perPage) {
      break;
    }

    page++;
    // Throttle between list pages
    await delay(listDelayMs);
  }

  if (page > MAX_LIST_PAGES) {
    core.warning(`⚠️ Reached maximum page limit (${MAX_LIST_PAGES}). Some caches may not have been listed.`);
  }

  return caches;
}

/**
 * Group caches by their group key (everything except run ID),
 * then for each group keep only the entry with the highest run ID
 * and return the rest for deletion.
 *
 * @param {CacheEntry[]} caches - List of cache entries
 * @returns {{ toDelete: CacheEntry[], kept: CacheEntry[] }}
 */
function identifyCachesToDelete(caches) {
  /** @type {Map<string, CacheEntry[]>} */
  const groups = new Map();

  for (const cache of caches) {
    if (cache.runId === null) {
      // Skip caches without a recognizable run ID
      continue;
    }
    const group = groups.get(cache.groupKey) || [];
    group.push(cache);
    groups.set(cache.groupKey, group);
  }

  /** @type {CacheEntry[]} */
  const toDelete = [];
  /** @type {CacheEntry[]} */
  const kept = [];

  for (const [, group] of groups) {
    if (group.length <= 1) {
      // Only one entry in this group, nothing to clean up
      if (group.length === 1) {
        kept.push(group[0]);
      }
      continue;
    }

    // Sort by run ID descending (highest first = latest)
    group.sort((a, b) => (b.runId ?? 0) - (a.runId ?? 0));

    // Keep the first (latest), mark the rest for deletion
    kept.push(group[0]);
    toDelete.push(...group.slice(1));
  }

  return { toDelete, kept };
}

/**
 * Main entry point: cleanup outdated cache-memory caches.
 *
 * Lists all caches with "memory-" prefix, groups them by key prefix,
 * keeps the latest run ID per group, and deletes the rest.
 * Includes timeouts to avoid GitHub API throttling and skips
 * if rate limiting is too high.
 *
 * @param {Object} [options] - Optional configuration for testing
 * @param {number} [options.deleteDelayMs] - Delay between deletions (default: DELETE_DELAY_MS)
 * @param {number} [options.listDelayMs] - Delay between list pages (default: LIST_DELAY_MS)
 */
async function main(options = {}) {
  const deleteDelayMs = options.deleteDelayMs ?? DELETE_DELAY_MS;
  const listDelayMs = options.listDelayMs ?? LIST_DELAY_MS;

  const owner = context.repo.owner;
  const repo = context.repo.repo;

  core.info("🧹 Starting cache-memory cleanup");
  core.info(`   Repository: ${owner}/${repo}`);

  // Log initial rate limit snapshot for observability
  await fetchAndLogRateLimit(github, "cleanup_cache_memory_start");

  // Check rate limit before starting
  const { ok: rateLimitOk, remaining: initialRemaining } = await checkRateLimit(github, "cleanup_cache_memory_initial");
  if (!rateLimitOk) {
    core.warning(`⚠️ Rate limit too low (${initialRemaining} remaining, minimum: ${MIN_RATE_LIMIT_REMAINING}). Skipping cache cleanup.`);
    core.summary.addRaw(`## Cache Memory Cleanup\n\n⚠️ Skipped: Rate limit too low (${initialRemaining} remaining, minimum required: ${MIN_RATE_LIMIT_REMAINING})\n`);
    await core.summary.write();
    return;
  }

  core.info(`   Rate limit remaining: ${initialRemaining === -1 ? "unknown" : initialRemaining}`);

  // List all memory caches
  core.info("📋 Listing caches with 'memory-' prefix...");
  let caches;
  try {
    caches = await listMemoryCaches(github, owner, repo, listDelayMs);
  } catch (error) {
    core.error(`❌ Failed to list caches: ${getErrorMessage(error)}`);
    core.summary.addRaw(`## Cache Memory Cleanup\n\n❌ Failed to list caches: ${getErrorMessage(error)}\n`);
    await core.summary.write();
    return;
  }

  core.info(`   Found ${caches.length} cache(s) with 'memory-' prefix`);

  if (caches.length === 0) {
    core.info("✅ No memory caches found. Nothing to clean up.");
    core.summary.addRaw("## Cache Memory Cleanup\n\n✅ No memory caches found. Nothing to clean up.\n");
    await core.summary.write();
    return;
  }

  // Identify which caches to delete
  const { toDelete, kept } = identifyCachesToDelete(caches);

  core.info(`   Groups with latest entries kept: ${kept.length}`);
  for (const entry of kept) {
    core.info(`     ✓ Keeping: ${entry.key} (run ID: ${entry.runId})`);
  }
  core.info(`   Outdated entries to delete: ${toDelete.length}`);

  if (toDelete.length === 0) {
    core.info("✅ No outdated caches to clean up. All entries are current.");
    core.summary.addRaw(`## Cache Memory Cleanup\n\n✅ No outdated caches to clean up.\n- Total memory caches: ${caches.length}\n- Groups: ${kept.length}\n`);
    await core.summary.write();
    return;
  }

  // Delete outdated caches with throttling
  core.info(`🗑️ Deleting ${toDelete.length} outdated cache(s)...`);
  let deletedCount = 0;
  let failedCount = 0;
  /** @type {string[]} */
  const errors = [];

  for (const cache of toDelete) {
    // Check rate limit periodically (every 10 deletions)
    if (deletedCount > 0 && deletedCount % 10 === 0) {
      const { ok, remaining } = await checkRateLimit(github, "cleanup_cache_memory_periodic");
      if (!ok) {
        core.warning(`⚠️ Rate limit getting low (${remaining} remaining). Stopping deletion early.`);
        core.warning(`   Deleted ${deletedCount} of ${toDelete.length} caches before stopping.`);
        break;
      }
      core.info(`   Rate limit check: ${remaining} remaining`);
    }

    try {
      await github.rest.actions.deleteActionsCacheById({
        owner,
        repo,
        cache_id: cache.id,
      });
      deletedCount++;
      core.info(`   ✓ Deleted cache: ${cache.key} (run ID: ${cache.runId})`);
    } catch (error) {
      failedCount++;
      const msg = `Failed to delete cache ${cache.key}: ${getErrorMessage(error)}`;
      errors.push(msg);
      core.warning(`   ✗ ${msg}`);
    }

    // Throttle between deletions
    await delay(deleteDelayMs);
  }

  // Log final rate limit snapshot for observability
  await fetchAndLogRateLimit(github, "cleanup_cache_memory_end");

  // Summary
  core.info(`\n📊 Cache cleanup complete:`);
  core.info(`   Total memory caches found: ${caches.length}`);
  core.info(`   Groups (latest kept): ${kept.length}`);
  core.info(`   Outdated deleted: ${deletedCount}`);
  if (failedCount > 0) {
    core.info(`   Failed to delete: ${failedCount}`);
  }

  // Write job summary
  let summary = `## Cache Memory Cleanup\n\n`;
  summary += `| Metric | Count |\n|--------|-------|\n`;
  summary += `| Total memory caches | ${caches.length} |\n`;
  summary += `| Groups (latest kept) | ${kept.length} |\n`;
  summary += `| Outdated deleted | ${deletedCount} |\n`;
  if (failedCount > 0) {
    summary += `| Failed to delete | ${failedCount} |\n`;
  }
  if (errors.length > 0) {
    summary += `\n### Errors\n\n`;
    for (const err of errors) {
      summary += `- ${err}\n`;
    }
  }
  core.summary.addRaw(summary);
  await core.summary.write();

  core.info("✅ Cache memory cleanup finished");
}

module.exports = {
  main,
  parseCacheKey,
  identifyCachesToDelete,
  listMemoryCaches,
  MAX_LIST_PAGES,
};
