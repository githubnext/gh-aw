// @ts-check
import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { parseDeduplicateByTitle, normalizeTitleForDedup, findDuplicateByTitle } = req("./issue_title_dedup.cjs");

describe("parseDeduplicateByTitle", () => {
  it("returns disabled for undefined", () => {
    expect(parseDeduplicateByTitle(undefined)).toEqual({ enabled: false, maxDistance: 0 });
  });

  it("returns disabled for null", () => {
    expect(parseDeduplicateByTitle(null)).toEqual({ enabled: false, maxDistance: 0 });
  });

  it("returns disabled for false", () => {
    expect(parseDeduplicateByTitle(false)).toEqual({ enabled: false, maxDistance: 0 });
  });

  it("returns disabled for string 'false'", () => {
    expect(parseDeduplicateByTitle("false")).toEqual({ enabled: false, maxDistance: 0 });
  });

  it("returns enabled with distance 0 for true", () => {
    expect(parseDeduplicateByTitle(true)).toEqual({ enabled: true, maxDistance: 0 });
  });

  it("returns enabled with distance 0 for string 'true'", () => {
    expect(parseDeduplicateByTitle("true")).toEqual({ enabled: true, maxDistance: 0 });
  });

  it("returns enabled with numeric distance for integer", () => {
    expect(parseDeduplicateByTitle(5)).toEqual({ enabled: true, maxDistance: 5 });
  });

  it("returns enabled with numeric distance for string integer", () => {
    expect(parseDeduplicateByTitle("10")).toEqual({ enabled: true, maxDistance: 10 });
  });

  it("returns enabled with distance 0 for integer 0", () => {
    expect(parseDeduplicateByTitle(0)).toEqual({ enabled: true, maxDistance: 0 });
  });

  it("throws for out-of-range integer (above 100)", () => {
    expect(() => parseDeduplicateByTitle(101)).toThrow("deduplicate-by-title");
  });

  it("throws for negative integer", () => {
    expect(() => parseDeduplicateByTitle(-1)).toThrow("deduplicate-by-title");
  });

  it("throws for float", () => {
    expect(() => parseDeduplicateByTitle(1.5)).toThrow("deduplicate-by-title");
  });

  it("throws for non-numeric string", () => {
    expect(() => parseDeduplicateByTitle("abc")).toThrow("deduplicate-by-title");
  });

  it("accepts max distance 100", () => {
    expect(parseDeduplicateByTitle(100)).toEqual({ enabled: true, maxDistance: 100 });
  });
});

describe("normalizeTitleForDedup", () => {
  it("lowercases title", () => {
    expect(normalizeTitleForDedup("Hello World")).toBe("hello world");
  });

  it("collapses multiple spaces", () => {
    expect(normalizeTitleForDedup("hello   world")).toBe("hello world");
  });

  it("trims leading/trailing whitespace", () => {
    expect(normalizeTitleForDedup("  hello  ")).toBe("hello");
  });

  it("handles empty string", () => {
    expect(normalizeTitleForDedup("")).toBe("");
  });

  it("handles mixed case and extra whitespace together", () => {
    expect(normalizeTitleForDedup("  Fix  BUG  ")).toBe("fix bug");
  });
});

describe("findDuplicateByTitle", () => {
  it("returns null when candidates is empty", () => {
    expect(findDuplicateByTitle("foo", [], 0)).toBeNull();
  });

  it("finds exact match with maxDistance 0", () => {
    const result = findDuplicateByTitle("fix bug", [{ title: "Fix Bug" }], 0);
    expect(result).not.toBeNull();
    expect(result?.distance).toBe(0);
    expect(result?.title).toBe("Fix Bug");
  });

  it("returns null when no match within maxDistance", () => {
    const result = findDuplicateByTitle("fix bug", [{ title: "add feature" }], 2);
    expect(result).toBeNull();
  });

  it("finds closest match within maxDistance", () => {
    const candidates = [{ title: "Fix Bug in Module A" }, { title: "fix bug" }];
    const result = findDuplicateByTitle("fix bug", candidates, 5);
    expect(result?.title).toBe("fix bug");
    expect(result?.distance).toBe(0);
  });

  it("uses normalizedTitle when provided", () => {
    const candidates = [{ title: "Original Title", normalizedTitle: "fix bug" }];
    const result = findDuplicateByTitle("fix bug", candidates, 0);
    expect(result?.title).toBe("Original Title");
    expect(result?.distance).toBe(0);
  });

  it("returns best match (lowest distance) among multiple candidates", () => {
    const candidates = [{ title: "fix bugs" }, { title: "fix bug" }];
    const result = findDuplicateByTitle("fix bug", candidates, 5);
    expect(result?.title).toBe("fix bug");
    expect(result?.distance).toBe(0);
  });

  it("returns null if best match exceeds maxDistance", () => {
    const candidates = [{ title: "completely different title entirely" }];
    const result = findDuplicateByTitle("fix bug", candidates, 2);
    expect(result).toBeNull();
  });
});
