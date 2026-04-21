import { describe, expect, it } from "vitest";
import { isGitHubExpression, normalizeAllowedExtension, parseAllowedExtensionsEnv } from "./allowed_extensions_helpers.cjs";

describe("allowed_extensions_helpers", () => {
  describe("isGitHubExpression", () => {
    it("returns true for full GitHub Actions expression", () => {
      expect(isGitHubExpression("${{ inputs.allowed_exts }}")).toBe(true);
    });

    it("returns false for non-expression text", () => {
      expect(isGitHubExpression("prefix ${{ inputs.allowed_exts }}")).toBe(false);
    });
  });

  describe("normalizeAllowedExtension", () => {
    it("normalizes case, trims spaces, and adds missing dot", () => {
      expect(normalizeAllowedExtension(" PNG ")).toBe(".png");
    });

    it("returns empty string for blank input", () => {
      expect(normalizeAllowedExtension("   ")).toBe("");
    });
  });

  describe("parseAllowedExtensionsEnv", () => {
    it("returns null when env value is undefined", () => {
      expect(parseAllowedExtensionsEnv(undefined)).toBeNull();
    });

    it("parses and normalizes literal extension values", () => {
      expect(parseAllowedExtensionsEnv("TXT, md")).toEqual({
        rawValues: ["TXT", "md"],
        normalizedValues: [".txt", ".md"],
        hasUnresolvedExpression: false,
      });
    });

    it("detects unresolved GitHub Actions expressions", () => {
      expect(parseAllowedExtensionsEnv(".txt,${{ inputs.allowed_exts }}")).toEqual({
        rawValues: [".txt", "${{ inputs.allowed_exts }}"],
        normalizedValues: [".txt", "${{ inputs.allowed_exts }}"],
        hasUnresolvedExpression: true,
      });
    });
  });
});
