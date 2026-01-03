// @ts-check
/// <reference types="@actions/github-script" />

const { processCloseEntityItems, PULL_REQUEST_CONFIG } = require("./close_entity_helpers.cjs");

/**
 * @typedef {import('./safe_output_handler_manager.cjs').HandlerFactoryFunction} HandlerFactoryFunction
 */

const HANDLER_TYPE = "close_pull_request";

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

/**
 * Handler factory for close-pull-request safe outputs
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  /**
   * Message handler for close-pull-request
   * @param {any} message - Close PR message from agent output
   * @param {Object<string, number>} resolvedTemporaryIds - Map of temporary IDs to resolved IDs
   * @returns {Promise<{success: boolean, pull_request_number?: number, pull_request_url?: string, error?: string}>}
   */
  async function handleClosePullRequest(message, resolvedTemporaryIds) {
    // Extract handler config from config parameter
    const handlerConfig = {
      required_labels: config.required_labels || [],
      required_title_prefix: config.required_title_prefix || "",
      target: config.target || "triggering",
      max: config.max || 1,
    };

    // Use the shared helper with PR-specific callbacks
    const result = await processCloseEntityItems(
      PULL_REQUEST_CONFIG,
      {
        getDetails: getPullRequestDetails,
        addComment: addPullRequestComment,
        closeEntity: closePullRequest,
      },
      handlerConfig
    );

    // Convert result to handler format
    if (!result || !result.success) {
      return {
        success: false,
        error: result?.error || "Failed to close pull request",
      };
    }

    return {
      success: true,
      pull_request_number: result.number,
      pull_request_url: result.html_url,
    };
  }

  return handleClosePullRequest;
}

module.exports = { main };
