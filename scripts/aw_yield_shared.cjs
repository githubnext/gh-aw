#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

class InputError extends Error {}
class FinalizeError extends Error {}

const LAMBDA = 0.25;
const OVERLAP_THRESHOLD = 0.7;
const ALLOWED_RECOMMENDATIONS = ["Keep", "Revise", "Merge", "Instrument", "Retire"];
const STOPWORDS = new Set([
  "a",
  "about",
  "after",
  "all",
  "also",
  "an",
  "and",
  "any",
  "are",
  "as",
  "at",
  "be",
  "been",
  "before",
  "being",
  "by",
  "can",
  "do",
  "for",
  "from",
  "get",
  "github",
  "have",
  "if",
  "in",
  "into",
  "is",
  "it",
  "its",
  "job",
  "of",
  "on",
  "or",
  "repo",
  "repository",
  "that",
  "the",
  "their",
  "this",
  "to",
  "use",
  "using",
  "workflow",
  "workflows",
  "with",
  "you",
  "your",
]);
const RISKY_PERMISSION_LEVELS = new Set(["write", "admin"]);
const TELEMETRY_KEYS = new Set([
  "input_tokens",
  "output_tokens",
  "runtime_duration",
  "tool_calls",
  "retries",
  "success_rate",
  "safe_output_success",
  "workflow_invocation_count",
  "user_interaction_count",
  "reviewer_interaction_count",
  "accepted_outputs",
  "outputs_acted_upon",
  "actionable_comments",
  "pr_impact",
  "issues_resolved",
  "bugs_found",
  "manual_minutes_saved",
]);

function clamp(value, lower = 0, upper = 1) {
  let numeric = Number(value);
  if (!Number.isFinite(numeric)) {
    numeric = lower;
  }
  return Math.max(lower, Math.min(upper, numeric));
}

function roundScore(value) {
  return roundTo(clamp(value), 4);
}

function roundTo(value, places = 4) {
  const factor = 10 ** places;
  return Math.round(Number(value) * factor) / factor;
}

function normalizeText(value) {
  if (value === null || value === undefined) {
    return "";
  }
  return String(value).trim();
}

function coerceBool(value, defaultValue = false) {
  if (value === null || value === undefined) {
    return defaultValue;
  }
  if (typeof value === "boolean") {
    return value;
  }
  if (typeof value === "number") {
    return value !== 0;
  }
  if (typeof value === "string") {
    const lowered = value.trim().toLowerCase();
    if (["true", "yes", "1", "observed", "validated"].includes(lowered)) {
      return true;
    }
    if (["false", "no", "0", "missing", "absent"].includes(lowered)) {
      return false;
    }
  }
  return defaultValue;
}

function ensureRequired(obj, keys) {
  for (const key of keys) {
    if (!obj[key]) {
      throw new Error(`${keys.join(", ")} are required`);
    }
  }
}

function ensureParentDir(filePath) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
}

function sortObject(value) {
  if (Array.isArray(value)) {
    return value.map(sortObject);
  }
  if (value && typeof value === "object") {
    const out = {};
    for (const key of Object.keys(value).sort()) {
      out[key] = sortObject(value[key]);
    }
    return out;
  }
  return value;
}

function writeJson(filePath, payload, { sortKeys = true } = {}) {
  ensureParentDir(filePath);
  const finalPayload = sortKeys ? sortObject(payload) : payload;
  fs.writeFileSync(filePath, `${JSON.stringify(finalPayload, null, 2)}\n`, "utf8");
}

function loadJson(filePath, ErrorType = Error) {
  try {
    return JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch (error) {
    if (error && error.code === "ENOENT") {
      throw new ErrorType(`Missing JSON file: ${filePath}`);
    }
    throw new ErrorType(`Malformed JSON in ${filePath}: ${error.message}`);
  }
}

function asList(value) {
  if (value === null || value === undefined) {
    return [];
  }
  return Array.isArray(value) ? value : [value];
}

function splitFrontmatter(text) {
  if (!text.startsWith("---")) {
    return ["", text];
  }
  const lines = text.split(/\r?\n/);
  let endIndex = -1;
  for (let i = 1; i < lines.length; i += 1) {
    if (lines[i].trim() === "---") {
      endIndex = i;
      break;
    }
  }
  if (endIndex < 0) {
    return ["", text];
  }
  return [lines.slice(1, endIndex).join("\n"), lines.slice(endIndex + 1).join("\n")];
}

function splitInlineItems(text) {
  const items = [];
  let current = "";
  let depth = 0;
  let quote = null;
  for (const char of text) {
    if (quote) {
      current += char;
      if (char === quote) {
        quote = null;
      }
      continue;
    }
    if (char === "'" || char === '"') {
      quote = char;
      current += char;
      continue;
    }
    if (char === "[" || char === "{") {
      depth += 1;
      current += char;
      continue;
    }
    if (char === "]" || char === "}") {
      depth = Math.max(0, depth - 1);
      current += char;
      continue;
    }
    if (char === "," && depth === 0) {
      if (current.trim() !== "") {
        items.push(current.trim());
      }
      current = "";
      continue;
    }
    current += char;
  }
  if (current.trim() !== "") {
    items.push(current.trim());
  }
  return items;
}

function parseScalar(value) {
  const trimmed = value.trim();
  if (trimmed === "") {
    return "";
  }
  const lower = trimmed.toLowerCase();
  if (lower === "true") {
    return true;
  }
  if (lower === "false") {
    return false;
  }
  if (["null", "none", "~"].includes(lower)) {
    return null;
  }
  if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
    const inner = trimmed.slice(1, -1).trim();
    if (!inner) {
      return [];
    }
    return splitInlineItems(inner).map((item) => parseScalar(item));
  }
  if (trimmed.startsWith("{") && trimmed.endsWith("}")) {
    const inner = trimmed.slice(1, -1).trim();
    if (!inner) {
      return {};
    }
    const parsed = {};
    for (const item of splitInlineItems(inner)) {
      const [key, raw] = splitKeyValue(item);
      parsed[key] = parseScalar(raw || "");
    }
    return parsed;
  }
  if ((trimmed.startsWith('"') && trimmed.endsWith('"')) || (trimmed.startsWith("'") && trimmed.endsWith("'"))) {
    return trimmed.slice(1, -1);
  }
  if (/^-?\d+$/.test(trimmed)) {
    const n = Number.parseInt(trimmed, 10);
    return Number.isNaN(n) ? trimmed : n;
  }
  if (/^-?\d+\.\d+$/.test(trimmed)) {
    const n = Number.parseFloat(trimmed);
    return Number.isNaN(n) ? trimmed : n;
  }
  return trimmed;
}

function splitKeyValue(text) {
  let depth = 0;
  let quote = null;
  for (let i = 0; i < text.length; i += 1) {
    const char = text[i];
    if (quote) {
      if (char === quote) {
        quote = null;
      }
      continue;
    }
    if (char === "'" || char === '"') {
      quote = char;
      continue;
    }
    if (char === "[" || char === "{") {
      depth += 1;
      continue;
    }
    if (char === "]" || char === "}") {
      depth = Math.max(0, depth - 1);
      continue;
    }
    if (char === ":" && depth === 0) {
      const key = text.slice(0, i).trim();
      const rest = text.slice(i + 1).trim();
      return [key, rest === "" ? null : rest];
    }
  }
  throw new InputError(`Invalid frontmatter line: ${text}`);
}

function maybeSplitMapping(text) {
  try {
    const [key, rest] = splitKeyValue(text);
    if (!/^[A-Za-z0-9_-]+$/.test(key)) {
      return null;
    }
    return [key, rest];
  } catch {
    return null;
  }
}

function nextSignificant(lines, start) {
  let index = start;
  while (index < lines.length) {
    const stripped = lines[index].trim();
    if (stripped && !stripped.startsWith("#")) {
      break;
    }
    index += 1;
  }
  return index;
}

function parseBlockScalar(lines, start, indent) {
  const chunks = [];
  let index = start;
  while (index < lines.length) {
    const raw = lines[index];
    const stripped = raw.trim();
    const currentIndent = raw.length - raw.replace(/^\s+/, "").length;
    if (stripped && currentIndent < indent) {
      break;
    }
    if (stripped === "") {
      chunks.push("");
      index += 1;
      continue;
    }
    if (currentIndent < indent) {
      break;
    }
    chunks.push(raw.slice(indent));
    index += 1;
  }
  return [chunks.join("\n").replace(/\s+$/, ""), index];
}

function parseYamlBlock(lines, start = 0, indent = 0) {
  let currentStart = nextSignificant(lines, start);
  if (currentStart >= lines.length) {
    return [{}, currentStart];
  }
  const line = lines[currentStart];
  let currentIndent = line.length - line.replace(/^\s+/, "").length;
  if (currentIndent < indent) {
    return [{}, currentStart];
  }
  indent = currentIndent;
  const isList = line.trimStart().startsWith("- ");

  if (isList) {
    const items = [];
    let index = currentStart;
    while (index < lines.length) {
      index = nextSignificant(lines, index);
      if (index >= lines.length) {
        break;
      }
      const raw = lines[index];
      const itemIndent = raw.length - raw.replace(/^\s+/, "").length;
      if (itemIndent < indent) {
        break;
      }
      const stripped = raw.slice(itemIndent);
      if (itemIndent !== indent || !stripped.startsWith("- ")) {
        break;
      }
      const payload = stripped.slice(2).trim();
      index += 1;
      if (!payload) {
        const [child, nextIndex] = parseYamlBlock(lines, index, indent + 2);
        items.push(child);
        index = nextIndex;
        continue;
      }
      const mapping = maybeSplitMapping(payload);
      if (mapping) {
        const [key, rest] = mapping;
        const item = {};
        if (["|", ">", "|-", ">-"].includes(rest)) {
          const [child, nextIndex] = parseBlockScalar(lines, index, indent + 4);
          item[key] = child;
          index = nextIndex;
        } else if (rest === null) {
          const [child, nextIndex] = parseYamlBlock(lines, index, indent + 2);
          item[key] = child;
          index = nextIndex;
        } else {
          item[key] = parseScalar(rest);
        }
        while (true) {
          const lookahead = nextSignificant(lines, index);
          if (lookahead >= lines.length) {
            break;
          }
          const nextRaw = lines[lookahead];
          const nextIndent = nextRaw.length - nextRaw.replace(/^\s+/, "").length;
          if (nextIndent < indent + 2 || (nextIndent === indent && nextRaw.trimStart().startsWith("- ")) || nextIndent > indent + 2) {
            break;
          }
          const [extraKey, extraRest] = splitKeyValue(nextRaw.trim());
          index = lookahead + 1;
          if (["|", ">", "|-", ">-"].includes(extraRest)) {
            const [child, nextIndex] = parseBlockScalar(lines, index, indent + 4);
            item[extraKey] = child;
            index = nextIndex;
          } else if (extraRest === null) {
            const [child, nextIndex] = parseYamlBlock(lines, index, indent + 4);
            item[extraKey] = child;
            index = nextIndex;
          } else {
            item[extraKey] = parseScalar(extraRest);
          }
        }
        items.push(item);
        continue;
      }
      items.push(parseScalar(payload));
    }
    return [items, index];
  }

  const mapping = {};
  let index = currentStart;
  while (index < lines.length) {
    index = nextSignificant(lines, index);
    if (index >= lines.length) {
      break;
    }
    const raw = lines[index];
    currentIndent = raw.length - raw.replace(/^\s+/, "").length;
    if (currentIndent < indent || currentIndent > indent) {
      break;
    }
    const stripped = raw.trim();
    if (stripped.startsWith("- ")) {
      break;
    }
    const [key, rest] = splitKeyValue(stripped);
    index += 1;
    if (["|", ">", "|-", ">-"].includes(rest)) {
      const [child, nextIndex] = parseBlockScalar(lines, index, indent + 2);
      mapping[key] = child;
      index = nextIndex;
    } else if (rest === null) {
      if (index < lines.length && nextSignificant(lines, index) < lines.length) {
        const [child, nextIndex] = parseYamlBlock(lines, index, indent + 2);
        mapping[key] = child;
        index = nextIndex;
      } else {
        mapping[key] = {};
      }
    } else {
      mapping[key] = parseScalar(rest);
    }
  }
  return [mapping, index];
}

function parseFrontmatterText(frontmatter) {
  if (!frontmatter.trim()) {
    return {};
  }
  const [parsed] = parseYamlBlock(frontmatter.split(/\r?\n/));
  if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new InputError("Workflow frontmatter must parse to an object");
  }
  return parsed;
}

function readWorkflow(workflowPath) {
  const text = fs.readFileSync(workflowPath, "utf8");
  const [frontmatterText, body] = splitFrontmatter(text);
  return [parseFrontmatterText(frontmatterText), body];
}

function tokenize(text) {
  return (text.toLowerCase().match(/[a-z0-9]+/g) || []).filter((token) => !STOPWORDS.has(token) && token.length > 2);
}

module.exports = {
  InputError,
  FinalizeError,
  LAMBDA,
  OVERLAP_THRESHOLD,
  ALLOWED_RECOMMENDATIONS,
  STOPWORDS,
  RISKY_PERMISSION_LEVELS,
  TELEMETRY_KEYS,
  clamp,
  roundScore,
  roundTo,
  normalizeText,
  coerceBool,
  ensureRequired,
  ensureParentDir,
  writeJson,
  loadJson,
  asList,
  splitFrontmatter,
  parseFrontmatterText,
  readWorkflow,
  tokenize,
};
