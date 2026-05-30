// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock core and context globals
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn(),
  },
  setOutput: vi.fn(),
};

const mockContext = {
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
};

global.core = mockCore;
global.context = mockContext;

describe("cleanup_cache_memory", () => {
  let mockGithub;

  beforeEach(() => {
    vi.clearAllMocks();
    mockGithub = {
      rest: {
        actions: {
          getActionsCacheList: vi.fn(),
          deleteActionsCacheById: vi.fn(),
        },
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
    global.github = mockGithub;
  });

  describe("parseCacheKey", () => {
    it("should extract run ID and group key from standard cache key", async () => {
      const { parseCacheKey } = await import("./cleanup_cache_memory.cjs");
      const result = parseCacheKey("memory-none-nopolicy-workflow-12345");
      expect(result.runId).toBe(12345);
      expect(result.groupKey).toBe("memory-none-nopolicy-workflow");
    });

    it("should extract run ID and group key from integrity-aware cache key", async () => {
      const { parseCacheKey } = await import("./cleanup_cache_memory.cjs");
      const result = parseCacheKey("memory-unapproved-7e4d9f12-session-workflow-67890");
      expect(result.runId).toBe(67890);
      expect(result.groupKey).toBe("memory-unapproved-7e4d9f12-session-workflow");
    });

    it("should return null runId when no numeric segment exists", async () => {
      const { parseCacheKey } = await import("./cleanup_cache_memory.cjs");
      const result = parseCacheKey("memory-none-nopolicy-workflow");
      expect(result.runId).toBeNull();
      expect(result.groupKey).toBe("memory-none-nopolicy-workflow");
    });

    it("should handle cache key with only run ID as numeric part", async () => {
      const { parseCacheKey } = await import("./cleanup_cache_memory.cjs");
      const result = parseCacheKey("memory-abc-def-99999");
      expect(result.runId).toBe(99999);
      expect(result.groupKey).toBe("memory-abc-def");
    });
  });

  describe("identifyCachesToDelete", () => {
    it("should keep latest run ID and mark older ones for deletion", async () => {
      const { identifyCachesToDelete } = await import("./cleanup_cache_memory.cjs");

      const caches = [
        { id: 1, key: "memory-none-nopolicy-workflow-100", runId: 100, groupKey: "memory-none-nopolicy-workflow" },
        { id: 2, key: "memory-none-nopolicy-workflow-200", runId: 200, groupKey: "memory-none-nopolicy-workflow" },
        { id: 3, key: "memory-none-nopolicy-workflow-150", runId: 150, groupKey: "memory-none-nopolicy-workflow" },
      ];

      const { toDelete, kept } = identifyCachesToDelete(caches);

      expect(kept).toHaveLength(1);
      expect(kept[0].runId).toBe(200);
      expect(toDelete).toHaveLength(2);
      expect(toDelete.map(c => c.runId).sort((a, b) => a - b)).toEqual([100, 150]);
    });

    it("should handle multiple groups independently", async () => {
      const { identifyCachesToDelete } = await import("./cleanup_cache_memory.cjs");

      const caches = [
        { id: 1, key: "memory-none-nopolicy-wf1-100", runId: 100, groupKey: "memory-none-nopolicy-wf1" },
        { id: 2, key: "memory-none-nopolicy-wf1-200", runId: 200, groupKey: "memory-none-nopolicy-wf1" },
        { id: 3, key: "memory-none-nopolicy-wf2-50", runId: 50, groupKey: "memory-none-nopolicy-wf2" },
        { id: 4, key: "memory-none-nopolicy-wf2-75", runId: 75, groupKey: "memory-none-nopolicy-wf2" },
      ];

      const { toDelete, kept } = identifyCachesToDelete(caches);

      expect(kept).toHaveLength(2);
      expect(kept.map(c => c.runId).sort((a, b) => a - b)).toEqual([75, 200]);
      expect(toDelete).toHaveLength(2);
      expect(toDelete.map(c => c.runId).sort((a, b) => a - b)).toEqual([50, 100]);
    });

    it("should not delete when only one entry per group", async () => {
      const { identifyCachesToDelete } = await import("./cleanup_cache_memory.cjs");

      const caches = [
        { id: 1, key: "memory-none-nopolicy-wf1-100", runId: 100, groupKey: "memory-none-nopolicy-wf1" },
        { id: 2, key: "memory-none-nopolicy-wf2-200", runId: 200, groupKey: "memory-none-nopolicy-wf2" },
      ];

      const { toDelete, kept } = identifyCachesToDelete(caches);

      expect(kept).toHaveLength(2);
      expect(toDelete).toHaveLength(0);
    });

    it("should skip entries with null run ID", async () => {
      const { identifyCachesToDelete } = await import("./cleanup_cache_memory.cjs");

      const caches = [
        { id: 1, key: "memory-none-nopolicy-workflow", runId: null, groupKey: "memory-none-nopolicy-workflow" },
        { id: 2, key: "memory-none-nopolicy-wf-100", runId: 100, groupKey: "memory-none-nopolicy-wf" },
      ];

      const { toDelete, kept } = identifyCachesToDelete(caches);

      expect(kept).toHaveLength(1);
      expect(kept[0].runId).toBe(100);
      expect(toDelete).toHaveLength(0);
    });

    it("should handle empty input", async () => {
      const { identifyCachesToDelete } = await import("./cleanup_cache_memory.cjs");

      const { toDelete, kept } = identifyCachesToDelete([]);

      expect(kept).toHaveLength(0);
      expect(toDelete).toHaveLength(0);
    });
  });

  describe("main - no caches found", () => {
    it("should handle case when no memory caches exist", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      mockGithub.rest.actions.getActionsCacheList.mockResolvedValueOnce({
        data: {
          total_count: 0,
          actions_caches: [],
        },
      });

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No memory caches found"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("No memory caches found"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });

  describe("main - rate limit too low", () => {
    it("should skip cleanup when rate limit is below threshold", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      mockGithub.rest.rateLimit.get.mockResolvedValue({
        data: {
          rate: { remaining: 50, limit: 5000, used: 4950 },
          resources: {},
        },
      });

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Rate limit too low"));
      expect(mockGithub.rest.actions.getActionsCacheList).not.toHaveBeenCalled();
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Skipped: Rate limit too low"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });

  describe("main - deletes outdated caches", () => {
    it("should delete older caches and keep latest per group", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      mockGithub.rest.actions.getActionsCacheList.mockResolvedValueOnce({
        data: {
          total_count: 3,
          actions_caches: [
            { id: 1, key: "memory-none-nopolicy-workflow-100" },
            { id: 2, key: "memory-none-nopolicy-workflow-200" },
            { id: 3, key: "memory-none-nopolicy-workflow-150" },
          ],
        },
      });

      mockGithub.rest.actions.deleteActionsCacheById.mockResolvedValue({});

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      // Should have deleted 2 caches (run IDs 100 and 150)
      expect(mockGithub.rest.actions.deleteActionsCacheById).toHaveBeenCalledTimes(2);
      expect(mockGithub.rest.actions.deleteActionsCacheById).toHaveBeenCalledWith(expect.objectContaining({ owner: "testowner", repo: "testrepo" }));

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Cache memory cleanup finished"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });

  describe("main - all caches are current", () => {
    it("should not delete when each group has only one entry", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      mockGithub.rest.actions.getActionsCacheList.mockResolvedValueOnce({
        data: {
          total_count: 2,
          actions_caches: [
            { id: 1, key: "memory-none-nopolicy-wf1-100" },
            { id: 2, key: "memory-none-nopolicy-wf2-200" },
          ],
        },
      });

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      expect(mockGithub.rest.actions.deleteActionsCacheById).not.toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No outdated caches to clean up"));
    });
  });

  describe("main - handles delete errors gracefully", () => {
    it("should continue deleting after individual failures", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      mockGithub.rest.actions.getActionsCacheList.mockResolvedValueOnce({
        data: {
          total_count: 3,
          actions_caches: [
            { id: 1, key: "memory-none-nopolicy-wf-100" },
            { id: 2, key: "memory-none-nopolicy-wf-200" },
            { id: 3, key: "memory-none-nopolicy-wf-300" },
          ],
        },
      });

      // First delete fails, second succeeds
      mockGithub.rest.actions.deleteActionsCacheById.mockRejectedValueOnce(new Error("API error")).mockResolvedValueOnce({});

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      // Should have attempted to delete 2 caches (IDs 100 and 200, keeping 300)
      expect(mockGithub.rest.actions.deleteActionsCacheById).toHaveBeenCalledTimes(2);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to delete cache"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Errors"));
    });
  });

  describe("main - handles list error", () => {
    it("should handle errors when listing caches", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      mockGithub.rest.actions.getActionsCacheList.mockRejectedValueOnce(new Error("API error"));

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to list caches"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });

  describe("main - pagination", () => {
    it("should handle paginated cache list results", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      // First page - full page of 100 items
      const page1Caches = [];
      for (let i = 0; i < 100; i++) {
        page1Caches.push({ id: i + 1, key: `memory-none-nopolicy-wf-${1000 + i}` });
      }

      // Second page - partial page
      const page2Caches = [{ id: 101, key: "memory-none-nopolicy-wf-2000" }];

      mockGithub.rest.actions.getActionsCacheList
        .mockResolvedValueOnce({
          data: { total_count: 101, actions_caches: page1Caches },
        })
        .mockResolvedValueOnce({
          data: { total_count: 101, actions_caches: page2Caches },
        });

      mockGithub.rest.actions.deleteActionsCacheById.mockResolvedValue({});

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      // Should have fetched 2 pages
      expect(mockGithub.rest.actions.getActionsCacheList).toHaveBeenCalledTimes(2);

      // Should delete 100 caches (keep only run 2000, delete 1000-1099)
      expect(mockGithub.rest.actions.deleteActionsCacheById).toHaveBeenCalledTimes(100);
    });
  });

  describe("main - rate limit stops early", () => {
    it("should stop deleting when rate limit drops below threshold", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      // Create 15 caches in same group (need >10 to trigger the rate limit check)
      const caches = [];
      for (let i = 0; i < 15; i++) {
        caches.push({ id: i + 1, key: `memory-none-nopolicy-wf-${100 + i}` });
      }

      mockGithub.rest.actions.getActionsCacheList.mockResolvedValueOnce({
        data: { total_count: 15, actions_caches: caches },
      });

      // Rate limit calls:
      // 1. fetchAndLogRateLimit at start of main → rateLimit.get
      // 2-3. checkRateLimit (initial) → fetchAndLogRateLimit + rateLimit.get
      // ... 10 deletions ...
      // 4-5. checkRateLimit (periodic) → fetchAndLogRateLimit + rateLimit.get
      // We want call 4 or 5 to return low rate limit to trigger early stop
      let callCount = 0;
      mockGithub.rest.rateLimit.get.mockImplementation(() => {
        callCount++;
        // Return low rate limit starting from the periodic check (call 4+)
        if (callCount >= 4) {
          return Promise.resolve({
            data: {
              rate: { remaining: 50, limit: 5000, used: 4950 },
              resources: {},
            },
          });
        }
        return Promise.resolve({
          data: {
            rate: { remaining: 5000, limit: 5000, used: 0 },
            resources: {},
          },
        });
      });

      mockGithub.rest.actions.deleteActionsCacheById.mockResolvedValue({});

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      // Should have stopped after 10 deletions (checked rate limit, it was low)
      expect(mockGithub.rest.actions.deleteActionsCacheById).toHaveBeenCalledTimes(10);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Rate limit getting low"));
    });
  });

  describe("main - logging", () => {
    it("should log kept entries with their keys", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      mockGithub.rest.actions.getActionsCacheList.mockResolvedValueOnce({
        data: {
          total_count: 2,
          actions_caches: [
            { id: 1, key: "memory-none-nopolicy-wf-100" },
            { id: 2, key: "memory-none-nopolicy-wf-200" },
          ],
        },
      });

      mockGithub.rest.actions.deleteActionsCacheById.mockResolvedValue({});

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Keeping: memory-none-nopolicy-wf-200"));
    });

    it("should log repository info", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      mockGithub.rest.actions.getActionsCacheList.mockResolvedValueOnce({
        data: { total_count: 0, actions_caches: [] },
      });

      await module.main({ deleteDelayMs: 0, listDelayMs: 0 });

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Repository: testowner/testrepo"));
    });
  });

  describe("listMemoryCaches - sort order", () => {
    it("should request caches sorted by last_accessed_at descending", async () => {
      const module = await import("./cleanup_cache_memory.cjs");

      mockGithub.rest.actions.getActionsCacheList.mockResolvedValueOnce({
        data: { total_count: 0, actions_caches: [] },
      });

      await module.listMemoryCaches(mockGithub, "testowner", "testrepo", 0);

      expect(mockGithub.rest.actions.getActionsCacheList).toHaveBeenCalledWith(
        expect.objectContaining({
          sort: "last_accessed_at",
          direction: "desc",
        })
      );
    });
  });

  describe("listMemoryCaches - upper bound", () => {
    it("should respect MAX_LIST_PAGES limit", async () => {
      const { listMemoryCaches, MAX_LIST_PAGES } = await import("./cleanup_cache_memory.cjs");

      // Return full pages forever
      mockGithub.rest.actions.getActionsCacheList.mockImplementation(({ page }) => {
        const caches = [];
        for (let i = 0; i < 100; i++) {
          caches.push({ id: page * 100 + i, key: `memory-none-nopolicy-wf-${page * 1000 + i}` });
        }
        return Promise.resolve({
          data: { total_count: 10000, actions_caches: caches },
        });
      });

      const result = await listMemoryCaches(mockGithub, "testowner", "testrepo", 0);

      // Should stop at MAX_LIST_PAGES
      expect(mockGithub.rest.actions.getActionsCacheList).toHaveBeenCalledTimes(MAX_LIST_PAGES);
      expect(result.length).toBe(MAX_LIST_PAGES * 100);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("maximum page limit"));
    });
  });
});
