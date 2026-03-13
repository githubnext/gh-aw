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

describe("normalizeBranchName - preserveCase option", () => {
  it("should preserve original casing when preserveCase is true", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("Feature/Add-Login", { preserveCase: true })).toBe("Feature/Add-Login");
    expect(normalizeBranchName("MY-BRANCH", { preserveCase: true })).toBe("MY-BRANCH");
    expect(normalizeBranchName("bugfix/BR-329-red", { preserveCase: true })).toBe("bugfix/BR-329-red");
  });

  it("should still sanitize dangerous characters when preserveCase is true", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("feature; rm -rf /", { preserveCase: true })).toBe("feature-rm-rf-/");
    expect(normalizeBranchName("branch$(malicious)", { preserveCase: true })).toBe("branch-malicious");
    expect(normalizeBranchName("UPPER@CASE", { preserveCase: true })).toBe("UPPER-CASE");
  });

  it("should still collapse dashes and truncate when preserveCase is true", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("Feature---Branch", { preserveCase: true })).toBe("Feature-Branch");
    const longName = "A".repeat(150);
    const result = normalizeBranchName(longName, { preserveCase: true });
    expect(result.length).toBe(128);
  });

  it("should behave the same as default when preserveCase is false or omitted", async () => {
    const { normalizeBranchName } = await import("./normalize_branch_name.cjs");

    expect(normalizeBranchName("Feature/Add-Login", { preserveCase: false })).toBe("feature/add-login");
    expect(normalizeBranchName("Feature/Add-Login")).toBe("feature/add-login");
  });
});
