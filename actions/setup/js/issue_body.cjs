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
const { normalizeReportText, renderReport } = require("./report_ast.cjs");

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
  const bodyHeader = getBodyHeader({ workflowName, runUrl });
  const disclosureHeader = getDisclosureHeader({ workflowName, runUrl });
  const detectionCaution = getDetectionCautionAlert(workflowName, runUrl);
  const trackerIDComment = getTrackerID("markdown");
  /** @type {import("./messages_footer.cjs").FooterPart[]} */
  const parts = [[detectionCaution, disclosureHeader, bodyHeader, body, trackerIDComment].filter(Boolean).join("\n\n")];

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
    const footerParts = buildGeneratedFooterParts(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, triggeringDiscussionNumber, historyUrl, { skipDetectionCaution: true });
    // Expiration belongs before the final metadata marker, even with Claim nodes.
    const suffix = footerParts.pop();
    if (typeof suffix !== "string") throw new TypeError("Generated footer must end with Markdown metadata");
    footerParts.push(addExpirationToFooter(suffix.trimEnd(), expiresHours, "Issue"));
    parts.push("\n\n", ...footerParts);
  }

  const configuredFooter = getBodyFooterMessage(bodyFooter, { workflowName, runUrl });
  if (configuredFooter) parts.push("\n\n", configuredFooter);
  if (workflowId) parts.push("\n\n", generateWorkflowIdMarker(workflowId));
  if (callerWorkflowId) parts.push("\n\n", generateWorkflowCallIdMarker(callerWorkflowId));
  if (closeOlderKey) parts.push("\n\n", generateCloseKeyMarker(closeOlderKey));

  const literalMarkdown = normalizeReportText(stripExecutionProvenance(parts.filter(part => typeof part === "string").join("")));
  /** @type {Record<string, import("./report_ast.cjs").ReportInline[]>} */
  const inlineSlots = {};
  const markdown = parts
    .map(part => {
      if (typeof part === "string") return part;
      let key = `gh_aw_issue_footer_${Object.keys(inlineSlots).length}`;
      while (literalMarkdown.includes(`{${key}}`)) key += "_";
      inlineSlots[key] = part.children;
      return `> {${key}}`;
    })
    .join("");
  // Parse wrappers and adjacent quote lines together, while Claim values remain typed.
  const children = parseReportMarkdown(stripExecutionProvenance(markdown), { inlineSlots, minimumHeadingLevel: 2, allowMetadata: true, allowedMentions });

  children.push(...parseReportMarkdown(appendExecutionProvenance("", workflowName, runUrl), { minimumHeadingLevel: 2, allowMetadata: true }));
  return { type: "report", title, children };
}

module.exports = { formatIssueBody, buildIssueBodyReport };
