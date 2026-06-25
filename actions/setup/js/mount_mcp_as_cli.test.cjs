// @ts-check
import { describe, expect, it } from "vitest";

import { AWF_GATEWAY_IP, parseMCPResponseBody, toContainerUrl } from "./mount_mcp_as_cli.cjs";

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
});
