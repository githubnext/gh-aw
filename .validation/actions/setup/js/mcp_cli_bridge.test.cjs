import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { formatResponse, hasStdinJsonPayload, parseToolArgs, readStdinSync } from "./mcp_cli_bridge.cjs";

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
});
