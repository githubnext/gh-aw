// @ts-check
/// <reference types="@actions/github-script" />

const { BUILT_IN_PATTERNS, extractMCPGatewayTokens, MCP_GATEWAY_CONFIG_PATHS } = require("./redact_secrets.cjs");

const REDACTED = "***REDACTED***";
const PATCH_FLAG = Symbol.for("gh-aw.step-summary-helper-installed");
const BUILT_IN_REDACTION_REGEXES = BUILT_IN_PATTERNS.map(builtInPattern => new RegExp(builtInPattern.pattern.source, builtInPattern.pattern.flags));

/**
 * @param {any} fn
 * @returns {boolean}
 */
function isMockFunction(fn) {
  return Boolean(fn && typeof fn === "function" && (fn._isMockFunction || fn.mock));
}

/**
 * @param {string} text
 * @returns {string}
 */
function stripLeadingEmoji(text) {
  return String(text || "")
    .replace(/^[\p{Extended_Pictographic}\p{Emoji_Presentation}\uFE0F\s]+/u, "")
    .trimStart();
}

/**
 * @param {number|undefined} level
 * @returns {number}
 */
function normalizeHeadingLevel(level) {
  const parsed = Number.isInteger(level) ? Number(level) : 2;
  if (parsed < 2) return 2;
  if (parsed > 3) return 3;
  return parsed;
}

/**
 * @param {string} markdown
 * @returns {string}
 */
function normalizeHeadingMarkdown(markdown) {
  return String(markdown || "")
    .split("\n")
    .map(line => {
      const headingMatch = line.match(/^(#{1,6})\s+(.*)$/);
      if (headingMatch) {
        const level = normalizeHeadingLevel(headingMatch[1].length);
        const title = stripLeadingEmoji(headingMatch[2]);
        return `${"#".repeat(level)} ${title}`;
      }
      const detailsMatch = line.match(/^<summary>(.*)<\/summary>$/);
      if (detailsMatch) {
        const label = stripLeadingEmoji(detailsMatch[1]);
        return `<summary>${label}</summary>`;
      }
      return line;
    })
    .join("\n");
}

/**
 * @param {string} line
 * @returns {string[]}
 */
function splitTableCells(line) {
  const trimmed = line.trim();
  const content = trimmed.replace(/^\|/, "").replace(/\|$/, "");
  return content.split("|").map(cell => cell.trim());
}

/**
 * @param {string[]} cells
 * @returns {boolean}
 */
function isMarkdownDivider(cells) {
  return cells.length > 0 && cells.every(cell => /^:?-{3,}:?$/.test(cell));
}

/**
 * @param {string} markdown
 * @returns {string}
 */
function normalizeMarkdownTables(markdown) {
  const lines = String(markdown || "").split("\n");
  const normalized = [];

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const isTableLine = /^\s*\|.*\|\s*$/.test(line);
    if (!isTableLine) {
      normalized.push(line);
      continue;
    }

    const headerCells = splitTableCells(line);
    normalized.push(`| ${headerCells.join(" | ")} |`);

    const next = lines[i + 1];
    if (next && /^\s*\|.*\|\s*$/.test(next)) {
      const dividerCells = splitTableCells(next);
      if (isMarkdownDivider(dividerCells)) {
        normalized.push(`| ${headerCells.map(() => "---").join(" | ")} |`);
        i += 1;
      }
    }
  }

  return normalized.join("\n");
}

/**
 * @returns {string[]}
 */
function collectSecretsFromEnv() {
  /** @type {string[]} */
  const values = [];
  const secretNames = process.env.GH_AW_SECRET_NAMES;
  if (secretNames) {
    for (const rawName of secretNames.split(",")) {
      const name = rawName.trim();
      if (!name) continue;
      const value = process.env[`SECRET_${name}`];
      if (typeof value === "string" && value.trim().length >= 6) {
        values.push(value.trim());
      }
    }
  }

  for (const token of extractMCPGatewayTokens(MCP_GATEWAY_CONFIG_PATHS)) {
    if (token && token.trim().length >= 6) {
      values.push(token.trim());
    }
  }

  return values.sort((a, b) => b.length - a.length);
}

/**
 * @param {string} markdown
 * @returns {string}
 */
function redactMarkdown(markdown) {
  let result = String(markdown || "");

  for (const pattern of BUILT_IN_REDACTION_REGEXES) {
    result = result.replace(pattern, REDACTED);
  }

  for (const value of collectSecretsFromEnv()) {
    result = result.replaceAll(value, REDACTED);
  }

  return result;
}

/**
 * @param {string} markdown
 * @returns {string}
 */
function normalizeStepSummaryMarkdown(markdown) {
  return redactMarkdown(normalizeMarkdownTables(normalizeHeadingMarkdown(String(markdown || ""))));
}

/**
 * @param {any[][]} rows
 * @returns {any[][]}
 */
function normalizeSummaryTableRows(rows) {
  return rows.map(row =>
    row.map(cell => {
      if (cell !== null && typeof cell === "object" && "data" in cell) {
        return {
          ...cell,
          data: redactMarkdown(String(cell.data ?? "")),
        };
      }
      return redactMarkdown(String(cell ?? ""));
    })
  );
}

/**
 * Install summary normalization helpers onto core.summary.
 *
 * @param {any} coreObj
 */
function installStepSummaryHelpers(coreObj) {
  if (!coreObj || !coreObj.summary || typeof coreObj.summary !== "object") return;

  const summary = coreObj.summary;
  if (summary[PATCH_FLAG]) return;
  summary[PATCH_FLAG] = true;

  const original = {
    addRaw: typeof summary.addRaw === "function" ? summary.addRaw : null,
    addHeading: typeof summary.addHeading === "function" ? summary.addHeading : null,
    addTable: typeof summary.addTable === "function" ? summary.addTable : null,
    addDetails: typeof summary.addDetails === "function" ? summary.addDetails : null,
  };

  if (original.addRaw) {
    if (!isMockFunction(original.addRaw)) {
      summary.addRaw = function addRawNormalized(text, addEOL) {
        return original.addRaw.call(summary, normalizeStepSummaryMarkdown(String(text ?? "")), addEOL);
      };
    }
  }

  if (original.addHeading) {
    if (!isMockFunction(original.addHeading)) {
      summary.addHeading = function addHeadingNormalized(text, level) {
        return original.addHeading.call(summary, stripLeadingEmoji(String(text ?? "")), normalizeHeadingLevel(level));
      };
    }
  }

  if (original.addTable) {
    if (!isMockFunction(original.addTable)) {
      summary.addTable = function addTableNormalized(rows) {
        if (Array.isArray(rows)) {
          return original.addTable.call(summary, normalizeSummaryTableRows(rows));
        }
        return original.addTable.call(summary, rows);
      };
    }
  }

  if (original.addDetails) {
    if (!isMockFunction(original.addDetails)) {
      summary.addDetails = function addDetailsNormalized(label, content) {
        return original.addDetails.call(summary, stripLeadingEmoji(String(label ?? "")), normalizeStepSummaryMarkdown(String(content ?? "")));
      };
    }
  }
}

module.exports = {
  installStepSummaryHelpers,
  stripLeadingEmoji,
  normalizeHeadingLevel,
  normalizeHeadingMarkdown,
  normalizeMarkdownTables,
  normalizeStepSummaryMarkdown,
  normalizeSummaryTableRows,
};
