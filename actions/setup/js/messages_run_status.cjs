// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Run Status Message Module
 *
 * This module provides run status messages (started, success, failure)
 * for workflow execution notifications.
 */

const { getMessages, renderTemplate, toSnakeCase } = require("./messages_core.cjs");

/**
 * @typedef {Object} RunStartedContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} eventType - Event type description (e.g., "issue", "pull request", "discussion")
 */

/**
 * Get the run-started message, using custom template if configured.
 * @param {RunStartedContext} ctx - Context for run-started message generation
 * @returns {string} Run-started message
 */
function getRunStartedMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default run-started template
  const defaultMessage = "🚀 [{workflow_name}]({run_url}) has started processing this {event_type}";

  // Use custom message if configured
  return messages?.runStarted ? renderTemplate(messages.runStarted, templateContext) : renderTemplate(defaultMessage, templateContext);
}

/**
 * @typedef {Object} RunSuccessContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 */

/**
 * Get the run-success message, using custom template if configured.
 * @param {RunSuccessContext} ctx - Context for run-success message generation
 * @returns {string} Run-success message
 */
function getRunSuccessMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default run-success template
  const defaultMessage = "✅ [{workflow_name}]({run_url}) completed successfully!";

  // Use custom message if configured
  return messages?.runSuccess ? renderTemplate(messages.runSuccess, templateContext) : renderTemplate(defaultMessage, templateContext);
}

/**
 * @typedef {Object} RunFailureContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} status - Status text (e.g., "failed", "was cancelled", "timed out")
 */

/**
 * Get the run-failure message, using custom template if configured.
 * @param {RunFailureContext} ctx - Context for run-failure message generation
 * @returns {string} Run-failure message
 */
function getRunFailureMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default run-failure template
  const defaultMessage = "❌ [{workflow_name}]({run_url}) {status}. Please review the logs for details.";

  // Use custom message if configured
  return messages?.runFailure ? renderTemplate(messages.runFailure, templateContext) : renderTemplate(defaultMessage, templateContext);
}

/**
 * @typedef {Object} DetectionFailureContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 */

/**
 * Get the detection-failure message, using custom template if configured.
 * @param {DetectionFailureContext} ctx - Context for detection-failure message generation
 * @returns {string} Detection-failure message
 */
function getDetectionFailureMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default detection-failure template
  const defaultMessage = "⚠️ Security scanning failed for [{workflow_name}]({run_url}). Review the logs for details.";

  // Use custom message if configured
  return messages?.detectionFailure ? renderTemplate(messages.detectionFailure, templateContext) : renderTemplate(defaultMessage, templateContext);
}

/**
 * @typedef {Object} PullRequestCreatedContext
 * @property {number} itemNumber - PR number
 * @property {string} itemUrl - URL of the pull request
 */

/**
 * Get the pull-request-created message, using custom template if configured.
 * @param {PullRequestCreatedContext} ctx - Context for message generation
 * @returns {string} Pull-request-created message
 */
function getPullRequestCreatedMessage(ctx) {
  const messages = getMessages();
  const templateContext = toSnakeCase(ctx);
  const defaultMessage = "Pull request created: [#{item_number}]({item_url})";
  return messages?.pullRequestCreated ? renderTemplate(messages.pullRequestCreated, templateContext) : renderTemplate(defaultMessage, templateContext);
}

/**
 * @typedef {Object} IssueCreatedContext
 * @property {number} itemNumber - Issue number
 * @property {string} itemUrl - URL of the issue
 */

/**
 * Get the issue-created message, using custom template if configured.
 * @param {IssueCreatedContext} ctx - Context for message generation
 * @returns {string} Issue-created message
 */
function getIssueCreatedMessage(ctx) {
  const messages = getMessages();
  const templateContext = toSnakeCase(ctx);
  const defaultMessage = "Issue created: [#{item_number}]({item_url})";
  return messages?.issueCreated ? renderTemplate(messages.issueCreated, templateContext) : renderTemplate(defaultMessage, templateContext);
}

/**
 * @typedef {Object} CommitPushedContext
 * @property {string} commitSha - Full commit SHA
 * @property {string} shortSha - Short (7-char) commit SHA
 * @property {string} commitUrl - URL of the commit
 */

/**
 * Get the commit-pushed message, using custom template if configured.
 * @param {CommitPushedContext} ctx - Context for message generation
 * @returns {string} Commit-pushed message
 */
function getCommitPushedMessage(ctx) {
  const messages = getMessages();
  const templateContext = toSnakeCase(ctx);
  const defaultMessage = "Commit pushed: [`{short_sha}`]({commit_url})";
  return messages?.commitPushed ? renderTemplate(messages.commitPushed, templateContext) : renderTemplate(defaultMessage, templateContext);
}

module.exports = {
  getRunStartedMessage,
  getRunSuccessMessage,
  getRunFailureMessage,
  getDetectionFailureMessage,
  getPullRequestCreatedMessage,
  getIssueCreatedMessage,
  getCommitPushedMessage,
};
