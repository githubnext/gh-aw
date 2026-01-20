// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

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
 * Search for open issues with a matching title prefix and/or labels
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} titlePrefix - Title prefix to match (empty string to skip prefix matching)
 * @param {string[]} labels - Labels to match (empty array to skip label matching)
 * @param {number} excludeNumber - Issue number to exclude (the newly created one)
 * @returns {Promise<Array<{number: number, title: string, html_url: string, labels: Array<{name: string}>}>>} Matching issues
 */
async function searchOlderIssues(github, owner, repo, titlePrefix, labels, excludeNumber) {
  // Build REST API search query
  // Search for open issues, optionally with title prefix or labels
  let searchQuery = `repo:${owner}/${repo} is:issue is:open`;

  if (titlePrefix) {
    // Escape quotes in title prefix to prevent query injection
    const escapedPrefix = titlePrefix.replace(/"/g, '\\"');
    searchQuery += ` in:title "${escapedPrefix}"`;
  }

  // Add label filters to the search query
  // Note: GitHub search uses AND logic for multiple labels, so issues must have ALL labels.
  // We add each label as a separate filter and also validate client-side for extra safety.
  if (labels && labels.length > 0) {
    for (const label of labels) {
      // Escape quotes in label names to prevent query injection
      const escapedLabel = label.replace(/"/g, '\\"');
      searchQuery += ` label:"${escapedLabel}"`;
    }
  }

  core.info(`Searching with query: ${searchQuery}`);

  const result = await github.rest.search.issuesAndPullRequests({
    q: searchQuery,
    per_page: 50,
  });

  if (!result || !result.data || !result.data.items) {
    return [];
  }

  // Filter results:
  // 1. Must not be the excluded issue (newly created one)
  // 2. Must not be a pull request
  // 3. If titlePrefix is specified, must have title starting with the prefix
  // 4. If labels are specified, must have ALL specified labels (AND logic, not OR)
  return result.data.items
    .filter(item => {
      // Exclude pull requests
      if (item.pull_request) {
        return false;
      }

      // Exclude the newly created issue
      if (item.number === excludeNumber) {
        return false;
      }

      // Check title prefix if specified
      if (titlePrefix && item.title && !item.title.startsWith(titlePrefix)) {
        return false;
      }

      // Check labels if specified - requires ALL labels to match (AND logic)
      // This is intentional: we only want to close issues that have ALL the specified labels
      if (labels && labels.length > 0) {
        const issueLabels = item.labels?.map(l => l.name) || [];
        const hasAllLabels = labels.every(label => issueLabels.includes(label));
        if (!hasAllLabels) {
          return false;
        }
      }

      return true;
    })
    .map(item => ({
      number: item.number,
      title: item.title,
      html_url: item.html_url,
      labels: item.labels || [],
    }));
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
  const result = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
    body: message,
  });

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
  const result = await github.rest.issues.update({
    owner,
    repo,
    issue_number: issueNumber,
    state: "closed",
    state_reason: "not_planned",
  });

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
 * Close older issues that match the title prefix and/or labels
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {string} titlePrefix - Title prefix to match (empty string to skip)
 * @param {string[]} labels - Labels to match (empty array to skip)
 * @param {{number: number, html_url: string}} newIssue - The newly created issue
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @returns {Promise<Array<{number: number, html_url: string}>>} List of closed issues
 */
async function closeOlderIssues(github, owner, repo, titlePrefix, labels, newIssue, workflowName, runUrl) {
  // Build search criteria description for logging
  const searchCriteria = [];
  if (titlePrefix) searchCriteria.push(`title prefix: "${titlePrefix}"`);
  if (labels && labels.length > 0) searchCriteria.push(`labels: [${labels.join(", ")}]`);
  core.info(`Searching for older issues with ${searchCriteria.join(" and ")}`);

  const olderIssues = await searchOlderIssues(github, owner, repo, titlePrefix, labels, newIssue.number);

  if (olderIssues.length === 0) {
    core.info("No older issues found to close");
    return [];
  }

  core.info(`Found ${olderIssues.length} older issue(s) to close`);

  // Limit to MAX_CLOSE_COUNT issues
  const issuesToClose = olderIssues.slice(0, MAX_CLOSE_COUNT);

  if (olderIssues.length > MAX_CLOSE_COUNT) {
    core.warning(`Found ${olderIssues.length} older issues, but only closing the first ${MAX_CLOSE_COUNT}`);
  }

  const closedIssues = [];

  for (let i = 0; i < issuesToClose.length; i++) {
    const issue = issuesToClose[i];
    try {
      // Generate closing message
      const closingMessage = getCloseOlderIssueMessage({
        newIssueUrl: newIssue.html_url,
        newIssueNumber: newIssue.number,
        workflowName,
        runUrl,
      });

      // Add comment first
      core.info(`Adding closing comment to issue #${issue.number}`);
      await addIssueComment(github, owner, repo, issue.number, closingMessage);

      // Then close the issue as "not planned"
      core.info(`Closing issue #${issue.number} as not planned`);
      await closeIssueAsNotPlanned(github, owner, repo, issue.number);

      closedIssues.push({
        number: issue.number,
        html_url: issue.html_url,
      });

      core.info(`✓ Closed issue #${issue.number}: ${issue.html_url}`);
    } catch (error) {
      core.error(`✗ Failed to close issue #${issue.number}: ${getErrorMessage(error)}`);
      // Continue with other issues even if one fails
    }

    // Add delay between API operations to avoid rate limiting (except for the last item)
    if (i < issuesToClose.length - 1) {
      await delay(API_DELAY_MS);
    }
  }

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
