import { describe, it, expect } from "vitest";

// normalizeMcpToolResult is the shared helper used by BOTH tool-registration
// paths in safe_outputs_mcp_server_http.cjs (predefined tools and dynamic
// safe-job tools). The HTTP server cannot be unit-tested without real
// transport/config bootstrap, so the isError-preservation logic that both
// `server.tool(...)` callbacks rely on is exercised here directly.
const { normalizeMcpToolResult } = require("./safe_outputs_mcp_server_http.cjs");

describe("safe_outputs_mcp_server_http.cjs normalizeMcpToolResult", () => {
  it("preserves isError:true from the handler result", () => {
    const result = {
      content: [{ type: "text", text: JSON.stringify({ result: "error", error: "boom" }) }],
      isError: true,
    };
    expect(normalizeMcpToolResult(result)).toEqual({
      content: result.content,
      isError: true,
    });
  });

  it("reports isError:false for a successful handler result", () => {
    const result = {
      content: [{ type: "text", text: JSON.stringify({ result: "success", id: 1 }) }],
    };
    expect(normalizeMcpToolResult(result)).toEqual({
      content: result.content,
      isError: false,
    });
  });

  it("coerces a truthy non-boolean isError to a boolean", () => {
    const result = { content: [{ type: "text", text: "x" }], isError: "yes" };
    expect(normalizeMcpToolResult(result).isError).toBe(true);
  });

  it("defaults content to an empty array when the handler returns no content", () => {
    expect(normalizeMcpToolResult({ isError: true })).toEqual({ content: [], isError: true });
  });

  it("returns a safe shape for null/undefined results", () => {
    expect(normalizeMcpToolResult(null)).toEqual({ content: [], isError: false });
    expect(normalizeMcpToolResult(undefined)).toEqual({ content: [], isError: false });
  });
});
