// @ts-check
import { describe, it, expect } from "vitest";
import fs from "fs";
import path from "path";
import { extractBaseBranchFromAgentOutput, isSameWorkflowRepo } from "./extract_base_branch_from_agent_output.cjs";

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
});
