import { describe, it as test, expect } from "vitest";
const { isTruthy } = require("./is_truthy.cjs");
describe("is_truthy.cjs", () => {
  describe("isTruthy", () => {
    (test("should return false for empty string", () => {
      expect(isTruthy("")).toBe(!1);
    }),
      test('should return false for "false"', () => {
        (expect(isTruthy("false")).toBe(!1), expect(isTruthy("FALSE")).toBe(!1), expect(isTruthy("False")).toBe(!1));
      }),
      test('should return false for "0"', () => {
        expect(isTruthy("0")).toBe(!1);
      }),
      test('should return false for "null"', () => {
        (expect(isTruthy("null")).toBe(!1), expect(isTruthy("NULL")).toBe(!1));
      }),
      test('should return false for "undefined"', () => {
        (expect(isTruthy("undefined")).toBe(!1), expect(isTruthy("UNDEFINED")).toBe(!1));
      }),
      test('should return true for "true"', () => {
        (expect(isTruthy("true")).toBe(!0), expect(isTruthy("TRUE")).toBe(!0));
      }),
      test("should return true for any non-falsy string", () => {
        (expect(isTruthy("yes")).toBe(!0), expect(isTruthy("1")).toBe(!0), expect(isTruthy("hello")).toBe(!0));
      }),
      test("should trim whitespace", () => {
        (expect(isTruthy("  false  ")).toBe(!1), expect(isTruthy("  true  ")).toBe(!0), expect(isTruthy("  ")).toBe(!1));
      }),
      test("should handle numeric strings", () => {
        (expect(isTruthy("0")).toBe(!1), expect(isTruthy("1")).toBe(!0), expect(isTruthy("123")).toBe(!0), expect(isTruthy("-1")).toBe(!0));
      }),
      test("should handle case-insensitive falsy values", () => {
        (expect(isTruthy("FaLsE")).toBe(!1), expect(isTruthy("NuLl")).toBe(!1), expect(isTruthy("UnDeFiNeD")).toBe(!1));
      }));
  });
});
