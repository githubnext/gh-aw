// @ts-check
/// <reference types="@actions/github-script" />

const { sanitizeContent } = require("./sanitize_content.cjs");
const { closeOlderEntities, MAX_CLOSE_COUNT: SHARED_MAX_CLOSE_COUNT } = require("./close_older_entities.cjs");
const { buildMarkerSearchQuery, filterByMarker, logFilterSummary } = require("./close_older_search_helpers.cjs");

/**
 * Maximum number of older issues to close
 */
const MAX_CLOSE_COUNT = SHARED_MAX_CLOSE_COUNT;

/**
 * Delay between API calls in milliseconds to avoid rate limiting
 */
const API_DELAY_MS = 500;

/**
 * Search for open issues with a matching workflow-id marker
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} workflowId - Workflow ID to match in the marker
 * @param {number} excludeNumber - Issue number to exclude (the newly created one)
 * @param {string} [callerWorkflowId] - Optional calling workflow identity for precise filtering.
 *   When set, filters by the `gh-aw-workflow-call-id` marker so callers sharing the same
 *   reusable workflow do not close each other's issues. Falls back to `gh-aw-workflow-id`
 *   when not provided (backward compat for issues created before this fix).
 * @param {string} [closeOlderKey] - Optional explicit deduplication key. When set, the
 *   `gh-aw-close-key` marker is used as the primary search term and exact filter instead
 *   of the workflow-id / workflow-call-id markers.
 * @returns {Promise<Array<{number: number, title: string, html_url: string, labels: Array<{name: string}>, created_at: string}>>} Matching issues
 */
async function searchOlderIssues(github, owner, repo, workflowId, excludeNumber, callerWorkflowId, closeOlderKey) {
  core.info(`Starting search for older issues in ${owner}/${repo}`);
  core.info(`  Workflow ID: ${workflowId || "(none)"}`);
  core.info(`  Exclude issue number: ${excludeNumber}`);

  if (!workflowId && !closeOlderKey) {
    core.info("No workflow ID or close-older-key provided - cannot search for older issues");
    return [];
  }

  const { searchQuery, exactMarker } = buildMarkerSearchQuery({
    owner,
    repo,
    workflowId,
    callerWorkflowId,
    closeOlderKey,
    entityQualifier: "is:issue",
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
    entityType: "issue",
    additionalFilter: (item, extra) => {
      if (item.pull_request) {
        extra.pullRequestCount = (extra.pullRequestCount || 0) + 1;
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
    entityTypePlural: "issues",
    counters,
    extraLabels: [["pullRequestCount", "Excluded pull requests"]],
  });

  return filtered;
}

/**
 * Add comment to a GitHub Issue using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} message - Comment body
 * @returns {Promise<{id: number, html_url: string}>} Comment details
 */
async function addIssueComment(github, owner, repo, issueNumber, message) {
  core.info(`Adding comment to issue #${issueNumber} in ${owner}/${repo}`);
  core.info(`  Comment length: ${message.length} characters`);

  const result = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
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
 * Close a GitHub Issue as "not planned" using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<{number: number, html_url: string}>} Issue details
 */
async function closeIssueAsNotPlanned(github, owner, repo, issueNumber) {
  core.info(`Closing issue #${issueNumber} in ${owner}/${repo} as "not planned"`);

  const result = await github.rest.issues.update({
    owner,
    repo,
    issue_number: issueNumber,
    state: "closed",
    state_reason: "not_planned",
  });

  core.info(`  ✓ Issue #${result.data.number} closed successfully`);
  core.info(`  Issue URL: ${result.data.html_url}`);

  return {
    number: result.data.number,
    html_url: result.data.html_url,
  };
}

/**
 * Generate closing message for older issues
 * @param {object} params - Parameters for the message
 * @param {string} params.newIssueUrl - URL of the new issue
 * @param {number} params.newIssueNumber - Number of the new issue
 * @param {string} params.workflowName - Name of the workflow
 * @param {string} params.runUrl - URL of the workflow run
 * @returns {string} Closing message
 */
function getCloseOlderIssueMessage({ newIssueUrl, newIssueNumber, workflowName, runUrl }) {
  return `This issue is being closed as outdated. A newer issue has been created: #${newIssueNumber}

[View newer issue](${newIssueUrl})

---

*This action was performed automatically by the [\`${workflowName}\`](${runUrl}) workflow.*`;
}

/**
 * Close older issues that match the workflow-id marker
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} workflowId - Workflow ID to match in the marker
 * @param {{number: number, html_url: string}} newIssue - The newly created issue
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @param {string} [callerWorkflowId] - Optional calling workflow identity for precise filtering
 * @param {string} [closeOlderKey] - Optional explicit deduplication key for close-older matching
 * @returns {Promise<Array<{number: number, html_url: string}>>} List of closed issues
 */
async function closeOlderIssues(github, owner, repo, workflowId, newIssue, workflowName, runUrl, callerWorkflowId, closeOlderKey) {
  const result = await closeOlderEntities(github, owner, repo, workflowId, newIssue, workflowName, runUrl, {
    entityType: "issue",
    entityTypePlural: "issues",
    // Use a closure so callerWorkflowId and closeOlderKey are forwarded to searchOlderIssues
    // without going through the closeOlderEntities extraArgs mechanism (which appends
    // excludeNumber last)
    searchOlderEntities: (gh, o, r, wid, excludeNumber) => searchOlderIssues(gh, o, r, wid, excludeNumber, callerWorkflowId, closeOlderKey),
    getCloseMessage: params =>
      getCloseOlderIssueMessage({
        newIssueUrl: params.newEntityUrl,
        newIssueNumber: params.newEntityNumber,
        workflowName: params.workflowName,
        runUrl: params.runUrl,
      }),
    addComment: addIssueComment,
    closeEntity: closeIssueAsNotPlanned,
    delayMs: API_DELAY_MS,
    getEntityId: entity => entity.number,
    getEntityUrl: entity => entity.html_url,
  });

  // Map to issue-specific return type
  return result.map(item => ({
    number: item.number,
    html_url: item.html_url || "",
  }));
}

module.exports = {
  closeOlderIssues,
  searchOlderIssues,
  addIssueComment,
  closeIssueAsNotPlanned,
  getCloseOlderIssueMessage,
  MAX_CLOSE_COUNT,
  API_DELAY_MS,
};
