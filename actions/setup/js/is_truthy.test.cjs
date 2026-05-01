import { describe, it, expect } from "vitest";

const { isTruthy } = require("./is_truthy.cjs");

describe("is_truthy.cjs", () => {
  describe("isTruthy", () => {
    it("should return false for empty string", () => {
      expect(isTruthy("")).toBe(false);
    });

    it('should return false for "false" (case-insensitive)', () => {
      expect(isTruthy("false")).toBe(false);
      expect(isTruthy("FALSE")).toBe(false);
      expect(isTruthy("False")).toBe(false);
    });

    it('should return false for "0"', () => {
      expect(isTruthy("0")).toBe(false);
    });

    it('should return false for "null" (case-insensitive)', () => {
      expect(isTruthy("null")).toBe(false);
      expect(isTruthy("NULL")).toBe(false);
    });

    it('should return false for "undefined" (case-insensitive)', () => {
      expect(isTruthy("undefined")).toBe(false);
      expect(isTruthy("UNDEFINED")).toBe(false);
    });

    it('should return true for "true" (case-insensitive)', () => {
      expect(isTruthy("true")).toBe(true);
      expect(isTruthy("TRUE")).toBe(true);
    });

    it("should return true for any non-falsy string", () => {
      expect(isTruthy("yes")).toBe(true);
      expect(isTruthy("1")).toBe(true);
      expect(isTruthy("hello")).toBe(true);
    });

    it('should return false for "no" (case-insensitive)', () => {
      expect(isTruthy("no")).toBe(false);
      expect(isTruthy("NO")).toBe(false);
      expect(isTruthy("No")).toBe(false);
    });

    it("should trim whitespace", () => {
      expect(isTruthy("  false  ")).toBe(false);
      expect(isTruthy("  true  ")).toBe(true);
      expect(isTruthy("  ")).toBe(false);
    });

    it("should handle numeric strings", () => {
      expect(isTruthy("0")).toBe(false);
      expect(isTruthy("1")).toBe(true);
      expect(isTruthy("123")).toBe(true);
      expect(isTruthy("-1")).toBe(true);
    });

    it("should handle case-insensitive falsy values", () => {
      expect(isTruthy("FaLsE")).toBe(false);
      expect(isTruthy("NuLl")).toBe(false);
      expect(isTruthy("UnDeFiNeD")).toBe(false);
    });

    describe("(eq VALUE LITERAL) helper", () => {
      it("should return true when value matches literal", () => {
        expect(isTruthy('(eq concise "concise")')).toBe(true);
        expect(isTruthy('(eq verbose "verbose")')).toBe(true);
        expect(isTruthy('(eq yes "yes")')).toBe(true);
      });

      it("should return false when value does not match literal", () => {
        expect(isTruthy('(eq verbose "concise")')).toBe(false);
        expect(isTruthy('(eq concise "verbose")')).toBe(false);
        expect(isTruthy('(eq no "yes")')).toBe(false);
      });

      it("should be case-insensitive for the (eq ...) wrapper but case-sensitive for values", () => {
        expect(isTruthy('(EQ concise "concise")')).toBe(true);
        expect(isTruthy('(Eq concise "concise")')).toBe(true);
      });

      it("should handle values with spaces via trim", () => {
        expect(isTruthy('(eq  concise  "concise")')).toBe(true);
      });
    });
  });
});
