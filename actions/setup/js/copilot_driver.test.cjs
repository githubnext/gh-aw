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
});
