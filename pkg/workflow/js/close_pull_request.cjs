// @ts-check
/// <reference types="@actions/github-script" />

const { processCloseEntityItems, PULL_REQUEST_CONFIG } = require("./close_entity_helpers.cjs");

/**
 * Get pull request details using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @returns {Promise<{number: number, title: string, labels: Array<{name: string}>, html_url: string, state: string}>} Pull request details
 */
async function getPullRequestDetails(github, owner, repo, prNumber) {
  const { data: pr } = await github.rest.pulls.get({
    owner,
    repo,
    pull_number: prNumber,
  });

  if (!pr) {
    throw new Error(`Pull request #${prNumber} not found in ${owner}/${repo}`);
  }

  return pr;
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
  const { data: comment } = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: prNumber,
    body: message,
  });

  return comment;
}

/**
 * Close a GitHub Pull Request using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @returns {Promise<{number: number, html_url: string, title: string}>} Pull request details
 */
async function closePullRequest(github, owner, repo, prNumber) {
  const { data: pr } = await github.rest.pulls.update({
    owner,
    repo,
    pull_number: prNumber,
    state: "closed",
  });

  return pr;
}

async function main() {
  return processCloseEntityItems(PULL_REQUEST_CONFIG, {
    getDetails: getPullRequestDetails,
    addComment: addPullRequestComment,
    closeEntity: closePullRequest,
  });
}

await main();
