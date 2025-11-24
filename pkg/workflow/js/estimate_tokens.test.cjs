import { describe, it, expect } from "vitest";

describe("estimateTokens", () => {
  it("should estimate tokens for text", async () => {
    const { estimateTokens } = await import("./estimate_tokens.cjs");

    expect(estimateTokens("hello")).toBe(2); // 5 chars / 4 = 1.25, ceil = 2
    expect(estimateTokens("test")).toBe(1); // 4 chars / 4 = 1
    expect(estimateTokens("testing")).toBe(2); // 7 chars / 4 = 1.75, ceil = 2
  });

  it("should handle empty strings and null", async () => {
    const { estimateTokens } = await import("./estimate_tokens.cjs");

    expect(estimateTokens("")).toBe(0);
    expect(estimateTokens(null)).toBe(0);
    expect(estimateTokens(undefined)).toBe(0);
  });

  it("should handle long text", async () => {
    const { estimateTokens } = await import("./estimate_tokens.cjs");

    const longText = "a".repeat(1000);
    expect(estimateTokens(longText)).toBe(250); // 1000 / 4 = 250
  });

  it("should round up using Math.ceil", async () => {
    const { estimateTokens } = await import("./estimate_tokens.cjs");

    expect(estimateTokens("a")).toBe(1); // 1 / 4 = 0.25, ceil = 1
    expect(estimateTokens("ab")).toBe(1); // 2 / 4 = 0.5, ceil = 1
    expect(estimateTokens("abc")).toBe(1); // 3 / 4 = 0.75, ceil = 1
    expect(estimateTokens("abcd")).toBe(1); // 4 / 4 = 1
    expect(estimateTokens("abcde")).toBe(2); // 5 / 4 = 1.25, ceil = 2
  });

  it("should handle multi-byte characters", async () => {
    const { estimateTokens } = await import("./estimate_tokens.cjs");

    // Note: This uses string length, not byte length
    expect(estimateTokens("😀")).toBe(1); // 1 char (even though it's 4 bytes)
    expect(estimateTokens("你好世界")).toBe(1); // 4 chars / 4 = 1
  });

  it("should handle whitespace and special characters", async () => {
    const { estimateTokens } = await import("./estimate_tokens.cjs");

    expect(estimateTokens("   ")).toBe(1); // 3 spaces / 4 = 0.75, ceil = 1
    expect(estimateTokens("\n\n\n\n")).toBe(1); // 4 newlines / 4 = 1
    expect(estimateTokens("!@#$%")).toBe(2); // 5 special chars / 4 = 1.25, ceil = 2
  });
});
