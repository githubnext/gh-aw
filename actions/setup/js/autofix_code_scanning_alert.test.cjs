// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock @actions/core
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setOutput: vi.fn(),
  setFailed: vi.fn(),
};

// Mock @actions/github
const mockGithub = {
  request: vi.fn(),
};

const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  payload: {},
};

// Set up global mocks
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("autofix_code_scanning_alert handler", () => {
  let handler;

  beforeEach(async () => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;

    // Import the module fresh for each test
    const module = await import("./autofix_code_scanning_alert.cjs");
    handler = await module.main({ max: 10 });
  });

  describe("valid autofix creation", () => {
    it("should create autofix successfully", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 42,
        fix_description: "Fix SQL injection vulnerability",
        fix_code: "const query = db.prepare('SELECT * FROM users WHERE id = ?').bind(userId);",
      };

      mockGithub.request.mockResolvedValue({
        data: {
          id: 123,
          alert_number: 42,
        },
      });

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.alertNumber).toBe(42);
      expect(result.autofixUrl).toContain("security/code-scanning/42");

      expect(mockGithub.request).toHaveBeenCalledWith("POST /repos/{owner}/{repo}/code-scanning/alerts/{alert_number}/fixes", {
        owner: "test-owner",
        repo: "test-repo",
        alert_number: 42,
        fix: {
          description: "Fix SQL injection vulnerability",
          code: "const query = db.prepare('SELECT * FROM users WHERE id = ?').bind(userId);",
        },
        headers: {
          "X-GitHub-Api-Version": "2022-11-28",
        },
      });

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processing autofix_code_scanning_alert"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully created autofix"));
    });

    it("should handle string alert_number", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: "42",
        fix_description: "Fix XSS vulnerability",
        fix_code: "const escaped = escapeHtml(userInput);",
      };

      mockGithub.request.mockResolvedValue({
        data: { id: 123 },
      });

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(result.alertNumber).toBe(42);
    });
  });

  describe("validation errors", () => {
    it("should fail when alert_number is missing", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        fix_description: "Fix vulnerability",
        fix_code: "const fixed = true;",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("alert_number is required");
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("alert_number is missing"));
    });

    it("should fail when fix_description is missing", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 42,
        fix_code: "const fixed = true;",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("fix_description is required");
    });

    it("should fail when fix_code is missing", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 42,
        fix_description: "Fix vulnerability",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("fix_code is required");
    });

    it("should fail with invalid alert_number", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: "invalid",
        fix_description: "Fix vulnerability",
        fix_code: "const fixed = true;",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid alert_number");
    });

    it("should fail with zero alert_number", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 0,
        fix_description: "Fix vulnerability",
        fix_code: "const fixed = true;",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid alert_number");
    });

    it("should fail with negative alert_number", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: -1,
        fix_description: "Fix vulnerability",
        fix_code: "const fixed = true;",
      };

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid alert_number");
    });
  });

  describe("max count enforcement", () => {
    it("should respect max count limit", async () => {
      const handlerWithMax = await (await import("./autofix_code_scanning_alert.cjs")).main({ max: 2 });

      mockGithub.request.mockResolvedValue({ data: {} });

      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 1,
        fix_description: "Fix",
        fix_code: "code",
      };

      // First autofix succeeds
      const result1 = await handlerWithMax(message, {});
      expect(result1.success).toBe(true);

      // Second autofix succeeds
      const result2 = await handlerWithMax({ ...message, alert_number: 2 }, {});
      expect(result2.success).toBe(true);

      // Third autofix fails due to max count
      const result3 = await handlerWithMax({ ...message, alert_number: 3 }, {});
      expect(result3.success).toBe(false);
      expect(result3.error).toContain("Max count of 2 reached");
    });
  });

  describe("staged mode", () => {
    it("should collect autofixes in staged mode", async () => {
      process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

      const stagedHandler = await (await import("./autofix_code_scanning_alert.cjs")).main({ max: 10 });

      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 42,
        fix_description: "Fix vulnerability",
        fix_code: "const fixed = true;",
      };

      const result = await stagedHandler(message, {});

      expect(result.success).toBe(true);
      expect(result.staged).toBe(true);
      expect(result.alertNumber).toBe(42);

      // Should not call the API in staged mode
      expect(mockGithub.request).not.toHaveBeenCalled();
    });
  });

  describe("API error handling", () => {
    it("should handle 404 errors with helpful message", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 999,
        fix_description: "Fix vulnerability",
        fix_code: "const fixed = true;",
      };

      mockGithub.request.mockRejectedValue(new Error("404 Not Found"));

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("404");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Alert 999 not found"));
    });

    it("should handle 403 permission errors", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 42,
        fix_description: "Fix vulnerability",
        fix_code: "const fixed = true;",
      };

      mockGithub.request.mockRejectedValue(new Error("403 Forbidden"));

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Permission denied"));
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("security-events: write"));
    });

    it("should handle 422 validation errors", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 42,
        fix_description: "Fix vulnerability",
        fix_code: "const fixed = true;",
      };

      mockGithub.request.mockRejectedValue(new Error("422 Unprocessable Entity"));

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("422");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Invalid request"));
    });

    it("should handle generic API errors", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 42,
        fix_description: "Fix vulnerability",
        fix_code: "const fixed = true;",
      };

      mockGithub.request.mockRejectedValue(new Error("Network error"));

      const result = await handler(message, {});

      expect(result.success).toBe(false);
      expect(result.error).toContain("Network error");
    });
  });

  describe("extensive logging", () => {
    it("should log all relevant information", async () => {
      const message = {
        type: "autofix_code_scanning_alert",
        alert_number: 42,
        fix_description: "Fix SQL injection vulnerability by using prepared statements",
        fix_code: "const query = db.prepare('SELECT * FROM users WHERE id = ?').bind(userId);",
      };

      mockGithub.request.mockResolvedValue({
        data: { id: 123 },
      });

      await handler(message, {});

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processing autofix_code_scanning_alert"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("alert_number=42"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Creating autofix for code scanning alert 42"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Fix description:"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Fix code length:"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully created autofix"));
    });
  });
});
