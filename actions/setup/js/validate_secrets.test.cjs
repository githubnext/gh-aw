// @ts-check

import { describe, it, expect, vi, afterEach } from "vitest";
import https from "https";
import { EventEmitter } from "events";
import {
  makePostRequest,
  testGitHubRESTAPI,
  testGitHubGraphQLAPI,
  testCopilotCLI,
  testCopilotToken,
  testAnthropicAPI,
  testOpenAIAPI,
  testBraveSearchAPI,
  testNotionAPI,
  generateMarkdownReport,
  isForkRepository,
  statusEmoji,
  Status,
} from "./validate_secrets.cjs";

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

  describe("testCopilotToken", () => {
    it("should return SKIPPED when token is not set and org billing is active", async () => {
      const result = await testCopilotToken("", true);
      expect(result.status).toBe("skipped");
      expect(result.message).toContain("org billing");
    });

    it("should return SKIPPED when token is undefined and org billing is active", async () => {
      const result = await testCopilotToken(undefined, true);
      expect(result.status).toBe("skipped");
      expect(result.message).toContain("GITHUB_TOKEN");
    });

    it("should return NOT_SET when token is not set and org billing is not active", async () => {
      const result = await testCopilotToken("", false);
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("Token not set");
    });

    it("should delegate to testCopilotCLI when token is set regardless of org billing", async () => {
      // testCopilotCLI with a non-empty token checks CLI availability (skipped if not installed)
      const result = await testCopilotToken("some-token", true);
      // Result should be skipped or success depending on environment, but NOT the org billing skip
      expect(result.message).not.toContain("org billing");
    });

    it("should not suppress warning when token is missing and org billing is false", async () => {
      const result = await testCopilotToken(undefined, false);
      expect(result.status).toBe("not_set");
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

  describe("makePostRequest", () => {
    /** @type {EventEmitter & {setTimeout: any, destroy: any, write: any, end: any, timeoutCallback?: () => void}} */
    let mockRequest;
    /** @type {EventEmitter & {statusCode: number}} */
    let mockResponse;

    afterEach(() => {
      vi.restoreAllMocks();
    });

    function setupHttpsMock(onEnd) {
      mockResponse = Object.assign(new EventEmitter(), { statusCode: 200 });
      mockRequest = Object.assign(new EventEmitter(), {
        setTimeout: vi.fn().mockImplementation((ms, cb) => {
          mockRequest.timeoutCallback = cb;
        }),
        destroy: vi.fn(),
        write: vi.fn(),
        end: vi.fn().mockImplementation(() => {
          if (onEnd) onEnd();
        }),
      });
      vi.spyOn(https, "request").mockImplementation((_options, callback) => {
        process.nextTick(() => callback?.(/** @type {any} */ mockResponse));
        return /** @type {any} */ mockRequest;
      });
    }

    it("resolves with statusCode and data on success", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse.emit("data", '{"ok":true}');
          mockResponse.emit("end");
        });
      });

      const result = await makePostRequest("api.example.com", "/v1/test", { "Content-Type": "application/json" }, '{"query":"test"}');
      expect(result.statusCode).toBe(200);
      expect(result.data).toBe('{"ok":true}');
    });

    it("rejects on request error", async () => {
      setupHttpsMock(null);
      const networkError = new Error("connection refused");

      const promise = makePostRequest("api.example.com", "/v1/test", {}, "{}");
      process.nextTick(() => mockRequest.emit("error", networkError));

      await expect(promise).rejects.toThrow("connection refused");
    });

    it("rejects with timeout error after 10 s", async () => {
      setupHttpsMock(null);

      const promise = makePostRequest("api.example.com", "/v1/test", {}, "{}");
      // Trigger the timeout callback registered via req.setTimeout
      process.nextTick(() => mockRequest.timeoutCallback?.());

      await expect(promise).rejects.toThrow("Request timeout");
      expect(mockRequest.destroy).toHaveBeenCalled();
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

  describe("isForkRepository", () => {
    it("should return true when repository.fork is true", () => {
      const payload = { repository: { fork: true } };
      expect(isForkRepository(payload)).toBe(true);
    });

    it("should return false when repository.fork is false", () => {
      const payload = { repository: { fork: false } };
      expect(isForkRepository(payload)).toBe(false);
    });

    it("should return false when repository.fork is absent", () => {
      const payload = { repository: {} };
      expect(isForkRepository(payload)).toBe(false);
    });

    it("should return false when repository is absent", () => {
      const payload = {};
      expect(isForkRepository(payload)).toBe(false);
    });

    it("should return false when payload is null", () => {
      expect(isForkRepository(null)).toBe(false);
    });

    it("should return false when payload is undefined", () => {
      expect(isForkRepository(undefined)).toBe(false);
    });
  });

  describe("statusEmoji", () => {
    it("should return ✅ for success", () => {
      expect(statusEmoji(Status.SUCCESS)).toBe("✅");
    });

    it("should return ❌ for failure", () => {
      expect(statusEmoji(Status.FAILURE)).toBe("❌");
    });

    it("should return ⚪ for not_set", () => {
      expect(statusEmoji(Status.NOT_SET)).toBe("⚪");
    });

    it("should return ⏭️ for skipped", () => {
      expect(statusEmoji(Status.SKIPPED)).toBe("⏭️");
    });

    it("should return ❓ for unknown status", () => {
      expect(statusEmoji("unknown")).toBe("❓");
    });

    it("should return ❓ for empty string", () => {
      expect(statusEmoji("")).toBe("❓");
    });
  });
});
