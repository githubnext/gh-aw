import { describe, it as test, expect } from "vitest";
const { isTruthy } = require("./is_truthy.cjs");

describe("is_truthy.cjs", () => {
  describe("isTruthy", () => {
    test("should return false for empty string", () => {
      expect(isTruthy("")).toBe(false);
    });

    test('should return false for "false"', () => {
      expect(isTruthy("false")).toBe(false);
      expect(isTruthy("FALSE")).toBe(false);
      expect(isTruthy("False")).toBe(false);
    });

    test('should return false for "0"', () => {
      expect(isTruthy("0")).toBe(false);
    });

    test('should return false for "null"', () => {
      expect(isTruthy("null")).toBe(false);
      expect(isTruthy("NULL")).toBe(false);
    });

    test('should return false for "undefined"', () => {
      expect(isTruthy("undefined")).toBe(false);
      expect(isTruthy("UNDEFINED")).toBe(false);
    });

    test('should return true for "true"', () => {
      expect(isTruthy("true")).toBe(true);
      expect(isTruthy("TRUE")).toBe(true);
    });

    test("should return true for any non-falsy string", () => {
      expect(isTruthy("yes")).toBe(true);
      expect(isTruthy("1")).toBe(true);
      expect(isTruthy("hello")).toBe(true);
    });

    test("should trim whitespace", () => {
      expect(isTruthy("  false  ")).toBe(false);
      expect(isTruthy("  true  ")).toBe(true);
      expect(isTruthy("  ")).toBe(false);
    });

    test("should handle numeric strings", () => {
      expect(isTruthy("0")).toBe(false);
      expect(isTruthy("1")).toBe(true);
      expect(isTruthy("123")).toBe(true);
      expect(isTruthy("-1")).toBe(true);
    });

    test("should handle case-insensitive falsy values", () => {
      expect(isTruthy("FaLsE")).toBe(false);
      expect(isTruthy("NuLl")).toBe(false);
      expect(isTruthy("UnDeFiNeD")).toBe(false);
    });
  });
});
