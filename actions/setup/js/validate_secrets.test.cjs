// @ts-check

import { describe, it, expect, vi, afterEach } from "vitest";
import https from "https";
import { EventEmitter } from "events";
import {
  makeRequest,
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
  /** @type {(EventEmitter & {setTimeout: any, destroy: any, write: any, end: any, timeoutCallback?: () => void})|null} */
  let mockRequest = null;
  /** @type {(EventEmitter & {statusCode: number})|null} */
  let mockResponse = null;

  afterEach(() => {
    vi.restoreAllMocks();
    mockRequest = null;
    mockResponse = null;
  });

  /**
   * @param {((data?: string) => void)|null} onEnd
   * @param {number} [statusCode]
   */
  function setupHttpsMock(onEnd, statusCode = 200) {
    mockResponse = Object.assign(new EventEmitter(), { statusCode });
    mockRequest = Object.assign(new EventEmitter(), {
      setTimeout: vi.fn().mockImplementation((_ms, cb) => {
        /** @type {any} */ mockRequest.timeoutCallback = cb;
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

    it("should return SUCCESS on 200 response", async () => {
      const body = JSON.stringify({ full_name: "owner/repo", private: false });
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", body);
          mockResponse?.emit("end");
        });
      });
      const result = await testGitHubRESTAPI("token", "owner", "repo");
      expect(result.status).toBe("success");
      expect(result.details?.repoName).toBe("owner/repo");
    });

    it("should return FAILURE on non-200 response", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", "");
          mockResponse?.emit("end");
        });
      }, 401);
      const result = await testGitHubRESTAPI("bad-token", "owner", "repo");
      expect(result.status).toBe("failure");
      expect(result.message).toContain("401");
    });

    it("should return FAILURE on network error", async () => {
      setupHttpsMock(null);
      const promise = testGitHubRESTAPI("token", "owner", "repo");
      process.nextTick(() => mockRequest?.emit("error", new Error("ECONNREFUSED")));
      const result = await promise;
      expect(result.status).toBe("failure");
      expect(result.message).toContain("ECONNREFUSED");
    });
  });

  describe("testGitHubGraphQLAPI", () => {
    it("should return NOT_SET when token is not provided", async () => {
      const result = await testGitHubGraphQLAPI("", "owner", "repo");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("Token not set");
    });

    it("should return SUCCESS on 200 response with valid data", async () => {
      const body = JSON.stringify({ data: { repository: { name: "repo", owner: { login: "owner" }, isPrivate: false } } });
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", body);
          mockResponse?.emit("end");
        });
      });
      const result = await testGitHubGraphQLAPI("token", "owner", "repo");
      expect(result.status).toBe("success");
      expect(result.details?.repoName).toBe("repo");
    });

    it("should return FAILURE when GraphQL response contains errors", async () => {
      const body = JSON.stringify({ errors: [{ message: "Not found" }] });
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", body);
          mockResponse?.emit("end");
        });
      });
      const result = await testGitHubGraphQLAPI("token", "owner", "repo");
      expect(result.status).toBe("failure");
      expect(result.message).toContain("errors");
    });

    it("should return FAILURE on non-200 response", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", "");
          mockResponse?.emit("end");
        });
      }, 403);
      const result = await testGitHubGraphQLAPI("token", "owner", "repo");
      expect(result.status).toBe("failure");
      expect(result.message).toContain("403");
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

    it("should return SUCCESS on 200 response", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", "{}");
          mockResponse?.emit("end");
        });
      });
      const result = await testAnthropicAPI("valid-key");
      expect(result.status).toBe("success");
    });

    it("should return FAILURE on 401 invalid key", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", "");
          mockResponse?.emit("end");
        });
      }, 401);
      const result = await testAnthropicAPI("bad-key");
      expect(result.status).toBe("failure");
      expect(result.message).toContain("Invalid Anthropic API key");
    });

    it("should return FAILURE on unexpected status code", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", "");
          mockResponse?.emit("end");
        });
      }, 429);
      const result = await testAnthropicAPI("key");
      expect(result.status).toBe("failure");
      expect(result.message).toContain("429");
    });
  });

  describe("testOpenAIAPI", () => {
    it("should return NOT_SET when API key is not provided", async () => {
      const result = await testOpenAIAPI("");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("API key not set");
    });

    it("should return FAILURE on 401 invalid key", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", "");
          mockResponse?.emit("end");
        });
      }, 401);
      const result = await testOpenAIAPI("bad-key");
      expect(result.status).toBe("failure");
      expect(result.message).toContain("Invalid OpenAI API key");
    });
  });

  describe("testBraveSearchAPI", () => {
    it("should return NOT_SET when API key is not provided", async () => {
      const result = await testBraveSearchAPI("");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("API key not set");
    });

    it("should return FAILURE on 403 invalid key", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", "");
          mockResponse?.emit("end");
        });
      }, 403);
      const result = await testBraveSearchAPI("bad-key");
      expect(result.status).toBe("failure");
      expect(result.message).toContain("Invalid Brave Search API key");
    });
  });

  describe("testNotionAPI", () => {
    it("should return NOT_SET when token is not provided", async () => {
      const result = await testNotionAPI("");
      expect(result.status).toBe("not_set");
      expect(result.message).toBe("Token not set");
    });

    it("should return FAILURE on 401 invalid token", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", "");
          mockResponse?.emit("end");
        });
      }, 401);
      const result = await testNotionAPI("bad-token");
      expect(result.status).toBe("failure");
      expect(result.message).toContain("Invalid Notion API token");
    });
  });

  describe("makeRequest", () => {
    it("resolves with statusCode and data for GET requests", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", '{"status":"ok"}');
          mockResponse?.emit("end");
        });
      });
      const result = await makeRequest("api.example.com", "/v1/test", { Accept: "application/json" });
      expect(result.statusCode).toBe(200);
      expect(result.data).toBe('{"status":"ok"}');
    });

    it("does not call write for GET requests", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", "");
          mockResponse?.emit("end");
        });
      });
      await makeRequest("api.example.com", "/v1/test", {});
      expect(mockRequest?.write).not.toHaveBeenCalled();
    });

    it("rejects on request error", async () => {
      setupHttpsMock(null);
      const promise = makeRequest("api.example.com", "/v1/test", {});
      process.nextTick(() => mockRequest?.emit("error", new Error("ENOTFOUND")));
      await expect(promise).rejects.toThrow("ENOTFOUND");
    });

    it("rejects with timeout error after 10 s", async () => {
      setupHttpsMock(null);
      const promise = makeRequest("api.example.com", "/v1/test", {});
      process.nextTick(() => /** @type {any} */ mockRequest?.timeoutCallback?.());
      await expect(promise).rejects.toThrow("Request timeout");
      expect(mockRequest?.destroy).toHaveBeenCalled();
    });
  });

  describe("makePostRequest", () => {
    it("resolves with statusCode and data on success", async () => {
      setupHttpsMock(() => {
        process.nextTick(() => {
          mockResponse?.emit("data", '{"ok":true}');
          mockResponse?.emit("end");
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
      process.nextTick(() => mockRequest?.emit("error", networkError));

      await expect(promise).rejects.toThrow("connection refused");
    });

    it("rejects with timeout error after 10 s", async () => {
      setupHttpsMock(null);

      const promise = makePostRequest("api.example.com", "/v1/test", {}, "{}");
      // Trigger the timeout callback registered via req.setTimeout
      process.nextTick(() => /** @type {any} */ mockRequest?.timeoutCallback?.());

      await expect(promise).rejects.toThrow("Request timeout");
      expect(mockRequest?.destroy).toHaveBeenCalled();
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
      // Percentages should be 0%, not NaN%
      expect(report).toContain("| ✅ Successful | 0 | 0% |");
      expect(report).not.toContain("NaN");
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
