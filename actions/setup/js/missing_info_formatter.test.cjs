import { describe, it, expect } from "vitest";

describe("missing_info_formatter.cjs", () => {
  let formatter;

  beforeEach(async () => {
    formatter = await import("./missing_info_formatter.cjs");
  });

  describe("escapeMarkdown", () => {
    it("should escape markdown special characters", () => {
      const { escapeMarkdown } = formatter;
      expect(escapeMarkdown("**bold**")).toBe("\\*\\*bold\\*\\*");
      expect(escapeMarkdown("_italic_")).toBe("\\_italic\\_");
      expect(escapeMarkdown("[link](url)")).toBe("\\[link\\]\\(url\\)");
      expect(escapeMarkdown("<tag>")).toBe("&lt;tag&gt;");
    });

    it("should handle empty and null strings", () => {
      const { escapeMarkdown } = formatter;
      expect(escapeMarkdown("")).toBe("");
      expect(escapeMarkdown(null)).toBe("");
    });
  });

  describe("formatMissingTools", () => {
    it("should format missing tools into markdown list", () => {
      const { formatMissingTools } = formatter;
      const tools = [
        { tool: "docker", reason: "Need containerization", alternatives: "Use VM" },
        { tool: "kubectl", reason: "Kubernetes management" },
      ];

      const result = formatMissingTools(tools);
      expect(result).toContain("**docker**");
      expect(result).toContain("Need containerization");
      expect(result).toContain("*Alternatives*: Use VM");
      expect(result).toContain("**kubectl**");
      expect(result).toContain("Kubernetes management");
    });

    it("should return empty string for empty array", () => {
      const { formatMissingTools } = formatter;
      expect(formatMissingTools([])).toBe("");
      expect(formatMissingTools(null)).toBe("");
    });

    it("should escape special characters in tool info", () => {
      const { formatMissingTools } = formatter;
      const tools = [{ tool: "my_tool", reason: "Need <special> chars" }];

      const result = formatMissingTools(tools);
      expect(result).toContain("my\\_tool");
      expect(result).toContain("&lt;special&gt;");
    });
  });

  describe("formatMissingData", () => {
    it("should format missing data into markdown list", () => {
      const { formatMissingData } = formatter;
      const data = [
        {
          data_type: "api_key",
          reason: "API credentials missing",
          context: "GitHub API access",
          alternatives: "Use read-only token",
        },
        {
          data_type: "config",
          reason: "Configuration not found",
        },
      ];

      const result = formatMissingData(data);
      expect(result).toContain("**api\\_key**");
      expect(result).toContain("API credentials missing");
      expect(result).toContain("*Context*: GitHub API access");
      expect(result).toContain("*Alternatives*: Use read-only token");
      expect(result).toContain("**config**");
      expect(result).toContain("Configuration not found");
    });

    it("should return empty string for empty array", () => {
      const { formatMissingData } = formatter;
      expect(formatMissingData([])).toBe("");
      expect(formatMissingData(null)).toBe("");
    });
  });

  describe("generateMissingToolsSection", () => {
    it("should generate HTML details section for missing tools", () => {
      const { generateMissingToolsSection } = formatter;
      const tools = [{ tool: "docker", reason: "Need containerization" }];

      const result = generateMissingToolsSection(tools);
      expect(result).toContain("<details>");
      expect(result).toContain("<summary><b>Missing Tools</b></summary>");
      expect(result).toContain("**docker**");
      expect(result).toContain("</details>");
    });

    it("should return empty string for no tools", () => {
      const { generateMissingToolsSection } = formatter;
      expect(generateMissingToolsSection([])).toBe("");
      expect(generateMissingToolsSection(null)).toBe("");
    });
  });

  describe("generateMissingDataSection", () => {
    it("should generate HTML details section for missing data", () => {
      const { generateMissingDataSection } = formatter;
      const data = [{ data_type: "api_key", reason: "Credentials missing" }];

      const result = generateMissingDataSection(data);
      expect(result).toContain("<details>");
      expect(result).toContain("<summary><b>Missing Data</b></summary>");
      expect(result).toContain("**api\\_key**");
      expect(result).toContain("</details>");
    });

    it("should return empty string for no data", () => {
      const { generateMissingDataSection } = formatter;
      expect(generateMissingDataSection([])).toBe("");
      expect(generateMissingDataSection(null)).toBe("");
    });
  });

  describe("formatNoopMessages", () => {
    it("should format noop messages into markdown list", () => {
      const { formatNoopMessages } = formatter;
      const messages = [{ message: "No issues found in this review" }, { message: "Analysis complete, no action needed" }];

      const result = formatNoopMessages(messages);
      expect(result).toContain("No issues found in this review");
      expect(result).toContain("Analysis complete, no action needed");
    });

    it("should return empty string for empty array", () => {
      const { formatNoopMessages } = formatter;
      expect(formatNoopMessages([])).toBe("");
      expect(formatNoopMessages(null)).toBe("");
    });

    it("should escape special characters in noop messages", () => {
      const { formatNoopMessages } = formatter;
      const messages = [{ message: "Found <special> chars **bold**" }];

      const result = formatNoopMessages(messages);
      expect(result).toContain("&lt;special&gt;");
      expect(result).toContain("\\*\\*bold\\*\\*");
    });
  });

  describe("generateNoopMessagesSection", () => {
    it("should generate HTML details section for noop messages", () => {
      const { generateNoopMessagesSection } = formatter;
      const messages = [{ message: "No action required" }];

      const result = generateNoopMessagesSection(messages);
      expect(result).toContain("<details>");
      expect(result).toContain("<summary><b>No-Op Messages</b></summary>");
      expect(result).toContain("No action required");
      expect(result).toContain("</details>");
    });

    it("should return empty string for no messages", () => {
      const { generateNoopMessagesSection } = formatter;
      expect(generateNoopMessagesSection([])).toBe("");
      expect(generateNoopMessagesSection(null)).toBe("");
    });
  });

  describe("generateMissingInfoSections", () => {
    it("should generate both tools and data sections", () => {
      const { generateMissingInfoSections } = formatter;
      const missings = {
        missingTools: [{ tool: "docker", reason: "Need containers" }],
        missingData: [{ data_type: "api_key", reason: "No credentials" }],
      };

      const result = generateMissingInfoSections(missings);
      expect(result).toContain("Missing Tools");
      expect(result).toContain("Missing Data");
      expect(result).toContain("docker");
      expect(result).toContain("api\\_key"); // Escaped underscore
    });

    it("should generate sections with noop messages", () => {
      const { generateMissingInfoSections } = formatter;
      const missings = {
        missingTools: [{ tool: "docker", reason: "Need containers" }],
        missingData: [{ data_type: "api_key", reason: "No credentials" }],
        noopMessages: [{ message: "No issues found" }],
      };

      const result = generateMissingInfoSections(missings);
      expect(result).toContain("Missing Tools");
      expect(result).toContain("Missing Data");
      expect(result).toContain("No-Op Messages");
      expect(result).toContain("docker");
      expect(result).toContain("api\\_key");
      expect(result).toContain("No issues found");
    });

    it("should handle only noop messages", () => {
      const { generateMissingInfoSections } = formatter;
      const missings = {
        noopMessages: [{ message: "Analysis complete" }],
      };

      const result = generateMissingInfoSections(missings);
      expect(result).not.toContain("Missing Tools");
      expect(result).not.toContain("Missing Data");
      expect(result).toContain("No-Op Messages");
      expect(result).toContain("Analysis complete");
    });

    it("should handle only tools", () => {
      const { generateMissingInfoSections } = formatter;
      const missings = {
        missingTools: [{ tool: "docker", reason: "Need containers" }],
      };

      const result = generateMissingInfoSections(missings);
      expect(result).toContain("Missing Tools");
      expect(result).not.toContain("Missing Data");
    });

    it("should handle only data", () => {
      const { generateMissingInfoSections } = formatter;
      const missings = {
        missingData: [{ data_type: "api_key", reason: "No credentials" }],
      };

      const result = generateMissingInfoSections(missings);
      expect(result).not.toContain("Missing Tools");
      expect(result).toContain("Missing Data");
    });

    it("should return empty string for no missings", () => {
      const { generateMissingInfoSections } = formatter;
      expect(generateMissingInfoSections(null)).toBe("");
      expect(generateMissingInfoSections({})).toBe("");
      expect(generateMissingInfoSections({ missingTools: [], missingData: [] })).toBe("");
    });
  });
});
