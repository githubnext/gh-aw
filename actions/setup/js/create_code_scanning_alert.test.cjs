import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockContext = {
  runId: "12345",
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  payload: {
    repository: {
      html_url: "https://github.com/test-owner/test-repo",
    },
  },
};

global.core = mockCore;
global.context = mockContext;

describe("create_code_scanning_alert (Handler Factory Architecture)", () => {
  let handler;
  const sarifFile = path.join(process.cwd(), "code-scanning-alert.sarif");

  beforeEach(async () => {
    vi.clearAllMocks();

    // Clean up any existing SARIF file
    if (fs.existsSync(sarifFile)) {
      fs.unlinkSync(sarifFile);
    }

    const { main } = require("./create_code_scanning_alert.cjs");
    handler = await main({
      max: 0, // unlimited
      driver: "Test Security Scanner",
      workflow_filename: "test-workflow",
    });
  });

  afterEach(() => {
    // Clean up SARIF file after each test
    if (fs.existsSync(sarifFile)) {
      fs.unlinkSync(sarifFile);
    }
  });

  it("should return a function from main()", async () => {
    const { main } = require("./create_code_scanning_alert.cjs");
    const result = await main({});
    expect(typeof result).toBe("function");
  });

  it("should create SARIF file for a single security finding", async () => {
    const message = {
      type: "create_code_scanning_alert",
      file: "src/app.js",
      line: 42,
      severity: "error",
      message: "SQL injection vulnerability detected",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.findingsCount).toBe(1);
    expect(fs.existsSync(sarifFile)).toBe(true);

    const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
    expect(sarifContent.version).toBe("2.1.0");
    expect(sarifContent.runs).toHaveLength(1);
    expect(sarifContent.runs[0].results).toHaveLength(1);

    const firstResult = sarifContent.runs[0].results[0];
    expect(firstResult.message.text).toBe("SQL injection vulnerability detected");
    expect(firstResult.level).toBe("error");
    expect(firstResult.locations[0].physicalLocation.artifactLocation.uri).toBe("src/app.js");
    expect(firstResult.locations[0].physicalLocation.region.startLine).toBe(42);
  });

  it("should accumulate multiple findings", async () => {
    const message1 = {
      type: "create_code_scanning_alert",
      file: "src/app.js",
      line: 42,
      severity: "error",
      message: "SQL injection vulnerability",
    };

    const message2 = {
      type: "create_code_scanning_alert",
      file: "src/utils.js",
      line: 15,
      severity: "warning",
      message: "Potential XSS vulnerability",
    };

    const result1 = await handler(message1, {});
    expect(result1.success).toBe(true);
    expect(result1.findingsCount).toBe(1);

    const result2 = await handler(message2, {});
    expect(result2.success).toBe(true);
    expect(result2.findingsCount).toBe(2);

    const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
    expect(sarifContent.runs[0].results).toHaveLength(2);
  });

  it("should respect max findings limit", async () => {
    const { main } = require("./create_code_scanning_alert.cjs");
    const limitedHandler = await main({ max: 1 });

    const message1 = {
      type: "create_code_scanning_alert",
      file: "src/app.js",
      line: 42,
      severity: "error",
      message: "First vulnerability",
    };

    const message2 = {
      type: "create_code_scanning_alert",
      file: "src/utils.js",
      line: 15,
      severity: "warning",
      message: "Second vulnerability",
    };

    const result1 = await limitedHandler(message1, {});
    expect(result1.success).toBe(true);

    const result2 = await limitedHandler(message2, {});
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("Max count");
  });

  it("should validate required fields", async () => {
    const message = {
      type: "create_code_scanning_alert",
      // Missing required fields
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain('Missing required field "file"');
  });

  it("should validate severity levels", async () => {
    const message = {
      type: "create_code_scanning_alert",
      file: "src/app.js",
      line: 42,
      severity: "invalid-severity",
      message: "Test message",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid severity level");
  });

  it("should map severity to SARIF levels", async () => {
    const testCases = [
      { severity: "error", expected: "error" },
      { severity: "warning", expected: "warning" },
      { severity: "info", expected: "note" },
      { severity: "note", expected: "note" },
    ];

    for (const testCase of testCases) {
      // Clean up before each iteration
      if (fs.existsSync(sarifFile)) {
        fs.unlinkSync(sarifFile);
      }

      const { main } = require("./create_code_scanning_alert.cjs");
      const testHandler = await main({});

      const message = {
        type: "create_code_scanning_alert",
        file: "src/test.js",
        line: 10,
        severity: testCase.severity,
        message: "Test",
      };

      const result = await testHandler(message, {});
      expect(result.success).toBe(true);

      const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
      expect(sarifContent.runs[0].results[0].level).toBe(testCase.expected);
    }
  });

  it("should support optional column specification", async () => {
    const message = {
      type: "create_code_scanning_alert",
      file: "src/app.js",
      line: 42,
      column: 10,
      severity: "error",
      message: "Vulnerability",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);

    const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
    expect(sarifContent.runs[0].results[0].locations[0].physicalLocation.region.startColumn).toBe(10);
  });

  it("should support optional ruleIdSuffix", async () => {
    const message = {
      type: "create_code_scanning_alert",
      file: "src/app.js",
      line: 42,
      severity: "error",
      message: "Vulnerability",
      ruleIdSuffix: "sql-injection",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);

    const sarifContent = JSON.parse(fs.readFileSync(sarifFile, "utf8"));
    expect(sarifContent.runs[0].results[0].ruleId).toBe("test-workflow-sql-injection");
  });

  it("should validate ruleIdSuffix format", async () => {
    const message = {
      type: "create_code_scanning_alert",
      file: "src/app.js",
      line: 42,
      severity: "error",
      message: "Vulnerability",
      ruleIdSuffix: "invalid suffix!",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("must contain only alphanumeric characters");
  });

  it("should set correct outputs", async () => {
    const message = {
      type: "create_code_scanning_alert",
      file: "src/app.js",
      line: 42,
      severity: "error",
      message: "Vulnerability",
    };

    await handler(message, {});

    expect(mockCore.setOutput).toHaveBeenCalledWith("sarif_file", sarifFile);
    expect(mockCore.setOutput).toHaveBeenCalledWith("findings_count", 1);
    expect(mockCore.setOutput).toHaveBeenCalledWith("artifact_uploaded", "pending");
    expect(mockCore.setOutput).toHaveBeenCalledWith("codeql_uploaded", "pending");
  });
});
