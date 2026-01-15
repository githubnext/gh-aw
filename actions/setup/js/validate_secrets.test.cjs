// @ts-check

import { describe, it, expect } from "vitest";
import { testGitHubRESTAPI, testGitHubGraphQLAPI, testCopilotCLI, testAnthropicAPI, testOpenAIAPI, testBraveSearchAPI, testNotionAPI, generateMarkdownReport } from "./validate_secrets.cjs";

describe("validate_secrets", () => {
  describe("testGitHubRESTAPI", () => {
    it("should return NOT_SET when token is not provided", async () => {
      const result = await testGitHubRESTAPI("", "owner", "repo");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("Token not set");
    });

    it("should return NOT_SET when token is null", async () => {
      const result = await testGitHubRESTAPI(null, "owner", "repo");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("Token not set");
    });

    it("should return NOT_SET when token is undefined", async () => {
      const result = await testGitHubRESTAPI(undefined, "owner", "repo");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("Token not set");
    });
  });

  describe("testGitHubGraphQLAPI", () => {
    it("should return NOT_SET when token is not provided", async () => {
      const result = await testGitHubGraphQLAPI("", "owner", "repo");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("Token not set");
    });
  });

  describe("testCopilotCLI", () => {
    it("should return NOT_SET when token is not provided", async () => {
      const result = await testCopilotCLI("");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("Token not set");
    });
  });

  describe("testAnthropicAPI", () => {
    it("should return NOT_SET when API key is not provided", async () => {
      const result = await testAnthropicAPI("");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("API key not set");
    });
  });

  describe("testOpenAIAPI", () => {
    it("should return NOT_SET when API key is not provided", async () => {
      const result = await testOpenAIAPI("");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("API key not set");
    });
  });

  describe("testBraveSearchAPI", () => {
    it("should return NOT_SET when API key is not provided", async () => {
      const result = await testBraveSearchAPI("");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("API key not set");
    });
  });

  describe("testNotionAPI", () => {
    it("should return NOT_SET when token is not provided", async () => {
      const result = await testNotionAPI("");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("Token not set");
    });
  });

  describe("generateMarkdownReport", () => {
    it("should generate a report with summary and detailed results", () => {
      const results = [
        {
          secret: "TEST_SECRET",
          test: "Test API",
          status: "success",
          message: "Test passed",
          details: { statusCode: 200 },
        },
        {
          secret: "ANOTHER_SECRET",
          test: "Another Test",
          status: "failure",
          message: "Test failed",
          details: { statusCode: 401 },
        },
        {
          secret: "NOT_SET_SECRET",
          test: "Not Set Test",
          status: "not_set",
          message: "Token not set",
        },
      ];

      const report = generateMarkdownReport(results);

      // Check that report contains expected sections
      expect(report).toContain("📊 Summary");
      expect(report).toContain("🔍 Detailed Results");
      expect(report).toContain("TEST_SECRET");
      expect(report).toContain("ANOTHER_SECRET");
      expect(report).toContain("NOT_SET_SECRET");

      // Check for status emojis
      expect(report).toContain("✅");
      expect(report).toContain("❌");
      expect(report).toContain("⚪");

      // Check for summary table
      expect(report).toContain("| Status | Count | Percentage |");

      // Check for recommendations
      expect(report).toContain("[!WARNING]");
      expect(report).toContain("[!NOTE]");
    });

    it("should generate a successful report when all secrets are valid", () => {
      const results = [
        {
          secret: "TEST_SECRET",
          test: "Test API",
          status: "success",
          message: "Test passed",
          details: { statusCode: 200 },
        },
      ];

      const report = generateMarkdownReport(results);

      expect(report).toContain("📊 Summary");
      expect(report).toContain("[!TIP]");
      expect(report).toContain("All configured secrets are working correctly!");
    });

    it("should include documentation links for secrets", () => {
      const results = [
        {
          secret: "GH_AW_GITHUB_TOKEN",
          test: "GitHub REST API",
          status: "failure",
          message: "Invalid token",
          details: { statusCode: 401 },
        },
        {
          secret: "ANTHROPIC_API_KEY",
          test: "Anthropic API",
          status: "not_set",
          message: "API key not set",
        },
      ];

      const report = generateMarkdownReport(results);

      // Check for GitHub docs link
      expect(report).toContain("docs.github.com");
      expect(report).toContain("docs.anthropic.com");
    });

    it("should handle empty results gracefully", () => {
      const results = [];

      const report = generateMarkdownReport(results);

      expect(report).toContain("📊 Summary");
      expect(report).toContain("| **Total** | **0** | **100%** |");
    });

    it("should handle skipped tests", () => {
      const results = [
        {
          secret: "SKIPPED_SECRET",
          test: "Skipped Test",
          status: "skipped",
          message: "Test skipped",
        },
      ];

      const report = generateMarkdownReport(results);

      expect(report).toContain("⏭️");
      expect(report).toContain("Skipped");
    });

    it("should group tests by secret", () => {
      const results = [
        {
          secret: "GH_AW_GITHUB_TOKEN",
          test: "GitHub REST API",
          status: "success",
          message: "REST API successful",
        },
        {
          secret: "GH_AW_GITHUB_TOKEN",
          test: "GitHub GraphQL API",
          status: "success",
          message: "GraphQL API successful",
        },
      ];

      const report = generateMarkdownReport(results);

      // Should show the secret once with 2 tests
      expect(report).toContain("GH_AW_GITHUB_TOKEN");
      expect(report).toContain("(2 tests)");
      expect(report).toContain("GitHub REST API");
      expect(report).toContain("GitHub GraphQL API");
    });
  });
});
