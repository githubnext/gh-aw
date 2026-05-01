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

  describe("LOW_RATE_LIMIT_THRESHOLD_PERCENT", () => {
    it("should be 20", async () => {
      const { LOW_RATE_LIMIT_THRESHOLD_PERCENT } = await import("./rate_limit_helpers.cjs");
      expect(LOW_RATE_LIMIT_THRESHOLD_PERCENT).toBe(20);
    });
  });

  describe("checkRateLimitHeadroom", () => {
    it("should return remaining, limit, and percentRemaining", async () => {
      const { checkRateLimitHeadroom } = await import("./rate_limit_helpers.cjs");
      mockGithub.rest.rateLimit.get.mockResolvedValue({
        data: {
          rate: { remaining: 4000, limit: 5000, used: 1000 },
          resources: {},
        },
      });
      const result = await checkRateLimitHeadroom(mockGithub, "test");
      expect(result.remaining).toBe(4000);
      expect(result.limit).toBe(5000);
      expect(result.percentRemaining).toBe(80);
    });

    it("should log info when headroom is above threshold", async () => {
      const { checkRateLimitHeadroom } = await import("./rate_limit_helpers.cjs");
      mockGithub.rest.rateLimit.get.mockResolvedValue({
        data: {
          rate: { remaining: 4000, limit: 5000, used: 1000 },
          resources: {},
        },
      });
      await checkRateLimitHeadroom(mockGithub, "test");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Rate-limit headroom: 4000/5000"));
      expect(mockCore.warning).not.toHaveBeenCalled();
    });

    it("should emit warning when headroom is below threshold", async () => {
      const { checkRateLimitHeadroom } = await import("./rate_limit_helpers.cjs");
      mockGithub.rest.rateLimit.get.mockResolvedValue({
        data: {
          rate: { remaining: 500, limit: 5000, used: 4500 },
          resources: {},
        },
      });
      await checkRateLimitHeadroom(mockGithub, "test");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Rate-limit headroom low: 500/5000"));
    });

    it("should return -1 values and warn with error details on error", async () => {
      const { checkRateLimitHeadroom } = await import("./rate_limit_helpers.cjs");
      mockGithub.rest.rateLimit.get.mockRejectedValue(new Error("API error"));
      const result = await checkRateLimitHeadroom(mockGithub, "test");
      expect(result.remaining).toBe(-1);
      expect(result.limit).toBe(-1);
      expect(result.percentRemaining).toBe(-1);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not check rate-limit headroom"));
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("API error"));
    });

    it("should use Math.floor so 19.9% triggers warning (not Math.round which would give 20%)", async () => {
      const { checkRateLimitHeadroom } = await import("./rate_limit_helpers.cjs");
      // 199/1000 = 19.9% — Math.floor gives 19, Math.round would give 20 (and skip the warning)
      mockGithub.rest.rateLimit.get.mockResolvedValue({
        data: {
          rate: { remaining: 199, limit: 1000, used: 801 },
          resources: {},
        },
      });
      await checkRateLimitHeadroom(mockGithub, "test");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Rate-limit headroom low:"));
    });
  });
});
