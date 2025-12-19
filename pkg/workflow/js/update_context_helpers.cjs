// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared context helper functions for update workflows (issues, pull requests, etc.)
 *
 * This module provides reusable functions for determining if we're in a valid
 * context for updating a specific entity type and extracting entity numbers
 * from GitHub event payloads.
 *
 * @module update_context_helpers
 */

/**
 * Check if the current context is a valid issue context
 * @param {string} eventName - GitHub event name
 * @param {any} _payload - GitHub event payload (unused but kept for interface consistency)
 * @returns {boolean} Whether context is valid for issue updates
 */
function isIssueContext(eventName, _payload) {
  return eventName === "issues" || eventName === "issue_comment";
}

/**
 * Get issue number from the context payload
 * @param {any} payload - GitHub event payload
 * @returns {number|undefined} Issue number or undefined
 */
function getIssueNumber(payload) {
  return payload?.issue?.number;
}

/**
 * Check if the current context is a valid pull request context
 * @param {string} eventName - GitHub event name
 * @param {any} payload - GitHub event payload
 * @returns {boolean} Whether context is valid for PR updates
 */
function isPRContext(eventName, payload) {
  const isPR = eventName === "pull_request" || eventName === "pull_request_review" || eventName === "pull_request_review_comment" || eventName === "pull_request_target";

  // Also check for issue_comment on a PR
  const isIssueCommentOnPR = eventName === "issue_comment" && payload?.issue && payload?.issue?.pull_request;

  return isPR || !!isIssueCommentOnPR;
}

/**
 * Get pull request number from the context payload
 * @param {any} payload - GitHub event payload
 * @returns {number|undefined} PR number or undefined
 */
function getPRNumber(payload) {
  if (payload?.pull_request) {
    return payload.pull_request.number;
  }
  // For issue_comment events on PRs, the PR number is in issue.number
  if (payload?.issue && payload?.issue?.pull_request) {
    return payload.issue.number;
  }
  return undefined;
}

/**
 * Check if the current context is a valid discussion context
 * @param {string} eventName - GitHub event name
 * @param {any} _payload - GitHub event payload (unused but kept for interface consistency)
 * @returns {boolean} Whether context is valid for discussion updates
 */
function isDiscussionContext(eventName, _payload) {
  return eventName === "discussion" || eventName === "discussion_comment";
}

/**
 * Get discussion number from the context payload
 * @param {any} payload - GitHub event payload
 * @returns {number|undefined} Discussion number or undefined
 */
function getDiscussionNumber(payload) {
  return payload?.discussion?.number;
}

module.exports = {
  isIssueContext,
  getIssueNumber,
  isPRContext,
  getPRNumber,
  isDiscussionContext,
  getDiscussionNumber,
};
