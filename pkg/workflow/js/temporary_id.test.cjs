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
    it("should generate an aw_ prefixed 12-character hex string", async () => {
      const { generateTemporaryId } = await import("./temporary_id.cjs");
      const id = generateTemporaryId();
      expect(id).toMatch(/^aw_[0-9a-f]{12}$/);
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
    it("should return true for valid aw_ prefixed 12-char hex strings", async () => {
      const { isTemporaryId } = await import("./temporary_id.cjs");
      expect(isTemporaryId("aw_abc123def456")).toBe(true);
      expect(isTemporaryId("aw_000000000000")).toBe(true);
      expect(isTemporaryId("aw_AABBCCDD1122")).toBe(true);
      expect(isTemporaryId("aw_aAbBcCdDeEfF")).toBe(true);
    });

    it("should return false for invalid strings", async () => {
      const { isTemporaryId } = await import("./temporary_id.cjs");
      expect(isTemporaryId("abc123def456")).toBe(false); // Missing aw_ prefix
      expect(isTemporaryId("aw_abc123")).toBe(false); // Too short
      expect(isTemporaryId("aw_abc123def4567")).toBe(false); // Too long
      expect(isTemporaryId("aw_parent123456")).toBe(false); // Contains non-hex chars
      expect(isTemporaryId("aw_ghijklmnopqr")).toBe(false); // Non-hex letters
      expect(isTemporaryId("")).toBe(false);
      expect(isTemporaryId("temp_abc123def456")).toBe(false); // Wrong prefix
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
      expect(normalizeTemporaryId("aw_ABC123DEF456")).toBe("aw_abc123def456");
      expect(normalizeTemporaryId("AW_aAbBcCdDeEfF")).toBe("aw_aabbccddeeff");
    });
  });

  describe("replaceTemporaryIdReferences", () => {
    it("should replace #aw_ID with issue numbers", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map([["aw_abc123def456", 100]]);
      const text = "Check #aw_abc123def456 for details";
      expect(replaceTemporaryIdReferences(text, map)).toBe("Check #100 for details");
    });

    it("should handle multiple references", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map([
        ["aw_abc123def456", 100],
        ["aw_111222333444", 200],
      ]);
      const text = "See #aw_abc123def456 and #aw_111222333444";
      expect(replaceTemporaryIdReferences(text, map)).toBe("See #100 and #200");
    });

    it("should preserve unresolved references", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map();
      const text = "Check #aw_000000000000 for details";
      expect(replaceTemporaryIdReferences(text, map)).toBe("Check #aw_000000000000 for details");
    });

    it("should be case-insensitive", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map([["aw_abc123def456", 100]]);
      const text = "Check #AW_ABC123DEF456 for details";
      expect(replaceTemporaryIdReferences(text, map)).toBe("Check #100 for details");
    });

    it("should not match invalid temporary ID formats", async () => {
      const { replaceTemporaryIdReferences } = await import("./temporary_id.cjs");
      const map = new Map([["aw_abc123def456", 100]]);
      const text = "Check #aw_abc123 and #temp:abc123def456 for details";
      expect(replaceTemporaryIdReferences(text, map)).toBe("Check #aw_abc123 and #temp:abc123def456 for details");
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
      process.env.GH_AW_TEMPORARY_ID_MAP = JSON.stringify({ aw_abc123def456: 100, aw_111222333444: 200 });
      const { loadTemporaryIdMap } = await import("./temporary_id.cjs");
      const map = loadTemporaryIdMap();
      expect(map.size).toBe(2);
      expect(map.get("aw_abc123def456")).toBe(100);
      expect(map.get("aw_111222333444")).toBe(200);
    });

    it("should normalize keys to lowercase", async () => {
      process.env.GH_AW_TEMPORARY_ID_MAP = JSON.stringify({ AW_ABC123DEF456: 100 });
      const { loadTemporaryIdMap } = await import("./temporary_id.cjs");
      const map = loadTemporaryIdMap();
      expect(map.get("aw_abc123def456")).toBe(100);
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
