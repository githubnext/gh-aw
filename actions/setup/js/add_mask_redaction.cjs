// @ts-check

/**
 * Helpers that make rendered agent logs aware of GitHub Actions `::add-mask::`
 * workflow commands.
 *
 * The GitHub Actions runner masks values registered with `::add-mask::` in the
 * live job log, but raw log files captured as artifacts (e.g. agent-stdio.log)
 * still contain both the `::add-mask::` command lines and the unmasked values.
 * Rendering those raw lines into issues or comments would leak the secrets, so
 * every log excerpt copied into generated content must be redacted first.
 */

/** Matches an `::add-mask::` workflow command and captures its payload. */
const ADD_MASK_COMMAND_RE = /::add-mask::(.*)$/;

/** Replacement used for masked values, matching the runner's own rendering. */
const MASK_REPLACEMENT = "***";

/**
 * Decode a GitHub Actions workflow command payload.
 *
 * @param {string} value
 * @returns {string}
 */
function unescapeWorkflowCommandValue(value) {
  return value.replace(/%0D/gi, "\r").replace(/%0A/gi, "\n").replace(/%25/g, "%");
}

/**
 * Check whether a log line is an `::add-mask::` workflow command.
 *
 * @param {string} line
 * @returns {boolean}
 */
function isAddMaskCommandLine(line) {
  return ADD_MASK_COMMAND_RE.test(line);
}

/**
 * Collect every value registered through `::add-mask::` in the given log content.
 *
 * Multi-line masked values are expanded into their individual lines as well, so
 * that partial occurrences are redacted too.
 *
 * @param {string} logContent
 * @returns {string[]} Unique masked values, longest first
 */
function collectAddMaskedValues(logContent) {
  /** @type {Set<string>} */
  const values = new Set();
  if (!logContent) return [];
  for (const line of logContent.split("\n")) {
    const match = line.match(ADD_MASK_COMMAND_RE);
    if (!match) continue;
    const decoded = unescapeWorkflowCommandValue(match[1]);
    for (const candidate of decoded.split(/\r\n|\r|\n/)) {
      if (!candidate.trim()) continue;
      // Register both the verbatim value and its trimmed form so that surrounding
      // whitespace in either the command payload or the log text never defeats redaction.
      values.add(candidate);
      values.add(candidate.trim());
    }
  }
  // Both verbatim and trimmed forms are included; trimming cannot increase length,
  // so descending-length order always prefers verbatim before trimmed variants.
  return Array.from(values).sort((a, b) => b.length - a.length);
}

/**
 * Escape a literal string for use inside a regular expression.
 *
 * @param {string} value
 * @returns {string}
 */
function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

/**
 * Replace all occurrences of the masked values with `***`.
 *
 * @param {string} text
 * @param {string[]} maskedValues
 * @returns {string}
 */
function redactMaskedValues(text, maskedValues) {
  if (!text || !maskedValues || maskedValues.length === 0) return text;
  const pattern = new RegExp(`(?:${maskedValues.map(escapeRegExp).join("|")})`, "g");
  return text.replace(pattern, MASK_REPLACEMENT);
}

/**
 * Remove `::add-mask::` command lines and redact every masked value from the text.
 *
 * @param {string} text - Text about to be rendered into generated content
 * @param {string[]} maskedValues - Values collected via {@link collectAddMaskedValues}
 * @returns {string}
 */
function applyAddMaskRedaction(text, maskedValues) {
  if (!text) return text;
  const withoutCommands = text
    .split("\n")
    .filter(line => !isAddMaskCommandLine(line))
    .join("\n");
  return redactMaskedValues(withoutCommands, maskedValues);
}

module.exports = {
  ADD_MASK_COMMAND_RE,
  MASK_REPLACEMENT,
  applyAddMaskRedaction,
  collectAddMaskedValues,
  isAddMaskCommandLine,
  redactMaskedValues,
  unescapeWorkflowCommandValue,
};
