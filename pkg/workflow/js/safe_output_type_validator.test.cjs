import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock core global for tests
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
};
global.core = mockCore;

describe("safe_output_type_validator.cjs", () => {
  let validator;

  beforeEach(async () => {
    vi.clearAllMocks();
    // Import fresh copy of the module
    validator = await import("./safe_output_type_validator.cjs");
  });

  describe("getMaxAllowedForType", () => {
    it("should return default max for known types", () => {
      expect(validator.getMaxAllowedForType("create_issue", {})).toBe(1);
      expect(validator.getMaxAllowedForType("add_labels", {})).toBe(5);
      expect(validator.getMaxAllowedForType("missing_tool", {})).toBe(20);
      expect(validator.getMaxAllowedForType("create_code_scanning_alert", {})).toBe(40);
      expect(validator.getMaxAllowedForType("link_sub_issue", {})).toBe(5);
    });

    it("should return config max when provided", () => {
      const config = { create_issue: { max: 5 } };
      expect(validator.getMaxAllowedForType("create_issue", config)).toBe(5);
    });

    it("should return 1 for unknown types", () => {
      expect(validator.getMaxAllowedForType("unknown_type", {})).toBe(1);
    });
  });

  describe("getMinRequiredForType", () => {
    it("should return 0 by default", () => {
      expect(validator.getMinRequiredForType("create_issue", {})).toBe(0);
    });

    it("should return config min when provided", () => {
      const config = { create_issue: { min: 1 } };
      expect(validator.getMinRequiredForType("create_issue", config)).toBe(1);
    });
  });

  describe("validatePositiveInteger", () => {
    it("should validate positive integers", () => {
      const result = validator.validatePositiveInteger(5, "test_field", 1);
      expect(result.isValid).toBe(true);
      expect(result.normalizedValue).toBe(5);
    });

    it("should validate string numbers", () => {
      const result = validator.validatePositiveInteger("10", "test_field", 1);
      expect(result.isValid).toBe(true);
      expect(result.normalizedValue).toBe(10);
    });

    it("should reject null/undefined", () => {
      const result = validator.validatePositiveInteger(null, "test_field", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("is required");
    });

    it("should reject zero", () => {
      const result = validator.validatePositiveInteger(0, "test_field", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("must be a valid positive integer");
    });

    it("should reject negative numbers", () => {
      const result = validator.validatePositiveInteger(-5, "test_field", 1);
      expect(result.isValid).toBe(false);
    });

    it("should reject non-integer numbers", () => {
      const result = validator.validatePositiveInteger(1.5, "test_field", 1);
      expect(result.isValid).toBe(false);
    });
  });

  describe("validateOptionalPositiveInteger", () => {
    it("should accept undefined", () => {
      const result = validator.validateOptionalPositiveInteger(undefined, "test_field", 1);
      expect(result.isValid).toBe(true);
    });

    it("should validate positive integers", () => {
      const result = validator.validateOptionalPositiveInteger(5, "test_field", 1);
      expect(result.isValid).toBe(true);
      expect(result.normalizedValue).toBe(5);
    });

    it("should reject invalid values when present", () => {
      const result = validator.validateOptionalPositiveInteger(-1, "test_field", 1);
      expect(result.isValid).toBe(false);
    });
  });

  describe("validateIssueOrPRNumber", () => {
    it("should accept undefined", () => {
      const result = validator.validateIssueOrPRNumber(undefined, "test_field", 1);
      expect(result.isValid).toBe(true);
    });

    it("should accept numbers", () => {
      const result = validator.validateIssueOrPRNumber(123, "test_field", 1);
      expect(result.isValid).toBe(true);
    });

    it("should accept strings", () => {
      const result = validator.validateIssueOrPRNumber("123", "test_field", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject invalid types", () => {
      const result = validator.validateIssueOrPRNumber({}, "test_field", 1);
      expect(result.isValid).toBe(false);
    });
  });

  describe("validateIssueNumberOrTemporaryId", () => {
    it("should accept positive integers", () => {
      const result = validator.validateIssueNumberOrTemporaryId(123, "test_field", 1);
      expect(result.isValid).toBe(true);
      expect(result.normalizedValue).toBe(123);
      expect(result.isTemporary).toBe(false);
    });

    it("should accept temporary IDs", () => {
      const result = validator.validateIssueNumberOrTemporaryId("aw_abc123def456", "test_field", 1);
      expect(result.isValid).toBe(true);
      expect(result.normalizedValue).toBe("aw_abc123def456");
      expect(result.isTemporary).toBe(true);
    });

    it("should normalize temporary ID case", () => {
      const result = validator.validateIssueNumberOrTemporaryId("AW_ABC123DEF456", "test_field", 1);
      expect(result.isValid).toBe(true);
      expect(result.normalizedValue).toBe("aw_abc123def456");
    });

    it("should reject null/undefined", () => {
      const result = validator.validateIssueNumberOrTemporaryId(null, "test_field", 1);
      expect(result.isValid).toBe(false);
    });

    it("should reject invalid strings", () => {
      const result = validator.validateIssueNumberOrTemporaryId("invalid", "test_field", 1);
      expect(result.isValid).toBe(false);
    });
  });

  describe("validateItem - create_issue", () => {
    it("should validate valid create_issue item", () => {
      const item = {
        type: "create_issue",
        title: "Test Issue",
        body: "Test body content",
      };
      const result = validator.validateItem(item, "create_issue", 1);
      expect(result.isValid).toBe(true);
      expect(result.normalizedItem.title).toBe("Test Issue");
      expect(result.normalizedItem.body).toBe("Test body content");
    });

    it("should reject create_issue without title", () => {
      const item = {
        type: "create_issue",
        body: "Test body content",
      };
      const result = validator.validateItem(item, "create_issue", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("title");
    });

    it("should reject create_issue without body", () => {
      const item = {
        type: "create_issue",
        title: "Test Issue",
      };
      const result = validator.validateItem(item, "create_issue", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("body");
    });

    it("should sanitize labels array", () => {
      const item = {
        type: "create_issue",
        title: "Test Issue",
        body: "Test body",
        labels: ["bug", "feature"],
      };
      const result = validator.validateItem(item, "create_issue", 1);
      expect(result.isValid).toBe(true);
      expect(result.normalizedItem.labels).toEqual(["bug", "feature"]);
    });

    it("should validate parent field", () => {
      const item = {
        type: "create_issue",
        title: "Test Issue",
        body: "Test body",
        parent: 123,
      };
      const result = validator.validateItem(item, "create_issue", 1);
      expect(result.isValid).toBe(true);
    });
  });

  describe("validateItem - add_labels", () => {
    it("should validate valid add_labels item", () => {
      const item = {
        type: "add_labels",
        labels: ["bug", "enhancement"],
      };
      const result = validator.validateItem(item, "add_labels", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject add_labels without labels array", () => {
      const item = {
        type: "add_labels",
      };
      const result = validator.validateItem(item, "add_labels", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("labels");
    });

    it("should reject add_labels with non-string items", () => {
      const item = {
        type: "add_labels",
        labels: ["bug", 123],
      };
      const result = validator.validateItem(item, "add_labels", 1);
      expect(result.isValid).toBe(false);
    });
  });

  describe("validateItem - update_issue", () => {
    it("should validate update_issue with status", () => {
      const item = {
        type: "update_issue",
        status: "closed",
      };
      const result = validator.validateItem(item, "update_issue", 1);
      expect(result.isValid).toBe(true);
    });

    it("should validate update_issue with title", () => {
      const item = {
        type: "update_issue",
        title: "New Title",
      };
      const result = validator.validateItem(item, "update_issue", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject update_issue without any valid field", () => {
      const item = {
        type: "update_issue",
      };
      const result = validator.validateItem(item, "update_issue", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("at least one of");
    });

    it("should validate status enum values", () => {
      const item = { type: "update_issue", status: "invalid" };
      const result = validator.validateItem(item, "update_issue", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("must be 'open' or 'closed'");
    });
  });

  describe("validateItem - update_pull_request", () => {
    it("should validate update_pull_request with title", () => {
      const item = {
        type: "update_pull_request",
        title: "New PR Title",
      };
      const result = validator.validateItem(item, "update_pull_request", 1);
      expect(result.isValid).toBe(true);
    });

    it("should validate operation enum", () => {
      const item = {
        type: "update_pull_request",
        body: "New content",
        operation: "append",
      };
      const result = validator.validateItem(item, "update_pull_request", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject without title or body", () => {
      const item = {
        type: "update_pull_request",
        operation: "replace",
      };
      const result = validator.validateItem(item, "update_pull_request", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("at least one of");
    });
  });

  describe("validateItem - create_pull_request_review_comment", () => {
    it("should validate valid review comment", () => {
      const item = {
        type: "create_pull_request_review_comment",
        path: "src/file.js",
        line: 10,
        body: "Comment text",
      };
      const result = validator.validateItem(item, "create_pull_request_review_comment", 1);
      expect(result.isValid).toBe(true);
    });

    it("should validate start_line less than line", () => {
      const item = {
        type: "create_pull_request_review_comment",
        path: "src/file.js",
        line: 10,
        start_line: 5,
        body: "Comment text",
      };
      const result = validator.validateItem(item, "create_pull_request_review_comment", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject start_line greater than line", () => {
      const item = {
        type: "create_pull_request_review_comment",
        path: "src/file.js",
        line: 5,
        start_line: 10,
        body: "Comment text",
      };
      const result = validator.validateItem(item, "create_pull_request_review_comment", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("start_line");
    });

    it("should validate side enum", () => {
      const item = {
        type: "create_pull_request_review_comment",
        path: "src/file.js",
        line: 10,
        body: "Comment text",
        side: "LEFT",
      };
      const result = validator.validateItem(item, "create_pull_request_review_comment", 1);
      expect(result.isValid).toBe(true);
    });
  });

  describe("validateItem - create_code_scanning_alert", () => {
    it("should validate valid code scanning alert", () => {
      const item = {
        type: "create_code_scanning_alert",
        file: "src/auth.js",
        line: 42,
        severity: "error",
        message: "SQL injection vulnerability",
      };
      const result = validator.validateItem(item, "create_code_scanning_alert", 1);
      expect(result.isValid).toBe(true);
    });

    it("should validate severity enum (case-insensitive)", () => {
      const item = {
        type: "create_code_scanning_alert",
        file: "src/auth.js",
        line: 42,
        severity: "WARNING",
        message: "Potential issue",
      };
      const result = validator.validateItem(item, "create_code_scanning_alert", 1);
      expect(result.isValid).toBe(true);
      expect(result.normalizedItem.severity).toBe("warning");
    });

    it("should validate ruleIdSuffix pattern", () => {
      const item = {
        type: "create_code_scanning_alert",
        file: "src/auth.js",
        line: 42,
        severity: "error",
        message: "Issue",
        ruleIdSuffix: "sql-injection",
      };
      const result = validator.validateItem(item, "create_code_scanning_alert", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject invalid ruleIdSuffix pattern", () => {
      const item = {
        type: "create_code_scanning_alert",
        file: "src/auth.js",
        line: 42,
        severity: "error",
        message: "Issue",
        ruleIdSuffix: "invalid rule!",
      };
      const result = validator.validateItem(item, "create_code_scanning_alert", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("alphanumeric");
    });
  });

  describe("validateItem - link_sub_issue", () => {
    it("should validate link_sub_issue with numbers", () => {
      const item = {
        type: "link_sub_issue",
        parent_issue_number: 100,
        sub_issue_number: 101,
      };
      const result = validator.validateItem(item, "link_sub_issue", 1);
      expect(result.isValid).toBe(true);
    });

    it("should validate link_sub_issue with temporary IDs", () => {
      const item = {
        type: "link_sub_issue",
        parent_issue_number: "aw_abc123def456",
        sub_issue_number: 101,
      };
      const result = validator.validateItem(item, "link_sub_issue", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject same parent and sub issue", () => {
      const item = {
        type: "link_sub_issue",
        parent_issue_number: 100,
        sub_issue_number: 100,
      };
      const result = validator.validateItem(item, "link_sub_issue", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("must be different");
    });

    it("should reject same temporary IDs (case insensitive)", () => {
      const item = {
        type: "link_sub_issue",
        parent_issue_number: "aw_abc123def456",
        sub_issue_number: "AW_ABC123DEF456",
      };
      const result = validator.validateItem(item, "link_sub_issue", 1);
      expect(result.isValid).toBe(false);
      expect(result.error).toContain("must be different");
    });
  });

  describe("validateItem - close_discussion", () => {
    it("should validate valid close_discussion", () => {
      const item = {
        type: "close_discussion",
        body: "Closing this discussion",
      };
      const result = validator.validateItem(item, "close_discussion", 1);
      expect(result.isValid).toBe(true);
    });

    it("should validate reason enum", () => {
      const item = {
        type: "close_discussion",
        body: "Closing this discussion",
        reason: "RESOLVED",
      };
      const result = validator.validateItem(item, "close_discussion", 1);
      expect(result.isValid).toBe(true);
    });

    it("should normalize reason case", () => {
      const item = {
        type: "close_discussion",
        body: "Closing this discussion",
        reason: "resolved",
      };
      const result = validator.validateItem(item, "close_discussion", 1);
      expect(result.isValid).toBe(true);
      // Enum values are returned as defined in the config
      expect(result.normalizedItem.reason).toBe("RESOLVED");
    });
  });

  describe("hasValidationConfig", () => {
    it("should return true for known types", () => {
      expect(validator.hasValidationConfig("create_issue")).toBe(true);
      expect(validator.hasValidationConfig("add_labels")).toBe(true);
      expect(validator.hasValidationConfig("link_sub_issue")).toBe(true);
    });

    it("should return false for unknown types", () => {
      expect(validator.hasValidationConfig("unknown_type")).toBe(false);
    });
  });

  describe("getKnownTypes", () => {
    it("should return all known types", () => {
      const types = validator.getKnownTypes();
      expect(types).toContain("create_issue");
      expect(types).toContain("add_labels");
      expect(types).toContain("create_pull_request");
      expect(types).toContain("link_sub_issue");
      expect(types.length).toBeGreaterThan(10);
    });
  });

  describe("validateItem - missing_tool", () => {
    it("should validate valid missing_tool item", () => {
      const item = {
        type: "missing_tool",
        tool: "some_tool",
        reason: "It's needed for X",
      };
      const result = validator.validateItem(item, "missing_tool", 1);
      expect(result.isValid).toBe(true);
    });

    it("should validate with alternatives", () => {
      const item = {
        type: "missing_tool",
        tool: "some_tool",
        reason: "It's needed for X",
        alternatives: "Use Y instead",
      };
      const result = validator.validateItem(item, "missing_tool", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject without tool", () => {
      const item = {
        type: "missing_tool",
        reason: "It's needed for X",
      };
      const result = validator.validateItem(item, "missing_tool", 1);
      expect(result.isValid).toBe(false);
    });
  });

  describe("validateItem - noop", () => {
    it("should validate valid noop item", () => {
      const item = {
        type: "noop",
        message: "No action needed",
      };
      const result = validator.validateItem(item, "noop", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject without message", () => {
      const item = {
        type: "noop",
      };
      const result = validator.validateItem(item, "noop", 1);
      expect(result.isValid).toBe(false);
    });
  });

  describe("validateItem - upload_asset", () => {
    it("should validate valid upload_asset item", () => {
      const item = {
        type: "upload_asset",
        path: "/tmp/image.png",
      };
      const result = validator.validateItem(item, "upload_asset", 1);
      expect(result.isValid).toBe(true);
    });

    it("should reject without path", () => {
      const item = {
        type: "upload_asset",
      };
      const result = validator.validateItem(item, "upload_asset", 1);
      expect(result.isValid).toBe(false);
    });
  });
});
