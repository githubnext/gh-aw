// @ts-check
import { describe, expect, it, vi } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { redactMaskedValues } = require("./mask_value_matcher.cjs");

describe("mask value matcher", () => {
  it("preserves per-value replacement markers across adjacent and overlapping matches", () => {
    const replacement = value => (value === "configured" ? "***REDACTED***" : "***");
    expect(redactMaskedValues("configuredruntimeconfiguredconfigured runtime", ["configured", "runtime", "figure"], replacement)).toBe("***REDACTED***" + "***" + "***REDACTED***".repeat(2) + " ***");
    expect(redactMaskedValues("ababa", ["aba"])).toBe("***ba");
    expect(redactMaskedValues("ababa", ["aba"], replacement)).toBe("***");
  });

  it("replaces every occurrence with the default marker", () => {
    expect(redactMaskedValues("token=abc and abc again", ["abc"])).toBe("token=*** and *** again");
  });

  it("treats mask text literally instead of as a regular expression", () => {
    expect(redactMaskedValues("value a.c here", ["a.c"])).toBe("value *** here");
    expect(redactMaskedValues("value abc here", ["a.c"])).toBe("value abc here");
  });

  it("ignores empty masks and empty input", () => {
    expect(redactMaskedValues("plain", [])).toBe("plain");
    expect(redactMaskedValues("plain", ["", ""])).toBe("plain");
    expect(redactMaskedValues("", ["secret"])).toBe("");
  });

  it("redacts masks exceeding the regular expression size limit", () => {
    const mask = "x".repeat(200000);
    expect(redactMaskedValues(`before ${mask} after ${mask}`, [mask])).toBe("before *** after ***");
    expect(redactMaskedValues("unrelated output", [mask])).toBe("unrelated output");
  });

  it("preserves leftmost matching and supplied mask priority for overlaps", () => {
    expect(redactMaskedValues("abc ab", ["abc", "ab"])).toBe("*** ***");
    expect(redactMaskedValues("abc", ["ab", "abc"])).toBe("***c");
    expect(redactMaskedValues("zabc", ["abc", "za"])).toBe("***bc");
  });

  it("does not rescan replacements or create matches across removed values", () => {
    expect(redactMaskedValues("secret *", ["secret", "*"])).toBe("*** ***");
    expect(redactMaskedValues("axb", ["x", "ab"])).toBe("a***b");
  });

  it("matches ordered regex semantics while refreshing overlapping cached candidates", () => {
    const values = ["ab", "aba", "ba", "b", "a", "", "ab"];
    for (let offset = 0; offset < values.length; offset++) {
      const masks = [...values.slice(offset), ...values.slice(0, offset)];
      const pattern = new RegExp(masks.filter(Boolean).join("|"), "g");
      for (let length = 0; length <= 6; length++) {
        for (let bits = 0; bits < 2 ** length; bits++) {
          const text = Array.from({ length }, (_, index) => (bits & (1 << index) ? "a" : "b")).join("");
          expect(redactMaskedValues(text, masks)).toBe(text.replace(pattern, "***"));
        }
      }
    }
  });

  it("does not mutate mask ordering when choosing overlap replacements", () => {
    const masks = ["ab", "abc"];
    expect(redactMaskedValues("abc", masks, value => `[${value}]`)).toBe("[abc]");
    expect(masks).toEqual(["ab", "abc"]);
  });

  it("does not re-search later masks for every frequent match", () => {
    const masks = Array.from({ length: 500 }, (_, index) => `mask-${index.toString().padStart(4, "0")}`);
    const repeats = 2000;
    const suffix = masks.join(" ");
    const text = "hot".repeat(repeats) + " ".repeat(2 * 1024 * 1024) + suffix;
    const indexOf = String.prototype.indexOf;
    let searches = 0;
    const spy = vi.spyOn(String.prototype, "indexOf").mockImplementation(function (value, from) {
      if (this === text) searches++;
      return indexOf.call(this, value, from);
    });
    let result;
    try {
      result = redactMaskedValues(text, [...masks, "hot"]);
    } finally {
      spy.mockRestore();
    }
    expect(result).toBe("***".repeat(repeats) + " ".repeat(2 * 1024 * 1024) + masks.map(() => "***").join(" "));
    expect(searches).toBeLessThanOrEqual(masks.length * 2 + repeats + 1);
  });
});
