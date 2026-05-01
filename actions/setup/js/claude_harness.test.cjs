import { afterEach, describe, it, expect, vi } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const { ERROR_MAX_TURNS_PATTERN, isMaxTurnsError, parseArgs } = require("./claude_harness.cjs");

describe("claude_harness.cjs", () => {
  // Test the core logic patterns used by the driver without importing the module
  // (importing the module would invoke main() which calls process.exit).

  describe("error_max_turns detection pattern", () => {
    it("matches the exact error from the failed workflow run", () => {
      const output = '{"type":"result","subtype":"error_max_turns","is_error":true,"num_turns":13,' + '"terminal_reason":"max_turns","errors":["Reached maximum number of turns (12)"]}';
      expect(ERROR_MAX_TURNS_PATTERN.test(output)).toBe(true);
    });

    it("matches with whitespace around the colon", () => {
      expect(ERROR_MAX_TURNS_PATTERN.test('"subtype" : "error_max_turns"')).toBe(true);
      expect(ERROR_MAX_TURNS_PATTERN.test('"subtype":"error_max_turns"')).toBe(true);
    });

    it("matches when embedded in larger JSONL output", () => {
      const output = '{"type":"assistant","message":{"id":"msg_123"}}\n' + '{"type":"result","subtype":"error_max_turns","is_error":true,"num_turns":50}\n';
      expect(ERROR_MAX_TURNS_PATTERN.test(output)).toBe(true);
    });

    it("does not match other result subtypes", () => {
      expect(ERROR_MAX_TURNS_PATTERN.test('"subtype":"success"')).toBe(false);
      expect(ERROR_MAX_TURNS_PATTERN.test('"subtype":"error"')).toBe(false);
      expect(ERROR_MAX_TURNS_PATTERN.test('"subtype":"error_max_turns_exceeded"')).toBe(false);
    });

    it("does not match unrelated output", () => {
      expect(ERROR_MAX_TURNS_PATTERN.test("Error: API overloaded")).toBe(false);
      expect(ERROR_MAX_TURNS_PATTERN.test("max_turns reached")).toBe(false);
      expect(ERROR_MAX_TURNS_PATTERN.test("")).toBe(false);
    });
  });

  describe("isMaxTurnsError", () => {
    it("returns true for error_max_turns in output", () => {
      const output = '{"type":"result","subtype":"error_max_turns","is_error":true,"num_turns":13}';
      expect(isMaxTurnsError(output)).toBe(true);
    });

    it("returns false for non-max-turns output", () => {
      expect(isMaxTurnsError("Error: API overloaded, please retry")).toBe(false);
      expect(isMaxTurnsError("CAPIError: 400 Bad Request")).toBe(false);
      expect(isMaxTurnsError("")).toBe(false);
    });

    it("returns false for similar but non-matching subtype", () => {
      expect(isMaxTurnsError('"subtype":"error"')).toBe(false);
      expect(isMaxTurnsError('"type":"error_max_turns"')).toBe(false);
    });
  });

  describe("retry policy: no retry on error_max_turns", () => {
    // Inline the same retry-eligibility logic as the driver for unit testing.
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @param {boolean} useContinueOnRetry
     * @returns {{ shouldRetry: boolean, useContinueOnRetry: boolean }}
     */
    function applyRetryPolicy(result, attempt, useContinueOnRetry = false) {
      if (result.exitCode === 0) return { shouldRetry: false, useContinueOnRetry };
      if (isMaxTurnsError(result.output)) return { shouldRetry: false, useContinueOnRetry };
      if (attempt < MAX_RETRIES && result.hasOutput) {
        return { shouldRetry: true, useContinueOnRetry: true };
      }
      return { shouldRetry: false, useContinueOnRetry };
    }

    it("does not retry on error_max_turns even when output was produced", () => {
      const output = '{"type":"result","subtype":"error_max_turns","is_error":true,"num_turns":13,' + '"terminal_reason":"max_turns","errors":["Reached maximum number of turns (12)"]}';
      const result = { exitCode: 1, hasOutput: true, output };
      const { shouldRetry } = applyRetryPolicy(result, 0);
      expect(shouldRetry).toBe(false);
    });

    it("does not retry on error_max_turns regardless of attempt number", () => {
      const output = '{"type":"result","subtype":"error_max_turns","is_error":true,"num_turns":50}';
      const result = { exitCode: 1, hasOutput: true, output };
      expect(applyRetryPolicy(result, 0).shouldRetry).toBe(false);
      expect(applyRetryPolicy(result, 1).shouldRetry).toBe(false);
      expect(applyRetryPolicy(result, 2).shouldRetry).toBe(false);
    });

    it("retries on partial execution (non-max-turns failure with output)", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Error: API temporarily unavailable" };
      const { shouldRetry, useContinueOnRetry: newContinue } = applyRetryPolicy(result, 0);
      expect(shouldRetry).toBe(true);
      expect(newContinue).toBe(true);
    });

    it("does not retry when no output was produced", () => {
      const result = { exitCode: 1, hasOutput: false, output: "" };
      expect(applyRetryPolicy(result, 0).shouldRetry).toBe(false);
    });

    it("does not retry after retry budget exhausted", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Error: transient failure" };
      expect(applyRetryPolicy(result, MAX_RETRIES).shouldRetry).toBe(false);
    });

    it("does not retry on success", () => {
      const result = { exitCode: 0, hasOutput: true, output: "Done." };
      expect(applyRetryPolicy(result, 0).shouldRetry).toBe(false);
    });

    it("distinguishes error_max_turns from other JSON result subtypes", () => {
      const successResult = {
        exitCode: 1,
        hasOutput: true,
        output: '{"type":"result","subtype":"success","is_error":false}',
      };
      // Non-max-turns error result should be retried
      expect(applyRetryPolicy(successResult, 0).shouldRetry).toBe(true);
    });
  });

  describe("parseArgs: --prompt-file resolution", () => {
    let tempDir = "";

    afterEach(() => {
      if (tempDir) {
        try {
          fs.rmSync(tempDir, { recursive: true });
        } catch {
          // ignore cleanup errors
        }
        tempDir = "";
      }
    });

    it("reads prompt file and appends content as positional arg in initialArgs", () => {
      tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "claude-harness-test-"));
      const promptFile = path.join(tempDir, "prompt.txt");
      fs.writeFileSync(promptFile, "Do the task please");

      const { initialArgs, continueArgs } = parseArgs(["--print", "--no-chrome", "--prompt-file", promptFile]);

      expect(initialArgs).toEqual(["--print", "--no-chrome", "Do the task please"]);
      expect(continueArgs).toEqual(["--print", "--no-chrome", "--continue"]);
    });

    it("places prompt content as the last positional arg", () => {
      tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "claude-harness-test-"));
      const promptFile = path.join(tempDir, "prompt.txt");
      fs.writeFileSync(promptFile, "Fix the bug");

      const { initialArgs } = parseArgs(["--print", "--allowed-tools", "Bash", "--prompt-file", promptFile]);

      // Last element should be the prompt content
      expect(initialArgs[initialArgs.length - 1]).toBe("Fix the bug");
      expect(initialArgs).not.toContain("--prompt-file");
      expect(initialArgs).not.toContain(promptFile);
    });

    it("omits --prompt-file from continueArgs and appends --continue instead", () => {
      tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "claude-harness-test-"));
      const promptFile = path.join(tempDir, "prompt.txt");
      fs.writeFileSync(promptFile, "Fix the bug");

      const { continueArgs } = parseArgs(["--print", "--prompt-file", promptFile]);

      expect(continueArgs).toContain("--continue");
      expect(continueArgs).not.toContain("--prompt-file");
      expect(continueArgs).not.toContain(promptFile);
      expect(continueArgs).not.toContain("Fix the bug");
    });

    it("passes non-prompt args through unchanged", () => {
      tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "claude-harness-test-"));
      const promptFile = path.join(tempDir, "prompt.txt");
      fs.writeFileSync(promptFile, "Hello");

      const { initialArgs, continueArgs } = parseArgs(["--print", "--no-chrome", "--verbose", "--prompt-file", promptFile]);

      expect(initialArgs).toContain("--print");
      expect(initialArgs).toContain("--no-chrome");
      expect(initialArgs).toContain("--verbose");
      expect(continueArgs).toContain("--print");
      expect(continueArgs).toContain("--no-chrome");
      expect(continueArgs).toContain("--verbose");
    });

    it("handles args without --prompt-file unchanged", () => {
      const { initialArgs, continueArgs } = parseArgs(["--print", "--no-chrome"]);

      expect(initialArgs).toEqual(["--print", "--no-chrome"]);
      expect(continueArgs).toEqual(["--print", "--no-chrome", "--continue"]);
    });

    it("leaves args unchanged when --prompt-file has no following path", () => {
      const { initialArgs } = parseArgs(["--print", "--prompt-file"]);

      // Warning is logged but args are passed through
      expect(initialArgs).toContain("--prompt-file");
    });

    it("leaves args unchanged when prompt file cannot be read", () => {
      const { initialArgs } = parseArgs(["--print", "--prompt-file", "/nonexistent/path/prompt.txt"]);

      // Warning is logged; original --prompt-file args preserved
      expect(initialArgs).toContain("--prompt-file");
      expect(initialArgs).toContain("/nonexistent/path/prompt.txt");
    });
  });
});
