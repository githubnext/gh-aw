// @ts-check

const { parseReportMarkdown } = require("./report_markdown.cjs");

/** @typedef {import("./report_ast.cjs").Report} Report */
/** @typedef {import("./report_ast.cjs").ReportInline} ReportInline */
/** @typedef {import("./report_ast.cjs").ReportBlock} ReportBlock */

/**
 * Keep legacy diagnostic templates at an explicit import boundary. Scalar metadata
 * stays literal, and diagnostic headings cannot become top-level report sections.
 * @param {string} template
 * @param {string} title
 * @param {Record<string, string>} values
 * @returns {Report}
 */
function buildFailureReport(template, title, values) {
  /** @type {import("./report_markdown.cjs").ReportContext} */
  const context = {};
  for (const [key, value] of Object.entries(values)) {
    context[key] = key.endsWith("_context") ? parseReportMarkdown(value, { minimumHeadingLevel: 4, agentTainted: true }) : value;
  }

  /** @type {ReportInline[]} */
  const links = [];
  const addLink = (label, url) => {
    if (links.length) links.push({ type: "text", value: " · ", agentTainted: false });
    links.push({ type: "link", url, children: [{ type: "text", value: label, agentTainted: false }], agentTainted: false });
  };
  if (values.run_url) addLink(/^\d+$/.test(values.run_id) ? `View run #${values.run_id}` : "View run", values.run_url);
  else links.push({ type: "text", value: "Run link unavailable: run URL was not recorded.", agentTainted: false });
  if (values.workflow_source_url && values.workflow_source_url !== "#") addLink("Workflow source", values.workflow_source_url);
  if (values.history_url) addLink("Previous failures", values.history_url);
  context.run_links_context = [{ type: "paragraph", children: links, agentTainted: false }];
  context.pull_request_context = parseReportMarkdown(values.pull_request_info || "", { agentTainted: false });

  const evidence = [context.report_incomplete_context, context.failure_diagnostics_context].flatMap(value => (Array.isArray(value) ? value : []));
  context.runtime_evidence_context = evidence.length ? [{ type: "details", summary: "Runtime evidence", children: evidence, agentTainted: false }] : [];

  return { type: "report", title, children: parseReportMarkdown(template, { context, minimumHeadingLevel: 3, agentTainted: false }), agentTainted: false };
}

module.exports = { buildFailureReport };
