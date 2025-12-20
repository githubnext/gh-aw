import { describe, it, expect, beforeEach, afterEach } from "vitest";

describe("validate_environment.cjs", () => {
  let validateEnvironment;
  let originalEnv;

  beforeEach(async () => {
    // Save original environment
    originalEnv = { ...process.env };

    // Import the module
    const module = await import("./validate_environment.cjs");
    validateEnvironment = module.validateEnvironment;
  });

  afterEach(() => {
    // Restore original environment
    process.env = originalEnv;
  });

  it("should not throw when all required variables are present", () => {
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    process.env.GITHUB_TOKEN = "test-token";

    expect(() => {
      validateEnvironment(["GH_AW_WORKFLOW_ID", "GITHUB_TOKEN"]);
    }).not.toThrow();
  });

  it("should throw when a single required variable is missing", () => {
    delete process.env.GH_AW_WORKFLOW_ID;

    expect(() => {
      validateEnvironment(["GH_AW_WORKFLOW_ID"]);
    }).toThrow(/Missing required environment variable: GH_AW_WORKFLOW_ID/);
  });

  it("should throw when multiple required variables are missing", () => {
    delete process.env.GH_AW_WORKFLOW_ID;
    delete process.env.GITHUB_TOKEN;

    expect(() => {
      validateEnvironment(["GH_AW_WORKFLOW_ID", "GITHUB_TOKEN"]);
    }).toThrow(/Missing required environment variables: GH_AW_WORKFLOW_ID, GITHUB_TOKEN/);
  });

  it("should include helpful message about safe_outputs configuration", () => {
    delete process.env.GH_AW_WORKFLOW_ID;

    expect(() => {
      validateEnvironment(["GH_AW_WORKFLOW_ID"]);
    }).toThrow(/Please ensure these are set in the safe_outputs job configuration/);
  });

  it("should treat empty string as missing", () => {
    process.env.GH_AW_WORKFLOW_ID = "";

    expect(() => {
      validateEnvironment(["GH_AW_WORKFLOW_ID"]);
    }).toThrow(/Missing required environment variable: GH_AW_WORKFLOW_ID/);
  });

  it("should treat whitespace-only string as missing", () => {
    process.env.GH_AW_WORKFLOW_ID = "   ";

    expect(() => {
      validateEnvironment(["GH_AW_WORKFLOW_ID"]);
    }).toThrow(/Missing required environment variable: GH_AW_WORKFLOW_ID/);
  });

  it("should not throw when required array is empty", () => {
    expect(() => {
      validateEnvironment([]);
    }).not.toThrow();
  });

  it("should handle mix of present and missing variables", () => {
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    delete process.env.GH_AW_BASE_BRANCH;

    expect(() => {
      validateEnvironment(["GH_AW_WORKFLOW_ID", "GH_AW_BASE_BRANCH"]);
    }).toThrow(/Missing required environment variable: GH_AW_BASE_BRANCH/);
  });

  it("should not include present variables in error message", () => {
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
    delete process.env.GH_AW_BASE_BRANCH;

    try {
      validateEnvironment(["GH_AW_WORKFLOW_ID", "GH_AW_BASE_BRANCH"]);
      expect.fail("Should have thrown an error");
    } catch (error) {
      expect(error.message).not.toContain("GH_AW_WORKFLOW_ID");
      expect(error.message).toContain("GH_AW_BASE_BRANCH");
    }
  });

  it("should handle validation with a single variable", () => {
    process.env.GITHUB_TOKEN = "test-token";

    expect(() => {
      validateEnvironment(["GITHUB_TOKEN"]);
    }).not.toThrow();
  });

  it("should handle validation with many variables", () => {
    process.env.VAR1 = "value1";
    process.env.VAR2 = "value2";
    process.env.VAR3 = "value3";
    process.env.VAR4 = "value4";
    process.env.VAR5 = "value5";

    expect(() => {
      validateEnvironment(["VAR1", "VAR2", "VAR3", "VAR4", "VAR5"]);
    }).not.toThrow();
  });

  it("should report all missing variables when multiple are absent", () => {
    delete process.env.VAR1;
    delete process.env.VAR2;
    delete process.env.VAR3;

    try {
      validateEnvironment(["VAR1", "VAR2", "VAR3"]);
      expect.fail("Should have thrown an error");
    } catch (error) {
      expect(error.message).toContain("VAR1");
      expect(error.message).toContain("VAR2");
      expect(error.message).toContain("VAR3");
    }
  });
});
