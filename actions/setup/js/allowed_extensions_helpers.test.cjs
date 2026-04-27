import { describe, expect, it } from "vitest";
import { isGitHubExpression, normalizeAllowedExtension, parseAllowedExtensionsEnv } from "./allowed_extensions_helpers.cjs";

describe("allowed_extensions_helpers", () => {
  describe("isGitHubExpression", () => {
    it("returns true for full GitHub Actions expression", () => {
      expect(isGitHubExpression("${{ inputs.allowed_exts }}")).toBe(true);
    });

    it("returns true for expression with surrounding whitespace", () => {
      expect(isGitHubExpression("  ${{ inputs.allowed_exts }}  ")).toBe(true);
    });

    it("returns true for expression with no inner spaces", () => {
      expect(isGitHubExpression("${{inputs.x}}")).toBe(true);
    });

    it("returns false for non-expression text", () => {
      expect(isGitHubExpression("prefix ${{ inputs.allowed_exts }}")).toBe(false);
    });

    it("returns false for plain extension", () => {
      expect(isGitHubExpression(".txt")).toBe(false);
    });

    it("returns false for empty string", () => {
      expect(isGitHubExpression("")).toBe(false);
    });

    it("returns false for expression with trailing text", () => {
      expect(isGitHubExpression("${{ inputs.x }} extra")).toBe(false);
    });
  });

  describe("normalizeAllowedExtension", () => {
    it("normalizes case, trims spaces, and adds missing dot", () => {
      expect(normalizeAllowedExtension(" PNG ")).toBe(".png");
    });

    it("returns empty string for blank input", () => {
      expect(normalizeAllowedExtension("   ")).toBe("");
    });

    it("returns empty string for empty input", () => {
      expect(normalizeAllowedExtension("")).toBe("");
    });

    it("preserves existing leading dot and lowercases", () => {
      expect(normalizeAllowedExtension(".TXT")).toBe(".txt");
    });

    it("does not double-add dot when already present", () => {
      expect(normalizeAllowedExtension(".md")).toBe(".md");
    });

    it("passes through GitHub expression unchanged", () => {
      expect(normalizeAllowedExtension("${{ inputs.ext }}")).toBe("${{ inputs.ext }}");
    });

    it("handles mixed case extension without dot", () => {
      expect(normalizeAllowedExtension("JpEg")).toBe(".jpeg");
    });
  });

  describe("parseAllowedExtensionsEnv", () => {
    it("returns null when env value is undefined", () => {
      expect(parseAllowedExtensionsEnv(undefined)).toBeNull();
    });

    it("returns null when env value is empty string", () => {
      expect(parseAllowedExtensionsEnv("")).toBeNull();
    });

    it("parses single extension", () => {
      expect(parseAllowedExtensionsEnv(".md")).toEqual({
        rawValues: [".md"],
        normalizedValues: [".md"],
        hasUnresolvedExpression: false,
      });
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

    it("detects expression-only value", () => {
      const result = parseAllowedExtensionsEnv("${{ inputs.exts }}");
      expect(result).not.toBeNull();
      expect(result?.hasUnresolvedExpression).toBe(true);
    });

    it("filters out blank values after normalization", () => {
      const result = parseAllowedExtensionsEnv("  , .md,  ");
      expect(result?.normalizedValues).toEqual([".md"]);
    });
  });
});
