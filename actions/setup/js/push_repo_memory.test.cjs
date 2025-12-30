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

    it("should correctly match .jsonl files with *.jsonl pattern", () => {
      // Test case for validating .jsonl file pattern matching
      // This validates the fix for: https://github.com/githubnext/gh-aw/actions/runs/20601784686/job/59169295542#step:7:1
      // The daily-code-metrics workflow uses file-glob: ["*.json", "*.jsonl", "*.csv", "*.md"]
      // and writes history.jsonl file to repo memory at memory/default/history.jsonl

      const fileGlobFilter = "*.json *.jsonl *.csv *.md";
      const patterns = fileGlobFilter.split(/\s+/).map(pattern => {
        const regexPattern = pattern
          .replace(/\\/g, "\\\\")
          .replace(/\./g, "\\.")
          .replace(/\*\*/g, "<!DOUBLESTAR>")
          .replace(/\*/g, "[^/]*")
          .replace(/<!DOUBLESTAR>/g, ".*");
        return new RegExp(`^${regexPattern}$`);
      });

      // Should match .jsonl files (the actual file from workflow run: history.jsonl)
      // Note: Pattern matching is done on relative filename only, not full path
      expect(patterns.some(p => p.test("history.jsonl"))).toBe(true);
      expect(patterns.some(p => p.test("data.jsonl"))).toBe(true);
      expect(patterns.some(p => p.test("metrics.jsonl"))).toBe(true);

      // Should also match other allowed extensions
      expect(patterns.some(p => p.test("config.json"))).toBe(true);
      expect(patterns.some(p => p.test("data.csv"))).toBe(true);
      expect(patterns.some(p => p.test("README.md"))).toBe(true);

      // Should NOT match disallowed extensions
      expect(patterns.some(p => p.test("script.js"))).toBe(false);
      expect(patterns.some(p => p.test("image.png"))).toBe(false);
      expect(patterns.some(p => p.test("document.txt"))).toBe(false);

      // Edge case: Should NOT match .json when pattern is *.jsonl
      expect(patterns.some(p => p.test("file.json"))).toBe(true); // matches *.json pattern
      const jsonlOnlyPattern = "*.jsonl";
      const jsonlRegex = new RegExp(`^${jsonlOnlyPattern.replace(/\\/g, "\\\\").replace(/\./g, "\\.").replace(/\*/g, "[^/]*")}$`);
      expect(jsonlRegex.test("file.json")).toBe(false); // should NOT match .json with *.jsonl pattern
      expect(jsonlRegex.test("file.jsonl")).toBe(true); // should match .jsonl with *.jsonl pattern
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

  describe("subdirectory glob pattern support", () => {
    // Tests for the new ** wildcard support added for subdirectory handling

    it("should handle ** wildcard to match any path including slashes", () => {
      // Test the new ** pattern that matches across directories
      const pattern = "metrics/**";

      // New conversion logic: ** -> .* (matches everything including /)
      const regexPattern = pattern
        .replace(/\\/g, "\\\\")
        .replace(/\./g, "\\.")
        .replace(/\*\*/g, "<!DOUBLESTAR>")
        .replace(/\*/g, "[^/]*")
        .replace(/<!DOUBLESTAR>/g, ".*");
      const regex = new RegExp(`^${regexPattern}$`);

      // Should match nested files in metrics directory (including any extension)
      expect(regex.test("metrics/latest.json")).toBe(true);
      expect(regex.test("metrics/daily/2024-12-26.json")).toBe(true);
      expect(regex.test("metrics/daily/archive/2024-01-01.json")).toBe(true);
      expect(regex.test("metrics/readme.md")).toBe(true);

      // Should NOT match files outside metrics directory
      expect(regex.test("data/file.json")).toBe(false);
      expect(regex.test("file.json")).toBe(false);
    });

    it("should differentiate between * and ** wildcards", () => {
      // Test that * doesn't cross directories but ** does

      // Single * pattern - should NOT match subdirectories
      const singleStarPattern = "metrics/*";
      const singleStarRegex = singleStarPattern
        .replace(/\\/g, "\\\\")
        .replace(/\./g, "\\.")
        .replace(/\*\*/g, "<!DOUBLESTAR>")
        .replace(/\*/g, "[^/]*")
        .replace(/<!DOUBLESTAR>/g, ".*");
      const singleStar = new RegExp(`^${singleStarRegex}$`);

      // Should match direct children only
      expect(singleStar.test("metrics/file.json")).toBe(true);
      expect(singleStar.test("metrics/latest.json")).toBe(true);

      // Should NOT match nested files
      expect(singleStar.test("metrics/daily/file.json")).toBe(false);

      // Double ** pattern - should match subdirectories
      const doubleStarPattern = "metrics/**";
      const doubleStarRegex = doubleStarPattern
        .replace(/\\/g, "\\\\")
        .replace(/\./g, "\\.")
        .replace(/\*\*/g, "<!DOUBLESTAR>")
        .replace(/\*/g, "[^/]*")
        .replace(/<!DOUBLESTAR>/g, ".*");
      const doubleStar = new RegExp(`^${doubleStarRegex}$`);

      // Should match both direct and nested files
      expect(doubleStar.test("metrics/file.json")).toBe(true);
      expect(doubleStar.test("metrics/daily/file.json")).toBe(true);
      expect(doubleStar.test("metrics/daily/archive/file.json")).toBe(true);
    });

    it("should handle **/* pattern correctly", () => {
      // Test **/* which requires at least one directory level
      // Note: ** matches one or more path segments in this implementation
      const pattern = "**/*";

      const regexPattern = pattern
        .replace(/\\/g, "\\\\")
        .replace(/\./g, "\\.")
        .replace(/\*\*/g, "<!DOUBLESTAR>")
        .replace(/\*/g, "[^/]*")
        .replace(/<!DOUBLESTAR>/g, ".*");
      const regex = new RegExp(`^${regexPattern}$`);

      // With current implementation, **/* requires at least one slash
      expect(regex.test("dir/file.txt")).toBe(true);
      expect(regex.test("dir/subdir/file.txt")).toBe(true);
      expect(regex.test("very/deep/nested/path/file.json")).toBe(true);

      // Does not match files in root (no slash)
      expect(regex.test("file.txt")).toBe(false);
    });

    it("should handle mixed * and ** in same pattern", () => {
      // Test patterns with both single and double wildcards
      const pattern = "logs/**";

      const regexPattern = pattern
        .replace(/\\/g, "\\\\")
        .replace(/\./g, "\\.")
        .replace(/\*\*/g, "<!DOUBLESTAR>")
        .replace(/\*/g, "[^/]*")
        .replace(/<!DOUBLESTAR>/g, ".*");
      const regex = new RegExp(`^${regexPattern}$`);

      // Should match any logs at any depth in logs directory
      expect(regex.test("logs/error-123.log")).toBe(true);
      expect(regex.test("logs/2024/error-456.log")).toBe(true);
      expect(regex.test("logs/2024/12/error-789.log")).toBe(true);
      expect(regex.test("logs/info-123.log")).toBe(true);
      expect(regex.test("logs/2024/warning-456.log")).toBe(true);

      // Should NOT match logs outside logs directory
      expect(regex.test("error-123.log")).toBe(false);
    });

    it("should handle subdirectory patterns for metrics use case", () => {
      // Real-world test for the metrics collector use case
      // Note: metrics/**/* requires at least one directory level under metrics
      const pattern = "metrics/**/*";

      const regexPattern = pattern
        .replace(/\\/g, "\\\\")
        .replace(/\./g, "\\.")
        .replace(/\*\*/g, "<!DOUBLESTAR>")
        .replace(/\*/g, "[^/]*")
        .replace(/<!DOUBLESTAR>/g, ".*");
      const regex = new RegExp(`^${regexPattern}$`);

      // Should match files in subdirectories
      expect(regex.test("metrics/daily/2024-12-26.json")).toBe(true);
      expect(regex.test("metrics/daily/2024-12-25.json")).toBe(true);
      expect(regex.test("metrics/subdir/config.yaml")).toBe(true);

      // Does NOT match direct children (needs at least one subdir)
      // This is current behavior - could be improved in future
      expect(regex.test("metrics/latest.json")).toBe(false);

      // Should NOT match files outside metrics directory
      expect(regex.test("data/metrics.json")).toBe(false);
      expect(regex.test("latest.json")).toBe(false);
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

    it("should prevent directory traversal with ** wildcard", () => {
      // Ensure ** wildcard doesn't enable directory traversal attacks
      const pattern = "data/**";

      const regexPattern = pattern
        .replace(/\\/g, "\\\\")
        .replace(/\./g, "\\.")
        .replace(/\*\*/g, "<!DOUBLESTAR>")
        .replace(/\*/g, "[^/]*")
        .replace(/<!DOUBLESTAR>/g, ".*");
      const regex = new RegExp(`^${regexPattern}$`);

      // Should match legitimate nested files
      expect(regex.test("data/file.json")).toBe(true);
      expect(regex.test("data/subdir/file.json")).toBe(true);

      // Should NOT match files outside data directory
      // Note: The pattern is anchored with ^ and $, so it must match the full path
      expect(regex.test("../sensitive/file.json")).toBe(false);
      expect(regex.test("/etc/passwd")).toBe(false);
      expect(regex.test("other/data/file.json")).toBe(false);
    });
  });

  describe("multi-pattern filter support", () => {
    it("should support multiple space-separated patterns", () => {
      // Test multiple patterns like "campaign-id/cursor.json campaign-id/metrics/**"
      const patterns = "security-q1/cursor.json security-q1/metrics/**".split(/\s+/).filter(Boolean);

      // Each pattern should be validated independently
      expect(patterns).toHaveLength(2);
      expect(patterns[0]).toBe("security-q1/cursor.json");
      expect(patterns[1]).toBe("security-q1/metrics/**");
    });

    it("should validate each pattern in multi-pattern filter", () => {
      // Test that each pattern can be converted to regex independently
      const patterns = "data/**.json logs/**.log".split(/\s+/).filter(Boolean);

      const regexPatterns = patterns.map(pattern => {
        const regexPattern = pattern
          .replace(/\\/g, "\\\\")
          .replace(/\./g, "\\.")
          .replace(/\*\*/g, "<!DOUBLESTAR>")
          .replace(/\*/g, "[^/]*")
          .replace(/<!DOUBLESTAR>/g, ".*");
        return new RegExp(`^${regexPattern}$`);
      });

      // First pattern should match .json files in data/
      expect(regexPatterns[0].test("data/file.json")).toBe(true);
      expect(regexPatterns[0].test("data/subdir/file.json")).toBe(true);
      expect(regexPatterns[0].test("logs/file.log")).toBe(false);

      // Second pattern should match .log files in logs/
      expect(regexPatterns[1].test("logs/file.log")).toBe(true);
      expect(regexPatterns[1].test("logs/subdir/file.log")).toBe(true);
      expect(regexPatterns[1].test("data/file.json")).toBe(false);
    });

    it("should handle campaign-specific multi-pattern filters", () => {
      // Real-world campaign use case: multiple specific patterns
      const patterns = "security-q1/cursor.json security-q1/metrics/**".split(/\s+/).filter(Boolean);

      const regexPatterns = patterns.map(pattern => {
        const regexPattern = pattern
          .replace(/\\/g, "\\\\")
          .replace(/\./g, "\\.")
          .replace(/\*\*/g, "<!DOUBLESTAR>")
          .replace(/\*/g, "[^/]*")
          .replace(/<!DOUBLESTAR>/g, ".*");
        return new RegExp(`^${regexPattern}$`);
      });

      // First pattern: exact cursor file
      expect(regexPatterns[0].test("security-q1/cursor.json")).toBe(true);
      expect(regexPatterns[0].test("security-q1/cursor.txt")).toBe(false);
      expect(regexPatterns[0].test("security-q1/metrics/2024-12-29.json")).toBe(false);

      // Second pattern: any metrics files
      expect(regexPatterns[1].test("security-q1/metrics/2024-12-29.json")).toBe(true);
      expect(regexPatterns[1].test("security-q1/metrics/daily/snapshot.json")).toBe(true);
      expect(regexPatterns[1].test("security-q1/cursor.json")).toBe(false);
    });
  });

  describe("campaign ID validation", () => {
    it("should extract campaign ID from first pattern", () => {
      // Test extracting campaign ID from pattern like "security-q1/**"
      const pattern = "security-q1/**";
      const match = /^([^*?/]+)\/\*\*/.exec(pattern);

      expect(match).not.toBeNull();
      expect(match[1]).toBe("security-q1");
    });

    it("should validate all patterns start with campaign ID", () => {
      // Test that all patterns must be under campaign-id/ subdirectory
      const campaignId = "security-q1";
      const validPatterns = ["security-q1/cursor.json", "security-q1/metrics/**", "security-q1/data/*.txt"];

      for (const pattern of validPatterns) {
        expect(pattern.startsWith(`${campaignId}/`)).toBe(true);
      }

      const invalidPatterns = ["other-campaign/cursor.json", "cursor.json", "metrics/**"];

      for (const pattern of invalidPatterns) {
        expect(pattern.startsWith(`${campaignId}/`)).toBe(false);
      }
    });

    it("should handle campaign ID with hyphens and underscores", () => {
      // Test various campaign ID formats
      const patterns = ["security-q1-2025/**", "incident_response/**", "rollout-v2_phase1/**"];

      for (const pattern of patterns) {
        const match = /^([^*?/]+)\/\*\*/.exec(pattern);
        expect(match).not.toBeNull();

        // Extracted campaign ID should match the prefix
        const campaignId = match[1];
        expect(pattern.startsWith(`${campaignId}/`)).toBe(true);
      }
    });

    it("should reject patterns not under campaign ID subdirectory", () => {
      // Test enforcement that patterns must be under campaign-id/
      const campaignId = "security-q1";

      // Valid: under campaign-id/
      expect("security-q1/metrics/**".startsWith(`${campaignId}/`)).toBe(true);
      expect("security-q1/cursor.json".startsWith(`${campaignId}/`)).toBe(true);

      // Invalid: not under campaign-id/
      expect("metrics/**".startsWith(`${campaignId}/`)).toBe(false);
      expect("other-campaign/data.json".startsWith(`${campaignId}/`)).toBe(false);
      expect("cursor.json".startsWith(`${campaignId}/`)).toBe(false);
    });

    it("should support explicit GH_AW_CAMPAIGN_ID override", () => {
      // Test that environment variable can override campaign ID detection
      // This would be simulated in the actual code by process.env.GH_AW_CAMPAIGN_ID
      const explicitCampaignId = "rollout-v2";
      const patterns = ["rollout-v2/cursor.json", "rollout-v2/metrics/**"];

      // All patterns should validate against explicit campaign ID
      for (const pattern of patterns) {
        expect(pattern.startsWith(`${explicitCampaignId}/`)).toBe(true);
      }
    });
  });
});
