// @ts-check

const { hardenUnicodeText } = require("./sanitize_content_core.cjs");
const { redactDiagnosticText } = require("./diagnostic_sanitization.cjs");
const { escapeMarkdown } = require("./missing_info_formatter.cjs");
const { createExpirationLine } = require("./ephemerals.cjs");

/**
 * @typedef {{type: "text" | "code", value: string} | {type: "strong" | "emphasis", children: ReportInline[]} | {type: "link", url: string, children: ReportInline[]}} ReportInline
 * @typedef {{type: "paragraph", children: ReportInline[]} |
 *   {type: "heading", level: number, children: ReportInline[]} |
 *   {type: "code_block", value: string, language: string} |
 *   {type: "list", ordered: boolean, items: ReportBlock[][]} |
 *   {type: "quote", children: ReportBlock[]} |
 *   {type: "alert", kind: string, children: ReportBlock[]} |
 *   {type: "details", summary: string, children: ReportBlock[]} |
 *   {type: "table", headers: ReportInline[][], rows: ReportInline[][][]} |
 *   {type: "marker", name: "gh-aw-threat-detected" | "gh-aw-threat-engine-error"}} ReportBlock
 * @typedef {{name: "gh-aw-agentic-workflow" | "gh-aw-failure-issue", value: string, fields: {key: string, value: string}[]}} ReportMarker
 * @typedef {{type: "footer", children: ReportBlock[], expiresAt?: string, markers: ReportMarker[]}} ReportFooter
 * @typedef {{type: "report", title: string, children: ReportBlock[], preamble?: ReportBlock[], footer?: ReportFooter}} Report
 */

const ALERT_KINDS = new Set(["NOTE", "TIP", "IMPORTANT", "WARNING", "CAUTION"]);
const MAX_DEPTH = 16;

/** @param {string} value */
function normalizeReportText(value) {
  if (typeof value !== "string") throw new TypeError("Report text must be a string");
  let current = redactDiagnosticText(value);
  for (let pass = 0; pass < 16; pass++) {
    const normalized = hardenUnicodeText(current)
      .replace(/\r\n?/g, "\n")
      .replace(/\x1b\[[0-?]*[ -/]*[@-~]/g, "")
      .replace(/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]/g, "");
    const redacted = redactDiagnosticText(normalized);
    if (redacted === current) return redacted;
    current = redacted;
  }
  throw new TypeError("Report text contains excessively nested encodings");
}

/** @param {string} value */
function normalizeReportTitle(value) {
  return Array.from(normalizeReportText(value).replace(/\s+/g, " ").trim()).slice(0, 256).join("");
}

/** @param {string} value */
function normalizeReportURL(value) {
  const normalized = normalizeReportText(value);
  if (normalized !== value || /[\s\\]/.test(value)) throw new TypeError("Report links must contain a canonical HTTPS URL");
  let url;
  try {
    url = new URL(value);
  } catch (error) {
    throw new TypeError("Report link is not a valid HTTPS URL", { cause: error });
  }
  if (url.protocol !== "https:" || url.username || url.password) throw new TypeError("Report links must use HTTPS without credentials");
  return url.href;
}

/** @param {number} depth */
function checkDepth(depth) {
  if (depth > MAX_DEPTH) throw new TypeError("Report nesting exceeds the maximum depth");
}

/**
 * @param {ReportInline[]} nodes
 * @param {number} depth
 * @returns {ReportInline[]}
 */
function normalizeInlines(nodes, depth, inLink = false) {
  checkDepth(depth);
  /** @type {ReportInline[]} */
  const result = [];
  for (const node of nodes) {
    switch (node.type) {
      case "text":
      case "code": {
        const value = normalizeReportText(node.value).replace(/\s+/g, " ");
        if (!value) break;
        const previous = result.at(-1);
        if ((node.type === "text" || node.type === "code") && previous?.type === node.type) previous.value = (previous.value + value).replace(/\s+/g, " ");
        else result.push({ type: node.type, value });
        break;
      }
      case "strong":
      case "emphasis":
      case "link": {
        if (node.type === "link" && inLink) throw new TypeError("Report links cannot contain other links");
        const children = normalizeInlines(node.children, depth + 1, inLink || node.type === "link");
        if (!children.length) break;
        result.push(node.type === "link" ? { type: "link", url: normalizeReportURL(node.url), children } : { type: node.type, children });
        break;
      }
      default:
        throw new TypeError("Unsupported report inline node");
    }
  }
  return result;
}

/**
 * @param {ReportBlock[]} nodes
 * @param {number} depth
 * @returns {ReportBlock[]}
 */
function normalizeBlocks(nodes, depth) {
  checkDepth(depth);
  /** @type {ReportBlock[]} */
  const result = [];
  for (const node of nodes) {
    switch (node.type) {
      case "paragraph":
      case "heading": {
        const children = normalizeInlines(node.children, depth + 1);
        if (!children.length || children.every(child => child.type === "text" && !child.value.trim())) break;
        if (node.type === "heading") {
          if (!Number.isInteger(node.level)) throw new TypeError("Report heading level must be an integer");
          result.push({ type: "heading", level: Math.max(depth ? 3 : 2, Math.min(6, node.level)), children });
        } else result.push({ type: "paragraph", children });
        break;
      }
      case "code_block": {
        if (!/^[a-zA-Z0-9_+-]*$/.test(node.language)) throw new TypeError("Invalid report code language");
        const value = normalizeReportText(node.value);
        if (value.trim()) result.push({ type: "code_block", language: node.language, value });
        break;
      }
      case "list": {
        const items = node.items.map(item => normalizeBlocks(item, depth + 1)).filter(item => item.length);
        if (items.length) result.push({ type: "list", ordered: Boolean(node.ordered), items });
        break;
      }
      case "details":
      case "alert":
      case "quote": {
        const children = normalizeBlocks(node.children, depth + 1);
        if (node.type === "alert" && !ALERT_KINDS.has(node.kind)) throw new TypeError("Unsupported report alert kind");
        if (!children.length) break;
        if (node.type === "details") {
          const summary = normalizeReportText(node.summary).replace(/\s+/g, " ").trim();
          if (!summary) throw new TypeError("Report details must have a summary");
          result.push({ type: "details", summary, children });
        } else if (node.type === "alert") result.push({ type: "alert", kind: node.kind, children });
        else result.push({ type: "quote", children });
        break;
      }
      case "table": {
        if (!node.headers.length || node.rows.some(row => row.length !== node.headers.length)) throw new TypeError("Report table rows must match the header width");
        result.push({
          type: "table",
          headers: node.headers.map(cell => normalizeInlines(cell, depth + 1)),
          rows: node.rows.map(row => row.map(cell => normalizeInlines(cell, depth + 1))),
        });
        break;
      }
      case "marker":
        if (node.name !== "gh-aw-threat-detected" && node.name !== "gh-aw-threat-engine-error") throw new TypeError("Unsupported report flag marker");
        result.push({ type: "marker", name: node.name });
        break;
      default:
        throw new TypeError("Unsupported report block node");
    }
  }
  return result;
}

/** @param {string} value */
function normalizeMarkerValue(value) {
  return normalizeReportText(value).replace(/\s+/g, " ").trim().replace(/--/g, "%2D%2D").replace(/</g, "%3C").replace(/>/g, "%3E").replace(/,/g, "%2C");
}

/**
 * @param {ReportFooter} footer
 * @returns {ReportFooter}
 */
function normalizeFooter(footer) {
  if (footer.type !== "footer") throw new TypeError("Expected a report footer");
  const markers = footer.markers.map(marker => {
    if (marker.name !== "gh-aw-agentic-workflow" && marker.name !== "gh-aw-failure-issue") throw new TypeError("Unsupported report metadata marker");
    /** @type {Set<string>} */
    const keys = new Set([marker.name]);
    const fields = marker.fields.map(field => {
      if (!/^[a-zA-Z0-9_-]+$/.test(field.key) || keys.has(field.key)) throw new TypeError("Invalid or duplicate report metadata field");
      keys.add(field.key);
      return { key: field.key, value: normalizeMarkerValue(field.value) };
    });
    return { name: marker.name, value: normalizeMarkerValue(marker.value), fields };
  });
  if (new Set(markers.map(marker => marker.name)).size !== markers.length) throw new TypeError("Duplicate report metadata marker");
  /** @type {ReportFooter} */
  const result = { type: "footer", children: normalizeBlocks(footer.children, 0), markers };
  if (footer.expiresAt !== undefined) {
    const date = new Date(footer.expiresAt);
    if (Number.isNaN(date.getTime())) throw new TypeError("Invalid report expiration date");
    return { ...result, expiresAt: date.toISOString() };
  }
  return result;
}

/**
 * Normalize before rendering; reject raw Markdown/HTML nodes rather than bypassing escaping.
 * @param {Report} report
 * @returns {Report}
 */
function normalizeReport(report) {
  if (report.type !== "report") throw new TypeError("Expected a report document");
  const title = normalizeReportTitle(report.title);
  if (!title) throw new TypeError("Report title must not be empty");
  return {
    type: "report",
    title,
    children: normalizeBlocks(report.children, 0),
    ...(report.preamble ? { preamble: normalizeBlocks(report.preamble, 0) } : {}),
    ...(report.footer ? { footer: normalizeFooter(report.footer) } : {}),
  };
}

/** @param {string} value */
function escapeReportText(value) {
  return value
    .split(/(@[a-zA-Z0-9][a-zA-Z0-9_/-]*)/g)
    .map(part =>
      /^@[a-zA-Z0-9]/.test(part)
        ? `\`${part}\``
        : escapeMarkdown(part.replace(/&/g, "&amp;"))
            .replace(/\\([()])/g, "$1")
            .replace(/[|#!~]/g, "\\$&")
            .replace(/^([-+])(?= )/, "\\$1")
            .replace(/^(\d+)\.(?= )/, "$1\\.")
    )
    .join("");
}

/** @param {string} value */
function codeFence(value) {
  return "`".repeat(Array.from(value.matchAll(/`+/g)).reduce((length, match) => Math.max(length, match[0].length + 1), 3));
}

/** @param {ReportInline[]} nodes */
function renderInlines(nodes) {
  return nodes
    .map(node => {
      switch (node.type) {
        case "text":
          return escapeReportText(node.value);
        case "code": {
          const fence = "`".repeat(Array.from(node.value.matchAll(/`+/g)).reduce((length, match) => Math.max(length, match[0].length + 1), 1));
          const padding = /^[` ]|[` ]$/.test(node.value) && !/^ +$/.test(node.value) ? " " : "";
          return `${fence}${padding}${node.value}${padding}${fence}`;
        }
        case "strong":
          return `**${renderInlines(node.children)}**`;
        case "emphasis":
          return `*${renderInlines(node.children)}*`;
        case "link":
          return `[${renderInlines(node.children)}](${node.url.replace(/[()<>"'`]/g, char => `%${char.charCodeAt(0).toString(16).toUpperCase()}`)})`;
      }
    })
    .join("");
}

/** @param {ReportBlock[]} nodes */
function renderBlocks(nodes) {
  return nodes
    .map(node => {
      switch (node.type) {
        case "paragraph":
          return renderInlines(node.children).trim();
        case "heading":
          return `${"#".repeat(node.level)} ${renderInlines(node.children).trim()}`;
        case "code_block": {
          const fence = codeFence(node.value);
          return `${fence}${node.language}\n${node.value}${node.value.endsWith("\n") ? "" : "\n"}${fence}`;
        }
        case "list":
          return node.items
            .map((item, index) => {
              const marker = node.ordered ? `${index + 1}. ` : "- ";
              return marker + renderBlocks(item).replace(/\n/g, "\n" + " ".repeat(marker.length));
            })
            .join("\n");
        case "quote":
        case "alert":
          return (node.type === "alert" ? `[!${node.kind}]\n` : "") + renderBlocks(node.children);
        case "details":
          return `<details>\n<summary>${node.summary
            .replace(/&/g, "&amp;")
            .replace(/</g, "&lt;")
            .replace(/>/g, "&gt;")
            .replace(/@[a-zA-Z0-9][a-zA-Z0-9_/-]*/g, mention => `<code>${mention}</code>`)}</summary>\n\n${renderBlocks(node.children)}\n\n</details>`;
        case "table": {
          const row = cells => `| ${cells.map(cell => renderInlines(cell).replace(/(?<!\\)\|/g, "\\|")).join(" | ")} |`;
          return [row(node.headers), `| ${node.headers.map(() => "---").join(" | ")} |`, ...node.rows.map(row)].join("\n");
        }
        case "marker":
          return `<!-- ${node.name} -->`;
      }
    })
    .map((value, index) =>
      nodes[index].type === "quote" || nodes[index].type === "alert"
        ? value
            .split("\n")
            .map(line => (line ? `> ${line}` : ">"))
            .join("\n")
        : value
    )
    .join("\n\n");
}

/** @param {ReportFooter} footer */
function renderFooter(footer) {
  const attribution = renderBlocks(footer.children);
  const expiration = footer.expiresAt ? `> ${createExpirationLine(new Date(footer.expiresAt))}` : "";
  const markers = footer.markers.map(marker => `<!-- ${[{ key: marker.name, value: marker.value }, ...marker.fields].map(field => `${field.key}: ${field.value}`).join(", ")} -->`).join("\n");
  return [[attribution, expiration].filter(Boolean).join("\n"), markers].filter(Boolean).join("\n\n");
}

/**
 * @param {Report} report
 * @returns {{title: string, body: string}}
 */
function renderReport(report) {
  const normalized = normalizeReport(report);
  const parts = [normalized.preamble ? renderBlocks(normalized.preamble) : "", renderBlocks(normalized.children), normalized.footer ? renderFooter(normalized.footer) : ""];
  return { title: normalized.title, body: parts.filter(Boolean).join("\n\n").trim() + "\n" };
}

module.exports = { normalizeReport, normalizeReportText, normalizeReportTitle, normalizeReportURL, renderReport };
