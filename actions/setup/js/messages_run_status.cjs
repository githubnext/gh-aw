// @ts-check

/**
 * Run Status Message Module
 *
 * This module provides run status messages (started, success, failure)
 * for workflow execution notifications.
 */

const { getMessages, renderTemplate, toSnakeCase } = require("./messages_core.cjs");

/**
 * Renders a message using a custom template from config or a default template.
 * @param {string} messageKey - Key in the messages config (e.g., "runStarted")
 * @param {string} defaultTemplate - Default template string with {placeholder} syntax
 * @param {Object} ctx - Context object for template substitution
 * @returns {string} Rendered message
 */
function renderConfiguredMessage(messageKey, defaultTemplate, ctx) {
  const messages = getMessages();
  const template = messages?.[messageKey] ?? defaultTemplate;
  return renderTemplate(template, toSnakeCase(ctx));
}

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
  return renderConfiguredMessage("runStarted", "🚀 [{workflow_name}]({run_url}) has started processing this {event_type}", ctx);
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
  return renderConfiguredMessage("runSuccess", "✅ [{workflow_name}]({run_url}) completed successfully!", ctx);
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
  return renderConfiguredMessage("runFailure", "❌ [{workflow_name}]({run_url}) {status}. Please review the logs for details.", ctx);
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
  return renderConfiguredMessage("detectionFailure", "⚠️ Security scanning failed for [{workflow_name}]({run_url}). Review the logs for details.", ctx);
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
  return renderConfiguredMessage("pullRequestCreated", "Pull request created: [#{item_number}]({item_url})", ctx);
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
  return renderConfiguredMessage("issueCreated", "Issue created: [#{item_number}]({item_url})", ctx);
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
  return renderConfiguredMessage("commitPushed", "Commit pushed: [`{short_sha}`]({commit_url})", ctx);
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
