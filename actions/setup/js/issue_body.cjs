// @ts-check
/// <reference types="@actions/github-script" />

const { buildGeneratedFooterParts, getBodyFooterMessage, getDetectionCautionAlert } = require("./messages_footer.cjs");
const { getBodyHeader, getDisclosureHeader } = require("./messages_header.cjs");
const { generateWorkflowIdMarker, generateWorkflowCallIdMarker, generateCloseKeyMarker } = require("./generate_footer.cjs");
const { generateHistoryUrl } = require("./generate_history_link.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { addExpirationToFooter } = require("./ephemerals.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { appendExecutionProvenance, stripExecutionProvenance } = require("./execution_provenance.cjs");
const { parseReportMarkdown } = require("./report_markdown.cjs");
const { renderReport } = require("./report_ast.cjs");

/**
 * @typedef {{repo: {owner: string, repo: string}, footer?: boolean, bodyFooter?: string, expiresHours?: number, closeOlderKey?: string, allowedMentions?: string[]}} IssueBodyOptions
 */

/**
 * Decorate a trusted issue body with the standard safe-output attribution.
 * Content sanitization belongs to the caller so custom metadata is preserved.
 * @param {string} body
 * @param {IssueBodyOptions} options
 * @returns {string}
 */
function formatIssueBody(body, options) {
  return renderReport(buildIssueBodyReport("Issue", body, options)).body.trimEnd();
}

/**
 * Assemble content, attribution, and provenance without rendering intermediate trees.
 * @param {string} title
 * @param {string} body
 * @param {IssueBodyOptions} options
 * @returns {import("./report_ast.cjs").Report}
 */
function buildIssueBodyReport(title, body, { repo, footer = true, bodyFooter, expiresHours = 0, closeOlderKey, allowedMentions = [] }) {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME ?? "Workflow";
  const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE ?? "";
  const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL ?? "";
  const workflowId = process.env.GH_AW_WORKFLOW_ID ?? "";
  const callerWorkflowId = process.env.GH_AW_CALLER_WORKFLOW_ID ?? "";
  const runUrl = buildWorkflowRunUrl(context, context.repo);
  /** @type {import("./report_ast.cjs").ReportBlock[]} */
  const children = [];
  const parse = markdown => parseReportMarkdown(stripExecutionProvenance(markdown), { minimumHeadingLevel: 2, allowMetadata: true, allowedMentions });

  const bodyHeader = getBodyHeader({ workflowName, runUrl });
  const disclosureHeader = getDisclosureHeader({ workflowName, runUrl });
  const detectionCaution = getDetectionCautionAlert(workflowName, runUrl);
  const preamble = parse([detectionCaution, disclosureHeader, bodyHeader].filter(Boolean).join("\n\n"));
  children.push(...parse(body));

  const trackerIDComment = getTrackerID("markdown");
  if (trackerIDComment) children.push(...parse(trackerIDComment));

  if (footer) {
    const triggeringIssueNumber = context.payload?.issue?.number && !context.payload?.issue?.pull_request ? context.payload.issue.number : undefined;
    const triggeringPRNumber = context.payload?.pull_request?.number || (context.payload?.issue?.pull_request ? context.payload.issue.number : undefined);
    const triggeringDiscussionNumber = context.payload?.discussion?.number;
    const historyUrl = generateHistoryUrl({
      owner: repo.owner,
      repo: repo.repo,
      itemType: "issue",
      workflowCallId: callerWorkflowId,
      workflowId,
      serverUrl: context.serverUrl,
    });
    const parts = buildGeneratedFooterParts(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber, historyUrl, { skipDetectionCaution: true });
    // Expiration belongs before the final metadata marker, even with Claim nodes.
    const suffix = parts.pop();
    if (typeof suffix !== "string") throw new TypeError("Generated footer must end with Markdown metadata");
    parts.push(addExpirationToFooter(suffix.trimEnd(), expiresHours, "Issue"));
    let markdown = "";
    for (const part of parts) {
      if (typeof part === "string") markdown += part;
      else {
        children.push(...parse(markdown), part);
        markdown = "";
      }
    }
    children.push(...parse(markdown));
  }

  const configuredFooter = getBodyFooterMessage(bodyFooter, { workflowName, runUrl });
  if (configuredFooter) children.push(...parse(configuredFooter));
  if (workflowId) children.push(...parse(generateWorkflowIdMarker(workflowId)));
  if (callerWorkflowId) children.push(...parse(generateWorkflowCallIdMarker(callerWorkflowId)));
  if (closeOlderKey) children.push(...parse(generateCloseKeyMarker(closeOlderKey)));

  children.push(...parseReportMarkdown(appendExecutionProvenance("", workflowName, runUrl), { minimumHeadingLevel: 2, allowMetadata: true }));
  return { type: "report", title, preamble, children };
}

module.exports = { formatIssueBody, buildIssueBodyReport };
