// @ts-check
import { describe, it, expect } from "vitest";
import fs from "fs";
import path from "path";
import { extractBaseBranchFromAgentOutput, isSameWorkflowRepo, isValidBaseBranchName } from "./extract_base_branch_from_agent_output.cjs";

describe("extract_base_branch_from_agent_output", () => {
  it("matches fully-qualified repos", () => {
    expect(isSameWorkflowRepo("owner/repo", "owner/repo")).toBe(true);
  });

  it("matches bare repo names against workflow repo suffix", () => {
    expect(isSameWorkflowRepo("repo", "owner/repo")).toBe(true);
  });

  it("skips cross-repo items", () => {
    expect(isSameWorkflowRepo("other/repo", "owner/repo")).toBe(false);
  });

  it("extracts branch for bare repo match", () => {
    const tmpDir = fs.mkdtempSync(path.join("/tmp", "gh-aw-extract-base-"));
    try {
      const jsonPath = path.join(tmpDir, "agent_output.json");
      fs.writeFileSync(
        jsonPath,
        JSON.stringify({
          items: [
            { type: "create_pull_request", repo: "other/repo", base_branch: "feature/WRONG" },
            { type: "create_pull_request", repo: "repo", base_branch: "feature/CORRECT" },
          ],
        })
      );

      expect(extractBaseBranchFromAgentOutput({ agentOutputPath: jsonPath, workflowRepo: "owner/repo" })).toBe("feature/CORRECT");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("extracts branch when repo is omitted", () => {
    const tmpDir = fs.mkdtempSync(path.join("/tmp", "gh-aw-extract-base-"));
    try {
      const jsonPath = path.join(tmpDir, "agent_output.json");
      fs.writeFileSync(
        jsonPath,
        JSON.stringify({
          items: [{ type: "create_pull_request", base_branch: "feature/DEFAULT-REPO" }],
        })
      );

      expect(extractBaseBranchFromAgentOutput({ agentOutputPath: jsonPath, workflowRepo: "owner/repo" })).toBe("feature/DEFAULT-REPO");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("trims repo values before matching", () => {
    const tmpDir = fs.mkdtempSync(path.join("/tmp", "gh-aw-extract-base-"));
    try {
      const jsonPath = path.join(tmpDir, "agent_output.json");
      fs.writeFileSync(
        jsonPath,
        JSON.stringify({
          items: [{ type: "create_pull_request", repo: " repo ", base_branch: "feature/TRIMMED" }],
        })
      );

      expect(extractBaseBranchFromAgentOutput({ agentOutputPath: jsonPath, workflowRepo: " owner/repo " })).toBe("feature/TRIMMED");
    } finally {
      fs.rmSync(tmpDir, { recursive: true, force: true });
    }
  });

  it("accepts valid git branch names used in safe outputs", () => {
    expect(isValidBaseBranchName("feature/x")).toBe(true);
    expect(isValidBaseBranchName("release/v1.2+hotfix")).toBe(true);
  });

  it("rejects invalid git branch names even if they look regex-safe", () => {
    expect(isValidBaseBranchName("foo..bar")).toBe(false);
    expect(isValidBaseBranchName("main.lock")).toBe(false);
    expect(isValidBaseBranchName(".foo")).toBe(false);
    expect(isValidBaseBranchName("foo/.bar")).toBe(false);
  });

  it("rejects git branch expressions (@{-N} notation)", () => {
    expect(isValidBaseBranchName("@{-1}")).toBe(false);
    expect(isValidBaseBranchName("@{-2}")).toBe(false);
  });

  it("enforces the 255-character length limit", () => {
    const atLimit = "a".repeat(255);
    const overLimit = "a".repeat(256);
    expect(isValidBaseBranchName(atLimit)).toBe(true);
    expect(isValidBaseBranchName(overLimit)).toBe(false);
  });
});
