// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Output Messages Module
 *
 * This module provides customizable message templates for safe-output workflows.
 * It reads configuration from the GH_AW_SAFE_OUTPUT_MESSAGES environment variable
 * and renders templates with placeholder support.
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
 *
 * Both camelCase and snake_case placeholder formats are supported.
 */

/**
 * @typedef {Object} SafeOutputMessages
 * @property {string} [footer] - Custom footer message template
 * @property {string} [footerInstall] - Custom installation instructions template
 * @property {string} [stagedTitle] - Custom staged mode title template
 * @property {string} [stagedDescription] - Custom staged mode description template
 * @property {string} [runStarted] - Custom workflow activation message template
 * @property {string} [runSuccess] - Custom workflow success message template
 * @property {string} [runFailure] - Custom workflow failure message template
 */

/**
 * @typedef {Object} FooterContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} [workflowSource] - Source of the workflow (owner/repo/path@ref)
 * @property {string} [workflowSourceUrl] - GitHub URL for the workflow source
 * @property {number|string} [triggeringNumber] - Issue, PR, or discussion number that triggered this workflow
 */

/**
 * @typedef {Object} StagedContext
 * @property {string} operation - The operation name (e.g., "Create Issues", "Add Comments")
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
    // Parse JSON, converting camelCase keys from Go struct
    const rawMessages = JSON.parse(messagesEnv);

    // Map Go struct field names (PascalCase in JSON) to camelCase
    return {
      footer: rawMessages.Footer || rawMessages.footer,
      footerInstall: rawMessages.FooterInstall || rawMessages.footerInstall,
      stagedTitle: rawMessages.StagedTitle || rawMessages.stagedTitle,
      stagedDescription: rawMessages.StagedDescription || rawMessages.stagedDescription,
      runStarted: rawMessages.RunStarted || rawMessages.runStarted,
      runSuccess: rawMessages.RunSuccess || rawMessages.runSuccess,
      runFailure: rawMessages.RunFailure || rawMessages.runFailure,
    };
  } catch (error) {
    core.warning(`Failed to parse GH_AW_SAFE_OUTPUT_MESSAGES: ${error instanceof Error ? error.message : String(error)}`);
    return null;
  }
}

/**
 * Replace placeholders in a template string with values from context.
 * Supports {key} syntax for placeholder replacement.
 * @param {string} template - Template string with {key} placeholders
 * @param {Record<string, string|number|undefined>} context - Key-value pairs for replacement
 * @returns {string} Template with placeholders replaced
 */
function renderTemplate(template, context) {
  return template.replace(/\{(\w+)\}/g, (match, key) => {
    const value = context[key];
    return value !== undefined && value !== null ? String(value) : match;
  });
}

/**
 * Convert context object keys to snake_case for template rendering
 * @param {Record<string, any>} obj - Object with camelCase keys
 * @returns {Record<string, any>} Object with snake_case keys
 */
function toSnakeCase(obj) {
  /** @type {Record<string, any>} */
  const result = {};
  for (const [key, value] of Object.entries(obj)) {
    // Convert camelCase to snake_case
    const snakeKey = key.replace(/([A-Z])/g, "_$1").toLowerCase();
    result[snakeKey] = value;
    // Also keep original key for backwards compatibility
    result[key] = value;
  }
  return result;
}

/**
 * Get the footer message, using custom template if configured.
 * @param {FooterContext} ctx - Context for footer generation
 * @returns {string} Footer message
 */
function getFooterMessage(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default footer template - pirate themed! 🏴‍☠️
  const defaultFooter = "> 🏴‍☠️ Ahoy! This treasure was crafted by [{workflow_name}]({run_url})";

  // Use custom footer if configured
  let footer = messages?.footer ? renderTemplate(messages.footer, templateContext) : renderTemplate(defaultFooter, templateContext);

  // Add triggering reference if available
  if (ctx.triggeringNumber) {
    footer += ` fer issue #{triggering_number} 🗺️`.replace("{triggering_number}", String(ctx.triggeringNumber));
  }

  return footer;
}

/**
 * Get the footer installation instructions, using custom template if configured.
 * @param {FooterContext} ctx - Context for footer generation
 * @returns {string} Footer installation message or empty string if no source
 */
function getFooterInstallMessage(ctx) {
  if (!ctx.workflowSource || !ctx.workflowSourceUrl) {
    return "";
  }

  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default installation template - pirate themed! 🏴‍☠️
  const defaultInstall =
    "> 🦜 Arr! To plunder this workflow fer yer own ship, run `gh aw add {workflow_source}`. Chart yer course at [{workflow_source_url}]({workflow_source_url})!";

  // Use custom installation message if configured
  return messages?.footerInstall
    ? renderTemplate(messages.footerInstall, templateContext)
    : renderTemplate(defaultInstall, templateContext);
}

/**
 * Generate the complete footer with AI attribution and optional installation instructions.
 * This is a drop-in replacement for the original generateFooter function.
 * @param {string} workflowName - Name of the workflow
 * @param {string} runUrl - URL of the workflow run
 * @param {string} workflowSource - Source of the workflow (owner/repo/path@ref)
 * @param {string} workflowSourceURL - GitHub URL for the workflow source
 * @param {number|undefined} triggeringIssueNumber - Issue number that triggered this workflow
 * @param {number|undefined} triggeringPRNumber - Pull request number that triggered this workflow
 * @param {number|undefined} triggeringDiscussionNumber - Discussion number that triggered this workflow
 * @returns {string} Complete footer text
 */
function generateFooterWithMessages(
  workflowName,
  runUrl,
  workflowSource,
  workflowSourceURL,
  triggeringIssueNumber,
  triggeringPRNumber,
  triggeringDiscussionNumber
) {
  // Determine triggering number (issue takes precedence, then PR, then discussion)
  let triggeringNumber;
  if (triggeringIssueNumber) {
    triggeringNumber = triggeringIssueNumber;
  } else if (triggeringPRNumber) {
    triggeringNumber = triggeringPRNumber;
  } else if (triggeringDiscussionNumber) {
    triggeringNumber = `discussion #${triggeringDiscussionNumber}`;
  }

  const ctx = {
    workflowName,
    runUrl,
    workflowSource,
    workflowSourceUrl: workflowSourceURL,
    triggeringNumber,
  };

  let footer = "\n\n" + getFooterMessage(ctx);

  // Add installation instructions if source is available
  const installMessage = getFooterInstallMessage(ctx);
  if (installMessage) {
    footer += "\n>\n" + installMessage;
  }

  footer += "\n";
  return footer;
}

/**
 * Get the staged mode title, using custom template if configured.
 * @param {StagedContext} ctx - Context for staged title generation
 * @returns {string} Staged mode title
 */
function getStagedTitle(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default staged title template - pirate themed! 🏴‍☠️
  const defaultTitle = "## 🏴‍☠️ Ahoy Matey! Staged Waters: {operation} Preview";

  // Use custom title if configured
  return messages?.stagedTitle ? renderTemplate(messages.stagedTitle, templateContext) : renderTemplate(defaultTitle, templateContext);
}

/**
 * Get the staged mode description, using custom template if configured.
 * @param {StagedContext} ctx - Context for staged description generation
 * @returns {string} Staged mode description
 */
function getStagedDescription(ctx) {
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default staged description template - pirate themed! 🏴‍☠️
  const defaultDescription = "🗺️ Shiver me timbers! The following booty would be plundered if we set sail (staged mode disabled):";

  // Use custom description if configured
  return messages?.stagedDescription
    ? renderTemplate(messages.stagedDescription, templateContext)
    : renderTemplate(defaultDescription, templateContext);
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
  const messages = getMessages();

  // Create context with both camelCase and snake_case keys
  const templateContext = toSnakeCase(ctx);

  // Default run-started template - pirate themed! 🏴‍☠️
  const defaultMessage = "⚓ Avast! [{workflow_name}]({run_url}) be settin' sail on this {event_type}! 🏴‍☠️";

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

  // Default run-success template - pirate themed! 🏴‍☠️
  const defaultMessage = "🎉 Yo ho ho! [{workflow_name}]({run_url}) found the treasure and completed successfully! ⚓💰";

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

  // Default run-failure template - pirate themed! 🏴‍☠️
  const defaultMessage = "💀 Blimey! [{workflow_name}]({run_url}) {status} and walked the plank! No treasure today, matey! ☠️";

  // Use custom message if configured
  return messages?.runFailure ? renderTemplate(messages.runFailure, templateContext) : renderTemplate(defaultMessage, templateContext);
}

module.exports = {
  getMessages,
  renderTemplate,
  getFooterMessage,
  getFooterInstallMessage,
  generateFooterWithMessages,
  getStagedTitle,
  getStagedDescription,
  getRunStartedMessage,
  getRunSuccessMessage,
  getRunFailureMessage,
};
