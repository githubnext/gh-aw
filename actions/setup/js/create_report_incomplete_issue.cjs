// @ts-check
/// <reference types="@actions/github-script" />

const { buildMissingIssueHandler } = require("./missing_issue_helpers.cjs");
const { getPromptPath } = require("./messages_core.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/** @type {string} Safe output type handled by this module */
const HANDLER_TYPE = "create_report_incomplete_issue";

/**
 * Main handler factory for create_report_incomplete_issue
 * Returns a message handler function that creates or updates a tracking issue
 * when the agent emitted report_incomplete signals, aggregating all reasons
 * into a single issue comment per workflow run.
 * @type {HandlerFactoryFunction}
 */
const main = buildMissingIssueHandler({
  handlerType: HANDLER_TYPE,
  defaultTitlePrefix: "[incomplete]",
  defaultLabels: ["agentic-workflows"],
  itemsField: "incomplete_signals",
  templatePath: getPromptPath("missing_tool_issue.md"),
  templateListKey: "incomplete_signals_list",
  buildCommentHeader: runUrl => [`## Incomplete Run Reported`, ``, `The agent reported that the task could not be completed during [workflow run](${runUrl}):`, ``],
  renderCommentItem: (item, index) => {
    const lines = [`### ${index + 1}. Incomplete signal`, `**Reason:** ${item.reason}`];
    if (item.details) lines.push(`**Details:** ${item.details}`);
    lines.push(``);
    return lines;
  },
  renderIssueItem: (item, index) => {
    const lines = [`#### ${index + 1}. Incomplete signal`, `**Reason:** ${item.reason}`];
    if (item.details) lines.push(`**Details:** ${item.details}`);
    lines.push(`**Reported at:** ${item.timestamp}`, ``);
    return lines;
  },
});

module.exports = { main };
