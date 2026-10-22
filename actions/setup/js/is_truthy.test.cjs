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

    describe('GitHub Actions script style equality (v == "b")', () => {
      it.each(["'", '"'])("compares literals delimited by %s for every supported operator", quote => {
        for (const operator of ["==", "==="]) {
          expect(isTruthy(`  inline_strict ${operator} ${quote}inline_strict${quote}  `)).toBe(true);
          expect(isTruthy(`inline_strict ${operator} ${quote}single_agent_control${quote}`)).toBe(false);
          expect(isTruthy(`${operator} ${quote}inline_strict${quote}`)).toBe(false);
          expect(isTruthy(`${operator} ${quote}${quote}`)).toBe(true);
          expect(isTruthy(`inline_strict ${operator} ${quote}${quote}`)).toBe(false);
        }
        for (const operator of ["!=", "!=="]) {
          expect(isTruthy(`inline_strict ${operator} ${quote}inline_strict${quote}`)).toBe(false);
          expect(isTruthy(`inline_strict ${operator} ${quote}single_agent_control${quote}`)).toBe(true);
          expect(isTruthy(`${operator} ${quote}${quote}`)).toBe(false);
          expect(isTruthy(`${operator} ${quote}inline_strict${quote}`)).toBe(true);
        }
      });

      it("preserves the other quote style inside a literal", () => {
        expect(isTruthy(`it's == "it's"`)).toBe(true);
        expect(isTruthy(`say "yes" == 'say "yes"'`)).toBe(true);
        expect(isTruthy(`other == "it's"`)).toBe(false);
        expect(isTruthy(`other == 'say "yes"'`)).toBe(false);
      });

      it("should return true when value matches literal with ==", () => {
        expect(isTruthy('concise == "concise"')).toBe(true);
        expect(isTruthy('verbose == "verbose"')).toBe(true);
        expect(isTruthy('yes == "yes"')).toBe(true);
      });

      it("should return false when value does not match literal with ==", () => {
        expect(isTruthy('verbose == "concise"')).toBe(false);
        expect(isTruthy('concise == "verbose"')).toBe(false);
        expect(isTruthy('no == "yes"')).toBe(false);
      });

      it("should return true when value matches literal with ===", () => {
        expect(isTruthy('concise === "concise"')).toBe(true);
      });

      it("should return false when value does not match literal with ===", () => {
        expect(isTruthy('verbose === "concise"')).toBe(false);
      });

      it("should return true when value does not match literal with !=", () => {
        expect(isTruthy('verbose != "concise"')).toBe(true);
      });

      it("should return false when value matches literal with !=", () => {
        expect(isTruthy('concise != "concise"')).toBe(false);
      });

      it("should handle whitespace around operators", () => {
        expect(isTruthy('  concise  ==  "concise"  ')).toBe(true);
      });

      it("should return false for empty LHS (unset experiment variable)", () => {
        // If the experiment env var was not set, substitution leaves an empty LHS:
        // {{#if  == "concise"}} — should be false, not truthy fallback
        expect(isTruthy(' == "concise"')).toBe(false);
        expect(isTruthy('== "concise"')).toBe(false);
      });
    });
  });
});
