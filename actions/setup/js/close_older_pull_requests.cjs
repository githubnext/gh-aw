// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeContent } = require("./sanitize_content.cjs");
const { closeOlderEntities, MAX_CLOSE_COUNT: SHARED_MAX_CLOSE_COUNT } = require("./close_older_entities.cjs");
const { buildMarkerSearchQuery, filterByMarker, logFilterSummary } = require("./close_older_search_helpers.cjs");

/**
 * Maximum number of older pull requests to close
 */
const MAX_CLOSE_COUNT = SHARED_MAX_CLOSE_COUNT;

/**
 * Delay between API calls in milliseconds to avoid rate limiting
 */
const API_DELAY_MS = 500;

/**
 * Search for open pull requests with a matching workflow-id marker
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} workflowId - Workflow ID to match in the marker
 * @param {number} excludeNumber - PR number to exclude (the newly created one)
 * @param {string} [callerWorkflowId] - Optional calling workflow identity for precise filtering.
 *   When set, filters by the `gh-aw-workflow-call-id` marker so callers sharing the same
 *   reusable workflow do not close each other's PRs. Falls back to `gh-aw-workflow-id`
 *   when not provided (backward compat for PRs created before this fix).
 * @param {string} [closeOlderKey] - Optional explicit deduplication key. When set, the
 *   `gh-aw-close-key` marker is used as the primary search term and exact filter instead
 *   of the workflow-id / workflow-call-id markers.
 * @returns {Promise<Array<{number: number, title: string, html_url: string, labels: Array<{name: string}>, created_at: string}>>} Matching pull requests
 */
async function searchOlderPullRequests(github, owner, repo, workflowId, excludeNumber, callerWorkflowId, closeOlderKey) {
  core.info(`Starting search for older pull requests in ${owner}/${repo}`);
  core.info(`  Workflow ID: ${workflowId || "(none)"}`);
  core.info(`  Exclude PR number: ${excludeNumber}`);

  if (!workflowId && !closeOlderKey) {
    core.info("No workflow ID or close-older-key provided - cannot search for older pull requests");
    return [];
  }

  const { searchQuery, exactMarker } = buildMarkerSearchQuery({
    owner,
    repo,
    workflowId,
    callerWorkflowId,
    closeOlderKey,
    entityQualifier: "is:pr",
  });
  core.info(`Executing GitHub search with query: ${searchQuery}`);

  const result = await github.rest.search.issuesAndPullRequests({
    q: searchQuery,
    per_page: 50,
  });

  core.info(`Search API returned ${result?.data?.items?.length || 0} total results`);

  if (!result || !result.data || !result.data.items) {
    core.info("No results returned from search API");
    return [];
  }

  core.info("Filtering search results...");

  const { filtered: filteredItems, counters } = filterByMarker({
    items: result.data.items,
    excludeNumber,
    exactMarker,
    entityType: "pull request",
    additionalFilter: (item, extra) => {
      if (!item.pull_request) {
        extra.issueCount = (extra.issueCount || 0) + 1;
        return false;
      }
      return true;
    },
  });

  const filtered = filteredItems.map(item => ({
    number: item.number,
    title: item.title,
    html_url: item.html_url,
    labels: item.labels || [],
    created_at: item.created_at,
  }));

  logFilterSummary({
    entityTypePlural: "pull requests",
    counters,
    extraLabels: [["issueCount", "Excluded issues"]],
  });

  return filtered;
}

/**
 * Add comment to a GitHub Pull Request using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @param {string} message - Comment body
 * @returns {Promise<{id: number, html_url: string}>} Comment details
 */
async function addPullRequestComment(github, owner, repo, prNumber, message) {
  core.info(`Adding comment to pull request #${prNumber} in ${owner}/${repo}`);
  core.info(`  Comment length: ${message.length} characters`);

  const result = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: prNumber,
    body: sanitizeContent(message),
  });

  core.info(`  ✓ Comment created successfully with ID: ${result.data.id}`);
  core.info(`  Comment URL: ${result.data.html_url}`);

  return {
    id: result.data.id,
    html_url: result.data.html_url,
  };
}

/**
 * Close a GitHub Pull Request using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @returns {Promise<{number: number, html_url: string}>} Pull request details
 */
async function closePullRequest(github, owner, repo, prNumber) {
  core.info(`Closing pull request #${prNumber} in ${owner}/${repo}`);

  const result = await github.rest.pulls.update({
    owner,
    repo,
    pull_number: prNumber,
    state: "closed",
  });

  core.info(`  ✓ Pull request #${result.data.number} closed successfully`);
  core.info(`  Pull request URL: ${result.data.html_url}`);

  return {
    number: result.data.number,
    html_url: result.data.html_url,
  };
}

/**
 * Generate closing message for older pull requests
 * @param {object} params - Parameters for the message
 * @param {string} params.newPullRequestUrl - URL of the new pull request
 * @param {number} params.newPullRequestNumber - Number of the new pull request
 * @param {string} params.workflowName - Name of the workflow
 * @param {string} params.runUrl - URL of the workflow run
 * @returns {string} Closing message
 */
function getCloseOlderPullRequestMessage({ newPullRequestUrl, newPullRequestNumber, workflowName, runUrl }) {
  return `This pull request is being closed as superseded. A newer pull request has been created: #${newPullRequestNumber}

[View newer pull request](${newPullRequestUrl})

---

*This action was performed automatically by the [\`${workflowName}\`](${runUrl}) workflow.*`;
}

/**
 * Close older pull requests that match the workflow-id marker
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} workflowId - Workflow ID to match in the marker
 * @param {{number: number, html_url: string}} newPullRequest - The newly created pull request
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @param {string} [callerWorkflowId] - Optional calling workflow identity for precise filtering
 * @param {string} [closeOlderKey] - Optional explicit deduplication key for close-older matching
 * @returns {Promise<Array<{number: number, html_url: string}>>} List of closed pull requests
 */
async function closeOlderPullRequests(github, owner, repo, workflowId, newPullRequest, workflowName, runUrl, callerWorkflowId, closeOlderKey) {
  const result = await closeOlderEntities(github, owner, repo, workflowId, newPullRequest, workflowName, runUrl, {
    entityType: "pull request",
    entityTypePlural: "pull requests",
    // Use a closure so callerWorkflowId and closeOlderKey are forwarded to searchOlderPullRequests
    // without going through the closeOlderEntities extraArgs mechanism (which appends
    // excludeNumber last)
    searchOlderEntities: (gh, o, r, wid, excludeNumber) => searchOlderPullRequests(gh, o, r, wid, excludeNumber, callerWorkflowId, closeOlderKey),
    getCloseMessage: params =>
      getCloseOlderPullRequestMessage({
        newPullRequestUrl: params.newEntityUrl,
        newPullRequestNumber: params.newEntityNumber,
        workflowName: params.workflowName,
        runUrl: params.runUrl,
      }),
    addComment: addPullRequestComment,
    closeEntity: closePullRequest,
    delayMs: API_DELAY_MS,
    getEntityId: entity => entity.number,
    getEntityUrl: entity => entity.html_url,
  });

  // Map to pull-request-specific return type
  return result.map(item => ({
    number: item.number,
    html_url: item.html_url || "",
  }));
}

module.exports = {
  closeOlderPullRequests,
  searchOlderPullRequests,
  addPullRequestComment,
  closePullRequest,
  getCloseOlderPullRequestMessage,
  MAX_CLOSE_COUNT,
  API_DELAY_MS,
};
