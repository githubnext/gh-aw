// @ts-check

const { renderReport } = require("./report_ast.cjs");

/** @typedef {"agent-output" | "runtime-failure" | "tracking" | "maintenance"} IssueClusteringKind */

const GUIDANCE = {
  "agent-output": "Cluster by the underlying finding and required fix, not the generating workflow, engine, model, run, date, or title prefix.",
  "runtime-failure":
    "Operational workflow failure, not an agent finding about the target project. Cluster across workflows only when diagnostics demonstrate a shared cause and required fix; matching categories or repeated titles alone do not prove duplicates.",
  tracking: "Bookkeeping or tracking container, not an independent finding. Inspect the linked work or child issues before clustering; do not count this container as another instance of their findings.",
  maintenance: "Workflow maintenance request. Cluster by the required maintenance operation and affected component, not the generating workflow or report date.",
};

/**
 * Advisory metadata only: categories describe symptoms, not proven root causes.
 * @param {IssueClusteringKind} kind
 * @param {string[]} [categories]
 * @returns {import("./report_ast.cjs").ReportJSONMarker}
 */
function buildIssueClusteringHint(kind, categories = []) {
  if (!Object.hasOwn(GUIDANCE, kind)) throw new TypeError(`Unknown issue clustering kind: ${kind}`);
  return {
    type: "json_marker",
    name: "gh-aw-clustering-hint",
    agentTainted: false,
    data: { version: 1, kind, categories: [...new Set(categories)].sort(), guidance: GUIDANCE[kind] },
  };
}

/**
 * @param {IssueClusteringKind} kind
 * @param {string[]} [categories]
 * @returns {string}
 */
function generateIssueClusteringHint(kind, categories = []) {
  return renderReport({ type: "report", title: "Issue clustering hint", children: [buildIssueClusteringHint(kind, categories)] }).body.trimEnd();
}

module.exports = { buildIssueClusteringHint, generateIssueClusteringHint };
