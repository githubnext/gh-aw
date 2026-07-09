import { describe, it, expect } from "vitest";
import { spawnSync } from "child_process";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const {
  resolveClaudePromptFileArgs,
  stripPromptFileArgs,
  isRateLimitError,
  isAuthenticationFailedError,
  isMaxTurnsExit,
  isNoDeferredMarkerError,
  isInvalidModelError,
  isSignalTerminationExitCode,
  shouldRetryWithContinue,
  countPermissionDeniedIssues,
  hasNumerousPermissionDeniedIssues,
  extractDeniedCommands,
  buildMissingToolPermissionIssuePayload,
  resolveRetryConfig,
  resolveStartupRetryLimit,
} = require("./claude_harness.cjs");

const agentTempDir = "/tmp/gh-aw/agent";

function makeHarnessTempDir(name) {
  fs.mkdirSync(agentTempDir, { recursive: true });
  return fs.mkdtempSync(path.join(agentTempDir, name));
}

function runHarnessWithStub({ stubScript, prompt = "fix the bug", extraArgs = [], extraEnv = {} }) {
  const tempDir = makeHarnessTempDir("claude-harness-");
  const stubPath = path.join(tempDir, "stub.cjs");
  const promptPath = path.join(tempDir, "prompt.txt");
  const callsPath = path.join(tempDir, "calls.jsonl");
  fs.writeFileSync(stubPath, stubScript, "utf8");
  fs.writeFileSync(promptPath, prompt, "utf8");

  const result = spawnSync(process.execPath, ["claude_harness.cjs", process.execPath, stubPath, "--print", ...extraArgs, "--prompt-file", promptPath], {
    cwd: path.dirname(require.resolve("./claude_harness.cjs")),
    env: { ...process.env, ...extraEnv, CLAUDE_HARNESS_STUB_CALLS: callsPath },
    encoding: "utf8",
    timeout: 45000,
  });
  const calls = fs
    .readFileSync(callsPath, "utf8")
    .trim()
    .split("\n")
    .filter(Boolean)
    .map(line => JSON.parse(line));
  return { result, calls };
}

describe("claude_harness.cjs", () => {
  describe("resolveClaudePromptFileArgs", () => {
    it("replaces --prompt-file with ['--', content] as the last two positional args", () => {
      const promptFile = path.join(os.tmpdir(), `claude-harness-prompt-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "fix the bug", "utf8");
      try {
        const result = resolveClaudePromptFileArgs(["--print", "--prompt-file", promptFile, "--output-format", "stream-json"]);
        expect(result).toEqual(["--print", "--output-format", "stream-json", "--", "fix the bug"]);
      } finally {
        fs.rmSync(promptFile);
      }
    });

    it("appends -- and prompt content as the last two args", () => {
      const promptFile = path.join(os.tmpdir(), `claude-harness-prompt-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "my task", "utf8");
      try {
        const result = resolveClaudePromptFileArgs(["--prompt-file", promptFile]);
        expect(result).toEqual(["--", "my task"]);
      } finally {
        fs.rmSync(promptFile);
      }
    });

    it("passes through args that have no --prompt-file", () => {
      const result = resolveClaudePromptFileArgs(["--print", "--output-format", "json"]);
      expect(result).toEqual(["--print", "--output-format", "json"]);
    });

    it("preserves args when --prompt-file is provided without a path", () => {
      const result = resolveClaudePromptFileArgs(["--print", "--prompt-file"]);
      // When no path follows --prompt-file, it is preserved as-is
      expect(result).toEqual(["--print", "--prompt-file"]);
    });

    it("throws when the prompt file does not exist", () => {
      const missingFile = path.join(os.tmpdir(), `claude-harness-missing-${Date.now()}.txt`);
      expect(() => resolveClaudePromptFileArgs(["--prompt-file", missingFile])).toThrow(`--prompt-file '${missingFile}' is not readable`);
    });

    it("throws when the prompt file cannot be read (directory)", () => {
      const dir = fs.mkdtempSync(path.join(os.tmpdir(), "claude-harness-dir-"));
      try {
        expect(() => resolveClaudePromptFileArgs(["--prompt-file", dir])).toThrow(`--prompt-file '${dir}' is not readable`);
      } finally {
        fs.rmdirSync(dir);
      }
    });

    it("places -- between --mcp-config and prompt to prevent ENAMETOOLONG (Claude Code 2.x variadic flag)", () => {
      // Claude Code 2.x treats any non-flag positional argument that follows
      // --mcp-config as a second config file path. A large prompt without the --
      // separator would exceed PATH_MAX (~4096 bytes) and fail with ENAMETOOLONG.
      const promptFile = path.join(os.tmpdir(), `claude-harness-prompt-${Date.now()}.txt`);
      const longPrompt = "<system>".padEnd(5000, "x");
      fs.writeFileSync(promptFile, longPrompt, "utf8");
      try {
        const result = resolveClaudePromptFileArgs(["--mcp-config", "/tmp/mcp-servers.json", "--prompt-file", promptFile]);
        // The -- must immediately precede the prompt content, not adjacent to --mcp-config.
        expect(result).toEqual(["--mcp-config", "/tmp/mcp-servers.json", "--", longPrompt]);
      } finally {
        fs.rmSync(promptFile);
      }
    });
  });

  describe("stripPromptFileArgs", () => {
    it("removes --prompt-file and its path argument", () => {
      const result = stripPromptFileArgs(["--print", "--prompt-file", "/tmp/prompt.txt", "--output-format", "json"]);
      expect(result).toEqual(["--print", "--output-format", "json"]);
    });

    it("passes through args with no --prompt-file", () => {
      const result = stripPromptFileArgs(["--print", "--output-format", "json"]);
      expect(result).toEqual(["--print", "--output-format", "json"]);
    });

    it("keeps a trailing --prompt-file with no following path (edge case)", () => {
      // When --prompt-file has no path, both resolveClaudePromptFileArgs (logs warning)
      // and stripPromptFileArgs leave it in place, so --continue retries also see it.
      const result = stripPromptFileArgs(["--print", "--prompt-file"]);
      expect(result).toEqual(["--print", "--prompt-file"]);
    });

    it("removes --prompt-file at the start", () => {
      const result = stripPromptFileArgs(["--prompt-file", "/tmp/prompt.txt", "--print"]);
      expect(result).toEqual(["--print"]);
    });
  });

  describe("isMaxTurnsExit", () => {
    it('returns true for a JSON result with "subtype":"error_max_turns"', () => {
      const output = '{"type":"result","subtype":"error_max_turns","is_error":true,"num_turns":13,' + '"terminal_reason":"max_turns","errors":["Reached maximum number of turns (12)"]}';
      expect(isMaxTurnsExit(output)).toBe(true);
    });

    it("returns true when subtype has extra whitespace around the colon", () => {
      expect(isMaxTurnsExit('"subtype" : "error_max_turns"')).toBe(true);
    });

    it("returns false for an overloaded_error output", () => {
      expect(isMaxTurnsExit('{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}')).toBe(false);
    });

    it("returns false for a rate_limit_error output", () => {
      expect(isMaxTurnsExit('{"type":"error","error":{"type":"rate_limit_error","message":"429 Too Many Requests"}}')).toBe(false);
    });

    it("returns false for an empty string", () => {
      expect(isMaxTurnsExit("")).toBe(false);
    });

    it("returns false for a successful result output", () => {
      expect(isMaxTurnsExit('{"type":"result","subtype":"success","is_error":false}')).toBe(false);
    });
  });

  describe("isRateLimitError", () => {
    it("returns true for stream-json api_error_status 429", () => {
      expect(isRateLimitError('{"type":"result","subtype":"success","is_error":true,"api_error_status":429}')).toBe(true);
    });

    it("returns true for stream-json request rejected 429 message", () => {
      expect(isRateLimitError("API Error: Request rejected (429) · This request would exceed your account's rate limit.")).toBe(true);
    });

    it("returns false for non-rate-limit output", () => {
      expect(isRateLimitError('{"type":"result","subtype":"success","is_error":false}')).toBe(false);
    });
  });

  describe("isAuthenticationFailedError", () => {
    it("returns true for authentication failed with request id", () => {
      expect(isAuthenticationFailedError("Authentication failed (Request ID: C818:3ED713:19D401B:1C446B7:69D653CA)")).toBe(true);
    });

    describe("isInvalidModelError", () => {
      it("returns true for model-not-supported errors", () => {
        expect(isInvalidModelError("Execution failed: CAPIError: 400 The requested model is not supported.")).toBe(true);
      });

      it("returns true for invalid model name errors", () => {
        expect(isInvalidModelError("invalid model name 'claude-sonnet-999'")).toBe(true);
        expect(isInvalidModelError("model 'claude-ultra' does not exist")).toBe(true);
        expect(isInvalidModelError("model claude-fake is not supported")).toBe(true);
        expect(isInvalidModelError("model gemini-v99 is unavailable")).toBe(true);
        expect(isInvalidModelError("model 'claude-3-5-sonnet@20241022' not found")).toBe(true);
      });

      it("returns false for unrelated errors", () => {
        expect(isInvalidModelError("rate_limit_error")).toBe(false);
        expect(isInvalidModelError("Error: invalid model response format")).toBe(false);
        expect(isInvalidModelError('{"type":"result","subtype":"error_max_turns","is_error":true}')).toBe(false);
        expect(isInvalidModelError("")).toBe(false);
      });
    });

    it("returns false for unrelated output", () => {
      expect(isAuthenticationFailedError("No authentication information found")).toBe(false);
      expect(isAuthenticationFailedError("rate_limit_error")).toBe(false);
    });
  });

  describe("isNoDeferredMarkerError", () => {
    it("returns true for the canonical no-deferred-marker error message", () => {
      const output =
        "Error: No deferred tool marker found in the resumed session. " +
        "Either the session was not deferred, the marker is stale (tool already ran), " +
        "or it exceeds the tail-scan window. Provide a prompt to continue the conversation.";
      expect(isNoDeferredMarkerError(output)).toBe(true);
    });

    it("returns true for mixed-case variant", () => {
      expect(isNoDeferredMarkerError("no deferred tool marker found")).toBe(true);
    });

    it("returns true when the error appears inside a larger log block", () => {
      const output = "[claude-harness] 2026-05-07T05:00:00.000Z attempt 1 failed: exitCode=1\n" + "Error: No deferred tool marker found in the resumed session.\n" + "[claude-harness] done: exitCode=1";
      expect(isNoDeferredMarkerError(output)).toBe(true);
    });

    it("returns false for an overloaded_error output", () => {
      expect(isNoDeferredMarkerError('{"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}')).toBe(false);
    });

    it("returns false for a max_turns exit output", () => {
      expect(isNoDeferredMarkerError('{"type":"result","subtype":"error_max_turns","is_error":true}')).toBe(false);
    });

    it("returns false for an empty string", () => {
      expect(isNoDeferredMarkerError("")).toBe(false);
    });

    it("returns false for a successful result output", () => {
      expect(isNoDeferredMarkerError('{"type":"result","subtype":"success","is_error":false}')).toBe(false);
    });
  });

  describe("isSignalTerminationExitCode", () => {
    it("returns true for SIGKILL/SIGTERM-style exit codes", () => {
      expect(isSignalTerminationExitCode(137)).toBe(true);
      expect(isSignalTerminationExitCode(143)).toBe(true);
    });

    it("returns false for non-signal exit codes", () => {
      expect(isSignalTerminationExitCode(1)).toBe(false);
      expect(isSignalTerminationExitCode(2)).toBe(false);
    });
  });

  describe("permission-denied classification helpers", () => {
    it("counts repeated permission-denied signals", () => {
      const output = "permission denied\nEACCES: permission denied\npermissions denied";
      expect(countPermissionDeniedIssues(output)).toBe(4);
    });

    it("detects numerous permission-denied issues at threshold", () => {
      const output = "permission denied\npermission denied\npermission denied";
      expect(hasNumerousPermissionDeniedIssues(output)).toBe(true);
    });

    it("does not classify sparse permission-denied output as numerous", () => {
      expect(hasNumerousPermissionDeniedIssues("permission denied")).toBe(false);
    });

    it("builds missing_tool payload for permission issues", () => {
      const payload = JSON.parse(buildMissingToolPermissionIssuePayload());
      expect(payload.type).toBe("missing_tool");
      expect(payload.reason).toContain("missing tool/permission issue");
      expect(payload.denied_commands).toEqual([]);
    });

    it("builds missing_tool payload with denied commands", () => {
      const payload = JSON.parse(buildMissingToolPermissionIssuePayload(["go version", "ls /usr/local/go"]));
      expect(payload.type).toBe("missing_tool");
      expect(payload.denied_commands).toEqual(["go version", "ls /usr/local/go"]);
    });
  });

  describe("extractDeniedCommands", () => {
    it("returns empty array for empty output", () => {
      expect(extractDeniedCommands("")).toEqual([]);
      expect(extractDeniedCommands(null)).toEqual([]);
    });

    it("extracts command from line with box-drawing pipe marker (│) before permission denied", () => {
      const output = ["\u2713 Some successful step", "\u2717 Check if go command works (shell)", "  \u2502 go version 2>&1", "  \u2514 Permission denied and could not request permission from user"].join("\n");
      expect(extractDeniedCommands(output)).toEqual(["go version 2>&1"]);
    });

    it("extracts command with plain pipe (|) before permission denied", () => {
      const output = ["| ls -la", "Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual(["ls -la"]);
    });

    it("deduplicates repeated denied commands", () => {
      const output = ["  \u2502 go version", "  Permission denied", "  \u2502 go version", "  Permission denied", "  \u2502 go version", "  Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual(["go version"]);
    });

    it("extracts multiple distinct denied commands", () => {
      const output = ["  \u2502 go version 2>&1", "  Permission denied", "  \u2502 ls /usr/local/go/bin/go", "  Permission denied", "  \u2502 which go", "  Permission denied"].join("\n");
      const result = extractDeniedCommands(output);
      expect(result).toContain("go version 2>&1");
      expect(result).toContain("ls /usr/local/go/bin/go");
      expect(result).toContain("which go");
    });

    it("returns empty array when no pipe markers are present before permission denied", () => {
      const output = "Some output\nPermission denied\nMore output";
      expect(extractDeniedCommands(output)).toEqual([]);
    });

    it("does not capture suffix of a command containing an internal pipe", () => {
      // "find . -name '*.go' | sort" should not match by splitting on the internal |
      const output = ["  find . -name '*.go' | sort", "  Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual([]);
    });
  });

  describe("shouldRetryWithContinue", () => {
    it("does not use --continue for signal-style termination exit codes", () => {
      for (const exitCode of [137, 143]) {
        const result = shouldRetryWithContinue({
          attempt: 0,
          maxRetries: 3,
          exitCode,
          hasOutput: true,
          isNoDeferredMarker: false,
          continueDisabledPermanently: false,
        });
        expect(result).toBe(false);
      }
    });

    it("uses a fresh retry after a --continue attempt hits no-deferred-marker", () => {
      const stubScript = `
const fs = require("fs");
const callsPath = process.env.CLAUDE_HARNESS_STUB_CALLS;
const args = process.argv.slice(2);
const priorCalls = fs.existsSync(callsPath) ? fs.readFileSync(callsPath, "utf8").trim().split("\\n").filter(Boolean).length : 0;
fs.appendFileSync(callsPath, JSON.stringify({ args }) + "\\n", "utf8");

if (priorCalls === 0) {
  process.stdout.write("partial execution before retry\\n");
  process.exit(1);
}

if (priorCalls === 1) {
  if (!args.includes("--continue")) {
    process.stderr.write("expected --continue on first retry\\n");
    process.exit(9);
  }
  process.stderr.write("Error: No deferred tool marker found in the resumed session.\\n");
  process.exit(1);
}

if (args.includes("--continue")) {
  process.stderr.write("fresh retry unexpectedly used --continue\\n");
  process.exit(9);
}
process.stdout.write("fresh retry succeeded\\n");
process.exit(0);
`;
      const { result, calls } = runHarnessWithStub({ stubScript });

      expect(result.status, result.stderr).toBe(0);
      expect(calls.map(call => call.args.includes("--continue"))).toEqual([false, true, false]);
      expect(calls[2].args).toContain("fix the bug");
      expect(result.stderr).toContain("failure_reason=harness_retry_path_invalid");
    }, 50000);

    it("strips user-supplied --continue on fresh retry after invalid continue-path detection", () => {
      const stubScript = `
const fs = require("fs");
const callsPath = process.env.CLAUDE_HARNESS_STUB_CALLS;
const args = process.argv.slice(2);
const priorCalls = fs.existsSync(callsPath) ? fs.readFileSync(callsPath, "utf8").trim().split("\\n").filter(Boolean).length : 0;
fs.appendFileSync(callsPath, JSON.stringify({ args }) + "\\n", "utf8");

if (priorCalls === 0) {
  process.stdout.write("partial execution before retry\\n");
  process.exit(1);
}

if (priorCalls === 1) {
  if (args.filter(arg => arg === "--continue").length !== 1) {
    process.stderr.write("expected exactly one --continue on first retry\\n");
    process.exit(9);
  }
  process.stderr.write("Error: No deferred tool marker found in the resumed session.\\n");
  process.exit(1);
}

if (args.includes("--continue")) {
  process.stderr.write("fresh retry unexpectedly used --continue\\n");
  process.exit(9);
}
process.stdout.write("fresh retry succeeded\\n");
process.exit(0);
`;
      const { result, calls } = runHarnessWithStub({ stubScript, extraArgs: ["--continue"] });

      expect(result.status, result.stderr).toBe(0);
      expect(calls.map(call => call.args.includes("--continue"))).toEqual([true, true, false]);
    }, 50000);

    it("uses a fresh retry after signal-style termination instead of --continue", () => {
      const stubScript = `
const fs = require("fs");
const callsPath = process.env.CLAUDE_HARNESS_STUB_CALLS;
const args = process.argv.slice(2);
const priorCalls = fs.existsSync(callsPath) ? fs.readFileSync(callsPath, "utf8").trim().split("\\n").filter(Boolean).length : 0;
fs.appendFileSync(callsPath, JSON.stringify({ args }) + "\\n", "utf8");

if (priorCalls === 0) {
  process.stdout.write("partial execution before SIGTERM-style exit\\n");
  process.exit(143);
}

if (args.includes("--continue")) {
  process.stderr.write("signal retry unexpectedly used --continue\\n");
  process.exit(9);
}
process.stdout.write("fresh retry after signal exit succeeded\\n");
process.exit(0);
`;
      const { result, calls } = runHarnessWithStub({ stubScript });

      expect(result.status, result.stderr).toBe(0);
      expect(calls.map(call => call.args.includes("--continue"))).toEqual([false, false]);
      expect(calls[1].args).toContain("fix the bug");
      expect(result.stderr).toContain("failure_reason=cancelled_or_timed_out");
    }, 30000);

    it("retries one no-output startup failure as a fresh run by default", () => {
      const stubScript = `
const fs = require("fs");
const callsPath = process.env.CLAUDE_HARNESS_STUB_CALLS;
const args = process.argv.slice(2);
const priorCalls = fs.existsSync(callsPath) ? fs.readFileSync(callsPath, "utf8").trim().split("\\n").filter(Boolean).length : 0;
fs.appendFileSync(callsPath, JSON.stringify({ args }) + "\\n", "utf8");
if (priorCalls === 0) process.exit(1);
process.stdout.write("startup retry succeeded\\n");
process.exit(0);
`;
      const { result, calls } = runHarnessWithStub({ stubScript });
      expect(result.status, result.stderr).toBe(0);
      expect(calls.length).toBe(2);
      expect(calls[1].args.includes("--continue")).toBe(false);
      expect(result.stderr).toContain("no output produced — retrying startup as fresh run");
    });

    it("does not retry no-output startup failure when GH_AW_CLAUDE_STARTUP_RETRIES=0", () => {
      const stubScript = `
const fs = require("fs");
const callsPath = process.env.CLAUDE_HARNESS_STUB_CALLS;
fs.appendFileSync(callsPath, JSON.stringify({ args: process.argv.slice(2) }) + "\\n", "utf8");
process.exit(1);
`;
      const { result, calls } = runHarnessWithStub({
        stubScript,
        extraEnv: { GH_AW_CLAUDE_STARTUP_RETRIES: "0" },
      });
      expect(result.status).toBe(1);
      expect(calls.length).toBe(1);
      expect(calls[0].args).toContain("fix the bug");
      expect(result.stderr).toContain("startup retry budget exhausted: 0/0");
    });

    it("returns true for normal partial-execution retry", () => {
      const result = shouldRetryWithContinue({
        attempt: 0,
        maxRetries: 3,
        exitCode: 1,
        hasOutput: true,
        isNoDeferredMarker: false,
        continueDisabledPermanently: false,
      });
      expect(result).toBe(true);
    });

    it("returns false when no-deferred-marker error is present", () => {
      const result = shouldRetryWithContinue({
        attempt: 0,
        maxRetries: 3,
        exitCode: 1,
        hasOutput: true,
        isNoDeferredMarker: true,
        continueDisabledPermanently: false,
      });
      expect(result).toBe(false);
    });
  });

  describe("auth failure retry policy", () => {
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @returns {boolean}
     */
    function shouldRetry(result, attempt) {
      if (result.exitCode === 0) return false;
      if (attempt === 0 && isAuthenticationFailedError(result.output)) return false;
      return attempt < MAX_RETRIES && result.hasOutput;
    }

    it("does not retry when first attempt fails authentication", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Authentication failed (Request ID: 123)" };
      expect(shouldRetry(result, 0)).toBe(false);
    });
  });

  describe("retry configuration", () => {
    it("uses the default retry settings when env vars are unset", () => {
      expect(resolveRetryConfig({})).toEqual({
        maxRetries: 3,
        initialDelayMs: 5000,
        backoffMultiplier: 2,
        maxDelayMs: 60000,
      });
    });

    it("accepts env overrides for retry settings", () => {
      expect(
        resolveRetryConfig({
          GH_AW_HARNESS_MAX_RETRIES: "6",
          GH_AW_HARNESS_INITIAL_DELAY_MS: "250",
          GH_AW_HARNESS_BACKOFF_MULTIPLIER: "1.25",
          GH_AW_HARNESS_MAX_DELAY_MS: "10000",
        })
      ).toEqual({
        maxRetries: 6,
        initialDelayMs: 250,
        backoffMultiplier: 1.25,
        maxDelayMs: 10000,
      });
    });

    it("falls back to defaults for invalid env values and clamps max delay", () => {
      const logs = [];
      const retryConfig = resolveRetryConfig(
        {
          GH_AW_HARNESS_MAX_RETRIES: "-1",
          GH_AW_HARNESS_INITIAL_DELAY_MS: "6000",
          GH_AW_HARNESS_BACKOFF_MULTIPLIER: "0",
          GH_AW_HARNESS_MAX_DELAY_MS: "1000",
        },
        msg => logs.push(msg)
      );
      expect(retryConfig).toEqual({
        maxRetries: 3,
        initialDelayMs: 6000,
        backoffMultiplier: 2,
        maxDelayMs: 6000,
      });
      expect(logs.some(msg => msg.includes("GH_AW_HARNESS_MAX_RETRIES"))).toBe(true);
      expect(logs.some(msg => msg.includes("GH_AW_HARNESS_BACKOFF_MULTIPLIER"))).toBe(true);
      expect(logs.some(msg => msg.includes("clamping max delay"))).toBe(true);
    });

    it("accepts max-retries=0 to disable retries entirely", () => {
      const retryConfig = resolveRetryConfig({ GH_AW_HARNESS_MAX_RETRIES: "0" });
      expect(retryConfig.maxRetries).toBe(0);
    });

    it("clamps max-retries to 100 when given an excessively large value", () => {
      const logs = [];
      const retryConfig = resolveRetryConfig({ GH_AW_HARNESS_MAX_RETRIES: "9999" }, msg => logs.push(msg));
      expect(retryConfig.maxRetries).toBe(100);
      expect(logs.some(msg => msg.includes("GH_AW_HARNESS_MAX_RETRIES"))).toBe(true);
    });

    it("rejects non-decimal integer formats such as '1e3' and '0x10'", () => {
      const config1 = resolveRetryConfig({ GH_AW_HARNESS_MAX_RETRIES: "1e3" });
      expect(config1.maxRetries).toBe(3);
      const config2 = resolveRetryConfig({ GH_AW_HARNESS_INITIAL_DELAY_MS: "0x10" });
      expect(config2.initialDelayMs).toBe(5000);
    });

    it("uses startup retry default and clamps overrides to [0..2]", () => {
      expect(resolveStartupRetryLimit({})).toBe(1);
      expect(resolveStartupRetryLimit({ GH_AW_CLAUDE_STARTUP_RETRIES: "2" })).toBe(2);
      expect(resolveStartupRetryLimit({ GH_AW_CLAUDE_STARTUP_RETRIES: "-5" })).toBe(0);
      expect(resolveStartupRetryLimit({ GH_AW_CLAUDE_STARTUP_RETRIES: "9" })).toBe(2);
      expect(resolveStartupRetryLimit({ GH_AW_CLAUDE_STARTUP_RETRIES: "nope" })).toBe(1);
    });
  });

  describe("noop pre-flight and retry guard", () => {
    it("skips the agent when a noop is already in safe-outputs before the run", () => {
      const tempDir = makeHarnessTempDir("claude-noop-preflight-");
      const safeOutputsPath = path.join(tempDir, "safe-outputs.jsonl");
      fs.writeFileSync(safeOutputsPath, '{"type":"noop","message":"nothing to do"}\n', "utf8");
      const stubPath = path.join(tempDir, "stub.cjs");
      const promptPath = path.join(tempDir, "prompt.txt");
      const callsPath = path.join(tempDir, "calls.jsonl");
      fs.writeFileSync(
        stubPath,
        `const fs = require("fs");
const callsPath = process.env.CLAUDE_HARNESS_STUB_CALLS;
fs.appendFileSync(callsPath, JSON.stringify({args: process.argv.slice(2)}) + "\\n");
process.exit(0);`,
        "utf8"
      );
      fs.writeFileSync(promptPath, "fix the bug", "utf8");

      const result = spawnSync(process.execPath, ["claude_harness.cjs", process.execPath, stubPath, "--print", "--prompt-file", promptPath], {
        cwd: path.dirname(require.resolve("./claude_harness.cjs")),
        env: { ...process.env, CLAUDE_HARNESS_STUB_CALLS: callsPath, GH_AW_SAFE_OUTPUTS: safeOutputsPath },
        encoding: "utf8",
        timeout: 10000,
      });
      // Agent stub should never have been invoked
      const stubCallCount = fs.existsSync(callsPath) ? fs.readFileSync(callsPath, "utf8").trim().split("\n").filter(Boolean).length : 0;
      expect(stubCallCount).toBe(0);
      expect(result.status).toBe(0);
      expect(result.stderr).toContain("pre-flight: noop message found in safe-outputs");
    });

    it("does not retry after a failed run when a noop was written to safe-outputs", () => {
      const tempDir = makeHarnessTempDir("claude-noop-retry-");
      const safeOutputsPath = path.join(tempDir, "safe-outputs.jsonl");
      const stubPath = path.join(tempDir, "stub.cjs");
      const promptPath = path.join(tempDir, "prompt.txt");
      const callsPath = path.join(tempDir, "calls.jsonl");
      // Stub writes a noop on the first call then fails; harness must not retry.
      fs.writeFileSync(
        stubPath,
        `const fs = require("fs");
const callsPath = process.env.CLAUDE_HARNESS_STUB_CALLS;
const safeOutputsPath = process.env.GH_AW_SAFE_OUTPUTS;
fs.appendFileSync(callsPath, JSON.stringify({args: process.argv.slice(2)}) + "\\n");
fs.appendFileSync(safeOutputsPath, JSON.stringify({type:"noop",message:"nothing to do"}) + "\\n");
process.exit(1);`,
        "utf8"
      );
      fs.writeFileSync(promptPath, "fix the bug", "utf8");

      const result = spawnSync(process.execPath, ["claude_harness.cjs", process.execPath, stubPath, "--print", "--prompt-file", promptPath], {
        cwd: path.dirname(require.resolve("./claude_harness.cjs")),
        env: { ...process.env, CLAUDE_HARNESS_STUB_CALLS: callsPath, GH_AW_SAFE_OUTPUTS: safeOutputsPath },
        encoding: "utf8",
        timeout: 10000,
      });
      const callCount = fs.readFileSync(callsPath, "utf8").trim().split("\n").filter(Boolean).length;
      // Only one attempt — no retries after noop detected
      expect(callCount).toBe(1);
      // Harness exits 0 because noop means the work is done
      expect(result.status).toBe(0);
      expect(result.stderr).toContain("noop message found in safe-outputs — not retrying");
    });
  });
});
