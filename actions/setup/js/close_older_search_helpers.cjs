// @ts-check
/// <reference types="@actions/github-script" />

const { getWorkflowIdMarkerContent, generateWorkflowIdMarker, generateWorkflowCallIdMarker, generateCloseKeyMarker, getCloseKeyMarkerContent } = require("./generate_footer.cjs");

/**
 * Build the search query string and the exact marker used for body-level
 * filtering after the GitHub search API returns results.
 *
 * The logic is shared between the REST (issues) and GraphQL (discussions)
 * search paths – only the `entityQualifier` differs (e.g. `"is:issue"` for
 * issues, or `""` for discussions which don't need one).
 *
 * @param {object} params
 * @param {string} params.owner - Repository owner
 * @param {string} params.repo - Repository name
 * @param {string} params.workflowId - Workflow ID to match in the marker
 * @param {string} [params.callerWorkflowId] - Optional calling workflow identity
 * @param {string} [params.closeOlderKey] - Optional explicit deduplication key
 * @param {string} [params.entityQualifier] - Extra qualifier appended to the query (e.g. "is:issue")
 * @returns {{ searchQuery: string, exactMarker: string }}
 */
function buildMarkerSearchQuery({ owner, repo, workflowId, callerWorkflowId, closeOlderKey, entityQualifier }) {
  const qualifierSegment = entityQualifier ? ` ${entityQualifier}` : "";

  if (closeOlderKey) {
    const closeKeyMarkerContent = getCloseKeyMarkerContent(closeOlderKey);
    const escapedMarker = closeKeyMarkerContent.replace(/"/g, '\\"');
    const searchQuery = `repo:${owner}/${repo}${qualifierSegment} is:open "${escapedMarker}" in:body`;
    const exactMarker = generateCloseKeyMarker(closeOlderKey);
    core.info(`  Using close-older-key for search: "${escapedMarker}" in:body`);
    return { searchQuery, exactMarker };
  }

  const workflowIdMarker = getWorkflowIdMarkerContent(workflowId);
  const escapedMarker = workflowIdMarker.replace(/"/g, '\\"');
  const searchQuery = `repo:${owner}/${repo}${qualifierSegment} is:open "${escapedMarker}" in:body`;
  const exactMarker = callerWorkflowId ? generateWorkflowCallIdMarker(callerWorkflowId) : generateWorkflowIdMarker(workflowId);
  core.info(`  Added workflow-id marker filter to query: "${escapedMarker}" in:body`);
  return { searchQuery, exactMarker };
}

/**
 * Shared older-entity search pipeline used by issues, pull requests, and
 * discussions. Entity-specific callers provide the search executor,
 * result-shape extractor, and item mapper while this helper centralizes the
 * query/log/filter flow.
 *
 * @param {object} params
 * @param {string} params.owner - Repository owner
 * @param {string} params.repo - Repository name
 * @param {string} params.workflowId - Workflow ID to match in the marker
 * @param {number} params.excludeNumber - Entity number to exclude (the newly created one)
 * @param {string} params.entityType - Entity type name for logging (e.g. "issue", "discussion")
 * @param {string} [params.callerWorkflowId] - Optional calling workflow identity
 * @param {string} [params.closeOlderKey] - Optional explicit deduplication key
 * @param {string} [params.entityQualifier] - Extra qualifier appended to the query
 * @param {Set<number>} [params.additionalExcludeNumbers] - Optional additional entity numbers
 *   to exclude from the results.
 * @param {(searchQuery: string) => Promise<any>} params.executeSearch - Entity-specific API call
 * @param {(result: any) => Array<any>|undefined|null} params.getItems - Extract search items from API result
 * @param {(item: any) => any} params.mapItem - Map a filtered item to the caller's return shape
 * @param {(item: any, counters: Record<string, number>) => boolean} [params.additionalFilter] - Optional
 *   entity-specific filter callback.
 * @param {Array<[string, string]>} [params.extraLabels] - Extra summary counters to log
 * @returns {Promise<Array<any>>} Filtered and mapped search results
 */
async function searchOlderEntitiesByMarker({
  owner,
  repo,
  workflowId,
  excludeNumber,
  entityType,
  callerWorkflowId,
  closeOlderKey,
  entityQualifier,
  additionalExcludeNumbers,
  executeSearch,
  getItems,
  mapItem,
  additionalFilter,
  extraLabels,
}) {
  core.info(`Starting search for older ${entityType}s in ${owner}/${repo}`);
  core.info(`  Workflow ID: ${workflowId || "(none)"}`);
  core.info(`  Exclude ${entityType} number: ${excludeNumber}`);

  if (!workflowId && !closeOlderKey) {
    core.info(`No workflow ID or close-older-key provided - cannot search for older ${entityType}s`);
    return [];
  }

  const { searchQuery, exactMarker } = buildMarkerSearchQuery({
    owner,
    repo,
    workflowId,
    callerWorkflowId,
    closeOlderKey,
    entityQualifier,
  });
  core.info(`Executing GitHub search with query: ${searchQuery}`);

  const result = await executeSearch(searchQuery);
  const items = getItems(result);

  core.info(`Search API returned ${items?.length || 0} total results`);

  if (!items) {
    core.info("No results returned from search API");
    return [];
  }

  core.info("Filtering search results...");

  const { filtered: filteredItems, counters } = filterByMarker({
    items,
    excludeNumber,
    additionalExcludeNumbers,
    exactMarker,
    entityType,
    additionalFilter,
  });

  const filtered = filteredItems.map(mapItem);

  logFilterSummary({
    entityTypePlural: `${entityType}s`,
    counters,
    extraLabels,
  });

  return filtered;
}

/**
 * @typedef {Object} SearchResultItem
 * @property {number} number - Entity number
 * @property {string} [body] - Entity body text
 */

/**
 * @typedef {Object} FilterCounters
 * @property {number} filteredCount - Number of entities that passed all filters
 * @property {number} excludedCount - Number of entities excluded as the "new" entity
 * @property {number} markerMismatchCount - Number of entities excluded due to marker mismatch
 */

/**
 * Filter search results by excluding the newly created entity and verifying
 * the exact marker is present in the body. Entity-specific additional filters
 * (e.g. pull-request exclusion for issues, closed/category checks for
 * discussions) are handled by the optional `additionalFilter` callback.
 *
 * @param {object} params
 * @param {Array<any>} params.items - Raw search result items
 * @param {number} params.excludeNumber - Entity number to exclude (the newly created one)
 * @param {Set<number>} [params.additionalExcludeNumbers] - Optional set of additional entity
 *   numbers to exclude (e.g. all issues created in the current run)
 * @param {string} params.exactMarker - Exact marker string that must appear in the body
 * @param {string} params.entityType - Entity type name for logging (e.g. "issue", "discussion")
 * @param {(item: any, counters: Record<string, number>) => boolean} [params.additionalFilter] -
 *   Optional callback for entity-specific filtering. Return `true` to keep the item.
 *   The `counters` object can be mutated to track extra exclusion reasons.
 * @returns {{ filtered: Array<any>, counters: FilterCounters & Record<string, number> }}
 */
function filterByMarker({ items, excludeNumber, additionalExcludeNumbers, exactMarker, entityType, additionalFilter }) {
  let filteredCount = 0;
  let excludedCount = 0;
  let markerMismatchCount = 0;
  /** @type {Record<string, number>} */
  const extraCounters = {};

  const filtered = items.filter(item => {
    if (!item) {
      return false;
    }

    // Exclude the newly created entity and any other entities created in the
    // same run before running any other filters, so counters/logs consistently
    // attribute these items to the dedicated exclusion.
    if (item.number === excludeNumber) {
      excludedCount++;
      core.info(`  Excluding ${entityType} #${item.number} (the newly created ${entityType})`);
      return false;
    }
    if (additionalExcludeNumbers?.has(item.number)) {
      excludedCount++;
      core.info(`  Excluding ${entityType} #${item.number} (created earlier in the same run)`);
      return false;
    }

    // Run entity-specific filters next (e.g. pull_request, closed, category)
    if (additionalFilter && !additionalFilter(item, extraCounters)) {
      return false;
    }

    // Exact-match the marker in the body to prevent GitHub search
    // substring tokenization from matching related workflow IDs
    // (e.g. "foo" would otherwise match entities from "foo-bar")
    if (!item.body?.includes(exactMarker)) {
      markerMismatchCount++;
      core.info(`  Excluding ${entityType} #${item.number} (body does not contain exact marker)`);
      return false;
    }

    filteredCount++;
    core.info(`  ✓ ${entityType.charAt(0).toUpperCase() + entityType.slice(1)} #${item.number} matches criteria: ${item.title}`);
    return true;
  });

  return {
    filtered,
    counters: { filteredCount, excludedCount, markerMismatchCount, ...extraCounters },
  };
}

/**
 * Log the filtering summary counters.
 *
 * @param {object} params
 * @param {string} params.entityTypePlural - Plural entity name (e.g. "issues")
 * @param {FilterCounters & Record<string, number>} params.counters - Counters from filterByMarker
 * @param {Array<[string, string]>} [params.extraLabels] - Additional counter labels to log
 *   as `[counterKey, label]` pairs (e.g. `[["pullRequestCount", "Excluded pull requests"]]`)
 */
function logFilterSummary({ entityTypePlural, counters, extraLabels }) {
  core.info(`Filtering complete:`);
  core.info(`  - Matched ${entityTypePlural}: ${counters.filteredCount}`);
  if (extraLabels) {
    for (const [key, label] of extraLabels) {
      core.info(`  - ${label}: ${counters[key] || 0}`);
    }
  }
  core.info(`  - Excluded new ${entityTypePlural.slice(0, -1)}: ${counters.excludedCount}`);
  core.info(`  - Excluded marker mismatch: ${counters.markerMismatchCount}`);
}

module.exports = {
  buildMarkerSearchQuery,
  searchOlderEntitiesByMarker,
  filterByMarker,
  logFilterSummary,
};
