import { describe, it, expect } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const { resolveClaudePromptFileArgs, stripPromptFileArgs, isMaxTurnsExit, isNoDeferredMarkerError } = require("./claude_harness.cjs");

describe("claude_harness.cjs", () => {
  describe("resolveClaudePromptFileArgs", () => {
    it("replaces --prompt-file with the file's content as the last positional arg", () => {
      const promptFile = path.join(os.tmpdir(), `claude-harness-prompt-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "fix the bug", "utf8");
      try {
        const result = resolveClaudePromptFileArgs(["--print", "--prompt-file", promptFile, "--output-format", "stream-json"]);
        expect(result).toEqual(["--print", "--output-format", "stream-json", "fix the bug"]);
      } finally {
        fs.rmSync(promptFile);
      }
    });

    it("appends prompt content as the last arg when other positional args precede it", () => {
      const promptFile = path.join(os.tmpdir(), `claude-harness-prompt-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "my task", "utf8");
      try {
        const result = resolveClaudePromptFileArgs(["--prompt-file", promptFile]);
        expect(result).toEqual(["my task"]);
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
});
