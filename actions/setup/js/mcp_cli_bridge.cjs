// @ts-check

/**
 * mcp_cli_bridge.cjs
 *
 * @safe-outputs-exempt SEC-004: "body" references are transport payloads/responses, not user-authored comment bodies
 *
 * Node.js bridge that handles MCP session protocol for CLI wrapper scripts.
 * Each CLI wrapper is a thin bash script that invokes this bridge with the
 * server configuration and user-provided command + arguments.
 *
 * Protocol flow: initialize → notifications/initialized → (periodic ping) → tools/call
 *
 * All interactions are logged via core.* (shim.cjs) to console and
 * appended as JSONL entries to /tmp/gh-aw/mcp-cli-audit/<server>.jsonl
 * for auditing.
 *
 * Usage (called by generated CLI wrappers):
 *   node mcp_cli_bridge.cjs \
 *     --server-name <name> --server-url <url> \
 *     --tools-file <path> --api-key <key> \
 *     [<command> [--param value ...]]
 *   node mcp_cli_bridge.cjs \
 *     --server-name <name> --server-url <url> \
 *     --tools-file <path> --api-key <key> \
 *     --help
 *   node mcp_cli_bridge.cjs \
 *     --server-name <name> --server-url <url> \
 *     --tools-file <path> --api-key <key> \
 *     <command> --help
 */

require("./shim.cjs");

const fs = require("fs");
const path = require("path");
const http = require("http");

/** Directory for JSONL audit logs (writable inside AWF sandbox via /tmp mount) */
const AUDIT_LOG_DIR = "/tmp/gh-aw/mcp-cli-audit";

/** Default timeout (ms) for HTTP calls to the local MCP gateway */
const DEFAULT_HTTP_TIMEOUT_MS = 15000;

/** Timeout (ms) for tool invocation calls (may be long-running) */
const TOOL_CALL_TIMEOUT_MS = 120000;

/** Timeout (ms) for the notifications/initialized handshake step */
const NOTIFY_TIMEOUT_MS = 10000;

/** Interval (ms) for MCP keepalive pings during long-running tool calls */
const KEEPALIVE_PING_INTERVAL_MS = 10000;

/** Starting JSON-RPC ID for keepalive ping requests */
const KEEPALIVE_PING_ID_START = 1000;

// ---------------------------------------------------------------------------
// Audit logging
// ---------------------------------------------------------------------------

/**
 * Ensure the JSONL audit log directory exists.
 */
function ensureAuditDir() {
  try {
    fs.mkdirSync(AUDIT_LOG_DIR, { recursive: true });
  } catch (err) {
    const core = global.core;
    core.warning(`Failed to create audit log directory ${AUDIT_LOG_DIR}: ${err instanceof Error ? err.message : String(err)}`);
  }
}

/**
 * Append a JSONL entry to the audit log for a given server.
 *
 * @param {string} serverName - Server name (used as filename prefix)
 * @param {Record<string, unknown>} entry - Log entry object
 */
function auditLog(serverName, entry) {
  try {
    const logPath = path.join(AUDIT_LOG_DIR, `${serverName}.jsonl`);
    const record = {
      timestamp: new Date().toISOString(),
      server: serverName,
      ...entry,
    };
    fs.appendFileSync(logPath, JSON.stringify(record) + "\n", { mode: 0o644 });
  } catch (err) {
    const core = global.core;
    core.warning(`Failed to write audit log for ${serverName}: ${err instanceof Error ? err.message : String(err)}`);
  }
}

// ---------------------------------------------------------------------------
// HTTP helpers
// ---------------------------------------------------------------------------

/**
 * Make an HTTP POST request with a JSON body and return the parsed response.
 *
 * @param {string} urlStr - Full URL to POST to
 * @param {Record<string, string>} headers - Request headers
 * @param {unknown} body - Request body (will be JSON-serialized)
 * @param {number} [timeoutMs] - Request timeout in milliseconds
 * @returns {Promise<{statusCode: number, body: unknown, headers: Record<string, string | string[] | undefined>}>}
 */
function httpPostJSON(urlStr, headers, body, timeoutMs = DEFAULT_HTTP_TIMEOUT_MS) {
  return new Promise((resolve, reject) => {
    const parsedUrl = new URL(urlStr);
    const bodyStr = JSON.stringify(body);

    const options = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port || 80,
      path: parsedUrl.pathname + parsedUrl.search,
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json, text/event-stream",
        "Content-Length": Buffer.byteLength(bodyStr),
        ...headers,
      },
    };

    const req = http.request(options, res => {
      let data = "";
      res.on("data", chunk => {
        data += chunk;
      });
      res.on("end", () => {
        let parsed;
        try {
          parsed = JSON.parse(data);
        } catch {
          parsed = data;
        }
        resolve({
          statusCode: res.statusCode || 0,
          body: parsed,
          headers: /** @type {Record<string, string | string[] | undefined>} */ res.headers,
        });
      });
    });

    req.on("error", err => reject(err));

    req.setTimeout(timeoutMs, () => {
      req.destroy();
      reject(new Error(`HTTP request timed out after ${timeoutMs}ms`));
    });

    req.write(bodyStr);
    req.end();
  });
}

// ---------------------------------------------------------------------------
// MCP session protocol
// ---------------------------------------------------------------------------

/**
 * Perform the MCP initialize handshake and return the session ID (if any).
 *
 * @param {string} serverUrl - HTTP URL of the MCP server endpoint
 * @param {string} apiKey - Bearer token for gateway authentication
 * @param {string} serverName - Server name (for logging/auditing)
 * @returns {Promise<string>} Session ID or empty string
 */
async function mcpInitialize(serverUrl, apiKey, serverName) {
  const core = global.core;
  const startMs = Date.now();
  core.info(`[${serverName}] MCP initialize: POST ${serverUrl}`);

  auditLog(serverName, { event: "initialize_start", url: serverUrl });

  try {
    const resp = await httpPostJSON(
      serverUrl,
      { Authorization: apiKey },
      {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {
          capabilities: {},
          clientInfo: { name: "mcp-cli-bridge", version: "1.0.0" },
          protocolVersion: "2024-11-05",
        },
      },
      DEFAULT_HTTP_TIMEOUT_MS
    );

    const sessionId = typeof resp.headers["mcp-session-id"] === "string" ? resp.headers["mcp-session-id"] : "";
    const elapsedMs = Date.now() - startMs;

    core.info(`[${serverName}] MCP initialize: status=${resp.statusCode}, sessionId=${sessionId ? sessionId.slice(0, 8) + "..." : "(none)"}, elapsed=${elapsedMs}ms`);

    auditLog(serverName, {
      event: "initialize_done",
      statusCode: resp.statusCode,
      hasSession: !!sessionId,
      elapsedMs,
    });

    return sessionId;
  } catch (err) {
    const elapsedMs = Date.now() - startMs;
    const message = err instanceof Error ? err.message : String(err);
    core.warning(`[${serverName}] MCP initialize failed (${elapsedMs}ms): ${message}`);
    auditLog(serverName, { event: "initialize_error", error: message, elapsedMs });
    return "";
  }
}

/**
 * Send the notifications/initialized message to complete the MCP handshake.
 *
 * @param {string} serverUrl - HTTP URL of the MCP server endpoint
 * @param {string} apiKey - Bearer token for gateway authentication
 * @param {string} sessionId - Session ID from initialize (may be empty)
 * @param {string} serverName - Server name (for logging/auditing)
 * @returns {Promise<void>}
 */
async function mcpNotifyInitialized(serverUrl, apiKey, sessionId, serverName) {
  const core = global.core;
  const startMs = Date.now();
  core.info(`[${serverName}] MCP notifications/initialized`);

  auditLog(serverName, { event: "notify_initialized_start" });

  /** @type {Record<string, string>} */
  const headers = { Authorization: apiKey };
  if (sessionId) {
    headers["Mcp-Session-Id"] = sessionId;
  }

  try {
    await httpPostJSON(serverUrl, headers, { jsonrpc: "2.0", method: "notifications/initialized", params: {} }, NOTIFY_TIMEOUT_MS);
    const elapsedMs = Date.now() - startMs;
    core.info(`[${serverName}] MCP notifications/initialized: done (${elapsedMs}ms)`);
    auditLog(serverName, { event: "notify_initialized_done", elapsedMs });
  } catch (err) {
    const elapsedMs = Date.now() - startMs;
    const message = err instanceof Error ? err.message : String(err);
    core.warning(`[${serverName}] MCP notifications/initialized failed (${elapsedMs}ms): ${message}`);
    auditLog(serverName, { event: "notify_initialized_error", error: message, elapsedMs });
  }
}

/**
 * Call a tool via the MCP tools/call method.
 *
 * @param {string} serverUrl - HTTP URL of the MCP server endpoint
 * @param {string} apiKey - Bearer token for gateway authentication
 * @param {string} sessionId - Session ID from initialize (may be empty)
 * @param {string} toolName - Name of the tool to call
 * @param {Record<string, unknown>} toolArgs - Tool arguments
 * @param {string} serverName - Server name (for logging/auditing)
 * @returns {Promise<{statusCode: number, body: unknown}>}
 */
async function mcpToolsCall(serverUrl, apiKey, sessionId, toolName, toolArgs, serverName) {
  const core = global.core;
  const startMs = Date.now();
  core.info(`[${serverName}] MCP tools/call: tool=${toolName}, args=${JSON.stringify(toolArgs)}`);

  auditLog(serverName, {
    event: "tools_call_start",
    tool: toolName,
    arguments: toolArgs,
  });

  /** @type {Record<string, string>} */
  const headers = { Authorization: apiKey };
  if (sessionId) {
    headers["Mcp-Session-Id"] = sessionId;
  }

  const resp = await httpPostJSON(
    serverUrl,
    headers,
    {
      jsonrpc: "2.0",
      id: 2,
      method: "tools/call",
      params: { name: toolName, arguments: toolArgs },
    },
    TOOL_CALL_TIMEOUT_MS
  );

  const elapsedMs = Date.now() - startMs;
  core.info(`[${serverName}] MCP tools/call: status=${resp.statusCode}, elapsed=${elapsedMs}ms`);

  auditLog(serverName, {
    event: "tools_call_done",
    tool: toolName,
    statusCode: resp.statusCode,
    elapsedMs,
    response: resp.body,
  });

  return resp;
}

/**
 * Start periodic MCP ping requests to keep a session alive while a tool call runs.
 *
 * @param {string} serverUrl - HTTP URL of the MCP server endpoint
 * @param {string} apiKey - Bearer token for gateway authentication
 * @param {string} sessionId - Session ID from initialize (required for keepalive)
 * @param {string} serverName - Server name (for logging/auditing)
 * @returns {() => void} Stop function to clear the ping timer
 */
function startMcpKeepalivePings(serverUrl, apiKey, sessionId, serverName) {
  const core = global.core;

  if (!sessionId) {
    core.warning(`[${serverName}] MCP keepalive disabled: no sessionId`);
    return () => {};
  }

  /** @type {Record<string, string>} */
  const headers = {
    Authorization: apiKey,
    "Mcp-Session-Id": sessionId,
  };

  let stopped = false;
  let pingId = KEEPALIVE_PING_ID_START;
  /** @type {NodeJS.Timeout | null} */
  let nextTimer = null;

  const runPing = async () => {
    if (stopped) {
      return;
    }
    const startMs = Date.now();
    const currentPingId = pingId++;

    try {
      await httpPostJSON(
        serverUrl,
        headers,
        {
          jsonrpc: "2.0",
          id: currentPingId,
          method: "ping",
        },
        DEFAULT_HTTP_TIMEOUT_MS
      );

      const elapsedMs = Date.now() - startMs;
      core.info(`[${serverName}] MCP keepalive ping: id=${currentPingId}, elapsed=${elapsedMs}ms`);
      auditLog(serverName, { event: "keepalive_ping_done", pingId: currentPingId, elapsedMs });
    } catch (err) {
      const elapsedMs = Date.now() - startMs;
      const message = err instanceof Error ? err.message : String(err);
      core.warning(`[${serverName}] MCP keepalive ping failed: ${message}`);
      auditLog(serverName, { event: "keepalive_ping_error", pingId: currentPingId, error: message, elapsedMs });
    }
    if (!stopped) {
      nextTimer = setTimeout(runPing, KEEPALIVE_PING_INTERVAL_MS);
    }
  };

  nextTimer = setTimeout(runPing, KEEPALIVE_PING_INTERVAL_MS);

  auditLog(serverName, { event: "keepalive_started", intervalMs: KEEPALIVE_PING_INTERVAL_MS });
  core.info(`[${serverName}] MCP keepalive started (interval=${KEEPALIVE_PING_INTERVAL_MS}ms)`);

  return () => {
    if (stopped) {
      return;
    }
    stopped = true;
    if (nextTimer) {
      clearTimeout(nextTimer);
      nextTimer = null;
    }
    auditLog(serverName, { event: "keepalive_stopped" });
    core.info(`[${serverName}] MCP keepalive stopped`);
  };
}

// ---------------------------------------------------------------------------
// CLI argument parsing
// ---------------------------------------------------------------------------

/**
 * Parse the bridge's own arguments from process.argv.
 * Bridge args (--server-name, --server-url, etc.) come before the user command.
 *
 * @param {string[]} argv - process.argv (includes node path and script path)
 * @returns {{serverName: string, serverUrl: string, toolsFile: string, apiKey: string, userArgs: string[]}}
 */
function parseBridgeArgs(argv) {
  // Skip first two entries (node binary + script path)
  const args = argv.slice(2);

  let serverName = "";
  let serverUrl = "";
  let toolsFile = "";
  let apiKey = "";
  let userArgsStart = -1;

  // Bridge args are always paired: --flag value
  // The first argument that doesn't match a known bridge flag marks the start of user args
  const bridgeFlags = new Set(["--server-name", "--server-url", "--tools-file", "--api-key"]);

  for (let i = 0; i < args.length; i++) {
    if (bridgeFlags.has(args[i]) && i + 1 < args.length) {
      switch (args[i]) {
        case "--server-name":
          serverName = args[++i];
          break;
        case "--server-url":
          serverUrl = args[++i];
          break;
        case "--tools-file":
          toolsFile = args[++i];
          break;
        case "--api-key":
          apiKey = args[++i];
          break;
      }
    } else {
      userArgsStart = i;
      break;
    }
  }

  const userArgs = userArgsStart >= 0 ? args.slice(userArgsStart) : [];
  return { serverName, serverUrl, toolsFile, apiKey, userArgs };
}

/**
 * Parse user-provided --key value pairs into a tool arguments object.
 * Supports both --key value and --key=value styles.
 * Boolean flags (--key without a value) are set to true.
 *
 * @param {string[]} args - User arguments after the tool name
 * @param {Record<string, {type?: string|string[]}>} [schemaProperties] - Tool input schema properties
 * @returns {{args: Record<string, unknown>, json: boolean}}
 */
function parseToolArgs(args, schemaProperties = {}) {
  /** @type {Record<string, unknown>} */
  const result = {};
  let jsonOutput = false;
  const hasSchemaProperties = Object.keys(schemaProperties).length > 0;
  const { normalizedSchemaKeyMap, ambiguousNormalizedSchemaKeys } = buildNormalizedSchemaKeyMap(schemaProperties);

  for (let i = 0; i < args.length; i++) {
    if (args[i].startsWith("--")) {
      const raw = args[i].slice(2);
      const eqIdx = raw.indexOf("=");
      if (eqIdx >= 0) {
        // --key=value style
        const key = raw.slice(0, eqIdx);
        if (key === "json") {
          jsonOutput = true;
        } else {
          const canonicalKey = resolveSchemaPropertyKey(key, schemaProperties, normalizedSchemaKeyMap, ambiguousNormalizedSchemaKeys);
          result[canonicalKey] = coerceToolArgValue(canonicalKey, raw.slice(eqIdx + 1), schemaProperties[canonicalKey], result[canonicalKey], !hasSchemaProperties);
        }
      } else if (raw === "json") {
        jsonOutput = true;
      } else if (i + 1 < args.length && !args[i + 1].startsWith("--")) {
        const canonicalKey = resolveSchemaPropertyKey(raw, schemaProperties, normalizedSchemaKeyMap, ambiguousNormalizedSchemaKeys);
        result[canonicalKey] = coerceToolArgValue(canonicalKey, args[i + 1], schemaProperties[canonicalKey], result[canonicalKey], !hasSchemaProperties);
        i++;
      } else {
        const canonicalKey = resolveSchemaPropertyKey(raw, schemaProperties, normalizedSchemaKeyMap, ambiguousNormalizedSchemaKeys);
        result[canonicalKey] = true;
      }
    }
    // Skip non-flag arguments
  }

  return { args: result, json: jsonOutput };
}

/**
 * Normalize a CLI argument/schema key by removing separators and lowercasing.
 *
 * @param {string} key - Raw CLI or schema key
 * @example
 * normalizeSchemaKey("issue-number")
 * // => "issuenumber"
 * @example
 * normalizeSchemaKey("issue_number")
 * // => "issuenumber"
 * @returns {string}
 */
function normalizeSchemaKey(key) {
  return key.replace(/[-_]/g, "").toLowerCase();
}

/**
 * Build a map from normalized key -> canonical schema key.
 *
 * @param {Record<string, {type?: string|string[]}>} schemaProperties - Tool input schema properties
 * @returns {{
 *   normalizedSchemaKeyMap: Map<string, string>,
 *   ambiguousNormalizedSchemaKeys: Set<string>
 * }} Object containing resolvable normalized keys and ambiguous normalized keys
 */
function buildNormalizedSchemaKeyMap(schemaProperties) {
  const normalizedSchemaKeyMap = new Map();
  const ambiguousNormalizedSchemaKeys = new Set();
  for (const key of Object.keys(schemaProperties)) {
    const normalized = normalizeSchemaKey(key);
    if (ambiguousNormalizedSchemaKeys.has(normalized)) {
      continue;
    }
    const existing = normalizedSchemaKeyMap.get(normalized);
    if (existing === undefined) {
      normalizedSchemaKeyMap.set(normalized, key);
    } else if (existing !== key) {
      ambiguousNormalizedSchemaKeys.add(normalized);
      normalizedSchemaKeyMap.delete(normalized);
    }
  }
  return { normalizedSchemaKeyMap, ambiguousNormalizedSchemaKeys };
}

/**
 * Resolve a user-provided CLI key to the canonical schema key when possible.
 * Falls back to the original key when no schema match exists.
 *
 * @param {string} key - User-provided CLI argument key (without leading `--`)
 * @param {Record<string, {type?: string|string[]}>} schemaProperties - Tool input schema properties
 * @param {Map<string, string>} normalizedSchemaKeyMap - Map from normalized key to canonical schema key
 * @param {Set<string>} ambiguousNormalizedSchemaKeys - Normalized keys that map to multiple schema keys
 * @returns {string}
 */
function resolveSchemaPropertyKey(key, schemaProperties, normalizedSchemaKeyMap, ambiguousNormalizedSchemaKeys) {
  if (Object.prototype.hasOwnProperty.call(schemaProperties, key)) {
    return key;
  }
  const normalized = normalizeSchemaKey(key);
  if (ambiguousNormalizedSchemaKeys.has(normalized)) {
    return key;
  }
  return normalizedSchemaKeyMap.get(normalized) || key;
}

/**
 * Parse and coerce a CLI argument value based on the MCP tool schema property type.
 *
 * @param {string} key - Argument key name
 * @param {string} rawValue - Raw CLI value
 * @param {{type?: string|string[]}|undefined} schemaProperty - JSON schema property
 * @param {unknown} existingValue - Existing value (for repeated flags)
 * @param {boolean} [allowNumericFallback=false] - Allow numeric parsing when schema is unavailable
 * @returns {unknown}
 */
function coerceToolArgValue(key, rawValue, schemaProperty, existingValue, allowNumericFallback = false) {
  /** @type {string[]} */
  const types = [];
  if (schemaProperty && typeof schemaProperty === "object" && "type" in schemaProperty && schemaProperty.type != null) {
    if (Array.isArray(schemaProperty.type)) {
      for (const t of schemaProperty.type) {
        if (typeof t === "string") {
          types.push(t);
        }
      }
    } else if (typeof schemaProperty.type === "string") {
      types.push(schemaProperty.type);
    }
  }

  if (types.includes("array")) {
    /** @type {unknown[]} */
    let values;
    const trimmed = rawValue.trim();
    if (trimmed.startsWith("[")) {
      try {
        const parsed = JSON.parse(trimmed);
        if (Array.isArray(parsed)) {
          values = parsed;
        } else {
          values = [rawValue];
        }
      } catch {
        values = [rawValue];
      }
    } else if (rawValue.includes(",")) {
      values = rawValue
        .split(",")
        .map(v => v.trim())
        .filter(v => v.length > 0);
    } else {
      values = [rawValue];
    }

    if (Array.isArray(existingValue)) {
      return [...existingValue, ...values];
    }
    return values;
  }

  if (types.includes("integer") && /^-?\d+$/.test(rawValue)) {
    const parsed = Number.parseInt(rawValue, 10);
    if (Number.isSafeInteger(parsed)) {
      return parsed;
    }
  }

  if (types.includes("number")) {
    const parsed = Number(rawValue);
    if (!Number.isNaN(parsed) && Number.isFinite(parsed)) {
      return parsed;
    }
  }

  if (types.includes("boolean")) {
    const normalized = rawValue.trim().toLowerCase();
    if (normalized === "true" || normalized === "1") {
      return true;
    }
    if (normalized === "false" || normalized === "0") {
      return false;
    }
  }

  // When schema metadata is unavailable (e.g. empty tools cache), apply
  // conservative numeric coercion fallback for CLI ergonomics.
  if (allowNumericFallback && types.length === 0) {
    const trimmedValue = rawValue.trim();

    if (/^-?\d+$/.test(trimmedValue)) {
      const parsedInt = Number.parseInt(trimmedValue, 10);
      if (!Number.isNaN(parsedInt) && Number.isSafeInteger(parsedInt)) {
        return parsedInt;
      }
    }

    if (/^-?(?:(?:\d+\.\d*|\.\d+)(?:[eE][+-]?\d+)?|\d+[eE][+-]?\d+)$/.test(trimmedValue)) {
      const parsedFloat = Number.parseFloat(trimmedValue);
      if (!Number.isNaN(parsedFloat) && Number.isFinite(parsedFloat)) {
        return parsedFloat;
      }
    }
  }

  return rawValue;
}

// ---------------------------------------------------------------------------
// Tool information / help
// ---------------------------------------------------------------------------

/**
 * Load the cached tool list for a server.
 *
 * @param {string} toolsFile - Path to the JSON tools file
 * @returns {Array<{name: string, description?: string, inputSchema?: {properties?: Record<string, {description?: string, type?: string}>, required?: string[]}}>}
 */
function loadTools(toolsFile) {
  try {
    if (fs.existsSync(toolsFile)) {
      return JSON.parse(fs.readFileSync(toolsFile, "utf8"));
    }
  } catch {
    // Fall through to empty array
  }
  return [];
}

/**
 * Show top-level help: list all available commands for a server.
 *
 * @param {string} serverName - Server name
 * @param {Array<{name: string, description?: string}>} tools - Tool list
 */
function showHelp(serverName, tools) {
  const lines = [`Usage: ${serverName} <command> [options]`, ""];
  lines.push("Available commands:");
  if (tools.length > 0) {
    // Calculate column width for aligned output
    const maxNameLen = Math.max(...tools.map(t => t.name.length));
    for (const tool of tools) {
      const padded = tool.name.padEnd(maxNameLen + 2);
      lines.push(`  ${padded}${tool.description || "No description"}`);
    }
  } else {
    lines.push("  (tool list unavailable)");
  }
  lines.push("");
  lines.push(`Run '${serverName} <command> --help' for more information on a command.`);
  process.stdout.write(lines.join("\n") + "\n");
}

/**
 * Show help for a specific tool.
 *
 * @param {string} serverName - Server name
 * @param {string} toolName - Tool name
 * @param {Array<{name: string, description?: string, inputSchema?: {properties?: Record<string, {description?: string, type?: string}>, required?: string[]}}>} tools
 */
function showToolHelp(serverName, toolName, tools) {
  const tool = tools.find(t => t.name === toolName);
  if (!tool) {
    process.stderr.write(`Error: Unknown command '${toolName}'\n`);
    process.stderr.write(`Run '${serverName} --help' to see available commands.\n`);
    process.exitCode = 1;
    return;
  }

  const lines = [`Command: ${toolName}`, `Description: ${tool.description || "No description"}`];

  const props = tool.inputSchema?.properties;
  if (props && Object.keys(props).length > 0) {
    lines.push("");
    lines.push("Options:");
    const maxKeyLen = Math.max(...Object.keys(props).map(k => k.length));
    for (const [key, val] of Object.entries(props)) {
      const padded = `--${key}`.padEnd(maxKeyLen + 4);
      lines.push(`  ${padded}${val.description || val.type || "string"}`);
    }

    const required = tool.inputSchema?.required;
    if (required && required.length > 0) {
      lines.push("");
      lines.push(`Required: ${required.join(", ")}`);
    }
  }

  process.stdout.write(lines.join("\n") + "\n");
}

// ---------------------------------------------------------------------------
// Response formatting
// ---------------------------------------------------------------------------

/**
 * Extract JSON-RPC messages from a response body that may be:
 * - A JSON object
 * - A JSON string
 * - Server-Sent Events (SSE) payload containing multiple `data:` lines
 *
 * @param {unknown} responseBody
 * @returns {unknown[]}
 */
function extractJSONRPCMessages(responseBody) {
  if (responseBody == null) {
    return [];
  }

  if (Array.isArray(responseBody)) {
    return responseBody;
  }

  if (typeof responseBody === "object") {
    return [responseBody];
  }

  if (typeof responseBody !== "string") {
    return [];
  }

  const trimmed = responseBody.trim();
  if (!trimmed) {
    return [];
  }

  try {
    return [JSON.parse(trimmed)];
  } catch {
    // Fall through to SSE parsing.
  }

  /** @type {unknown[]} */
  const messages = [];
  for (const line of trimmed.split(/\r?\n/)) {
    if (!line.startsWith("data:")) {
      continue;
    }
    const payload = line.slice(5).trim();
    if (!payload || payload === "[DONE]") {
      continue;
    }
    try {
      messages.push(JSON.parse(payload));
    } catch {
      // Ignore non-JSON SSE data lines.
    }
  }

  return messages;
}

/**
 * Render MCP progress notifications to stderr.
 *
 * @param {unknown[]} messages - Parsed JSON-RPC message stream
 */
function renderProgressMessages(messages) {
  for (const message of messages) {
    if (!message || typeof message !== "object" || !("method" in message) || message.method !== "notifications/progress") {
      continue;
    }

    const params = "params" in message && message.params && typeof message.params === "object" ? message.params : null;
    if (!params) {
      continue;
    }

    const progressText = "message" in params && params.message ? String(params.message) : "";
    const progress = "progress" in params && typeof params.progress === "number" ? params.progress : null;
    const total = "total" in params && typeof params.total === "number" ? params.total : null;

    if (progressText) {
      process.stderr.write(progressText + "\n");
      continue;
    }

    if (progress != null && total != null) {
      process.stderr.write(`Progress: ${progress}/${total}\n`);
      continue;
    }

    if (progress != null) {
      process.stderr.write(`Progress: ${progress}\n`);
      continue;
    }

    process.stderr.write(`Progress: ${JSON.stringify(params)}\n`);
  }
}

/**
 * @param {unknown} message
 * @returns {boolean}
 */
function isErrorMessage(message) {
  return !!(message && typeof message === "object" && "error" in message);
}

/**
 * @param {unknown} message
 * @returns {boolean}
 */
function isResultMessage(message) {
  return !!(message && typeof message === "object" && "result" in message);
}

/**
 * Format and display the MCP tool call response.
 *
 * @param {unknown} responseBody - Parsed JSON-RPC response body
 * @param {string} serverName - Server name (for logging)
 */
function formatResponse(responseBody, serverName) {
  const core = global.core;
  const messages = extractJSONRPCMessages(responseBody);
  renderProgressMessages(messages);

  const resp = messages.find(isErrorMessage) || messages.find(isResultMessage) || responseBody;

  // Check for JSON-RPC error
  if (resp && typeof resp === "object" && "error" in resp && resp.error && typeof resp.error === "object") {
    const errRecord = resp.error;
    const message = "message" in errRecord ? String(errRecord.message || "Unknown error") : "Unknown error";
    const code = "code" in errRecord && errRecord.code != null ? String(errRecord.code) : "";
    const errText = code ? `Error [${code}]: ${message}` : `Error: ${message}`;
    process.stderr.write(errText + "\n");
    core.error(`[${serverName}] Tool call error: ${errText}`);
    auditLog(serverName, { event: "tool_error", error: errText });
    process.exitCode = 1;
    return;
  }

  // Extract result content
  if (resp && typeof resp === "object" && "result" in resp && resp.result && typeof resp.result === "object") {
    const result = resp.result;
    const isErrorResult = "isError" in result && result.isError === true;
    if ("content" in result && Array.isArray(result.content)) {
      const outputParts = [];
      for (const item of result.content) {
        const entry = /** @type {Record<string, unknown>} */ item;
        if (entry.type === "text") {
          outputParts.push(String(entry.text));
        } else if (entry.type === "image") {
          outputParts.push(`[image data - ${String(entry.mimeType || "unknown")}]`);
        } else {
          outputParts.push(JSON.stringify(entry));
        }
      }
      const output = outputParts.join("\n");
      if (isErrorResult) {
        process.stderr.write(output + "\n");
        core.error(`[${serverName}] Tool returned isError=true: ${output.length} chars`);
        auditLog(serverName, { event: "tool_error", error: output });
        process.exitCode = 1;
      } else {
        process.stdout.write(output + "\n");
        core.info(`[${serverName}] Tool output: ${output.length} chars`);
      }
      return;
    }
    // Fallback: print raw result
    const resultStr = typeof result === "string" ? result : JSON.stringify(result);
    if (isErrorResult) {
      process.stderr.write(resultStr + "\n");
      core.error(`[${serverName}] Tool returned isError=true`);
      auditLog(serverName, { event: "tool_error", error: resultStr });
      process.exitCode = 1;
    } else {
      process.stdout.write(resultStr + "\n");
    }
    return;
  }

  // Fallback: print raw response
  const rawStr = typeof resp === "string" ? resp : JSON.stringify(resp);
  process.stdout.write(rawStr + "\n");
}

// ---------------------------------------------------------------------------
// Main entry point
// ---------------------------------------------------------------------------

async function main() {
  const core = global.core;
  const { serverName, serverUrl, toolsFile, apiKey, userArgs } = parseBridgeArgs(process.argv);

  if (!serverName || !serverUrl) {
    core.setFailed("mcp_cli_bridge: --server-name and --server-url are required");
    return;
  }

  ensureAuditDir();

  core.info(`[${serverName}] Bridge invoked: url=${serverUrl}, toolsFile=${toolsFile}, userArgs=${JSON.stringify(userArgs)}`);
  auditLog(serverName, {
    event: "bridge_invoked",
    url: serverUrl,
    toolsFile,
    userArgs,
    pid: process.pid,
  });

  // Load cached tools for help display
  const tools = loadTools(toolsFile);

  // Route: --help or no args → show top-level help
  if (userArgs.length === 0 || userArgs[0] === "--help" || userArgs[0] === "-h") {
    core.info(`[${serverName}] Showing top-level help (${tools.length} tools)`);
    auditLog(serverName, { event: "show_help", toolCount: tools.length });
    showHelp(serverName, tools);
    return;
  }

  const toolName = userArgs[0];
  const toolUserArgs = userArgs.slice(1);

  // Route: <command> --help → show command-specific help
  if (toolUserArgs.length > 0 && (toolUserArgs[0] === "--help" || toolUserArgs[0] === "-h")) {
    core.info(`[${serverName}] Showing help for tool '${toolName}'`);
    auditLog(serverName, { event: "show_tool_help", tool: toolName });
    showToolHelp(serverName, toolName, tools);
    return;
  }

  // Route: <command> [--param value ...] → call tool via MCP
  const matchedTool = tools.find(tool => tool && typeof tool === "object" && tool.name === toolName);
  const schemaProperties = matchedTool && matchedTool.inputSchema && matchedTool.inputSchema.properties ? matchedTool.inputSchema.properties : {};
  const { args: toolArgs, json: jsonOutput } = parseToolArgs(toolUserArgs, schemaProperties);

  core.info(`[${serverName}] Calling tool '${toolName}' with args: ${JSON.stringify(toolArgs)}${jsonOutput ? " (--json)" : ""}`);
  auditLog(serverName, { event: "call_start", tool: toolName, arguments: toolArgs });

  const callStartMs = Date.now();
  /** @type {(() => void) | null} */
  let stopKeepalive = null;

  try {
    // MCP session protocol: initialize → notifications/initialized → tools/call
    const sessionId = await mcpInitialize(serverUrl, apiKey, serverName);
    await mcpNotifyInitialized(serverUrl, apiKey, sessionId, serverName);
    stopKeepalive = startMcpKeepalivePings(serverUrl, apiKey, sessionId, serverName);
    const resp = await mcpToolsCall(serverUrl, apiKey, sessionId, toolName, toolArgs, serverName);

    const totalMs = Date.now() - callStartMs;
    core.info(`[${serverName}] Tool call complete: total=${totalMs}ms`);
    auditLog(serverName, { event: "call_complete", tool: toolName, totalElapsedMs: totalMs });

    if (jsonOutput) {
      // --json: print the raw JSON-RPC response body
      process.stdout.write(JSON.stringify(resp.body, null, 2) + "\n");
    } else {
      formatResponse(resp.body, serverName);
    }
  } catch (err) {
    const totalMs = Date.now() - callStartMs;
    const message = err instanceof Error ? err.message : String(err);
    core.error(`[${serverName}] Tool call failed (${totalMs}ms): ${message}`);
    auditLog(serverName, {
      event: "call_error",
      tool: toolName,
      error: message,
      totalElapsedMs: totalMs,
    });
    process.stderr.write(`Error: ${message}\n`);
    process.exitCode = 1;
  } finally {
    stopKeepalive?.();
  }
}

if (require.main === module) {
  main().catch(err => {
    const core = global.core;
    const message = err instanceof Error ? err.stack || err.message : String(err);
    core.error(`mcp_cli_bridge fatal: ${message}`);
    process.exitCode = 1;
  });
}

module.exports = {
  parseToolArgs,
  coerceToolArgValue,
  extractJSONRPCMessages,
  renderProgressMessages,
  formatResponse,
  main,
};
