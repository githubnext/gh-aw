// @ts-check

/**
 * Resolve gh-aw model aliases to concrete catalog model names before passing
 * COPILOT_MODEL to the Copilot CLI (spec §8 fallback resolution algorithm).
 */

/**
 * @param {string} model
 * @returns {{ base: string, params: URLSearchParams }}
 */
function splitModelIdentifier(model) {
  const raw = String(model || "").trim();
  const qIndex = raw.indexOf("?");
  if (qIndex === -1) {
    return { base: raw, params: new URLSearchParams() };
  }
  const base = raw.slice(0, qIndex);
  const params = new URLSearchParams(raw.slice(qIndex + 1));
  return { base, params };
}

/**
 * Caller params win over entry params (spec §8.3 step 2b ii).
 * @param {URLSearchParams} caller
 * @param {URLSearchParams} entry
 * @returns {URLSearchParams}
 */
function mergeParams(caller, entry) {
  const merged = new URLSearchParams();
  for (const [key, value] of entry.entries()) {
    merged.set(key, value);
  }
  for (const [key, value] of caller.entries()) {
    merged.set(key, value);
  }
  return merged;
}

/**
 * @param {URLSearchParams} params
 * @returns {string}
 */
function marshalParams(params) {
  const serialized = params.toString();
  return serialized ? `?${serialized}` : "";
}

/**
 * Case-insensitive glob match; * does not cross / (spec §8.4).
 * @param {string} pattern
 * @param {string} entry
 * @returns {boolean}
 */
function globMatch(pattern, entry) {
  const patternParts = pattern.split("/", 2);
  const entryParts = entry.split("/", 2);
  if (patternParts.length === 1) {
    if (!pattern.includes("*")) {
      return pattern.toLowerCase() === entry.toLowerCase();
    }
    const regex = new RegExp(`^${escapeRegex(pattern).replace(/\*/g, "[^/]*")}$`, "i");
    return regex.test(entry);
  }
  if (entryParts.length === 1) {
    return false;
  }
  if (patternParts[0].toLowerCase() !== entryParts[0].toLowerCase()) {
    return false;
  }
  const regex = new RegExp(`^${escapeRegex(patternParts[1]).replace(/\*/g, "[^/]*")}$`, "i");
  return regex.test(entryParts[1]);
}

/**
 * @param {string} value
 * @returns {string}
 */
function escapeRegex(value) {
  return value.replace(/[.+?^${}()|[\]\\]/g, "\\$&");
}

/**
 * @param {string} modelToken
 * @returns {number[]}
 */
function extractVersionTuple(modelToken) {
  const match = modelToken.match(/(\d+(?:\.\d+)*)/g);
  if (!match || match.length === 0) {
    return [0, 0, 0];
  }
  const tuple = [];
  for (const m of match) {
    for (const part of m.split(".")) {
      tuple.push(Number.parseInt(part, 10) || 0);
    }
  }
  return tuple;
}

/**
 * @param {number[]} left
 * @param {number[]} right
 * @returns {number}
 */
function compareVersionTuples(left, right) {
  const maxLen = Math.max(left.length, right.length);
  for (let i = 0; i < maxLen; i += 1) {
    const a = left[i] ?? 0;
    const b = right[i] ?? 0;
    if (a !== b) {
      return a - b;
    }
  }
  return 0;
}

/**
 * @param {string} pattern
 * @param {string[]} catalog
 * @returns {string|null}
 */
function selectLatestGlobMatch(pattern, catalog) {
  const matches = catalog.filter(entry => globMatch(pattern, entry));
  if (matches.length === 0) {
    return null;
  }
  matches.sort((left, right) => {
    const leftToken = left.includes("/") ? left.split("/", 2)[1] : left;
    const rightToken = right.includes("/") ? right.split("/", 2)[1] : right;
    return compareVersionTuples(extractVersionTuple(rightToken), extractVersionTuple(leftToken));
  });
  return matches[0];
}

/**
 * Build provider-scoped catalog entries from AWF /reflect data.
 * @param {object|null|undefined} reflectData
 * @returns {string[]}
 */
function buildCatalogFromReflect(reflectData) {
  /** @type {Set<string>} */
  const catalog = new Set();
  const endpoints = Array.isArray(reflectData?.endpoints) ? reflectData.endpoints : [];
  for (const endpoint of endpoints) {
    if (!endpoint || endpoint.configured !== true || !Array.isArray(endpoint.models)) {
      continue;
    }
    const providerRaw = String(endpoint.provider || "")
      .trim()
      .toLowerCase();
    const provider = providerRaw === "github-copilot" ? "copilot" : providerRaw;
    for (const model of endpoint.models) {
      const modelId = String(model || "").trim();
      if (!modelId) {
        continue;
      }
      if (modelId.includes("/")) {
        catalog.add(modelId);
        continue;
      }
      if (provider) {
        catalog.add(`${provider}/${modelId}`);
      }
      catalog.add(modelId);
    }
  }
  return [...catalog];
}

/**
 * @param {string} target
 * @param {Record<string, string[]>} aliasMap
 * @param {string[]} catalog
 * @param {{ visited?: Set<string>, logger?: (msg: string) => void }} [options]
 * @returns {string|null}
 */
function resolveModelAlias(target, aliasMap, catalog, options = {}) {
  const visited = options.visited ?? new Set();
  const logger = options.logger ?? (() => {});
  const { base, params } = splitModelIdentifier(target);
  if (!base) {
    return null;
  }

  if (Object.prototype.hasOwnProperty.call(aliasMap, base)) {
    if (visited.has(base)) {
      logger(`model alias resolution: circular alias reference at '${base}'`);
      return null;
    }
    visited.add(base);
    const entries = aliasMap[base];
    if (!Array.isArray(entries)) {
      return null;
    }
    for (const entry of entries) {
      const resolved = resolveAliasEntry(String(entry), params, aliasMap, catalog, new Set(visited), logger);
      if (resolved) {
        return resolved;
      }
    }
  }

  const exact = catalog.find(entry => {
    const entryBase = splitModelIdentifier(entry).base;
    return entryBase.toLowerCase() === base.toLowerCase() || entry.toLowerCase() === base.toLowerCase();
  });
  if (exact) {
    return exact + marshalParams(params);
  }

  return null;
}

/**
 * @param {string} entry
 * @param {URLSearchParams} inheritedParams
 * @param {Record<string, string[]>} aliasMap
 * @param {string[]} catalog
 * @param {Set<string>} visited
 * @param {(msg: string) => void} logger
 * @returns {string|null}
 */
function resolveAliasEntry(entry, inheritedParams, aliasMap, catalog, visited, logger) {
  const { base: eBase, params: eParams } = splitModelIdentifier(entry);
  const mergedParams = mergeParams(inheritedParams, eParams);
  const suffix = marshalParams(mergedParams);

  if (Object.prototype.hasOwnProperty.call(aliasMap, eBase)) {
    return resolveModelAlias(eBase + suffix, aliasMap, catalog, { visited, logger });
  }

  if (eBase.includes("*")) {
    const match = selectLatestGlobMatch(eBase, catalog);
    if (match) {
      return match + suffix;
    }
    return null;
  }

  if (eBase.includes("/")) {
    const found = catalog.find(item => item.toLowerCase() === eBase.toLowerCase());
    return found ? found + suffix : null;
  }

  const bare = catalog.find(item => splitModelIdentifier(item).base.toLowerCase() === eBase.toLowerCase());
  return bare ? bare + suffix : null;
}

/**
 * Copilot CLI expects bare model ids (no copilot/ prefix).
 * @param {string} resolved
 * @returns {string}
 */
function normalizeForCopilotCLI(resolved) {
  const { base, params } = splitModelIdentifier(resolved);
  const slash = base.indexOf("/");
  if (slash > 0) {
    const provider = base.slice(0, slash).toLowerCase();
    if (provider === "copilot" || provider === "github-copilot") {
      return base.slice(slash + 1) + marshalParams(params);
    }
  }
  return resolved;
}

/**
 * Resolve configured COPILOT_MODEL when it is a gh-aw alias.
 * @param {{
 *   configuredModel: string,
 *   aliasMap: Record<string, string[]>|null|undefined,
 *   reflectData: object|null|undefined,
 *   logger?: (msg: string) => void,
 * }} options
 * @returns {string}
 */
function resolveConfiguredCopilotModel(options) {
  const configuredModel = String(options.configuredModel || "").trim();
  const logger = options.logger ?? (() => {});
  const aliasMap = options.aliasMap;
  if (!configuredModel) {
    return configuredModel;
  }
  if (!aliasMap || typeof aliasMap !== "object") {
    return normalizeForCopilotCLI(configuredModel);
  }

  const aliasKey = splitModelIdentifier(configuredModel).base;
  if (!Object.prototype.hasOwnProperty.call(aliasMap, aliasKey)) {
    return normalizeForCopilotCLI(configuredModel);
  }

  const catalog = buildCatalogFromReflect(options.reflectData);
  if (catalog.length === 0) {
    logger(`copilot model alias resolution skipped (empty catalog from awf-reflect)`);
    return configuredModel;
  }

  const resolved = resolveModelAlias(configuredModel, aliasMap, catalog, { logger });
  if (!resolved) {
    logger(`copilot model alias resolution: '${configuredModel}' did not resolve against catalog`);
    return configuredModel;
  }

  const normalized = normalizeForCopilotCLI(resolved);
  if (normalized !== configuredModel) {
    logger(`copilot model alias resolution: '${configuredModel}' -> '${normalized}'`);
  }
  return normalized;
}

module.exports = {
  buildCatalogFromReflect,
  globMatch,
  mergeParams,
  normalizeForCopilotCLI,
  resolveConfiguredCopilotModel,
  resolveModelAlias,
  selectLatestGlobMatch,
  splitModelIdentifier,
};
