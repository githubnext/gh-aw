// @ts-check
import { describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

import { AWF_GATEWAY_IP, parseMCPResponseBody, recoverSafeOutputsToolsIfNeeded, toContainerUrl } from "./mount_mcp_as_cli.cjs";

describe("mount_mcp_as_cli.cjs", () => {
  it("parses JSON object responses unchanged", () => {
    const body = { jsonrpc: "2.0", result: { tools: [{ name: "logs" }] } };
    expect(parseMCPResponseBody(body)).toEqual(body);
  });

  it("parses raw JSON string responses", () => {
    const body = '{"jsonrpc":"2.0","result":{"tools":[{"name":"logs"}]}}';
    expect(parseMCPResponseBody(body)).toEqual({
      jsonrpc: "2.0",
      result: { tools: [{ name: "logs" }] },
    });
  });

  it("parses SSE data lines and returns the JSON payload", () => {
    const sseToolListPayload = 'data: {"jsonrpc":"2.0","id":2,"result":{"tools":[{"name":"logs","inputSchema":{"properties":{"count":{"type":"integer"}}}}]}}';
    const body = ["event: message", sseToolListPayload, ""].join("\n");

    expect(parseMCPResponseBody(body)).toEqual({
      jsonrpc: "2.0",
      id: 2,
      result: {
        tools: [
          {
            name: "logs",
            inputSchema: {
              properties: {
                count: { type: "integer" },
              },
            },
          },
        ],
      },
    });
  });

  it("rewrites host.docker.internal to the AWF gateway IP for CLI wrappers", () => {
    const originalDomain = process.env.MCP_GATEWAY_DOMAIN;
    const originalPort = process.env.MCP_GATEWAY_PORT;
    process.env.MCP_GATEWAY_DOMAIN = "host.docker.internal";
    process.env.MCP_GATEWAY_PORT = "8080";

    try {
      expect(toContainerUrl("http://0.0.0.0:8080/mcp/safeoutputs")).toBe(`http://${AWF_GATEWAY_IP}:8080/mcp/safeoutputs`);
    } finally {
      if (originalDomain === undefined) {
        delete process.env.MCP_GATEWAY_DOMAIN;
      } else {
        process.env.MCP_GATEWAY_DOMAIN = originalDomain;
      }
      if (originalPort === undefined) {
        delete process.env.MCP_GATEWAY_PORT;
      } else {
        process.env.MCP_GATEWAY_PORT = originalPort;
      }
    }
  });

  it("recovers empty safeoutputs tools from fallback tools.json", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "mount-safeoutputs-"));
    const fallbackPath = path.join(tempDir, "tools.json");
    fs.writeFileSync(fallbackPath, JSON.stringify([{ name: "create_issue" }]), "utf8");
    const originalPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = fallbackPath;
    try {
      const warnings = [];
      const recovered = recoverSafeOutputsToolsIfNeeded([], { warning: message => warnings.push(message) });
      expect(recovered).toHaveLength(1);
      expect(recovered[0].name).toBe("create_issue");
      expect(warnings.join("\n")).toContain("recovered 1 tool(s)");
    } finally {
      if (originalPath === undefined) {
        delete process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
      } else {
        process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = originalPath;
      }
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("throws when safeoutputs tools remain empty after fallback", () => {
    const originalPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    delete process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    try {
      expect(() => recoverSafeOutputsToolsIfNeeded([], { warning: () => {} })).toThrow(/safeoutputs tool schema is empty/);
    } finally {
      if (originalPath !== undefined) {
        process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = originalPath;
      }
    }
  });
});
