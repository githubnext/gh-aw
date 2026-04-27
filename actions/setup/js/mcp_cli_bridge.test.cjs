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

  it("maps dashed arg names to underscored schema keys", () => {
    const schemaProperties = {
      issue_number: { type: "integer" },
    };

    const { args } = parseToolArgs(["--issue-number", "42"], schemaProperties);

    expect(args).toEqual({
      issue_number: 42,
    });
  });

  it("maps underscored arg names to dashed schema keys", () => {
    const schemaProperties = {
      "issue-number": { type: "integer" },
    };

    const { args } = parseToolArgs(["--issue_number=99"], schemaProperties);

    expect(args).toEqual({
      "issue-number": 99,
    });
  });

  it("keeps exact schema keys when normalized forms collide", () => {
    const schemaProperties = {
      "issue-number": { type: "integer" },
      issue_number: { type: "integer" },
    };

    const dashed = parseToolArgs(["--issue-number", "7"], schemaProperties);
    const underscored = parseToolArgs(["--issue_number", "8"], schemaProperties);

    expect(dashed.args).toEqual({
      "issue-number": 7,
    });
    expect(underscored.args).toEqual({
      issue_number: 8,
    });
  });

  it("falls back to raw key when normalized schema key is ambiguous", () => {
    const schemaProperties = {
      "issue-number": { type: "integer" },
      issue_number: { type: "integer" },
    };

    const { args } = parseToolArgs(["--issuenumber", "11"], schemaProperties);

    expect(args).toEqual({
      issuenumber: "11",
    });
  });

  it("keeps normalized key unresolved when 3+ schema keys collide", () => {
    const schemaProperties = {
      "issue-number": { type: "integer" },
      issue_number: { type: "integer" },
      issueNumber: { type: "integer" },
    };

    const { args } = parseToolArgs(["--issuenumber", "15"], schemaProperties);

    expect(args).toEqual({
      issuenumber: "15",
    });
  });

  it("keeps unknown argument keys unchanged", () => {
    const schemaProperties = {
      issue_number: { type: "integer" },
    };

    const { args } = parseToolArgs(["--custom-field", "value"], schemaProperties);

    expect(args).toEqual({
      "custom-field": "value",
    });
  });

  it("normalizes repeated mixed dash/underscore arguments for array schema", () => {
    const schemaProperties = {
      issue_number: { type: "array" },
    };

    const { args } = parseToolArgs(["--issue-number", "1", "--issue_number", "2"], schemaProperties);

    expect(args).toEqual({
      issue_number: ["1", "2"],
    });
  });

  it("falls back to numeric coercion when schema properties are unavailable", () => {
    const { args } = parseToolArgs(["--count", "3", "--max_tokens", "3000"], {});

    expect(args).toEqual({
      count: 3,
      max_tokens: 3000,
    });
  });

  it("coerces scientific notation when schema properties are unavailable", () => {
    const { args } = parseToolArgs(["--max_tokens", "1e3", "--threshold", "-2E-4"], {});

    expect(args).toEqual({
      max_tokens: 1000,
      threshold: -0.0002,
    });
  });

  it("preserves non-numeric values when schema properties are unavailable", () => {
    const { args } = parseToolArgs(["--start_date", "-1d", "--workflow_name", "daily-issues-report"], {});

    expect(args).toEqual({
      start_date: "-1d",
      workflow_name: "daily-issues-report",
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

  it("prints progress notifications to stderr and final text result to stdout for SSE responses", () => {
    const sseBody = [
      'data: {"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":"abc","progress":1,"total":3,"message":"Step 1/3"}}',
      'data: {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"done"}]}}',
      "",
    ].join("\n");

    formatResponse(sseBody, "agenticworkflows");

    expect(stderrChunks.join("")).toContain("Step 1/3");
    expect(stdoutChunks.join("")).toBe("done\n");
    expect(process.exitCode).toBe(0);
  });

  it("prints numeric progress to stderr when progress notification has no message", () => {
    const sseBody = ['data: {"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":"abc","progress":2,"total":5}}', 'data: {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"ok"}]}}', ""].join("\n");

    formatResponse(sseBody, "agenticworkflows");

    expect(stderrChunks.join("")).toContain("Progress: 2/5");
    expect(stdoutChunks.join("")).toBe("ok\n");
    expect(process.exitCode).toBe(0);
  });
});
