// @ts-check

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("validateNumericValue", () => {
  let validateNumericValue;
  let NUMERIC_CONTEXT_VARS;

  beforeEach(async () => {
    const module = await import("./validate_context_variables.cjs");
    validateNumericValue = module.validateNumericValue;
    NUMERIC_CONTEXT_VARS = module.NUMERIC_CONTEXT_VARS;
  });

  it("should accept empty values", () => {
    const result = validateNumericValue("", "TEST_VAR");
    expect(result.valid).toBe(true);
    expect(result.message).toContain("empty");
  });

  it("should accept undefined values", () => {
    const result = validateNumericValue(undefined, "TEST_VAR");
    expect(result.valid).toBe(true);
    expect(result.message).toContain("empty");
  });

  it("should accept whitespace-only values", () => {
    const result = validateNumericValue("   ", "TEST_VAR");
    expect(result.valid).toBe(true);
    expect(result.message).toContain("empty");
  });

  it("should accept valid positive integers", () => {
    const result = validateNumericValue("12345", "ISSUE_NUMBER");
    expect(result.valid).toBe(true);
    expect(result.message).toContain("valid");
    expect(result.message).toContain("12345");
  });

  it("should accept valid negative integers", () => {
    const result = validateNumericValue("-42", "TEST_VAR");
    expect(result.valid).toBe(true);
    expect(result.message).toContain("valid");
  });

  it("should accept zero", () => {
    const result = validateNumericValue("0", "TEST_VAR");
    expect(result.valid).toBe(true);
    expect(result.message).toContain("valid");
  });

  it("should accept integers with leading/trailing whitespace", () => {
    const result = validateNumericValue("  42  ", "TEST_VAR");
    expect(result.valid).toBe(true);
    expect(result.message).toContain("valid");
  });

  it("should reject strings with letters", () => {
    const result = validateNumericValue("abc123", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject strings with special characters", () => {
    const result = validateNumericValue("123$456", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject strings with injection attempts", () => {
    const result = validateNumericValue("123; rm -rf /", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject floating point numbers", () => {
    const result = validateNumericValue("123.456", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject numbers with commas", () => {
    const result = validateNumericValue("1,234", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject scientific notation", () => {
    const result = validateNumericValue("1e5", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject hex numbers", () => {
    const result = validateNumericValue("0x123", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject octal numbers", () => {
    const result = validateNumericValue("0o777", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject binary numbers", () => {
    const result = validateNumericValue("0b1010", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject numbers with spaces in the middle", () => {
    const result = validateNumericValue("12 34", "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("non-numeric");
  });

  it("should reject malicious payloads", () => {
    const maliciousPayloads = ["'; DROP TABLE users; --", "<script>alert('xss')</script>", "${7*7}", "{{constructor.constructor('alert(1)')()}}", "../../../etc/passwd", "$(whoami)", "`ls -la`"];

    maliciousPayloads.forEach(payload => {
      const result = validateNumericValue(payload, "TEST_VAR");
      expect(result.valid).toBe(false);
      expect(result.message).toContain("non-numeric");
    });
  });

  it("should reject extremely large numbers outside safe integer range", () => {
    const tooLarge = "9007199254740992"; // Number.MAX_SAFE_INTEGER + 1
    const result = validateNumericValue(tooLarge, "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("outside safe integer range");
  });

  it("should reject extremely small numbers outside safe integer range", () => {
    const tooSmall = "-9007199254740992"; // Number.MIN_SAFE_INTEGER - 1
    const result = validateNumericValue(tooSmall, "TEST_VAR");
    expect(result.valid).toBe(false);
    expect(result.message).toContain("outside safe integer range");
  });

  it("should accept numbers at the edge of safe integer range", () => {
    const maxSafe = "9007199254740991"; // Number.MAX_SAFE_INTEGER
    const result = validateNumericValue(maxSafe, "TEST_VAR");
    expect(result.valid).toBe(true);

    const minSafe = "-9007199254740991"; // Number.MIN_SAFE_INTEGER
    const result2 = validateNumericValue(minSafe, "TEST_VAR");
    expect(result2.valid).toBe(true);
  });
});

describe("NUMERIC_CONTEXT_VARS", () => {
  let NUMERIC_CONTEXT_VARS;

  beforeEach(async () => {
    const module = await import("./validate_context_variables.cjs");
    NUMERIC_CONTEXT_VARS = module.NUMERIC_CONTEXT_VARS;
  });

  it("should include all expected numeric variables", () => {
    const expectedVars = [
      "ISSUE_NUMBER",
      "PULL_REQUEST_NUMBER",
      "DISCUSSION_NUMBER",
      "MILESTONE_NUMBER",
      "CHECK_RUN_NUMBER",
      "CHECK_SUITE_NUMBER",
      "WORKFLOW_RUN_NUMBER",
      "CHECK_RUN_ID",
      "CHECK_SUITE_ID",
      "COMMENT_ID",
      "DEPLOYMENT_ID",
      "DEPLOYMENT_STATUS_ID",
      "HEAD_COMMIT_ID",
      "INSTALLATION_ID",
      "WORKFLOW_JOB_RUN_ID",
      "LABEL_ID",
      "MILESTONE_ID",
      "ORGANIZATION_ID",
      "PAGE_ID",
      "PROJECT_ID",
      "PROJECT_CARD_ID",
      "PROJECT_COLUMN_ID",
      "RELEASE_ID",
      "REPOSITORY_ID",
      "REVIEW_ID",
      "REVIEW_COMMENT_ID",
      "SENDER_ID",
      "WORKFLOW_RUN_ID",
      "WORKFLOW_JOB_ID",
      "RUN_ID",
      "RUN_NUMBER",
    ];

    expectedVars.forEach(varName => {
      expect(NUMERIC_CONTEXT_VARS).toContain(varName);
    });
  });

  it("should not include duplicate entries", () => {
    const uniqueVars = [...new Set(NUMERIC_CONTEXT_VARS)];
    expect(uniqueVars.length).toBe(NUMERIC_CONTEXT_VARS.length);
  });
});
