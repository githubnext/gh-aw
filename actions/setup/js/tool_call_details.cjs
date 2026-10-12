// @ts-check

/**
 * Formats an argument vector for display only, preserving shell word boundaries.
 * Returns undefined for malformed arrays instead of displaying a partial command.
 * @param {unknown} args
 * @returns {string|undefined}
 */
function formatShellArguments(args) {
  if (!Array.isArray(args)) return undefined;
  const words = [];
  for (const arg of args) {
    if (typeof arg !== "string") return undefined;
    words.push(/^[a-zA-Z0-9_@%+=:,./-]+$/.test(arg) ? arg : `'${arg.replace(/'/g, `'\\''`)}'`);
  }
  return words.join(" ");
}

/**
 * Best-effort extraction of shell command text from a tool.execution_start payload.
 * @param {any} data
 * @returns {string}
 */
function extractShellCommandFromToolData(data) {
  return extractShellCommandDetailsFromToolData(data).command;
}

/**
 * Extracts command text and its shape so displays need not normalize literal argv.
 * @param {any} data
 * @returns {{command: string, isArgumentVector: boolean}}
 */
function extractShellCommandDetailsFromToolData(data) {
  const empty = { command: "", isArgumentVector: false };
  if (!data || typeof data !== "object") return empty;
  // Priority order prefers top-level command-like fields emitted by tool wrappers,
  // then object-shaped payloads used by MCP/SDK tool schemas.
  /** @type {Array<any>} */
  const commandFieldCandidates = [];
  if ("command" in data) commandFieldCandidates.push(data.command);
  if ("input" in data) commandFieldCandidates.push(data.input);
  if ("arguments" in data) commandFieldCandidates.push(data.arguments);
  if ("args" in data) commandFieldCandidates.push(data.args);
  if ("toolInput" in data) commandFieldCandidates.push(data.toolInput);
  if ("parameters" in data) commandFieldCandidates.push(data.parameters);
  for (const candidate of commandFieldCandidates) {
    if (typeof candidate === "string" && candidate.trim()) {
      return { command: candidate.trim(), isArgumentVector: false };
    }
    if (Array.isArray(candidate)) {
      const command = formatShellArguments(candidate);
      if (command) return { command, isArgumentVector: true };
      continue;
    }
    if (!candidate || typeof candidate !== "object") continue;
    if (typeof candidate.command === "string" && candidate.command.trim()) {
      return { command: candidate.command.trim(), isArgumentVector: false };
    }
    if (typeof candidate.cmd === "string" && candidate.cmd.trim()) {
      return { command: candidate.cmd.trim(), isArgumentVector: false };
    }
    const command = formatShellArguments(candidate.args);
    if (command) return { command, isArgumentVector: true };
  }
  return empty;
}

/**
 * Best-effort extraction of the structured tool input from a tool event payload.
 * Copilot CLI events.jsonl and the Copilot SDK spell the payload differently
 * (`input`, `arguments`, `parameters`, ...), so the first populated candidate wins.
 * Returns undefined when the payload carries no structured input.
 * @param {any} data
 * @returns {any}
 */
function extractStructuredToolInput(data) {
  if (!data || typeof data !== "object") return undefined;
  for (const key of ["input", "arguments", "parameters", "args", "toolInput"]) {
    const candidate = data[key];
    if (candidate !== null && typeof candidate === "object") return candidate;
    if (typeof candidate === "string" && candidate.trim()) return candidate;
  }
  return undefined;
}

module.exports = {
  formatShellArguments,
  extractShellCommandDetailsFromToolData,
  extractShellCommandFromToolData,
  extractStructuredToolInput,
};
