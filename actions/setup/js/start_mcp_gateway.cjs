// @ts-check
"use strict";

// Ensures global.core is available when running outside github-script context
require("./shim.cjs");

/**
 * start_mcp_gateway.cjs
 *
 * @safe-outputs-exempt SEC-004: "body" references are local healthcheck response payloads, not user-authored comment bodies
 *
 * Starts the MCP gateway process that proxies MCP servers through a unified HTTP endpoint.
 * Following the MCP Gateway Specification:
 *   https://github.com/github/gh-aw/blob/main/docs/src/content/docs/reference/mcp-gateway.md
 * Per MCP Gateway Specification v1.0.0: Only container-based execution is supported.
 *
 * This script reads the MCP configuration from stdin and pipes it to the gateway container.
 *
 * Required environment variables:
 * - MCP_GATEWAY_DOCKER_COMMAND: Container image to run (required)
 * - MCP_GATEWAY_API_KEY: API key for gateway authentication (required for converter scripts)
 * - MCP_GATEWAY_PORT: Port for MCP gateway
 * - MCP_GATEWAY_DOMAIN: Domain for MCP server URLs (e.g., host.docker.internal)
 * - RUNNER_TEMP: GitHub Actions runner temp directory
 * - GITHUB_OUTPUT: Path to GitHub Actions output file
 *
 * Optional:
 * - GH_AW_ENGINE: Engine type (copilot, codex, claude, gemini)
 * - GH_AW_MCP_CLI_SERVERS: JSON array of server names to exclude from agent config
 */

const { spawn, execSync } = require("child_process");
const fs = require("fs");
const http = require("http");
const path = require("path");
const { withRetry } = require("./error_recovery.cjs");
const { lstatGuard } = require("./symlink_guard.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

// ---------------------------------------------------------------------------
// Timing helpers
// ---------------------------------------------------------------------------

function nowMs() {
  return Date.now();
}

/**
 * @param {number} startMs
 * @param {string} label
 */
function printTiming(startMs, label) {
  const elapsed = nowMs() - startMs;
  core.info(`⏱️  TIMING: ${label} took ${elapsed}ms`);
}

// ---------------------------------------------------------------------------
// Utility helpers
// ---------------------------------------------------------------------------

/**
 * @param {number} ms
 * @returns {Promise<void>}
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Builds targeted context for a JSON.parse error so logs can point at the likely key.
 *
 * @param {string} jsonText
 * @param {string} parseErrorMessage
 * @returns {{line: number, column: number, lineText: string, key: string | null} | null}
 */
function getJSONParseErrorContext(jsonText, parseErrorMessage) {
  const posMatch = parseErrorMessage.match(/position\s+(\d+)/i);
  if (!posMatch) {
    return null;
  }

  const pos = Number(posMatch[1]);
  if (!Number.isFinite(pos) || pos < 0) {
    return null;
  }

  const safePos = Math.min(pos, Math.max(0, jsonText.length - 1));
  const before = jsonText.slice(0, safePos);
  const line = before.split("\n").length;
  const lineStart = before.lastIndexOf("\n") + 1;
  const lineEnd = jsonText.indexOf("\n", safePos);
  const resolvedLineEnd = lineEnd === -1 ? jsonText.length : lineEnd;
  const lineText = jsonText.slice(lineStart, resolvedLineEnd);
  const column = safePos - lineStart + 1;
  const keyMatch = lineText.match(/"([^"]+)"\s*:/);
  const key = keyMatch ? keyMatch[1] : null;

  return { line, column, lineText, key };
}

/**
 * Normalizes GH_AW_OTLP_IF_MISSING to a supported mode.
 * @param {string | undefined} value
 * @returns {"error" | "warn" | "ignore"}
 */
function getOTLPIfMissingMode(value) {
  const normalized = (value || "").trim().toLowerCase();
  if (normalized === "warn" || normalized === "ignore") {
    return normalized;
  }
  return "error";
}

/**
 * Returns true when GH_AW_OTLP_IF_MISSING is set to "ignore".
 * @param {string | undefined} value
 * @returns {boolean}
 */
function isOTLPIfMissingIgnore(value) {
  return getOTLPIfMissingMode(value) === "ignore";
}

/**
 * Returns true when OTLP headers contain at least one non-empty value.
 * NOTE: Unexpected primitive values are treated as non-empty so they are
 * preserved for downstream validation/error handling.
 * @param {unknown} headers
 * @returns {boolean}
 */
function hasNonEmptyOTLPHeaders(headers) {
  if (headers == null) {
    return false;
  }
  if (typeof headers === "string") {
    return headers.trim() !== "";
  }
  if (Array.isArray(headers)) {
    return headers.some(item => hasNonEmptyOTLPHeaders(item));
  }
  if (typeof headers === "object") {
    return Object.values(headers).some(value => hasNonEmptyOTLPHeaders(value));
  }
  // For unexpected primitive types, keep headers
  // intact so downstream validation can fail explicitly rather than silently
  // treating them as "missing".
  return true;
}

/**
 * Apply observability.otlp.if-missing: ignore behavior to gateway OTLP config.
 * When enabled, missing endpoint values are downgraded to warnings and OTLP
 * gateway configuration is skipped instead of hard-failing the workflow.
 *
 * @param {Record<string, unknown>} configObj
 */
function applyOTLPIgnoreIfMissing(configObj) {
  const mode = getOTLPIfMissingMode(process.env.GH_AW_OTLP_IF_MISSING);
  const shouldDowngradeMissing = mode === "warn" || mode === "ignore";
  const shouldWarn = mode === "warn";
  if (!shouldDowngradeMissing) {
    return;
  }
  const gw = configObj.gateway;
  if (!gw || typeof gw !== "object" || Array.isArray(gw)) {
    return;
  }
  const gateway = /** @type {Record<string, unknown>} */ gw;
  const otel = gateway["opentelemetry"];
  if (!otel || typeof otel !== "object" || Array.isArray(otel)) {
    return;
  }
  const otelConfig = /** @type {Record<string, unknown>} */ otel;
  const endpoint = typeof otelConfig["endpoint"] === "string" ? otelConfig["endpoint"].trim() : "";
  if (!endpoint) {
    delete gateway["opentelemetry"];
    if (shouldWarn) {
      core.warning("OTLP endpoint is missing/empty and GH_AW_OTLP_IF_MISSING=warn; skipping MCP gateway OTLP configuration.");
    }
    return;
  }
  const headers = otelConfig["headers"];
  if ("headers" in otelConfig && !hasNonEmptyOTLPHeaders(headers)) {
    delete otelConfig["headers"];
    if (shouldWarn) {
      core.warning("OTLP headers are missing/empty and GH_AW_OTLP_IF_MISSING=warn; continuing without MCP gateway OTLP headers.");
    }
  }
}

/**
 * Check whether a process is alive.
 * @param {number} pid
 * @returns {boolean}
 */
function isProcessAlive(pid) {
  try {
    process.kill(pid, 0);
    return true;
  } catch {
    return false;
  }
}

/**
 * HTTP GET helper – returns { statusCode, body }.
 * @param {string} url
 * @param {number} timeoutMs
 * @returns {Promise<{ statusCode: number; body: string }>}
 */
function httpGet(url, timeoutMs) {
  return new Promise((resolve, reject) => {
    const req = http.get(url, { timeout: timeoutMs }, res => {
      let data = "";
      res.on("data", chunk => (data += chunk));
      res.on("end", () => resolve({ statusCode: res.statusCode || 0, body: data }));
    });
    req.on("timeout", () => {
      req.destroy();
      reject(new Error("timeout"));
    });
    req.on("error", reject);
  });
}

// ---------------------------------------------------------------------------
// Validation helpers
// ---------------------------------------------------------------------------

/**
 * Validate that a path is not a symlink (symlink attack prevention).
 * @param {string} p
 */
function assertNotSymlink(p) {
  try {
    if (lstatGuard(p) === null) {
      core.error(`ERROR: ${p} is a symlink — possible symlink attack, aborting`);
      process.exit(1);
    }
  } catch {
    // Path does not exist yet – that's fine.
  }
}

/**
 * Resolves the Copilot CLI config directory and the MCP config file path from
 * the runtime $HOME. The Copilot CLI uses ~/.copilot, which is
 * /home/runner/.copilot on standard GitHub-hosted runners (HOME=/home/runner)
 * but may differ on self-hosted or containerized runners. HOME is a standard
 * POSIX environment variable inherited from the runner's parent process and
 * passed through to shell steps; other generators (copilot_mcp.go,
 * copilot_engine_execution.go) rely on it the same way.
 *
 * Exported for testability; throws Error rather than exiting so tests can
 * exercise the missing-HOME branch.
 *
 * @returns {{ dir: string, file: string }}
 */
function resolveCopilotConfigPaths() {
  const home = process.env.HOME;
  if (!home) {
    throw new Error("HOME environment variable is not set; cannot locate Copilot CLI config directory");
  }
  const dir = path.join(home, ".copilot");
  return { dir, file: path.join(dir, "mcp-config.json") };
}

/**
 * Determines which engine-specific MCP config converter to use.
 * Copilot auto-detection consults HOME only when HOME is available so
 * explicit non-Copilot engines do not fail just because HOME is unset.
 *
 * @param {string} configDir
 * @param {NodeJS.ProcessEnv} [env]
 * @param {(p: string) => boolean} [existsSync]
 * @returns {string}
 */
function detectEngineType(configDir, env = process.env, existsSync = fs.existsSync) {
  if (env.GH_AW_ENGINE) {
    return env.GH_AW_ENGINE;
  }
  if (env.GITHUB_COPILOT_CLI_MODE) {
    return "copilot";
  }
  if (env.HOME && existsSync(path.join(env.HOME, ".copilot"))) {
    return "copilot";
  }
  if (existsSync(path.join(configDir, "config.toml"))) {
    return "codex";
  }
  if (existsSync(path.join(configDir, "mcp-servers.json"))) {
    return "claude";
  }
  return "unknown";
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

async function main() {
  // Restrict default file creation mode to owner-only (rw-------)
  process.umask(0o077);

  const dockerCommand = process.env.MCP_GATEWAY_DOCKER_COMMAND;
  const apiKey = process.env.MCP_GATEWAY_API_KEY;
  const gatewayPort = process.env.MCP_GATEWAY_PORT;
  const gatewayDomain = process.env.MCP_GATEWAY_DOMAIN;
  const runnerTemp = process.env.RUNNER_TEMP;
  const githubOutput = process.env.GITHUB_OUTPUT;

  // -----------------------------------------------------------------------
  // Validate required env vars
  // -----------------------------------------------------------------------
  if (!dockerCommand) {
    core.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND must be set (command-based execution is not supported per MCP Gateway Specification v1.0.0)");
    process.exit(1);
  }

  // Validate port is numeric to prevent injection in shell commands and URLs
  if (gatewayPort && !/^\d+$/.test(gatewayPort)) {
    core.error(`ERROR: MCP_GATEWAY_PORT must be a numeric value, got: '${gatewayPort}'`);
    process.exit(1);
  }

  core.info("=== MCP Gateway Startup ===");
  core.info(`Engine: ${process.env.GH_AW_ENGINE || "(auto-detect)"}`);
  core.info(`Runner temp: ${runnerTemp || "(not set)"}`);
  core.info(`Gateway port: ${gatewayPort || "(not set)"}`);
  core.info(`Gateway domain: ${gatewayDomain || "(not set)"}`);
  core.info("");

  // -----------------------------------------------------------------------
  // Create directories
  // -----------------------------------------------------------------------
  // Config and CLI manifest are stored under ${RUNNER_TEMP}/gh-aw/ to prevent
  // tampering.  RUNNER_TEMP is a per-runner directory that is not
  // world-writable, unlike /tmp.  Logs remain under /tmp/gh-aw/mcp-logs/
  // because the Docker gateway container mounts -v /tmp:/tmp:rw and writes
  // there via MCP_GATEWAY_LOG_DIR.
  const configDir = path.join(runnerTemp || "/tmp", "gh-aw/mcp-config");
  const cliDir = path.join(runnerTemp || "/tmp", "gh-aw/mcp-cli");

  try {
    fs.mkdirSync("/tmp/gh-aw/mcp-logs", { recursive: true });
  } catch (err) {
    throw new Error(`Failed to create directory /tmp/gh-aw/mcp-logs: ${String(err)}`, { cause: err });
  }

  // Symlink attack prevention on the config directory
  assertNotSymlink(configDir);
  try {
    fs.mkdirSync(configDir, { recursive: true });
  } catch (err) {
    throw new Error(`Failed to create directory ${configDir}: ${String(err)}`, { cause: err });
  }
  // Post-creation check
  assertNotSymlink(configDir);
  try {
    fs.chmodSync(configDir, 0o700);
  } catch (err) {
    throw new Error(`Failed to set permissions on ${configDir}: ${String(err)}`, { cause: err });
  }

  // -----------------------------------------------------------------------
  // Validate container syntax
  // -----------------------------------------------------------------------
  if (!/^docker run/.test(dockerCommand)) {
    core.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND has incorrect syntax");
    core.error("Expected: docker run command with image and arguments");
    core.error(`Got: ${dockerCommand}`);
    process.exit(1);
  }
  if (!/-i/.test(dockerCommand)) {
    core.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND must include -i flag for interactive mode");
    process.exit(1);
  }
  if (!/--rm/.test(dockerCommand)) {
    core.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND must include --rm flag for cleanup");
    process.exit(1);
  }
  if (!/--network/.test(dockerCommand)) {
    core.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND must include --network flag for networking");
    process.exit(1);
  }

  // -----------------------------------------------------------------------
  // Read MCP configuration from stdin
  // -----------------------------------------------------------------------
  const scriptStartTime = nowMs();

  core.info("Reading MCP configuration from stdin...");
  const configReadStart = nowMs();
  let mcpConfig;
  try {
    mcpConfig = fs.readFileSync(0, "utf8"); // fd 0 = stdin
  } catch (err) {
    throw new Error(`Failed to read MCP configuration from stdin: ${String(err)}`, { cause: err });
  }
  printTiming(configReadStart, "Configuration read from stdin");
  core.info("");

  // Log configuration for debugging
  core.info("-------START MCP CONFIG-----------");
  core.info(mcpConfig);
  core.info("-------END MCP CONFIG-----------");
  core.info("");

  // -----------------------------------------------------------------------
  // Validate JSON
  // -----------------------------------------------------------------------
  const configValidationStart = nowMs();
  /** @type {Record<string, unknown>} */
  let configObj;
  try {
    configObj = JSON.parse(mcpConfig);
  } catch (err) {
    core.error("ERROR: Configuration is not valid JSON");
    core.error("");
    core.error("JSON validation error:");
    const parseMessage = getErrorMessage(err);
    core.error(parseMessage);
    const parseContext = getJSONParseErrorContext(mcpConfig, parseMessage);
    if (parseContext) {
      core.error(`Likely offending location: line ${parseContext.line}, column ${parseContext.column}`);
      if (parseContext.key) {
        core.error(`Likely offending key: ${parseContext.key}`);
      }
      core.error(`Context line: ${parseContext.lineText}`);
    }
    core.error("");
    core.error("Configuration content:");
    const lines = mcpConfig.split("\n");
    core.error(lines.slice(0, 50).join("\n"));
    if (lines.length > 50) {
      core.error("... (truncated, showing first 50 lines)");
    }
    process.exit(1);
  }

  // Validate gateway section
  core.info("Validating gateway configuration...");
  const gw = configObj.gateway;
  if (!gw || typeof gw !== "object") {
    core.error("ERROR: Configuration is missing required 'gateway' section");
    core.error("Per MCP Gateway Specification v1.0.0 section 4.1.3, the gateway section is required");
    process.exit(1);
  }
  if (!("port" in gw) || gw.port == null) {
    core.error("ERROR: Gateway configuration is missing required 'port' field");
    process.exit(1);
  }
  if (!("domain" in gw) || gw.domain == null) {
    core.error("ERROR: Gateway configuration is missing required 'domain' field");
    process.exit(1);
  }
  if (!("apiKey" in gw) || gw.apiKey == null) {
    core.error("ERROR: Gateway configuration is missing required 'apiKey' field");
    process.exit(1);
  }

  applyOTLPIgnoreIfMissing(configObj);

  core.info("Configuration validated successfully");
  printTiming(configValidationStart, "Configuration validation");
  core.info("");

  // -----------------------------------------------------------------------
  // Start gateway container
  // -----------------------------------------------------------------------
  const logDir = "/tmp/gh-aw/mcp-logs/";
  const outputPath = path.join(configDir, "gateway-output.json");
  const stderrLogPath = "/tmp/gh-aw/mcp-logs/stderr.log";

  // Clean up any stale gateway container from a previous run on this runner.
  // On persistent self-hosted runners a prior job's gateway container may still
  // be running and holding the host port, causing "bind: address already in use"
  // when we try to start the new one.  Force-removing by the well-known container
  // name is idempotent: docker rm -f exits non-zero when the container doesn't
  // exist, but the trailing || true (and 2>/dev/null) make the command succeed.
  core.info("Cleaning up any stale awmg-mcpg container from a previous run...");
  try {
    execSync("docker rm -f awmg-mcpg 2>/dev/null && echo 'Removed stale awmg-mcpg container' || true", { stdio: "inherit" });
  } catch {
    // Non-fatal: proceed even if the cleanup command itself fails
    core.info("Could not remove stale awmg-mcpg container (may not exist)");
  }
  core.info("");

  core.info(`Starting gateway with container: ${dockerCommand}`);
  core.info(`Full docker command: ${dockerCommand}`);
  core.info("");

  const gatewayStartTime = nowMs();

  // Split docker command into args, respecting simple quoting
  const args = dockerCommand.match(/(?:[^\s"']+|"[^"]*"|'[^']*')+/g) || [];
  const cmd = args.shift();
  if (!cmd) {
    core.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND did not contain an executable command");
    process.exit(1);
  }

  const outputFd = fs.openSync(outputPath, "w", 0o600);
  const stderrFd = fs.openSync(stderrLogPath, "w", 0o600);

  const child = spawn(cmd, args, {
    stdio: ["pipe", outputFd, stderrFd],
    env: { ...process.env, MCP_GATEWAY_LOG_DIR: logDir },
    detached: true,
  });

  // Write configuration to stdin then close
  if (!child.stdin) {
    core.error("ERROR: Gateway process stdin is not available");
    process.exit(1);
  }
  child.stdin.write(JSON.stringify(configObj));
  child.stdin.end();

  // Allow the child to run independently
  child.unref();

  const gatewayPid = child.pid;
  if (!gatewayPid) {
    core.error("ERROR: Failed to start gateway container");
    process.exit(1);
  }

  core.info(`Gateway started with PID: ${gatewayPid}`);
  printTiming(gatewayStartTime, "Gateway container launch");
  core.info("Verifying gateway process is running...");

  if (isProcessAlive(gatewayPid)) {
    core.info(`Gateway process confirmed running (PID: ${gatewayPid})`);
  } else {
    core.error("ERROR: Gateway process exited immediately after start");
    core.error("");
    core.error("Gateway stdout output:");
    try {
      core.error(fs.readFileSync(outputPath, "utf8"));
    } catch {
      core.error("No stdout output available");
    }
    core.error("");
    core.error("Gateway stderr logs:");
    try {
      core.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      core.error("No stderr logs available");
    }
    process.exit(1);
  }
  core.info("");

  // -----------------------------------------------------------------------
  // Wait for gateway to initialise
  // -----------------------------------------------------------------------
  core.info("Waiting for gateway to initialize...");
  await sleep(5000);
  core.info("Checking if gateway process is still alive after initialization...");

  if (!isProcessAlive(gatewayPid)) {
    core.error(`ERROR: Gateway process (PID: ${gatewayPid}) exited during initialization`);
    core.error("");
    core.error("Gateway stdout (errors are written here per MCP Gateway Specification):");
    try {
      core.error(fs.readFileSync(outputPath, "utf8"));
    } catch {
      core.error("No stdout output available");
    }
    core.error("");
    core.error("Gateway stderr logs (debug output):");
    try {
      core.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      core.error("No stderr logs available");
    }
    process.exit(1);
  }
  core.info(`Gateway process is still running (PID: ${gatewayPid})`);
  core.info("");

  // -----------------------------------------------------------------------
  // Health check loop
  // -----------------------------------------------------------------------
  core.info("Waiting for gateway to be ready...");
  const healthCheckStart = nowMs();
  const healthHost = "localhost";
  const healthUrl = `http://${healthHost}:${gatewayPort}/health`;

  core.info(`Health endpoint: ${healthUrl}`);
  core.info(`(Note: MCP_GATEWAY_DOMAIN is '${gatewayDomain}' for container access)`);
  core.info("Retrying up to 120 times with exponential backoff (250ms to 1s, ~120s total timeout)");
  core.info("");

  const maxTotalAttempts = 120;
  // withRetry's maxRetries excludes the initial attempt.
  const maxRetryCount = maxTotalAttempts - 1;
  const initialRetryDelayMs = 250;
  let httpCode = 0;
  let healthBody = "";
  let succeeded = false;
  let attemptsMade = 0;

  core.info("=== Health Check Progress ===");
  try {
    await withRetry(
      async () => {
        // Counts total health-check attempts, including the final successful attempt.
        attemptsMade += 1;
        const elapsedSec = Math.floor((nowMs() - healthCheckStart) / 1000);
        if (attemptsMade % 10 === 1 || attemptsMade === 1) {
          core.info(`Attempt ${attemptsMade}/${maxTotalAttempts} (${elapsedSec}s elapsed)...`);
        }

        const res = await httpGet(healthUrl, 2000);
        httpCode = res.statusCode;
        healthBody = res.body;
        if (httpCode === 200 && healthBody) {
          core.info(`✓ Health check succeeded on attempt ${attemptsMade} (${elapsedSec}s elapsed)`);
          succeeded = true;
          return;
        }
        throw new Error(`Health endpoint not ready (HTTP ${httpCode || 0})`);
      },
      {
        maxRetries: maxRetryCount,
        initialDelayMs: initialRetryDelayMs,
        maxDelayMs: 1000,
        backoffMultiplier: 2,
        jitterMs: 0,
        // Preserve previous loop behavior: retry any health-check failure until attempts are exhausted.
        shouldRetry: () => true,
      },
      "MCP gateway health check"
    );
  } catch {
    // Retry exhaustion is handled below using existing diagnostics.
  }

  core.info("=== End Health Check Progress ===");
  core.info("");

  core.info(`Final HTTP code: ${httpCode}`);
  core.info(`Total attempts: ${attemptsMade}`);
  if (healthBody) {
    core.info(`Health response body: ${healthBody}`);
  } else {
    core.info("Health response body: (empty)");
  }

  if (succeeded) {
    core.info("Gateway is ready!");
    printTiming(healthCheckStart, "Health check wait");
  } else {
    core.error("");
    core.error("ERROR: Gateway failed to become ready");
    core.error(`Last HTTP code: ${httpCode}`);
    core.error(`Last health response: ${healthBody || "(empty)"}`);
    core.error("");
    core.error("Checking if gateway process is still alive...");
    if (isProcessAlive(gatewayPid)) {
      core.error(`Gateway process (PID: ${gatewayPid}) is still running`);
    } else {
      core.error(`Gateway process (PID: ${gatewayPid}) has exited`);
    }
    core.error("");
    core.error("Docker container status:");
    try {
      execSync("docker ps -a 2>/dev/null | head -20", { stdio: "inherit" });
    } catch {
      core.error("Could not list docker containers");
    }
    core.error("");
    core.error("Gateway stdout (errors are written here per MCP Gateway Specification):");
    try {
      core.error(fs.readFileSync(outputPath, "utf8"));
    } catch {
      core.error("No stdout output available");
    }
    core.error("");
    core.error("Gateway stderr logs (debug output):");
    try {
      core.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      core.error("No stderr logs available");
    }
    core.error("");
    core.error("Checking network connectivity to gateway port...");
    try {
      // Validate gatewayPort is numeric to prevent shell injection
      const safePort = String(gatewayPort).replace(/[^0-9]/g, "");
      execSync(`netstat -tlnp 2>/dev/null | grep ":${safePort}" || ss -tlnp 2>/dev/null | grep ":${safePort}" || echo "Port ${safePort} does not appear to be listening"`, { stdio: "inherit" });
    } catch {
      // ignore
    }
    try {
      process.kill(gatewayPid);
    } catch {
      // ignore
    }
    process.exit(1);
  }
  core.info("");

  // -----------------------------------------------------------------------
  // Wait for gateway output (rewritten configuration)
  // -----------------------------------------------------------------------
  core.info("Reading gateway output configuration...");
  const outputWaitStart = nowMs();
  const waitAttempts = 10;
  for (let i = 0; i < waitAttempts; i++) {
    try {
      const stat = fs.statSync(outputPath);
      if (stat.size > 0) {
        core.info("Gateway output received!");
        break;
      }
    } catch {
      // not ready yet
    }
    if (i < waitAttempts - 1) {
      await sleep(1000);
    }
  }
  printTiming(outputWaitStart, "Gateway output wait");
  core.info("");

  // Verify output was written
  let outputSize = 0;
  try {
    outputSize = fs.statSync(outputPath).size;
  } catch {
    // file doesn't exist
  }
  if (outputSize === 0) {
    core.error("ERROR: Gateway did not write output configuration");
    core.error("");
    core.error("Gateway stdout (should contain error or config):");
    try {
      core.error(fs.readFileSync(outputPath, "utf8"));
    } catch {
      core.error("No stdout output available");
    }
    core.error("");
    core.error("Gateway stderr logs:");
    try {
      core.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      core.error("No stderr logs available");
    }
    try {
      process.kill(gatewayPid);
    } catch {
      // ignore
    }
    process.exit(1);
  }

  // Restrict permissions
  try {
    fs.chmodSync(outputPath, 0o600);
  } catch (err) {
    throw new Error(`Failed to set permissions on ${outputPath}: ${String(err)}`, { cause: err });
  }

  // Check for error payload
  let gatewayOutput;
  try {
    gatewayOutput = JSON.parse(fs.readFileSync(outputPath, "utf8"));
  } catch (err) {
    throw new Error("Failed to parse gateway output file " + outputPath + ": " + getErrorMessage(err), { cause: err });
  }
  if (gatewayOutput.error) {
    core.error("ERROR: Gateway returned an error payload instead of configuration");
    core.error("");
    core.error("Gateway error details:");
    core.error(JSON.stringify(gatewayOutput, null, 2));
    core.error("");
    core.error("Gateway stderr logs:");
    try {
      core.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      core.error("No stderr logs available");
    }
    try {
      process.kill(gatewayPid);
    } catch {
      // ignore
    }
    process.exit(1);
  }

  // -----------------------------------------------------------------------
  // Convert gateway output to agent-specific format
  // -----------------------------------------------------------------------
  core.info("Converting gateway configuration to agent format...");
  const configConvertStart = nowMs();
  process.env.MCP_GATEWAY_OUTPUT = outputPath;

  // Validate MCP_GATEWAY_API_KEY
  if (!apiKey) {
    core.error("ERROR: MCP_GATEWAY_API_KEY environment variable must be set for converter scripts");
    core.error("This variable should be set in the workflow before calling start_mcp_gateway.cjs");
    process.exit(1);
  }

  // Determine engine type
  const engineType = detectEngineType(configDir);

  core.info(`Detected engine type: ${engineType}`);

  const converters = {
    copilot: "convert_gateway_config_copilot.cjs",
    codex: "convert_gateway_config_codex.cjs",
    claude: "convert_gateway_config_claude.cjs",
    gemini: "convert_gateway_config_gemini.cjs",
  };

  const converterFile = converters[/** @type {keyof typeof converters} */ engineType];
  if (converterFile) {
    core.info(`Using ${engineType} converter...`);
    const converterPath = path.join(runnerTemp || "", "gh-aw/actions", converterFile);
    execSync(`node "${converterPath}"`, { stdio: "inherit", env: process.env });
  } else {
    let copilotConfigDir, copilotConfigFile;
    try {
      ({ dir: copilotConfigDir, file: copilotConfigFile } = resolveCopilotConfigPaths());
    } catch (err) {
      core.error(`ERROR: ${getErrorMessage(err)}`);
      process.exit(1);
    }
    core.info(`No agent-specific converter found for engine: ${engineType}`);
    core.info("Using gateway output directly");
    // Default fallback – copy to most common location, filtering CLI-mounted servers
    try {
      fs.mkdirSync(copilotConfigDir, { recursive: true });
    } catch (err) {
      throw new Error(`Failed to create directory ${copilotConfigDir}: ${String(err)}`, { cause: err });
    }
    const cliServersRaw = process.env.GH_AW_MCP_CLI_SERVERS;
    if (cliServersRaw) {
      try {
        const cliServers = new Set(JSON.parse(cliServersRaw));
        const filtered = { ...gatewayOutput };
        if (filtered.mcpServers && typeof filtered.mcpServers === "object") {
          const servers = /** @type {Record<string, unknown>} */ filtered.mcpServers;
          for (const key of Object.keys(servers)) {
            if (cliServers.has(key)) {
              delete servers[key];
            }
          }
        }
        fs.writeFileSync(copilotConfigFile, JSON.stringify(filtered, null, 2), { mode: 0o600 });
      } catch {
        core.error("ERROR: Failed to filter CLI-mounted servers from agent MCP config");
        core.info("Falling back to unfiltered config");
        try {
          fs.copyFileSync(outputPath, copilotConfigFile);
        } catch (err) {
          throw new Error(`Failed to copy file ${outputPath} to ${copilotConfigFile}: ${String(err)}`, { cause: err });
        }
      }
    } else {
      try {
        fs.copyFileSync(outputPath, copilotConfigFile);
      } catch (err) {
        throw new Error(`Failed to copy file ${outputPath} to ${copilotConfigFile}: ${String(err)}`, { cause: err });
      }
    }
    let copilotConfigContent;
    try {
      copilotConfigContent = fs.readFileSync(copilotConfigFile, "utf8");
    } catch (err) {
      throw new Error(`Failed to read file ${copilotConfigFile}: ${String(err)}`, { cause: err });
    }
    core.info(copilotConfigContent);
  }
  printTiming(configConvertStart, "Configuration conversion");
  core.info("");

  // -----------------------------------------------------------------------
  // Check MCP server functionality
  // -----------------------------------------------------------------------
  core.info("Checking MCP server functionality...");
  const mcpCheckStart = nowMs();
  const checkScript = path.join(runnerTemp || "", "gh-aw/actions/check_mcp_servers.sh");

  if (fs.existsSync(checkScript)) {
    core.info("Running MCP server checks...");
    // Store diagnostics in /tmp/gh-aw/mcp-logs/start-gateway.log
    // Pass apiKey via MCP_GATEWAY_API_KEY env var (already set) rather than
    // as a shell argument to avoid shell metacharacter injection risks.
    const safePort = String(gatewayPort).replace(/[^0-9]/g, "");
    try {
      execSync(`bash "${checkScript}" "${outputPath}" "http://localhost:${safePort}" "$MCP_GATEWAY_API_KEY" 2>&1 | tee /tmp/gh-aw/mcp-logs/start-gateway.log`, { stdio: "inherit", env: process.env });
    } catch {
      core.error("ERROR: MCP server checks failed - no servers could be connected");
      core.error("Gateway process will be terminated");
      try {
        process.kill(gatewayPid);
      } catch {
        // ignore
      }
      process.exit(1);
    }
    printTiming(mcpCheckStart, "MCP server connectivity checks");
  } else {
    core.info(`WARNING: MCP server check script not found at ${checkScript}`);
    core.info("Skipping MCP server functionality checks");
  }
  core.info("");

  // -----------------------------------------------------------------------
  // Save CLI manifest for mount_mcp_as_cli.cjs
  // -----------------------------------------------------------------------
  core.info("Saving MCP CLI manifest...");
  try {
    fs.mkdirSync(cliDir, { recursive: true });
  } catch (err) {
    throw new Error(`Failed to create directory ${cliDir}: ${String(err)}`, { cause: err });
  }

  try {
    const gwOut = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    if (gwOut.mcpServers && typeof gwOut.mcpServers === "object") {
      const allEntries = Object.entries(/** @type {Record<string, Record<string, unknown>>} */ gwOut.mcpServers);
      const servers = allEntries
        .filter(([name, v]) => {
          if (typeof v.url !== "string") {
            core.info(`  Skipping server '${name}' from manifest: missing url`);
            return false;
          }
          // Validate server name — only alphanumeric, hyphen, underscore
          if (!/^[a-zA-Z0-9_-]+$/.test(name) || name.length > 64) {
            core.info(`  Skipping server '${name}' from manifest: invalid name`);
            return false;
          }
          return true;
        })
        .map(([name, v]) => ({ name, url: v.url }));
      const manifest = JSON.stringify({ servers }, null, 2);
      fs.writeFileSync(path.join(cliDir, "manifest.json"), manifest, {
        mode: 0o600,
      });
      core.info(`CLI manifest saved with ${servers.length} server(s): ${servers.map(s => s.name).join(", ")}`);
    } else {
      core.info("WARNING: No mcpServers in gateway output, CLI manifest not created");
    }
  } catch {
    core.info("WARNING: No mcpServers in gateway output, CLI manifest not created");
  }
  core.info("");

  // -----------------------------------------------------------------------
  // Delete gateway configuration file
  // -----------------------------------------------------------------------
  core.info("Cleaning up gateway configuration file...");
  try {
    fs.unlinkSync(outputPath);
    core.info("Gateway configuration file deleted");
  } catch {
    core.info("Gateway configuration file not found (already deleted or never created)");
  }
  core.info("");

  // -----------------------------------------------------------------------
  // Summary
  // -----------------------------------------------------------------------
  core.info("MCP gateway is running:");
  core.info(`  - From host: http://localhost:${gatewayPort}`);
  core.info(`  - From containers: http://${gatewayDomain}:${gatewayPort}`);
  core.info(`Gateway PID: ${gatewayPid}`);

  printTiming(scriptStartTime, "Overall gateway startup");
  core.info("");

  // -----------------------------------------------------------------------
  // Write GitHub Actions step outputs
  // -----------------------------------------------------------------------
  if (githubOutput) {
    const outputs = [`gateway-pid=${gatewayPid}`, `gateway-port=${gatewayPort}`, `gateway-api-key=${apiKey}`, `gateway-domain=${gatewayDomain}`].join("\n");
    try {
      fs.appendFileSync(githubOutput, outputs + "\n");
    } catch {
      /* ignore */
    }
  }
}

if (require.main === module) {
  main().catch(err => {
    const message = getErrorMessage(err);
    const stack = err instanceof Error ? err.stack : undefined;
    if (stack) core.error(stack);
    core.setFailed(`FATAL: ${message}`);
  });
}

module.exports = {
  applyOTLPIgnoreIfMissing,
  detectEngineType,
  getOTLPIfMissingMode,
  hasNonEmptyOTLPHeaders,
  isOTLPIfMissingIgnore,
  getJSONParseErrorContext,
  resolveCopilotConfigPaths,
};
