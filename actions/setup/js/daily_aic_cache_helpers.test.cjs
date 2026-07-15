// @ts-check
import { beforeEach, describe, expect, it, vi } from "vitest";

let exports;

describe("daily_aic_cache_helpers", () => {
  beforeEach(async () => {
    vi.resetModules();
    const mod = await import("./daily_aic_cache_helpers.cjs");
    exports = mod.default || mod;
  });

  describe("AIC_USAGE_CACHE_FILE_PATH", () => {
    it("exports a non-empty string path", () => {
      expect(typeof exports.AIC_USAGE_CACHE_FILE_PATH).toBe("string");
      expect(exports.AIC_USAGE_CACHE_FILE_PATH.length).toBeGreaterThan(0);
    });
  });

  describe("CACHE_RETENTION_MS", () => {
    it("exports a positive number equal to 48 hours in milliseconds", () => {
      expect(typeof exports.CACHE_RETENTION_MS).toBe("number");
      expect(exports.CACHE_RETENTION_MS).toBe(48 * 60 * 60 * 1000);
    });
  });

  describe("pruneStaleJSONLCacheLines", () => {
    it("returns empty result for empty content", () => {
      const result = exports.pruneStaleJSONLCacheLines("", Date.now());
      expect(result.keptLines).toEqual([]);
      expect(result.prunedCount).toBe(0);
      expect(result.totalCount).toBe(0);
    });

    it("returns empty result for whitespace-only content", () => {
      const result = exports.pruneStaleJSONLCacheLines("   \n\n  \n", Date.now());
      expect(result.keptLines).toEqual([]);
      expect(result.prunedCount).toBe(0);
      expect(result.totalCount).toBe(0);
    });

    it("keeps entries with a recent timestamp", () => {
      const recentTs = new Date(Date.now() - 60 * 60 * 1000).toISOString(); // 1 hour ago
      const line = JSON.stringify({ run_id: 1, aic: 5, timestamp: recentTs });
      const cutoff = Date.now() - 48 * 60 * 60 * 1000;
      const result = exports.pruneStaleJSONLCacheLines(line + "\n", cutoff);
      expect(result.keptLines).toHaveLength(1);
      expect(result.keptLines[0]).toBe(line);
      expect(result.prunedCount).toBe(0);
      expect(result.totalCount).toBe(1);
    });

    it("prunes entries with a stale timestamp", () => {
      const staleTs = new Date(Date.now() - 50 * 60 * 60 * 1000).toISOString(); // 50 hours ago
      const line = JSON.stringify({ run_id: 2, aic: 3, timestamp: staleTs });
      const cutoff = Date.now() - 48 * 60 * 60 * 1000;
      const result = exports.pruneStaleJSONLCacheLines(line + "\n", cutoff);
      expect(result.keptLines).toHaveLength(0);
      expect(result.prunedCount).toBe(1);
      expect(result.totalCount).toBe(1);
    });

    it("keeps entries that have no timestamp (backward compatibility)", () => {
      const line = JSON.stringify({ run_id: 3, aic: 7 });
      const cutoff = Date.now();
      const result = exports.pruneStaleJSONLCacheLines(line + "\n", cutoff);
      expect(result.keptLines).toHaveLength(1);
      expect(result.keptLines[0]).toBe(line);
      expect(result.prunedCount).toBe(0);
    });

    it("drops non-object lines and keeps malformed object-like lines", () => {
      const content = "not-valid-json\n{bad json\n";
      const result = exports.pruneStaleJSONLCacheLines(content, Date.now());
      expect(result.keptLines).toHaveLength(1);
      expect(result.keptLines).not.toContain("not-valid-json");
      expect(result.keptLines).toContain("{bad json");
      expect(result.prunedCount).toBe(1);
      expect(result.totalCount).toBe(2);
    });

    it("discards empty lines", () => {
      const recentTs = new Date(Date.now() - 60 * 1000).toISOString();
      const content = "\n" + JSON.stringify({ run_id: 10, aic: 1, timestamp: recentTs }) + "\n\n";
      const cutoff = Date.now() - 48 * 60 * 60 * 1000;
      const result = exports.pruneStaleJSONLCacheLines(content, cutoff);
      expect(result.keptLines).toHaveLength(1);
      expect(result.totalCount).toBe(1);
    });

    it("handles a mix of recent, stale, and timestamp-less entries", () => {
      const recentTs = new Date(Date.now() - 60 * 60 * 1000).toISOString(); // 1 h ago
      const staleTs = new Date(Date.now() - 50 * 60 * 60 * 1000).toISOString(); // 50 h ago
      const lines = [
        JSON.stringify({ run_id: 1, aic: 1, timestamp: recentTs }),
        JSON.stringify({ run_id: 2, aic: 2, timestamp: staleTs }),
        JSON.stringify({ run_id: 3, aic: 3 }), // no timestamp
        "malformed",
      ];
      const content = lines.join("\n") + "\n";
      const cutoff = Date.now() - 48 * 60 * 60 * 1000;
      const result = exports.pruneStaleJSONLCacheLines(content, cutoff);

      expect(result.totalCount).toBe(4);
      expect(result.prunedCount).toBe(2);
      expect(result.keptLines).toHaveLength(2);

      const keptRunIds = result.keptLines.filter(l => l.startsWith("{")).map(l => JSON.parse(l).run_id);
      expect(keptRunIds).toContain(1);
      expect(keptRunIds).not.toContain(2);
      expect(keptRunIds).toContain(3);
      expect(result.keptLines).not.toContain("malformed");
    });

    it("keeps an entry whose timestamp exactly equals cutoffMs (boundary: not pruned)", () => {
      const cutoff = Date.now() - 48 * 60 * 60 * 1000;
      const exactTs = new Date(cutoff).toISOString();
      const line = JSON.stringify({ run_id: 5, aic: 1, timestamp: exactTs });
      // ts === cutoffMs should be kept (condition is ts < cutoffMs)
      const result = exports.pruneStaleJSONLCacheLines(line + "\n", cutoff);
      expect(result.keptLines).toHaveLength(1);
      expect(result.prunedCount).toBe(0);
    });
  });
});
