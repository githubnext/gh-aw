// @ts-check

import { describe, it, expect } from "vitest";
import { formatCompactInteger } from "./compact_numbers.cjs";

describe("formatCompactInteger", () => {
  // --- exact integer range (< 1,000) ---

  it("returns '0' for 0", () => {
    expect(formatCompactInteger(0)).toBe("0");
  });

  it("returns exact string for single-digit value", () => {
    expect(formatCompactInteger(7)).toBe("7");
  });

  it("returns exact string for three-digit value", () => {
    expect(formatCompactInteger(900)).toBe("900");
  });

  it("returns exact string at boundary 999", () => {
    expect(formatCompactInteger(999)).toBe("999");
  });

  // --- K range (1,000–999,999) ---

  it("returns '1K' for exactly 1000", () => {
    expect(formatCompactInteger(1000)).toBe("1K");
  });

  it("returns '1.2K' for 1200", () => {
    expect(formatCompactInteger(1200)).toBe("1.2K");
  });

  it("returns '450K' for 450000 (no trailing decimal)", () => {
    expect(formatCompactInteger(450000)).toBe("450K");
  });

  it("returns '999.9K' for 999900", () => {
    expect(formatCompactInteger(999900)).toBe("999.9K");
  });

  it("promotes to '1M' at K/M seam (999950)", () => {
    // toFixed(1) of 999.95 rounds up to "1000.0"; the seam guard returns "1M" instead of "1000K"
    expect(formatCompactInteger(999950)).toBe("1M");
  });

  it("returns '1M' for 999999 (K upper boundary)", () => {
    expect(formatCompactInteger(999999)).toBe("1M");
  });

  // --- M range (>= 1,000,000) ---

  it("returns '1M' for exactly 1000000", () => {
    expect(formatCompactInteger(1_000_000)).toBe("1M");
  });

  it("returns '1.2M' for 1200000", () => {
    expect(formatCompactInteger(1_200_000)).toBe("1.2M");
  });

  it("returns '3M' for 3000000 (no trailing decimal)", () => {
    expect(formatCompactInteger(3_000_000)).toBe("3M");
  });

  it("returns '10.5M' for 10500000", () => {
    expect(formatCompactInteger(10_500_000)).toBe("10.5M");
  });

  it("returns '1000M' for 1 billion (M range is unbounded, no B suffix)", () => {
    expect(formatCompactInteger(1_000_000_000)).toBe("1000M");
  });

  // --- edge cases ---

  it("clamps negative values to 0", () => {
    expect(formatCompactInteger(-500)).toBe("0");
  });

  it("rounds non-integer input before formatting", () => {
    expect(formatCompactInteger(1499.6)).toBe("1.5K"); // Math.round(1499.6) = 1500 → "1.5K"
  });

  it("returns '0' for NaN", () => {
    expect(formatCompactInteger(NaN)).toBe("0");
  });

  it("returns '0' for Infinity", () => {
    expect(formatCompactInteger(Infinity)).toBe("0");
  });

  it("returns '0' for -Infinity", () => {
    expect(formatCompactInteger(-Infinity)).toBe("0");
  });
});
