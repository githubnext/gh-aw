// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Output Messages Module (Barrel File)
 *
 * This module re-exports all message functions from the modular message files.
 * It provides backward compatibility for existing code that imports from messages.cjs.
 *
 * For new code, prefer importing directly from the specific modules:
 * - ./messages_core.cjs - Core utilities (getMessages, renderTemplate, toSnakeCase)
 * - ./messages_footer.cjs - Footer messages (getFooterMessage, getFooterInstallMessage, generateFooterWithMessages)
 * - ./messages_staged.cjs - Staged mode messages (getStagedTitle, getStagedDescription)
 * - ./messages_run_status.cjs - Run status messages (getRunStartedMessage, getRunSuccessMessage, getRunFailureMessage)
 * - ./messages_close_discussion.cjs - Close discussion messages (getCloseOlderDiscussionMessage)
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

// Re-export core utilities
const { getMessages, renderTemplate } = require("./messages_core.cjs");

// Re-export footer messages
const { getFooterMessage, getFooterInstallMessage, getFooterAgentFailureIssueMessage, getFooterAgentFailureCommentMessage, generateFooterWithMessages, generateXMLMarker } = require("./messages_footer.cjs");

// Re-export staged mode messages
const { getStagedTitle, getStagedDescription } = require("./messages_staged.cjs");

// Re-export run status messages
const { getRunStartedMessage, getRunSuccessMessage, getRunFailureMessage } = require("./messages_run_status.cjs");

// Re-export close discussion messages
const { getCloseOlderDiscussionMessage } = require("./messages_close_discussion.cjs");

module.exports = {
  getMessages,
  renderTemplate,
  getFooterMessage,
  getFooterInstallMessage,
  getFooterAgentFailureIssueMessage,
  getFooterAgentFailureCommentMessage,
  generateFooterWithMessages,
  generateXMLMarker,
  getStagedTitle,
  getStagedDescription,
  getRunStartedMessage,
  getRunSuccessMessage,
  getRunFailureMessage,
  getCloseOlderDiscussionMessage,
};
