import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() } },
  mockContext = { repo: { owner: "test-owner", repo: "test-repo" }, eventName: "issues", payload: { issue: { number: 123 } } },
  mockGithub = { rest: { issues: { update: vi.fn(), listMilestones: vi.fn() } } };
((global.core = mockCore),
  (global.context = mockContext),
  (global.github = mockGithub),
  describe("assign_milestone", () => {
    let assignMilestoneScript, tempFilePath;
    const setAgentOutput = data => {
      tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
      const content = "string" == typeof data ? data : JSON.stringify(data);
      (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
    };
    (beforeEach(() => {
      (vi.clearAllMocks(), delete process.env.GH_AW_AGENT_OUTPUT, delete process.env.GH_AW_SAFE_OUTPUTS_STAGED, delete process.env.GH_AW_MILESTONE_ALLOWED, delete process.env.GH_AW_MILESTONE_MAX_COUNT);
      const scriptPath = path.join(process.cwd(), "assign_milestone.cjs");
      assignMilestoneScript = fs.readFileSync(scriptPath, "utf8");
    }),
      afterEach(() => {
        tempFilePath && fs.existsSync(tempFilePath) && fs.unlinkSync(tempFilePath);
      }),
      it("should handle empty agent output", async () => {
        (setAgentOutput({ items: [], errors: [] }), await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No assign_milestone items found")));
      }),
      it("should handle missing agent output", async () => {
        (delete process.env.GH_AW_AGENT_OUTPUT, await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found"));
      }),
      it("should assign milestone successfully", async () => {
        (setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 5 }], errors: [] }),
          mockGithub.rest.issues.update.mockResolvedValue({}),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({ owner: "test-owner", repo: "test-repo", issue_number: 42, milestone: 5 }),
          expect(mockCore.info).toHaveBeenCalledWith("Successfully assigned milestone #5 to issue #42"),
          expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_milestones", "42:5"),
          expect(mockCore.summary.addRaw).toHaveBeenCalled());
      }),
      it("should handle staged mode correctly", async () => {
        ((process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"),
          setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 5 }], errors: [] }),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.summary.addRaw).toHaveBeenCalled());
        const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
        (expect(summaryCall).toContain("🎭 Staged Mode"), expect(summaryCall).toContain("Issue:** #42"), expect(summaryCall).toContain("Milestone Number:** 5"));
      }),
      it("should respect max count configuration", async () => {
        ((process.env.GH_AW_MILESTONE_MAX_COUNT = "2"),
          setAgentOutput({
            items: [
              { type: "assign_milestone", issue_number: 1, milestone_number: 5 },
              { type: "assign_milestone", issue_number: 2, milestone_number: 5 },
              { type: "assign_milestone", issue_number: 3, milestone_number: 5 },
            ],
            errors: [],
          }),
          mockGithub.rest.issues.update.mockResolvedValue({}),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledTimes(2),
          expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Found 3 milestone assignments, but max is 2")));
      }),
      it("should validate against allowed milestones list", async () => {
        ((process.env.GH_AW_MILESTONE_ALLOWED = "v1.0,v2.0"),
          setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 5 }], errors: [] }),
          mockGithub.rest.issues.listMilestones.mockResolvedValue({
            data: [
              { number: 5, title: "v1.0" },
              { number: 6, title: "v3.0" },
            ],
          }),
          mockGithub.rest.issues.update.mockResolvedValue({}),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.listMilestones).toHaveBeenCalledWith({ owner: "test-owner", repo: "test-repo", state: "all", per_page: 100 }),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({ owner: "test-owner", repo: "test-repo", issue_number: 42, milestone: 5 }));
      }),
      it("should reject milestone not in allowed list", async () => {
        ((process.env.GH_AW_MILESTONE_ALLOWED = "v1.0,v2.0"),
          setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 6 }], errors: [] }),
          mockGithub.rest.issues.listMilestones.mockResolvedValue({
            data: [
              { number: 5, title: "v1.0" },
              { number: 6, title: "v3.0" },
            ],
          }),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining('Milestone "v3.0" (#6) is not in the allowed list')));
      }),
      it("should handle API errors gracefully", async () => {
        setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 5 }], errors: [] });
        const apiError = new Error("API rate limit exceeded");
        (mockGithub.rest.issues.update.mockRejectedValue(apiError),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to assign milestone #5 to issue #42")),
          expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to assign 1 milestone(s)")));
      }),
      it("should handle invalid issue numbers", async () => {
        (setAgentOutput({ items: [{ type: "assign_milestone", issue_number: -1, milestone_number: 5 }], errors: [] }),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Invalid issue_number")));
      }),
      it("should handle invalid milestone numbers", async () => {
        (setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 0 }], errors: [] }),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Invalid milestone_number")));
      }),
      it("should handle multiple milestone assignments", async () => {
        ((process.env.GH_AW_MILESTONE_MAX_COUNT = "2"),
          setAgentOutput({
            items: [
              { type: "assign_milestone", issue_number: 1, milestone_number: 5 },
              { type: "assign_milestone", issue_number: 2, milestone_number: 6 },
            ],
            errors: [],
          }),
          mockGithub.rest.issues.update.mockResolvedValue({}),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledTimes(2),
          expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_milestones", "1:5\n2:6"));
      }),
      it("should skip milestone not found in repository", async () => {
        ((process.env.GH_AW_MILESTONE_ALLOWED = "v1.0"),
          setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 999 }], errors: [] }),
          mockGithub.rest.issues.listMilestones.mockResolvedValue({
            data: [
              { number: 5, title: "v1.0" },
              { number: 6, title: "v2.0" },
            ],
          }),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.warning).toHaveBeenCalledWith("Milestone #999 not found in repository. Skipping."));
      }),
      it("should validate allowed milestones by milestone number", async () => {
        ((process.env.GH_AW_MILESTONE_ALLOWED = "5"),
          setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 5 }], errors: [] }),
          mockGithub.rest.issues.listMilestones.mockResolvedValue({
            data: [
              { number: 5, title: "v1.0" },
              { number: 6, title: "v2.0" },
            ],
          }),
          mockGithub.rest.issues.update.mockResolvedValue({}),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({ owner: "test-owner", repo: "test-repo", issue_number: 42, milestone: 5 }));
      }),
      it("should handle fetch milestones API error", async () => {
        ((process.env.GH_AW_MILESTONE_ALLOWED = "v1.0"),
          setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 5 }], errors: [] }),
          mockGithub.rest.issues.listMilestones.mockRejectedValue(new Error("API error")),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockCore.error).toHaveBeenCalledWith("Failed to fetch milestones: API error"),
          expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to fetch milestones for validation: API error"));
      }),
      it("should handle mixed success and failure results", async () => {
        ((process.env.GH_AW_MILESTONE_MAX_COUNT = "3"),
          setAgentOutput({
            items: [
              { type: "assign_milestone", issue_number: 1, milestone_number: 5 },
              { type: "assign_milestone", issue_number: 2, milestone_number: 6 },
              { type: "assign_milestone", issue_number: 3, milestone_number: 7 },
            ],
            errors: [],
          }),
          mockGithub.rest.issues.update.mockResolvedValueOnce({}).mockRejectedValueOnce(new Error("API failure")).mockResolvedValueOnce({}),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledTimes(3),
          expect(mockCore.info).toHaveBeenCalledWith("Successfully assigned milestone #5 to issue #1"),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to assign milestone #6 to issue #2")),
          expect(mockCore.info).toHaveBeenCalledWith("Successfully assigned milestone #7 to issue #3"),
          expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to assign 1 milestone(s)"),
          expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_milestones", "1:5\n3:7"));
        const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
        (expect(summaryCall).toContain("✅ Successfully assigned 2 milestone(s)"), expect(summaryCall).toContain("❌ Failed to assign 1 milestone(s)"));
      }),
      it("should handle string issue and milestone numbers", async () => {
        (setAgentOutput({ items: [{ type: "assign_milestone", issue_number: "42", milestone_number: "5" }], errors: [] }),
          mockGithub.rest.issues.update.mockResolvedValue({}),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({ owner: "test-owner", repo: "test-repo", issue_number: 42, milestone: 5 }),
          expect(mockCore.info).toHaveBeenCalledWith("Successfully assigned milestone #5 to issue #42"));
      }),
      it("should handle non-numeric issue numbers", async () => {
        (setAgentOutput({ items: [{ type: "assign_milestone", issue_number: "abc", milestone_number: 5 }], errors: [] }),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.error).toHaveBeenCalledWith("Invalid issue_number: abc"));
      }),
      it("should handle non-numeric milestone numbers", async () => {
        (setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: "xyz" }], errors: [] }),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.update).not.toHaveBeenCalled(),
          expect(mockCore.error).toHaveBeenCalledWith("Invalid milestone_number: xyz"));
      }),
      it("should handle empty allowed milestones list", async () => {
        ((process.env.GH_AW_MILESTONE_ALLOWED = ""),
          setAgentOutput({ items: [{ type: "assign_milestone", issue_number: 42, milestone_number: 5 }], errors: [] }),
          mockGithub.rest.issues.update.mockResolvedValue({}),
          await eval(`(async () => { ${assignMilestoneScript}; await main(); })()`),
          expect(mockGithub.rest.issues.listMilestones).not.toHaveBeenCalled(),
          expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({ owner: "test-owner", repo: "test-repo", issue_number: 42, milestone: 5 }));
      }));
  }));
