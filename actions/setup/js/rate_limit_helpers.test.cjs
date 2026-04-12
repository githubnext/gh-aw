// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock core global (needed by github_rate_limit_logger.cjs)
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

global.core = mockCore;

describe("rate_limit_helpers", () => {
  let mockGithub;

  beforeEach(() => {
    vi.clearAllMocks();
    mockGithub = {
      rest: {
        rateLimit: {
          get: vi.fn().mockResolvedValue({
            data: {
              rate: { remaining: 5000, limit: 5000, used: 0 },
              resources: {},
            },
          }),
        },
      },
    };
  });

  describe("getRateLimitRemaining", () => {
    it("should return remaining rate limit", async () => {
      const { getRateLimitRemaining } = await import("./rate_limit_helpers.cjs");
      const remaining = await getRateLimitRemaining(mockGithub, "test");
      expect(remaining).toBe(5000);
    });

    it("should return -1 on error", async () => {
      const { getRateLimitRemaining } = await import("./rate_limit_helpers.cjs");
      mockGithub.rest.rateLimit.get.mockRejectedValueOnce(new Error("API error")).mockRejectedValueOnce(new Error("API error"));
      const remaining = await getRateLimitRemaining(mockGithub, "test");
      expect(remaining).toBe(-1);
    });
  });

  describe("checkRateLimit", () => {
    it("should return ok when rate limit is sufficient", async () => {
      const { checkRateLimit } = await import("./rate_limit_helpers.cjs");
      const result = await checkRateLimit(mockGithub, "test");
      expect(result.ok).toBe(true);
      expect(result.remaining).toBe(5000);
    });

    it("should return not ok when rate limit is too low", async () => {
      const { checkRateLimit } = await import("./rate_limit_helpers.cjs");
      mockGithub.rest.rateLimit.get.mockResolvedValue({
        data: {
          rate: { remaining: 50, limit: 5000, used: 4950 },
          resources: {},
        },
      });
      const result = await checkRateLimit(mockGithub, "test");
      expect(result.ok).toBe(false);
      expect(result.remaining).toBe(50);
    });

    it("should return ok when rate limit check fails", async () => {
      const { checkRateLimit } = await import("./rate_limit_helpers.cjs");
      mockGithub.rest.rateLimit.get.mockRejectedValue(new Error("API error"));
      const result = await checkRateLimit(mockGithub, "test");
      expect(result.ok).toBe(true);
      expect(result.remaining).toBe(-1);
    });
  });

  describe("MIN_RATE_LIMIT_REMAINING", () => {
    it("should be 100", async () => {
      const { MIN_RATE_LIMIT_REMAINING } = await import("./rate_limit_helpers.cjs");
      expect(MIN_RATE_LIMIT_REMAINING).toBe(100);
    });
  });
});
