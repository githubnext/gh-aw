import { describe, it, expect } from "vitest";
import { createRequire } from "module";
import fs from "fs";
import os from "os";
import path from "path";

const require = createRequire(import.meta.url);
const { resolveCodexPromptFileArgs, isRateLimitError, isServerError, countPermissionDeniedIssues, hasNumerousPermissionDeniedIssues, extractDeniedCommands, buildMissingToolPermissionIssuePayload } = require("./codex_harness.cjs");

describe("codex_harness.cjs", () => {
  describe("resolveCodexPromptFileArgs", () => {
    it("replaces --prompt-file with the file's content as the last positional arg", () => {
      const promptFile = path.join(os.tmpdir(), `codex-harness-prompt-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "fix the bug", "utf8");
      try {
        const result = resolveCodexPromptFileArgs(["exec", "--dangerously-bypass-approvals-and-sandbox", "--prompt-file", promptFile]);
        expect(result).toEqual(["exec", "--dangerously-bypass-approvals-and-sandbox", "fix the bug"]);
      } finally {
        fs.rmSync(promptFile);
      }
    });

    it("appends prompt content as the last arg when only --prompt-file is provided", () => {
      const promptFile = path.join(os.tmpdir(), `codex-harness-prompt-${Date.now()}.txt`);
      fs.writeFileSync(promptFile, "my task", "utf8");
      try {
        const result = resolveCodexPromptFileArgs(["--prompt-file", promptFile]);
        expect(result).toEqual(["my task"]);
      } finally {
        fs.rmSync(promptFile);
      }
    });

    it("passes through args that have no --prompt-file", () => {
      const result = resolveCodexPromptFileArgs(["exec", "--dangerously-bypass-approvals-and-sandbox"]);
      expect(result).toEqual(["exec", "--dangerously-bypass-approvals-and-sandbox"]);
    });

    it("preserves args when --prompt-file is provided without a path", () => {
      const result = resolveCodexPromptFileArgs(["exec", "--prompt-file"]);
      // When no path follows --prompt-file, it is preserved as-is
      expect(result).toEqual(["exec", "--prompt-file"]);
    });

    it("throws when the prompt file does not exist", () => {
      const missingFile = path.join(os.tmpdir(), `codex-harness-missing-${Date.now()}.txt`);
      expect(() => resolveCodexPromptFileArgs(["--prompt-file", missingFile])).toThrow(`--prompt-file '${missingFile}' is not readable`);
    });

    it("throws when the prompt file cannot be read (directory)", () => {
      const dir = fs.mkdtempSync(path.join(os.tmpdir(), "codex-harness-dir-"));
      try {
        expect(() => resolveCodexPromptFileArgs(["--prompt-file", dir])).toThrow(`--prompt-file '${dir}' is not readable`);
      } finally {
        fs.rmdirSync(dir);
      }
    });
  });

  describe("isRateLimitError", () => {
    it("returns true for rate_limit_exceeded error", () => {
      expect(isRateLimitError("Error: rate_limit_exceeded")).toBe(true);
    });

    it("returns true for 429 Too Many Requests", () => {
      expect(isRateLimitError("429 Too Many Requests")).toBe(true);
    });

    it("returns true for RateLimitError", () => {
      expect(isRateLimitError("RateLimitError: You exceeded your current quota")).toBe(true);
    });

    it("returns false for unrelated errors", () => {
      expect(isRateLimitError("Error: ENOENT: no such file")).toBe(false);
      expect(isRateLimitError("Fatal: out of memory")).toBe(false);
      expect(isRateLimitError("")).toBe(false);
    });

    it("returns false for a 500 server error", () => {
      expect(isRateLimitError("500 Internal Server Error")).toBe(false);
    });
  });

  describe("isServerError", () => {
    it("returns true for InternalServerError", () => {
      expect(isServerError("InternalServerError: The server had an error processing your request")).toBe(true);
    });

    it("returns true for ServiceUnavailableError", () => {
      expect(isServerError("ServiceUnavailableError: The server is temporarily unable to service your request")).toBe(true);
    });

    it("returns true for 500 Internal Server Error", () => {
      expect(isServerError("500 Internal Server Error")).toBe(true);
    });

    it("returns true for 503 Service Unavailable", () => {
      expect(isServerError("503 Service Unavailable")).toBe(true);
    });

    it("returns false for rate limit errors", () => {
      expect(isServerError("rate_limit_exceeded")).toBe(false);
      expect(isServerError("429 Too Many Requests")).toBe(false);
    });

    it("returns false for unrelated errors", () => {
      expect(isServerError("Error: ENOENT: no such file")).toBe(false);
      expect(isServerError("")).toBe(false);
    });
  });

  describe("permission-denied classification helpers", () => {
    it("counts repeated permission-denied signals", () => {
      const output = "permission denied\npermissions denied\nEACCES: permission denied";
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
      const result = extractDeniedCommands(output);
      expect(result).toEqual(["go version"]);
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

    it("looks back up to 3 lines for command context", () => {
      const output = ["  \u2502 make test", "Running...", "Still running...", "  Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual(["make test"]);
    });

    it("does not look back more than 3 lines", () => {
      const output = ["  \u2502 make test", "line2", "line3", "line4", "  Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual([]);
    });

    it("does not capture suffix of a command containing an internal pipe", () => {
      // "find . -name '*.go' | sort" should not match by splitting on the internal |
      const output = ["  find . -name '*.go' | sort", "  Permission denied"].join("\n");
      expect(extractDeniedCommands(output)).toEqual([]);
    });
  });

  describe("retry policy: fresh run on partial execution", () => {
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @returns {boolean}
     */
    function shouldRetry(result, attempt) {
      if (result.exitCode === 0) return false;
      const RATE_LIMIT_ERROR_PATTERN = /rate_limit_exceeded|429 Too Many Requests|RateLimitError/i;
      const SERVER_ERROR_PATTERN = /InternalServerError|ServiceUnavailableError|500 Internal Server Error|503 Service Unavailable/i;
      if (hasNumerousPermissionDeniedIssues(result.output)) return false;
      const isTransient = RATE_LIMIT_ERROR_PATTERN.test(result.output) || SERVER_ERROR_PATTERN.test(result.output);
      return attempt < MAX_RETRIES && (result.hasOutput || isTransient);
    }

    it("retries on rate limit error even without output", () => {
      const result = { exitCode: 1, hasOutput: false, output: "rate_limit_exceeded" };
      expect(shouldRetry(result, 0)).toBe(true);
    });

    it("retries on server error even without output", () => {
      const result = { exitCode: 1, hasOutput: false, output: "InternalServerError" };
      expect(shouldRetry(result, 0)).toBe(true);
    });

    it("retries on any other non-zero exit when session produced output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Error: connection reset" };
      expect(shouldRetry(result, 0)).toBe(true);
    });

    it("does not retry when no output was produced and no transient error", () => {
      const result = { exitCode: 1, hasOutput: false, output: "" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry after retries are exhausted", () => {
      const result = { exitCode: 1, hasOutput: true, output: "rate_limit_exceeded" };
      expect(shouldRetry(result, MAX_RETRIES)).toBe(false);
    });

    it("does not retry on success", () => {
      const result = { exitCode: 0, hasOutput: true, output: "Task complete" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry when numerous permission-denied issues are present", () => {
      const result = { exitCode: 1, hasOutput: true, output: "permission denied\npermission denied\npermission denied" };
      expect(shouldRetry(result, 0)).toBe(false);
    });
  });
});
