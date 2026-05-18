// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);

describe("expired_entity_cleanup_helpers", () => {
  /** @type {Record<string, Function>} */
  let mockCore;
  /** @type {Record<string, unknown>} */
  let originalGlobals;

  beforeEach(() => {
    originalGlobals = { core: global.core };
    mockCore = {
      debug: vi.fn(),
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
    };
    global.core = mockCore;
  });

  afterEach(() => {
    global.core = originalGlobals.core;
    vi.clearAllMocks();
  });

  const { delay, validateCreationDate, categorizeByExpiration, processExpiredEntities, buildExpirationSummary, DEFAULT_MAX_UPDATES_PER_RUN, DEFAULT_GRAPHQL_DELAY_MS } = req("./expired_entity_cleanup_helpers.cjs");

  /** Build an expiration marker body in the expected gh-aw-expires format */
  const makeExpirationBody = date => `> - [x] expires <!-- gh-aw-expires: ${date.toISOString()} --> on ${date.toUTCString()} UTC`;

  describe("delay", () => {
    it("resolves after the specified time", async () => {
      vi.useFakeTimers();
      try {
        const promise = delay(10);
        await vi.advanceTimersByTimeAsync(9);

        let resolved = false;
        promise.then(() => {
          resolved = true;
        });
        await Promise.resolve();
        expect(resolved).toBe(false);

        await vi.advanceTimersByTimeAsync(1);
        await expect(promise).resolves.toBeUndefined();
      } finally {
        vi.useRealTimers();
      }
    });

    it("resolves immediately for 0 ms", async () => {
      await expect(delay(0)).resolves.toBeUndefined();
    });
  });

  describe("validateCreationDate", () => {
    it("returns true for a valid ISO 8601 date", () => {
      expect(validateCreationDate("2024-01-15T10:00:00Z")).toBe(true);
    });

    it("returns true for a valid date-only string", () => {
      expect(validateCreationDate("2024-01-15")).toBe(true);
    });

    it("returns false for an invalid date string", () => {
      expect(validateCreationDate("not-a-date")).toBe(false);
    });

    it("returns false for an empty string", () => {
      expect(validateCreationDate("")).toBe(false);
    });

    it("returns false for a purely numeric string", () => {
      expect(validateCreationDate("abc123")).toBe(false);
    });
  });

  describe("categorizeByExpiration", () => {
    const makeEntity = (number, body, createdAt = "2024-01-01T00:00:00Z") => ({
      number,
      title: `Issue ${number}`,
      url: `https://github.com/owner/repo/issues/${number}`,
      body,
      createdAt,
    });

    it("puts expired entities in the expired array", () => {
      const pastDate = new Date(Date.now() - 1000 * 60 * 60 * 24 * 2); // 2 days ago
      const entity = makeEntity(1, makeExpirationBody(pastDate));
      const { expired, notExpired } = categorizeByExpiration([entity], { entityLabel: "Issue" });
      expect(expired).toHaveLength(1);
      expect(notExpired).toHaveLength(0);
    });

    it("puts non-expired entities in the notExpired array", () => {
      const futureDate = new Date(Date.now() + 1000 * 60 * 60 * 24 * 7); // 7 days from now
      const entity = makeEntity(2, makeExpirationBody(futureDate));
      const { expired, notExpired } = categorizeByExpiration([entity], { entityLabel: "Issue" });
      expect(expired).toHaveLength(0);
      expect(notExpired).toHaveLength(1);
    });

    it("skips entities with invalid creation dates and logs a warning", () => {
      const futureDate = new Date(Date.now() + 1000 * 60 * 60 * 24);
      const entity = makeEntity(3, makeExpirationBody(futureDate), "invalid-date");
      const { expired, notExpired } = categorizeByExpiration([entity], { entityLabel: "Issue" });
      expect(expired).toHaveLength(0);
      expect(notExpired).toHaveLength(0);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("invalid creation date"));
    });

    it("skips entities without expiration markers and logs a warning", () => {
      const entity = makeEntity(4, "No expiration marker here");
      const { expired, notExpired } = categorizeByExpiration([entity], { entityLabel: "Issue" });
      expect(expired).toHaveLength(0);
      expect(notExpired).toHaveLength(0);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("invalid expiration date format"));
    });

    it("returns a now Date object", () => {
      const before = Date.now();
      const { now } = categorizeByExpiration([], { entityLabel: "Issue" });
      expect(now.getTime()).toBeGreaterThanOrEqual(before);
    });

    it("attaches the expirationDate to each categorized entity", () => {
      const pastDate = new Date(Date.now() - 1000 * 60 * 60 * 24);
      const entity = makeEntity(5, makeExpirationBody(pastDate));
      const { expired } = categorizeByExpiration([entity], { entityLabel: "Issue" });
      expect(expired[0].expirationDate).toBeInstanceOf(Date);
    });
  });

  describe("processExpiredEntities", () => {
    const makeExpiredEntity = number => ({
      number,
      title: `Issue ${number}`,
      url: `https://github.com/owner/repo/issues/${number}`,
      expirationDate: new Date(Date.now() - 1000),
    });

    it("processes all entities and returns closed records", async () => {
      const entities = [makeExpiredEntity(1), makeExpiredEntity(2)];
      const processEntity = vi.fn().mockResolvedValue({ status: "closed", record: { number: 1 } });
      const { closed, skipped, failed } = await processExpiredEntities(entities, {
        entityLabel: "Issue",
        delayMs: 0,
        processEntity,
      });
      expect(closed).toHaveLength(2);
      expect(skipped).toHaveLength(0);
      expect(failed).toHaveLength(0);
    });

    it("tracks skipped entities separately", async () => {
      const entities = [makeExpiredEntity(1)];
      const processEntity = vi.fn().mockResolvedValue({ status: "skipped", record: { number: 1 } });
      const { closed, skipped } = await processExpiredEntities(entities, {
        entityLabel: "Issue",
        delayMs: 0,
        processEntity,
      });
      expect(closed).toHaveLength(0);
      expect(skipped).toHaveLength(1);
    });

    it("tracks failed entities when processEntity throws", async () => {
      const entities = [makeExpiredEntity(1)];
      const processEntity = vi.fn().mockRejectedValue(new Error("API error"));
      const { failed } = await processExpiredEntities(entities, {
        entityLabel: "Issue",
        delayMs: 0,
        processEntity,
      });
      expect(failed).toHaveLength(1);
      expect(failed[0].error).toBe("API error");
    });

    it("respects the maxPerRun limit", async () => {
      const entities = [makeExpiredEntity(1), makeExpiredEntity(2), makeExpiredEntity(3)];
      const processEntity = vi.fn().mockResolvedValue({ status: "closed", record: {} });
      await processExpiredEntities(entities, {
        entityLabel: "Issue",
        maxPerRun: 2,
        delayMs: 0,
        processEntity,
      });
      expect(processEntity).toHaveBeenCalledTimes(2);
    });

    it("returns empty arrays when input is empty", async () => {
      const processEntity = vi.fn();
      const { closed, skipped, failed } = await processExpiredEntities([], {
        entityLabel: "Issue",
        delayMs: 0,
        processEntity,
      });
      expect(closed).toHaveLength(0);
      expect(skipped).toHaveLength(0);
      expect(failed).toHaveLength(0);
      expect(processEntity).not.toHaveBeenCalled();
    });
  });

  describe("buildExpirationSummary", () => {
    const baseParams = {
      heading: "Cleanup Summary",
      entityLabel: "Issue",
      searchStats: { totalScanned: 50, pageCount: 2 },
      withExpirationCount: 10,
      expired: [{ number: 1, title: "Old Issue", url: "https://github.com/owner/repo/issues/1" }],
      notExpired: [],
      closed: [{ number: 1, title: "Old Issue", url: "https://github.com/owner/repo/issues/1" }],
      failed: [],
      maxPerRun: 100,
    };

    it("includes the heading and scan summary", () => {
      const result = buildExpirationSummary(baseParams);
      expect(result).toContain("## Cleanup Summary");
      expect(result).toContain("Scanned: 50");
      expect(result).toContain("With expiration markers: 10");
    });

    it("lists successfully closed entities", () => {
      const result = buildExpirationSummary(baseParams);
      expect(result).toContain("Successfully Closed Issues");
      expect(result).toContain("Old Issue");
    });

    it("shows failed entities when present", () => {
      const params = {
        ...baseParams,
        failed: [{ number: 2, title: "Failed Issue", url: "https://example.com/2", error: "timeout" }],
      };
      const result = buildExpirationSummary(params);
      expect(result).toContain("Failed to Close");
      expect(result).toContain("timeout");
    });

    it("shows remaining count when expired exceeds maxPerRun", () => {
      const params = {
        ...baseParams,
        expired: Array.from({ length: 5 }, (_, i) => ({ number: i + 1, title: `Issue ${i + 1}`, url: "" })),
        closed: [],
        maxPerRun: 2,
      };
      const result = buildExpirationSummary(params);
      expect(result).toContain("Remaining for next run: 3");
    });

    it("shows skipped section when includeSkippedHeading is true and skipped exist", () => {
      const params = {
        ...baseParams,
        skipped: [{ number: 3, title: "Skipped Issue", url: "https://example.com/3" }],
        includeSkippedHeading: true,
      };
      const result = buildExpirationSummary(params);
      expect(result).toContain("Skipped (Already Had Comment)");
      expect(result).toContain("Skipped Issue");
    });

    it("omits skipped section when includeSkippedHeading is false", () => {
      const params = {
        ...baseParams,
        skipped: [{ number: 3, title: "Skipped Issue", url: "https://example.com/3" }],
        includeSkippedHeading: false,
      };
      const result = buildExpirationSummary(params);
      expect(result).not.toContain("Skipped (Already Had Comment)");
    });

    it("caps notExpired list at 10 and shows 'showing first 10' text", () => {
      const fixedNow = new Date("2024-06-01T00:00:00Z");
      const futureDate = new Date("2024-07-01T00:00:00Z");
      const notExpired = Array.from({ length: 12 }, (_, i) => ({
        number: i + 10,
        title: `Not Expired ${i + 10}`,
        url: `https://github.com/owner/repo/issues/${i + 10}`,
        expirationDate: futureDate,
      }));
      const params = { ...baseParams, notExpired, expired: [], closed: [], now: fixedNow };
      const result = buildExpirationSummary(params);
      expect(result).toContain("showing first 10");
      // Entries 10–19 should appear; entry 21 (index 11) must not
      expect(result).toContain("Not Expired 10");
      expect(result).toContain("Not Expired 19");
      expect(result).not.toContain("Not Expired 21");
    });
  });

  describe("constants", () => {
    it("exports DEFAULT_MAX_UPDATES_PER_RUN as 100", () => {
      expect(DEFAULT_MAX_UPDATES_PER_RUN).toBe(100);
    });

    it("exports DEFAULT_GRAPHQL_DELAY_MS as 500", () => {
      expect(DEFAULT_GRAPHQL_DELAY_MS).toBe(500);
    });
  });
});
