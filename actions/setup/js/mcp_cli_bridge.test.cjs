import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import fs from "fs";
import os from "os";
import path from "path";

import { ensureSafeOutputsTools, formatResponse, getToolCallTimeoutMs, hasStdinJsonPayload, parseToolArgs, readStdinSync, shouldShowToolHelpForEmptyArgs, showHelp, showToolHelp, writeStdoutAndFlush } from "./mcp_cli_bridge.cjs";

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

  it("recovers empty safeoutputs schema from fallback tools path", () => {
    const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "bridge-safeoutputs-"));
    const fallbackPath = path.join(tempDir, "tools.json");
    fs.writeFileSync(fallbackPath, JSON.stringify([{ name: "report_incomplete" }]), "utf8");
    const originalPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = fallbackPath;
    try {
      const recovered = ensureSafeOutputsTools([], "safeoutputs", path.join(tempDir, "empty.json"));
      expect(recovered).toHaveLength(1);
      expect(recovered[0].name).toBe("report_incomplete");
      expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("recovered"));
    } finally {
      if (originalPath === undefined) {
        delete process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
      } else {
        process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = originalPath;
      }
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("fails fast when safeoutputs schema is empty", () => {
    const originalPath = process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    delete process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    try {
      expect(() => ensureSafeOutputsTools([], "safeoutputs", "/tmp/gh-aw/mcp-cli/tools/safeoutputs.json")).toThrow(/tool schema is empty/);
    } finally {
      if (originalPath !== undefined) {
        process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = originalPath;
      }
    }
  });

  it("shows help instead of calling safeoutputs tools with an empty args object", () => {
    expect(shouldShowToolHelpForEmptyArgs("safeoutputs", {})).toBe(true);
    expect(shouldShowToolHelpForEmptyArgs("safeoutputs", { title: "Bug report" })).toBe(false);
    expect(shouldShowToolHelpForEmptyArgs("other-server", {})).toBe(false);
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

  it("uses default 120s timeout for non-logs tools", () => {
    expect(getToolCallTimeoutMs("audit", {})).toBe(120000);
  });

  it("uses a longer timeout for logs calls without explicit timeout (default count=100, no filter)", () => {
    // effectiveCount=100, base=ceil(100/40)=3, no workflow_name → max(5,3)=5 minutes
    expect(getToolCallTimeoutMs("logs", {})).toBe(315000);
  });

  it("scales logs timeout from count when no explicit timeout is set (count=250, no filter)", () => {
    // effectiveCount=250, base=ceil(250/40)=7, no workflow_name → max(5,7)=7 minutes
    expect(getToolCallTimeoutMs("logs", { count: 250 })).toBe(435000);
  });

  it("scales logs timeout from count with workflow_name filter (count=250, filtered)", () => {
    // effectiveCount=250, base=ceil(250/40)=7, workflow_name present → 7 minutes (no min floor applied)
    expect(getToolCallTimeoutMs("logs", { count: 250, workflow_name: "ci" })).toBe(435000);
  });

  it("clamps count-based timeout to global minimum for small filtered counts", () => {
    // effectiveCount=40, base=ceil(40/40)=1, workflow_name present → 1 minute → 75000ms < 120000ms → clamped
    expect(getToolCallTimeoutMs("logs", { count: 40, workflow_name: "ci" })).toBe(120000);
  });

  it("applies 5-minute no-filter floor for small unfiltered counts", () => {
    // effectiveCount=40, base=1, no workflow_name → max(5,1)=5 minutes
    expect(getToolCallTimeoutMs("logs", { count: 40 })).toBe(315000);
  });

  it("uses logs timeout argument with bridge buffer when provided", () => {
    // timeout=10min, floor=5min (default count=100, no filter) → max(120000, 315000, 615000) = 615000
    expect(getToolCallTimeoutMs("logs", { timeout: 10 })).toBe(615000);
  });

  it("floors small explicit timeout to the count-derived minimum", () => {
    // timeout=2.5min → explicit=165000ms; floor=5min → 315000ms; floor wins
    expect(getToolCallTimeoutMs("logs", { timeout: 2.5 })).toBe(315000);
  });

  it("caps explicit timeout at LOGS_TOOL_MAX_EXPLICIT_TIMEOUT_MINUTES (60)", () => {
    // timeout=999min → clamped to 60min → 3615000ms; floor=315000ms; capped value wins
    expect(getToolCallTimeoutMs("logs", { timeout: 999 })).toBe(3615000);
  });

  it("rejects non-numeric timeout types and falls back to count-derived timeout", () => {
    // typeof-check rejects strings and booleans even when Number() would accept them
    expect(getToolCallTimeoutMs("logs", { timeout: 0 })).toBe(315000);
    expect(getToolCallTimeoutMs("logs", { timeout: -5 })).toBe(315000);
    expect(getToolCallTimeoutMs("logs", { timeout: "invalid" })).toBe(315000);
    expect(getToolCallTimeoutMs("logs", { timeout: "5" })).toBe(315000);
    expect(getToolCallTimeoutMs("logs", { timeout: true })).toBe(315000);
  });

  it("treats MCP result envelopes with isError=true as errors", async () => {
    await formatResponse(
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

  it("prints progress notifications to stderr and final text result to stdout for SSE responses", async () => {
    const sseBody = [
      'data: {"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":"abc","progress":1,"total":3,"message":"Step 1/3"}}',
      'data: {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"done"}]}}',
      "",
    ].join("\n");

    await formatResponse(sseBody, "agenticworkflows");

    expect(stderrChunks.join("")).toContain("Step 1/3");
    expect(stdoutChunks.join("")).toBe("done\n");
    expect(process.exitCode).toBe(0);
  });

  it("prints numeric progress to stderr when progress notification has no message", async () => {
    const sseBody = ['data: {"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":"abc","progress":2,"total":5}}', 'data: {"jsonrpc":"2.0","id":2,"result":{"content":[{"type":"text","text":"ok"}]}}', ""].join("\n");

    await formatResponse(sseBody, "agenticworkflows");

    expect(stderrChunks.join("")).toContain("Progress: 2/5");
    expect(stdoutChunks.join("")).toBe("ok\n");
    expect(process.exitCode).toBe(0);
  });

  it("adds a non-retry hint for safeoutputs empty-argument rejections", async () => {
    await formatResponse(
      {
        error: {
          code: -32602,
          message: "Empty arguments are not allowed — this tool is write-once, not a discovery probe.",
        },
      },
      "safeoutputs",
      "create_issue"
    );

    const stderr = stderrChunks.join("");
    expect(stderr).toContain("Error [-32602]: Empty arguments are not allowed");
    expect(stderr).toContain("do not retry 'safeoutputs create_issue' with empty arguments");
    expect(stderr).toContain("safeoutputs create_issue --help");
    expect(process.exitCode).toBe(1);
  });

  it("omits non-retry hint when toolName is absent", async () => {
    await formatResponse(
      {
        error: {
          code: -32602,
          message: "Empty arguments are not allowed — this tool is write-once, not a discovery probe.",
        },
      },
      "safeoutputs"
      // toolName omitted → defaults to ""
    );

    const stderr = stderrChunks.join("");
    expect(stderr).toContain("Error [-32602]");
    expect(stderr).not.toContain("do not retry");
    expect(process.exitCode).toBe(1);
  });

  it("does not add a non-retry hint for -32602 errors from non-safeoutputs servers", async () => {
    await formatResponse(
      {
        error: {
          code: -32602,
          message: "Empty arguments are not allowed — this tool is write-once, not a discovery probe.",
        },
      },
      "agenticworkflows",
      "some_tool"
    );

    const stderr = stderrChunks.join("");
    expect(stderr).toContain("Error [-32602]");
    expect(stderr).not.toContain("do not retry");
    expect(stderr).not.toContain("--help");
    expect(process.exitCode).toBe(1);
  });

  it("keeps top-level help compact for many commands", () => {
    const tools = Array.from({ length: 25 }, (_, i) => ({
      name: `tool_${i + 1}`,
      description: `Description for command ${i + 1} that is intentionally verbose for truncation checks.`,
    }));

    showHelp("safeoutputs", tools);

    const outputLines = stdoutChunks.join("").trimEnd().split("\n");
    const output = outputLines.join("\n");
    expect(outputLines.length).toBeLessThanOrEqual(20);
    expect(output).not.toMatch(/\.\.\. \+\d+ more command\(s\)/);
    for (const tool of tools) {
      expect(output).toContain(tool.name);
    }
  });

  it("does not truncate top-level help when commands exactly fit the line budget", () => {
    const tools = Array.from({ length: 14 }, (_, i) => ({
      name: `tool_${i + 1}`,
      description: `Description for command ${i + 1}.`,
    }));

    showHelp("safeoutputs", tools);

    const outputLines = stdoutChunks.join("").trimEnd().split("\n");
    const output = outputLines.join("\n");
    expect(outputLines.length).toBeLessThanOrEqual(20);
    expect(output).not.toMatch(/\.\.\. \+\d+ more command\(s\)/);
    for (const tool of tools) {
      expect(output).toContain(tool.name);
    }
  });

  it("keeps command help compact for many options", () => {
    const properties = {};
    for (let i = 1; i <= 24; i++) {
      properties[`field_${i}`] = { type: "string", description: `Field ${i} description with additional details for truncation.` };
    }

    showToolHelp("safeoutputs", "create_issue", [
      {
        name: "create_issue",
        description: "Create an issue with many available fields and optional metadata.",
        inputSchema: {
          properties,
          required: ["field_1", "field_2"],
        },
      },
    ]);

    const outputLines = stdoutChunks.join("").trimEnd().split("\n");
    const output = outputLines.join("\n");
    expect(outputLines.length).toBeLessThanOrEqual(30);
    expect(output).not.toMatch(/\.\.\. \+\d+ more option\(s\)/);
    expect(output).toContain("Required options are marked with *.");
    for (let i = 1; i <= 24; i++) {
      expect(output).toContain(`--field_${i}`);
    }
    expect(output).toContain("--field_1*");
    expect(output).toContain("--field_2*");
  });

  it("does not truncate command help when options exactly fit the line budget", () => {
    const properties = {};
    for (let i = 1; i <= 13; i++) {
      properties[`field_${i}`] = { type: "string", description: `Field ${i}.` };
    }

    showToolHelp("safeoutputs", "create_issue", [
      {
        name: "create_issue",
        description: "Create an issue.",
        inputSchema: {
          properties,
          required: ["field_1"],
        },
      },
    ]);

    const outputLines = stdoutChunks.join("").trimEnd().split("\n");
    const output = outputLines.join("\n");
    expect(outputLines.length).toBeLessThanOrEqual(30);
    expect(output).not.toMatch(/\.\.\. \+\d+ more option\(s\)/);
    expect(output).toContain("Required options are marked with *.");
    for (let i = 1; i <= 13; i++) {
      expect(output).toContain(`--field_${i}`);
    }
  });

  it("keeps required note when required options are in the compact list", () => {
    const properties = {};
    for (let i = 1; i <= 24; i++) {
      properties[`field_${i}`] = { type: "string", description: `Field ${i}.` };
    }

    showToolHelp("safeoutputs", "create_issue", [
      {
        name: "create_issue",
        description: "Create an issue.",
        inputSchema: {
          properties,
          required: ["field_23", "field_24"],
        },
      },
    ]);

    const outputLines = stdoutChunks.join("").trimEnd().split("\n");
    const output = outputLines.join("\n");
    expect(output).not.toMatch(/\.\.\. \+\d+ more option\(s\)/);
    expect(output).toContain("Required options are marked with *.");
    expect(output).toContain("--field_23*");
    expect(output).toContain("--field_24*");
  });

  describe("stdin placeholder removed — '-' is always a literal value", () => {
    it("passes '--key -' as literal '-' (space-separated form)", () => {
      const schemaProperties = { body: { type: "string" } };
      const stdinContent = "some stdin content";

      const { args } = parseToolArgs(["--body", "-"], schemaProperties, stdinContent);

      expect(args).toEqual({ body: "-" });
    });

    it("passes '--key=-' as literal '-' (equals form)", () => {
      const schemaProperties = { body: { type: "string" } };
      const stdinContent = "some stdin content";

      const { args } = parseToolArgs(["--body=-"], schemaProperties, stdinContent);

      expect(args).toEqual({ body: "-" });
    });

    it("throws when stdin exceeds maximum allowed size", () => {
      const fs = require("fs");
      // Simulate reading more than 10 MB total by making readSync return data repeatedly
      const STDIN_MAX_BYTES = 10 * 1024 * 1024;
      const callCount = { n: 0 };
      const readSyncSpy = vi.spyOn(fs, "readSync").mockImplementation((_fd, buf, _offset, length) => {
        callCount.n++;
        // Each call fills the buffer until we exceed the limit
        if (callCount.n > STDIN_MAX_BYTES / length + 1) return 0;
        buf.fill(0x41, 0, length); // fill with 'A'
        return length;
      });

      try {
        expect(() => readStdinSync()).toThrow(/exceeds maximum allowed size/);
      } finally {
        readSyncSpy.mockRestore();
      }
    });

    it("returns empty string when readSync errors before any bytes are read", () => {
      const fs = require("fs");
      const readSyncSpy = vi.spyOn(fs, "readSync").mockImplementation(() => {
        throw new Error("EBADF: bad file descriptor");
      });

      try {
        expect(readStdinSync()).toBe("");
      } finally {
        readSyncSpy.mockRestore();
      }
    });

    it("rethrows readSync errors that occur after some bytes have already been read", () => {
      const fs = require("fs");
      let callCount = 0;
      const readSyncSpy = vi.spyOn(fs, "readSync").mockImplementation((_fd, buf, _offset, length) => {
        callCount++;
        if (callCount === 1) {
          // First call: return some data
          buf.fill(0x41, 0, length);
          return length;
        }
        // Second call: simulate a mid-stream read error
        throw new Error("EIO: i/o error");
      });

      try {
        expect(() => readStdinSync()).toThrow(/EIO/);
      } finally {
        readSyncSpy.mockRestore();
      }
    });
  });

  describe("stdin JSON payload support", () => {
    it("returns true for '.' sentinel", () => {
      expect(hasStdinJsonPayload(["."])).toBe(true);
    });

    it("returns true for empty args when stdin is not a TTY", () => {
      const origIsTTY = process.stdin.isTTY;
      process.stdin.isTTY = undefined;
      try {
        expect(hasStdinJsonPayload([])).toBe(true);
      } finally {
        process.stdin.isTTY = origIsTTY;
      }
    });

    it("returns false for empty args when stdin is a TTY", () => {
      const origIsTTY = process.stdin.isTTY;
      // @ts-ignore
      process.stdin.isTTY = true;
      try {
        expect(hasStdinJsonPayload([])).toBe(false);
      } finally {
        process.stdin.isTTY = origIsTTY;
      }
    });

    it("returns false when args contain flags", () => {
      expect(hasStdinJsonPayload(["--body", "hello"])).toBe(false);
    });

    it("returns false when args has more than just '.'", () => {
      expect(hasStdinJsonPayload([".", "--extra", "value"])).toBe(false);
    });

    it("returns true for '--key .' (per-field stdin marker, space-separated)", () => {
      expect(hasStdinJsonPayload(["--body", "."])).toBe(true);
    });

    it("returns true for '--key=.' (per-field stdin marker, equals-separated)", () => {
      expect(hasStdinJsonPayload(["--body=."])).toBe(true);
    });

    it("returns true for mixed flags when one uses the per-field sentinel", () => {
      expect(hasStdinJsonPayload(["--title", "My title", "--body", "."])).toBe(true);
    });

    it("returns false when flag value is not '.' (not a sentinel)", () => {
      expect(hasStdinJsonPayload(["--body", "hello"])).toBe(false);
    });

    it("parses stdin JSON object when '.' sentinel is used", () => {
      const schemaProperties = {
        issue_number: { type: "integer" },
        body: { type: "string" },
      };
      const stdinContent = '{"issue_number": 42, "body": "hello world"}';

      const { args } = parseToolArgs(["."], schemaProperties, stdinContent);

      expect(args).toEqual({ issue_number: 42, body: "hello world" });
    });

    it("parses stdin JSON object when no args and stdinContent is provided", () => {
      const schemaProperties = {
        issue_number: { type: "integer" },
        body: { type: "string" },
      };
      const stdinContent = '{"issue_number": 7, "body": "test body"}';

      const { args } = parseToolArgs([], schemaProperties, stdinContent);

      expect(args).toEqual({ issue_number: 7, body: "test body" });
    });

    it("preserves types from JSON payload without coercion", () => {
      const schemaProperties = {
        count: { type: "integer" },
        enabled: { type: "boolean" },
        tags: { type: "array" },
      };
      const stdinContent = '{"count": 5, "enabled": true, "tags": ["a", "b"]}';

      const { args } = parseToolArgs(["."], schemaProperties, stdinContent);

      expect(args).toEqual({ count: 5, enabled: true, tags: ["a", "b"] });
    });

    it("normalizes dashed JSON keys to schema underscore keys", () => {
      const schemaProperties = {
        issue_number: { type: "integer" },
      };
      const stdinContent = '{"issue-number": 99}';

      const { args } = parseToolArgs(["."], schemaProperties, stdinContent);

      expect(args).toEqual({ issue_number: 99 });
    });

    it("falls through to empty args when stdinContent is null and sentinel is used", () => {
      const { args } = parseToolArgs(["."], {}, null);

      expect(args).toEqual({});
    });

    it("falls through to empty args when stdinContent is empty string", () => {
      const { args } = parseToolArgs(["."], {}, "");

      expect(args).toEqual({});
    });

    it("falls through to normal parsing when stdinContent is not valid JSON", () => {
      const schemaProperties = { body: { type: "string" } };

      const { args } = parseToolArgs(["."], schemaProperties, "not json at all");

      expect(args).toEqual({});
    });

    it("falls through when JSON is an array rather than an object", () => {
      const { args } = parseToolArgs(["."], {}, '["a","b","c"]');

      expect(args).toEqual({});
    });

    it("handles multiline JSON payload", () => {
      const schemaProperties = { body: { type: "string" } };
      const stdinContent = `{
  "body": "### Title\\n\\nLine one.\\n\\nLine two."
}`;

      const { args } = parseToolArgs(["."], schemaProperties, stdinContent);

      expect(args).toEqual({ body: "### Title\n\nLine one.\n\nLine two." });
    });
  });

  describe("per-field stdin marker ('.')", () => {
    it("substitutes '--body .' with stdin content (space-separated form)", () => {
      const schemaProperties = { title: { type: "string" }, body: { type: "string" } };
      const stdinContent = "This is a long body from stdin.";

      const { args } = parseToolArgs(["--title", "My issue", "--body", "."], schemaProperties, stdinContent);

      expect(args).toEqual({ title: "My issue", body: "This is a long body from stdin." });
    });

    it("substitutes '--body=.' with stdin content (equals-separated form)", () => {
      const schemaProperties = { body: { type: "string" } };
      const stdinContent = "Body content from stdin.";

      const { args } = parseToolArgs(["--body=."], schemaProperties, stdinContent);

      expect(args).toEqual({ body: "Body content from stdin." });
    });

    it("trims leading/trailing whitespace from stdin content", () => {
      const schemaProperties = { body: { type: "string" } };
      const stdinContent = "  \n  Trimmed content.  \n  ";

      const { args } = parseToolArgs(["--body", "."], schemaProperties, stdinContent);

      expect(args).toEqual({ body: "Trimmed content." });
    });

    it("falls back to literal '.' when stdinContent is null", () => {
      const schemaProperties = { body: { type: "string" } };

      const { args } = parseToolArgs(["--body", "."], schemaProperties, null);

      expect(args).toEqual({ body: "." });
    });

    it("falls back to literal '.' when stdinContent is empty", () => {
      const schemaProperties = { body: { type: "string" } };

      const { args } = parseToolArgs(["--body", "."], schemaProperties, "");

      expect(args).toEqual({ body: "." });
    });

    it("falls back to literal '.' when stdinContent is whitespace-only", () => {
      const schemaProperties = { body: { type: "string" } };

      const { args } = parseToolArgs(["--body", "."], schemaProperties, "   \n   ");

      expect(args).toEqual({ body: "." });
    });

    it("all fields using '.' receive the same stdin content", () => {
      const schemaProperties = { title: { type: "string" }, body: { type: "string" } };
      const stdinContent = "Shared stdin content.";

      const { args } = parseToolArgs(["--title", ".", "--body", "."], schemaProperties, stdinContent);

      expect(args).toEqual({ title: "Shared stdin content.", body: "Shared stdin content." });
    });
  });

  describe("writeStdoutAndFlush", () => {
    it("resolves immediately when stdout.write returns true (no backpressure)", async () => {
      // The beforeEach mock captures chunks and returns true (no backpressure).
      // writeStdoutAndFlush should resolve synchronously in this case.
      await writeStdoutAndFlush("hello world\n");

      expect(stdoutChunks[0]).toBe("hello world\n");
    });

    it("waits for drain event when stdout.write returns false (pipe buffer full)", async () => {
      // Arrange: stdout.write returns false (simulates full pipe buffer like a ~64KiB payload)
      /** @type {any} */
      let drainCb = null;
      stdoutSpy.mockImplementation(chunk => {
        stdoutChunks.push(String(chunk));
        return false; // signal backpressure
      });
      const onceStub = vi.spyOn(process.stdout, "once").mockImplementation((event, cb) => {
        if (event === "drain") {
          drainCb = cb;
        }
        return process.stdout;
      });

      try {
        const writePromise = writeStdoutAndFlush("large payload\n");

        // Drain callback not yet called — promise should still be pending
        let resolved = false;
        writePromise.then(() => {
          resolved = true;
        });

        // Let microtasks run; drain hasn't fired yet so still pending
        await Promise.resolve();
        expect(resolved).toBe(false);
        expect(drainCb).not.toBeNull();

        // Fire the drain event
        drainCb();

        await writePromise;
        expect(resolved).toBe(true);
        expect(stdoutChunks).toContain("large payload\n");
      } finally {
        onceStub.mockRestore();
      }
    });

    it("rejects when stdout emits error while waiting for drain (EPIPE)", async () => {
      // Arrange: stdout.write returns false, then stdout emits an error
      stdoutSpy.mockImplementation(chunk => {
        stdoutChunks.push(String(chunk));
        return false; // signal backpressure
      });
      const error = new Error("EPIPE");
      /** @type {any} */
      let errorCb = null;
      const onceStub = vi.spyOn(process.stdout, "once").mockImplementation((event, cb) => {
        if (event === "error") {
          errorCb = cb;
        }
        return process.stdout;
      });

      try {
        const writePromise = writeStdoutAndFlush("data\n");
        // Verify the error callback was registered before firing it
        expect(errorCb).not.toBeNull();
        // Fire the error event asynchronously (simulates broken pipe)
        Promise.resolve().then(() => errorCb(error));
        await expect(writePromise).rejects.toThrow("EPIPE");
      } finally {
        onceStub.mockRestore();
      }
    });

    it("formatResponse awaits stdout drain before writing to stderr (no interleaving)", async () => {
      // This test verifies that core.info (→ stderr) is NOT called until after
      // stdout has been fully drained. Before the fix, process.stdout.write()
      // returning false would allow subsequent core.info calls to reach stderr
      // while stdout was still buffering, corrupting combined output.
      const callOrder = [];
      /** @type {any} */
      let drainCb = null;

      stdoutSpy.mockImplementation(chunk => {
        callOrder.push({ stream: "stdout", data: String(chunk) });
        return false; // simulate pipe buffer full
      });
      const onceStub = vi.spyOn(process.stdout, "once").mockImplementation((event, cb) => {
        if (event === "drain") {
          drainCb = cb;
        }
        return process.stdout;
      });
      global.core.info = vi.fn(msg => {
        callOrder.push({ stream: "stderr-info", data: String(msg) });
      });

      const body = {
        result: {
          content: [{ type: "text", text: "large json output" }],
        },
      };

      try {
        const formatPromise = formatResponse(body, "agenticworkflows");

        // Yield to microtasks — stdout write is queued, drain not yet fired
        await Promise.resolve();

        // core.info should NOT have been called yet (stdout hasn't drained)
        expect(callOrder.filter(e => e.stream === "stderr-info")).toHaveLength(0);

        // Now fire the drain event
        if (drainCb) drainCb();
        await formatPromise;

        // After drain: stdout write AND then core.info (order preserved)
        const stdoutIdx = callOrder.findIndex(e => e.stream === "stdout");
        const infoIdx = callOrder.findIndex(e => e.stream === "stderr-info");
        expect(stdoutIdx).toBeGreaterThanOrEqual(0);
        expect(infoIdx).toBeGreaterThan(stdoutIdx);
      } finally {
        onceStub.mockRestore();
      }
    });
  });
});
