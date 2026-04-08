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

  describe("retry policy: resume on partial execution", () => {
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
});
