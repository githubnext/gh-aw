// @ts-check
import { describe, expect, it } from "vitest";

import { fetchMCPTools, getMissingExpectedTools, parseMCPResponseBody } from "./mount_mcp_as_cli.cjs";

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

  it("computes missing expected tools from the discovered tool list", () => {
    const expectedTools = ["list_datasources", "tempo_traceql-search", "tempo_get-trace"];
    const discoveredTools = [{ name: "list_datasources" }, { name: "tempo_get-trace" }];

    expect(getMissingExpectedTools(expectedTools, discoveredTools)).toEqual(["tempo_traceql-search"]);
  });

  it("retries tools/list until expected tools appear", async () => {
    const info = [];
    const warning = [];
    const core = {
      info: message => info.push(message),
      warning: message => warning.push(message),
    };
    let listAttempt = 0;
    /**
     * @param {string} _url
     * @param {Record<string, string>} _headers
     * @param {{ jsonrpc?: string, id?: number, method?: string, params?: unknown }} body
     */
    const httpPostJSON = async (_url, _headers, body) => {
      const method = body.method;
      if (method === "initialize") {
        return { statusCode: 200, body: { jsonrpc: "2.0", result: {} }, headers: { "mcp-session-id": "session-1" } };
      }
      if (method === "notifications/initialized") {
        return { statusCode: 204, body: "", headers: {} };
      }
      if (method === "tools/list") {
        listAttempt += 1;
        const tools = listAttempt === 1 ? [{ name: "list_datasources" }] : [{ name: "list_datasources" }, { name: "tempo_traceql-search" }, { name: "tempo_get-trace" }];
        return { statusCode: 200, body: { jsonrpc: "2.0", result: { tools } }, headers: {} };
      }
      throw new Error(`unexpected method: ${method}`);
    };

    const tools = await fetchMCPTools("http://localhost:8080/mcp/grafana", "token", core, {
      expectedTools: ["list_datasources", "tempo_traceql-search", "tempo_get-trace"],
      maxAttempts: 2,
      httpPostJSON,
    });

    expect(listAttempt).toBe(2);
    expect(tools.map(tool => tool.name)).toEqual(["list_datasources", "tempo_traceql-search", "tempo_get-trace"]);
    expect(info).toContainEqual(expect.stringMatching(/attempt 1\/2; retrying: tempo_traceql-search, tempo_get-trace/));
    expect(warning).toEqual([]);
  });
});
