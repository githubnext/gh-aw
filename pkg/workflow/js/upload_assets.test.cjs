import { describe, it, expect } from "vitest";

/**
 * Normalizes a branch name by removing all characters that are not a-z, A-Z, 0-9, -, _, or /
 * This ensures the branch name is safe for git operations
 * @param {string} branchName - The branch name to normalize
 * @returns {string} The normalized branch name
 */
function normalizeBranchName(branchName) {
  // Remove all characters that are not alphanumeric, dash, underscore, or forward slash
  let normalized = branchName.replace(/[^a-zA-Z0-9\-_/]/g, "");

  // Clean up consecutive slashes
  normalized = normalized.replace(/\/+/g, "/");

  // Remove leading/trailing slashes and dashes
  normalized = normalized.replace(/^[/-]+|[/-]+$/g, "");

  return normalized;
}

describe("normalizeBranchName", () => {
  it("should normalize branch with spaces", () => {
    expect(normalizeBranchName("assets/Documentation Unbloat")).toBe("assets/DocumentationUnbloat");
  });

  it("should normalize branch with multiple spaces", () => {
    expect(normalizeBranchName("assets/My Test Branch Name")).toBe("assets/MyTestBranchName");
  });

  it("should remove special characters", () => {
    expect(normalizeBranchName("assets/test@branch#name")).toBe("assets/testbranchname");
  });

  it("should preserve valid characters", () => {
    expect(normalizeBranchName("assets/valid-branch_name/test")).toBe("assets/valid-branch_name/test");
  });

  it("should clean up consecutive slashes", () => {
    expect(normalizeBranchName("assets//test///branch")).toBe("assets/test/branch");
  });

  it("should remove leading/trailing slashes", () => {
    expect(normalizeBranchName("/assets/test/")).toBe("assets/test");
  });

  it("should remove leading/trailing dashes", () => {
    expect(normalizeBranchName("-assets/test-")).toBe("assets/test");
  });

  it("should handle simple branch names", () => {
    expect(normalizeBranchName("main")).toBe("main");
  });

  it("should remove dots", () => {
    expect(normalizeBranchName("feature/test.branch.name")).toBe("feature/testbranchname");
  });

  it("should remove parentheses", () => {
    expect(normalizeBranchName("assets/test(branch)name")).toBe("assets/testbranchname");
  });

  it("should handle empty string", () => {
    expect(normalizeBranchName("")).toBe("");
  });

  it("should handle only invalid characters", () => {
    expect(normalizeBranchName("@#$%^&*()")).toBe("");
  });
});
