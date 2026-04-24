// @ts-check
/// <reference types="@actions/github-script" />

const { buildMissingIssueHandler } = require("./missing_issue_helpers.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/**
 * @typedef {Object} MissingIssueHandlerDescriptor
 * @property {string} handlerType - Handler type identifier used in log/warning messages
 * @property {string} defaultTitlePrefix - Default issue title prefix (e.g. "[missing data]")
 * @property {string[]} defaultLabels - Labels always applied to created issues
 * @property {string} itemsField - Field name in the message containing the items array
 * @property {string} templatePath - Absolute path to the issue body template file
 * @property {string} templateListKey - Template variable name for the rendered items list
 * @property {function(string): string[]} buildCommentHeader - Returns header lines given runUrl
 * @property {function(Object, number): string[]} renderCommentItem - Renders a single item for a comment
 * @property {function(Object, number): string[]} renderIssueItem - Renders a single item for a new issue body
 */

/** @type {MissingIssueHandlerDescriptor[]} */
const HANDLER_DESCRIPTORS = [
  {
    handlerType: "create_missing_data_issue",
    defaultTitlePrefix: "[missing data]",
    defaultLabels: ["agentic-workflows"],
    itemsField: "missing_data",
    templatePath: `${process.env.RUNNER_TEMP}/gh-aw/prompts/missing_data_issue.md`,
    templateListKey: "missing_data_list",
    buildCommentHeader: runUrl => [`## Missing Data Reported`, ``, `The following data was reported as missing during [workflow run](${runUrl}):`, ``],
    renderCommentItem: (item, index) => {
      const lines = [`### ${index + 1}. **${item.data_type}**`, `**Reason:** ${item.reason}`];
      if (item.context) lines.push(`**Context:** ${item.context}`);
      if (item.alternatives) lines.push(`**Alternatives:** ${item.alternatives}`);
      lines.push(``);
      return lines;
    },
    renderIssueItem: (item, index) => {
      const lines = [`#### ${index + 1}. **${item.data_type}**`, `**Reason:** ${item.reason}`];
      if (item.context) lines.push(`**Context:** ${item.context}`);
      if (item.alternatives) lines.push(`**Alternatives:** ${item.alternatives}`);
      lines.push(`**Reported at:** ${item.timestamp}`, ``);
      return lines;
    },
  },
  {
    handlerType: "create_missing_tool_issue",
    defaultTitlePrefix: "[missing tool]",
    defaultLabels: ["agentic-workflows"],
    itemsField: "missing_tools",
    templatePath: `${process.env.RUNNER_TEMP}/gh-aw/prompts/missing_tool_issue.md`,
    templateListKey: "missing_tools_list",
    buildCommentHeader: runUrl => [`## Missing Tools Reported`, ``, `The following tools were reported as missing during [workflow run](${runUrl}):`, ``],
    renderCommentItem: (tool, index) => {
      const lines = [`### ${index + 1}. \`${tool.tool}\``, `**Reason:** ${tool.reason}`];
      if (tool.alternatives) lines.push(`**Alternatives:** ${tool.alternatives}`);
      lines.push(``);
      return lines;
    },
    renderIssueItem: (tool, index) => {
      const lines = [`#### ${index + 1}. \`${tool.tool}\``, `**Reason:** ${tool.reason}`];
      if (tool.alternatives) lines.push(`**Alternatives:** ${tool.alternatives}`);
      lines.push(`**Reported at:** ${tool.timestamp}`, ``);
      return lines;
    },
  },
  {
    handlerType: "create_report_incomplete_issue",
    defaultTitlePrefix: "[incomplete]",
    defaultLabels: ["agentic-workflows"],
    itemsField: "incomplete_signals",
    templatePath: `${process.env.RUNNER_TEMP}/gh-aw/prompts/missing_tool_issue.md`,
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
  },
];

/**
 * Registry mapping handler type identifiers to their factory functions.
 * Generated from HANDLER_DESCRIPTORS via a single shared factory loop.
 * @type {Map<string, HandlerFactoryFunction>}
 */
const handlerRegistry = new Map(HANDLER_DESCRIPTORS.map(desc => [desc.handlerType, buildMissingIssueHandler(desc)]));

module.exports = { HANDLER_DESCRIPTORS, handlerRegistry };
