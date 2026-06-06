// @ts-check
"use strict";

/**
 * bash_command_parser.cjs
 *
 * Dedicated bash command parser for permission checking in the Copilot SDK driver.
 *
 * Provides utilities to:
 *   - Split a shell command text on pipeline operators (&&, ||, |, ;)
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
 * Split a shell command text into individual pipeline segments.
 * Splits on the following shell operators: &&, ||, |, ;
 *
 * The split respects:
 *   - Single-quoted strings (no escaping inside)
 *   - Double-quoted strings (backslash-escape aware)
 *   - $(...) subshell expressions (balanced parentheses)
 *
 * Operators embedded inside any of these constructs are not treated as separators.
 *
 * @param {string} commandText - Raw bash command text that may contain pipeline operators
 * @returns {string[]} Non-empty trimmed segments (operators removed)
 */
function splitOnPipelineOperators(commandText) {
  if (!commandText || typeof commandText !== "string") return [];

  const segments = [];
  let current = "";
  let i = 0;
  const len = commandText.length;

  while (i < len) {
    const ch = commandText[i];

    // ── Single-quoted string: no escape sequences, copy verbatim until closing ' ──
    if (ch === "'") {
      current += ch;
      i++;
      while (i < len && commandText[i] !== "'") {
        current += commandText[i];
        i++;
      }
      if (i < len) {
        current += commandText[i]; // closing '
        i++;
      }
      continue;
    }

    // ── Double-quoted string: backslash escapes are recognised ──
    if (ch === '"') {
      current += ch;
      i++;
      while (i < len && commandText[i] !== '"') {
        if (commandText[i] === "\\" && i + 1 < len) {
          current += commandText[i] + commandText[i + 1];
          i += 2;
        } else {
          current += commandText[i];
          i++;
        }
      }
      if (i < len) {
        current += commandText[i]; // closing "
        i++;
      }
      continue;
    }

    // ── $(...) subshell: skip balanced parentheses as a unit ──
    if (ch === "$" && i + 1 < len && commandText[i + 1] === "(") {
      current += ch;
      i++;
      let depth = 0;
      while (i < len) {
        const sc = commandText[i];
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
      continue;
    }

    // ── Pipeline operators ──

    // && (AND-then)
    if (ch === "&" && i + 1 < len && commandText[i + 1] === "&") {
      segments.push(current);
      current = "";
      i += 2;
      while (i < len && commandText[i] !== undefined && /\s/.test(commandText[i])) i++;
      continue;
    }

    // || (OR-else) — must be checked before lone |
    if (ch === "|" && i + 1 < len && commandText[i + 1] === "|") {
      segments.push(current);
      current = "";
      i += 2;
      while (i < len && /\s/.test(commandText[i])) i++;
      continue;
    }

    // | (pipe)
    if (ch === "|") {
      segments.push(current);
      current = "";
      i++;
      while (i < len && /\s/.test(commandText[i])) i++;
      continue;
    }

    // ; (sequential)
    if (ch === ";") {
      segments.push(current);
      current = "";
      i++;
      while (i < len && /\s/.test(commandText[i])) i++;
      continue;
    }

    current += ch;
    i++;
  }

  // Push the final segment
  if (current.trim()) {
    segments.push(current);
  }

  return segments.map(s => s.trim()).filter(s => s.length > 0);
}

/**
 * Shell flow-control keywords that can appear as the first word of a segment
 * but do not represent an executable command.  They must be excluded so the
 * permission checker does not attempt to look up keywords like "then" or "fi"
 * as command names and incorrectly deny (or allow) a pipeline that contains
 * them as part of a compound statement (e.g. `if …; then cat …; fi`).
 */
const SHELL_KEYWORDS = new Set(["then", "else", "elif", "fi", "do", "done", "esac", "in", "function", "time", "coproc"]);

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

  let remaining = segment.trim();
  if (!remaining) return null;

  // Skip leading env-var assignments: IDENTIFIER=anything  (repeat)
  const envAssignRe = /^[A-Za-z_][A-Za-z0-9_]*=\S*\s*/;
  for (;;) {
    const m = remaining.match(envAssignRe);
    if (!m) break;
    remaining = remaining.slice(m[0].length).trim();
  }

  if (!remaining) return null;

  // Get the first word
  const wordMatch = remaining.match(/^(\S+)/);
  if (!wordMatch) return null;

  const word = wordMatch[1];

  // Redirection operators (<, >, 2>, 2>&1, …)
  if (/^[<>]/.test(word) || /^\d+[<>&]/.test(word)) {
    return null;
  }

  // Shell negation / grouping — recurse on the remainder
  if (word === "!" || word === "{" || word === "}") {
    const rest = remaining.slice(word.length).trim();
    return extractCommandName(rest);
  }

  // Flow-control keywords are not executable commands
  if (SHELL_KEYWORDS.has(word)) {
    return null;
  }

  return word;
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

  const text = commandText.trim();
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
