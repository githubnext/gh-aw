import { describe, expect, it } from "vitest";

describe("effective_token_limits", () => {
  it("normalizes ET suffix strings", async () => {
    const { parsePositiveEffectiveTokenLimitString } = await import("./effective_token_limits.cjs");

    expect(parsePositiveEffectiveTokenLimitString("100M")).toBe("100000000");
    expect(parsePositiveEffectiveTokenLimitString("100000k")).toBe("100000000");
    expect(parsePositiveEffectiveTokenLimitString(" 100M ")).toBe("100000000");
    expect(parsePositiveEffectiveTokenLimitString(2500)).toBe("2500");
    expect(parsePositiveEffectiveTokenLimitString(9007199254740992)).toBe("");
    expect(parsePositiveEffectiveTokenLimitString("2500")).toBe("2500");
    expect(parsePositiveEffectiveTokenLimitString("0")).toBe("");
    expect(parsePositiveEffectiveTokenLimitString("-1")).toBe("");
  });

  it("parses safe integer ET suffix numbers", async () => {
    const { parsePositiveEffectiveTokenLimitNumber } = await import("./effective_token_limits.cjs");

    expect(parsePositiveEffectiveTokenLimitNumber("100M")).toBe(100000000);
    expect(parsePositiveEffectiveTokenLimitNumber("100000K")).toBe(100000000);
    expect(parsePositiveEffectiveTokenLimitNumber("abc")).toBe(0);
  });
});
