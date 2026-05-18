import { describe, it, expect } from "vitest";
import { levenshteinDistance } from "./levenshtein_distance.cjs";

describe("levenshtein_distance", () => {
  it("returns zero for identical strings", () => {
    expect(levenshteinDistance("create issue", "create issue")).toBe(0);
  });

  it("handles empty strings", () => {
    expect(levenshteinDistance("", "")).toBe(0);
    expect(levenshteinDistance("", "abc")).toBe(3);
    expect(levenshteinDistance("abc", "")).toBe(3);
  });

  it("computes insertion, deletion and substitution costs", () => {
    expect(levenshteinDistance("abc", "abdc")).toBe(1); // insertion
    expect(levenshteinDistance("abdc", "abc")).toBe(1); // deletion
    expect(levenshteinDistance("abc", "axc")).toBe(1); // substitution
  });

  it("matches known examples", () => {
    expect(levenshteinDistance("kitten", "sitting")).toBe(3);
    expect(levenshteinDistance("flaw", "lawn")).toBe(2);
    expect(levenshteinDistance("Saturday", "Sunday")).toBe(3);
  });

  it("is symmetric", () => {
    const a = "feature: deduplicate by title";
    const b = "feature: dedupe by title";
    expect(levenshteinDistance(a, b)).toBe(levenshteinDistance(b, a));
  });

  it("supports unicode characters", () => {
    expect(levenshteinDistance("café", "cafe")).toBe(1);
    expect(levenshteinDistance("🧪test", "🧪tests")).toBe(1);
  });

  it("coerces non-string inputs safely", () => {
    expect(levenshteinDistance(1234, 1234)).toBe(0);
    expect(levenshteinDistance(null, "x")).toBe(1);
    expect(levenshteinDistance(undefined, "")).toBe(0);
  });
});
