// @ts-check
/// <reference types="@actions/github-script" />

/**
 * mount_mcp_as_cli.cjs
 *
 * Mounts MCP servers as local CLI tools by reading the manifest written by
 * start_mcp_gateway.cjs, querying each server for its tool list, and generating
 * a standalone bash wrapper script per server in ${RUNNER_TEMP}/gh-aw/mcp-cli/bin/.
 *
 * The bin directory is locked (chmod 555) so the agent cannot modify or inject
 * scripts. The directory is added to PATH via core.addPath().
 *
 * Scripts are placed under ${RUNNER_TEMP}/gh-aw/ (not /tmp/gh-aw/) so they are
 * accessible inside the AWF sandbox, which mounts ${RUNNER_TEMP}/gh-aw read-only.
 *
 * Generated CLI wrapper usage:
 *   <server> --help                         Show all available commands
 *   <server> <command> --help               Show help for a specific command
 *   <server> <command> [--param value ...]  Execute a command
 */

const fs = require("fs");
const http = require("http");
const path = require("path");

const MANIFEST_FILE = path.join(process.env.RUNNER_TEMP || "/home/runner/work/_temp", "gh-aw/mcp-cli/manifest.json");
// Use RUNNER_TEMP so the bin and tools directories are inside the AWF sandbox mount
// (AWF mounts ${RUNNER_TEMP}/gh-aw read-only; /tmp/gh-aw is not accessible inside AWF)
const RUNNER_TEMP = process.env.RUNNER_TEMP || "/home/runner/work/_temp";
const CLI_BIN_DIR = `${RUNNER_TEMP}/gh-aw/mcp-cli/bin`;
const TOOLS_DIR = `${RUNNER_TEMP}/gh-aw/mcp-cli/tools`;

/** MCP servers that are handled differently and should not be user-facing CLIs.
 *  Note: safeoutputs and mcpscripts are NOT excluded — they are always CLI-mounted
 *  when mount-as-clis is enabled. */
const INTERNAL_SERVERS = new Set(["github"]);

/** Default timeout (ms) for HTTP calls to the local MCP gateway */
const DEFAULT_HTTP_TIMEOUT_MS = 15000;

/**
 * Validate that a server name is safe to use as a filename and in shell scripts.
 * Prevents path traversal, shell metacharacter injection, and other abuse.
 *
 * @param {string} name - Server name from the manifest
 * @returns {boolean} true if the name is safe
 */
function isValidServerName(name) {
  // Only allow alphanumeric, hyphen, underscore — no dots, slashes, spaces, etc.
  return /^[a-zA-Z0-9_-]+$/.test(name) && name.length > 0 && name.length <= 64;
}

/**
 * Escape a string for safe embedding inside double-quoted shell strings.
 * Handles the characters that are special inside double quotes: $ ` \ " !
 * Also strips newlines and carriage returns to prevent line injection.
 *
 * @param {string} str - Raw string
 * @returns {string} Escaped string safe for use inside double-quoted shell strings
 */
function shellEscapeDoubleQuoted(str) {
  return str.replace(/[\r\n]/g, "").replace(/[\\"$`!]/g, "\\$&");
}

/**
 * Rewrite a raw gateway manifest URL to use the container-accessible domain.
 *
 * The manifest stores raw gateway-output URLs (e.g., http://0.0.0.0:80/mcp/server)
 * that work from the host. Inside the AWF sandbox the gateway is reachable via
 * MCP_GATEWAY_DOMAIN:MCP_GATEWAY_PORT (typically host.docker.internal:80).
 *
 * @param {string} rawUrl - URL from the manifest (host-accessible)
 * @returns {string} URL suitable for use inside AWF containers
 */
function toContainerUrl(rawUrl) {
  const domain = process.env.MCP_GATEWAY_DOMAIN;
  const port = process.env.MCP_GATEWAY_PORT;
  if (domain && port) {
    return rawUrl.replace(/^https?:\/\/[^/]+\/mcp\//, `http://${domain}:${port}/mcp/`);
  }
  return rawUrl;
}

/**
 * Make an HTTP POST request with a JSON body and return the parsed response.
 *
 * @param {string} urlStr - Full URL to POST to
 * @param {Record<string, string>} headers - Request headers
 * @param {unknown} body - Request body (will be JSON-serialized)
 * @param {number} [timeoutMs=DEFAULT_HTTP_TIMEOUT_MS] - Request timeout in milliseconds.
 *   `initialize` and `tools/list` calls are expected to be fast (local gateway),
 *   but tool invocations may take longer; callers can override as needed.
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

    req.on("error", err => {
      reject(err);
    });

    req.setTimeout(timeoutMs, () => {
      // req.destroy() is the correct modern API; req.abort() is deprecated since Node.js v14
      req.destroy();
      reject(new Error(`HTTP request timed out after ${timeoutMs}ms`));
    });

    req.write(bodyStr);
    req.end();
  });
}

/**
 * Query the tools list from an MCP server via JSON-RPC.
 * Follows the standard MCP handshake: initialize → notifications/initialized → tools/list.
 *
 * @param {string} serverUrl - HTTP URL of the MCP server endpoint
 * @param {string} apiKey - Bearer token for gateway authentication
 * @param {typeof import("@actions/core")} core - GitHub Actions core
 * @returns {Promise<Array<{name: string, description?: string, inputSchema?: unknown}>>}
 */
async function fetchMCPTools(serverUrl, apiKey, core) {
  const authHeaders = { Authorization: apiKey };

  // Step 1: initialize – establish the session and capture Mcp-Session-Id if present
  let sessionHeader = {};
  try {
    const initResp = await httpPostJSON(
      serverUrl,
      authHeaders,
      {
        jsonrpc: "2.0",
        id: 1,
        method: "initialize",
        params: {
          capabilities: {},
          clientInfo: { name: "mcp-cli-mount", version: "1.0.0" },
          protocolVersion: "2024-11-05",
        },
      },
      DEFAULT_HTTP_TIMEOUT_MS
    );
    const sessionId = initResp.headers["mcp-session-id"];
    if (sessionId && typeof sessionId === "string") {
      sessionHeader = { "Mcp-Session-Id": sessionId };
    }
  } catch (err) {
    core.warning(`  initialize failed for ${serverUrl}: ${err instanceof Error ? err.message : String(err)}`);
    return [];
  }

  // Step 2: notifications/initialized – required by MCP spec to complete the handshake.
  // The server responds with 204 No Content; errors here are non-fatal.
  try {
    await httpPostJSON(serverUrl, { ...authHeaders, ...sessionHeader }, { jsonrpc: "2.0", method: "notifications/initialized", params: {} }, 10000);
  } catch (err) {
    core.warning(`  notifications/initialized failed for ${serverUrl}: ${err instanceof Error ? err.message : String(err)}`);
  }

  // Step 3: tools/list – get the available tool definitions
  try {
    const listResp = await httpPostJSON(serverUrl, { ...authHeaders, ...sessionHeader }, { jsonrpc: "2.0", id: 2, method: "tools/list" }, DEFAULT_HTTP_TIMEOUT_MS);
    const respBody = /** @type {Record<string, unknown>} */ listResp.body;
    if (respBody && respBody.result && typeof respBody.result === "object") {
      const result = /** @type {Record<string, unknown>} */ respBody.result;
      if (Array.isArray(result.tools)) {
        return /** @type {Array<{name: string, description?: string, inputSchema?: unknown}>} */ result.tools;
      }
    }
    return [];
  } catch (err) {
    core.warning(`  tools/list failed for ${serverUrl}: ${err instanceof Error ? err.message : String(err)}`);
    return [];
  }
}

/**
 * Generate the bash wrapper script content for a given MCP server.
 * The generated script is a self-contained CLI that delegates all calls
 * to the MCP gateway via curl, following the proper MCP session protocol:
 * initialize → notifications/initialized → tools/call.
 *
 * The gateway API key is baked directly into the generated script at
 * generation time because MCP_GATEWAY_API_KEY is excluded from the AWF
 * sandbox environment (--exclude-env MCP_GATEWAY_API_KEY) and would not
 * be accessible to the agent at runtime.
 *
 * @param {string} serverName - Name of the MCP server
 * @param {string} serverUrl - HTTP URL of the MCP server endpoint
 * @param {string} toolsFile - Path to the cached tools JSON file
 * @param {string} apiKey - Gateway API key, baked into the script at generation time
 * @returns {string} Content of the bash wrapper script
 */
function generateCLIWrapperScript(serverName, serverUrl, toolsFile, apiKey) {
  // Sanitize all values that are embedded in the shell script to prevent injection.
  // Server names are pre-validated by isValidServerName(), but we still escape all
  // values for defense-in-depth.
  const safeName = shellEscapeDoubleQuoted(serverName);
  const safeUrl = shellEscapeDoubleQuoted(serverUrl);
  const safeToolsFile = shellEscapeDoubleQuoted(toolsFile);
  const safeApiKey = shellEscapeDoubleQuoted(apiKey);

  return `#!/usr/bin/env bash
# MCP CLI wrapper for: ${safeName}
# Auto-generated by gh-aw. Do not modify.
#
# Usage:
#   ${safeName} --help                        Show all available commands
#   ${safeName} <command> --help              Show help for a specific command
#   ${safeName} <command> [--param value...]  Execute a command

SERVER_NAME="${safeName}"
SERVER_URL="${safeUrl}"
TOOLS_FILE="${safeToolsFile}"

# API key is baked in at generation time; MCP_GATEWAY_API_KEY is not available
# inside the AWF sandbox (excluded via --exclude-env MCP_GATEWAY_API_KEY).
API_KEY="${safeApiKey}"

load_tools() {
  if [ -f "\$TOOLS_FILE" ]; then
    cat "\$TOOLS_FILE"
  else
    echo "[]"
  fi
}

show_help() {
  local tools
  tools=\$(load_tools)
  echo "Usage: \$SERVER_NAME <command> [options]"
  echo ""
  echo "Available commands:"
  if command -v jq &>/dev/null && echo "\$tools" | jq -e "length > 0" >/dev/null 2>&1; then
    echo "\$tools" | jq -r ".[] | \\"  \(.name)\\t\(.description // \\"No description\\")\\"" \\
      | column -t -s $'\\t' 2>/dev/null \\
      || echo "\$tools" | jq -r ".[] | \\"  \(.name)  \(.description // \\"\\")\\""
  else
    echo "  (tool list unavailable)"
  fi
  echo ""
  echo "Run '\$SERVER_NAME <command> --help' for more information on a command."
}

show_tool_help() {
  local tool_name="\$1"
  local tools tool
  tools=\$(load_tools)
  tool=\$(echo "\$tools" | jq -r ".[] | select(.name == \\"\$tool_name\\")" 2>/dev/null || echo "")

  if [ -z "\$tool" ]; then
    echo "Error: Unknown command '"\$tool_name"'" >&2
    echo "Run '"\$SERVER_NAME" --help' to see available commands." >&2
    exit 1
  fi

  echo "Command: \$tool_name"
  echo "Description: \$(echo "\$tool" | jq -r ".description // \\"No description\\"")"

  local has_props
  has_props=\$(echo "\$tool" | jq -r "has(\\"inputSchema\\") and ((.inputSchema.properties // {}) | length > 0)")

  if [ "\$has_props" = "true" ]; then
    echo ""
    echo "Options:"
    echo "\$tool" | jq -r ".inputSchema.properties | to_entries[] | \\"  --\(.key)  \(.value.description // .value.type // \\"string\\")\\"" 
    local required
    required=\$(echo "\$tool" | jq -r "(.inputSchema.required // []) | join(\\", \\")")
    if [ -n "\$required" ]; then
      echo ""
      echo "Required: \$required"
    fi
  fi
}

call_tool() {
  local tool_name="\$1"
  shift

  local args="{}"
  while [[ \$# -gt 0 ]]; do
    if [[ "\$1" == --* ]]; then
      local key="\${1#--}"
      if [[ \$# -ge 2 && "\$2" != --* ]]; then
        local val="\$2"
        args=\$(echo "\$args" | jq --arg k "\$key" --arg v "\$val" '. + {($k): $v}')
        shift 2
      else
        args=\$(echo "\$args" | jq --arg k "\$key" '. + {($k): true}')
        shift 1
      fi
    else
      shift
    fi
  done

  # MCP session protocol: initialize → notifications/initialized → tools/call
  # A separate headers file is used to capture the Mcp-Session-Id without mixing
  # headers and body (curl -i mixes them, making parsing fragile).
  local headers_file
  headers_file=\$(mktemp)

  # Step 1: initialize – establish the session
  curl -s -D "\$headers_file" --max-time 30 -X POST "\$SERVER_URL" \\
    -H "Authorization: \$API_KEY" \\
    -H "Content-Type: application/json" \\
    -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"capabilities":{},"clientInfo":{"name":"mcp-cli","version":"1.0.0"},"protocolVersion":"2024-11-05"}}' \\
    >/dev/null 2>/dev/null || true

  local session_id
  session_id=\$(grep -i "^mcp-session-id:" "\$headers_file" 2>/dev/null | awk '{print $2}' | tr -d "\\r" || echo "")
  rm -f "\$headers_file"

  local session_header_args=()
  if [ -n "\$session_id" ]; then
    session_header_args=(-H "Mcp-Session-Id: \$session_id")
  fi

  # Step 2: notifications/initialized – required by MCP spec to complete the handshake.
  # The server responds with 204 No Content; failures here are non-fatal.
  curl -s --max-time 10 -X POST "\$SERVER_URL" \\
    -H "Authorization: \$API_KEY" \\
    -H "Content-Type: application/json" \\
    "\${session_header_args[@]}" \\
    -d '{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}' \\
    >/dev/null 2>/dev/null || true

  # Step 3: tools/call – execute the tool within the established session
  local request
  request=\$(jq -n --arg name "\$tool_name" --argjson args "\$args" \\
    '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":$name,"arguments":$args}}')

  local response
  response=\$(curl -s --max-time 120 -X POST "\$SERVER_URL" \\
    -H "Authorization: \$API_KEY" \\
    -H "Content-Type: application/json" \\
    "\${session_header_args[@]}" \\
    -d "\$request" \\
    2>/dev/null)

  if echo "\$response" | jq -e ".error" >/dev/null 2>&1; then
    local err_msg err_code
    err_msg=\$(echo "\$response" | jq -r ".error.message // \\"Unknown error\\"")
    err_code=\$(echo "\$response" | jq -r ".error.code // \\"\\""  )
    if [ -n "\$err_code" ]; then
      echo "Error [\$err_code]: \$err_msg" >&2
    else
      echo "Error: \$err_msg" >&2
    fi
    exit 1
  fi

  if echo "\$response" | jq -e ".result.content" >/dev/null 2>&1; then
    echo "\$response" | jq -r '.result.content[] |
      if .type == "text" then .text
      elif .type == "image" then "[image data - \(.mimeType // "unknown")]"
      else (. | tostring)
      end'
  elif echo "\$response" | jq -e ".result" >/dev/null 2>&1; then
    echo "\$response" | jq -r ".result"
  else
    echo "\$response"
  fi
}

if [[ \$# -eq 0 || "\$1" == "--help" || "\$1" == "-h" ]]; then
  show_help
  exit 0
fi

COMMAND="\$1"
shift

if [[ "\$1" == "--help" || "\$1" == "-h" ]]; then
  show_tool_help "\$COMMAND"
  exit 0
fi

call_tool "\$COMMAND" "\$@"
`;
}

/**
 * Mount MCP servers as CLI tools by reading the manifest and generating wrapper scripts.
 *
 * @returns {Promise<void>}
 */
async function main() {
  const core = global.core;

  core.info("Mounting MCP servers as CLI tools...");

  if (!fs.existsSync(MANIFEST_FILE)) {
    core.info("No MCP CLI manifest found, skipping CLI mounting");
    return;
  }

  /** @type {{servers: Array<{name: string, url: string}>}} */
  let manifest;
  try {
    manifest = JSON.parse(fs.readFileSync(MANIFEST_FILE, "utf8"));
  } catch (err) {
    core.warning(`Failed to read MCP CLI manifest: ${err instanceof Error ? err.message : String(err)}`);
    return;
  }

  const servers = (manifest.servers || []).filter(s => !INTERNAL_SERVERS.has(s.name));

  if (servers.length === 0) {
    core.info("No user-facing MCP servers in manifest, skipping CLI mounting");
    return;
  }

  core.info(`Found ${servers.length} user-facing server(s) in manifest (after filtering internal: ${[...INTERNAL_SERVERS].join(", ")})`);

  fs.mkdirSync(CLI_BIN_DIR, { recursive: true });
  fs.mkdirSync(TOOLS_DIR, { recursive: true });

  const apiKey = process.env.MCP_GATEWAY_API_KEY || "";
  if (!apiKey) {
    core.warning("MCP_GATEWAY_API_KEY is not set; generated CLI wrappers will not be able to authenticate with the gateway");
  }

  const gatewayDomain = process.env.MCP_GATEWAY_DOMAIN || "";
  const gatewayPort = process.env.MCP_GATEWAY_PORT || "";
  if (!gatewayDomain || !gatewayPort) {
    core.warning("MCP_GATEWAY_DOMAIN or MCP_GATEWAY_PORT is not set; CLI wrappers will use raw manifest URLs which may not be reachable inside the AWF sandbox");
  }

  const mountedServers = [];
  const skippedServers = [];

  for (const server of servers) {
    const { name, url } = server;

    // Validate server name to prevent path traversal and shell injection.
    // Server names become filenames in CLI_BIN_DIR and are embedded in shell scripts.
    if (!isValidServerName(name)) {
      core.warning(`Skipping server '${name}': name contains invalid characters (only alphanumeric, hyphen, underscore allowed)`);
      skippedServers.push(name);
      continue;
    }

    // Validate URL format
    try {
      new URL(url);
    } catch {
      core.warning(`Skipping server '${name}': invalid URL '${url}'`);
      skippedServers.push(name);
      continue;
    }
    // The manifest URL is the host-accessible raw gateway address (e.g., http://0.0.0.0:80/mcp/server).
    // Rewrite it to the container-accessible URL for the generated CLI wrapper scripts,
    // which run inside the AWF sandbox where the gateway is reached via MCP_GATEWAY_DOMAIN.
    const containerUrl = toContainerUrl(url);
    core.info(`Mounting MCP server '${name}' (host url: ${url}, container url: ${containerUrl})...`);

    const toolsFile = path.join(TOOLS_DIR, `${name}.json`);

    // Query tools from the server using the host-accessible URL (mount step runs on host)
    const tools = await fetchMCPTools(url, apiKey, core);
    core.info(`  Found ${tools.length} tool(s)`);

    // Cache the tool list
    try {
      fs.writeFileSync(toolsFile, JSON.stringify(tools, null, 2), { mode: 0o644 });
    } catch (err) {
      core.warning(`  Failed to write tools cache for ${name}: ${err instanceof Error ? err.message : String(err)}`);
    }

    // Write the CLI wrapper script using the container-accessible URL
    const scriptPath = path.join(CLI_BIN_DIR, name);
    try {
      fs.writeFileSync(scriptPath, generateCLIWrapperScript(name, containerUrl, toolsFile, apiKey), { mode: 0o755 });
      mountedServers.push(name);
      core.info(`  ✓ Mounted as: ${scriptPath}`);
    } catch (err) {
      core.warning(`  Failed to write CLI wrapper for ${name}: ${err instanceof Error ? err.message : String(err)}`);
    }
  }

  if (mountedServers.length === 0) {
    core.info("No MCP servers were successfully mounted as CLI tools");
    return;
  }

  // Lock the bin directory so the agent cannot modify or inject scripts
  try {
    fs.chmodSync(CLI_BIN_DIR, 0o555);
    core.info(`CLI bin directory locked (read-only): ${CLI_BIN_DIR}`);
  } catch (err) {
    core.warning(`Failed to lock CLI bin directory: ${err instanceof Error ? err.message : String(err)}`);
  }

  // Add the bin directory to PATH for subsequent steps
  core.addPath(CLI_BIN_DIR);

  core.info("");
  core.info(`Successfully mounted ${mountedServers.length} MCP server(s) as CLI tools:`);
  for (const name of mountedServers) {
    core.info(`  - ${name}`);
  }
  if (skippedServers.length > 0) {
    core.warning(`Skipped ${skippedServers.length} server(s) due to validation errors: ${skippedServers.join(", ")}`);
  }
  core.info(`CLI bin directory added to PATH: ${CLI_BIN_DIR}`);
  core.setOutput("mounted-servers", mountedServers.join(","));
}

module.exports = { main, fetchMCPTools, generateCLIWrapperScript, isValidServerName, shellEscapeDoubleQuoted };
