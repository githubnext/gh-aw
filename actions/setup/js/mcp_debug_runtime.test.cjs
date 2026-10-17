import { afterEach, describe, expect, it } from "vitest";
import { spawnSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const directories = [];

function runtimeDirectory() {
  const directory = fs.mkdtempSync(path.join(process.cwd(), ".mcp-debug-test-"));
  directories.push(directory);
  return directory;
}

function runScript(script, debug, env = {}, input = "") {
  const result = spawnSync(process.execPath, ["-e", script], {
    cwd: process.cwd(),
    env: { ...process.env, DEBUG: debug, ACTIONS_RUNNER_DEBUG: "", RUNNER_DEBUG: "", ...env },
    encoding: "utf8",
    input,
    timeout: 10000,
  });
  expect(result.error).toBeUndefined();
  expect(result.status, result.stderr).toBe(0);
  return result;
}

afterEach(() => {
  for (const directory of directories.splice(0)) fs.rmSync(directory, { recursive: true, force: true });
});

describe("MCP runtime DEBUG selection", () => {
  it.each([
    ["", false],
    ["unrelated:*", false],
    ["safeoutputs", true],
    ["safe*", true],
    ["*", true],
    ["*,-safeoutputs", false],
    ["-safe*,*", false],
  ])("gates safe-outputs stderr and server.log without suppressing outputs with DEBUG=%s", (debug, enabled) => {
    const directory = runtimeDirectory();
    const logDir = path.join(directory, "logs");
    const configPath = path.join(directory, "config.json");
    const toolsPath = path.join(directory, "tools.json");
    const outputPath = path.join(directory, "outputs.jsonl");
    const message = "No repository changes were required after reviewing this event.";
    fs.writeFileSync(configPath, JSON.stringify({ noop: { max: 1 } }));
    fs.writeFileSync(toolsPath, JSON.stringify([{ name: "noop", description: "No changes needed", inputSchema: { type: "object", properties: { message: { type: "string" } }, required: ["message"] } }]));
    const { stderr, stdout } = runScript(
      `
        require("./safe_outputs_mcp_server.cjs").startSafeOutputsServer();
      `,
      debug,
      { GH_AW_MCP_LOG_DIR: logDir, GH_AW_SAFE_OUTPUTS_CONFIG_PATH: configPath, GH_AW_SAFE_OUTPUTS_TOOLS_PATH: toolsPath, GH_AW_SAFE_OUTPUTS: outputPath },
      JSON.stringify({ jsonrpc: "2.0", id: 1, method: "tools/call", params: { name: "noop", arguments: { message } } }) + "\n"
    );
    expect(JSON.parse(stdout).result.isError).toBe(false);
    expect(JSON.parse(fs.readFileSync(outputPath, "utf8"))).toMatchObject({ type: "noop", message });
    expect(stderr.includes("[safeoutputs] Loading safe-outputs configuration")).toBe(enabled);
    if (debug === "" || debug === "unrelated:*") expect(stderr).toBe("");
    if (!enabled) expect(stderr).not.toContain("[safeoutputs]");
    expect(fs.existsSync(logDir)).toBe(enabled);
    if (enabled) expect(fs.readFileSync(path.join(logDir, "server.log"), "utf8")).toContain("[safeoutputs] Loading safe-outputs configuration");
  });

  it.each(["", "unrelated:*", "*,-safeoutputs"])("preserves error summaries and absolute-handler audits with DEBUG=%s", debug => {
    const logDir = path.join(runtimeDirectory(), "logs");
    const handler = path.resolve("mcp_logger.cjs");
    const { stderr } = runScript(
      `
        const { createServer, loadToolHandlers } = require("./mcp_server_core.cjs");
        const server = createServer(
          { name: "safeoutputs", version: "1.0.0" },
          { logDir: process.env.TEST_LOG_DIR, redactLogMessage: message => message.replaceAll("private-value", "[redacted]") }
        );
        server.debug("suppressed diagnostic");
        server.debugError("Failed: ", new Error("private-value"));
        server.replyError(null, -32603, "private-value");
        loadToolHandlers(server, [{ name: "example", handler: process.env.TEST_HANDLER }], process.cwd());
      `,
      debug,
      { TEST_LOG_DIR: logDir, TEST_HANDLER: handler }
    );
    const fileLog = fs.readFileSync(path.join(logDir, "server.log"), "utf8");
    for (const output of [stderr, fileLog]) {
      expect(output).toContain("Failed: [redacted]");
      expect(output).toContain("Error for notification: [redacted]");
      expect(output).toContain("Using absolute path (bypasses basePath validation)");
      expect(output).not.toContain("suppressed diagnostic");
      expect(output).not.toContain("private-value");
      expect(output).not.toContain("Stack trace:");
    }
  });

  it.each([
    ["", false],
    ["unrelated:*", false],
    ["mcpscripts", true],
    ["mcp*", true],
    ["*", true],
    ["*,-mcpscripts,-mcp_server_core,-mcp_http_transport", false],
    ["-mcp*,*", false],
  ])("gates actual MCP-scripts HTTP initialization stderr with DEBUG=%s", (debug, enabled) => {
    const configPath = path.join(runtimeDirectory(), "config.json");
    fs.writeFileSync(configPath, JSON.stringify({ tools: [] }));
    const { stderr, stdout } = runScript(
      `
        const { createMCPServer } = require("./mcp_scripts_mcp_server_http.cjs");
        const { server } = createMCPServer(process.env.TEST_CONFIG);
        server.handleRequest({ jsonrpc: "2.0", id: 1, method: "initialize", params: {} })
          .then(response => process.stdout.write(JSON.stringify(response)));
      `,
      debug,
      { TEST_CONFIG: configPath }
    );
    expect(JSON.parse(stdout).result.serverInfo.name).toBe("mcpscripts");
    expect(stderr.includes("[mcpscripts] === Creating MCP Server ===")).toBe(enabled);
    if (!enabled) expect(stderr).toBe("");
    expect(fs.existsSync(configPath)).toBe(false);
  });

  it("keeps HTTP startup failures visible without DEBUG", () => {
    const { stderr } = runScript(
      `
        const { logStartupError } = require("./mcp_http_server_runner.cjs");
        try { logStartupError(new Error("startup failed"), "mcp-scripts-startup-error", {}); }
        catch {}
      `,
      ""
    );
    expect(stderr).toContain("FATAL ERROR: Failed to start HTTP Server: startup failed");
    expect(stderr).not.toContain("Stack trace");
  });
});
