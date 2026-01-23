// @ts-check

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { createJavaScriptHandler } from "./mcp_handler_javascript.cjs";
import fs from "fs";
import path from "path";
import os from "os";

describe("createJavaScriptHandler", () => {
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
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "js-handler-test-"));
  });

  afterEach(() => {
    // Clean up temporary directory
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("should execute a simple JavaScript script", async () => {
    // Create a simple JavaScript script that echoes input
    testScriptPath = path.join(tempDir, "test.cjs");
    const jsCode = `
const fs = require('fs');

// Read JSON from stdin
let input = '';
process.stdin.on('data', chunk => {
  input += chunk;
});

process.stdin.on('end', () => {
  const inputs = JSON.parse(input);
  const result = {
    message: "Hello from JavaScript",
    input: inputs
  };
  console.log(JSON.stringify(result));
});
`;
    fs.writeFileSync(testScriptPath, jsCode);

    const handler = createJavaScriptHandler(mockServer, "test-tool", testScriptPath, 60);
    const result = await handler({ name: "World", count: 42 });

    expect(result).toBeDefined();
    expect(result.content).toBeDefined();
    expect(result.content.length).toBe(1);
    expect(result.content[0].type).toBe("text");

    const output = JSON.parse(result.content[0].text);
    expect(output.message).toBe("Hello from JavaScript");
    expect(output.input).toEqual({ name: "World", count: 42 });
  });

  it("should handle JavaScript script with no input", async () => {
    testScriptPath = path.join(tempDir, "no-input.cjs");
    const jsCode = `
const result = { status: "ok" };
console.log(JSON.stringify(result));
`;
    fs.writeFileSync(testScriptPath, jsCode);

    const handler = createJavaScriptHandler(mockServer, "no-input-tool", testScriptPath);
    const result = await handler({});

    const output = JSON.parse(result.content[0].text);
    expect(output.status).toBe("ok");
  });

  it("should handle non-JSON output", async () => {
    testScriptPath = path.join(tempDir, "text-output.cjs");
    const jsCode = `console.log("Plain text output");`;
    fs.writeFileSync(testScriptPath, jsCode);

    const handler = createJavaScriptHandler(mockServer, "text-tool", testScriptPath);
    const result = await handler({});

    const output = JSON.parse(result.content[0].text);
    expect(output.stdout).toContain("Plain text output");
  });

  it("should respect timeout setting", async () => {
    testScriptPath = path.join(tempDir, "slow.cjs");
    const jsCode = `
setTimeout(() => {
  console.log("Done");
}, 10000);
`;
    fs.writeFileSync(testScriptPath, jsCode);

    const handler = createJavaScriptHandler(mockServer, "slow-tool", testScriptPath, 1);

    await expect(handler({})).rejects.toThrow();
  }, 15000); // Increase test timeout to allow for process timeout

  it("should handle JavaScript script errors", async () => {
    testScriptPath = path.join(tempDir, "error.cjs");
    const jsCode = `
process.exit(1);
`;
    fs.writeFileSync(testScriptPath, jsCode);

    const handler = createJavaScriptHandler(mockServer, "error-tool", testScriptPath);

    await expect(handler({})).rejects.toThrow();
  });

  it("should pass complex input data", async () => {
    testScriptPath = path.join(tempDir, "complex.cjs");
    const jsCode = `
let input = '';
process.stdin.on('data', chunk => {
  input += chunk;
});

process.stdin.on('end', () => {
  const inputs = JSON.parse(input);
  // Echo back the input
  console.log(JSON.stringify(inputs));
});
`;
    fs.writeFileSync(testScriptPath, jsCode);

    const complexInput = {
      name: "test",
      numbers: [1, 2, 3],
      nested: {
        key: "value",
      },
    };

    const handler = createJavaScriptHandler(mockServer, "complex-tool", testScriptPath);
    const result = await handler(complexInput);

    const output = JSON.parse(result.content[0].text);
    expect(output).toEqual(complexInput);
  });

  it("should execute script from GITHUB_WORKSPACE directory", async () => {
    // Save original GITHUB_WORKSPACE
    const originalWorkspace = process.env.GITHUB_WORKSPACE;

    // Set GITHUB_WORKSPACE to tempDir
    process.env.GITHUB_WORKSPACE = tempDir;

    try {
      // Create a JavaScript script that outputs current working directory
      testScriptPath = path.join(tempDir, "test-cwd.cjs");
      const jsCode = `
const result = { cwd: process.cwd() };
console.log(JSON.stringify(result));
`;
      fs.writeFileSync(testScriptPath, jsCode);

      const handler = createJavaScriptHandler(mockServer, "cwd-tool", testScriptPath);
      const result = await handler({});

      const output = JSON.parse(result.content[0].text);
      expect(output.cwd).toBe(tempDir);
    } finally {
      // Restore original GITHUB_WORKSPACE
      if (originalWorkspace === undefined) {
        delete process.env.GITHUB_WORKSPACE;
      } else {
        process.env.GITHUB_WORKSPACE = originalWorkspace;
      }
    }
  });

  it("should use process.cwd() when GITHUB_WORKSPACE is not set", async () => {
    // Save original GITHUB_WORKSPACE
    const originalWorkspace = process.env.GITHUB_WORKSPACE;

    // Unset GITHUB_WORKSPACE
    delete process.env.GITHUB_WORKSPACE;

    try {
      // Create a JavaScript script that outputs current working directory
      testScriptPath = path.join(tempDir, "test-default-cwd.cjs");
      const jsCode = `
const result = { cwd: process.cwd() };
console.log(JSON.stringify(result));
`;
      fs.writeFileSync(testScriptPath, jsCode);

      const handler = createJavaScriptHandler(mockServer, "default-cwd-tool", testScriptPath);
      const result = await handler({});

      const output = JSON.parse(result.content[0].text);
      // When GITHUB_WORKSPACE is not set, should use process.cwd()
      expect(output.cwd).toBe(process.cwd());
    } finally {
      // Restore original GITHUB_WORKSPACE
      if (originalWorkspace === undefined) {
        delete process.env.GITHUB_WORKSPACE;
      } else {
        process.env.GITHUB_WORKSPACE = originalWorkspace;
      }
    }
  });
});
