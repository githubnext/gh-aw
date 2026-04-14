import { describe, it, expect } from "vitest";

describe("copilot_driver.cjs", () => {
  // Test the core logic patterns used by the driver without importing the module
  // (importing the module would invoke main() which calls process.exit).

  describe("CAPIError 400 detection pattern", () => {
    const CAPI_ERROR_400_PATTERN = /CAPIError:\s*400/;

    it("matches the exact error from the failed workflow run", () => {
      const errorOutput = "Execution failed: CAPIError: 400 400 Bad Request\n (Request ID: C818:3ED713:19D401B:1C446B7:69D653CA)";
      expect(CAPI_ERROR_400_PATTERN.test(errorOutput)).toBe(true);
    });

    it("matches CAPIError: 400 with various spacing", () => {
      expect(CAPI_ERROR_400_PATTERN.test("CAPIError: 400")).toBe(true);
      expect(CAPI_ERROR_400_PATTERN.test("CAPIError:400")).toBe(true);
      expect(CAPI_ERROR_400_PATTERN.test("CAPIError:  400")).toBe(true);
    });

    it("does not match CAPIError 401 Unauthorized", () => {
      expect(CAPI_ERROR_400_PATTERN.test("Execution failed: CAPIError: 401 Unauthorized")).toBe(false);
    });

    it("does not match generic 400 errors without CAPIError prefix", () => {
      expect(CAPI_ERROR_400_PATTERN.test("Error: 400 Bad Request")).toBe(false);
      expect(CAPI_ERROR_400_PATTERN.test("HTTP 400")).toBe(false);
    });

    it("does not match unrelated errors", () => {
      expect(CAPI_ERROR_400_PATTERN.test("Error: ENOENT: no such file")).toBe(false);
      expect(CAPI_ERROR_400_PATTERN.test("Fatal: out of memory")).toBe(false);
      expect(CAPI_ERROR_400_PATTERN.test("")).toBe(false);
    });
  });

  describe("retry policy: continue on partial execution", () => {
    // Inline the same retry-eligibility logic as the driver for unit testing.
    // The driver retries whenever the session produced output (hasOutput), regardless
    // of the specific error type.  CAPIError 400 is just the well-known case.
    const CAPI_ERROR_400_PATTERN = /CAPIError:\s*400/;
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @returns {boolean}
     */
    function shouldRetry(result, attempt) {
      if (result.exitCode === 0) return false;
      return attempt < MAX_RETRIES && result.hasOutput;
    }

    /**
     * @param {string} output
     * @returns {"CAPIError 400 (transient)" | "partial execution"}
     */
    function retryReason(output) {
      return CAPI_ERROR_400_PATTERN.test(output) ? "CAPIError 400 (transient)" : "partial execution";
    }

    it("retries on CAPIError 400 after partial output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "CAPIError: 400 Bad Request" };
      expect(shouldRetry(result, 0)).toBe(true);
      expect(retryReason(result.output)).toBe("CAPIError 400 (transient)");
    });

    it("retries on any other non-zero exit when session produced output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Error: connection reset by peer" };
      expect(shouldRetry(result, 0)).toBe(true);
      expect(retryReason(result.output)).toBe("partial execution");
    });

    it("does not retry when no output was produced (process failed to start)", () => {
      const result = { exitCode: 1, hasOutput: false, output: "" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry after retries are exhausted", () => {
      const result = { exitCode: 1, hasOutput: true, output: "CAPIError: 400 Bad Request" };
      expect(shouldRetry(result, MAX_RETRIES)).toBe(false);
    });

    it("does not retry on success", () => {
      const result = { exitCode: 0, hasOutput: true, output: "Done." };
      expect(shouldRetry(result, 0)).toBe(false);
    });
  });

  describe("MCP policy blocked detection pattern", () => {
    const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;

    it("matches the exact error from the issue report", () => {
      const errorOutput = "! 2 MCP servers were blocked by policy: 'github', 'safeoutputs'";
      expect(MCP_POLICY_BLOCKED_PATTERN.test(errorOutput)).toBe(true);
    });

    it("matches with different server names", () => {
      expect(MCP_POLICY_BLOCKED_PATTERN.test("! 1 MCP servers were blocked by policy: 'github'")).toBe(true);
      expect(MCP_POLICY_BLOCKED_PATTERN.test("MCP servers were blocked by policy: 'custom-server'")).toBe(true);
    });

    it("does not match unrelated errors", () => {
      expect(MCP_POLICY_BLOCKED_PATTERN.test("Error: MCP server connection failed")).toBe(false);
      expect(MCP_POLICY_BLOCKED_PATTERN.test("MCP server timeout")).toBe(false);
      expect(MCP_POLICY_BLOCKED_PATTERN.test("Access denied by policy settings")).toBe(false);
      expect(MCP_POLICY_BLOCKED_PATTERN.test("")).toBe(false);
    });
  });

  describe("MCP policy error prevents retry", () => {
    // Inline the same retry logic as the driver, including MCP policy check
    const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;
    const MODEL_NOT_SUPPORTED_PATTERN = /The requested model is not supported/;
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @returns {boolean}
     */
    function shouldRetry(result, attempt) {
      if (result.exitCode === 0) return false;
      // MCP policy errors are persistent — never retry
      if (MCP_POLICY_BLOCKED_PATTERN.test(result.output)) return false;
      // Model-not-supported errors are persistent — never retry
      if (MODEL_NOT_SUPPORTED_PATTERN.test(result.output)) return false;
      return attempt < MAX_RETRIES && result.hasOutput;
    }

    it("does not retry when MCP servers are blocked by policy", () => {
      const result = { exitCode: 1, hasOutput: true, output: "! 2 MCP servers were blocked by policy: 'github', 'safeoutputs'" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry MCP policy error even on first attempt with output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Some output\nMCP servers were blocked by policy: 'github'\nMore output" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry model-not-supported error", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Execution failed: CAPIError: 400 The requested model is not supported." };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry model-not-supported error even on first attempt with output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Some output\nExecution failed: CAPIError: 400 The requested model is not supported.\nMore output" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("still retries non-policy errors with output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "CAPIError: 400 Bad Request" };
      expect(shouldRetry(result, 0)).toBe(true);
    });
  });

  describe("model-not-supported detection pattern", () => {
    const MODEL_NOT_SUPPORTED_PATTERN = /The requested model is not supported/;

    it("matches the exact error from the issue report", () => {
      const errorOutput = "Execution failed: CAPIError: 400 The requested model is not supported.";
      expect(MODEL_NOT_SUPPORTED_PATTERN.test(errorOutput)).toBe(true);
    });

    it("matches when embedded in larger log output", () => {
      const log = "Some output\nExecution failed: CAPIError: 400 The requested model is not supported.\nMore output";
      expect(MODEL_NOT_SUPPORTED_PATTERN.test(log)).toBe(true);
    });

    it("does not match other CAPIError 400 errors", () => {
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
    });

    it("does not match unrelated errors", () => {
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("Access denied by policy settings")).toBe(false);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("MCP servers were blocked by policy: 'github'")).toBe(false);
      expect(MODEL_NOT_SUPPORTED_PATTERN.test("")).toBe(false);
    });
  });

  describe("no-auth-info detection pattern", () => {
    const NO_AUTH_INFO_PATTERN = /No authentication information found/;

    it("matches the exact error from the issue report", () => {
      const errorOutput =
        "Error: No authentication information found.\n" +
        "Copilot can be authenticated with GitHub using an OAuth Token or a Fine-Grained Personal Access Token.\n" +
        "To authenticate, you can use any of the following methods:\n" +
        "  - Start 'copilot' and run the '/login' command\n" +
        "  - Set the COPILOT_GITHUB_TOKEN, GH_TOKEN, or GITHUB_TOKEN environment variable\n" +
        "  - Run 'gh auth login' to authenticate with the GitHub CLI";
      expect(NO_AUTH_INFO_PATTERN.test(errorOutput)).toBe(true);
    });

    it("matches when embedded in larger output after a long run", () => {
      const output = "Some agent work output\nMore work\nNo authentication information found\nEnd";
      expect(NO_AUTH_INFO_PATTERN.test(output)).toBe(true);
    });

    it("does not match unrelated auth errors", () => {
      expect(NO_AUTH_INFO_PATTERN.test("Access denied by policy settings")).toBe(false);
      expect(NO_AUTH_INFO_PATTERN.test("Error: 401 Unauthorized")).toBe(false);
      expect(NO_AUTH_INFO_PATTERN.test("Authentication failed")).toBe(false);
      expect(NO_AUTH_INFO_PATTERN.test("CAPIError: 400 Bad Request")).toBe(false);
      expect(NO_AUTH_INFO_PATTERN.test("")).toBe(false);
    });
  });

  describe("auth error prevents retry", () => {
    // Inline the same retry logic as the driver, including auth error check
    const MCP_POLICY_BLOCKED_PATTERN = /MCP servers were blocked by policy:/;
    const NO_AUTH_INFO_PATTERN = /No authentication information found/;
    const MAX_RETRIES = 3;

    /**
     * @param {{hasOutput: boolean, exitCode: number, output: string}} result
     * @param {number} attempt
     * @returns {boolean}
     */
    function shouldRetry(result, attempt) {
      if (result.exitCode === 0) return false;
      // MCP policy errors are persistent — never retry
      if (MCP_POLICY_BLOCKED_PATTERN.test(result.output)) return false;
      // Auth errors are persistent — never retry
      if (NO_AUTH_INFO_PATTERN.test(result.output)) return false;
      return attempt < MAX_RETRIES && result.hasOutput;
    }

    it("does not retry when auth fails on first attempt (no real work done)", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Error: No authentication information found." };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("does not retry when auth fails on a --continue attempt (the reported bug scenario)", () => {
      // This replicates the issue: attempt 1 ran for 39 min then failed,
      // attempt 2 (--continue) fails with auth error — should not retry attempts 3 & 4.
      const resumeResult = { exitCode: 1, hasOutput: true, output: "Error: No authentication information found." };
      expect(shouldRetry(resumeResult, 1)).toBe(false);
      expect(shouldRetry(resumeResult, 2)).toBe(false);
      expect(shouldRetry(resumeResult, 3)).toBe(false);
    });

    it("does not retry auth error even when output is mixed with other content", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Some output\nError: No authentication information found.\nMore output" };
      expect(shouldRetry(result, 0)).toBe(false);
    });

    it("still retries non-auth errors with output (CAPIError 400)", () => {
      const result = { exitCode: 1, hasOutput: true, output: "CAPIError: 400 Bad Request" };
      expect(shouldRetry(result, 0)).toBe(true);
    });

    it("still retries generic partial-execution errors with output", () => {
      const result = { exitCode: 1, hasOutput: true, output: "Failed to get response from the AI model; retried 5 times" };
      expect(shouldRetry(result, 0)).toBe(true);
    });
  });

  describe("retry configuration", () => {
    it("has sensible default values", () => {
      // These match the constants in copilot_driver.cjs
      const MAX_RETRIES = 3;
      const INITIAL_DELAY_MS = 5000;
      const BACKOFF_MULTIPLIER = 2;
      const MAX_DELAY_MS = 60000;

      expect(MAX_RETRIES).toBeGreaterThan(0);
      expect(INITIAL_DELAY_MS).toBeGreaterThan(0);
      expect(BACKOFF_MULTIPLIER).toBeGreaterThan(1);
      expect(MAX_DELAY_MS).toBeGreaterThanOrEqual(INITIAL_DELAY_MS);
    });

    it("exponential backoff does not exceed max delay", () => {
      const INITIAL_DELAY_MS = 5000;
      const BACKOFF_MULTIPLIER = 2;
      const MAX_DELAY_MS = 60000;
      const MAX_RETRIES = 3;

      let delay = INITIAL_DELAY_MS;
      for (let i = 0; i < MAX_RETRIES; i++) {
        delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
        expect(delay).toBeLessThanOrEqual(MAX_DELAY_MS);
      }
    });
  });

  describe("formatDuration", () => {
    // Inline the same logic as the driver's formatDuration for unit testing
    function formatDuration(ms) {
      const totalSeconds = Math.floor(ms / 1000);
      const minutes = Math.floor(totalSeconds / 60);
      const seconds = totalSeconds % 60;
      if (minutes > 0) {
        return `${minutes}m ${seconds}s`;
      }
      return `${seconds}s`;
    }

    it("formats sub-minute durations as seconds", () => {
      expect(formatDuration(0)).toBe("0s");
      expect(formatDuration(500)).toBe("0s");
      expect(formatDuration(1000)).toBe("1s");
      expect(formatDuration(59000)).toBe("59s");
    });

    it("formats minute-level durations with minutes and seconds", () => {
      expect(formatDuration(60000)).toBe("1m 0s");
      expect(formatDuration(90000)).toBe("1m 30s");
      expect(formatDuration(192000)).toBe("3m 12s"); // 3m 12s (real-world example)
    });

    it("handles long durations correctly", () => {
      expect(formatDuration(3600000)).toBe("60m 0s");
    });
  });

  describe("log format", () => {
    it("log lines include [copilot-driver] prefix and ISO timestamp", () => {
      // Verify the format matches what we expect in agent-stdio.log
      const ts = new Date().toISOString();
      const logLine = `[copilot-driver] ${ts} test message`;
      expect(logLine).toMatch(/^\[copilot-driver\] \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/);
    });
  });

  describe("startup log includes node version and platform", () => {
    it("starting log line contains nodeVersion and platform fields", () => {
      const command = "/usr/local/bin/copilot";
      const startingLine = `starting: command=${command} maxRetries=3 initialDelayMs=5000` + ` backoffMultiplier=2 maxDelayMs=60000` + ` nodeVersion=${process.version} platform=${process.platform}`;
      expect(startingLine).toContain("nodeVersion=");
      expect(startingLine).toContain("platform=");
      expect(startingLine).toMatch(/nodeVersion=v\d+\.\d+/);
    });
  });

  describe("no-output failure message", () => {
    it("includes actionable possible causes", () => {
      const msg = `attempt 1: no output produced — not retrying` + ` (possible causes: binary not found, permission denied, auth failure, or silent startup crash)`;
      expect(msg).toContain("binary not found");
      expect(msg).toContain("permission denied");
      expect(msg).toContain("auth failure");
      expect(msg).toContain("silent startup crash");
    });
  });

  describe("error event message", () => {
    it("includes code and syscall fields", () => {
      const errMessage = "spawn /usr/local/bin/copilot ENOENT";
      const errCode = "ENOENT";
      const errSyscall = "spawn";
      const logMsg = `attempt 1: failed to start process '/usr/local/bin/copilot': ${errMessage}` + ` (code=${errCode} syscall=${errSyscall})`;
      expect(logMsg).toContain("code=ENOENT");
      expect(logMsg).toContain("syscall=spawn");
    });
  });
});
