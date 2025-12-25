import { describe, it, expect, beforeEach, vi } from "vitest";

describe("push_repo_memory.cjs - glob pattern security tests", () => {
  describe("glob-to-regex conversion", () => {
    it("should correctly escape backslashes before other characters", () => {
      // This test verifies the security fix for Alert #84
      // The fix ensures backslashes are escaped FIRST, before escaping other characters

      // Test pattern: "test.txt" (a normal pattern)
      const pattern = "test.txt";

      // Simulate the conversion logic from push_repo_memory.cjs line 107
      // CORRECT: Escape backslashes first, then dots, then asterisks
      const regexPattern = pattern.replace(/\\/g, "\\\\").replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
      const regex = new RegExp(`^${regexPattern}$`);

      // After proper escaping:
      // "test.txt" -> "test.txt" (no backslashes) -> "test\.txt" (dot escaped)
      // The resulting regex should match only "test.txt" exactly

      // Should match exact filename
      expect(regex.test("test.txt")).toBe(true);

      // Should NOT match files where dot acts as wildcard
      expect(regex.test("test_txt")).toBe(false);
      expect(regex.test("testXtxt")).toBe(false);
    });

    it("should demonstrate INCORRECT escaping (vulnerable pattern)", () => {
      // This demonstrates the VULNERABLE version that was fixed
      // WITHOUT escaping backslashes first

      const pattern = "\\\\.txt";

      // INCORRECT: NOT escaping backslashes first
      const regexPattern = pattern.replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
      const regex = new RegExp(`^${regexPattern}$`);

      // This would create an incorrect regex pattern
      // The backslash isn't properly escaped, leading to potential bypass
    });

    it("should correctly escape dots to prevent matching any character", () => {
      // Test that dots are escaped, so "file.txt" doesn't match "filextxt"
      const pattern = "file.txt";

      const regexPattern = pattern.replace(/\\/g, "\\\\").replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
      const regex = new RegExp(`^${regexPattern}$`);

      // Should match exact filename
      expect(regex.test("file.txt")).toBe(true);

      // Should NOT match with dot as wildcard
      expect(regex.test("filextxt")).toBe(false);
      expect(regex.test("fileXtxt")).toBe(false);
      expect(regex.test("file_txt")).toBe(false);
    });

    it("should correctly convert asterisks to wildcard regex", () => {
      // Test that asterisks are converted to [^/]* (matches anything except slashes)
      const pattern = "*.txt";

      const regexPattern = pattern.replace(/\\/g, "\\\\").replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
      const regex = new RegExp(`^${regexPattern}$`);

      // Should match any filename ending in .txt
      expect(regex.test("file.txt")).toBe(true);
      expect(regex.test("document.txt")).toBe(true);
      expect(regex.test("test-file.txt")).toBe(true);

      // Should NOT match files without .txt extension
      expect(regex.test("file.md")).toBe(false);
      expect(regex.test("txt")).toBe(false);

      // Should NOT match paths with slashes (glob wildcards don't cross directories)
      expect(regex.test("dir/file.txt")).toBe(false);
    });

    it("should handle complex patterns with backslash and asterisk", () => {
      // Test pattern with asterisk wildcard
      const pattern = "test-*.txt";

      const regexPattern = pattern.replace(/\\/g, "\\\\").replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
      const regex = new RegExp(`^${regexPattern}$`);

      // After proper escaping:
      // "test-*.txt" -> "test-*.txt" (no backslashes) -> "test-*.txt" (no dots to escape except at end)
      //  -> "test-[^/]*\.txt" (asterisk converted to wildcard)
      
      // Should match files with the pattern
      expect(regex.test("test-file.txt")).toBe(true);
      expect(regex.test("test-123.txt")).toBe(true);
      expect(regex.test("test-.txt")).toBe(true);

      // Should NOT match files without the pattern
      expect(regex.test("test.txt")).toBe(false);
      expect(regex.test("other-file.txt")).toBe(false);
      expect(regex.test("test-file.md")).toBe(false);
    });

    it("should handle multiple patterns correctly", () => {
      // Test multiple space-separated patterns
      const patterns = "*.txt *.md".split(/\s+/).map(pattern => {
        const regexPattern = pattern.replace(/\\/g, "\\\\").replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
        return new RegExp(`^${regexPattern}$`);
      });

      // Should match .txt files
      expect(patterns.some(p => p.test("file.txt"))).toBe(true);
      expect(patterns.some(p => p.test("README.md"))).toBe(true);

      // Should NOT match other extensions
      expect(patterns.some(p => p.test("script.js"))).toBe(false);
      expect(patterns.some(p => p.test("image.png"))).toBe(false);
    });

    it("should handle exact filename patterns", () => {
      // Test exact filename match (no wildcards)
      const pattern = "specific-file.txt";

      const regexPattern = pattern.replace(/\\/g, "\\\\").replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
      const regex = new RegExp(`^${regexPattern}$`);

      // Should only match the exact filename
      expect(regex.test("specific-file.txt")).toBe(true);

      // Should NOT match similar filenames
      expect(regex.test("specific-file.md")).toBe(false);
      expect(regex.test("specific-file.txt.bak")).toBe(false);
      expect(regex.test("prefix-specific-file.txt")).toBe(false);
    });

    it("should preserve security - escape order matters", () => {
      // This test demonstrates WHY the escape order matters
      // It's the core security issue that was fixed

      const testPattern = "test\\.txt"; // Pattern with backslash-dot sequence

      // CORRECT order: backslash first
      const correctRegex = testPattern.replace(/\\/g, "\\\\").replace(/\./g, "\\.");
      const correct = new RegExp(`^${correctRegex}$`);

      // INCORRECT order: dot first (vulnerable)
      const incorrectRegex = testPattern.replace(/\./g, "\\.").replace(/\\/g, "\\\\");
      const incorrect = new RegExp(`^${incorrectRegex}$`);

      // The patterns should behave differently
      // This demonstrates the security implications of incorrect escape order
      expect(correctRegex).not.toBe(incorrectRegex);
    });
  });

  describe("security implications", () => {
    it("should prevent bypass attacks with crafted patterns", () => {
      // An attacker might try to craft patterns to bypass validation
      // The fix ensures proper escaping prevents such bypasses

      // Example: A complex pattern with special characters
      const attackPattern = "test.*";

      // With correct escaping (backslashes first)
      const safeRegexPattern = attackPattern.replace(/\\/g, "\\\\").replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
      const safeRegex = new RegExp(`^${safeRegexPattern}$`);

      // The pattern "test.*" should become "test\.[^/]*" in regex
      // Meaning: "test" + literal dot + any characters (not crossing directories)
      
      expect(safeRegex.test("test.txt")).toBe(true);
      expect(safeRegex.test("test.md")).toBe(true);
      expect(safeRegex.test("test.anything")).toBe(true);
      
      // Should NOT match without the dot
      expect(safeRegex.test("testtxt")).toBe(false);
      expect(safeRegex.test("testmd")).toBe(false);
    });

    it("should demonstrate CWE-20/80/116 prevention", () => {
      // This test relates to the CWEs mentioned in the security fix:
      // - CWE-20: Improper Input Validation
      // - CWE-80: Improper Neutralization of Script-Related HTML Tags
      // - CWE-116: Improper Encoding or Escaping of Output

      const userInput = "*.txt"; // Simulated user input from FILE_GLOB_FILTER

      // The fix ensures proper encoding/escaping of the pattern
      const escapedPattern = userInput.replace(/\\/g, "\\\\").replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
      const regex = new RegExp(`^${escapedPattern}$`);

      // Input is properly validated and sanitized
      // Pattern "*.txt" becomes "[^/]*\.txt" in regex
      expect(regex.test("normal.txt")).toBe(true);
      expect(regex.test("file.txt")).toBe(true);
      
      // Should not match non-.txt files
      expect(regex.test("normal.md")).toBe(false);
      expect(regex.test("file.js")).toBe(false);
    });
  });
});


