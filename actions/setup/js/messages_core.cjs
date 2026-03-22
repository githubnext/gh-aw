// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Core Message Utilities Module
 *
 * This module provides shared utilities for message template processing.
 * It includes configuration parsing and template rendering functions.
 *
 * Supported placeholders:
 * - {workflow_name} - Name of the workflow
 * - {run_url} - URL to the workflow run
 * - {workflow_source} - Source specification (owner/repo/path@ref)
 * - {workflow_source_url} - GitHub URL for the workflow source
 * - {triggering_number} - Issue/PR/Discussion number that triggered this workflow
 * - {operation} - Operation name (for staged mode titles/descriptions)
 * - {event_type} - Event type description (for run-started messages)
 * - {status} - Workflow status text (for run-failure messages)
 * - {repository} - Repository name (for workflow recompile messages)
 *
 * Both camelCase and snake_case placeholder formats are supported.
 */

const { getErrorMessage } = require("./error_helpers.cjs");
const fs = require("fs");

/**
 * @typedef {Object} SafeOutputMessages
 * @property {string} [footer] - Custom footer message template
 * @property {string} [footerInstall] - Custom installation instructions template
 * @property {string} [footerWorkflowRecompile] - Custom footer template for workflow recompile issues
 * @property {string} [footerWorkflowRecompileComment] - Custom footer template for comments on workflow recompile issues
 * @property {string} [stagedTitle] - Custom staged mode title template
 * @property {string} [stagedDescription] - Custom staged mode description template
 * @property {string} [runStarted] - Custom workflow activation message template
 * @property {string} [runSuccess] - Custom workflow success message template
 * @property {string} [runFailure] - Custom workflow failure message template
 * @property {string} [detectionFailure] - Custom detection job failure message template
 * @property {string} [pullRequestCreated] - Custom template for pull request creation link. Placeholders: {item_number}, {item_url}
 * @property {string} [issueCreated] - Custom template for issue creation link. Placeholders: {item_number}, {item_url}
 * @property {string} [commitPushed] - Custom template for commit push link. Placeholders: {commit_sha}, {short_sha}, {commit_url}
 * @property {string} [agentFailureIssue] - Custom footer template for agent failure tracking issues
 * @property {string} [agentFailureComment] - Custom footer template for comments on agent failure tracking issues
 * @property {string} [closeOlderDiscussion] - Custom message for closing older discussions as outdated
 * @property {boolean} [appendOnlyComments] - If true, create new comments instead of updating the activation comment
 * @property {string|boolean} [activationComments] - If false or "false", disable all activation/fallback comments entirely. Supports templatable boolean values (default: true)
 */

/**
 * Get the safe-output messages configuration from environment variable.
 * @returns {SafeOutputMessages|null} Parsed messages config or null if not set
 */
function getMessages() {
  const messagesEnv = process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
  if (!messagesEnv) {
    return null;
  }

  try {
    // Parse JSON with camelCase keys from Go struct (using json struct tags)
    return JSON.parse(messagesEnv);
  } catch (error) {
    core.warning(`Failed to parse GH_AW_SAFE_OUTPUT_MESSAGES: ${getErrorMessage(error)}`);
    return null;
  }
}

/**
 * Replace placeholders in a template string with values from context.
 * Supports {key} syntax for placeholder replacement.
 * @param {string} template - Template string with {key} placeholders
 * @param {Record<string, string|number|boolean|undefined>} context - Key-value pairs for replacement
 * @returns {string} Template with placeholders replaced
 */
function renderTemplate(template, context) {
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    const value = context[key];
    return value !== undefined && value !== null ? String(value) : match;
  });
}

/**
 * Read a template file and render it with the given context.
 * Combines file loading and template rendering into a single helper.
 * @param {string} templatePath - Absolute path to the template file
 * @param {Record<string, string|number|boolean|undefined>} context - Key-value pairs for replacement
 * @returns {string} Rendered template with placeholders replaced
 */
function renderTemplateFromFile(templatePath, context) {
  const template = fs.readFileSync(templatePath, "utf8");
  return renderTemplate(template, context);
}

/**
 * Convert context object keys to snake_case for template rendering.
 * Also keeps original camelCase keys for backwards compatibility.
 * @param {Record<string, any>} obj - Object with camelCase keys
 * @returns {Record<string, any>} Object with both snake_case and original keys
 */
function toSnakeCase(obj) {
  return Object.fromEntries(
    Object.entries(obj).flatMap(([key, value]) => {
      const snakeKey = key.replace(/([A-Z])/g, "_$1").toLowerCase();
      return snakeKey === key
        ? [[key, value]]
        : [
            [snakeKey, value],
            [key, value],
          ];
    })
  );
}

module.exports = {
  getMessages,
  renderTemplate,
  renderTemplateFromFile,
  toSnakeCase,
};
