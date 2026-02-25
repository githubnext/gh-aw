import { describe, it, expect } from "vitest";

const { parseBoolTemplatable, parseIntTemplatable } = require("./templatable.cjs");

describe("templatable.cjs", () => {
  describe("parseBoolTemplatable", () => {
    it("returns defaultValue (true) for undefined", () => {
      expect(parseBoolTemplatable(undefined)).toBe(true);
    });

    it("returns defaultValue (true) for null", () => {
      expect(parseBoolTemplatable(null)).toBe(true);
    });

    it("respects a custom defaultValue", () => {
      expect(parseBoolTemplatable(undefined, false)).toBe(false);
      expect(parseBoolTemplatable(null, false)).toBe(false);
    });

    it("handles boolean true", () => {
      expect(parseBoolTemplatable(true)).toBe(true);
    });

    it("handles boolean false", () => {
      expect(parseBoolTemplatable(false)).toBe(false);
    });

    it('handles string "true"', () => {
      expect(parseBoolTemplatable("true")).toBe(true);
    });

    it('handles string "false"', () => {
      expect(parseBoolTemplatable("false")).toBe(false);
    });

    it("treats a resolved expression value other than false as truthy", () => {
      // GitHub Actions expressions that resolve to something other than "false"
      // (e.g. "yes", "1", an empty object representation) should be truthy.
      expect(parseBoolTemplatable("yes")).toBe(true);
      expect(parseBoolTemplatable("1")).toBe(true);
    });

    it("treats the string false-equivalent as falsy", () => {
      expect(parseBoolTemplatable("false", true)).toBe(false);
    });
  });

  describe("parseIntTemplatable", () => {
    it("returns defaultValue (0) for undefined", () => {
      expect(parseIntTemplatable(undefined)).toBe(0);
    });

    it("returns defaultValue (0) for null", () => {
      expect(parseIntTemplatable(null)).toBe(0);
    });

    it("respects a custom defaultValue", () => {
      expect(parseIntTemplatable(undefined, 5)).toBe(5);
      expect(parseIntTemplatable(null, 10)).toBe(10);
    });

    it("handles integer numbers", () => {
      expect(parseIntTemplatable(5)).toBe(5);
      expect(parseIntTemplatable(0)).toBe(0);
      expect(parseIntTemplatable(100)).toBe(100);
    });

    it("handles numeric strings", () => {
      expect(parseIntTemplatable("5")).toBe(5);
      expect(parseIntTemplatable("42")).toBe(42);
      expect(parseIntTemplatable("0")).toBe(0);
    });

    it("handles resolved GitHub Actions expression values (integers as strings)", () => {
      // After runtime evaluation, expressions like "${{ inputs.max }}" become strings like "5"
      expect(parseIntTemplatable("3")).toBe(3);
      expect(parseIntTemplatable("168")).toBe(168);
    });

    it("returns defaultValue for non-numeric strings", () => {
      expect(parseIntTemplatable("not-a-number")).toBe(0);
      expect(parseIntTemplatable("abc", 1)).toBe(1);
    });

    it("truncates floating point numbers to integers", () => {
      expect(parseIntTemplatable(3.7)).toBe(3);
      expect(parseIntTemplatable("2.9")).toBe(2);
    });
  });
});
