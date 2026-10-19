// @ts-check

const { normalizeReport } = require("./report_ast.cjs");

/** @typedef {import("./report_ast.cjs").Report} Report */
/** @typedef {import("./report_ast.cjs").ReportBlock} ReportBlock */
/** @typedef {import("./report_ast.cjs").ReportInline} ReportInline */
/** @typedef {(node: ReportBlock | ReportInline) => boolean} ReportRedactionPolicy */

/**
 * A coarse replacement must not swallow framework descendants.
 * @param {ReportBlock | ReportInline} node
 * @returns {boolean}
 */
function isAgentSubtree(node) {
  if (node.agentTainted === false) return false;
  if ("children" in node) return node.children.every(isAgentSubtree);
  if (node.type === "list") return node.items.every(item => item.every(isAgentSubtree));
  if (node.type === "table") return [...node.headers, ...node.rows.flat()].every(cell => cell.every(isAgentSubtree));
  return true;
}

/** @returns {ReportInline} */
function inlineRedactionNotice() {
  return { type: "text", value: "Content redacted.", agentTainted: false };
}

/** @returns {ReportBlock} */
function redactionNotice() {
  return { type: "paragraph", agentTainted: false, children: [inlineRedactionNotice()] };
}

/**
 * Replace selected agent-origin blocks as complete subtrees, not string fragments.
 * An inline match replaces its containing agent paragraph, heading, task, or table.
 * Framework and mixed-origin containers are traversed, never selected.
 * Missing provenance is tainted.
 * Tainted titles are offered as text nodes; selected titles become framework text.
 * This is an additional content policy, not a substitute for normal sanitization.
 * @param {Report} report
 * @param {ReportRedactionPolicy} shouldRedact
 * @returns {Report}
 */
function redactReport(report, shouldRedact) {
  if (typeof shouldRedact !== "function") throw new TypeError("Report redaction requires a block policy");
  const normalized = normalizeReport(report);
  /**
   * @param {ReportInline[]} nodes
   * @returns {{nodes: ReportInline[], redacted: boolean}}
   */
  function inlines(nodes) {
    let redacted = false;
    let matching = false;
    /** @type {ReportInline[]} */
    const result = [];
    for (const node of nodes) {
      const agentSubtree = isAgentSubtree(node);
      if (agentSubtree && shouldRedact(node)) {
        redacted = true;
        if (!matching) result.push(inlineRedactionNotice());
        matching = true;
        continue;
      }
      if ("children" in node) {
        const children = inlines(node.children);
        redacted ||= children.redacted;
        if (agentSubtree && children.redacted) {
          if (!matching) result.push(inlineRedactionNotice());
          matching = true;
          continue;
        }
        matching = false;
        result.push({ ...node, children: children.nodes });
      } else {
        matching = false;
        result.push(node);
      }
    }
    return { nodes: result, redacted };
  }
  /**
   * @param {ReportBlock[]} nodes
   * @returns {ReportBlock[]}
   */
  function blocks(nodes) {
    return nodes.map(node => {
      const agentSubtree = isAgentSubtree(node);
      if (agentSubtree && shouldRedact(node)) return redactionNotice();
      switch (node.type) {
        case "paragraph":
        case "heading":
        case "task": {
          const children = inlines(node.children);
          return children.redacted && agentSubtree ? redactionNotice() : { ...node, children: children.nodes };
        }
        case "table": {
          const headers = node.headers.map(inlines);
          const rows = node.rows.map(row => row.map(inlines));
          if (agentSubtree && [...headers, ...rows.flat()].some(cell => cell.redacted)) return redactionNotice();
          return { ...node, headers: headers.map(cell => cell.nodes), rows: rows.map(row => row.map(cell => cell.nodes)) };
        }
        case "details":
        case "alert":
        case "quote":
          return { ...node, children: blocks(node.children) };
        case "list":
          return { ...node, items: node.items.map(blocks) };
        default:
          return node;
      }
    });
  }
  const titleRedacted = normalized.agentTainted !== false && shouldRedact({ type: "text", value: normalized.title, agentTainted: true });
  const result = {
    ...normalized,
    ...(titleRedacted ? { title: "Content redacted.", agentTainted: false } : {}),
    children: blocks(normalized.children),
    ...(normalized.preamble ? { preamble: blocks(normalized.preamble) } : {}),
  };
  if (normalized.footer) {
    const notices = [];
    const markers = normalized.footer.markers.filter(marker => {
      if (marker.agentTainted === false || !shouldRedact(marker)) return true;
      notices.push(redactionNotice());
      return false;
    });
    result.footer = { ...normalized.footer, children: [...blocks(normalized.footer.children), ...notices], markers };
  }
  return normalizeReport(result);
}

module.exports = { redactReport };
