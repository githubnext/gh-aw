// @ts-check
/// <reference types="@actions/github-script" />

const { renderReport } = require("./report_ast.cjs");
const { parseReportMarkdown } = require("./report_markdown.cjs");
const { ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * Import existing issue Markdown at the publication boundary. Metadata is opt-in
 * here, not in diagnostic fragments; it remains typed and cannot close comments.
 * @param {string} title
 * @param {string} body
 * @param {{allowedMentions?: string[]}} [options]
 * @returns {import("./report_ast.cjs").Report}
 */
function buildIssueReport(title, body, { allowedMentions = [] } = {}) {
  return { type: "report", title, children: parseReportMarkdown(body, { minimumHeadingLevel: 2, allowMetadata: true, allowedMentions }) };
}

/**
 * @template {{title: string, body?: string}} T
 * @param {T} parameters
 * @param {import("./report_ast.cjs").Report} [report]
 * @returns {T & {title: string, body: string}}
 */
function renderIssueParameters(parameters, report = buildIssueReport(parameters.title, parameters.body ?? "")) {
  const rendered = renderReport(report);
  if (rendered.body.length > 65536) throw new Error(`${ERR_VALIDATION}: Issue body exceeds GitHub's maximum length of 65536 characters including attribution`);
  return { ...parameters, ...rendered };
}

/**
 * Keep authentication, retries, and API responses at the caller's existing boundary.
 * @param {typeof github} client
 * @param {NonNullable<Parameters<typeof github.rest.issues.create>[0]> & {title: string, body?: string}} parameters
 * @param {import("./report_ast.cjs").Report} [report]
 */
function createReportIssue(client, parameters, report) {
  return client.rest.issues.create(renderIssueParameters(parameters, report));
}

module.exports = { buildIssueReport, renderIssueParameters, createReportIssue };
