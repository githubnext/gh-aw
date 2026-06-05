// @ts-check

const { normalizeTool } = require("./mcp_server_core.cjs");

/**
 * Unwrap mistakenly nested MCP arguments like { create_discussion: { ... } }.
 * Applies only when the outer object does not already carry a type.
 * @param {string} toolName
 * @param {any} args
 * @param {{ debug?: (...args: any[]) => void }} [logger]
 * @returns {any}
 */
function normalizeSafeOutputToolArguments(toolName, args, logger) {
  if (!args || typeof args !== "object" || Array.isArray(args)) {
    return args;
  }
  if (typeof args.type === "string" && args.type.trim()) {
    return args;
  }

  const normalizedToolName = normalizeTool(toolName);
  const candidateKeys = [...new Set([toolName, normalizedToolName, toolName.replace(/_/g, "-"), normalizedToolName.replace(/_/g, "-")])];

  for (const candidateKey of candidateKeys) {
    const nestedArgs = args[candidateKey];
    if (nestedArgs && typeof nestedArgs === "object" && !Array.isArray(nestedArgs)) {
      const outerKeys = Object.keys(args);
      logger?.debug?.(`Recovered wrapped safe-output tool arguments for '${normalizedToolName}' by unwrapping key '${candidateKey}' from payload keys ${JSON.stringify(outerKeys)}`);
      return nestedArgs;
    }
  }

  return args;
}

module.exports = {
  normalizeSafeOutputToolArguments,
};
