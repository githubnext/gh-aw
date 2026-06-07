// @ts-check

/**
 * Copilot SDK Permission Helpers
 *
 * Provides reusable permission-enforcement utilities for Copilot SDK driver
 * implementations.  Extracts allow-tool rules from sidecar server args, builds
 * an on-permission handler that enforces those rules, and logs every denial.
 *
 * Consumed by copilot_sdk_session.cjs (the built-in driver) and available to
 * any custom driver that wants to mirror the same permission policy.
 */

"use strict";

const path = require("path");
const { extractCommandNamesFromPipeline } = require("./bash_command_parser.cjs");

/** @const {number} Default maximum number of permission denials before the session is stopped. */
const MAX_TOOL_DENIALS_DEFAULT = 5;

/**
 * @typedef {{
 *   allowAllTools?: boolean,
 *   allowedTools?: string[],
 * }} CopilotSDKPermissionConfig
 */

/**
 * @typedef {{
 *   info?: (message: string) => void,
 *   warning?: (message: string) => void,
 * }} CopilotSDKCoreLogger
 */

/**
 * Parse a strict positive integer from a number or string.
 * Returns undefined when the input is not a whole positive integer.
 *
 * @param {unknown} value
 * @returns {number | undefined}
 */
function parseStrictPositiveInteger(value) {
  if (typeof value === "number" && Number.isSafeInteger(value) && value > 0) {
    return value;
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (/^\d+$/.test(trimmed)) {
      const parsed = Number.parseInt(trimmed, 10);
      if (Number.isSafeInteger(parsed) && parsed > 0) {
        return parsed;
      }
    }
  }
  return undefined;
}

/**
 * Parse max tool denials threshold from input.
 * Falls back to MAX_TOOL_DENIALS_DEFAULT when unset/invalid.
 *
 * @param {unknown} value
 * @returns {number}
 */
function parseMaxToolDenialsLimit(value) {
  return parseStrictPositiveInteger(value) ?? MAX_TOOL_DENIALS_DEFAULT;
}

/**
 * Read a positive integer from an environment variable with fallback.
 *
 * @param {string} key
 * @param {number} fallback
 * @returns {number}
 */
function getEnvPositiveIntOrDefault(key, fallback) {
  return parseStrictPositiveInteger(process.env[key]) ?? fallback;
}

/**
 * Create a compact, human-readable permission-request summary for diagnostics.
 * Examples: shell(git status), mcp(github.get_file_contents), url(https://example.com).
 *
 * @param {import("@github/copilot-sdk").PermissionRequest} request
 * @returns {string}
 */
function summarizePermissionRequest(request) {
  switch (request.kind) {
    case "shell":
      return `shell(${String(request.fullCommandText || "").trim() || "unknown"})`;
    case "mcp":
      return `mcp(${request.serverName || "unknown"}.${request.toolName || "unknown"})`;
    case "url":
      return `url(${request.url || "unknown"})`;
    case "write":
      return `write(${request.fileName || "unknown"})`;
    case "read":
      return `read(${request.path || "unknown"})`;
    case "custom-tool":
      return `custom-tool(${request.toolName || "unknown"})`;
    default:
      return request.kind;
  }
}

/**
 * @param {CopilotSDKCoreLogger | undefined} coreLogger
 * @param {(msg: string) => void} logger
 * @param {import("@github/copilot-sdk").PermissionRequest} request
 */
function logPermissionDenied(coreLogger, logger, request) {
  const requestSummary = summarizePermissionRequest(request);
  logger(`permission denied by workflow tool permissions: ${requestSummary}`);
  if (coreLogger?.info) {
    coreLogger.info(`Copilot SDK permission denied: ${requestSummary}`);
  }
  if (coreLogger?.warning) {
    coreLogger.warning(`Copilot SDK permission denied by workflow tool permissions: ${requestSummary}`);
  }
}

/**
 * @param {string} value
 * @returns {string}
 */
function normalizePermissionPath(value) {
  return (
    String(value || "")
      .trim()
      .replace(/\\/g, "/")
      .replace(/\/+$/, "") || "/"
  );
}

/**
 * @param {string} shellRule
 * @returns {string[]}
 */
function extractReadablePathPatternsFromShellRule(shellRule) {
  const trimmed = String(shellRule || "").trim();
  if (!trimmed) return [];

  if (trimmed.startsWith("cat ")) {
    return [trimmed.slice("cat ".length).trim()];
  }

  const xargsCatMatch = trimmed.match(/^xargs\s+-a\s+(\S+)\s+cat(?:\s|$)/);
  if (xargsCatMatch) {
    return [xargsCatMatch[1]];
  }

  const lsMatch = trimmed.match(/^ls\s+(\S+)(?:\s|$)/);
  if (lsMatch) {
    return [lsMatch[1]];
  }

  return [];
}

/**
 * @param {string | undefined} requestedPath
 * @param {string[]} allowedPathPatterns
 * @returns {boolean}
 */
function isReadPathAllowedByShellRules(requestedPath, allowedPathPatterns) {
  if (typeof requestedPath !== "string" || requestedPath.trim().length === 0) {
    return false;
  }

  const normalizedRequestedPath = normalizePermissionPath(requestedPath);

  return allowedPathPatterns.some(pattern => {
    const normalizedPattern = normalizePermissionPath(pattern);
    if (normalizedRequestedPath === normalizedPattern) {
      return true;
    }
    return path.posix.matchesGlob(normalizedRequestedPath, normalizedPattern);
  });
}

/**
 * Build an SDK on-permission handler from Copilot CLI allow-tool rules.
 * A handler is always returned so session creation consistently wires explicit
 * permission behavior derived from configuration input.
 *
 * @param {CopilotSDKPermissionConfig | undefined} permissionConfig
 * @param {import("@github/copilot-sdk").PermissionHandler} approveAll
 * @param {{coreLogger?: CopilotSDKCoreLogger, logger?: (msg: string) => void, onDenied?: (requestSummary: string) => void}=} logOptions
 * @returns {import("@github/copilot-sdk").PermissionHandler}
 */
function buildCopilotSDKPermissionHandler(permissionConfig, approveAll, logOptions) {
  const logger = logOptions?.logger ?? (() => {});

  const allowAll = permissionConfig?.allowAllTools === true;
  const allowedTools = Array.isArray(permissionConfig?.allowedTools) ? permissionConfig.allowedTools : [];
  const normalizedAllowedTools = allowedTools
    .filter(tool => typeof tool === "string")
    .map(tool => tool.trim())
    .filter(tool => tool.length > 0);
  const allowedToolEntries = new Set(normalizedAllowedTools);

  // Keep explicit allow-all behavior when requested by config input.
  if (allowAll || allowedToolEntries.size === 0) {
    return approveAll;
  }

  const shellRules = [...allowedToolEntries]
    .filter(tool => tool.startsWith("shell(") && tool.endsWith(")"))
    .map(tool => tool.slice("shell(".length, -1).trim())
    .filter(Boolean);
  const readablePathPatterns = shellRules.flatMap(extractReadablePathPatternsFromShellRule);

  /**
   * Returns true if a single command identifier matches any of the shell rules.
   *
   * Three rule formats are recognised:
   *  - **Wildcard** (`cmd:*`)  — the identifier must equal the prefix before `:*`.
   *    Example: rule `"safeoutputs:*"` matches identifier `"safeoutputs"`.
   *  - **Single-word** (`cmd`) — the identifier must equal the rule exactly.
   *    Example: rule `"ls"` matches identifier `"ls"` only.
   *  - **Full-command** (`cmd arg …`) — rules that contain a space are intentionally
   *    **not** tested here.  They represent exact full-command constraints and are
   *    only meaningful when compared against the whole command text, not against
   *    individual pipeline stages.
   *
   * @param {string} identifier - A single command name (e.g. "ls", "git", "safeoutputs")
   * @returns {boolean} True when any shell rule permits the identifier
   */
  function isIdentifierAllowedByShellRules(identifier) {
    return shellRules.some(rule => {
      if (rule.endsWith(":*")) {
        const prefix = rule.slice(0, -2).trim();
        return prefix.length > 0 && identifier === prefix;
      }
      if (!rule.includes(" ")) {
        return identifier === rule;
      }
      return false;
    });
  }

  /**
   * @param {import("@github/copilot-sdk").PermissionRequest} request
   * @returns {boolean}
   */
  function isAllowed(request) {
    switch (request.kind) {
      case "shell": {
        if (allowedToolEntries.has("shell")) return true;
        const commandIdentifiers = Array.isArray(request.commands) ? request.commands.map(cmd => cmd?.identifier).filter(Boolean) : [];
        const normalizedCommandIdentifiers = [
          ...new Set(
            commandIdentifiers.flatMap(identifier => {
              const text = String(identifier || "").trim();
              if (!text) return [];
              const parsedNames = extractCommandNamesFromPipeline(text);
              return parsedNames.length > 0 ? [text, ...parsedNames] : [text];
            })
          ),
        ];
        const fullCommand = String(request.fullCommandText || "").trim();

        // Primary path: the SDK provided command identifiers.
        // Use original matching logic: single-word and :* rules match identifiers,
        // rules with spaces are compared against the full command text.
        if (normalizedCommandIdentifiers.length > 0) {
          return shellRules.some(rule => {
            if (rule.endsWith(":*")) {
              const prefix = rule.slice(0, -2).trim();
              return prefix.length > 0 && normalizedCommandIdentifiers.includes(prefix);
            }
            if (!rule.includes(" ")) {
              return normalizedCommandIdentifiers.includes(rule);
            }
            return fullCommand === rule;
          });
        }

        // Fallback path: SDK did not supply command identifiers (common for complex
        // piped / chained commands such as `ls /tmp && cat file.json || echo "done"`).
        // Parse fullCommandText to extract the executable name from each pipeline
        // stage and verify that every stage is individually allowed.
        if (fullCommand) {
          const parsedNames = extractCommandNamesFromPipeline(fullCommand);

          if (parsedNames.length > 1) {
            // Multi-stage pipeline: ALL stages must be individually allowed.
            // Exact full-command rules (with spaces) do not apply to individual
            // pipeline stages — only single-word and :* prefix rules.
            return parsedNames.every(name => isIdentifierAllowedByShellRules(name));
          }

          if (parsedNames.length === 1) {
            // Single parsed command: apply the same logic as for a single SDK identifier,
            // including exact full-command rule matching for rules that contain spaces.
            const [name] = parsedNames;
            return shellRules.some(rule => {
              if (rule.endsWith(":*")) {
                const prefix = rule.slice(0, -2).trim();
                return prefix.length > 0 && name === prefix;
              }
              if (!rule.includes(" ")) {
                return name === rule;
              }
              return fullCommand === rule;
            });
          }

          // Could not extract any command names (e.g. complex subshell-only command).
          // Last resort: try an exact full-command match against rules with spaces.
          return shellRules.some(rule => rule.includes(" ") && !rule.endsWith(":*") && fullCommand === rule);
        }

        return false;
      }
      case "write":
        return allowedToolEntries.has("write");
      case "read":
        return allowedToolEntries.has("read") || allowedToolEntries.has("shell") || isReadPathAllowedByShellRules(request.path, readablePathPatterns);
      case "url":
        return allowedToolEntries.has("web_fetch");
      case "mcp":
        // Server-only entries (for example: "github") allow all tools from that server.
        // Server+tool entries (for example: "github(get_file_contents)") allow only that tool.
        return allowedToolEntries.has(request.serverName) || allowedToolEntries.has(`${request.serverName}(${request.toolName})`);
      case "custom-tool":
        return allowedToolEntries.has(request.toolName);
      default:
        return false;
    }
  }

  return request => {
    if (isAllowed(request)) {
      return { kind: "approve-once" };
    }
    const requestSummary = summarizePermissionRequest(request);
    logPermissionDenied(logOptions?.coreLogger, logger, request);
    if (logOptions?.onDenied) {
      logOptions.onDenied(requestSummary);
    }
    return { kind: "reject", feedback: "Tool invocation is not allowed by workflow tool permissions." };
  };
}

/**
 * Parse a CopilotSDKPermissionConfig from a JSON-encoded sidecar args array.
 *
 * Extracts --allow-tool values and the --allow-all-tools flag from the raw
 * GH_AW_COPILOT_SDK_SERVER_ARGS string that the Go engine writes. Returns
 * undefined when no permission-related flags are present so the session
 * on-permission handler can interpret config absence as unrestricted behavior.
 *
 * @param {string | undefined} serverArgsJson - Raw JSON value of GH_AW_COPILOT_SDK_SERVER_ARGS
 * @returns {CopilotSDKPermissionConfig | undefined}
 */
function parsePermissionConfigFromServerArgs(serverArgsJson) {
  if (!serverArgsJson) {
    return undefined;
  }
  /** @type {unknown} */
  let parsed;
  try {
    parsed = JSON.parse(serverArgsJson);
  } catch {
    return undefined;
  }
  if (!Array.isArray(parsed)) {
    return undefined;
  }
  const args = /** @type {unknown[]} */ parsed;

  // --allow-all-tools takes precedence: the sidecar was launched with blanket
  // tool approval, so the driver should mirror that policy.
  if (args.includes("--allow-all-tools")) {
    return { allowAllTools: true };
  }

  // Collect the value of every --allow-tool <entry> pair.
  /** @type {string[]} */
  const allowedTools = [];
  for (let i = 0; i < args.length - 1; i++) {
    if (args[i] === "--allow-tool" && typeof args[i + 1] === "string") {
      allowedTools.push(/** @type {string} */ args[i + 1]);
      i += 1; // consume the value so it is not re-examined as a flag
    }
  }

  return allowedTools.length > 0 ? { allowedTools } : undefined;
}

module.exports = {
  MAX_TOOL_DENIALS_DEFAULT,
  parseStrictPositiveInteger,
  parseMaxToolDenialsLimit,
  getEnvPositiveIntOrDefault,
  summarizePermissionRequest,
  logPermissionDenied,
  normalizePermissionPath,
  extractReadablePathPatternsFromShellRule,
  isReadPathAllowedByShellRules,
  buildCopilotSDKPermissionHandler,
  parsePermissionConfigFromServerArgs,
};
