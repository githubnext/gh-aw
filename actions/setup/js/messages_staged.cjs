// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Staged Mode Message Module
 *
 * This module provides staged mode title and description generation
 * for safe-output preview functionality.
 */

const { getMessages, renderTemplate, toSnakeCase } = require("./messages_core.cjs");

/**
 * @typedef {Object} StagedContext
 * @property {string} operation - The operation name (e.g., "Create Issues", "Add Comments")
 */

/**
 * Get the staged mode title, using custom template if configured.
 * @param {StagedContext} ctx - Context for staged title generation
 * @returns {string} Staged mode title
 */
function getStagedTitle(ctx) {
  const messages = getMessages();
  const templateContext = toSnakeCase(ctx);
  const configuredTemplate = typeof messages?.stagedTitle === "string" ? messages.stagedTitle : "";
  return renderTemplate(configuredTemplate || "## 🔍 Preview: {operation}", templateContext);
}

/**
 * Get the staged mode description, using custom template if configured.
 * @param {StagedContext} ctx - Context for staged description generation
 * @returns {string} Staged mode description
 */
function getStagedDescription(ctx) {
  const messages = getMessages();
  const templateContext = toSnakeCase(ctx);
  const configuredTemplate = typeof messages?.stagedDescription === "string" ? messages.stagedDescription : "";
  return renderTemplate(configuredTemplate || "📋 The following operations would be performed if staged mode was disabled:", templateContext);
}

module.exports = {
  getStagedTitle,
  getStagedDescription,
};
