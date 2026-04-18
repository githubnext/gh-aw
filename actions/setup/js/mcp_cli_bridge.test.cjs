import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { formatResponse, parseToolArgs } from "./mcp_cli_bridge.cjs";

describe("mcp_cli_bridge.cjs", () => {
  let originalCore;
  let stdoutSpy;
  let stderrSpy;
  /** @type {string[]} */
  let stdoutChunks;
  /** @type {string[]} */
  let stderrChunks;

  beforeEach(() => {
    originalCore = global.core;
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
    };
    process.exitCode = 0;
    stdoutChunks = [];
    stderrChunks = [];
    stdoutSpy = vi.spyOn(process.stdout, "write").mockImplementation(chunk => {
      stdoutChunks.push(String(chunk));
      return true;
    });
    stderrSpy = vi.spyOn(process.stderr, "write").mockImplementation(chunk => {
      stderrChunks.push(String(chunk));
      return true;
    });
  });

  afterEach(() => {
    stdoutSpy.mockRestore();
    stderrSpy.mockRestore();
    global.core = originalCore;
    process.exitCode = 0;
  });

  it("coerces integer and array arguments based on tool schema", () => {
    const schemaProperties = {
      count: { type: "integer" },
      workflows: { type: ["null", "array"] },
    };

    const { args } = parseToolArgs(["--count", "3", "--workflows", "daily-issues-report"], schemaProperties);

    expect(args).toEqual({
      count: 3,
      workflows: ["daily-issues-report"],
    });
  });

  it("treats MCP result envelopes with isError=true as errors", () => {
    formatResponse(
      {
        result: {
          isError: true,
          content: [{ type: "text", text: '{"error":"failed to audit workflow run"}' }],
        },
      },
      "agenticworkflows"
    );

    expect(stdoutChunks.join("")).toBe("");
    expect(stderrChunks.join("")).toContain("failed to audit workflow run");
    expect(process.exitCode).toBe(1);
  });
});
