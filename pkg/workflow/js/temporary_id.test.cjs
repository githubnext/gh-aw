import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock core for loadTemporaryIdMap
const mockCore = {
  warning: vi.fn(),
};
global.core = mockCore;

describe("temporary_id.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_TEMPORARY_ID_MAP;
  });

  describe("generateTemporaryId", () => {
    it("should generate a 12-character hex string", async () => {
      const { generateTemporaryId } = await import("./temporary_id.cjs");
      const id = generateTemporaryId();
      expect(id).toMatch(/^[0-9a-f]{12}$/);
    });

    it("should generate unique IDs", async () => {
      const { generateTemporaryId } = await import("./temporary_id.cjs");
      const ids = new Set();
      for (let i = 0; i < 100; i++) {
        ids.add(generateTemporaryId());
      }
      expect(ids.size).toBe(100);
    });
  });

  describe("isTemporaryId", () => {
    it("should return true for valid 12-char hex strings", async () => {
      const { isTemporaryId } = await import("./temporary_id.cjs");
      expect(isTemporaryId("abc123def456")).toBe(true);
      expect(isTemporaryId("000000000000")).toBe(true);
      expect(isTemporaryId("AABBCCDD1122")).toBe(true);
      expect(isTemporaryId("aAbBcCdDeEfF")).toBe(true);
    });

    it("should return false for invalid strings", async () => {
      const { isTemporaryId } = await import("./temporary_id.cjs");
      expect(isTemporaryId("abc123")).toBe(false); // Too short
      expect(isTemporaryId("abc123def4567")).toBe(false); // Too long
      expect(isTemporaryId("parent123456")).toBe(false); // Contains non-hex chars
      expect(isTemporaryId("ghijklmnopqr")).toBe(false); // Non-hex letters
      expect(isTemporaryId("")).toBe(false);
    });

    it("should return false for non-string values", async () => {
      const { isTemporaryId } = await import("./temporary_id.cjs");
      expect(isTemporaryId(123)).toBe(false);
      expect(isTemporaryId(null)).toBe(false);
      expect(isTemporaryId(undefined)).toBe(false);
      expect(isTemporaryId({})).toBe(false);
    });
  });

  describe("normalizeTemporaryId", () => {
    it("should convert to lowercase", async () => {
      const { normalizeTemporaryId } = await import("./temporary_id.cjs");
      expect(normalizeTemporaryId("ABC123DEF456")).toBe("abc123def456");
      expect(normalizeTemporaryId("aAbBcCdDeEfF")).toBe("aabbccddeeff");
    });
  });

  describe("replaceTemporaryIdReferences", () => {
    it("should replace #temp:ID with issue numbers", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map([["abc123def456", 100]]);
      const text = "Check #temp:abc123def456 for details";
      expect(replaceTemporaryIdReferences(text, map)).toBe("Check #100 for details");
    });

    it("should handle multiple references", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map([
        ["abc123def456", 100],
        ["111222333444", 200],
      ]);
      const text = "See #temp:abc123def456 and #temp:111222333444";
      expect(replaceTemporaryIdReferences(text, map)).toBe("See #100 and #200");
    });

    it("should preserve unresolved references", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map();
      const text = "Check #temp:000000000000 for details";
      expect(replaceTemporaryIdReferences(text, map)).toBe("Check #temp:000000000000 for details");
    });

    it("should be case-insensitive", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map([["abc123def456", 100]]);
      const text = "Check #temp:ABC123DEF456 for details";
      expect(replaceTemporaryIdReferences(text, map)).toBe("Check #100 for details");
    });

    it("should not match invalid temporary ID formats", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map([["abc123def456", 100]]);
      const text = "Check #temp:abc123 and #temp:parent123456 for details";
      expect(replaceTemporaryIdReferences(text, map)).toBe("Check #temp:abc123 and #temp:parent123456 for details");
    });
  });

  describe("loadTemporaryIdMap", () => {
    it("should return empty map when env var is not set", async () => {
      const { loadTemporaryIdMap } = await import("./temporary_id.cjs");
      const map = loadTemporaryIdMap();
      expect(map.size).toBe(0);
    });

    it("should return empty map when env var is empty object", async () => {
      process.env.GH_AW_TEMPORARY_ID_MAP = "{}";
      const { loadTemporaryIdMap } = await import("./temporary_id.cjs");
      const map = loadTemporaryIdMap();
      expect(map.size).toBe(0);
    });

    it("should parse valid JSON map", async () => {
      process.env.GH_AW_TEMPORARY_ID_MAP = JSON.stringify({ abc123def456: 100, 111222333444: 200 });
      const { loadTemporaryIdMap } = await import("./temporary_id.cjs");
      const map = loadTemporaryIdMap();
      expect(map.size).toBe(2);
      expect(map.get("abc123def456")).toBe(100);
      expect(map.get("111222333444")).toBe(200);
    });

    it("should normalize keys to lowercase", async () => {
      process.env.GH_AW_TEMPORARY_ID_MAP = JSON.stringify({ ABC123DEF456: 100 });
      const { loadTemporaryIdMap } = await import("./temporary_id.cjs");
      const map = loadTemporaryIdMap();
      expect(map.get("abc123def456")).toBe(100);
    });

    it("should warn and return empty map on invalid JSON", async () => {
      process.env.GH_AW_TEMPORARY_ID_MAP = "not valid json";
      const { loadTemporaryIdMap } = await import("./temporary_id.cjs");
      const map = loadTemporaryIdMap();
      expect(map.size).toBe(0);
      expect(mockCore.warning).toHaveBeenCalled();
    });
  });
});
