// @ts-check

/**
 * Best-effort extraction of shell command text from a tool.execution_start payload.
 * @param {any} data
 * @returns {string}
 */
function extractShellCommandFromToolData(data) {
  if (!data || typeof data !== "object") return "";
  // Priority order prefers top-level command-like fields emitted by tool wrappers,
  // then object-shaped payloads used by MCP/SDK tool schemas.
  const commandFieldCandidates = [data.command, data.input, data["arguments"], data.args, data.toolInput, data.parameters];
  for (const candidate of commandFieldCandidates) {
    if (typeof candidate === "string" && candidate.trim()) {
      return candidate.trim();
    }
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
