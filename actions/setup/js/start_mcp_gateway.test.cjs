import fs from "fs";
import os from "os";
import path from "path";
import http from "http";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  applyAgenticWorkflowsRunnerIdentity,
  applyOTLPIgnoreIfMissing,
  clearGatewayStartupMarker,
  createGatewayStderrPath,
  getGatewayStartupMarkerPath,
  redactGatewayDiagnostics,
  writeGatewayStartupMarker,
  customGatewayEnvNamesVar,
  customGatewayReservedEnvPrefix,
  customGatewayEnvTransportPrefix,
  detectEngineType,
  extractOptionalServerNames,
  getJSONParseErrorContext,
  getOTLPIfMissingMode,
  hasNonEmptyOTLPHeaders,
  injectCustomGatewayEnvArgs,
  computeHealthRetryBudgetMs,
  MCP_GATEWAY_BACKEND_STARTUP_TIMEOUT_DEFAULT_MS,
  MCP_GATEWAY_HEALTH_REGISTRATION_MARGIN_MS,
  MCP_GATEWAY_HEALTH_MAX_ATTEMPTS_LIMIT,
  MCP_GATEWAY_HEALTH_RETRY_DEFAULTS,
  MCP_GATEWAY_HEALTH_RETRY_ENV,
  resolveGatewayHealthRetryConfig,
  normalizeSinkVisibilityEncoding,
  resolveCopilotConfigPaths,
  validateGatewayAgentIdentifiers,
  waitForSafeOutputsReadiness,
  retryGatewayStartup,
  httpGet,
} from "./start_mcp_gateway.cjs";
import { writeMcpConfiguration } from "./mcp_configuration.cjs";

describe("agenticworkflows backend ownership", () => {
  it.each(["localhost/gh-aw:dev", "alpine:3.24"])("uses the runner identity without widening private artifact modes (%s)", container => {
    const env = { GITHUB_TOKEN: "${GITHUB_TOKEN}", GITHUB_ACTOR: "${GITHUB_ACTOR}" };
    const backend = { container, args: ["--network", "host", "-w", "/workspace"], env };
    const other = { container: "example/server", args: [] };
    const config = { mcpServers: { agenticworkflows: backend, other } };
    applyAgenticWorkflowsRunnerIdentity(config, 1001, 123);
    expect(backend.args).toEqual(["--network", "host", "-w", "/workspace", "--user", "1001:123"]);
    expect(backend.env).toEqual({ HOME: "/tmp", ...env });
    expect(other).toEqual({ container: "example/server", args: [] });
    applyAgenticWorkflowsRunnerIdentity(config, 1001, 123);
    expect(backend.args.filter(arg => arg === "--user")).toHaveLength(1);
  });

  it.each([["--user", "42:43"], ["--user=42:43"], ["-u", "42:43"], ["-u42:43"]])("preserves an explicitly configured container user: %j", (...args) => {
    const backend = { container: "example/server", args, env: { HOME: "/custom" } };
    const original = structuredClone(backend);
    applyAgenticWorkflowsRunnerIdentity({ mcpServers: { agenticworkflows: backend } }, 1001, 123);
    expect(backend).toEqual(original);
  });

  it("preserves a configured HOME and supports default container arguments", () => {
    const backend = { container: "alpine:3.24", env: { HOME: "/custom" } };
    applyAgenticWorkflowsRunnerIdentity({ mcpServers: { agenticworkflows: backend } }, 1001, 123);
    expect(backend).toEqual({ container: "alpine:3.24", args: ["--user", "1001:123"], env: { HOME: "/custom" } });
  });

  it.each([{}, { mcpServers: {} }, { mcpServers: { agenticworkflows: { url: "https://example.com/mcp" } } }])("leaves absent and HTTP backends unchanged", config => {
    const original = structuredClone(config);
    applyAgenticWorkflowsRunnerIdentity(config, 1001, 123);
    expect(config).toEqual(original);
  });

  it.each([
    [-1, 123],
    [1001, -1],
    [NaN, 123],
    [1001, Infinity],
    [1.5, 123],
  ])("rejects invalid runner identities (%s, %s)", (uid, gid) => {
    const backend = { container: "alpine:3.24" };
    expect(() => applyAgenticWorkflowsRunnerIdentity({ mcpServers: { agenticworkflows: backend } }, uid, gid)).toThrow("Runner identity is required");
    expect(backend).toEqual({ container: "alpine:3.24" });
  });

  it.each([{ args: "invalid" }, { args: [42] }, { env: [] }, { env: null }])("rejects malformed backend configuration without publishing it", fields => {
    expect(() => applyAgenticWorkflowsRunnerIdentity({ mcpServers: { agenticworkflows: { container: "alpine:3.24", ...fields } } }, 1001, 123)).toThrow(/Invalid agenticworkflows container/);
  });

  it("applies identity before serializing the configuration sent to the gateway", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source.indexOf("applyAgenticWorkflowsRunnerIdentity(configObj)")).toBeLessThan(source.indexOf("mcpConfig = JSON.stringify(configObj)"));
  });
});

describe("start_mcp_gateway logging", () => {
  it("records only a sanitized configuration snapshot before launching the gateway", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source).toContain("writeMcpConfiguration(logDir, configObj)");
    expect(source.indexOf("writeMcpConfiguration(logDir, configObj)")).toBeLessThan(source.indexOf('core.info("Starting gateway container...")'));
    expect(source.indexOf("writeMcpConfiguration(logDir, configObj)")).toBeLessThan(source.indexOf("const optionalServerNames = extractOptionalServerNames(configObj)"));
    expect(source).not.toContain("core.info(mcpConfig)");
    expect(source).not.toContain("core.error(parseMessage)");
    expect(source).not.toContain("parseContext.lineText");
    expect(source).not.toContain('core.error("Configuration content:")');
  });

  it("reads piped configuration as a Buffer before decoding UTF-8", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source).toContain('fs.readFileSync(0).toString("utf8")');
    expect(source).not.toContain('fs.readFileSync(0, "utf8")');
  });

  it("does not create the legacy MCP gateway stderr log", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source).not.toContain("/tmp/gh-aw/mcp-logs/stderr.log");
  });

  it("does not create the MCP gateway startup log", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source).not.toContain("/tmp/gh-aw/mcp-logs/start-gateway.log");
  });

  it("captures the gateway child process stderr outside the artifact directory", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source).toContain(`stdio: ["pipe", outputFd, stderrFd]`);
    expect(source).toContain(`path.join(os.tmpdir(), "gh-aw-mcp-gateway-")`);
  });

  it("exports the gateway agent ID as the AWF API key handoff", () => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    expect(source).toContain("`gateway-api-key=${agentId}`");
  });

  it.each([false, true])("times health and safe-output readiness separately when readiness fails: %s", async fails => {
    const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
    const start = source.indexOf("  if (succeeded) {");
    const end = source.indexOf("\n  } else {", start);
    const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor;
    const runSuccessBlock = new AsyncFunction(
      "{ nowMs, printTiming, waitForSafeOutputsReadiness, core, writeGatewayStartupMarker }",
      `const succeeded = true, healthCheckStart = 10, gatewayStartTime = 0;
       const configObj = {}, gatewayPort = 8080, agentId = "test", healthRetryConfig = {}, gatewayStartupMarker = "marker";
       ${source.slice(start, end)}
       }`
    );
    let clock = 30;
    const timings = [];
    const failure = new Error("readiness failed");
    const readiness = vi.fn(async () => {
      clock += 50;
      if (fails) throw failure;
    });
    const marker = vi.fn();
    const result = runSuccessBlock({
      nowMs: () => clock,
      printTiming: (startMs, label) => timings.push([label, clock - startMs]),
      waitForSafeOutputsReadiness: readiness,
      core: { info: vi.fn() },
      writeGatewayStartupMarker: marker,
    });
    if (fails) {
      await expect(result).rejects.toBe(failure);
      expect(marker).not.toHaveBeenCalled();
    } else {
      await result;
      expect(marker).toHaveBeenCalledWith("marker");
    }
    expect(timings).toEqual([
      ["Health check wait", 20],
      ["Safe outputs readiness wait", 50],
    ]);
    expect(readiness).toHaveBeenCalledWith({}, 8080, "test", {}, 0);
  });
});

describe("start_mcp_gateway health timeout", () => {
  it("allows cleanup and registration time after the backend startup timeout", () => {
    const config = resolveGatewayHealthRetryConfig({});

    expect(config).toMatchObject(MCP_GATEWAY_HEALTH_RETRY_DEFAULTS);
    expect(config.retryBudgetMs).toBe(computeHealthRetryBudgetMs(MCP_GATEWAY_HEALTH_RETRY_DEFAULTS));
    expect(config.retryBudgetMs).toBeGreaterThanOrEqual(MCP_GATEWAY_BACKEND_STARTUP_TIMEOUT_DEFAULT_MS + MCP_GATEWAY_HEALTH_REGISTRATION_MARGIN_MS);
  });

  it("applies environment overrides for every retry parameter", () => {
    const config = resolveGatewayHealthRetryConfig({
      [MCP_GATEWAY_HEALTH_RETRY_ENV.maxTotalAttempts]: "10",
      [MCP_GATEWAY_HEALTH_RETRY_ENV.initialDelayMs]: "100",
      [MCP_GATEWAY_HEALTH_RETRY_ENV.maxDelayMs]: "400",
      [MCP_GATEWAY_HEALTH_RETRY_ENV.backoffMultiplier]: "1.5",
      [MCP_GATEWAY_HEALTH_RETRY_ENV.backendStartupTimeoutMs]: "1000",
    });

    expect(config).toMatchObject({ maxTotalAttempts: 10, initialDelayMs: 100, maxDelayMs: 400, backoffMultiplier: 1.5, backendStartupTimeoutMs: 1000 });
    expect(config.retryBudgetMs).toBe(computeHealthRetryBudgetMs(config));
  });

  it("falls back to defaults for invalid overrides", () => {
    const config = resolveGatewayHealthRetryConfig({
      [MCP_GATEWAY_HEALTH_RETRY_ENV.maxTotalAttempts]: "not-a-number",
      [MCP_GATEWAY_HEALTH_RETRY_ENV.initialDelayMs]: "-5",
      [MCP_GATEWAY_HEALTH_RETRY_ENV.maxDelayMs]: "",
      [MCP_GATEWAY_HEALTH_RETRY_ENV.backoffMultiplier]: "0",
    });

    expect(config).toMatchObject(MCP_GATEWAY_HEALTH_RETRY_DEFAULTS);
  });

  it("raises the delay cap when it is below the initial delay", () => {
    const config = resolveGatewayHealthRetryConfig({
      [MCP_GATEWAY_HEALTH_RETRY_ENV.initialDelayMs]: "2000",
      [MCP_GATEWAY_HEALTH_RETRY_ENV.maxDelayMs]: "500",
    });

    expect(config.maxDelayMs).toBe(2000);
  });

  it("clamps a shrinking backoff multiplier to a constant delay", () => {
    const config = resolveGatewayHealthRetryConfig({
      [MCP_GATEWAY_HEALTH_RETRY_ENV.backoffMultiplier]: "0.5",
    });

    expect(config.backoffMultiplier).toBe(1);
  });

  it("caps an excessive attempt count", () => {
    const config = resolveGatewayHealthRetryConfig({
      [MCP_GATEWAY_HEALTH_RETRY_ENV.maxTotalAttempts]: "1000000000",
    });

    expect(config.maxTotalAttempts).toBe(MCP_GATEWAY_HEALTH_MAX_ATTEMPTS_LIMIT);
  });

  it("computes the cumulative startup retry delay", () => {
    // The retry loop multiplies before its first sleep: 500ms, then 1000ms capped.
    expect(computeHealthRetryBudgetMs({ maxTotalAttempts: 3, initialDelayMs: 250, maxDelayMs: 1000, backoffMultiplier: 2 })).toBe(1500);
  });
});

describe("start_mcp_gateway startup diagnostics", () => {
  it("creates the stderr capture file in an owner-only directory outside /tmp/gh-aw", () => {
    const stderrPath = createGatewayStderrPath();
    const dir = path.dirname(stderrPath);
    expect(dir.startsWith("/tmp/gh-aw/")).toBe(false);
    expect(fs.statSync(dir).mode & 0o777).toBe(0o700);
    fs.rmSync(dir, { recursive: true, force: true });
  });

  describe("start_mcp_gateway safeoutputs readiness", () => {
    let server;
    let originalCore;
    let requests;
    const config = { mcpServers: { safeoutputs: {} } };
    const retryConfig = {
      maxTotalAttempts: 4,
      initialDelayMs: 1,
      maxDelayMs: 1,
      backoffMultiplier: 1,
      backendStartupTimeoutMs: 1000,
      retryBudgetMs: 3,
    };

    beforeEach(() => {
      originalCore = global.core;
      global.core = { ...originalCore, info: vi.fn(), warning: vi.fn() };
      requests = [];
    });

    afterEach(async () => {
      vi.restoreAllMocks();
      vi.useRealTimers();
      if (server) {
        await new Promise(resolve => server.close(resolve));
        server = undefined;
      }
      global.core = originalCore;
    });

    async function startBackend(handleToolsList, { resetFirstInitialize = false, resetEveryInitialize = false, initStatus = 200, serverName = "safeoutputs", listResponse } = {}) {
      let initializes = 0;
      server = http.createServer((req, res) => {
        let data = "";
        req.on("data", chunk => (data += chunk));
        req.on("end", () => {
          const rpc = JSON.parse(data);
          requests.push(rpc.method);
          expect(req.url).toBe(`/mcp/${serverName}`);
          expect(req.headers.authorization).toBe("readiness-test-key");
          if (rpc.method === "initialize") {
            initializes += 1;
            if (resetEveryInitialize || (resetFirstInitialize && initializes === 1)) {
              req.socket.destroy();
              return;
            }
            res.writeHead(initStatus === 503 && initializes > 1 ? 200 : initStatus, { "Content-Type": "application/json", "Mcp-Session-Id": "readiness-test-session" });
            res.end(JSON.stringify({ jsonrpc: "2.0", id: rpc.id, result: {} }));
          } else {
            expect(req.headers["mcp-session-id"]).toBe("readiness-test-session");
            if (rpc.method === "tools/list") {
              res.writeHead(200, { "Content-Type": "application/json" });
              res.end(listResponse ?? JSON.stringify({ jsonrpc: "2.0", id: rpc.id, result: { tools: handleToolsList() } }));
            } else {
              res.writeHead(204);
              res.end();
            }
          }
        });
      });
      await new Promise(resolve => server.listen(0, "localhost", resolve));
      return String(server.address().port);
    }

    it("waits for delayed backend registration after process health succeeds", async () => {
      let lists = 0;
      const port = await startBackend(() => (++lists < 3 ? [] : [{ name: "noop" }]));

      await waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig);

      expect(lists).toBe(3);
      expect(global.core.info).toHaveBeenCalledWith(expect.stringContaining("ready with 1 tool(s) after 3 attempt(s)"));
    });

    it.each(["safeoutputs", "safe-outputs"])("probes the exact configured %s backend endpoint", async serverName => {
      const port = await startBackend(() => [{ name: "noop" }], { serverName });

      await waitForSafeOutputsReadiness({ mcpServers: { [serverName]: {} } }, port, "readiness-test-key", retryConfig);

      expect(requests.filter(method => method === "tools/list")).toHaveLength(1);
    });

    it("retries transient HTTP server failures before registration succeeds", async () => {
      const port = await startBackend(() => [{ name: "noop" }], { initStatus: 503 });

      await waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig);

      expect(requests.filter(method => method === "initialize")).toHaveLength(2);
    });

    it("retries connection resets and then empty tools before registration succeeds", async () => {
      let lists = 0;
      const port = await startBackend(() => (++lists < 2 ? [] : [{ name: "noop" }]), { resetFirstInitialize: true });

      await waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig);

      expect(requests.filter(method => method === "initialize")).toHaveLength(3);
      expect(lists).toBe(2);
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("initialize failed"));
    });

    it("fails setup explicitly after the bounded attempts when tools stay empty", async () => {
      const port = await startBackend(() => []);

      await expect(waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig)).rejects.toThrow(
        /MCP gateway safeoutputs readiness failed after 4 attempt\(s\): no tools registered or backend unavailable\. The agent will not start/
      );

      expect(requests.filter(method => method === "tools/list")).toHaveLength(4);
      expect(global.core.info).not.toHaveBeenCalledWith(expect.stringContaining("is ready"));
    });

    it("fails immediately without accepting tools after an authentication failure", async () => {
      const port = await startBackend(() => [{ name: "noop" }], { initStatus: 401 });

      const error = await waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig).catch(err => err);
      expect(error.message).toContain("permanent authentication or MCP protocol failure");
      expect(error.message).toContain("Last probe: initialize failed (HTTP 401).");
      expect(error.cause).toBeInstanceOf(Error);
      expect(error.stack).not.toContain("readiness-test-key");

      expect(requests).toEqual(["initialize"]);
      expect(JSON.stringify(global.core.warning.mock.calls)).not.toContain("readiness-test-key");
    });

    it.each(['{"jsonrpc":"2.0","id":2,"error":{"code":-32602,"message":"readiness-test-key"}}', "malformed readiness-test-key"])("fails immediately on permanent tools/list protocol failures", async listResponse => {
      const port = await startBackend(() => [], { listResponse });

      const error = await waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig).catch(err => err);
      expect(error.message).toContain("permanent authentication or MCP protocol failure");
      expect(error.message).toContain("Last probe: tools/list failed (HTTP 200).");
      expect(error.cause).toBeInstanceOf(Error);
      expect(error.stack).not.toContain("readiness-test-key");

      expect(requests.filter(method => method === "initialize")).toHaveLength(1);
      expect(JSON.stringify(global.core.warning.mock.calls)).not.toContain("readiness-test-key");
    });

    it("includes the failing phase and whitelisted connection code in the final error", async () => {
      const port = await startBackend(() => [], { resetEveryInitialize: true });
      const error = await waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig).catch(err => err);

      expect(error.message).toContain("Last probe: initialize failed (ECONNRESET).");
      expect(error.cause).toBeInstanceOf(Error);
      expect(error.stack).not.toContain("readiness-test-key");
      expect(error.stack).not.toContain(`localhost:${port}`);
      expect(requests).toEqual(["initialize", "initialize", "initialize", "initialize"]);
    });

    it.each([1, 300, 749])("skips a near-deadline probe with only %ims remaining", async remainingMs => {
      const port = await startBackend(() => [{ name: "noop" }]);
      const now = Date.now();
      vi.spyOn(Date, "now").mockReturnValue(now);
      const startupStart = now - retryConfig.backendStartupTimeoutMs - MCP_GATEWAY_HEALTH_REGISTRATION_MARGIN_MS + remainingMs;
      const error = await waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig, startupStart).catch(err => err);

      expect(error.message).toBe(
        "MCP gateway safeoutputs readiness failed after 0 attempt(s): no tools registered or backend unavailable. The agent will not start. Check MCP gateway startup logs for connection resets (ECONNRESET) or delayed backend registration. Last probe: insufficient startup budget for a readiness probe (requires at least 750ms)."
      );
      expect(error.cause.message).toBe("insufficient startup budget for a readiness probe (requires at least 750ms)");
      expect(requests).toEqual([]);
      expect(global.core.warning).not.toHaveBeenCalled();
      expect(Date.now()).toBe(now);
    });

    it("does not begin another readiness window after late health success exhausts startup", async () => {
      const port = await startBackend(() => [{ name: "noop" }]);
      const now = Date.now();
      vi.spyOn(Date, "now").mockReturnValue(now);
      const startupStart = now - retryConfig.backendStartupTimeoutMs - MCP_GATEWAY_HEALTH_REGISTRATION_MARGIN_MS;

      await expect(waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig, startupStart)).rejects.toThrow(/readiness failed after 0 attempt/);

      expect(requests).toEqual([]);
    });

    it("uses only the remaining original startup budget after late health success", async () => {
      const now = Date.now();
      let clock = now;
      vi.spyOn(Date, "now").mockImplementation(() => clock);
      const startupStart = now - retryConfig.backendStartupTimeoutMs - MCP_GATEWAY_HEALTH_REGISTRATION_MARGIN_MS + 750;
      const port = await startBackend(() => {
        clock += 750;
        return [];
      });

      await expect(waitForSafeOutputsReadiness(config, port, "readiness-test-key", retryConfig, startupStart)).rejects.toThrow(/readiness failed after 1 attempt/);

      expect(requests.filter(method => method === "initialize")).toHaveLength(1);
    });

    it("caps health probes and readiness backoff to one shared deadline", async () => {
      vi.useFakeTimers();
      vi.setSystemTime(1000);
      const deadline = 1100;
      const health = vi.fn(async remainingMs => {
        expect(remainingMs).toBe(100);
        await new Promise(resolve => setTimeout(resolve, 90));
      });
      const healthWait = retryGatewayStartup(health, retryConfig, deadline, "health");
      await vi.advanceTimersByTimeAsync(90);
      await healthWait;
      const readiness = vi.fn(async remainingMs => {
        expect(remainingMs).toBe(10);
        throw new Error("empty tools");
      });
      const readinessWait = retryGatewayStartup(readiness, { ...retryConfig, initialDelayMs: 1000, maxDelayMs: 1000 }, deadline, "readiness");
      const rejection = expect(readinessWait).rejects.toThrow(/startup deadline exceeded/);
      await vi.advanceTimersByTimeAsync(10);
      await rejection;

      expect(Date.now()).toBe(deadline);
      expect(readiness).toHaveBeenCalledTimes(1);
    });

    it("rejects a probe that finishes at the startup deadline", async () => {
      vi.useFakeTimers();
      vi.setSystemTime(1000);
      const wait = retryGatewayStartup(
        async remainingMs => {
          await new Promise(resolve => setTimeout(resolve, remainingMs));
        },
        retryConfig,
        1010,
        "readiness"
      );
      const rejection = expect(wait).rejects.toThrow(/startup deadline exceeded/);
      await vi.advanceTimersByTimeAsync(10);
      await rejection;
    });

    it("bounds health request duration even when the response body keeps streaming", async () => {
      let interval;
      server = http.createServer((_req, res) => {
        res.writeHead(200, { "Content-Type": "text/plain" });
        res.write("starting");
        interval = setInterval(() => res.write("."), 5);
        res.on("close", () => clearInterval(interval));
      });
      await new Promise(resolve => server.listen(0, "localhost", resolve));
      const started = Date.now();
      try {
        await expect(httpGet(`http://localhost:${server.address().port}/health`, 40)).rejects.toThrow("timeout");
        expect(Date.now() - started).toBeLessThan(500);
      } finally {
        clearInterval(interval);
      }
    });

    it.each([{}, { mcpServers: {} }, { mcpServers: { github: {} } }])("skips workflows without configured safe outputs: %j", async noSafeOutputs => {
      await waitForSafeOutputsReadiness(noSafeOutputs, "invalid-port", "readiness-test-key", retryConfig);
      expect(global.core.info).not.toHaveBeenCalled();
    });

    it("gates startup completion and outputs on safeoutputs readiness", () => {
      const source = fs.readFileSync(new URL("./start_mcp_gateway.cjs", import.meta.url), "utf8");
      const gate = source.indexOf("await waitForSafeOutputsReadiness(configObj, gatewayPort, agentId, healthRetryConfig, gatewayStartTime)");
      expect(gate).toBeGreaterThan(0);
      expect(gate).toBeLessThan(source.indexOf("writeGatewayStartupMarker(gatewayStartupMarker)"));
      expect(gate).toBeLessThan(source.indexOf("fs.appendFileSync(githubOutput, outputs"));
      expect(source).toContain("stopGatewayProcess(activeGatewayPid)");
      expect(source).toContain("core.setFailed(`FATAL: ${message}`)");
      expect(source).toContain("const startupDeadline = gatewayStartTime + healthRetryConfig.backendStartupTimeoutMs + MCP_GATEWAY_HEALTH_REGISTRATION_MARGIN_MS");
      expect(source).toContain("httpGet(healthUrl, Math.min(2000, remainingMs))");
    });
  });

  it("redacts bearer headers and structured credential keys", () => {
    const sample = ["Authorization: ******", '{"agentId":"redact-me"}', "token=redact-me"].join("\n");
    const redacted = redactGatewayDiagnostics(sample);
    expect(redacted).not.toContain("redact-me");
    expect(redacted.match(/\[REDACTED\]/g)).toHaveLength(3);
  });
});

describe("start_mcp_gateway startup marker", () => {
  it("defaults to the path read by print_firewall_logs.sh", () => {
    expect(getGatewayStartupMarkerPath({})).toBe("/tmp/gh-aw/mcp-gateway-started");
  });

  it("honors the MCP_GATEWAY_STARTUP_MARKER override", () => {
    expect(getGatewayStartupMarkerPath({ MCP_GATEWAY_STARTUP_MARKER: "/tmp/custom-marker" })).toBe("/tmp/custom-marker");
  });

  it("clears a stale marker and recreates it with owner-only permissions", () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-marker-"));
    const markerPath = path.join(dir, "nested", "mcp-gateway-started");
    fs.mkdirSync(path.dirname(markerPath), { recursive: true });
    fs.writeFileSync(markerPath, "stale");

    clearGatewayStartupMarker(markerPath);
    expect(fs.existsSync(markerPath)).toBe(false);

    writeGatewayStartupMarker(markerPath);
    expect(fs.existsSync(markerPath)).toBe(true);
    expect(fs.statSync(markerPath).mode & 0o777).toBe(0o600);

    fs.rmSync(dir, { recursive: true, force: true });
  });
});

describe("start_mcp_gateway custom environment arguments", () => {
  const marker = "__GH_AW_MCP_GATEWAY_CUSTOM_ENV__";

  it("passes hostile values as one atomic Docker argument", () => {
    const hostileValue = `x" --privileged -v /workspace/evil:/evil --entrypoint /evil -e X="x`;
    const args = injectCustomGatewayEnvArgs(["run", "--rm", marker, "gateway-image"], {
      [customGatewayEnvNamesVar]: '["BASH_ENV"]',
      [`${customGatewayEnvTransportPrefix}0`]: hostileValue,
    });

    expect(args).toEqual(["run", "--rm", "-e", `BASH_ENV=${hostileValue}`, "gateway-image"]);
  });

  it("preserves sorted multi-value index mapping and empty values", () => {
    const args = injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
      [customGatewayEnvNamesVar]: '["ALPHA","EMPTY","OMEGA"]',
      [`${customGatewayEnvTransportPrefix}0`]: "first",
      [`${customGatewayEnvTransportPrefix}1`]: "",
      [`${customGatewayEnvTransportPrefix}2`]: "last\nline",
    });

    expect(args).toEqual(["run", "-e", "ALPHA=first", "-e", "EMPTY=", "-e", "OMEGA=last\nline", "gateway-image"]);
  });

  it("uses an empty value when transport metadata is missing", () => {
    const args = injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
      [customGatewayEnvNamesVar]: '["PRESENT","MISSING"]',
      [`${customGatewayEnvTransportPrefix}0`]: "value",
    });

    expect(args).toEqual(["run", "-e", "PRESENT=value", "-e", "MISSING=", "gateway-image"]);
  });

  it("leaves commands without the marker unchanged", () => {
    const args = ["run", "--rm", "gateway-image"];
    expect(injectCustomGatewayEnvArgs(args, { [customGatewayEnvNamesVar]: "not-json" })).toBe(args);
  });

  it("rejects malformed JSON metadata", () => {
    expect(() =>
      injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
        [customGatewayEnvNamesVar]: "not-json",
      })
    ).toThrow(/must be valid JSON/);
  });

  it("rejects malformed or unsafe environment variable names", () => {
    expect(() =>
      injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
        [customGatewayEnvNamesVar]: '["BAD-NAME"]',
      })
    ).toThrow(/valid environment variable names/);
  });

  it.each([[`${customGatewayEnvTransportPrefix}0`], [customGatewayEnvNamesVar], [`${customGatewayReservedEnvPrefix}FOO`]])("rejects the reserved name %s", reservedName => {
    expect(() =>
      injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
        [customGatewayEnvNamesVar]: JSON.stringify([reservedName]),
      })
    ).toThrow(/reserved/);
  });

  it("rejects duplicate environment variable names", () => {
    expect(() =>
      injectCustomGatewayEnvArgs(["run", marker, "gateway-image"], {
        [customGatewayEnvNamesVar]: '["API_TOKEN","API_TOKEN"]',
      })
    ).toThrow(/duplicate/);
  });
});

describe("start_mcp_gateway OTLP if-missing helpers", () => {
  let originalWarning;

  beforeEach(() => {
    originalWarning = global.core.warning;
    global.core.warning = vi.fn();
  });

  afterEach(() => {
    delete process.env.GH_AW_OTLP_IF_MISSING;
    global.core.warning = originalWarning;
  });

  it("normalizes if-missing mode", () => {
    expect(getOTLPIfMissingMode(undefined)).toBe("error");
    expect(getOTLPIfMissingMode(" warn ")).toBe("warn");
    expect(getOTLPIfMissingMode("ignore")).toBe("ignore");
    expect(getOTLPIfMissingMode("invalid")).toBe("error");
  });

  it("detects non-empty OTLP headers for string/map/array forms", () => {
    expect(hasNonEmptyOTLPHeaders("")).toBe(false);
    expect(hasNonEmptyOTLPHeaders("Authorization=Bearer token")).toBe(true);
    expect(hasNonEmptyOTLPHeaders({ Authorization: "" })).toBe(false);
    expect(hasNonEmptyOTLPHeaders({ Authorization: "Bearer token" })).toBe(true);
    expect(hasNonEmptyOTLPHeaders(["", "  "])).toBe(false);
    expect(hasNonEmptyOTLPHeaders(["", "token"])).toBe(true);
  });

  it("is a no-op when if-missing mode is unset/error", () => {
    const config = {
      gateway: {
        opentelemetry: {
          endpoint: "   ",
          headers: "",
        },
      },
    };
    applyOTLPIgnoreIfMissing(config);
    expect(config.gateway.opentelemetry).toEqual({
      endpoint: "   ",
      headers: "",
    });
  });

  it("removes opentelemetry when endpoint is empty for warn mode and emits a warning", () => {
    const warningSpy = vi.fn();
    global.core.warning = warningSpy;
    process.env.GH_AW_OTLP_IF_MISSING = "warn";

    const config = {
      gateway: {
        opentelemetry: {
          endpoint: "   ",
          headers: { Authorization: "" },
        },
      },
    };

    applyOTLPIgnoreIfMissing(config);

    expect(config.gateway.opentelemetry).toBeUndefined();
    expect(warningSpy).toHaveBeenCalledOnce();
    expect(warningSpy).toHaveBeenCalledWith(expect.stringContaining("OTLP endpoint is missing/empty"));
  });

  it("removes empty headers object for warn mode and emits a warning", () => {
    const warningSpy = vi.fn();
    global.core.warning = warningSpy;
    process.env.GH_AW_OTLP_IF_MISSING = "warn";

    const config = {
      gateway: {
        opentelemetry: {
          endpoint: "https://collector.example/v1/traces",
          headers: { Authorization: "", "X-Tenant": "   " },
        },
      },
    };

    applyOTLPIgnoreIfMissing(config);

    expect(config.gateway.opentelemetry.headers).toBeUndefined();
    expect(warningSpy).toHaveBeenCalledOnce();
    expect(warningSpy).toHaveBeenCalledWith(expect.stringContaining("OTLP headers are missing/empty"));
  });

  it("removes empty headers object for ignore mode without warning", () => {
    const warningSpy = vi.fn();
    global.core.warning = warningSpy;
    process.env.GH_AW_OTLP_IF_MISSING = "ignore";

    const config = {
      gateway: {
        opentelemetry: {
          endpoint: "https://collector.example/v1/traces",
          headers: { Authorization: "" },
        },
      },
    };

    applyOTLPIgnoreIfMissing(config);

    expect(config.gateway.opentelemetry.headers).toBeUndefined();
    expect(warningSpy).not.toHaveBeenCalled();
  });
});

// -----------------------------------------------------------------------------
// resolveCopilotConfigPaths — guards against the regression where /home/runner
// was hard-coded and broke self-hosted runners with HOME != /home/runner.
// -----------------------------------------------------------------------------
describe("start_mcp_gateway resolveCopilotConfigPaths", () => {
  let originalHome;

  beforeEach(() => {
    originalHome = process.env.HOME;
  });

  afterEach(() => {
    if (originalHome === undefined) {
      delete process.env.HOME;
    } else {
      process.env.HOME = originalHome;
    }
  });

  it("resolves the Copilot config dir under the runtime $HOME", () => {
    process.env.HOME = "/home/runner";
    expect(resolveCopilotConfigPaths()).toEqual({
      dir: "/home/runner/.copilot",
      file: "/home/runner/.copilot/mcp-config.json",
    });
  });

  it("respects a self-hosted runner HOME (not /home/runner)", () => {
    process.env.HOME = "/home/actions";
    expect(resolveCopilotConfigPaths()).toEqual({
      dir: "/home/actions/.copilot",
      file: "/home/actions/.copilot/mcp-config.json",
    });
  });

  it("respects a containerized HOME (/root)", () => {
    process.env.HOME = "/root";
    expect(resolveCopilotConfigPaths()).toEqual({
      dir: "/root/.copilot",
      file: "/root/.copilot/mcp-config.json",
    });
  });

  it("handles HOME with spaces and special characters via path.join", () => {
    process.env.HOME = "/var/lib/actions runner";
    expect(resolveCopilotConfigPaths()).toEqual({
      dir: "/var/lib/actions runner/.copilot",
      file: "/var/lib/actions runner/.copilot/mcp-config.json",
    });
  });

  it("throws (not exits) when HOME is unset so tests can exercise the branch", () => {
    delete process.env.HOME;
    expect(() => resolveCopilotConfigPaths()).toThrow(/HOME environment variable is not set/);
  });

  it("throws when HOME is empty string", () => {
    process.env.HOME = "";
    expect(() => resolveCopilotConfigPaths()).toThrow(/HOME environment variable is not set/);
  });

  it("never returns a path containing the literal /home/runner when HOME is different", () => {
    process.env.HOME = "/opt/actions/home";
    const { dir, file } = resolveCopilotConfigPaths();
    expect(dir).not.toContain("/home/runner");
    expect(file).not.toContain("/home/runner");
  });
});

describe("start_mcp_gateway detectEngineType", () => {
  const configDir = "/tmp/gh-aw/mcp-config";

  it("does not require HOME for an explicit non-copilot engine", () => {
    expect(detectEngineType(configDir, { GH_AW_ENGINE: "codex" }, () => false)).toBe("codex");
  });

  it("does not require HOME when auto-detecting codex", () => {
    const existsSync = vi.fn(p => p === `${configDir}/config.toml`);
    expect(detectEngineType(configDir, {}, existsSync)).toBe("codex");
    expect(existsSync).not.toHaveBeenCalledWith("/.copilot");
  });

  it("auto-detects copilot from the HOME-scoped config directory", () => {
    const env = { HOME: "/var/lib/actions runner" };
    const existsSync = vi.fn(p => p === "/var/lib/actions runner/.copilot");
    expect(detectEngineType(configDir, env, existsSync)).toBe("copilot");
  });
});

describe("start_mcp_gateway getJSONParseErrorContext", () => {
  it("extracts line/column and key for invalid escape values", () => {
    const invalidConfig = `{
  "mcpServers": {
    "github": {
      "env": {
        "GITHUB_HOST": "\\https://github.com"
      }
    }
  }
}`;
    let parseErrorMessage = "";
    try {
      JSON.parse(invalidConfig);
    } catch (err) {
      parseErrorMessage = /** @type {Error} */ err.message;
    }
    const context = getJSONParseErrorContext(invalidConfig, parseErrorMessage);
    expect(context).toBeTruthy();
    expect(context?.key).toBe("GITHUB_HOST");
    expect(context?.lineText).toContain(`"GITHUB_HOST"`);
  });
});

describe("start_mcp_gateway normalizeSinkVisibilityEncoding", () => {
  it("normalizes double-encoded sink visibility values", () => {
    const invalidConfig = `{
  "guard-policies": {
    "write-sink": {
      "sink-visibility": ""public""
    }
  }
}`;
    expect(normalizeSinkVisibilityEncoding(invalidConfig)).toContain(`"sink-visibility": "public"`);
  });

  it("leaves correctly encoded sink visibility values unchanged", () => {
    const validConfig = `{
  "guard-policies": {
    "write-sink": {
      "sink-visibility": "public"
    }
  }
}`;
    expect(normalizeSinkVisibilityEncoding(validConfig)).toBe(validConfig);
  });
});

describe("start_mcp_gateway extractOptionalServerNames", () => {
  it("writes explicit required settings before extracting them for gateway startup", () => {
    const directory = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-mcp-required-"));
    try {
      const configObj = { mcpServers: { optional: { type: "http", required: false } } };
      writeMcpConfiguration(directory, configObj);
      expect(extractOptionalServerNames(configObj)).toEqual(["optional"]);
      const snapshot = JSON.parse(fs.readFileSync(path.join(directory, "configuration.jsonl"), "utf8"));
      expect(snapshot.data.servers).toEqual([{ serverName: "optional", transport: "http", required: false }]);
    } finally {
      fs.rmSync(directory, { recursive: true, force: true });
    }
  });

  it("collects servers declared with required: false and strips the flag from every server", () => {
    const configObj = {
      mcpServers: {
        datadog: { type: "http", url: "https://example.com/mcp", required: false },
        grafana: { type: "http", url: "https://example.com/grafana" },
        sentry: { type: "http", url: "https://example.com/sentry", required: true },
      },
    };

    expect(extractOptionalServerNames(configObj)).toEqual(["datadog"]);
    // The gateway configuration specification has no `required` field, so it is
    // removed for every server regardless of its value.
    expect(configObj.mcpServers.datadog).not.toHaveProperty("required");
    expect(configObj.mcpServers.sentry).not.toHaveProperty("required");
    expect(configObj.mcpServers.grafana).not.toHaveProperty("required");
  });

  it("returns an empty list when no servers are configured", () => {
    expect(extractOptionalServerNames({})).toEqual([]);
    expect(extractOptionalServerNames({ mcpServers: null })).toEqual([]);
  });
});

describe("start_mcp_gateway gateway agent identifier validation", () => {
  it.each([
    ["accepts a singular agent ID", { agentId: "agent-1" }, null],
    ["accepts one plural agent ID", { agentIds: ["agent-1"] }, null],
    ["accepts multiple plural agent IDs", { agentIds: ["agent-1", "agent-2"] }, null],
    ["rejects an empty plural agent ID list", { agentIds: [] }, "ERROR: Gateway 'agentIds' must be a non-empty array of non-empty strings"],
    ["rejects a non-array plural agent ID", { agentIds: "agent-1" }, "ERROR: Gateway 'agentIds' must be a non-empty array of non-empty strings"],
    ["rejects an empty ID in the plural list", { agentIds: ["agent-1", ""] }, "ERROR: Gateway 'agentIds' must be a non-empty array of non-empty strings"],
    ["rejects a non-string ID in the plural list", { agentIds: ["agent-1", 2] }, "ERROR: Gateway 'agentIds' must be a non-empty array of non-empty strings"],
    ["rejects both identifier forms", { agentId: "agent-1", agentIds: ["agent-1"] }, "ERROR: Gateway configuration must specify exactly one of 'agentId' or 'agentIds'"],
    ["rejects neither identifier form", {}, "ERROR: Gateway configuration must specify exactly one of 'agentId' or 'agentIds'"],
  ])("%s", (_name, gateway, expectedError) => {
    expect(validateGatewayAgentIdentifiers(gateway)).toBe(expectedError);
  });

  it("rejects an empty or malformed singular agent ID", () => {
    expect(validateGatewayAgentIdentifiers({ agentId: "" })).toBe("ERROR: Gateway 'agentId' must be a non-empty string");
    expect(validateGatewayAgentIdentifiers({ agentId: ["agent-1"] })).toBe("ERROR: Gateway 'agentId' must be a non-empty string");
  });
});
