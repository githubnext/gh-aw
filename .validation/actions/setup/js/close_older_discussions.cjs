// @ts-check
/// <reference types="@actions/github-script" />

const { getCloseOlderDiscussionMessage } = require("./messages_close_discussion.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { sanitizeContent } = require("./sanitize_content.cjs");
const { closeOlderEntities, MAX_CLOSE_COUNT: SHARED_MAX_CLOSE_COUNT } = require("./close_older_entities.cjs");
const { buildMarkerSearchQuery, filterByMarker, logFilterSummary } = require("./close_older_search_helpers.cjs");

/**
 * Maximum number of older discussions to close
 */
const MAX_CLOSE_COUNT = SHARED_MAX_CLOSE_COUNT;

/**
 * Delay between GraphQL API calls in milliseconds to avoid rate limiting
 */
const GRAPHQL_DELAY_MS = 500;

/**
 * Search for open discussions with a matching workflow-id marker
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} workflowId - Workflow ID to match in the marker
 * @param {string|undefined} categoryId - Optional category ID to filter by
 * @param {number} excludeNumber - Discussion number to exclude (the newly created one)
 * @param {string} [callerWorkflowId] - Optional calling workflow identity for precise filtering.
 *   When set, filters by the `gh-aw-workflow-call-id` marker so callers sharing the same
 *   reusable workflow do not close each other's discussions. Falls back to `gh-aw-workflow-id`
 *   when not provided (backward compat for discussions created before this fix).
 * @param {string} [closeOlderKey] - Optional explicit deduplication key. When set, the
 *   `gh-aw-close-key` marker is used as the primary search term and exact filter instead
 *   of the workflow-id / workflow-call-id markers.
 * @returns {Promise<Array<{id: string, number: number, title: string, url: string}>>} Matching discussions
 */
async function searchOlderDiscussions(github, owner, repo, workflowId, categoryId, excludeNumber, callerWorkflowId, closeOlderKey) {
  core.info(`Starting search for older discussions in ${owner}/${repo}`);
  core.info(`  Workflow ID: ${workflowId || "(none)"}`);
  core.info(`  Exclude discussion number: ${excludeNumber}`);

  if (!workflowId && !closeOlderKey) {
    core.info("No workflow ID or close-older-key provided - cannot search for older discussions");
    return [];
  }

  const { searchQuery, exactMarker } = buildMarkerSearchQuery({
    owner,
    repo,
    workflowId,
    callerWorkflowId,
    closeOlderKey,
  });
  core.info(`Executing GitHub search with query: ${searchQuery}`);

  const result = await github.graphql(
    `
    query($searchTerms: String!, $first: Int!) {
      search(query: $searchTerms, type: DISCUSSION, first: $first) {
        nodes {
          ... on Discussion {
            id
            number
            title
            url
            body
            category {
              id
            }
            closed
          }
        }
      }
    }`,
    { searchTerms: searchQuery, first: 50 }
  );

  core.info(`Search API returned ${result?.search?.nodes?.length || 0} total results`);

  if (!result || !result.search || !result.search.nodes) {
    core.info("No results returned from search API");
    return [];
  }

  core.info("Filtering search results...");

  const { filtered: filteredItems, counters } = filterByMarker({
    items: result.search.nodes,
    excludeNumber,
    exactMarker,
    entityType: "discussion",
    additionalFilter: (d, extra) => {
      if (d.closed) {
        extra.closedCount = (extra.closedCount || 0) + 1;
        return false;
      }
      if (categoryId && (!d.category || d.category.id !== categoryId)) {
        return false;
      }
      return true;
    },
  });

  const filtered = filteredItems.map(
    /** @param {any} d */ d => ({
      id: d.id,
      number: d.number,
      title: d.title,
      url: d.url,
    })
  );

  logFilterSummary({
    entityTypePlural: "discussions",
    counters,
    extraLabels: [["closedCount", "Excluded closed discussions"]],
  });

  return filtered;
}

/**
 * Add comment to a GitHub Discussion using GraphQL
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner (unused for GraphQL, but kept for consistency)
 * @param {string} repo - Repository name (unused for GraphQL, but kept for consistency)
 * @param {string} discussionId - Discussion node ID
 * @param {string} message - Comment body
 * @returns {Promise<{id: string, url: string}>} Comment details
 */
async function addDiscussionComment(github, owner, repo, discussionId, message) {
  const result = await github.graphql(
    `
    mutation($dId: ID!, $body: String!) {
      addDiscussionComment(input: { discussionId: $dId, body: $body }) {
        comment { 
          id 
          url
        }
      }
    }`,
    { dId: discussionId, body: sanitizeContent(message) }
  );

  return result.addDiscussionComment.comment;
}

/**
 * Close a GitHub Discussion as OUTDATED using GraphQL
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner (unused for GraphQL, but kept for consistency)
 * @param {string} repo - Repository name (unused for GraphQL, but kept for consistency)
 * @param {string} discussionId - Discussion node ID
 * @returns {Promise<{id: string, url: string}>} Discussion details
 */
async function closeDiscussionAsOutdated(github, owner, repo, discussionId) {
  const result = await github.graphql(
    `
    mutation($dId: ID!) {
      closeDiscussion(input: { discussionId: $dId, reason: OUTDATED }) {
        discussion { 
          id
          url
        }
      }
    }`,
    { dId: discussionId }
  );

  return result.closeDiscussion.discussion;
}

/**
 * Close older discussions that match the workflow-id marker
 * @param {any} github - GitHub GraphQL instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} workflowId - Workflow ID to match in the marker
 * @param {string|undefined} categoryId - Optional category ID to filter by
 * @param {{number: number, url: string}} newDiscussion - The newly created discussion
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @param {string} [callerWorkflowId] - Optional calling workflow identity for precise filtering
 * @param {string} [closeOlderKey] - Optional explicit deduplication key for close-older matching
 * @returns {Promise<Array<{number: number, url: string}>>} List of closed discussions
 */
async function closeOlderDiscussions(github, owner, repo, workflowId, categoryId, newDiscussion, workflowName, runUrl, callerWorkflowId, closeOlderKey) {
  const result = await closeOlderEntities(
    github,
    owner,
    repo,
    workflowId,
    newDiscussion,
    workflowName,
    runUrl,
    {
      entityType: "discussion",
      entityTypePlural: "discussions",
      // Use a closure so callerWorkflowId and closeOlderKey are forwarded to
      // searchOlderDiscussions without going through the closeOlderEntities extraArgs
      // mechanism (which appends excludeNumber last)
      searchOlderEntities: (gh, o, r, wid, categoryId, excludeNumber) => searchOlderDiscussions(gh, o, r, wid, categoryId, excludeNumber, callerWorkflowId, closeOlderKey),
      getCloseMessage: params =>
        getCloseOlderDiscussionMessage({
          newDiscussionUrl: params.newEntityUrl,
          newDiscussionNumber: params.newEntityNumber,
          workflowName: params.workflowName,
          runUrl: params.runUrl,
        }),
      addComment: addDiscussionComment,
      closeEntity: closeDiscussionAsOutdated,
      delayMs: GRAPHQL_DELAY_MS,
      getEntityId: entity => entity.id,
      getEntityUrl: entity => entity.url,
    },
    categoryId // Pass categoryId as extra arg
  );

  // Map to discussion-specific return type
  return result.map(item => ({
    number: item.number,
    url: item.url || "",
  }));
}

module.exports = {
  closeOlderDiscussions,
  searchOlderDiscussions,
  addDiscussionComment,
  closeDiscussionAsOutdated,
  MAX_CLOSE_COUNT,
  GRAPHQL_DELAY_MS,
};
