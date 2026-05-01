// @ts-check

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { createPythonHandler } from "./mcp_handler_python.cjs";
import fs from "fs";
import path from "path";
import os from "os";

describe("createPythonHandler", () => {
  let mockServer;
  let tempDir;
  let testScriptPath;

  beforeEach(() => {
    // Create mock server with debug logging
    mockServer = {
      debug: () => {},
      debugError: () => {},
    };

    // Create temporary directory for test scripts
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "python-handler-test-"));
  });

  afterEach(() => {
    // Clean up temporary directory
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("should execute a simple Python script", async () => {
    testScriptPath = path.join(tempDir, "test.py");
    const pyCode = `
import json, sys

inputs = json.load(sys.stdin)
result = {"message": "Hello from Python", "input": inputs}
print(json.dumps(result))
`;
    fs.writeFileSync(testScriptPath, pyCode);

    const handler = createPythonHandler(mockServer, "test-tool", testScriptPath, 60);
    const result = await handler({ name: "World", count: 42 });

    expect(result).toBeDefined();
    expect(result.content).toBeDefined();
    expect(result.content.length).toBe(1);
    expect(result.content[0].type).toBe("text");

    const output = JSON.parse(result.content[0].text);
    expect(output.message).toBe("Hello from Python");
    expect(output.input).toEqual({ name: "World", count: 42 });
  }, 15000);

  it("should handle Python script with no input", async () => {
    testScriptPath = path.join(tempDir, "no-input.py");
    const pyCode = `
import json, sys

# Read stdin but ignore it
sys.stdin.read()
result = {"status": "ok"}
print(json.dumps(result))
`;
    fs.writeFileSync(testScriptPath, pyCode);

    const handler = createPythonHandler(mockServer, "no-input-tool", testScriptPath);
    const result = await handler({});

    const output = JSON.parse(result.content[0].text);
    expect(output.status).toBe("ok");
  }, 15000);

  it("should handle non-JSON output", async () => {
    testScriptPath = path.join(tempDir, "text-output.py");
    const pyCode = `print("Plain text output")`;
    fs.writeFileSync(testScriptPath, pyCode);

    const handler = createPythonHandler(mockServer, "text-tool", testScriptPath);
    const result = await handler({});

    const output = JSON.parse(result.content[0].text);
    expect(output.stdout).toContain("Plain text output");
  }, 15000);

  it("should respect timeout setting", async () => {
    testScriptPath = path.join(tempDir, "slow.py");
    const pyCode = `
import time
time.sleep(30)
`;
    fs.writeFileSync(testScriptPath, pyCode);

    const handler = createPythonHandler(mockServer, "slow-tool", testScriptPath, 1);

    await expect(handler({})).rejects.toThrow();
  }, 15000);

  it("should handle Python script errors with enhanced error message", async () => {
    testScriptPath = path.join(tempDir, "error.py");
    const pyCode = `
import sys
sys.stderr.write("Python error details\\n")
sys.exit(1)
`;
    fs.writeFileSync(testScriptPath, pyCode);

    const handler = createPythonHandler(mockServer, "error-tool", testScriptPath);

    let caughtError;
    try {
      await handler({});
    } catch (e) {
      caughtError = e;
    }
    expect(caughtError).toBeDefined();
    expect(caughtError.message).toContain("Python error details");
  }, 15000);

  it("should pass complex input data", async () => {
    testScriptPath = path.join(tempDir, "complex.py");
    const pyCode = `
import json, sys

inputs = json.load(sys.stdin)
# Echo back the input
print(json.dumps(inputs))
`;
    fs.writeFileSync(testScriptPath, pyCode);

    const complexInput = {
      name: "test",
      numbers: [1, 2, 3],
      nested: {
        key: "value",
      },
    };

    const handler = createPythonHandler(mockServer, "complex-tool", testScriptPath);
    const result = await handler(complexInput);

    const output = JSON.parse(result.content[0].text);
    expect(output).toEqual(complexInput);
  }, 15000);

  it("should execute script from GITHUB_WORKSPACE directory", async () => {
    const originalWorkspace = process.env.GITHUB_WORKSPACE;
    process.env.GITHUB_WORKSPACE = tempDir;

    try {
      testScriptPath = path.join(tempDir, "test-cwd.py");
      const pyCode = `
import json, os, sys
sys.stdin.read()
result = {"cwd": os.getcwd()}
print(json.dumps(result))
`;
      fs.writeFileSync(testScriptPath, pyCode);

      const handler = createPythonHandler(mockServer, "cwd-tool", testScriptPath);
      const result = await handler({});

      const output = JSON.parse(result.content[0].text);
      expect(output.cwd).toBe(tempDir);
    } finally {
      if (originalWorkspace === undefined) {
        delete process.env.GITHUB_WORKSPACE;
      } else {
        process.env.GITHUB_WORKSPACE = originalWorkspace;
      }
    }
  }, 15000);
});
