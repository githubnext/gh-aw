// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { getWorkflowIdMarkerContent } = require("./generate_footer.cjs");

/**
 * Maximum number of older issues to close
 */
const MAX_CLOSE_COUNT = 10;

/**
 * Delay between API calls in milliseconds to avoid rate limiting
 */
const API_DELAY_MS = 500;

/**
 * Delay execution for a specified number of milliseconds
 * @param {number} ms - Milliseconds to delay
 * @returns {Promise<void>}
 */
function delay(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Search for open issues with a matching workflow-id marker
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} workflowId - Workflow ID to match in the marker
 * @param {number} excludeNumber - Issue number to exclude (the newly created one)
 * @returns {Promise<Array<{number: number, title: string, html_url: string, labels: Array<{name: string}>}>>} Matching issues
 */
async function searchOlderIssues(github, owner, repo, workflowId, excludeNumber) {
  core.info(`Starting search for older issues in ${owner}/${repo}`);
  core.info(`  Workflow ID: ${workflowId || "(none)"}`);
  core.info(`  Exclude issue number: ${excludeNumber}`);

  if (!workflowId) {
    core.info("No workflow ID provided - cannot search for older issues");
    return [];
  }

  // Build REST API search query
  // Search for open issues with the workflow-id marker in the body
  const workflowIdMarker = getWorkflowIdMarkerContent(workflowId);
  // Escape quotes in workflow ID to prevent query injection
  const escapedMarker = workflowIdMarker.replace(/"/g, '\\"');
  const searchQuery = `repo:${owner}/${repo} is:issue is:open "${escapedMarker}" in:body`;

  core.info(`  Added workflow-id marker filter to query: "${escapedMarker}" in:body`);
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

  // Filter results:
  // 1. Must not be the excluded issue (newly created one)
  // 2. Must not be a pull request
  core.info("Filtering search results...");
  let filteredCount = 0;
  let pullRequestCount = 0;
  let excludedCount = 0;

  const filtered = result.data.items
    .filter(item => {
      // Exclude pull requests
      if (item.pull_request) {
        pullRequestCount++;
        return false;
      }

      // Exclude the newly created issue
      if (item.number === excludeNumber) {
        excludedCount++;
        core.info(`  Excluding issue #${item.number} (the newly created issue)`);
        return false;
      }

      filteredCount++;
      core.info(`  ✓ Issue #${item.number} matches criteria: ${item.title}`);
      return true;
    })
    .map(item => ({
      number: item.number,
      title: item.title,
      html_url: item.html_url,
      labels: item.labels || [],
    }));

  core.info(`Filtering complete:`);
  core.info(`  - Matched issues: ${filteredCount}`);
  core.info(`  - Excluded pull requests: ${pullRequestCount}`);
  core.info(`  - Excluded new issue: ${excludedCount}`);

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
    body: message,
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
 * @returns {Promise<Array<{number: number, html_url: string}>>} List of closed issues
 */
async function closeOlderIssues(github, owner, repo, workflowId, newIssue, workflowName, runUrl) {
  core.info("=".repeat(70));
  core.info("Starting closeOlderIssues operation");
  core.info("=".repeat(70));

  core.info(`Search criteria: workflow-id marker: "${getWorkflowIdMarkerContent(workflowId)}"`);
  core.info(`New issue reference: #${newIssue.number} (${newIssue.html_url})`);
  core.info(`Workflow: ${workflowName}`);
  core.info(`Run URL: ${runUrl}`);
  core.info("");

  const olderIssues = await searchOlderIssues(github, owner, repo, workflowId, newIssue.number);

  if (olderIssues.length === 0) {
    core.info("✓ No older issues found to close - operation complete");
    core.info("=".repeat(70));
    return [];
  }

  core.info("");
  core.info(`Found ${olderIssues.length} older issue(s) matching the criteria`);
  for (const issue of olderIssues) {
    core.info(`  - Issue #${issue.number}: ${issue.title}`);
    core.info(`    Labels: ${issue.labels.map(l => l.name).join(", ") || "(none)"}`);
    core.info(`    URL: ${issue.html_url}`);
  }

  // Limit to MAX_CLOSE_COUNT issues
  const issuesToClose = olderIssues.slice(0, MAX_CLOSE_COUNT);

  if (olderIssues.length > MAX_CLOSE_COUNT) {
    core.warning("");
    core.warning(`⚠️  Found ${olderIssues.length} older issues, but only closing the first ${MAX_CLOSE_COUNT}`);
    core.warning(`    The remaining ${olderIssues.length - MAX_CLOSE_COUNT} issue(s) will be processed in subsequent runs`);
  }

  core.info("");
  core.info(`Preparing to close ${issuesToClose.length} issue(s)...`);
  core.info("");

  const closedIssues = [];

  for (let i = 0; i < issuesToClose.length; i++) {
    const issue = issuesToClose[i];
    core.info("-".repeat(70));
    core.info(`Processing issue ${i + 1}/${issuesToClose.length}: #${issue.number}`);
    core.info(`  Title: ${issue.title}`);
    core.info(`  URL: ${issue.html_url}`);

    try {
      // Generate closing message
      const closingMessage = getCloseOlderIssueMessage({
        newIssueUrl: newIssue.html_url,
        newIssueNumber: newIssue.number,
        workflowName,
        runUrl,
      });

      core.info(`  Message length: ${closingMessage.length} characters`);
      core.info("");

      // Add comment first
      await addIssueComment(github, owner, repo, issue.number, closingMessage);

      // Then close the issue as "not planned"
      await closeIssueAsNotPlanned(github, owner, repo, issue.number);

      closedIssues.push({
        number: issue.number,
        html_url: issue.html_url,
      });

      core.info("");
      core.info(`✓ Successfully closed issue #${issue.number}`);
    } catch (error) {
      core.info("");
      core.error(`✗ Failed to close issue #${issue.number}`);
      core.error(`  Error: ${getErrorMessage(error)}`);
      if (error instanceof Error && error.stack) {
        core.error(`  Stack trace: ${error.stack}`);
      }
      // Continue with other issues even if one fails
    }

    // Add delay between API operations to avoid rate limiting (except for the last item)
    if (i < issuesToClose.length - 1) {
      core.info("");
      core.info(`Waiting ${API_DELAY_MS}ms before processing next issue to avoid rate limiting...`);
      await delay(API_DELAY_MS);
    }
  }

  core.info("");
  core.info("=".repeat(70));
  core.info(`Closed ${closedIssues.length} of ${issuesToClose.length} issue(s) successfully`);
  if (closedIssues.length < issuesToClose.length) {
    core.warning(`Failed to close ${issuesToClose.length - closedIssues.length} issue(s) - check logs above for details`);
  }
  core.info("=".repeat(70));

  return closedIssues;
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
