// @ts-check

const { normalizeReportURL } = require("./report_ast.cjs");

/** @typedef {import("./report_ast.cjs").ReportInline} ReportInline */
/** @typedef {import("./report_ast.cjs").ReportBlock} ReportBlock */
/** @typedef {{type: "inline_slot", children: ReportInline[]}} ReportInlineSlot */
/** @typedef {Record<string, string | number | boolean | ReportBlock[] | ReportInlineSlot | undefined>} ReportContext */

/**
 * Import the small Markdown dialect used by runtime report templates into typed nodes.
 * Templates are parsed before scalar substitution; values cannot introduce markup.
 * Existing diagnostic fragments are imported separately, never passed through as raw HTML.
 * @param {string} markdown
 * @param {{context?: ReportContext, minimumHeadingLevel?: number, depth?: number, allowFlagMarkers?: boolean, allowMetadata?: boolean, allowedMentions?: string[]}} [options]
 * @returns {ReportBlock[]}
 */
function parseReportMarkdown(markdown, { context, minimumHeadingLevel = 3, depth = 0, allowFlagMarkers = false, allowMetadata = false, allowedMentions = [] } = {}) {
  if (depth > 16) throw new TypeError("Report template nesting exceeds the maximum depth");
  /** @param {string} key */
  function valueFor(key) {
    const value = context && Object.hasOwn(context, key) ? context[key] : undefined;
    if (context && value === undefined) throw new TypeError("Missing report template field");
    return value;
  }
  /** @param {string} source */
  function substitute(source) {
    return source.replace(/\{(\w+)\}/g, (match, key) => {
      const value = valueFor(key);
      if (typeof value === "object") throw new TypeError("Report nodes cannot appear in scalar fields");
      return value === undefined ? match : String(value);
    });
  }
  /**
   * @param {string} source
   * @param {number} [inlineDepth]
   * @returns {ReportInline[]}
   */
  function inline(source, inlineDepth = 0) {
    if (inlineDepth > 16) throw new TypeError("Report inline nesting exceeds the maximum depth");
    const original = source;
    /** @type {ReportInline[]} */
    const nodes = [];
    const text = value => {
      const previous = nodes.at(-1);
      if (previous?.type === "text") previous.value += value;
      else nodes.push({ type: "text", value });
    };
    while (source) {
      let match;
      if ((match = /^\n/.exec(source))) {
        nodes.push({ type: "soft_break" });
      } else if ((match = /^\\([\\`*_[\]{}()#+.!|>~-])/.exec(source))) {
        text(match[1]);
      } else if ((match = /^(close[sd]?|fix(?:es|ed)?|resolve[sd]?)\s+((?:[a-zA-Z0-9_-][a-zA-Z0-9_.-]*\/[a-zA-Z0-9_-][a-zA-Z0-9_.-]*)?)(\\?)#([1-9]\d*)\b/i.exec(source)) && !/\w/.test(original[original.length - source.length - 1] || "")) {
        const verb = match[1].toLowerCase();
        nodes.push({
          type: "closing_reference",
          action: verb.startsWith("clos") ? "closes" : verb.startsWith("fix") ? "fixes" : "resolves",
          target: { number: match[4], ...(match[2] ? { repository: match[2] } : {}) },
          enabled: !match[3],
        });
      } else if ((match = /^#([1-9]\d*)\b/.exec(source))) {
        nodes.push({ type: "issue_reference", number: match[1] });
      } else if ((match = /^\{(\w+)\}/.exec(source))) {
        const value = valueFor(match[1]);
        if (Array.isArray(value)) throw new TypeError("Report blocks cannot appear inline");
        if (typeof value === "object") nodes.push(...value.children);
        else text(value === undefined ? match[0] : String(value));
      } else if ((match = /^@([A-Za-z0-9](?:[A-Za-z0-9_-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)(?![a-zA-Z0-9_/-])/.exec(source)) && allowedMentions.some(alias => alias.replace(/^@/, "").toLowerCase() === match[1].toLowerCase())) {
        nodes.push({ type: "mention", login: match[1] });
      } else if ((match = /^(`+)(?!`)([\s\S]*?)(?<!`)\1(?!`)/.exec(source))) {
        nodes.push({ type: "code", value: substitute(match[2]) });
      } else if ((match = /^\[([^\]\n]*)\]\(([^)\s]+)\)/.exec(source))) {
        const label = inline(match[1], inlineDepth + 1);
        const url = substitute(match[2]);
        try {
          nodes.push({ type: "link", url: normalizeReportURL(url), children: label });
        } catch (error) {
          if (!(error instanceof TypeError)) throw error;
          nodes.push(...label, { type: "text", value: " (link unavailable: invalid HTTPS URL)" });
        }
      } else if ((match = /^<(https:\/\/[^<>\s]+)>/.exec(source))) {
        const url = substitute(match[1]);
        try {
          nodes.push({ type: "link", url: normalizeReportURL(url), children: [{ type: "text", value: url }] });
        } catch (error) {
          if (!(error instanceof TypeError)) throw error;
          text("Link unavailable: invalid HTTPS URL");
        }
      } else if ((match = /^<sub>([\s\S]*?)<\/sub>/.exec(source))) {
        nodes.push({ type: "subscript", children: inline(match[1], inlineDepth + 1) });
      } else if ((match = /^~~(?=\S)([\s\S]*?\S)~~/.exec(source))) {
        nodes.push({ type: "strike", children: inline(match[1], inlineDepth + 1) });
      } else if ((match = /^(\*{3}|_{3})(?=\S)([\s\S]*?\S)\1/.exec(source))) {
        nodes.push({ type: "strong", children: [{ type: "emphasis", children: inline(match[2], inlineDepth + 2) }] });
      } else if ((match = /^(\*\*|__)(?=\S)([\s\S]*?\S)\1/.exec(source))) {
        nodes.push({ type: "strong", children: inline(match[2], inlineDepth + 1) });
      } else if ((match = /^([*_])(?=\S)([^*_\n]*?\S)\1/.exec(source)) && (match[1] !== "_" || !/\w/.test(original[original.length - source.length - 1] || ""))) {
        nodes.push({ type: "emphasis", children: inline(match[2], inlineDepth + 1) });
      } else {
        text(source[0]);
        source = source.slice(1);
        continue;
      }
      source = source.slice(match[0].length);
    }
    return nodes;
  }

  const lines = markdown.replace(/\r\n?/g, "\n").split("\n");
  let index = 0;
  const nested = source => parseReportMarkdown(source, { context, minimumHeadingLevel: Math.max(3, minimumHeadingLevel), depth: depth + 1, allowFlagMarkers, allowMetadata, allowedMentions });
  const isFieldLine = line => /^(?:\{\w+\}\s*)+$/.test(line.trim());
  const isBlockStart = line =>
    /^(?:#{1,6}\s|```|~~~|>|<details>(?:\s*$|\s*<summary>)|<\/details>\s*$|[-*+]\s|\d+\.\s|---\s*$)/.test(line) ||
    isFieldLine(line) ||
    (allowMetadata && /^<!--/.test(line.trim())) ||
    (allowFlagMarkers && /^<!-- (?:gh-aw-threat-detected|gh-aw-threat-engine-error) -->$/.test(line.trim()));
  /** @param {string} line */
  function cells(line) {
    return line
      .trim()
      .replace(/^\|/, "")
      .replace(/(?<!\\)\|$/, "")
      .split(/(?<!\\)\|/)
      .map(cell => inline(cell.trim()));
  }
  /**
   * @param {boolean} inDetails
   * @param {number} detailsDepth
   * @returns {ReportBlock[]}
   */
  function blocks(inDetails = false, detailsDepth = depth) {
    if (detailsDepth > 16) throw new TypeError("Report details nesting exceeds the maximum depth");
    /** @type {ReportBlock[]} */
    const nodes = [];
    while (index < lines.length) {
      const line = lines[index];
      if (!line.trim()) {
        index++;
        continue;
      }
      if (inDetails && line.trim() === "</details>") {
        index++;
        break;
      }
      let match;
      if (allowMetadata && (match = /^- \[[xX]\] expires <!-- gh-aw-expires: ([^<>]+) --> on .+$/.exec(line.trim()))) {
        nodes.push({ type: "expiration", expiresAt: match[1].trim() });
        index++;
      } else if (allowMetadata && (match = /^<!-- ([a-zA-Z][a-zA-Z0-9_-]*)(?:: ([\s\S]*))? -->$/.exec(line.trim()))) {
        if (["gh-aw-execution", "aw-essential-meta"].includes(match[1])) {
          if (match[2] === undefined) throw new TypeError("JSON report metadata requires a value");
          nodes.push({ type: "json_marker", name: match[1], data: JSON.parse(match[2]) });
        } else if (["gh-aw-agentic-workflow", "gh-aw-failure-issue"].includes(match[1]) && match[2] !== undefined) {
          const [value, ...parts] = match[2].split(/,\s+(?=[a-zA-Z0-9_-]+: )/);
          const fields = parts.map(part => {
            const separator = part.indexOf(": ");
            return { key: part.slice(0, separator), value: part.slice(separator + 2) };
          });
          nodes.push({ type: "xml_marker", name: match[1], value, fields });
        } else {
          nodes.push({ type: "xml_marker", name: match[1], ...(match[2] === undefined ? {} : { value: match[2] }) });
        }
        index++;
      } else if (/^(?:-{3,}|\*{3,}|_{3,})\s*$/.test(line)) {
        nodes.push({ type: "separator" });
        index++;
      } else if (context && isFieldLine(line)) {
        for (const field of line.matchAll(/\{(\w+)\}/g)) {
          const value = valueFor(field[1]);
          if (Array.isArray(value)) nodes.push(...value);
          else if (typeof value === "object") nodes.push({ type: "paragraph", children: value.children });
          else if (String(value).trim()) nodes.push({ type: "paragraph", children: [{ type: "text", value: String(value) }] });
        }
        index++;
      } else if (allowFlagMarkers && (match = /^<!-- (gh-aw-threat-detected|gh-aw-threat-engine-error) -->$/.exec(line.trim()))) {
        nodes.push({ type: "marker", name: match[1] === "gh-aw-threat-detected" ? "gh-aw-threat-detected" : "gh-aw-threat-engine-error" });
        index++;
      } else if ((match = /^\s*(`{3,}|~{3,})(.*)$/.exec(line))) {
        const fence = match[1];
        const language = match[2].trim();
        const content = [];
        index++;
        while (index < lines.length) {
          const closing = /^\s*(`+|~+)\s*$/.exec(lines[index]);
          if (closing && closing[1][0] === fence[0] && closing[1].length >= fence.length) break;
          content.push(lines[index++]);
        }
        if (index < lines.length) index++;
        nodes.push({ type: "code_block", language: /^[a-zA-Z0-9_+-]*$/.test(language) ? language : "text", value: substitute(content.join("\n")) });
      } else if ((match = /^(#{1,6})\s+(.+)$/.exec(line))) {
        nodes.push({ type: "heading", level: Math.max(minimumHeadingLevel, match[1].length), children: inline(match[2]) });
        index++;
      } else if ((match = /^<details>\s*<summary>(.*)<\/summary>$/.exec(line.trim()))) {
        index++;
        nodes.push({ type: "details", summary: substitute(match[1].replace(/^<sub>(.*)<\/sub>$/, "$1")), children: blocks(true, detailsDepth + 1) });
      } else if (line.trim() === "<details>" && /^<summary>.*<\/summary>$/.test(lines[index + 1]?.trim() || "")) {
        const summary = lines[index + 1].trim().slice("<summary>".length, -"</summary>".length);
        index += 2;
        nodes.push({ type: "details", summary: substitute(summary.replace(/^<sub>(.*)<\/sub>$/, "$1")), children: blocks(true, detailsDepth + 1) });
      } else if (/^>/.test(line)) {
        const quoted = [];
        while (index < lines.length && /^>/.test(lines[index])) quoted.push(lines[index++].replace(/^> ?/, ""));
        const alert = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]$/.exec(quoted[0]);
        nodes.push(alert ? { type: "alert", kind: alert[1], children: nested(quoted.slice(1).join("\n")) } : { type: "quote", children: nested(quoted.join("\n")) });
      } else if ((match = /^([-*+]|\d+\.)\s+(.+)$/.exec(line))) {
        const ordered = /^\d/.test(match[1]);
        /** @type {ReportBlock[][]} */
        const items = [];
        while (index < lines.length) {
          const item = /^([-*+]|\d+\.)\s+(.+)$/.exec(lines[index]);
          if (!item || /^\d/.test(item[1]) !== ordered) break;
          const content = [item[2]];
          index++;
          while (index < lines.length && /^\s{2,}\S/.test(lines[index])) content.push(lines[index++].replace(/^ {2,4}/, ""));
          const task = /^\[([ xX])\] (.*)$/.exec(content[0]);
          if (task) items.push([{ type: "task", checked: task[1] !== " ", children: inline(task[2]) }, ...nested(content.slice(1).join("\n"))]);
          else items.push(nested(content.join("\n")));
        }
        nodes.push({ type: "list", ordered, items });
      } else if (line.includes("|") && /^\|?[\s:|-]+\|[\s:|-]*$/.test(lines[index + 1] || "")) {
        const tableLines = [line, lines[index + 1]];
        const headers = cells(line);
        /** @type {ReportInline[][][]} */
        const rows = [];
        index += 2;
        while (index < lines.length && lines[index].includes("|") && lines[index].trim()) {
          tableLines.push(lines[index]);
          rows.push(cells(lines[index++]));
        }
        if (rows.every(row => row.length === headers.length)) nodes.push({ type: "table", headers, rows });
        else nodes.push({ type: "paragraph", children: [{ type: "text", value: substitute(tableLines.join("\n")) }] });
      } else {
        const content = [line];
        index++;
        while (index < lines.length && lines[index].trim() && !isBlockStart(lines[index])) {
          if (lines[index].includes("|") && /^\|?[\s:|-]+\|[\s:|-]*$/.test(lines[index + 1] || "")) break;
          content.push(lines[index++]);
        }
        nodes.push({ type: "paragraph", children: inline(content.join("\n")) });
      }
    }
    return nodes;
  }
  return blocks();
}

module.exports = { parseReportMarkdown };
