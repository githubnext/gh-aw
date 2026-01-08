// @ts-check
const { getCommonErrorPatterns, getCopilotErrorPatterns, getCodexErrorPatterns, getClaudeErrorPatterns, getErrorPatternsForEngine, getCustomErrorPatternsFromEnv } = require("./error_patterns.cjs");

describe("error_patterns", () => {
  describe("getCommonErrorPatterns", () => {
    it("should return an array of common error patterns", () => {
      const patterns = getCommonErrorPatterns();
      expect(Array.isArray(patterns)).toBe(true);
      expect(patterns.length).toBeGreaterThan(0);
    });

    it("should have valid pattern structure", () => {
      const patterns = getCommonErrorPatterns();
      patterns.forEach(pattern => {
        expect(pattern).toHaveProperty("id");
        expect(pattern).toHaveProperty("pattern");
        expect(pattern).toHaveProperty("level_group");
        expect(pattern).toHaveProperty("message_group");
        expect(pattern).toHaveProperty("description");
        expect(typeof pattern.id).toBe("string");
        expect(typeof pattern.pattern).toBe("string");
        expect(typeof pattern.level_group).toBe("number");
        expect(typeof pattern.message_group).toBe("number");
        expect(typeof pattern.description).toBe("string");
      });
    });

    it("should have valid regex patterns", () => {
      const patterns = getCommonErrorPatterns();
      patterns.forEach(pattern => {
        expect(() => new RegExp(pattern.pattern)).not.toThrow();
      });
    });

    it("should match GitHub Actions error command", () => {
      const patterns = getCommonErrorPatterns();
      const errorPattern = patterns.find(p => p.id === "common-gh-actions-error");
      expect(errorPattern).toBeDefined();

      const regex = new RegExp(errorPattern.pattern);
      const match = regex.exec("::error::Something went wrong");
      expect(match).not.toBeNull();
      expect(match[1]).toBe("error");
      expect(match[2]).toBe("Something went wrong");
    });

    it("should match generic ERROR messages", () => {
      const patterns = getCommonErrorPatterns();
      const errorPattern = patterns.find(p => p.id === "common-generic-error");
      expect(errorPattern).toBeDefined();

      const regex = new RegExp(errorPattern.pattern);
      const match = regex.exec("ERROR: File not found");
      expect(match).not.toBeNull();
      expect(match[1]).toBe("ERROR");
      expect(match[2]).toBe("File not found");
    });
  });

  describe("getCopilotErrorPatterns", () => {
    it("should return an array of Copilot-specific patterns", () => {
      const patterns = getCopilotErrorPatterns();
      expect(Array.isArray(patterns)).toBe(true);
      expect(patterns.length).toBeGreaterThan(0);
    });

    it("should have valid pattern structure", () => {
      const patterns = getCopilotErrorPatterns();
      patterns.forEach(pattern => {
        expect(pattern).toHaveProperty("id");
        expect(pattern).toHaveProperty("pattern");
        expect(pattern).toHaveProperty("level_group");
        expect(pattern).toHaveProperty("message_group");
        expect(pattern).toHaveProperty("description");
      });
    });

    it("should match Copilot timestamped warning", () => {
      const patterns = getCopilotErrorPatterns();
      const warningPattern = patterns.find(p => p.id === "copilot-timestamp-warning");
      expect(warningPattern).toBeDefined();

      const regex = new RegExp(warningPattern.pattern);
      const match = regex.exec("2024-01-15T10:30:45.123Z [WARN] Something is wrong");
      expect(match).not.toBeNull();
      expect(match[2]).toBe("WARN");
      expect(match[3]).toBe("Something is wrong");
    });

    it("should match Copilot failed command indicator", () => {
      const patterns = getCopilotErrorPatterns();
      const failedPattern = patterns.find(p => p.id === "copilot-failed-command");
      expect(failedPattern).toBeDefined();

      const regex = new RegExp(failedPattern.pattern);
      const match = regex.exec("✗ Command failed with exit code 1");
      expect(match).not.toBeNull();
      expect(match[1]).toBe("Command failed with exit code 1");
    });
  });

  describe("getCodexErrorPatterns", () => {
    it("should return an array of Codex-specific patterns", () => {
      const patterns = getCodexErrorPatterns();
      expect(Array.isArray(patterns)).toBe(true);
      expect(patterns.length).toBeGreaterThan(0);
    });

    it("should match Codex Rust format error", () => {
      const patterns = getCodexErrorPatterns();
      const errorPattern = patterns.find(p => p.id === "codex-rust-error");
      expect(errorPattern).toBeDefined();

      const regex = new RegExp(errorPattern.pattern);
      const match = regex.exec("2024-01-15T10:30:45.123456Z ERROR Something failed");
      expect(match).not.toBeNull();
      expect(match[2]).toBe("ERROR");
      expect(match[3]).toBe("Something failed");
    });

    it("should match Codex Rust format warning", () => {
      const patterns = getCodexErrorPatterns();
      const warningPattern = patterns.find(p => p.id === "codex-rust-warning");
      expect(warningPattern).toBeDefined();

      const regex = new RegExp(warningPattern.pattern);
      const match = regex.exec("2024-01-15T10:30:45.123456Z WARN Deprecated API used");
      expect(match).not.toBeNull();
      expect(match[2]).toBe("WARN");
      expect(match[3]).toBe("Deprecated API used");
    });
  });

  describe("getClaudeErrorPatterns", () => {
    it("should return an empty array (uses common patterns only)", () => {
      const patterns = getClaudeErrorPatterns();
      expect(Array.isArray(patterns)).toBe(true);
      expect(patterns.length).toBe(0);
    });
  });

  describe("getErrorPatternsForEngine", () => {
    it("should return common + Copilot patterns for copilot engine", () => {
      const patterns = getErrorPatternsForEngine("copilot");
      const commonPatterns = getCommonErrorPatterns();
      const copilotPatterns = getCopilotErrorPatterns();
      expect(patterns.length).toBe(commonPatterns.length + copilotPatterns.length);
    });

    it("should return common + Codex patterns for codex engine", () => {
      const patterns = getErrorPatternsForEngine("codex");
      const commonPatterns = getCommonErrorPatterns();
      const codexPatterns = getCodexErrorPatterns();
      expect(patterns.length).toBe(commonPatterns.length + codexPatterns.length);
    });

    it("should return common patterns only for claude engine", () => {
      const patterns = getErrorPatternsForEngine("claude");
      const commonPatterns = getCommonErrorPatterns();
      expect(patterns.length).toBe(commonPatterns.length);
    });

    it("should return common patterns for custom engine", () => {
      const patterns = getErrorPatternsForEngine("custom");
      const commonPatterns = getCommonErrorPatterns();
      expect(patterns.length).toBe(commonPatterns.length);
    });

    it("should return common patterns for unknown engine", () => {
      const patterns = getErrorPatternsForEngine("unknown");
      const commonPatterns = getCommonErrorPatterns();
      expect(patterns.length).toBe(commonPatterns.length);
    });
  });

  describe("getCustomErrorPatternsFromEnv", () => {
    const originalEnv = process.env.GH_AW_CUSTOM_ERROR_PATTERNS;

    afterEach(() => {
      if (originalEnv === undefined) {
        delete process.env.GH_AW_CUSTOM_ERROR_PATTERNS;
      } else {
        process.env.GH_AW_CUSTOM_ERROR_PATTERNS = originalEnv;
      }
    });

    it("should return empty array when environment variable is not set", () => {
      delete process.env.GH_AW_CUSTOM_ERROR_PATTERNS;
      const patterns = getCustomErrorPatternsFromEnv();
      expect(Array.isArray(patterns)).toBe(true);
      expect(patterns.length).toBe(0);
    });

    it("should parse valid JSON array from environment variable", () => {
      const customPatterns = [
        {
          id: "custom-pattern",
          pattern: "CUSTOM:\\s+(.+)",
          level_group: 0,
          message_group: 1,
          description: "Custom pattern",
        },
      ];
      process.env.GH_AW_CUSTOM_ERROR_PATTERNS = JSON.stringify(customPatterns);

      const patterns = getCustomErrorPatternsFromEnv();
      expect(patterns).toEqual(customPatterns);
    });

    it("should return empty array for invalid JSON", () => {
      process.env.GH_AW_CUSTOM_ERROR_PATTERNS = "not valid json";
      const patterns = getCustomErrorPatternsFromEnv();
      expect(Array.isArray(patterns)).toBe(true);
      expect(patterns.length).toBe(0);
    });

    it("should return empty array for non-array JSON", () => {
      process.env.GH_AW_CUSTOM_ERROR_PATTERNS = '{"pattern": "test"}';
      const patterns = getCustomErrorPatternsFromEnv();
      expect(Array.isArray(patterns)).toBe(true);
      expect(patterns.length).toBe(0);
    });
  });
});
