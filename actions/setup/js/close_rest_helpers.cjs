// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared REST wrappers used by the close-entity flows (close_issue, close_pull_request,
 * close_older_*, close_expired_*). These functions own the raw Octokit calls so the
 * individual handlers only deal with entity-specific behavior (logging, sanitization,
 * result shaping).
 */

const { ERR_NOT_FOUND } = require("./error_codes.cjs");

/**
 * Get issue details using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @returns {Promise<any>} Issue details
 */
async function getIssueDetails(github, owner, repo, issueNumber) {
  const { data: issue } = await github.rest.issues.get({
    owner,
    repo,
    issue_number: issueNumber,
  });

  if (!issue) {
    throw new Error(`${ERR_NOT_FOUND}: Issue #${issueNumber} not found in ${owner}/${repo}`);
  }

  return issue;
}

/**
 * Get pull request details using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @returns {Promise<any>} Pull request details
 */
async function getPullRequestDetails(github, owner, repo, prNumber) {
  const { data: pr } = await github.rest.pulls.get({
    owner,
    repo,
    pull_number: prNumber,
  });

  if (!pr) {
    throw new Error(`${ERR_NOT_FOUND}: Pull request #${prNumber} not found in ${owner}/${repo}`);
  }

  return pr;
}

/**
 * Add a comment to an issue thread (issues and pull requests share this endpoint)
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue or pull request number
 * @param {string} body - Comment body (callers are responsible for sanitization)
 * @returns {Promise<any>} Comment details
 */
async function addIssueThreadComment(github, owner, repo, issueNumber, body) {
  const { data: comment } = await github.rest.issues.createComment({
    owner,
    repo,
    issue_number: issueNumber,
    body,
  });

  return comment;
}

/**
 * Close a GitHub Issue using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} issueNumber - Issue number
 * @param {string} [stateReason] - Close reason: "completed", "not_planned" or "duplicate"
 * @returns {Promise<any>} Issue details
 */
async function closeIssue(github, owner, repo, issueNumber, stateReason = "not_planned") {
  const { data: issue } = await github.rest.issues.update({
    owner,
    repo,
    issue_number: issueNumber,
    state: "closed",
    state_reason: stateReason,
  });

  return issue;
}

/**
 * Close a GitHub Pull Request using REST API
 * @param {any} github - GitHub REST API instance
 * @param {string} owner - Repository owner
 * @param {string} repo - Repository name
 * @param {number} prNumber - Pull request number
 * @returns {Promise<any>} Pull request details
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

module.exports = {
  getIssueDetails,
  getPullRequestDetails,
  addIssueThreadComment,
  closeIssue,
  closePullRequest,
};
