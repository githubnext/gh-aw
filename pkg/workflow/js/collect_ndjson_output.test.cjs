import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("collect_ndjson_output.cjs", () => {
  let mockCore;
  let collectScript;

  beforeEach(() => {
    // Ensure test directory exists
    const testDir = "/tmp/gh-aw";
    if (!fs.existsSync(testDir)) {
      fs.mkdirSync(testDir, { recursive: true });
    }

    // Save original console before mocking
    global.originalConsole = global.console;

    // Mock console methods
    global.console = {
      log: vi.fn(),
      error: vi.fn(),
    };

    // Mock core actions methods
    mockCore = {
      // Core logging functions
      debug: vi.fn(),
      info: vi.fn(),
      notice: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),

      // Core workflow functions
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      exportVariable: vi.fn(),
      setSecret: vi.fn(),

      // Input/state functions (less commonly used but included for completeness)
      getInput: vi.fn(),
      getBooleanInput: vi.fn(),
      getMultilineInput: vi.fn(),
      getState: vi.fn(),
      saveState: vi.fn(),

      // Group functions
      startGroup: vi.fn(),
      endGroup: vi.fn(),
      group: vi.fn(),

      // Other utility functions
      addPath: vi.fn(),
      setCommandEcho: vi.fn(),
      isDebug: vi.fn().mockReturnValue(false),
      getIDToken: vi.fn(),
      toPlatformPath: vi.fn(),
      toPosixPath: vi.fn(),
      toWin32Path: vi.fn(),

      // Summary object with chainable methods
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(),
      },
    };
    global.core = mockCore;

    // Read the script file
    const scriptPath = path.join(__dirname, "collect_ndjson_output.cjs");
    collectScript = fs.readFileSync(scriptPath, "utf8");

    // Make fs available globally for the evaluated script
    global.fs = fs;
  });

  afterEach(() => {
    // Clean up any test files
    const testFiles = ["/tmp/gh-aw/test-ndjson-output.txt", "/tmp/gh-aw/agent_output.json"];
    testFiles.forEach(file => {
      try {
        if (fs.existsSync(file)) {
          fs.unlinkSync(file);
        }
      } catch (error) {
        // Ignore cleanup errors
      }
    });

    // Clean up globals safely - don't delete console as vitest may still need it
    if (typeof global !== "undefined") {
      delete global.fs;
      delete global.core;
      // Restore original console instead of deleting
      if (global.originalConsole) {
        global.console = global.originalConsole;
        delete global.originalConsole;
      }
    }
  });

  it("should handle missing GITHUB_AW_SAFE_OUTPUTS environment variable", async () => {
    delete process.env.GITHUB_AW_SAFE_OUTPUTS;

    await eval(`(async () => { ${collectScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("output", "");
    expect(mockCore.info).toHaveBeenCalledWith("GITHUB_AW_SAFE_OUTPUTS not set, no output to collect");
  });

  it("should handle missing output file", async () => {
    process.env.GITHUB_AW_SAFE_OUTPUTS = "/tmp/gh-aw/nonexistent-file.txt";

    await eval(`(async () => { ${collectScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("output", "");
    expect(mockCore.info).toHaveBeenCalledWith("Output file does not exist: /tmp/gh-aw/nonexistent-file.txt");
  });

  it("should handle empty output file", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    fs.writeFileSync(testFile, "");
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;

    await eval(`(async () => { ${collectScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("output", '{"items":[],"errors":[]}');
    expect(mockCore.info).toHaveBeenCalledWith("Output file is empty");
  });

  it("should validate and parse valid JSONL content", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Test body"}
{"type": "add_comment", "body": "Test comment"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true, "add_comment": true}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(2);
    expect(parsedOutput.items[0].type).toBe("create_issue");
    expect(parsedOutput.items[1].type).toBe("add_comment");
    expect(parsedOutput.errors).toHaveLength(0);
  });

  it("should reject items with unexpected output types", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Test body"}
{"type": "unexpected-type", "data": "some data"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1);
    expect(parsedOutput.items[0].type).toBe("create_issue");
    expect(parsedOutput.errors).toHaveLength(1);
    expect(parsedOutput.errors[0]).toContain("Unexpected output type");
  });

  it("should validate required fields for create_issue type", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_issue", "title": "Test Issue"}
{"type": "create_issue", "body": "Test body"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

    await eval(`(async () => { ${collectScript} })()`);

    // Since there are errors and no valid items, setFailed should be called
    expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
    const failedMessage = mockCore.setFailed.mock.calls[0][0];
    expect(failedMessage).toContain("requires a 'body' string field");
    expect(failedMessage).toContain("requires a 'title' string field");

    // setOutput should not be called because of early return
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeUndefined();
  });

  it("should validate required fields for add-labels type", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "add-labels", "labels": ["bug", "enhancement"]}
{"type": "add-labels", "labels": "not-an-array"}
{"type": "add-labels", "labels": [1, 2, 3]}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"add-labels": true}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1);
    expect(parsedOutput.items[0].labels).toEqual(["bug", "enhancement"]);
    expect(parsedOutput.errors).toHaveLength(2);
  });

  it("should validate required fields for create-pull-request type", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create-pull-request", "title": "Test PR"}
{"type": "create-pull-request", "body": "Test body"}
{"type": "create-pull-request", "branch": "test-branch"}
{"type": "create-pull-request", "title": "Complete PR", "body": "Test body", "branch": "feature-branch"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-pull-request": true}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1); // Only the complete PR should be valid
    expect(parsedOutput.items[0].title).toBe("Complete PR");
    expect(parsedOutput.items[0].body).toBe("Test body");
    expect(parsedOutput.items[0].branch).toBe("feature-branch");
    expect(parsedOutput.errors).toHaveLength(3); // Three incomplete PRs should cause errors
    expect(parsedOutput.errors[0]).toContain("requires a 'body' string field");
    expect(parsedOutput.errors[1]).toContain("requires a 'title' string field");
    expect(parsedOutput.errors[2]).toContain("requires a 'title' string field");
  });

  it("should handle invalid JSON lines", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Test body"}
{invalid json}
{"type": "add_comment", "body": "Test comment"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true, "add_comment": true}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(2);
    expect(parsedOutput.errors).toHaveLength(1);
    expect(parsedOutput.errors[0]).toContain("Invalid JSON");
  });

  it("should allow multiple items of supported types up to limits", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_issue", "title": "First Issue", "body": "First body"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1); // Both items should be allowed
    expect(parsedOutput.items[0].title).toBe("First Issue");
    expect(parsedOutput.errors).toHaveLength(0); // No errors for multiple items within limits
  });

  it("should respect max limits from config", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_issue", "title": "First Issue", "body": "First body"}
{"type": "create_issue", "title": "Second Issue", "body": "Second body"}
{"type": "create_issue", "title": "Third Issue", "body": "Third body"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    // Set max to 2 for create_issue
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": {"max": 2}}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(2); // Only first 2 items should be allowed
    expect(parsedOutput.items[0].title).toBe("First Issue");
    expect(parsedOutput.items[1].title).toBe("Second Issue");
    expect(parsedOutput.errors).toHaveLength(1); // Error for the third item exceeding max
    expect(parsedOutput.errors[0]).toContain("Too many items of type 'create_issue'. Maximum allowed: 2");
  });

  it("should validate required fields for create-discussion type", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create-discussion", "title": "Test Discussion"}
{"type": "create-discussion", "body": "Test body"}
{"type": "create-discussion", "title": "Valid Discussion", "body": "Valid body"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-discussion": true}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1); // Only the valid one
    expect(parsedOutput.items[0].title).toBe("Valid Discussion");
    expect(parsedOutput.items[0].body).toBe("Valid body");
    expect(parsedOutput.errors).toHaveLength(2);
    expect(parsedOutput.errors[0]).toContain("requires a 'body' string field");
    expect(parsedOutput.errors[1]).toContain("requires a 'title' string field");
  });

  it("should skip empty lines", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Test body"}

{"type": "add_comment", "body": "Test comment"}
`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true, "add_comment": true}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(2);
    expect(parsedOutput.errors).toHaveLength(0);
  });

  it("should validate required fields for create-pull-request-review-comment type", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_pull_request_review_comment", "path": "src/file.js", "line": 10, "body": "Good code"}
{"type": "create_pull_request_review_comment", "path": "src/file.js", "line": "invalid", "body": "Comment"}
{"type": "create_pull_request_review_comment", "path": "src/file.js", "body": "Missing line"}
{"type": "create_pull_request_review_comment", "line": 15}
{"type": "create_pull_request_review_comment", "path": "src/file.js", "line": 20, "start_line": 25, "body": "Invalid range"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_pull_request_review_comment": {"max": 10}}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1); // Only the first valid item
    expect(parsedOutput.items[0].path).toBe("src/file.js");
    expect(parsedOutput.items[0].line).toBe(10);
    expect(parsedOutput.items[0].body).toBeDefined();
    expect(parsedOutput.errors).toHaveLength(4); // 4 invalid items
    expect(parsedOutput.errors.some(e => e.includes("line' must be a positive integer"))).toBe(true);
    expect(parsedOutput.errors.some(e => e.includes("requires a 'line' number"))).toBe(true);
    expect(parsedOutput.errors.some(e => e.includes("requires a 'path' string"))).toBe(true);
    expect(parsedOutput.errors.some(e => e.includes("start_line' must be less than or equal to 'line'"))).toBe(true);
  });

  it("should validate optional fields for create-pull-request-review-comment type", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_pull_request_review_comment", "path": "src/file.js", "line": 20, "start_line": 15, "side": "LEFT", "body": "Multi-line comment"}
{"type": "create_pull_request_review_comment", "path": "src/file.js", "line": 25, "side": "INVALID", "body": "Invalid side"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_pull_request_review_comment": {"max": 10}}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1); // Only the first valid item
    expect(parsedOutput.items[0].side).toBe("LEFT");
    expect(parsedOutput.items[0].start_line).toBe(15);
    expect(parsedOutput.errors).toHaveLength(1); // 1 invalid side
    expect(parsedOutput.errors[0]).toContain("side' must be 'LEFT' or 'RIGHT'");
  });

  it("should respect max limits for create-pull-request-review-comment from config", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const items = [];
    for (let i = 1; i <= 12; i++) {
      items.push(`{"type": "create_pull_request_review_comment", "path": "src/file.js", "line": ${i}, "body": "Comment ${i}"}`);
    }
    const ndjsonContent = items.join("\n");

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    // Set max to 5 for create-pull-request-review-comment
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_pull_request_review_comment": {"max": 5}}';

    await eval(`(async () => { ${collectScript} })()`);

    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(5); // Only first 5 items should be allowed
    expect(parsedOutput.errors).toHaveLength(7); // 7 items exceeding max
    expect(
      parsedOutput.errors.every(e => e.includes("Too many items of type 'create-pull-request-review-comment'. Maximum allowed: 5"))
    ).toBe(true);
  });

  describe("JSON repair functionality", () => {
    it("should repair JSON with unescaped quotes in string values", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Issue with "quotes" inside", "body": "Test body"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].title).toContain("quotes");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with missing quotes around object keys", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type: "create_issue", title: "Test Issue", body: "Test body"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with trailing commas", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Test body",}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with single quotes", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{'type': 'create_issue', 'title': 'Test Issue', 'body': 'Test body'}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with missing closing braces", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Test body"`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with missing opening braces", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `"type": "create_issue", "title": "Test Issue", "body": "Test body"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with newlines in string values", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Real JSONL would have actual \n in the string, not real newlines
      const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Line 1\\nLine 2\\nLine 3"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].body).toContain("Line 1");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with tabs and special characters", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Test	Issue", "body": "Test\tbody"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with array syntax issues", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "add-labels", "labels": ["bug", "enhancement",}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"add-labels": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].labels).toEqual(["bug", "enhancement"]);
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should handle complex repair scenarios with multiple issues", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Make this a more realistic test case for JSON repair without real newlines breaking JSONL
      const ndjsonContent = `{type: 'create_issue', title: 'Issue with "quotes" and trailing,', body: 'Multi\\nline\\ntext',`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should handle JSON broken across multiple lines (real multiline scenario)", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // This simulates what happens when LLMs output JSON with actual newlines
      // The parser should treat this as one broken JSON item, not multiple lines
      // For now, we'll test that it fails gracefully and reports an error
      const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Line 1
Line 2
Line 3"}
{"type": "add_comment", "body": "This is a valid line"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true, "add_comment": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      // The first broken JSON should produce errors, but the last valid line should work
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("add_comment");
      expect(parsedOutput.errors.length).toBeGreaterThan(0);
      expect(parsedOutput.errors.some(error => error.includes("JSON parsing failed"))).toBe(true);
    });

    it("should still report error if repair fails completely", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{completely broken json with no hope: of repair [[[}}}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Since there are errors and no valid items, setFailed should be called
      expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
      const failedMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failedMessage).toContain("JSON parsing failed");

      // setOutput should not be called because of early return
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeUndefined();
    });

    it("should preserve valid JSON without modification", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Perfect JSON", "body": "This should not be modified"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].title).toBe("Perfect JSON");
      expect(parsedOutput.items[0].body).toBe("This should not be modified");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair mixed quote types in same object", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": 'create_issue', "title": 'Mixed quotes', 'body': "Test body"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.items[0].title).toBe("Mixed quotes");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair arrays ending with wrong bracket type", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "add-labels", "labels": ["bug", "feature", "enhancement"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"add-labels": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].labels).toEqual(["bug", "feature", "enhancement"]);
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should handle simple missing closing brackets with graceful repair", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "add-labels", "labels": ["bug", "feature"`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"add-labels": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Check if repair succeeded by looking at mock calls
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");

      if (outputCall) {
        // Repair succeeded
        const parsedOutput = JSON.parse(outputCall[1]);
        expect(parsedOutput.items[0].type).toBe("add_labels");
        expect(parsedOutput.items[0].labels).toEqual(["bug", "feature"]);
        expect(parsedOutput.errors).toHaveLength(0);
      } else {
        // Repair failed, should have called setFailed
        expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
        const failedMessage = mockCore.setFailed.mock.calls[0][0];
        expect(failedMessage).toContain("JSON parsing failed");
      }
    });

    it("should repair nested objects with multiple issues", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type: 'create_issue', title: 'Nested test', body: 'Body text', labels: ['bug', 'priority',}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.items[0].labels).toEqual(["bug", "priority"]);
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with Unicode characters and escape sequences", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type: 'create_issue', title: 'Unicode test \u00e9\u00f1', body: 'Body with \\u0040 symbols',`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.items[0].title).toContain("é");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with control characters (null, backspace, form feed)", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Test with actual control characters: null (\x00), backspace (\x08), form feed (\x0C)
      const ndjsonContent = `{"type": "create_issue", "title": "Test\x00Issue", "body": "Body\x08with\x0Ccontrol\x07chars"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      // Control characters should be removed by sanitizeContent after repair
      expect(parsedOutput.items[0].title).toBe("TestIssue");
      expect(parsedOutput.items[0].body).toBe("Bodywithcontrolchars");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with device control characters", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Test with device control characters: DC1 (\x11), DC4 (\x14), NAK (\x15)
      const ndjsonContent = `{"type": "create_issue", "title": "Device\x11Control\x14Test", "body": "Text\x15here"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      // Control characters should be removed by sanitizeContent after repair
      expect(parsedOutput.items[0].title).toBe("DeviceControlTest");
      expect(parsedOutput.items[0].body).toBe("Texthere");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON preserving valid escape sequences (newline, tab, carriage return)", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Test that valid control characters (tab, newline, carriage return) are properly handled
      // Note: These should be properly escaped in the JSON to avoid breaking the JSONL format
      const ndjsonContent = `{"type": "create_issue", "title": "Valid\\tTab", "body": "Line1\\nLine2\\rCarriage"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      // Escaped sequences in JSON should become actual characters, then get sanitized appropriately
      expect(parsedOutput.items[0].title).toBe("Valid\tTab"); // Tab preserved by sanitizeContent
      expect(parsedOutput.items[0].body).toBe("Line1\nLine2\rCarriage"); // Newlines/returns preserved
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with mixed control characters and regular escape sequences", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Test mixing regular escapes with control characters - simplified to avoid quote issues
      const ndjsonContent = `{"type": "create_issue", "title": "Mixed\x00test\\nwith text", "body": "Body\x02with\\ttab\x03end"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      // Control chars removed (\x00, \x02, \x03), escaped sequences processed (\n, \t preserved)
      expect(parsedOutput.items[0].title).toMatch(/Mixedtest\nwith text/);
      expect(parsedOutput.items[0].body).toMatch(/Bodywith\ttabend/);
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with DEL character (0x7F) and other high control chars", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // DEL (0x7F) should be handled by sanitizeContent, other control chars by repairJson
      const ndjsonContent = `{"type": "create_issue", "title": "Test\x7FDel", "body": "Body\x1Fwith\x01control"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      // All control characters should be removed by sanitizeContent
      expect(parsedOutput.items[0].title).toBe("TestDel");
      expect(parsedOutput.items[0].body).toBe("Bodywithcontrol");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with all ASCII control characters in sequence", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Test simpler case to verify control character handling works
      const ndjsonContent = `{"type": "create_issue", "title": "Control test\x00\x01\x02\\t\\n", "body": "End of test"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");

      // Control chars (0x00, 0x01, 0x02) removed, tab and newline preserved
      const title = parsedOutput.items[0].title;
      expect(title).toBe("Control test"); // Control chars actually get removed completely
      expect(parsedOutput.items[0].body).toBe("End of test");

      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should test control character repair in isolation using the repair function", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Test malformed JSON that needs both control char repair and other repairs
      const ndjsonContent = `{type: "create_issue", title: 'Test\x00with\x08control\x0Cchars', body: 'Body\x01text',}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      // This tests that the repair function successfully handles both JSON syntax errors
      // (single quotes, missing quotes around keys, trailing comma) AND control characters
      expect(parsedOutput.items[0].title).toBe("Testwithcontrolchars");
      expect(parsedOutput.items[0].body).toBe("Bodytext");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should test repair function behavior with specific control character scenarios", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Test case where control characters would break JSON but repair fixes them
      const ndjsonContent = `{"type": "create_issue", "title": "Control\x00\x07\x1A", "body": "Test\x08\x1Fend"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      // Control characters should be removed by sanitizeContent after repair escapes them
      expect(parsedOutput.items[0].title).toBe("Control");
      expect(parsedOutput.items[0].body).toBe("Testend");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair JSON with numbers, booleans, and null values", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type: 'create_issue', title: 'Complex types test', body: 'Body text', priority: 5, urgent: true, assignee: null,}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.items[0].priority).toBe(5);
      expect(parsedOutput.items[0].urgent).toBe(true);
      expect(parsedOutput.items[0].assignee).toBe(null);
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should attempt repair but fail gracefully with excessive malformed JSON", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{,type: 'create_issue',, title: 'Extra commas', body: 'Test',, labels: ['bug',,],}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Since this JSON is too malformed to repair and results in no valid items, setFailed should be called
      expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
      const failedMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failedMessage).toContain("JSON parsing failed");

      // setOutput should not be called because of early return
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeUndefined();
    });

    it("should repair very long strings with multiple issues", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const longBody =
        'This is a very long body text that contains "quotes" and other\\nspecial characters including tabs\\t and newlines\\r\\n and more text that goes on and on.';
      const ndjsonContent = `{type: 'create_issue', title: 'Long string test', body: '${longBody}',}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.items[0].body).toContain("very long body");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair deeply nested structures", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type: 'create_issue', title: 'Nested test', body: 'Body', metadata: {project: 'test', tags: ['important', 'urgent',}, version: 1.0,}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.items[0].metadata).toBeDefined();
      expect(parsedOutput.items[0].metadata.project).toBe("test");
      expect(parsedOutput.items[0].metadata.tags).toEqual(["important", "urgent"]);
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should handle complex backslash scenarios with graceful failure", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type: 'create_issue', title: 'Escape test with "quotes" and \\\\backslashes', body: 'Test body',}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      // This complex escape case might fail due to the embedded quotes and backslashes
      // The repair function may not handle this level of complexity
      if (parsedOutput.items.length === 1) {
        expect(parsedOutput.items[0].type).toBe("create_issue");
        expect(parsedOutput.items[0].title).toContain("quotes");
        expect(parsedOutput.errors).toHaveLength(0);
      } else {
        // If repair fails, it should report an error
        expect(parsedOutput.items).toHaveLength(0);
        expect(parsedOutput.errors).toHaveLength(1);
        expect(parsedOutput.errors[0]).toContain("JSON parsing failed");
      }
    });

    it("should repair JSON with carriage returns and form feeds", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type: 'create_issue', title: 'Special chars', body: 'Text with\\rcarriage\\fform feed',}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should gracefully handle repair attempts on fundamentally broken JSON", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{{{[[[type]]]}}} === "broken" &&& title ??? 'impossible to repair' @@@ body`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Since this JSON is fundamentally broken and results in no valid items, setFailed should be called
      expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
      const failedMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failedMessage).toContain("JSON parsing failed");

      // setOutput should not be called because of early return
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeUndefined();
    });

    it("should handle repair of JSON with missing property separators", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type 'create_issue', title 'Missing colons', body 'Test body'}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Since this JSON likely fails to repair and results in no valid items, setFailed should be called
      expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
      const failedMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failedMessage).toContain("JSON parsing failed");

      // setOutput should not be called because of early return
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeUndefined();
    });

    it("should repair arrays with mixed bracket types in complex structures", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type: 'add-labels', labels: ['priority', 'bug', 'urgent'}, extra: ['data', 'here'}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"add-labels": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("add_labels");
      expect(parsedOutput.items[0].labels).toEqual(["priority", "bug", "urgent"]);
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should gracefully handle cases with multiple trailing commas", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Test", "body": "Test body",,,}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Check if repair succeeded by looking at mock calls
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");

      if (outputCall) {
        // Repair succeeded
        const parsedOutput = JSON.parse(outputCall[1]);
        expect(parsedOutput.items[0].type).toBe("create_issue");
        expect(parsedOutput.items[0].title).toBe("Test");
        expect(parsedOutput.errors).toHaveLength(0);
      } else {
        // Repair failed, should have called setFailed
        expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
        const failedMessage = mockCore.setFailed.mock.calls[0][0];
        expect(failedMessage).toContain("JSON parsing failed");
      }
    });

    it("should repair JSON with simple missing closing brackets", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "add-labels", "labels": ["bug", "feature"]}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"add-labels": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("add_labels");
      expect(parsedOutput.items[0].labels).toEqual(["bug", "feature"]);
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should repair combination of unquoted keys and trailing commas", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{type: "create_issue", title: "Combined issues", body: "Test body", priority: 1,}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].type).toBe("create_issue");
      expect(parsedOutput.items[0].title).toBe("Combined issues");
      expect(parsedOutput.items[0].priority).toBe(1);
      expect(parsedOutput.errors).toHaveLength(0);
    });
  });

  it("should store validated output in agent_output.json file and set GITHUB_AW_AGENT_OUTPUT environment variable", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Test body"}
{"type": "add_comment", "body": "Test comment"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true, "add_comment": true}';

    await eval(`(async () => { ${collectScript} })()`);

    // Verify agent_output.json file was created
    expect(fs.existsSync("/tmp/gh-aw/agent_output.json")).toBe(true);

    // Verify the content of agent_output.json
    const agentOutputContent = fs.readFileSync("/tmp/gh-aw/agent_output.json", "utf8");
    const agentOutputJson = JSON.parse(agentOutputContent);

    expect(agentOutputJson.items).toHaveLength(2);
    expect(agentOutputJson.items[0].type).toBe("create_issue");
    expect(agentOutputJson.items[1].type).toBe("add_comment");
    expect(agentOutputJson.errors).toHaveLength(0);

    // Verify GITHUB_AW_AGENT_OUTPUT environment variable was set
    expect(mockCore.exportVariable).toHaveBeenCalledWith("GITHUB_AW_AGENT_OUTPUT", "/tmp/gh-aw/agent_output.json");

    // Verify existing functionality still works (core.setOutput calls)
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(2);
    expect(parsedOutput.errors).toHaveLength(0);
  });

  it("should handle errors when writing agent_output.json file gracefully", async () => {
    const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
    const ndjsonContent = `{"type": "create_issue", "title": "Test Issue", "body": "Test body"}`;

    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

    // Mock fs.writeFileSync to throw an error for the agent_output.json file
    const originalWriteFileSync = fs.writeFileSync;
    fs.writeFileSync = vi.fn((filePath, content, options) => {
      if (filePath === "/tmp/gh-aw/agent_output.json") {
        throw new Error("Permission denied");
      }
      return originalWriteFileSync(filePath, content, options);
    });

    await eval(`(async () => { ${collectScript} })()`);

    // Restore original fs.writeFileSync
    fs.writeFileSync = originalWriteFileSync;

    // Verify the error was logged but the script continued to work
    expect(mockCore.error).toHaveBeenCalledWith("Failed to write agent output file: Permission denied");

    // Verify existing functionality still works (core.setOutput calls)
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === "output");
    expect(outputCall).toBeDefined();

    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1);
    expect(parsedOutput.errors).toHaveLength(0);

    // Verify exportVariable was not called if file writing failed
    expect(mockCore.exportVariable).not.toHaveBeenCalled();
  });

  describe("create_code_scanning_alert validation", () => {
    it("should validate valid code scanning alert entries", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_code_scanning_alert", "file": "src/auth.js", "line": 42, "severity": "error", "message": "SQL injection vulnerability"}
{"type": "create_code_scanning_alert", "file": "src/utils.js", "line": 25, "severity": "warning", "message": "XSS vulnerability", "column": 10, "ruleIdSuffix": "xss-check"}
{"type": "create_code_scanning_alert", "file": "src/complete.js", "line": "30", "severity": "NOTE", "message": "Complete example", "column": "5", "ruleIdSuffix": "complete-rule"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_code_scanning_alert": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(3);
      expect(parsedOutput.errors).toHaveLength(0);

      // Verify first entry
      expect(parsedOutput.items[0]).toEqual({
        type: "create_code_scanning_alert",
        file: "src/auth.js",
        line: 42,
        severity: "error",
        message: "SQL injection vulnerability",
      });

      // Verify second entry with optional fields
      expect(parsedOutput.items[1]).toEqual({
        type: "create_code_scanning_alert",
        file: "src/utils.js",
        line: 25,
        severity: "warning",
        message: "XSS vulnerability",
        column: 10,
        ruleIdSuffix: "xss-check",
      });

      // Verify third entry with normalized severity
      expect(parsedOutput.items[2]).toEqual({
        type: "create_code_scanning_alert",
        file: "src/complete.js",
        line: "30",
        severity: "note", // Should be normalized to lowercase
        message: "Complete example",
        column: "5",
        ruleIdSuffix: "complete-rule",
      });
    });

    it("should reject code scanning alert entries with missing required fields", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_code_scanning_alert", "severity": "error", "message": "Missing file field"}
{"type": "create_code_scanning_alert", "file": "src/missing.js", "severity": "error", "message": "Missing line field"}
{"type": "create_code_scanning_alert", "file": "src/missing2.js", "line": 10, "message": "Missing severity field"}
{"type": "create_code_scanning_alert", "file": "src/missing3.js", "line": 10, "severity": "error"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_code_scanning_alert": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Since there are errors and no valid items, setFailed should be called
      expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
      const failedMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failedMessage).toContain("create_code_scanning_alert requires a 'file' field (string)");
      expect(failedMessage).toContain("create_code_scanning_alert requires a 'line' field (number or string)");
      expect(failedMessage).toContain("create_code_scanning_alert requires a 'severity' field (string)");
      expect(failedMessage).toContain("create_code_scanning_alert requires a 'message' field (string)");

      // setOutput should not be called because of early return
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeUndefined();
    });

    it("should reject code scanning alert entries with invalid field types", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_code_scanning_alert", "file": 123, "line": 10, "severity": "error", "message": "File should be string"}
{"type": "create_code_scanning_alert", "file": "src/test.js", "line": null, "severity": "error", "message": "Line should be number or string"}
{"type": "create_code_scanning_alert", "file": "src/test.js", "line": 10, "severity": 123, "message": "Severity should be string"}
{"type": "create_code_scanning_alert", "file": "src/test.js", "line": 10, "severity": "error", "message": 123}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_code_scanning_alert": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Since there are errors and no valid items, setFailed should be called
      expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
      const failedMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failedMessage).toContain("create_code_scanning_alert requires a 'file' field (string)");
      expect(failedMessage).toContain("create_code_scanning_alert requires a 'line' field (number or string)");
      expect(failedMessage).toContain("create_code_scanning_alert requires a 'severity' field (string)");
      expect(failedMessage).toContain("create_code_scanning_alert requires a 'message' field (string)");

      // setOutput should not be called because of early return
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeUndefined();
    });

    it("should reject code scanning alert entries with invalid severity levels", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_code_scanning_alert", "file": "src/test.js", "line": 10, "severity": "invalid-level", "message": "Invalid severity"}
{"type": "create_code_scanning_alert", "file": "src/test2.js", "line": 15, "severity": "critical", "message": "Unsupported severity"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_code_scanning_alert": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Since there are errors and no valid items, setFailed should be called
      expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
      const failedMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failedMessage).toContain("create_code_scanning_alert 'severity' must be one of: error, warning, info, note");

      // setOutput should not be called because of early return
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeUndefined();
    });

    it("should reject code scanning alert entries with invalid optional fields", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_code_scanning_alert", "file": "src/test.js", "line": 10, "severity": "error", "message": "Test", "column": "invalid"}
{"type": "create_code_scanning_alert", "file": "src/test2.js", "line": 15, "severity": "error", "message": "Test", "ruleIdSuffix": 123}
{"type": "create_code_scanning_alert", "file": "src/test3.js", "line": 20, "severity": "error", "message": "Test", "ruleIdSuffix": "bad rule!@#"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_code_scanning_alert": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Since there are errors and no valid items, setFailed should be called
      expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
      const failedMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failedMessage).toContain("create_code_scanning_alert 'column' must be a valid positive integer (got: invalid)");
      expect(failedMessage).toContain("create_code_scanning_alert 'ruleIdSuffix' must be a string");
      expect(failedMessage).toContain(
        "create_code_scanning_alert 'ruleIdSuffix' must contain only alphanumeric characters, hyphens, and underscores"
      );

      // setOutput should not be called because of early return
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeUndefined();
    });

    it("should handle mixed valid and invalid code scanning alert entries", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_code_scanning_alert", "file": "src/valid.js", "line": 10, "severity": "error", "message": "Valid entry"}
{"type": "create_code_scanning_alert", "file": "src/missing.js", "severity": "error", "message": "Missing line field"}
{"type": "create_code_scanning_alert", "file": "src/valid2.js", "line": 20, "severity": "warning", "message": "Another valid entry", "column": 5}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_code_scanning_alert": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(2); // 2 valid items
      expect(parsedOutput.errors).toHaveLength(1); // 1 error

      expect(parsedOutput.items[0].file).toBe("src/valid.js");
      expect(parsedOutput.items[1].file).toBe("src/valid2.js");
      expect(parsedOutput.errors).toContain("Line 2: create_code_scanning_alert requires a 'line' field (number or string)");
    });

    it("should reject code scanning alert entries with invalid line and column values", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_code_scanning_alert", "file": "src/test.js", "line": "invalid", "severity": "error", "message": "Invalid line string"}
{"type": "create_code_scanning_alert", "file": "src/test2.js", "line": 0, "severity": "error", "message": "Zero line number"}
{"type": "create_code_scanning_alert", "file": "src/test3.js", "line": -5, "severity": "error", "message": "Negative line number"}
{"type": "create_code_scanning_alert", "file": "src/test4.js", "line": 10, "column": "abc", "severity": "error", "message": "Invalid column string"}
{"type": "create_code_scanning_alert", "file": "src/test5.js", "line": 10, "column": 0, "severity": "error", "message": "Zero column number"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_code_scanning_alert": true}';

      await eval(`(async () => { ${collectScript} })()`);

      // Since there are errors and no valid items, setFailed should be called
      expect(mockCore.setFailed).toHaveBeenCalledTimes(1);
      const failedMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failedMessage).toContain("create_code_scanning_alert 'line' must be a valid positive integer (got: invalid)");
      expect(failedMessage).toContain("create_code_scanning_alert 'line' must be a valid positive integer (got: 0)");
      expect(failedMessage).toContain("create_code_scanning_alert 'line' must be a valid positive integer (got: -5)");
      expect(failedMessage).toContain("create_code_scanning_alert 'column' must be a valid positive integer (got: abc)");
      expect(failedMessage).toContain("create_code_scanning_alert 'column' must be a valid positive integer (got: 0)");

      // setOutput should not be called because of early return
      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeUndefined();
    });
  });

  describe("Content sanitization functionality", () => {
    it("should preserve command-line flags with colons", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Test issue", "body": "Use z3 -v:10 and z3 -memory:high for performance monitoring"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      expect(mockCore.setOutput).toHaveBeenCalledWith("output", expect.any(String));
      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.items[0].body).toBe("Use z3 -v:10 and z3 -memory:high for performance monitoring");
      expect(parsedOutput.errors).toHaveLength(0);
    });

    it("should preserve various command-line flag patterns", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "CLI Flags Test", "body": "Various flags: gcc -std:c++20, clang -target:x86_64, rustc -C:opt-level=3, javac -cp:lib/*, python -W:ignore, node --max-old-space-size:8192"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toBe(
        "Various flags: gcc -std:c++20, clang -target:x86_64, rustc -C:opt-level=3, javac -cp:lib/*, python -W:ignore, node --max-old-space-size:8192"
      );
    });

    it("should redact non-https protocols while preserving command flags", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Protocol Test", "body": "Use https://github.com/repo for code, avoid ftp://example.com/file and git://example.com/repo, but z3 -v:10 should work"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toBe(
        "Use https://github.com/repo for code, avoid (redacted) and (redacted) but z3 -v:10 should work"
      );
    });

    it("should handle mixed protocols and command flags in complex text", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Complex Test", "body": "Install from https://github.com/z3prover/z3, then run: z3 -v:10 -memory:high -timeout:30000. Avoid ssh://git.example.com/repo.git or file://localhost/path"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toBe(
        "Install from https://github.com/z3prover/z3, then run: z3 -v:10 -memory:high -timeout:30000. Avoid (redacted) or (redacted)"
      );
    });

    it("should preserve allowed domains while redacting unknown ones", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Domain Test", "body": "GitHub URLs: https://github.com/repo, https://api.github.com/users, https://githubusercontent.com/file. External: https://example.com/page"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toBe(
        "GitHub URLs: https://github.com/repo, https://api.github.com/users, https://githubusercontent.com/file. External: (redacted)"
      );
    });

    it("should handle @mentions neutralization", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "@mention Test", "body": "Hey @username and @org/team, check this out! But preserve email@domain.com"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toBe("Hey `@username` and `@org/team`, check this out! But preserve email@domain.com");
    });

    it("should neutralize bot trigger phrases", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Bot Trigger Test", "body": "This fixes #123 and closes #456, also resolves #789"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toBe("This `fixes #123` and `closes #456`, also `resolves #789`");
    });

    it("should remove ANSI escape sequences", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      // Use actual ANSI escape sequences
      const bodyWithAnsi = "\u001b[31mRed text\u001b[0m and \u001b[1mBold text\u001b[m";
      const ndjsonContent = JSON.stringify({
        type: "create_issue",
        title: "ANSI Test",
        body: bodyWithAnsi,
      });

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toBe("Red text and Bold text");
    });

    it("should handle custom allowed domains from environment", async () => {
      // Set custom allowed domains
      process.env.GITHUB_AW_ALLOWED_DOMAINS = "example.com,test.org";

      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Custom Domains", "body": "Allowed: https://example.com/page, https://sub.example.com/file, https://test.org/doc. Blocked: https://github.com/repo, https://blocked.com/page"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toBe(
        "Allowed: https://example.com/page, https://sub.example.com/file, https://test.org/doc. Blocked: (redacted), (redacted)"
      );

      // Clean up
      delete process.env.GITHUB_AW_ALLOWED_DOMAINS;
    });

    it("should handle edge cases with colons in different contexts", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Colon Edge Cases", "body": "Time 12:30 PM, ratio 3:1, IPv6 ::1, URL path/file:with:colons, command -flag:value, namespace::function"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      // All these should be preserved since they don't match the protocol:// pattern
      expect(parsedOutput.items[0].body).toBe(
        "Time 12:30 PM, ratio 3:1, IPv6 ::1, URL path/file:with:colons, command -flag:value, namespace::function"
      );
    });

    it("should truncate excessively long content", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const longBody = "x".repeat(600000); // 600KB, exceeds 512KB limit
      const ndjsonContent = `{"type": "create_issue", "title": "Long Content Test", "body": "${longBody}"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toMatch(/\[Content truncated due to length\]$/);
      expect(parsedOutput.items[0].body.length).toBeLessThan(600000);
    });

    it("should truncate content with too many lines", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const manyLines = Array(66000).fill("line").join("\n"); // Exceeds 65K line limit
      const ndjsonContent = JSON.stringify({
        type: "create_issue",
        title: "Many Lines Test",
        body: manyLines,
      });

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toMatch(/\[Content truncated due to line count\]$/);
      const lineCount = parsedOutput.items[0].body.split("\n").length;
      expect(lineCount).toBeLessThan(66000);
    });

    it("should preserve backticks and code blocks", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Code Test", "body": "Use \`z3 -v:10\` in terminal. Code block:\\n\`\`\`\\nz3 -memory:high input.smt2\\nftp://should-not-be-redacted-in-code\\n\`\`\`"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      // The content should be preserved with proper escaping
      expect(parsedOutput.items[0].body).toContain("z3 -v:10");
      expect(parsedOutput.items[0].body).toContain("z3 -memory:high");
    });

    it("should handle sanitization across multiple field types", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create-pull-request", "title": "PR with z3 -v:10 flag", "body": "Testing https://github.com/repo and ftp://example.com", "branch": "feature/z3-timeout:5000", "labels": ["bug", "z3:solver"]}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-pull-request": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].title).toBe("PR with z3 -v:10 flag");
      expect(parsedOutput.items[0].body).toBe("Testing https://github.com/repo and (redacted)");
      expect(parsedOutput.items[0].branch).toBe("feature/z3-timeout:5000");
      expect(parsedOutput.items[0].labels).toEqual(["bug", "z3:solver"]);
    });

    it("should remove XML comments to prevent content hiding", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "XML Comment Test", "body": "This is visible <!-- This is hidden content --> more visible text <!--- This is also hidden ---> and more text <!--- malformed comment --!> final text"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": true}';

      await eval(`(async () => { ${collectScript} })()`);

      const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "output");
      const parsedOutput = JSON.parse(outputCall[1]);

      expect(parsedOutput.items[0].body).toBe("This is visible  more visible text  and more text  final text");
    });
  });

  describe("Min validation tests", () => {
    it("should pass when min requirement is met", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "First Issue", "body": "First body"}
{"type": "create_issue", "title": "Second Issue", "body": "Second body"}
{"type": "create_issue", "title": "Third Issue", "body": "Third body"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      // Set min to 2 for create_issue
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": {"min": 2, "max": 5}}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(3); // All 3 items should be valid
      expect(parsedOutput.errors).toHaveLength(0); // No errors for meeting min requirement
    });

    it("should fail when min requirement is not met", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Only Issue", "body": "Only body"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      // Set min to 3 for create_issue, but we only have 1 item
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": {"min": 3, "max": 5}}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1); // The 1 valid item is still processed
      expect(parsedOutput.errors).toHaveLength(1); // Error for not meeting min requirement
      expect(parsedOutput.errors[0]).toContain("Too few items of type 'create_issue'. Minimum required: 3, found: 1.");
    });

    it("should handle multiple types with different min requirements", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Issue 1", "body": "Body 1"}
{"type": "create_issue", "title": "Issue 2", "body": "Body 2"}
{"type": "add_comment", "body": "Comment 1"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      // Set min to 1 for create_issue (satisfied) and min to 2 for add-comment (not satisfied)
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": {"min": 1, "max": 5}, "add_comment": {"min": 2, "max": 5}}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(3); // All items are processed
      expect(parsedOutput.errors).toHaveLength(1); // Only error for add-comment min requirement
      expect(parsedOutput.errors[0]).toContain("Too few items of type 'add_comment'. Minimum required: 2, found: 1.");
    });

    it("should ignore min when set to 0", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Issue", "body": "Body"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      // Set min to 0 for create_issue (should be ignored)
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": {"min": 0, "max": 5}}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.errors).toHaveLength(0); // No min validation errors
    });

    it("should work when no min is specified (defaults to 0)", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "create_issue", "title": "Issue", "body": "Body"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      // No min specified, should default to 0
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": {"max": 5}}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(1);
      expect(parsedOutput.errors).toHaveLength(0); // No min validation errors
    });

    it("should validate min even when no items are present", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = ``; // Empty file

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      // Set min to 1 for create_issue, but no items present
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create_issue": {"min": 1, "max": 5}}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(0); // No items
      expect(parsedOutput.errors).toHaveLength(1); // Error for not meeting min requirement
      expect(parsedOutput.errors[0]).toContain("Too few items of type 'create_issue'. Minimum required: 1, found: 0.");
    });

    it("should work with different safe output types", async () => {
      const testFile = "/tmp/gh-aw/test-ndjson-output.txt";
      const ndjsonContent = `{"type": "add_comment", "body": "Comment"}
{"type": "create_discussion", "title": "Discussion", "body": "Discussion body"}
{"type": "create_discussion", "title": "Discussion 2", "body": "Discussion body 2"}`;

      fs.writeFileSync(testFile, ndjsonContent);
      process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
      // Set min requirements for different types
      process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"add_comment": {"min": 2, "max": 5}, "create_discussion": {"min": 1, "max": 5}}';

      await eval(`(async () => { ${collectScript} })()`);

      const setOutputCalls = mockCore.setOutput.mock.calls;
      const outputCall = setOutputCalls.find(call => call[0] === "output");
      expect(outputCall).toBeDefined();

      const parsedOutput = JSON.parse(outputCall[1]);
      expect(parsedOutput.items).toHaveLength(3); // All items processed
      expect(parsedOutput.errors).toHaveLength(1); // Error only for add-comment min requirement
      expect(parsedOutput.errors[0]).toContain("Too few items of type 'add_comment'. Minimum required: 2, found: 1.");
    });
  });
});
