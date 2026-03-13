import { describe, it, expect } from "vitest";

describe("normalizeBranchName", () => {
  it("should handle valid branch names", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("feature/add-login")).toBe("feature/add-login");
    expect(normalizeBranchName("my-branch")).toBe("my-branch");
    expect(normalizeBranchName("v1.0.0")).toBe("v1.0.0");
  });

  it("should replace invalid characters with dashes", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("feature@test")).toBe("feature-test");
    expect(normalizeBranchName("branch#with#hashes")).toBe("branch-with-hashes");
    expect(normalizeBranchName("test branch name")).toBe("test-branch-name");
  });

  it("should collapse multiple dashes", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("test---branch")).toBe("test-branch");
    expect(normalizeBranchName("a--b--c")).toBe("a-b-c");
  });

  it("should remove leading and trailing dashes", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("-test-branch-")).toBe("test-branch");
    expect(normalizeBranchName("---test---")).toBe("test");
  });

  it("should truncate to 128 characters", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    const longName = "a".repeat(150);
    const result = normalizeBranchName(longName);
    expect(result.length).toBe(128);
    expect(result).toBe("a".repeat(128));
  });

  it("should convert to lowercase", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("Feature/Add-Login")).toBe("feature/add-login");
    expect(normalizeBranchName("MY-BRANCH")).toBe("my-branch");
  });

  it("should handle empty and invalid inputs", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("")).toBe("");
    expect(normalizeBranchName("   ")).toBe("   ");
    expect(normalizeBranchName(null)).toBe(null);
    expect(normalizeBranchName(undefined)).toBe(undefined);
  });

  it("should preserve valid special characters", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("feature/test_branch-v1.0")).toBe("feature/test_branch-v1.0");
    expect(normalizeBranchName("my_branch-123")).toBe("my_branch-123");
  });

  it("should handle complex combinations", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("Feature@Test/Branch#123")).toBe("feature-test/branch-123");
    expect(normalizeBranchName("__test__branch__")).toBe("__test__branch__");
  });

  it("should remove trailing dashes after truncation", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    // Create a string that will end with a dash after truncation
    const longName = "a".repeat(127) + "-b";
    const result = normalizeBranchName(longName);
    expect(result.length).toBeLessThanOrEqual(128);
    expect(result).not.toMatch(/-$/);
  });
});

describe("sanitizeBranchNamePreserveCase", () => {
  it("should preserve valid branch names with original casing", async () => {
    const { sanitizeBranchNamePreserveCase } = await import("./normalize_branch_name.cjs");

    expect(sanitizeBranchNamePreserveCase("bugfix/BR-329-red")).toBe("bugfix/BR-329-red");
    expect(sanitizeBranchNamePreserveCase("feature/JIRA-123-MyFeature")).toBe("feature/JIRA-123-MyFeature");
    expect(sanitizeBranchNamePreserveCase("hotfix/UPPERCASE")).toBe("hotfix/UPPERCASE");
  });

  it("should replace invalid characters with dashes", async () => {
    const { sanitizeBranchNamePreserveCase } = await import("./normalize_branch_name.cjs");

    expect(sanitizeBranchNamePreserveCase("Feature@Test")).toBe("Feature-Test");
    expect(sanitizeBranchNamePreserveCase("branch; rm -rf /")).toBe("branch-rm-rf-/");
    expect(sanitizeBranchNamePreserveCase("branch$(malicious)")).toBe("branch-malicious");
  });

  it("should NOT convert to lowercase", async () => {
    const { sanitizeBranchNamePreserveCase } = await import("./normalize_branch_name.cjs");

    expect(sanitizeBranchNamePreserveCase("Feature/Add-Login")).toBe("Feature/Add-Login");
    expect(sanitizeBranchNamePreserveCase("MY-BRANCH")).toBe("MY-BRANCH");
    expect(sanitizeBranchNamePreserveCase("bugfix/BR-329-red")).toBe("bugfix/BR-329-red");
  });

  it("should collapse multiple dashes", async () => {
    const { sanitizeBranchNamePreserveCase } = await import("./normalize_branch_name.cjs");

    expect(sanitizeBranchNamePreserveCase("Test---Branch")).toBe("Test-Branch");
    expect(sanitizeBranchNamePreserveCase("A--B--C")).toBe("A-B-C");
  });

  it("should remove leading and trailing dashes", async () => {
    const { sanitizeBranchNamePreserveCase } = await import("./normalize_branch_name.cjs");

    expect(sanitizeBranchNamePreserveCase("-Test-Branch-")).toBe("Test-Branch");
    expect(sanitizeBranchNamePreserveCase("---Test---")).toBe("Test");
  });

  it("should truncate to 128 characters", async () => {
    const { sanitizeBranchNamePreserveCase } = await import("./normalize_branch_name.cjs");

    const longName = "A".repeat(150);
    const result = sanitizeBranchNamePreserveCase(longName);
    expect(result.length).toBe(128);
    expect(result).toBe("A".repeat(128));
  });

  it("should handle empty and invalid inputs", async () => {
    const { sanitizeBranchNamePreserveCase } = await import("./normalize_branch_name.cjs");

    expect(sanitizeBranchNamePreserveCase("")).toBe("");
    expect(sanitizeBranchNamePreserveCase("   ")).toBe("   ");
    expect(sanitizeBranchNamePreserveCase(null)).toBe(null);
    expect(sanitizeBranchNamePreserveCase(undefined)).toBe(undefined);
  });
});
