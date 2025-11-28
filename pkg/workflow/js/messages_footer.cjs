// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Footer Message Module
 *
 * This module provides footer and installation instructions generation
 * for safe-output workflows.
 */

const { getMessages, renderTemplate, toSnakeCase } = require("./messages_core.cjs");

/**
 * @typedef {Object} FooterContext
 * @property {string} workflowName - Name of the workflow
 * @property {string} runUrl - URL of the workflow run
 * @property {string} [workflowSource] - Source of the workflow (owner/repo/path@ref)
 * @property {string} [workflowSourceUrl] - GitHub URL for the workflow source
 * @property {number|string} [triggeringNumber] - Issue, PR, or discussion number that triggered this workflow
 */

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

module.exports = {
  getFooterMessage,
  getFooterInstallMessage,
  generateFooterWithMessages,
};
