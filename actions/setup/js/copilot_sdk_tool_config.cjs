// @ts-check

"use strict";

const { createCopilotSDKWebFetchTool } = require("./copilot_sdk_web_fetch.cjs");

const COPILOT_SDK_TOOL_CONFIG_VERSION = 1;
const COPILOT_SDK_NEUTRAL_BUILTIN_TOOLS = Object.freeze(["view", "rg", "glob", "sql"]);
const COPILOT_SDK_SHELL_BUILTIN_TOOLS = Object.freeze(["bash", "read_bash", "stop_bash", "list_bash"]);
const COPILOT_SDK_EDIT_BUILTIN_TOOLS = Object.freeze(["apply_patch", "edit", "create", "delete", "move", "write_bash"]);

/**
 * @typedef {{
 *   bash: boolean,
 *   edit: boolean,
 *   webFetch: boolean,
 *   webSearch: boolean,
 *   mcp: boolean,
 *   cliProxy: boolean,
 * }} CopilotSDKToolCapabilities
 */

/**
 * @typedef {{
 *   version: number,
 *   capabilities: CopilotSDKToolCapabilities,
 *   permissions: {allowedTools: string[]},
 *   explicitlyDisabledTools: string[],
 * }} CopilotSDKToolConfig
 */

/**
 * @param {unknown} value
 * @returns {value is Record<string, unknown>}
 */
function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

/**
 * @param {unknown} value
 * @param {string} field
 * @returns {string[]}
 */
function parseStringArray(value, field) {
  if (!Array.isArray(value) || value.some(entry => typeof entry !== "string" || entry.trim() === "")) {
    throw new Error(`${field} must be an array of non-empty strings`);
  }
  const normalized = value.map(entry => entry.trim());
  if (new Set(normalized).size !== normalized.length) {
    throw new Error(`${field} must not contain duplicate entries`);
  }
  return normalized;
}

/**
 * @param {Record<string, unknown>} value
 * @returns {CopilotSDKToolCapabilities}
 */
function parseCapabilities(value) {
  const fields = ["bash", "edit", "webFetch", "webSearch", "mcp", "cliProxy"];
  for (const field of fields) {
    if (typeof value[field] !== "boolean") {
      throw new Error(`capabilities.${field} must be a boolean`);
    }
  }
  return /** @type {CopilotSDKToolCapabilities} */ {
    bash: Boolean(value.bash),
    edit: Boolean(value.edit),
    webFetch: Boolean(value.webFetch),
    webSearch: Boolean(value.webSearch),
    mcp: Boolean(value.mcp),
    cliProxy: Boolean(value.cliProxy),
  };
}

/**
 * @param {CopilotSDKToolConfig} config
 */
function validateToolPermissionParity(config) {
  const allowed = new Set(config.permissions.allowedTools);
  const hasShellPermission = config.permissions.allowedTools.some(tool => tool === "shell" || (tool.startsWith("shell(") && tool.endsWith(")")));
  const reserved = tool => tool === "read" || tool === "write" || tool === "web_fetch" || tool === "shell" || (tool.startsWith("read(") && tool.endsWith(")")) || (tool.startsWith("shell(") && tool.endsWith(")"));
  const hasMCPPermission = config.permissions.allowedTools.some(tool => !reserved(tool));

  if (config.capabilities.bash !== hasShellPermission) {
    throw new Error("SDK tool contract mismatch: bash visibility and shell permissions differ");
  }
  if (config.capabilities.edit !== allowed.has("write")) {
    throw new Error("SDK tool contract mismatch: edit visibility and write permissions differ");
  }
  if (config.capabilities.webFetch !== allowed.has("web_fetch")) {
    throw new Error("SDK tool contract mismatch: web_fetch visibility and permissions differ");
  }
  if (!config.capabilities.mcp && hasMCPPermission) {
    throw new Error("SDK tool contract mismatch: MCP permissions exist while MCP visibility is disabled");
  }
  const disabledCapability = new Map([
    ["bash", "bash"],
    ["edit", "edit"],
    ["web-fetch", "webFetch"],
    ["web-search", "webSearch"],
    ["cli-proxy", "cliProxy"],
  ]);
  for (const toolName of config.explicitlyDisabledTools) {
    const capability = disabledCapability.get(toolName);
    if (capability && config.capabilities[capability]) {
      throw new Error(`SDK tool contract mismatch: explicitly disabled ${toolName} is visible`);
    }
  }
}

/**
 * Parse the compiler-owned GH_AW_COPILOT_SDK_TOOL_CONFIG value. Missing,
 * malformed, inconsistent, or unsupported values fail closed.
 *
 * @param {string | undefined} value
 * @returns {CopilotSDKToolConfig}
 */
function parseCopilotSDKToolConfig(value) {
  if (!value) {
    throw new Error("GH_AW_COPILOT_SDK_TOOL_CONFIG is required");
  }
  let parsed;
  try {
    parsed = JSON.parse(value);
  } catch (error) {
    throw new Error(`GH_AW_COPILOT_SDK_TOOL_CONFIG must be valid JSON: ${error instanceof Error ? error.message : String(error)}`, { cause: error });
  }
  if (!isRecord(parsed)) {
    throw new Error("GH_AW_COPILOT_SDK_TOOL_CONFIG must be a JSON object");
  }
  if (parsed.version !== COPILOT_SDK_TOOL_CONFIG_VERSION) {
    throw new Error(`unsupported GH_AW_COPILOT_SDK_TOOL_CONFIG version: ${String(parsed.version)}`);
  }
  if (!isRecord(parsed.capabilities)) {
    throw new Error("GH_AW_COPILOT_SDK_TOOL_CONFIG capabilities must be an object");
  }
  if (!isRecord(parsed.permissions)) {
    throw new Error("GH_AW_COPILOT_SDK_TOOL_CONFIG permissions must be an object");
  }

  const allowedTools = parseStringArray(parsed.permissions.allowedTools, "permissions.allowedTools");
  if (allowedTools.length === 0) {
    throw new Error("permissions.allowedTools must not be empty");
  }
  const config = {
    version: COPILOT_SDK_TOOL_CONFIG_VERSION,
    capabilities: parseCapabilities(parsed.capabilities),
    permissions: {
      allowedTools,
    },
    explicitlyDisabledTools: parsed.explicitlyDisabledTools == null ? [] : parseStringArray(parsed.explicitlyDisabledTools, "explicitlyDisabledTools"),
  };
  validateToolPermissionParity(config);
  return config;
}

/**
 * @param {CopilotSDKToolConfig | null | undefined} config
 * @param {{
 *   ToolSet?: typeof import("@github/copilot-sdk").ToolSet,
 *   BuiltInTools?: typeof import("@github/copilot-sdk").BuiltInTools,
 *   defineTool?: typeof import("@github/copilot-sdk").defineTool,
 * }} sdk
 * @param {{fetchImpl?: typeof fetch, timeoutMs?: number, maxRedirects?: number}} [options]
 * @returns {Pick<import("@github/copilot-sdk").SessionConfig, "availableTools" | "tools">}
 */
function buildCopilotSDKSessionToolConfig(config, sdk, options = {}) {
  if (!config) return {};
  if (typeof sdk.ToolSet !== "function" || !sdk.BuiltInTools || !Array.isArray(sdk.BuiltInTools.Isolated)) {
    throw new Error("Copilot SDK ToolSet and BuiltInTools.Isolated are required for compiler-controlled tool filtering");
  }

  const availableTools = new sdk.ToolSet();
  availableTools.addBuiltIn(sdk.BuiltInTools.Isolated.filter(name => name !== "ask_user"));
  availableTools.addBuiltIn(COPILOT_SDK_NEUTRAL_BUILTIN_TOOLS);
  if (config.capabilities.bash) availableTools.addBuiltIn(COPILOT_SDK_SHELL_BUILTIN_TOOLS);
  if (config.capabilities.edit) availableTools.addBuiltIn(COPILOT_SDK_EDIT_BUILTIN_TOOLS);
  if (config.capabilities.webSearch) availableTools.addBuiltIn("web_search");
  if (config.capabilities.mcp) availableTools.addMcp("*");

  /** @type {import("@github/copilot-sdk").Tool<any>[]} */
  const tools = [];
  if (config.capabilities.webFetch) {
    if (typeof sdk.defineTool !== "function") {
      throw new Error("Copilot SDK defineTool is required when tools.web-fetch is enabled");
    }
    tools.push(createCopilotSDKWebFetchTool(sdk.defineTool, options));
    availableTools.addCustom("web_fetch");
  }
  return { availableTools, tools };
}

module.exports = {
  COPILOT_SDK_TOOL_CONFIG_VERSION,
  COPILOT_SDK_NEUTRAL_BUILTIN_TOOLS,
  COPILOT_SDK_SHELL_BUILTIN_TOOLS,
  COPILOT_SDK_EDIT_BUILTIN_TOOLS,
  parseStringArray,
  parseCapabilities,
  validateToolPermissionParity,
  parseCopilotSDKToolConfig,
  buildCopilotSDKSessionToolConfig,
};
