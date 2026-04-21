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
  return renderTemplate(messages?.stagedTitle ?? "## 🔍 Preview: {operation}", templateContext);
}

/**
 * Get the staged mode description, using custom template if configured.
 * @param {StagedContext} ctx - Context for staged description generation
 * @returns {string} Staged mode description
 */
function getStagedDescription(ctx) {
  const messages = getMessages();
  const templateContext = toSnakeCase(ctx);
  return renderTemplate(messages?.stagedDescription ?? "📋 The following operations would be performed if staged mode was disabled:", templateContext);
}

module.exports = {
  getStagedTitle,
  getStagedDescription,
};
