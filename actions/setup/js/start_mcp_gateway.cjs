// @ts-check
"use strict";

/**
 * start_mcp_gateway.cjs
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
  console.log(`⏱️  TIMING: ${label} took ${elapsed}ms`);
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
    const stat = fs.lstatSync(p);
    if (stat.isSymbolicLink()) {
      console.error(`ERROR: ${p} is a symlink — possible symlink attack, aborting`);
      process.exit(1);
    }
  } catch {
    // Path does not exist yet – that's fine.
  }
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
    console.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND must be set (command-based execution is not supported per MCP Gateway Specification v1.0.0)");
    process.exit(1);
  }

  // -----------------------------------------------------------------------
  // Create directories
  // -----------------------------------------------------------------------
  fs.mkdirSync("/tmp/gh-aw/mcp-logs", { recursive: true });

  // Symlink attack prevention on /tmp/gh-aw and /tmp/gh-aw/mcp-config
  assertNotSymlink("/tmp/gh-aw");
  assertNotSymlink("/tmp/gh-aw/mcp-config");
  fs.mkdirSync("/tmp/gh-aw/mcp-config", { recursive: true });
  // Post-creation check
  assertNotSymlink("/tmp/gh-aw");
  assertNotSymlink("/tmp/gh-aw/mcp-config");
  fs.chmodSync("/tmp/gh-aw/mcp-config", 0o700);

  // -----------------------------------------------------------------------
  // Validate container syntax
  // -----------------------------------------------------------------------
  if (!/^docker run/.test(dockerCommand)) {
    console.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND has incorrect syntax");
    console.error("Expected: docker run command with image and arguments");
    console.error(`Got: ${dockerCommand}`);
    process.exit(1);
  }
  if (!/-i/.test(dockerCommand)) {
    console.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND must include -i flag for interactive mode");
    process.exit(1);
  }
  if (!/--rm/.test(dockerCommand)) {
    console.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND must include --rm flag for cleanup");
    process.exit(1);
  }
  if (!/--network/.test(dockerCommand)) {
    console.error("ERROR: MCP_GATEWAY_DOCKER_COMMAND must include --network flag for networking");
    process.exit(1);
  }

  // -----------------------------------------------------------------------
  // Read MCP configuration from stdin
  // -----------------------------------------------------------------------
  const scriptStartTime = nowMs();

  console.log("Reading MCP configuration from stdin...");
  const configReadStart = nowMs();
  const mcpConfig = fs.readFileSync(0, "utf8"); // fd 0 = stdin
  printTiming(configReadStart, "Configuration read from stdin");
  console.log("");

  // Log configuration for debugging
  console.log("-------START MCP CONFIG-----------");
  console.log(mcpConfig);
  console.log("-------END MCP CONFIG-----------");
  console.log("");

  // -----------------------------------------------------------------------
  // Validate JSON
  // -----------------------------------------------------------------------
  const configValidationStart = nowMs();
  /** @type {Record<string, unknown>} */
  let configObj;
  try {
    configObj = JSON.parse(mcpConfig);
  } catch (err) {
    console.error("ERROR: Configuration is not valid JSON");
    console.error("");
    console.error("JSON validation error:");
    console.error(/** @type {Error} */ err.message);
    console.error("");
    console.error("Configuration content:");
    const lines = mcpConfig.split("\n");
    console.error(lines.slice(0, 50).join("\n"));
    if (lines.length > 50) {
      console.error("... (truncated, showing first 50 lines)");
    }
    process.exit(1);
  }

  // Validate gateway section
  console.log("Validating gateway configuration...");
  const gw = /** @type {Record<string, unknown> | undefined} */ configObj.gateway;
  if (!gw) {
    console.error("ERROR: Configuration is missing required 'gateway' section");
    console.error("Per MCP Gateway Specification v1.0.0 section 4.1.3, the gateway section is required");
    process.exit(1);
  }
  if (gw.port == null) {
    console.error("ERROR: Gateway configuration is missing required 'port' field");
    process.exit(1);
  }
  if (gw.domain == null) {
    console.error("ERROR: Gateway configuration is missing required 'domain' field");
    process.exit(1);
  }
  if (gw.apiKey == null) {
    console.error("ERROR: Gateway configuration is missing required 'apiKey' field");
    process.exit(1);
  }

  console.log("Configuration validated successfully");
  printTiming(configValidationStart, "Configuration validation");
  console.log("");

  // -----------------------------------------------------------------------
  // Start gateway container
  // -----------------------------------------------------------------------
  const logDir = "/tmp/gh-aw/mcp-logs/";
  const outputPath = "/tmp/gh-aw/mcp-config/gateway-output.json";
  const stderrLogPath = "/tmp/gh-aw/mcp-logs/stderr.log";

  console.log(`Starting gateway with container: ${dockerCommand}`);
  console.log(`Full docker command: ${dockerCommand}`);
  console.log("");

  const gatewayStartTime = nowMs();

  // Split docker command into args, respecting simple quoting
  const args = dockerCommand.match(/(?:[^\s"']+|"[^"]*"|'[^']*')+/g) || [];
  const cmd = /** @type {string} */ args.shift();

  const outputFd = fs.openSync(outputPath, "w", 0o600);
  const stderrFd = fs.openSync(stderrLogPath, "w", 0o600);

  const child = spawn(cmd, args, {
    stdio: ["pipe", outputFd, stderrFd],
    env: { ...process.env, MCP_GATEWAY_LOG_DIR: logDir },
    detached: true,
  });

  // Write configuration to stdin then close
  child.stdin.write(mcpConfig);
  child.stdin.end();

  // Allow the child to run independently
  child.unref();

  const gatewayPid = child.pid;
  if (!gatewayPid) {
    console.error("ERROR: Failed to start gateway container");
    process.exit(1);
  }

  console.log(`Gateway started with PID: ${gatewayPid}`);
  printTiming(gatewayStartTime, "Gateway container launch");
  console.log("Verifying gateway process is running...");

  if (isProcessAlive(gatewayPid)) {
    console.log(`Gateway process confirmed running (PID: ${gatewayPid})`);
  } else {
    console.error("ERROR: Gateway process exited immediately after start");
    console.error("");
    console.error("Gateway stdout output:");
    try {
      console.error(fs.readFileSync(outputPath, "utf8"));
    } catch {
      console.error("No stdout output available");
    }
    console.error("");
    console.error("Gateway stderr logs:");
    try {
      console.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      console.error("No stderr logs available");
    }
    process.exit(1);
  }
  console.log("");

  // -----------------------------------------------------------------------
  // Wait for gateway to initialise
  // -----------------------------------------------------------------------
  console.log("Waiting for gateway to initialize...");
  await sleep(5000);
  console.log("Checking if gateway process is still alive after initialization...");

  if (!isProcessAlive(gatewayPid)) {
    console.error(`ERROR: Gateway process (PID: ${gatewayPid}) exited during initialization`);
    console.error("");
    console.error("Gateway stdout (errors are written here per MCP Gateway Specification):");
    try {
      console.error(fs.readFileSync(outputPath, "utf8"));
    } catch {
      console.error("No stdout output available");
    }
    console.error("");
    console.error("Gateway stderr logs (debug output):");
    try {
      console.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      console.error("No stderr logs available");
    }
    process.exit(1);
  }
  console.log(`Gateway process is still running (PID: ${gatewayPid})`);
  console.log("");

  // -----------------------------------------------------------------------
  // Health check loop
  // -----------------------------------------------------------------------
  console.log("Waiting for gateway to be ready...");
  const healthCheckStart = nowMs();
  const healthHost = "localhost";
  const healthUrl = `http://${healthHost}:${gatewayPort}/health`;

  console.log(`Health endpoint: ${healthUrl}`);
  console.log(`(Note: MCP_GATEWAY_DOMAIN is '${gatewayDomain}' for container access)`);
  console.log("Retrying up to 120 times with 1s delay (120s total timeout)");
  console.log("");

  const maxRetries = 120;
  let httpCode = 0;
  let healthBody = "";
  let succeeded = false;

  console.log("=== Health Check Progress ===");
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    const elapsedSec = Math.floor((nowMs() - healthCheckStart) / 1000);
    if (attempt % 10 === 1 || attempt === 1) {
      console.log(`Attempt ${attempt}/${maxRetries} (${elapsedSec}s elapsed)...`);
    }

    try {
      const res = await httpGet(healthUrl, 2000);
      httpCode = res.statusCode;
      healthBody = res.body;
      if (httpCode === 200 && healthBody) {
        console.log(`✓ Health check succeeded on attempt ${attempt} (${elapsedSec}s elapsed)`);
        succeeded = true;
        break;
      }
    } catch {
      // Connection refused / timeout – retry
    }

    if (attempt < maxRetries) {
      await sleep(1000);
    }
  }
  console.log("=== End Health Check Progress ===");
  console.log("");

  console.log(`Final HTTP code: ${httpCode}`);
  console.log(`Total attempts: ${maxRetries}`);
  if (healthBody) {
    console.log(`Health response body: ${healthBody}`);
  } else {
    console.log("Health response body: (empty)");
  }

  if (succeeded) {
    console.log("Gateway is ready!");
    printTiming(healthCheckStart, "Health check wait");
  } else {
    console.error("");
    console.error("ERROR: Gateway failed to become ready");
    console.error(`Last HTTP code: ${httpCode}`);
    console.error(`Last health response: ${healthBody || "(empty)"}`);
    console.error("");
    console.error("Checking if gateway process is still alive...");
    if (isProcessAlive(gatewayPid)) {
      console.error(`Gateway process (PID: ${gatewayPid}) is still running`);
    } else {
      console.error(`Gateway process (PID: ${gatewayPid}) has exited`);
    }
    console.error("");
    console.error("Docker container status:");
    try {
      execSync("docker ps -a 2>/dev/null | head -20", { stdio: "inherit" });
    } catch {
      console.error("Could not list docker containers");
    }
    console.error("");
    console.error("Gateway stdout (errors are written here per MCP Gateway Specification):");
    try {
      console.error(fs.readFileSync(outputPath, "utf8"));
    } catch {
      console.error("No stdout output available");
    }
    console.error("");
    console.error("Gateway stderr logs (debug output):");
    try {
      console.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      console.error("No stderr logs available");
    }
    console.error("");
    console.error("Checking network connectivity to gateway port...");
    try {
      execSync(`netstat -tlnp 2>/dev/null | grep ":${gatewayPort}" || ss -tlnp 2>/dev/null | grep ":${gatewayPort}" || echo "Port ${gatewayPort} does not appear to be listening"`, { stdio: "inherit" });
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
  console.log("");

  // -----------------------------------------------------------------------
  // Wait for gateway output (rewritten configuration)
  // -----------------------------------------------------------------------
  console.log("Reading gateway output configuration...");
  const outputWaitStart = nowMs();
  const waitAttempts = 10;
  for (let i = 0; i < waitAttempts; i++) {
    try {
      const stat = fs.statSync(outputPath);
      if (stat.size > 0) {
        console.log("Gateway output received!");
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
  console.log("");

  // Verify output was written
  let outputSize = 0;
  try {
    outputSize = fs.statSync(outputPath).size;
  } catch {
    // file doesn't exist
  }
  if (outputSize === 0) {
    console.error("ERROR: Gateway did not write output configuration");
    console.error("");
    console.error("Gateway stdout (should contain error or config):");
    try {
      console.error(fs.readFileSync(outputPath, "utf8"));
    } catch {
      console.error("No stdout output available");
    }
    console.error("");
    console.error("Gateway stderr logs:");
    try {
      console.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      console.error("No stderr logs available");
    }
    try {
      process.kill(gatewayPid);
    } catch {
      // ignore
    }
    process.exit(1);
  }

  // Restrict permissions
  fs.chmodSync(outputPath, 0o600);

  // Check for error payload
  const gatewayOutput = JSON.parse(fs.readFileSync(outputPath, "utf8"));
  if (gatewayOutput.error) {
    console.error("ERROR: Gateway returned an error payload instead of configuration");
    console.error("");
    console.error("Gateway error details:");
    console.error(JSON.stringify(gatewayOutput, null, 2));
    console.error("");
    console.error("Gateway stderr logs:");
    try {
      console.error(fs.readFileSync(stderrLogPath, "utf8"));
    } catch {
      console.error("No stderr logs available");
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
  console.log("Converting gateway configuration to agent format...");
  const configConvertStart = nowMs();
  process.env.MCP_GATEWAY_OUTPUT = outputPath;

  // Validate MCP_GATEWAY_API_KEY
  if (!apiKey) {
    console.error("ERROR: MCP_GATEWAY_API_KEY environment variable must be set for converter scripts");
    console.error("This variable should be set in the workflow before calling start_mcp_gateway.cjs");
    process.exit(1);
  }

  // Determine engine type
  let engineType = process.env.GH_AW_ENGINE || "";
  if (!engineType) {
    if (fs.existsSync("/home/runner/.copilot") || process.env.GITHUB_COPILOT_CLI_MODE) {
      engineType = "copilot";
    } else if (fs.existsSync("/tmp/gh-aw/mcp-config/config.toml")) {
      engineType = "codex";
    } else if (fs.existsSync("/tmp/gh-aw/mcp-config/mcp-servers.json")) {
      engineType = "claude";
    } else {
      engineType = "unknown";
    }
  }

  console.log(`Detected engine type: ${engineType}`);

  const converters = {
    copilot: "convert_gateway_config_copilot.cjs",
    codex: "convert_gateway_config_codex.cjs",
    claude: "convert_gateway_config_claude.cjs",
    gemini: "convert_gateway_config_gemini.cjs",
  };

  const converterFile = converters[/** @type {keyof typeof converters} */ engineType];
  if (converterFile) {
    console.log(`Using ${engineType} converter...`);
    const converterPath = path.join(runnerTemp || "", "gh-aw/actions", converterFile);
    execSync(`node "${converterPath}"`, { stdio: "inherit", env: process.env });
  } else {
    console.log(`No agent-specific converter found for engine: ${engineType}`);
    console.log("Using gateway output directly");
    // Default fallback – copy to most common location, filtering CLI-mounted servers
    fs.mkdirSync("/home/runner/.copilot", { recursive: true });
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
        fs.writeFileSync("/home/runner/.copilot/mcp-config.json", JSON.stringify(filtered, null, 2), { mode: 0o600 });
      } catch {
        console.error("ERROR: Failed to filter CLI-mounted servers from agent MCP config");
        console.log("Falling back to unfiltered config");
        fs.copyFileSync(outputPath, "/home/runner/.copilot/mcp-config.json");
      }
    } else {
      fs.copyFileSync(outputPath, "/home/runner/.copilot/mcp-config.json");
    }
    console.log(fs.readFileSync("/home/runner/.copilot/mcp-config.json", "utf8"));
  }
  printTiming(configConvertStart, "Configuration conversion");
  console.log("");

  // -----------------------------------------------------------------------
  // Check MCP server functionality
  // -----------------------------------------------------------------------
  console.log("Checking MCP server functionality...");
  const mcpCheckStart = nowMs();
  const checkScript = path.join(runnerTemp || "", "gh-aw/actions/check_mcp_servers.sh");

  if (fs.existsSync(checkScript)) {
    console.log("Running MCP server checks...");
    // Store diagnostics in /tmp/gh-aw/mcp-logs/start-gateway.log
    try {
      execSync(`bash "${checkScript}" "${outputPath}" "http://localhost:${gatewayPort}" "${apiKey}" 2>&1 | tee /tmp/gh-aw/mcp-logs/start-gateway.log`, { stdio: "inherit" });
    } catch {
      console.error("ERROR: MCP server checks failed - no servers could be connected");
      console.error("Gateway process will be terminated");
      try {
        process.kill(gatewayPid);
      } catch {
        // ignore
      }
      process.exit(1);
    }
    printTiming(mcpCheckStart, "MCP server connectivity checks");
  } else {
    console.log(`WARNING: MCP server check script not found at ${checkScript}`);
    console.log("Skipping MCP server functionality checks");
  }
  console.log("");

  // -----------------------------------------------------------------------
  // Save CLI manifest for mount_mcp_as_cli.cjs
  // -----------------------------------------------------------------------
  console.log("Saving MCP CLI manifest...");
  fs.mkdirSync("/tmp/gh-aw/mcp-cli", { recursive: true });

  try {
    const gwOut = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    if (gwOut.mcpServers && typeof gwOut.mcpServers === "object") {
      const servers = Object.entries(/** @type {Record<string, Record<string, unknown>>} */ gwOut.mcpServers)
        .filter(([, v]) => typeof v.url === "string")
        .map(([name, v]) => ({ name, url: v.url }));
      const manifest = JSON.stringify({ servers }, null, 2);
      fs.writeFileSync("/tmp/gh-aw/mcp-cli/manifest.json", manifest, {
        mode: 0o600,
      });
      console.log(`CLI manifest saved with ${servers.length} server(s)`);
    } else {
      console.log("WARNING: No mcpServers in gateway output, CLI manifest not created");
    }
  } catch {
    console.log("WARNING: No mcpServers in gateway output, CLI manifest not created");
  }
  console.log("");

  // -----------------------------------------------------------------------
  // Delete gateway configuration file
  // -----------------------------------------------------------------------
  console.log("Cleaning up gateway configuration file...");
  try {
    fs.unlinkSync(outputPath);
    console.log("Gateway configuration file deleted");
  } catch {
    console.log("Gateway configuration file not found (already deleted or never created)");
  }
  console.log("");

  // -----------------------------------------------------------------------
  // Summary
  // -----------------------------------------------------------------------
  console.log("MCP gateway is running:");
  console.log(`  - From host: http://localhost:${gatewayPort}`);
  console.log(`  - From containers: http://${gatewayDomain}:${gatewayPort}`);
  console.log(`Gateway PID: ${gatewayPid}`);

  printTiming(scriptStartTime, "Overall gateway startup");
  console.log("");

  // -----------------------------------------------------------------------
  // Write GitHub Actions step outputs
  // -----------------------------------------------------------------------
  if (githubOutput) {
    const outputs = [`gateway-pid=${gatewayPid}`, `gateway-port=${gatewayPort}`, `gateway-api-key=${apiKey}`, `gateway-domain=${gatewayDomain}`].join("\n");
    fs.appendFileSync(githubOutput, outputs + "\n");
  }
}

main().catch(err => {
  console.error("FATAL:", err);
  process.exit(1);
});

module.exports = {};
