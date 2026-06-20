// @ts-check

/**
 * Best-effort extraction of shell command text from a tool.execution_start payload.
 * @param {any} data
 * @returns {string}
 */
function extractShellCommandFromToolData(data) {
  if (!data || typeof data !== "object") return "";
  const directCandidates = [data.command, data.input, data.arguments, data.args];
  for (const candidate of directCandidates) {
    if (typeof candidate === "string" && candidate.trim()) {
      return candidate.trim();
    }
  }
  const nestedCandidates = [data.input, data.arguments, data.args, data.toolInput, data.parameters];
  for (const candidate of nestedCandidates) {
    if (!candidate || typeof candidate !== "object") continue;
    if (typeof candidate.command === "string" && candidate.command.trim()) {
      return candidate.command.trim();
    }
    if (typeof candidate.cmd === "string" && candidate.cmd.trim()) {
      return candidate.cmd.trim();
    }
  }
  return "";
}

module.exports = {
  extractShellCommandFromToolData,
};
