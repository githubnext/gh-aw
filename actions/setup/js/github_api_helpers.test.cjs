import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  info: vi.fn(),
};

global.core = mockCore;

describe("github_api_helpers.cjs", () => {
  let getFileContent;
  let logGraphQLError;
  let mockGithub;

  beforeEach(async () => {
    vi.clearAllMocks();

    mockGithub = {
      rest: {
        repos: {
          getContent: vi.fn(),
        },
      },
    };

    // Dynamically import the module
    const module = await import("./github_api_helpers.cjs");
    getFileContent = module.getFileContent;
    logGraphQLError = module.logGraphQLError;
  });

  describe("getFileContent", () => {
    it("should fetch and decode base64 file content", async () => {
      const fileContent = "Hello, World!";
      mockGithub.rest.repos.getContent.mockResolvedValueOnce({
        data: {
          type: "file",
          encoding: "base64",
          content: Buffer.from(fileContent).toString("base64"),
        },
      });

      const result = await getFileContent(mockGithub, "owner", "repo", "file.txt", "main");

      expect(result).toBe(fileContent);
      expect(mockGithub.rest.repos.getContent).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        path: "file.txt",
        ref: "main",
      });
    });

    it("should handle non-base64 content", async () => {
      const fileContent = "Plain text content";
      mockGithub.rest.repos.getContent.mockResolvedValueOnce({
        data: {
          type: "file",
          encoding: "utf-8",
          content: fileContent,
        },
      });

      const result = await getFileContent(mockGithub, "owner", "repo", "file.txt", "main");

      expect(result).toBe(fileContent);
    });

    it("should return null for directory paths", async () => {
      mockGithub.rest.repos.getContent.mockResolvedValueOnce({
        data: [
          { name: "file1.txt", type: "file" },
          { name: "file2.txt", type: "file" },
        ],
      });

      const result = await getFileContent(mockGithub, "owner", "repo", "directory", "main");

      expect(result).toBeNull();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("is a directory"));
    });

    it("should return null for non-file types", async () => {
      mockGithub.rest.repos.getContent.mockResolvedValueOnce({
        data: {
          type: "symlink",
          encoding: "base64",
          content: "link-content",
        },
      });

      const result = await getFileContent(mockGithub, "owner", "repo", "symlink.txt", "main");

      expect(result).toBeNull();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("is not a file"));
    });

    it("should handle API errors gracefully", async () => {
      mockGithub.rest.repos.getContent.mockRejectedValueOnce(new Error("API error"));

      const result = await getFileContent(mockGithub, "owner", "repo", "file.txt", "main");

      expect(result).toBeNull();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch content"));
    });

    it("should handle missing content field", async () => {
      mockGithub.rest.repos.getContent.mockResolvedValueOnce({
        data: {
          type: "file",
          encoding: "base64",
          // content field is missing
        },
      });

      const result = await getFileContent(mockGithub, "owner", "repo", "file.txt", "main");

      expect(result).toBeNull();
    });
  });

  describe("logGraphQLError", () => {
    it("should log operation name and message", () => {
      const error = new Error("Something went wrong");
      logGraphQLError(error, "test operation");

      expect(mockCore.info).toHaveBeenCalledWith("GraphQL error during: test operation");
      expect(mockCore.info).toHaveBeenCalledWith("Message: Something went wrong");
    });

    it("should log errors array with type, path, and locations", () => {
      const error = Object.assign(new Error("GraphQL error"), {
        errors: [
          {
            type: "NOT_FOUND",
            message: "Resource not found",
            path: ["repository", "discussion"],
            locations: [{ line: 1, column: 1 }],
          },
        ],
      });

      logGraphQLError(error, "test");

      expect(mockCore.info).toHaveBeenCalledWith("Errors array (1 error(s)):");
      expect(mockCore.info).toHaveBeenCalledWith("  [1] Resource not found");
      expect(mockCore.info).toHaveBeenCalledWith("      Type: NOT_FOUND");
      expect(mockCore.info).toHaveBeenCalledWith('      Path: ["repository","discussion"]');
    });

    it("should log HTTP status when present", () => {
      const error = Object.assign(new Error("Unauthorized"), { status: 401 });
      logGraphQLError(error, "test");

      expect(mockCore.info).toHaveBeenCalledWith("HTTP status: 401");
    });

    it("should log request and response data when present", () => {
      const error = Object.assign(new Error("Error"), {
        request: { query: "..." },
        data: { repository: null },
      });
      logGraphQLError(error, "test");

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Request:"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Response data:"));
    });

    it("should show insufficientScopesHint when INSUFFICIENT_SCOPES error is present", () => {
      const error = Object.assign(new Error("Scopes error"), {
        errors: [{ type: "INSUFFICIENT_SCOPES", message: "Missing scope" }],
      });

      logGraphQLError(error, "test", {
        insufficientScopesHint: "You need to add permission X.",
      });

      expect(mockCore.info).toHaveBeenCalledWith("You need to add permission X.");
    });

    it("should not show insufficientScopesHint when no INSUFFICIENT_SCOPES error", () => {
      const error = Object.assign(new Error("Other error"), {
        errors: [{ type: "NOT_FOUND", message: "Not found" }],
      });

      logGraphQLError(error, "test", {
        insufficientScopesHint: "You need to add permission X.",
      });

      expect(mockCore.info).not.toHaveBeenCalledWith("You need to add permission X.");
    });

    it("should show notFoundHint when NOT_FOUND error is present and no predicate", () => {
      const error = Object.assign(new Error("Not found"), {
        errors: [{ type: "NOT_FOUND", message: "Resource missing" }],
      });

      logGraphQLError(error, "test", {
        notFoundHint: "Check the resource ID.",
      });

      expect(mockCore.info).toHaveBeenCalledWith("Check the resource ID.");
    });

    it("should show notFoundHint only when notFoundPredicate returns true", () => {
      const errorWithMatch = Object.assign(new Error("projectV2 not found"), {
        errors: [{ type: "NOT_FOUND", message: "projectV2 not found" }],
      });
      const errorNoMatch = Object.assign(new Error("discussion not found"), {
        errors: [{ type: "NOT_FOUND", message: "discussion not found" }],
      });

      const hints = {
        notFoundHint: "Check project settings.",
        notFoundPredicate: /** @param {string} msg */ msg => /projectV2\b/.test(msg),
      };

      logGraphQLError(errorWithMatch, "test", hints);
      expect(mockCore.info).toHaveBeenCalledWith("Check project settings.");

      vi.clearAllMocks();

      logGraphQLError(errorNoMatch, "test", hints);
      expect(mockCore.info).not.toHaveBeenCalledWith("Check project settings.");
    });

    it("should work without hints (no hints argument)", () => {
      const error = Object.assign(new Error("Error"), {
        errors: [{ type: "INSUFFICIENT_SCOPES", message: "Missing scope" }],
      });

      // Should not throw - just logs the generic info without hints
      expect(() => logGraphQLError(error, "test")).not.toThrow();
      expect(mockCore.info).toHaveBeenCalledWith("GraphQL error during: test");
    });
  });
});
