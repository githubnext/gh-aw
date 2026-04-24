// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

describe("record_missing_messages.cjs - buildHandlerConfig", () => {
  let buildHandlerConfig;

  beforeEach(async () => {
    // Provide minimal globals needed by require chain
    globalThis.core = { info: vi.fn(), warning: vi.fn(), setOutput: vi.fn() };
    globalThis.github = {};
    globalThis.context = { repo: { owner: "o", repo: "r" } };

    const mod = await import("./record_missing_messages.cjs");
    buildHandlerConfig = mod.buildHandlerConfig;
  });

  afterEach(() => {
    vi.unstubAllEnvs();
    vi.clearAllMocks();
  });

  it("should return default config when no env vars are set", () => {
    vi.unstubAllEnvs();
    const config = buildHandlerConfig("GH_AW_MISSING_TOOL");
    expect(config.max).toBe(1);
    expect(config.title_prefix).toBeUndefined();
    expect(config.labels).toBeUndefined();
  });

  it("should parse max from env var", () => {
    vi.stubEnv("GH_AW_MISSING_TOOL_MAX", "5");
    const config = buildHandlerConfig("GH_AW_MISSING_TOOL");
    expect(config.max).toBe(5);
  });

  it("should parse title_prefix from env var", () => {
    vi.stubEnv("GH_AW_REPORT_INCOMPLETE_TITLE_PREFIX", "[incomplete]");
    const config = buildHandlerConfig("GH_AW_REPORT_INCOMPLETE");
    expect(config.title_prefix).toBe("[incomplete]");
  });

  it("should parse labels JSON array from env var", () => {
    vi.stubEnv("GH_AW_MISSING_DATA_LABELS", '["agentic-workflows","bug"]');
    const config = buildHandlerConfig("GH_AW_MISSING_DATA");
    expect(config.labels).toEqual(["agentic-workflows", "bug"]);
  });

  it("should ignore malformed labels JSON", () => {
    vi.stubEnv("GH_AW_MISSING_TOOL_LABELS", "not-json");
    const config = buildHandlerConfig("GH_AW_MISSING_TOOL");
    expect(config.labels).toBeUndefined();
  });

  it("should handle all config values at once", () => {
    vi.stubEnv("GH_AW_MISSING_TOOL_MAX", "3");
    vi.stubEnv("GH_AW_MISSING_TOOL_TITLE_PREFIX", "[missing tool]");
    vi.stubEnv("GH_AW_MISSING_TOOL_LABELS", '["agentic-workflows"]');
    const config = buildHandlerConfig("GH_AW_MISSING_TOOL");
    expect(config.max).toBe(3);
    expect(config.title_prefix).toBe("[missing tool]");
    expect(config.labels).toEqual(["agentic-workflows"]);
  });
});

describe("record_missing_messages.cjs - main", () => {
  let mockCore, main;
  /** @type {string} */
  let tmpFile;

  /**
   * Write agent output items to a temp file and set GH_AW_AGENT_OUTPUT.
   * @param {any[]} items
   */
  function stubAgentOutput(items) {
    tmpFile = path.join(os.tmpdir(), `agent-output-${Math.random().toString(36).slice(2)}.json`);
    fs.writeFileSync(tmpFile, JSON.stringify({ items }));
    vi.stubEnv("GH_AW_AGENT_OUTPUT", tmpFile);
  }

  beforeEach(async () => {
    vi.resetModules();
    mockCore = { info: vi.fn(), warning: vi.fn(), setOutput: vi.fn() };
    globalThis.core = mockCore;
    globalThis.github = {};
    globalThis.context = { repo: { owner: "o", repo: "r" } };

    ({ main } = await import("./record_missing_messages.cjs"));
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllEnvs();
    if (tmpFile && fs.existsSync(tmpFile)) fs.unlinkSync(tmpFile);
    tmpFile = "";
  });

  it("should skip gracefully when agent output cannot be loaded", async () => {
    // No GH_AW_AGENT_OUTPUT set → loadAgentOutput returns failure
    await main();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not load agent output"));
  });

  it("should set zero outputs when output has no matching messages", async () => {
    stubAgentOutput([]);
    await main();
    expect(mockCore.setOutput).toHaveBeenCalledWith("tools_reported", "");
    expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "0");
    expect(mockCore.setOutput).toHaveBeenCalledWith("incomplete_count", "0");
  });

  it("should count and report missing_tool messages", async () => {
    stubAgentOutput([
      { type: "missing_tool", tool: "docker", reason: "not installed" },
      { type: "missing_tool", tool: "kubectl", reason: "unavailable" },
    ]);
    await main();
    expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "2");
    expect(mockCore.setOutput).toHaveBeenCalledWith("tools_reported", "docker, kubectl");
  });

  it("should count incomplete signals", async () => {
    stubAgentOutput([{ type: "report_incomplete", reason: "MCP crashed" }]);
    await main();
    expect(mockCore.setOutput).toHaveBeenCalledWith("incomplete_count", "1");
  });

  it("should ignore non-missing message types in counts", async () => {
    stubAgentOutput([
      { type: "create_issue", title: "Test issue" },
      { type: "missing_tool", tool: "helm", reason: "not found" },
    ]);
    await main();
    expect(mockCore.setOutput).toHaveBeenCalledWith("total_count", "1");
  });
});
