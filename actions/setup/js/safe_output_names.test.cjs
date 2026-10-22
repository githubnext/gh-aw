import { afterEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const { normalizeSafeOutputName, normalizeSafeOutputConfig } = require("./safe_output_names.cjs");
const { createServer, registerTool, handleRequest, handleMessage } = require("./mcp_server_core.cjs");
const { hasExpectedSafeOutputs, hasTerminalSafeOutput } = require("./safeoutputs_cli.cjs");

afterEach(() => vi.unstubAllEnvs());

describe("diagnostic safe-output names", () => {
  it("normalizes legacy configuration keys without changing unrelated keys", () => {
    expect(normalizeSafeOutputConfig({ "missing-tool": { max: 2 }, missing_data: {}, "create-issue": {} })).toEqual({
      report_missing_tool: { max: 2 },
      report_missing_data: {},
      create_issue: {},
    });
    expect(normalizeSafeOutputName("report_missing_tool")).toBe("report_missing_tool");
    expect(normalizeSafeOutputName("custom_tool")).toBe("custom_tool");
  });

  it.each(["missing_tool", "missing_data"])("accepts legacy %s calls in both core request paths without listing aliases", async legacy => {
    const canonical = `report_${legacy}`;
    const server = createServer({ name: "safeoutputs", version: "1.0.0" });
    const handler = vi.fn(() => ({ content: [{ type: "text", text: "recorded" }] }));
    registerTool(server, { name: canonical, description: "Diagnostic", inputSchema: { type: "object", properties: {} }, handler });

    const listing = await handleRequest(server, { jsonrpc: "2.0", id: 1, method: "tools/list" });
    expect(listing.result.tools.map(tool => tool.name)).toEqual([canonical]);
    for (const name of [canonical, legacy, legacy.replaceAll("_", "-")]) {
      const response = await handleRequest(server, { jsonrpc: "2.0", id: 2, method: "tools/call", params: { name, arguments: {} } });
      expect(response.error).toBeUndefined();
      expect(response.result.content[0].text).toBe("recorded");
    }

    server.replyResult = vi.fn();
    server.replyError = vi.fn();
    await handleMessage(server, { jsonrpc: "2.0", id: 3, method: "tools/call", params: { name: legacy, arguments: {} } });
    expect(server.replyError).not.toHaveBeenCalled();
    expect(handler).toHaveBeenCalledTimes(4);
  });

  it("does not rename tools belonging to other MCP servers", () => {
    const server = createServer({ name: "custom", version: "1.0.0" });
    registerTool(server, { name: "missing_data", description: "", inputSchema: { type: "object", properties: {} } });
    expect(Object.keys(server.tools)).toEqual(["missing_data"]);
  });

  it.each(["missing_tool", "report_missing_tool", "missing_data", "report_missing_data"])("keeps %s non-terminal", type => {
    const options = { readFileSync: () => JSON.stringify({ type, reason: "unavailable" }) };
    expect(hasTerminalSafeOutput("outputs.jsonl", options)).toBe(false);
    if (type.endsWith("missing_tool")) expect(hasExpectedSafeOutputs("outputs.jsonl", options)).toBe(false);
  });

  it("loads legacy tools and configuration over HTTP but writes canonical records", async () => {
    const dir = fs.mkdtempSync(path.join(os.tmpdir(), "safe-output-names-"));
    try {
      const configFile = path.join(dir, "config.json");
      const toolsFile = path.join(dir, "tools.json");
      const outputFile = path.join(dir, "outputs.jsonl");
      fs.writeFileSync(configFile, JSON.stringify({ missing_tool: {}, missing_data: {} }));
      const tools = require("./safe_outputs_tools.json").filter(tool => ["report_missing_tool", "report_missing_data"].includes(tool.name));
      fs.writeFileSync(toolsFile, JSON.stringify(tools.map(tool => ({ ...tool, name: tool.name.replace("report_", "") }))));
      vi.stubEnv("GH_AW_SAFE_OUTPUTS_CONFIG_PATH", configFile);
      vi.stubEnv("GH_AW_SAFE_OUTPUTS_TOOLS_PATH", toolsFile);
      vi.stubEnv("GH_AW_SAFE_OUTPUTS", outputFile);
      const { createMCPServer } = require("./safe_outputs_mcp_server_http.cjs");
      const { server } = createMCPServer();
      expect([...server.tools.keys()]).toEqual(["report_missing_tool", "report_missing_data"]);
      for (const name of ["missing_tool", "report_missing_tool", "missing_data", "report_missing_data"]) {
        const response = await server.handleRequest({ jsonrpc: "2.0", id: 1, method: "tools/call", params: { name, arguments: { reason: "unavailable" } } });
        expect(response.error).toBeUndefined();
        expect(response.result.isError).not.toBe(true);
      }
      expect(
        fs
          .readFileSync(outputFile, "utf8")
          .trim()
          .split("\n")
          .map(line => JSON.parse(line).type)
      ).toEqual(["report_missing_tool", "report_missing_tool", "report_missing_data", "report_missing_data"]);
    } finally {
      fs.rmSync(dir, { recursive: true, force: true });
    }
  });
});
