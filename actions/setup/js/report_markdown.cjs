// @ts-check

const { normalizeReportText, normalizeReportAllowedURL, normalizeReportImage } = require("./report_ast.cjs");
const { buildAllowedDomains, sanitizeUrlProtocols, sanitizeUrlDomains } = require("./sanitize_content_core.cjs");

/**
 * Read a single-line HTML token without interpreting backticks or quoted ">" as delimiters.
 * The token is literal text unless explicitly imported as a supported typed element.
 * @param {string} source
 */
function htmlTag(source) {
  const opening = /^<\/?([A-Za-z][A-Za-z0-9-]*)(?=[\s/>])/.exec(source);
  if (!opening) return undefined;
  let quote = "";
  for (let index = opening[0].length; index < source.length && source[index] !== "\n"; index++) {
    const character = source[index];
    if (quote) {
      if (character === quote) quote = "";
    } else if (character === '"' || character === "'") quote = character;
    else if (character === "<") return undefined;
    else if (character === ">") return { raw: source.slice(0, index + 1), name: opening[1].toLowerCase() };
  }
  return undefined;
}

/**
 * Import only explicit image presentation fields. Unsupported or malformed HTML
 * remains visible, escaped text; it never becomes a raw-HTML AST node.
 * @param {string} raw
 * @param {(value: string) => string} substitute
 * @returns {import("./report_ast.cjs").ReportImage | undefined}
 */
function htmlImage(raw, substitute) {
  if (!/^<img(?=[\s/>])/i.test(raw)) return undefined;
  /** @type {Map<string, string>} */
  const attributes = new Map();
  const attribute = /[ \t]+(src|alt|width|height|align)[ \t]*=[ \t]*(?:"([^"]*)"|'([^']*)'|([^\s"'`=<>]+))/giy;
  let index = 4;
  while (!/^[ \t]*\/?>$/.test(raw.slice(index))) {
    attribute.lastIndex = index;
    const match = attribute.exec(raw);
    if (!match) return undefined;
    const name = match[1].toLowerCase();
    if (attributes.has(name)) return undefined;
    attributes.set(name, substitute((match[2] ?? match[3] ?? match[4]).replace(/&quot;/gi, '"').replace(/&apos;/gi, "'")));
    index = attribute.lastIndex;
  }
  const url = attributes.get("src");
  if (url === undefined) return undefined;
  /** @type {import("./report_ast.cjs").ReportImage} */
  const image = { type: "image", url, alt: attributes.get("alt") ?? "" };
  for (const name of /** @type {const} */ ["width", "height"]) {
    const value = attributes.get(name);
    if (value !== undefined) {
      if (!/^[1-9]\d{0,4}$/.test(value)) return undefined;
      image[name] = Number(value);
    }
  }
  const alignment = attributes.get("align");
  if (alignment !== undefined) {
    const align = alignment.toLowerCase();
    if (align !== "left" && align !== "right" && align !== "center") return undefined;
    image.align = align;
  }
  try {
    return normalizeReportImage(image);
  } catch (error) {
    if (!(error instanceof TypeError)) throw error;
    return undefined;
  }
}

/**
 * Scan destinations before substitution, so scalar values cannot close a link.
 * @param {string} source
 */
function markdownLink(source) {
  const image = source.startsWith("!");
  const start = image ? 1 : 0;
  if (source[start] !== "[") return undefined;
  let labelDepth = 1;
  let index = start + 1;
  for (; index < source.length && source[index] !== "\n"; index++) {
    if (source[index] === "\\") index++;
    else if (source[index] === "[") labelDepth++;
    else if (source[index] === "]" && --labelDepth === 0) break;
  }
  if (labelDepth !== 0 || source.slice(index, index + 2) !== "](") return undefined;
  const label = source.slice(start + 1, index);
  const destinationStart = index + 2;
  let destinationDepth = 1;
  for (index = destinationStart; index < source.length; index++) {
    if (/\s/.test(source[index])) return undefined;
    if (source[index] === "\\") index++;
    else if (source[index] === "(") destinationDepth++;
    else if (source[index] === ")" && --destinationDepth === 0) {
      return { raw: source.slice(0, index + 1), label, destination: source.slice(destinationStart, index).replace(/\\([\\()])/g, "$1"), image };
    }
  }
  return undefined;
}

/** @param {string} source */
function bareURL(source) {
  const match = /^https:\/\/[^\s<>"'`*]+/i.exec(source);
  if (!match) return undefined;
  let url = match[0];
  for (;;) {
    const previous = url;
    url = url.replace(/[.,!?;:]+$/, "");
    for (const [open, close] of [
      ["(", ")"],
      ["[", "]"],
      ["{", "}"],
    ]) {
      if (url.endsWith(close) && url.split(close).length > url.split(open).length) url = url.slice(0, -1);
    }
    if (previous === url) return url;
  }
}

/** @typedef {import("./report_ast.cjs").ReportInline} ReportInline */
/** @typedef {import("./report_ast.cjs").ReportBlock} ReportBlock */
/** @typedef {{type: "inline_slot", children: ReportInline[]}} ReportInlineSlot */
/** @typedef {Record<string, string | number | boolean | ReportBlock[] | ReportInlineSlot | undefined>} ReportContext */

/**
 * Import the small Markdown dialect used by runtime report templates into typed nodes.
 * Templates are parsed before scalar substitution; values cannot introduce markup.
 * Existing diagnostic fragments are imported separately, never passed through as raw HTML.
 * Inline slots do not enable scalar template substitution in surrounding Markdown.
 * onBlockSlot observes actual typed insertions, not literal placeholder text.
 * @param {string} markdown
 * @param {{context?: ReportContext, inlineSlots?: Record<string, ReportInline[]>, blockSlots?: Record<string, ReportBlock[]>, onBlockSlot?: (key: string) => void, agentTainted?: boolean, minimumHeadingLevel?: number, depth?: number, allowFlagMarkers?: boolean, allowMetadata?: boolean, allowedMentions?: string[]}} [options]
 * @returns {ReportBlock[]}
 */
function parseReportMarkdown(markdown, { context, inlineSlots, blockSlots, onBlockSlot, agentTainted, minimumHeadingLevel = 3, depth = 0, allowFlagMarkers = false, allowMetadata = false, allowedMentions = [] } = {}) {
  if (depth > 16) throw new TypeError("Report template nesting exceeds the maximum depth");
  if (agentTainted !== undefined && typeof agentTainted !== "boolean") throw new TypeError("Report agent taint must be a boolean");
  /**
   * Only source nodes receive this origin; inserted nodes keep their own provenance.
   * @template {ReportInline | ReportBlock} T
   * @param {T} node
   * @returns {T}
   */
  const fromSource = node => (agentTainted === undefined ? node : { ...node, agentTainted });
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
   * @param {boolean} [inLink]
   * @returns {ReportInline[]}
   */
  function inline(source, inlineDepth = 0, inLink = false) {
    if (inlineDepth > 16) throw new TypeError("Report inline nesting exceeds the maximum depth");
    // Redact the complete fragment before references or formatting split a secret.
    source = normalizeReportText(source);
    const original = source;
    /** @type {ReportInline[]} */
    const nodes = [];
    const text = value => {
      const previous = nodes.at(-1);
      if (previous?.type === "text" && previous.agentTainted === agentTainted) previous.value += value;
      else nodes.push(fromSource({ type: "text", value }));
    };
    while (source) {
      let match;
      const link = markdownLink(source);
      const autolink = bareURL(source);
      const tag = htmlTag(source);
      if ((match = /^(?: {2,}|\\)\n/.exec(source))) {
        nodes.push(fromSource({ type: "hard_break" }));
      } else if ((match = /^\n/.exec(source))) {
        nodes.push(fromSource({ type: "soft_break" }));
      } else if ((match = /^\\([\\`*_[\]{}()#+.!|<>~-])/.exec(source))) {
        text(match[1]);
      } else if ((match = /^(close[sd]?|fix(?:es|ed)?|resolve[sd]?)\s+((?:[a-zA-Z0-9_-][a-zA-Z0-9_.-]*\/[a-zA-Z0-9_-][a-zA-Z0-9_.-]*)?)(\\?)#([1-9]\d*)\b/i.exec(source)) && !/\w/.test(original[original.length - source.length - 1] || "")) {
        const verb = match[1].toLowerCase();
        nodes.push(
          fromSource({
            type: "closing_reference",
            action: verb.startsWith("clos") ? "closes" : verb.startsWith("fix") ? "fixes" : "resolves",
            target: { number: match[4], ...(match[2] ? { repository: match[2] } : {}) },
            enabled: !match[3],
          })
        );
      } else if ((match = /^#([1-9]\d*)\b/.exec(source))) {
        nodes.push(fromSource({ type: "issue_reference", number: match[1] }));
      } else if ((match = /^\{(\w+)\}/.exec(source))) {
        if (inlineSlots && Object.hasOwn(inlineSlots, match[1])) nodes.push(...inlineSlots[match[1]].map(node => ({ ...node })));
        else {
          const value = valueFor(match[1]);
          if (Array.isArray(value)) throw new TypeError("Report blocks cannot appear inline");
          if (typeof value === "object") nodes.push(...value.children.map(node => ({ ...node })));
          else text(value === undefined ? match[0] : String(value));
        }
      } else if (
        (match = /^@([A-Za-z0-9](?:[A-Za-z0-9_-]{0,37}[A-Za-z0-9])?(?:\/[A-Za-z0-9._-]+)?)(?![a-zA-Z0-9_/-])/.exec(source)) &&
        !/[A-Za-z0-9]/.test(original[original.length - source.length - 1] || "") &&
        allowedMentions.some(alias => alias.replace(/^@/, "").toLowerCase() === match[1].toLowerCase())
      ) {
        nodes.push(fromSource({ type: "mention", login: match[1] }));
      } else if ((match = /^(`+)(?!`)([\s\S]*?)(?<!`)\1(?!`)/.exec(source))) {
        nodes.push(fromSource({ type: "code", value: substitute(match[2]) }));
      } else if ((match = /^`+/.exec(source))) {
        text(match[0]);
      } else if (link && (!inLink || link.image)) {
        /** @type {ReportInline[]} */
        const label = link.image ? [fromSource({ type: "text", value: substitute(link.label) })] : inline(link.label, inlineDepth + 1, true);
        const url = substitute(link.destination);
        try {
          nodes.push(fromSource(link.image ? normalizeReportImage({ type: "image", url, alt: substitute(link.label.replace(/\\([\\`*_[\]{}()#+.!|>~-])/g, "$1")) }) : { type: "link", url: normalizeReportAllowedURL(url), children: label }));
        } catch (error) {
          if (!(error instanceof TypeError)) throw error;
          nodes.push(...label, fromSource({ type: "text", value: link.image ? " (image unavailable: URL must use an allowed HTTPS domain)" : " (link unavailable: invalid HTTPS URL)" }));
        }
        source = source.slice(link.raw.length);
        continue;
      } else if (!inLink && (match = /^<(https:\/\/[^<>\s]+)>/.exec(source))) {
        const url = substitute(match[1]);
        try {
          nodes.push(fromSource({ type: "link", url: normalizeReportAllowedURL(url), children: [fromSource({ type: "text", value: url })] }));
        } catch (error) {
          if (!(error instanceof TypeError)) throw error;
          text("Link unavailable: invalid HTTPS URL");
        }
      } else if (!inLink && autolink && !/[A-Za-z0-9_]/.test(original[original.length - source.length - 1] || "")) {
        const url = substitute(autolink);
        try {
          nodes.push(fromSource({ type: "link", url: normalizeReportAllowedURL(url), children: [fromSource({ type: "text", value: url })] }));
        } catch (error) {
          if (!(error instanceof TypeError)) throw error;
          text("Link unavailable: invalid HTTPS URL");
        }
        source = source.slice(autolink.length);
        continue;
      } else if ((match = /^<sub>([\s\S]*?)<\/sub>/.exec(source))) {
        nodes.push(fromSource({ type: "subscript", children: inline(match[1], inlineDepth + 1, inLink) }));
      } else if (tag) {
        const image = tag.name === "img" ? htmlImage(tag.raw, substitute) : undefined;
        if (image) nodes.push(fromSource(image));
        else text(sanitizeUrlDomains(sanitizeUrlProtocols(substitute(tag.raw)), buildAllowedDomains()));
        source = source.slice(tag.raw.length);
        continue;
      } else if ((match = /^~~(?=\S)([\s\S]*?\S)~~/.exec(source))) {
        nodes.push(fromSource({ type: "strike", children: inline(match[1], inlineDepth + 1, inLink) }));
      } else if ((match = /^(\*{3}|_{3})(?=\S)([\s\S]*?\S)\1/.exec(source))) {
        nodes.push(fromSource({ type: "strong", children: [fromSource({ type: "emphasis", children: inline(match[2], inlineDepth + 2, inLink) })] }));
      } else if ((match = /^(\*\*|__)(?=\S)([\s\S]*?\S)\1/.exec(source))) {
        nodes.push(fromSource({ type: "strong", children: inline(match[2], inlineDepth + 1, inLink) }));
      } else if ((match = /^([*_])(?=\S)([^*_\n]*?\S)\1/.exec(source)) && (match[1] !== "_" || !/\w/.test(original[original.length - source.length - 1] || ""))) {
        nodes.push(fromSource({ type: "emphasis", children: inline(match[2], inlineDepth + 1, inLink) }));
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
  const nested = source =>
    parseReportMarkdown(source, { context, inlineSlots, blockSlots, onBlockSlot, agentTainted, minimumHeadingLevel: Math.max(3, minimumHeadingLevel), depth: depth + 1, allowFlagMarkers, allowMetadata, allowedMentions });
  const blockSlot = line => /^\{(\w+)\}$/.exec(line.trim())?.[1];
  const isBlockSlot = line => blockSlots && Object.hasOwn(blockSlots, blockSlot(line) ?? "");
  const isFieldLine = line => /^(?:\{\w+\}\s*)+$/.test(line.trim()) && (!inlineSlots || [...line.matchAll(/\{(\w+)\}/g)].some(field => !Object.hasOwn(inlineSlots, field[1])));
  const isBlockStart = line =>
    /^(?:#{1,6}\s|```|~~~|>|<details>(?:\s*$|\s*<summary>)|<\/details>\s*$|[-*+]\s|\d+\.\s|---\s*$)/.test(line) ||
    isFieldLine(line) ||
    isBlockSlot(line) ||
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
      const slot = blockSlot(line);
      if (blockSlots && slot !== undefined && Object.hasOwn(blockSlots, slot)) {
        nodes.push(...blockSlots[slot].map(node => ({ ...node })));
        onBlockSlot?.(slot);
        index++;
      } else if (allowMetadata && (match = /^- \[[xX]\] expires <!-- gh-aw-expires: ([^<>]+) --> on .+$/.exec(line.trim()))) {
        nodes.push(fromSource({ type: "expiration", expiresAt: match[1].trim() }));
        index++;
      } else if (allowMetadata && (match = /^<!-- ([a-zA-Z][a-zA-Z0-9_-]*)(?:: ([\s\S]*))? -->$/.exec(line.trim()))) {
        if (["gh-aw-execution", "aw-essential-meta"].includes(match[1])) {
          if (match[2] === undefined) throw new TypeError("JSON report metadata requires a value");
          nodes.push(fromSource({ type: "json_marker", name: match[1], data: JSON.parse(match[2]) }));
        } else if (["gh-aw-agentic-workflow", "gh-aw-failure-issue"].includes(match[1]) && match[2] !== undefined) {
          const [value, ...parts] = match[2].split(/,\s+(?=[a-zA-Z0-9_-]+: )/);
          const fields = parts.map(part => {
            const separator = part.indexOf(": ");
            return { key: part.slice(0, separator), value: part.slice(separator + 2) };
          });
          nodes.push(fromSource({ type: "xml_marker", name: match[1], value, fields }));
        } else {
          nodes.push(fromSource({ type: "xml_marker", name: match[1], ...(match[2] === undefined ? {} : { value: match[2] }) }));
        }
        index++;
      } else if (/^(?:-{3,}|\*{3,}|_{3,})\s*$/.test(line)) {
        nodes.push(fromSource({ type: "separator" }));
        index++;
      } else if (context && isFieldLine(line)) {
        for (const field of line.matchAll(/\{(\w+)\}/g)) {
          const value = valueFor(field[1]);
          if (Array.isArray(value)) nodes.push(...value);
          else if (typeof value === "object") nodes.push(fromSource({ type: "paragraph", children: value.children }));
          else if (String(value).trim()) nodes.push(fromSource({ type: "paragraph", children: [fromSource({ type: "text", value: String(value) })] }));
        }
        index++;
      } else if (allowFlagMarkers && (match = /^<!-- (gh-aw-threat-detected|gh-aw-threat-engine-error) -->$/.exec(line.trim()))) {
        nodes.push(fromSource({ type: "marker", name: match[1] === "gh-aw-threat-detected" ? "gh-aw-threat-detected" : "gh-aw-threat-engine-error" }));
        index++;
      } else if ((match = /^ {0,3}(`{3,}|~{3,})(.*)$/.exec(line)) && (match[1][0] !== "`" || !match[2].includes("`"))) {
        const fence = match[1];
        const language = match[2].trim();
        const content = [];
        index++;
        while (index < lines.length) {
          const closing = /^ {0,3}(`+|~+)[ \t]*$/.exec(lines[index]);
          if (closing && closing[1][0] === fence[0] && closing[1].length >= fence.length) break;
          content.push(lines[index++]);
        }
        if (index < lines.length) index++;
        const value = substitute(content.join("\n"));
        if (language === "json") {
          try {
            nodes.push(fromSource({ type: "structured_data", data: JSON.parse(value) }));
            continue;
          } catch (error) {
            if (!(error instanceof SyntaxError)) throw error;
            // Invalid JSON examples remain literal code.
          }
        }
        nodes.push(fromSource({ type: "code_block", language: /^[a-zA-Z0-9_+-]*$/.test(language) ? language : "text", value }));
      } else if ((match = /^(#{1,6})\s+(.+)$/.exec(line))) {
        nodes.push(fromSource({ type: "heading", level: Math.max(minimumHeadingLevel, match[1].length), children: inline(match[2]) }));
        index++;
      } else if ((match = /^<details>\s*<summary>(.*)<\/summary>$/.exec(line.trim()))) {
        index++;
        nodes.push(fromSource({ type: "details", summary: substitute(match[1].replace(/^<sub>(.*)<\/sub>$/, "$1")), children: blocks(true, detailsDepth + 1) }));
      } else if (line.trim() === "<details>" && /^<summary>.*<\/summary>$/.test(lines[index + 1]?.trim() || "")) {
        const summary = lines[index + 1].trim().slice("<summary>".length, -"</summary>".length);
        index += 2;
        nodes.push(fromSource({ type: "details", summary: substitute(summary.replace(/^<sub>(.*)<\/sub>$/, "$1")), children: blocks(true, detailsDepth + 1) }));
      } else if (/^>/.test(line)) {
        const quoted = [];
        while (index < lines.length && /^>/.test(lines[index])) quoted.push(lines[index++].replace(/^> ?/, ""));
        const alert = /^\[!(NOTE|TIP|IMPORTANT|WARNING|CAUTION)\]$/.exec(quoted[0]);
        nodes.push(fromSource(alert ? { type: "alert", kind: alert[1], children: nested(quoted.slice(1).join("\n")) } : { type: "quote", children: nested(quoted.join("\n")) }));
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
          if (task) items.push([fromSource({ type: "task", checked: task[1] !== " ", children: inline(task[2]) }), ...nested(content.slice(1).join("\n"))]);
          else items.push(nested(content.join("\n")));
        }
        nodes.push(fromSource({ type: "list", ordered, items }));
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
        if (rows.every(row => row.length === headers.length)) nodes.push(fromSource({ type: "table", headers, rows }));
        else nodes.push(fromSource({ type: "paragraph", children: [fromSource({ type: "text", value: substitute(tableLines.join("\n")) })] }));
      } else {
        const content = [line];
        index++;
        while (index < lines.length && lines[index].trim() && !isBlockStart(lines[index])) {
          if (lines[index].includes("|") && /^\|?[\s:|-]+\|[\s:|-]*$/.test(lines[index + 1] || "")) break;
          content.push(lines[index++]);
        }
        nodes.push(fromSource({ type: "paragraph", children: inline(content.join("\n")) }));
      }
    }
    return nodes;
  }
  return blocks();
}

module.exports = { parseReportMarkdown };
