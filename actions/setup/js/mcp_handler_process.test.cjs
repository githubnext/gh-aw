// @ts-check

import { describe, it, expect } from "vitest";
import { buildEnhancedError, parseStdoutAsJson, wrapMCPContent, executeProcess } from "./mcp_handler_process.cjs";

describe("buildEnhancedError", () => {
  it("should include script path and exit code for numeric code", () => {
    const error = Object.assign(new Error("Command failed"), { code: 2 });
    const result = buildEnhancedError(error, "/path/to/script.sh", "", "");
    expect(result.message).toContain("/path/to/script.sh");
    expect(result.message).toContain("exit code: 2");
  });

  it("should include signal when process was killed by a signal", () => {
    const error = Object.assign(new Error("Command failed"), { code: null, signal: "SIGTERM" });
    const result = buildEnhancedError(error, "script.sh", "", "");
    expect(result.message).toContain("signal: SIGTERM");
    expect(result.message).not.toContain("exit code:");
  });

  it("should preserve original error message for OS-level failures (e.g. ENOENT)", () => {
    const error = Object.assign(new Error("spawn python3 ENOENT"), { code: "ENOENT" });
    const result = buildEnhancedError(error, "script.py", "", "");
    expect(result.message).toContain("spawn python3 ENOENT");
    expect(result.message).not.toContain("exit code:");
    expect(result.message).not.toContain("signal:");
  });

  it("should include stderr when present", () => {
    const error = Object.assign(new Error("Command failed"), { code: 1 });
    const result = buildEnhancedError(error, "script.py", "", "error output");
    expect(result.message).toContain("stderr:");
    expect(result.message).toContain("error output");
  });

  it("should include stdout when present", () => {
    const error = Object.assign(new Error("Command failed"), { code: 1 });
    const result = buildEnhancedError(error, "script.py", "some output", "");
    expect(result.message).toContain("stdout:");
    expect(result.message).toContain("some output");
  });

  it("should omit empty stdout and stderr", () => {
    const error = Object.assign(new Error("Command failed"), { code: 1 });
    const result = buildEnhancedError(error, "script.sh", "", "");
    expect(result.message).not.toContain("stdout:");
    expect(result.message).not.toContain("stderr:");
  });

  it("should omit whitespace-only stdout and stderr", () => {
    const error = Object.assign(new Error("Command failed"), { code: 1 });
    const result = buildEnhancedError(error, "script.sh", "   ", "  \n  ");
    expect(result.message).not.toContain("stdout:");
    expect(result.message).not.toContain("stderr:");
  });

  it("should include both stdout and stderr when both are present", () => {
    const error = Object.assign(new Error("Command failed"), { code: 1 });
    const result = buildEnhancedError(error, "script.sh", "out data", "err data");
    expect(result.message).toContain("stderr:\nerr data");
    expect(result.message).toContain("stdout:\nout data");
  });
});

describe("parseStdoutAsJson", () => {
  it("should parse valid JSON stdout", () => {
    const result = parseStdoutAsJson('{"status":"ok"}', "");
    expect(result).toEqual({ status: "ok" });
  });

  it("should trim whitespace before parsing JSON", () => {
    const result = parseStdoutAsJson('  {"key":"value"}\n', "");
    expect(result).toEqual({ key: "value" });
  });

  it("should return stdout/stderr object when stdout is empty", () => {
    const result = parseStdoutAsJson("", "some stderr");
    expect(result).toEqual({ stdout: "", stderr: "some stderr" });
  });

  it("should return stdout/stderr object when stdout is whitespace only", () => {
    const result = parseStdoutAsJson("   ", "");
    expect(result).toEqual({ stdout: "   ", stderr: "" });
  });

  it("should return stdout/stderr object when JSON parsing fails", () => {
    const result = parseStdoutAsJson("not json", "");
    expect(result).toEqual({ stdout: "not json", stderr: "" });
  });

  it("should call onParseFailure when JSON parsing fails", () => {
    let called = false;
    parseStdoutAsJson("not json", "", () => {
      called = true;
    });
    expect(called).toBe(true);
  });

  it("should not call onParseFailure for valid JSON", () => {
    let called = false;
    parseStdoutAsJson('{"ok":true}', "", () => {
      called = true;
    });
    expect(called).toBe(false);
  });

  it("should not call onParseFailure for empty stdout", () => {
    let called = false;
    parseStdoutAsJson("", "", () => {
      called = true;
    });
    expect(called).toBe(false);
  });
});

describe("wrapMCPContent", () => {
  it("should wrap result in MCP content format", () => {
    const result = wrapMCPContent({ status: "ok" });
    expect(result).toEqual({
      content: [{ type: "text", text: '{"status":"ok"}' }],
    });
  });

  it("should JSON-stringify the result", () => {
    const result = wrapMCPContent({ count: 42, items: ["a", "b"] });
    expect(result.content[0].type).toBe("text");
    expect(JSON.parse(result.content[0].text)).toEqual({ count: 42, items: ["a", "b"] });
  });
});

describe("executeProcess", () => {
  const mockServer = {
    debug: () => {},
    debugError: () => {},
  };

  it("should execute a process and return MCP content", async () => {
    const result = await executeProcess({
      server: mockServer,
      toolName: "test-tool",
      languageLabel: "Test",
      command: process.execPath,
      args: ["-e", "process.stdout.write(JSON.stringify({status:'ok'}))"],
      env: process.env,
      inputJson: null,
      timeoutSeconds: 10,
      scriptPath: "test.js",
    });

    expect(result.content).toBeDefined();
    expect(result.content[0].type).toBe("text");
    expect(JSON.parse(result.content[0].text)).toEqual({ status: "ok" });
  });

  it("should write inputJson to stdin", async () => {
    const script = `process.stdin.on('data', d => process.stdout.write(d));`;
    const result = await executeProcess({
      server: mockServer,
      toolName: "stdin-tool",
      languageLabel: "Test",
      command: process.execPath,
      args: ["-e", script],
      env: process.env,
      inputJson: JSON.stringify({ hello: "world" }),
      timeoutSeconds: 10,
      scriptPath: "test.js",
    });

    expect(JSON.parse(result.content[0].text)).toEqual({ hello: "world" });
  });

  it("should reject with enhanced error on process failure", async () => {
    await expect(
      executeProcess({
        server: mockServer,
        toolName: "error-tool",
        languageLabel: "Test",
        command: process.execPath,
        args: ["-e", "process.stderr.write('oops'); process.exit(1)"],
        env: process.env,
        inputJson: null,
        timeoutSeconds: 10,
        scriptPath: "failing.js",
      })
    ).rejects.toThrow(/failing\.js/);
  });

  it("should include stderr in enhanced error", async () => {
    let caughtError;
    try {
      await executeProcess({
        server: mockServer,
        toolName: "err-tool",
        languageLabel: "Test",
        command: process.execPath,
        args: ["-e", "process.stderr.write('detailed error'); process.exit(2)"],
        env: process.env,
        inputJson: null,
        timeoutSeconds: 10,
        scriptPath: "script.js",
      });
    } catch (e) {
      caughtError = e;
    }
    expect(caughtError).toBeDefined();
    expect(caughtError.message).toContain("detailed error");
  });

  it("should call onError hook before rejecting", async () => {
    let onErrorCalled = false;
    await expect(
      executeProcess({
        server: mockServer,
        toolName: "hook-tool",
        languageLabel: "Test",
        command: process.execPath,
        args: ["-e", "process.exit(1)"],
        env: process.env,
        inputJson: null,
        timeoutSeconds: 10,
        scriptPath: "script.js",
        onError: () => {
          onErrorCalled = true;
        },
      })
    ).rejects.toThrow();
    expect(onErrorCalled).toBe(true);
  });

  it("should use buildResult when provided", async () => {
    const result = await executeProcess({
      server: mockServer,
      toolName: "custom-result-tool",
      languageLabel: "Test",
      command: process.execPath,
      args: ["-e", "process.stdout.write('raw output')"],
      env: process.env,
      inputJson: null,
      timeoutSeconds: 10,
      scriptPath: "script.js",
      buildResult: (stdout, stderr) => ({ custom: true, out: stdout, err: stderr }),
    });

    expect(JSON.parse(result.content[0].text)).toEqual({
      custom: true,
      out: "raw output",
      err: "",
    });
  });

  it("should fall back to stdout/stderr object for non-JSON output", async () => {
    const result = await executeProcess({
      server: mockServer,
      toolName: "text-tool",
      languageLabel: "Test",
      command: process.execPath,
      args: ["-e", "process.stdout.write('plain text')"],
      env: process.env,
      inputJson: null,
      timeoutSeconds: 10,
      scriptPath: "script.js",
    });

    const parsed = JSON.parse(result.content[0].text);
    expect(parsed.stdout).toBe("plain text");
  });

  it("should respect the timeout setting", async () => {
    await expect(
      executeProcess({
        server: mockServer,
        toolName: "slow-tool",
        languageLabel: "Test",
        command: process.execPath,
        args: ["-e", "setTimeout(() => {}, 30000)"],
        env: process.env,
        inputJson: null,
        timeoutSeconds: 1,
        scriptPath: "slow.js",
      })
    ).rejects.toThrow();
  }, 15000);

  it("should still reject with the original error when onError itself throws", async () => {
    let rejectedError;
    try {
      await executeProcess({
        server: mockServer,
        toolName: "throwing-hook-tool",
        languageLabel: "Test",
        command: process.execPath,
        args: ["-e", "process.stderr.write('original'); process.exit(1)"],
        env: process.env,
        inputJson: null,
        timeoutSeconds: 10,
        scriptPath: "script.js",
        onError: () => {
          throw new Error("cleanup exploded");
        },
      });
    } catch (e) {
      rejectedError = e;
    }
    // The promise must still reject with the original execution error, not the cleanup error
    expect(rejectedError).toBeDefined();
    expect(rejectedError.message).toContain("original");
    expect(rejectedError.message).not.toContain("cleanup exploded");
  });

  it("should fall back to { stdout, stderr } when buildResult throws", async () => {
    const result = await executeProcess({
      server: mockServer,
      toolName: "broken-builder-tool",
      languageLabel: "Test",
      command: process.execPath,
      args: ["-e", "process.stdout.write('some output')"],
      env: process.env,
      inputJson: null,
      timeoutSeconds: 10,
      scriptPath: "script.js",
      buildResult: () => {
        throw new Error("builder blew up");
      },
    });
    const parsed = JSON.parse(result.content[0].text);
    expect(parsed.stdout).toBe("some output");
  });

  it("should use GITHUB_WORKSPACE as cwd when set", async () => {
    const originalWorkspace = process.env.GITHUB_WORKSPACE;
    process.env.GITHUB_WORKSPACE = process.cwd();

    try {
      const result = await executeProcess({
        server: mockServer,
        toolName: "cwd-tool",
        languageLabel: "Test",
        command: process.execPath,
        args: ["-e", "process.stdout.write(JSON.stringify({cwd:process.cwd()}))"],
        env: process.env,
        inputJson: null,
        timeoutSeconds: 10,
        scriptPath: "script.js",
      });

      const parsed = JSON.parse(result.content[0].text);
      expect(parsed.cwd).toBe(process.cwd());
    } finally {
      if (originalWorkspace === undefined) {
        delete process.env.GITHUB_WORKSPACE;
      } else {
        process.env.GITHUB_WORKSPACE = originalWorkspace;
      }
    }
  });
});
