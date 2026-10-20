// @ts-check
"use strict";

const SHELL_BLANK_RE = /^[ \t]$/;

function trimShellWhitespace(value) {
  return value.replace(/^(?:[ \t]|\r?\n)+|(?:[ \t]|\r?\n)+$/g, "");
}

function findShellExpansionOpener(commandText, dollarIndex) {
  let openerIndex = dollarIndex + 1;
  while (commandText[openerIndex] === "\\" && commandText[openerIndex + 1] === "\n") {
    openerIndex += 2;
  }
  return ["{", "(", "'"].includes(commandText[openerIndex]) ? openerIndex : -1;
}

/**
 * bash_command_parser.cjs
 *
 * Dedicated bash command parser for permission checking in the Copilot SDK driver.
 *
 * Provides utilities to:
 *   - Split a shell command text on pipeline operators (&&, ||, |, ;, &)
 *   - Extract the executable command name from a shell segment
 *   - Extract all command names from a complex piped/chained command
 *
 * This parser enables the permission checker to handle chained shell commands such as
 *   ls /tmp && cat file.json 2>/dev/null || echo "not found"
 * by extracting individual command names and verifying each one against the allow-list.
 *
 * The parser uses a lightweight state machine that respects single-quoted and
 * double-quoted strings so that operators embedded inside quotes are not treated
 * as pipeline separators.  Subshell expressions $(...) are also skipped as a unit.
 *
 * Security invariant: when parsing is ambiguous or no command names can be extracted
 * the caller receives an empty array and the permission checker falls back to denying
 * the request, ensuring a safe default.
 */

/**
 * Find the end of a quoted word, rejecting nested syntax this parser cannot
 * safely distinguish from the surrounding quote.
 *
 * @param {string} commandText
 * @param {number} start
 * @returns {number} Exclusive end offset, or -1 for unsupported/unclosed quoting.
 */
function findShellQuoteEnd(commandText, start) {
  const quote = commandText[start];
  let parameterDepth = 0;
  for (let i = start + 1; i < commandText.length; i++) {
    const ch = commandText[i];
    if (quote === '"') {
      if (ch === "\\") {
        i++;
        continue;
      }
      if (ch === "`") return -1;
      if (ch === "$") {
        const openerIndex = findShellExpansionOpener(commandText, i);
        if (openerIndex > i + 1) return -1;
        if (openerIndex === i + 1 && (commandText[openerIndex] === "(" || commandText[openerIndex] === "'")) return -1;
        if (openerIndex === i + 1 && commandText[openerIndex] === "{") {
          parameterDepth++;
          i = openerIndex;
          continue;
        }
      }
      if (ch === "}" && parameterDepth > 0) {
        parameterDepth--;
        continue;
      }
      if (parameterDepth > 0 && (ch === "'" || ch === '"')) return -1;
    }
    if (ch === quote) return i + 1;
  }
  return -1;
}

/**
 * Split a shell command text into individual pipeline segments.
 * Splits on the following shell operators: &&, ||, |, ;, &, and newlines.
 *
 * The split respects:
 *   - Single-quoted strings (no escaping inside)
 *   - Double-quoted strings (backslash-escape aware)
 *   - Unquoted backslash escapes and shell comments (terminated by LF, not bare CR)
 *   - $(...) subshell expressions (balanced parentheses)
 *
 * Operators embedded inside any of these constructs are not treated as separators.
 * Unsupported nested quoting, unquoted grouping/glob parentheses, and unclosed
 * constructs fail closed.
 *
 * @param {string} commandText - Raw bash command text that may contain pipeline operators
 * @returns {string[]} Non-empty trimmed segments (operators removed)
 */
function splitOnPipelineOperators(commandText) {
  if (!commandText || typeof commandText !== "string") return [];

  const segments = [];
  let current = "";
  let atWordStart = true;
  let parameterDepth = 0;
  let i = 0;
  const len = commandText.length;

  while (i < len) {
    const ch = commandText[i];

    if (ch === "#" && atWordStart && parameterDepth === 0) {
      while (i < len && commandText[i] !== "\n") i++;
      continue;
    }

    if (ch === "`") return [];

    if (ch === "$") {
      const openerIndex = findShellExpansionOpener(commandText, i);
      if (openerIndex > i + 1) return [];
      if (openerIndex === i + 1 && commandText[openerIndex] === "'") return [];
    }

    if (ch === "\\" && i + 1 < len) {
      if (commandText[i + 1] === "\n") {
        i += 2;
        while (i < len && SHELL_BLANK_RE.test(commandText[i])) i++;
        if (current && !/[ \t]$/.test(current)) {
          current += " ";
        }
      } else {
        current += ch + commandText[i + 1];
        i += 2;
        atWordStart = false;
      }
      continue;
    }

    if (ch === "$" && commandText[i + 1] === "{") {
      current += "${";
      parameterDepth++;
      atWordStart = false;
      i += 2;
      continue;
    }
    if (ch === "}" && parameterDepth > 0) parameterDepth--;

    if (ch === "'" || ch === '"') {
      const end = findShellQuoteEnd(commandText, i);
      if (end < 0) return [];
      atWordStart = false;
      current += commandText.slice(i, end);
      i = end;
      continue;
    }

    // ── $(...) subshell: skip balanced parentheses as a unit ──
    if (ch === "$" && i + 1 < len && commandText[i + 1] === "(") {
      atWordStart = false;
      current += ch;
      i++;
      let depth = 0;
      let subshellParameterDepth = 0;
      while (i < len) {
        const sc = commandText[i];
        if (sc === "#" || sc === "`") return [];
        if (sc === "$") {
          const openerIndex = findShellExpansionOpener(commandText, i);
          if (openerIndex > i + 1) return [];
          if (openerIndex === i + 1 && commandText[openerIndex] === "'") return [];
          if (openerIndex === i + 1 && commandText[openerIndex] === "{") {
            current += commandText.slice(i, openerIndex + 1);
            subshellParameterDepth++;
            i = openerIndex + 1;
            continue;
          }
        }
        if (sc === "\\" && i + 1 < len) {
          current += sc + commandText[i + 1];
          i += 2;
          continue;
        }
        if (sc === "'" || sc === '"') {
          const end = findShellQuoteEnd(commandText, i);
          if (end < 0) return [];
          if (subshellParameterDepth > 0 && /[()]/.test(commandText.slice(i, end))) return [];
          current += commandText.slice(i, end);
          i = end;
          continue;
        }
        if (sc === "}" && subshellParameterDepth > 0) subshellParameterDepth--;
        if (subshellParameterDepth > 0 && (sc === "(" || sc === ")")) return [];
        if (sc === "(") depth++;
        else if (sc === ")") {
          depth--;
          current += sc;
          i++;
          if (depth === 0) break;
          continue;
        }
        current += sc;
        i++;
      }
      if (depth !== 0 || subshellParameterDepth !== 0) return [];
      continue;
    }

    // Extended globs can contain literal hashes; their parentheses need a full lexer.
    if (ch === "(") return [];

    if (parameterDepth > 0) {
      current += ch;
      i++;
      continue;
    }

    // ── Command separators ──

    // && (AND-then)
    if (ch === "&" && i + 1 < len && commandText[i + 1] === "&") {
      segments.push(current);
      current = "";
      atWordStart = true;
      i += 2;
      while (i < len && SHELL_BLANK_RE.test(commandText[i])) i++;
      continue;
    }

    // A lone ampersand backgrounds the preceding command. Ampersands adjacent
    // to redirection operators are part of the redirection, not separators.
    if (ch === "&" && commandText[i - 1] !== ">" && commandText[i + 1] !== ">") {
      segments.push(current);
      current = "";
      atWordStart = true;
      i++;
      while (i < len && SHELL_BLANK_RE.test(commandText[i])) i++;
      continue;
    }

    // || (OR-else) — must be checked before lone |
    if (ch === "|" && i + 1 < len && commandText[i + 1] === "|") {
      segments.push(current);
      current = "";
      atWordStart = true;
      i += 2;
      while (i < len && SHELL_BLANK_RE.test(commandText[i])) i++;
      continue;
    }

    // | (pipe)
    if (ch === "|") {
      segments.push(current);
      current = "";
      atWordStart = true;
      i++;
      while (i < len && SHELL_BLANK_RE.test(commandText[i])) i++;
      continue;
    }

    // ; (sequential)
    if (ch === ";") {
      segments.push(current);
      current = "";
      atWordStart = true;
      i++;
      while (i < len && SHELL_BLANK_RE.test(commandText[i])) i++;
      continue;
    }

    // Newline (sequential) — treat line breaks as command separators,
    // except when escaped as a shell line continuation ("\\" + newline).
    // Handles LF and CRLF; a bare CR remains part of the shell word.
    if (ch === "\r" && commandText[i + 1] === "\n") {
      let backslashRunLength = 0;
      for (let j = current.length - 1; j >= 0 && current[j] === "\\"; j--) {
        backslashRunLength++;
      }
      if (backslashRunLength % 2 === 1) {
        current += ch;
        i++;
        atWordStart = false;
      } else {
        i++;
      }
      continue;
    }

    if (ch === "\n") {
      let backslashRunLength = 0;
      for (let j = current.length - 1; j >= 0 && current[j] === "\\"; j--) {
        backslashRunLength++;
      }

      // Odd number of trailing backslashes means the newline is escaped.
      if (backslashRunLength % 2 === 1) {
        current = current.slice(0, -1);
        i++;
        while (i < len && SHELL_BLANK_RE.test(commandText[i])) i++;
        if (current && !/[ \t]$/.test(current)) {
          current += " ";
        }
        continue;
      }

      segments.push(current);
      current = "";
      atWordStart = true;
      i++;
      while (i < len && SHELL_BLANK_RE.test(commandText[i])) i++;
      continue;
    }

    current += ch;
    atWordStart = /[ \t&<>]/.test(ch);
    i++;
  }

  // Push the final segment
  if (parameterDepth !== 0) return [];
  if (trimShellWhitespace(current)) {
    segments.push(current);
  }

  return segments.map(trimShellWhitespace).filter(s => s.length > 0);
}

/**
 * Clause keywords can prefix an executable command in the same segment
 * (for example: "then cat file", "do git log"). These are skipped and
 * scanning continues to find the command token.
 */
const CLAUSE_KEYWORDS = new Set(["then", "else", "elif", "do"]);

/**
 * Structural shell keywords never represent an executable command token
 * for permission matching in this parser. They introduce/close control
 * structures and are treated as non-command segment starts.
 */
const STRUCTURE_KEYWORDS = new Set(["if", "fi", "for", "done", "while", "until", "case", "esac", "select", "in", "function", "time", "coproc"]);

const SHELL_KEYWORDS = new Set([...CLAUSE_KEYWORDS, ...STRUCTURE_KEYWORDS]);

// IDENTIFIER=VALUE where VALUE is one of:
// - "(...)" double-quoted text (supports escapes like \")
// - '(...)' single-quoted text
// - an unquoted non-space token
const ENV_ASSIGNMENT_PREFIX_RE = /^[A-Za-z_][A-Za-z0-9_]*=(?:"(?:\\.|[^"\\])*"|'[^']*'|[^ \t\n]*)[ \t]*/;

/**
 * Extract the executable command name from a single shell command segment.
 *
 * Skips:
 *   - Leading env-var assignments: VAR=value (any number of them)
 *   - Shell negation operator: !
 *   - Shell grouping braces: { }
 *   - Redirection words that begin with < > or a digit followed by < > &
 *   - Shell flow-control keywords (then, else, fi, do, done, …)
 *
 * Returns null when no executable command name can be determined.
 *
 * @param {string} segment - A single shell segment containing no pipeline operators
 * @returns {string | null} The command name, or null if not extractable
 */
function extractCommandName(segment) {
  if (!segment || typeof segment !== "string") return null;

  let remaining = trimShellWhitespace(segment);
  if (!remaining) return null;

  // Skip leading env-var assignments: IDENTIFIER=anything  (repeat)
  for (;;) {
    const m = remaining.match(ENV_ASSIGNMENT_PREFIX_RE);
    if (!m) break;
    remaining = trimShellWhitespace(remaining.slice(m[0].length));
  }

  if (!remaining) return null;

  for (;;) {
    // Get the first word
    const wordMatch = remaining.match(/^([^ \t\n]+)/);
    if (!wordMatch) return null;

    const word = wordMatch[1];

    // Redirection operators (<, >, 2>, 2>&1, …)
    if (/^[<>]/.test(word) || /^\d+[<>&]/.test(word)) {
      return null;
    }

    // Shell negation / grouping — skip and continue scanning
    if (word === "!" || word === "{" || word === "}") {
      remaining = trimShellWhitespace(remaining.slice(word.length));
      if (!remaining) return null;
      continue;
    }

    // Flow-control keywords are not executable commands
    if (SHELL_KEYWORDS.has(word)) {
      if (CLAUSE_KEYWORDS.has(word)) {
        remaining = trimShellWhitespace(remaining.slice(word.length));
        if (!remaining) return null;
        continue;
      }
      return null;
    }

    return word;
  }
}

/**
 * Extract all unique command names from a bash pipeline or command sequence.
 *
 * Splits the text on &&, ||, |, and ; and extracts the executable command name
 * from each resulting segment.  Returns a deduplicated array preserving
 * first-occurrence order.
 *
 * Returns an empty array when the text is empty, unparseable, or yields no
 * recognisable command names.  Callers should treat an empty result as
 * "unable to determine commands" and fall back to a safe default (deny).
 *
 * @param {string} commandText - Raw bash command text (may include pipeline operators)
 * @returns {string[]} Deduplicated array of command names in first-occurrence order
 */
function extractCommandNamesFromPipeline(commandText) {
  if (!commandText || typeof commandText !== "string") return [];

  const text = trimShellWhitespace(commandText);
  if (!text) return [];

  const segments = splitOnPipelineOperators(text);
  const seen = new Set();
  const names = [];

  for (const segment of segments) {
    const name = extractCommandName(segment);
    if (name && !seen.has(name)) {
      seen.add(name);
      names.push(name);
    }
  }

  return names;
}

module.exports = {
  splitOnPipelineOperators,
  extractCommandName,
  extractCommandNamesFromPipeline,
};
