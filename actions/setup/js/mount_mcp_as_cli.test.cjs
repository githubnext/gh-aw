// @ts-check
import { describe, expect, it } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

import { AWF_GATEWAY_IP, buildMCPCLIServersPromptList, getSafeOutputsGatewayEmptyFlagPath, parseMCPResponseBody, recoverSafeOutputsToolsIfNeeded, toContainerUrl } from "./mount_mcp_as_cli.cjs";

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

  it("fails hard even when tools.json has tools, and writes gateway-empty flag", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "mount-safeoutputs-"));
    const fallbackPath = path.join(tempDir, "tools.json");
    fs.writeFileSync(fallbackPath, JSON.stringify([{ name: "create_issue" }]), "utf8");
    const originalPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    const originalRunnerTemp = process.env.RUNNER_TEMP;
    // Point at the tools.json to prove the function does not use it even when present
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = fallbackPath;
    process.env.RUNNER_TEMP = tempDir;
    try {
      // Must throw regardless of whether a fallback tools.json exists
      expect(() => recoverSafeOutputsToolsIfNeeded([], { warning: () => {} })).toThrow(/safeoutputs tools\/list returned 0 tools/);
      // Flag file must be written so collect_ndjson_output.cjs can detect the outage
      const flagPath = getSafeOutputsGatewayEmptyFlagPath();
      expect(fs.existsSync(flagPath)).toBe(true);
    } finally {
      if (originalPath === undefined) {
        delete process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
      } else {
        process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = originalPath;
      }
      if (originalRunnerTemp === undefined) {
        delete process.env.RUNNER_TEMP;
      } else {
        process.env.RUNNER_TEMP = originalRunnerTemp;
      }
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("throws when safeoutputs gateway returns 0 tools and writes gateway-empty flag", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "mount-safeoutputs-empty-"));
    const originalPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    const originalRunnerTemp = process.env.RUNNER_TEMP;
    process.env.RUNNER_TEMP = tempDir;
    // Point at an explicitly missing file so there is no ambient fallback ambiguity
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = path.join(tempDir, "tools.json");
    try {
      expect(() => recoverSafeOutputsToolsIfNeeded([], { warning: () => {} })).toThrow(/safeoutputs tools\/list returned 0 tools/);
      // Flag file must still be written even when the function throws
      const flagPath = getSafeOutputsGatewayEmptyFlagPath();
      expect(fs.existsSync(flagPath)).toBe(true);
    } finally {
      if (originalPath === undefined) {
        delete process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
      } else {
        process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = originalPath;
      }
      if (originalRunnerTemp === undefined) {
        delete process.env.RUNNER_TEMP;
      } else {
        process.env.RUNNER_TEMP = originalRunnerTemp;
      }
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("does not write gateway-empty flag when tools list is non-empty", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "mount-safeoutputs-ok-"));
    const originalRunnerTemp = process.env.RUNNER_TEMP;
    process.env.RUNNER_TEMP = tempDir;
    try {
      const tools = [{ name: "push_to_pull_request_branch" }];
      const result = recoverSafeOutputsToolsIfNeeded(tools, { warning: () => {} });
      expect(result).toEqual(tools);
      const flagPath = getSafeOutputsGatewayEmptyFlagPath();
      expect(fs.existsSync(flagPath)).toBe(false);
    } finally {
      if (originalRunnerTemp === undefined) {
        delete process.env.RUNNER_TEMP;
      } else {
        process.env.RUNNER_TEMP = originalRunnerTemp;
      }
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("renders schema-derived safeoutputs docs for prompt substitution", () => {
    const docs = buildMCPCLIServersPromptList([
      {
        name: "safeoutputs",
        tools: [
          {
            name: "set_issue_type",
            inputSchema: {
              type: "object",
              properties: {
                issue_number: { type: "integer" },
                issue_type: { type: "string" },
                rationale: { type: "string", maxLength: 280 },
                confidence: { type: "string", enum: ["LOW", "MEDIUM", "HIGH"] },
              },
              required: ["issue_number", "issue_type"],
            },
          },
        ],
      },
      { name: "mcpscripts", tools: [] },
    ]);

    expect(docs).toContain("schema-derived command syntax");
    expect(docs).toContain("--issue_number <number>");
    expect(docs).toContain("[--rationale <reason, max 280 characters>]");
    expect(docs).toContain('--confidence \"HIGH\"');
    expect(docs).toContain("`mcpscripts` — run `mcpscripts --help`");
  });
});
