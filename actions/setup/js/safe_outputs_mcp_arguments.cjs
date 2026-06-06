// @ts-check

const { normalizeTool } = require("./mcp_server_core.cjs");

/**
 * Unwrap mistakenly nested MCP arguments like { create_discussion: { ... } }.
 * Applies only when the outer object does not already carry a type.
 * @param {string} toolName
 * @param {any} args
 * @param {{ debug?: (...args: any[]) => void }} [logger]
 * @param {any} [inputSchema]
 * @returns {any}
 */
function normalizeSafeOutputToolArguments(toolName, args, logger, inputSchema) {
  if (!args || typeof args !== "object" || Array.isArray(args)) {
    return args;
  }
  if (typeof args.type === "string" && args.type.trim()) {
    return args;
  }

  let normalizedArgs = args;
  const normalizedToolName = normalizeTool(toolName);
  const candidateKeys = [...new Set([toolName, normalizedToolName, toolName.replace(/_/g, "-"), normalizedToolName.replace(/_/g, "-")])];

  for (const candidateKey of candidateKeys) {
    const nestedArgs = normalizedArgs[candidateKey];
    if (nestedArgs && typeof nestedArgs === "object" && !Array.isArray(nestedArgs)) {
      const outerKeys = Object.keys(normalizedArgs);
      logger?.debug?.(`Recovered wrapped safe-output tool arguments for '${normalizedToolName}' by unwrapping key '${candidateKey}' from payload keys ${JSON.stringify(outerKeys)}`);
      normalizedArgs = nestedArgs;
      break;
    }
  }

  if (!inputSchema || !inputSchema.properties || typeof inputSchema.properties !== "object" || Array.isArray(inputSchema.properties)) {
    return normalizedArgs;
  }

  const parameterSynonyms = new Map();
  for (const [parameterName, parameterSchema] of Object.entries(inputSchema.properties)) {
    parameterSynonyms.set(normalizeTool(parameterName), parameterName);
    const synonyms = Array.isArray(parameterSchema?.["x-synonyms"]) ? parameterSchema["x-synonyms"] : [];
    for (const synonym of synonyms) {
      if (typeof synonym !== "string" || !synonym.trim()) {
        continue;
      }
      const normalizedSynonym = normalizeTool(synonym.trim());
      if (!parameterSynonyms.has(normalizedSynonym)) {
        parameterSynonyms.set(normalizedSynonym, parameterName);
      }
    }
  }

  const remapped = [];
  const remappedArgs = { ...normalizedArgs };
  for (const [providedName, value] of Object.entries(normalizedArgs)) {
    const mappedName = parameterSynonyms.get(normalizeTool(providedName));
    if (!mappedName || mappedName === providedName) {
      continue;
    }

    if (remappedArgs[mappedName] === undefined) {
      remappedArgs[mappedName] = value;
    }
    delete remappedArgs[providedName];
    remapped.push({ from: providedName, to: mappedName });
  }

  if (remapped.length > 0) {
    logger?.debug?.(`Recovered safe-output parameter synonyms for '${normalizedToolName}': ${JSON.stringify(remapped)}`);
  }

  return remappedArgs;
}

module.exports = {
  normalizeSafeOutputToolArguments,
};
