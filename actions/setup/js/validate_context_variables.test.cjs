// @ts-check

import { describe, it, expect, beforeEach, vi } from "vitest";

describe("validateNumericValue", () => {
  let validateNumericValue;

  beforeEach(async () => {
    const module = await import("./validate_context_variables.cjs");
    validateNumericValue = module.validateNumericValue;
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

  it("should accept null values", () => {
    const result = validateNumericValue(null, "TEST_VAR");
    expect(result.valid).toBe(true);
    expect(result.message).toContain("empty");
  });

  it("should accept valid positive integers as numbers", () => {
    const result = validateNumericValue(12345, "ISSUE_NUMBER");
    expect(result.valid).toBe(true);
    expect(result.message).toContain("valid");
    expect(result.message).toContain("12345");
  });

  it("should accept valid positive integers as strings", () => {
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

describe("getNestedValue", () => {
  let getNestedValue;

  beforeEach(async () => {
    const module = await import("./validate_context_variables.cjs");
    getNestedValue = module.getNestedValue;
  });

  it("should get nested values from objects", () => {
    const obj = {
      payload: {
        issue: {
          number: 123,
        },
      },
    };

    const result = getNestedValue(obj, ["payload", "issue", "number"]);
    expect(result).toBe(123);
  });

  it("should return undefined for missing paths", () => {
    const obj = {
      payload: {},
    };

    const result = getNestedValue(obj, ["payload", "issue", "number"]);
    expect(result).toBeUndefined();
  });

  it("should return undefined for null/undefined intermediate values", () => {
    const obj = {
      payload: null,
    };

    const result = getNestedValue(obj, ["payload", "issue", "number"]);
    expect(result).toBeUndefined();
  });

  it("should handle empty path", () => {
    const obj = { value: 42 };
    const result = getNestedValue(obj, []);
    expect(result).toEqual(obj);
  });
});

describe("NUMERIC_CONTEXT_PATHS", () => {
  let NUMERIC_CONTEXT_PATHS;

  beforeEach(async () => {
    const module = await import("./validate_context_variables.cjs");
    NUMERIC_CONTEXT_PATHS = module.NUMERIC_CONTEXT_PATHS;
  });

  it("should include all expected numeric variables", () => {
    const expectedPaths = [
      { path: ["payload", "issue", "number"], name: "github.event.issue.number" },
      { path: ["payload", "pull_request", "number"], name: "github.event.pull_request.number" },
      { path: ["payload", "discussion", "number"], name: "github.event.discussion.number" },
      { path: ["run_id"], name: "github.run_id" },
      { path: ["run_number"], name: "github.run_number" },
    ];

    expectedPaths.forEach(expected => {
      const found = NUMERIC_CONTEXT_PATHS.find(p => p.name === expected.name);
      expect(found).toBeDefined();
      expect(found.path).toEqual(expected.path);
    });
  });

  it("should have 31 context paths", () => {
    expect(NUMERIC_CONTEXT_PATHS.length).toBe(31);
  });

  it("should not include duplicate names", () => {
    const names = NUMERIC_CONTEXT_PATHS.map(p => p.name);
    const uniqueNames = [...new Set(names)];
    expect(uniqueNames.length).toBe(NUMERIC_CONTEXT_PATHS.length);
  });
});

describe("main", () => {
  let main;
  let mockCore;
  let mockContext;

  beforeEach(async () => {
    vi.resetModules();

    mockCore = {
      info: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
    };

    mockContext = {
      payload: {
        issue: {
          number: 123,
        },
        pull_request: {
          number: 456,
        },
      },
      run_id: 789,
      run_number: 10,
    };

    global.core = mockCore;
    global.context = mockContext;

    const module = await import("./validate_context_variables.cjs");
    main = module.main;
  });

  afterEach(() => {
    delete global.core;
    delete global.context;
  });

  it("should validate all numeric context variables successfully", async () => {
    await main();

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("✅ All context variables validated successfully"));
  });

  it("should fail when a numeric field contains non-numeric data", async () => {
    mockContext.payload.issue.number = "123; DROP TABLE users";

    await expect(main()).rejects.toThrow();
    expect(mockCore.setFailed).toHaveBeenCalled();
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("non-numeric"));
  });

  it("should pass when numeric fields are valid integers", async () => {
    mockContext.payload.issue.number = 42;
    mockContext.run_id = 12345;

    await main();

    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });
});
