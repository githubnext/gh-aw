import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";
const mockCore = { debug: vi.fn(), info: vi.fn(), notice: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() } },
  mockGithub = { rest: { pulls: { requestReviewers: vi.fn().mockResolvedValue({}) } } },
  mockContext = { eventName: "pull_request", repo: { owner: "testowner", repo: "testrepo" }, payload: { pull_request: { number: 123 } } };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("add_reviewer", () => {
    let tempDir, outputFile;
    (beforeEach(() => {
      ((tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "add-reviewer-test-"))),
        (outputFile = path.join(tempDir, "agent-output.json")),
        vi.clearAllMocks(),
        vi.resetModules(),
        delete process.env.GH_AW_SAFE_OUTPUTS_STAGED,
        delete process.env.GH_AW_REVIEWERS_ALLOWED,
        delete process.env.GH_AW_REVIEWERS_MAX_COUNT,
        delete process.env.GH_AW_REVIEWERS_TARGET,
        (global.context = { eventName: "pull_request", repo: { owner: "testowner", repo: "testrepo" }, payload: { pull_request: { number: 123 } } }));
    }),
      afterEach(() => {
        fs.existsSync(tempDir) && fs.rmSync(tempDir, { recursive: !0, force: !0 });
      }),
      it("should handle missing GH_AW_AGENT_OUTPUT", async () => {
        (delete process.env.GH_AW_AGENT_OUTPUT, await import("./add_reviewer.cjs"), expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found"));
      }),
      it("should handle missing add_reviewer item", async () => {
        (fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "create_issue", title: "Test", body: "Body" }] })),
          (process.env.GH_AW_AGENT_OUTPUT = outputFile),
          await import("./add_reviewer.cjs"),
          expect(mockCore.warning).toHaveBeenCalledWith("No add-reviewer item found in agent output"));
      }),
      it("should add reviewers to PR in non-staged mode", async () => {
        (fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "add_reviewer", reviewers: ["octocat", "github"] }] })),
          (process.env.GH_AW_AGENT_OUTPUT = outputFile),
          (process.env.GH_AW_REVIEWERS_MAX_COUNT = "3"),
          (process.env.GH_AW_REVIEWERS_TARGET = "triggering"),
          await import("./add_reviewer.cjs"),
          expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", pull_number: 123, reviewers: ["octocat", "github"] }),
          expect(mockCore.setOutput).toHaveBeenCalledWith("reviewers_added", "octocat\ngithub"));
      }),
      it("should generate staged preview in staged mode", async () => {
        (fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "add_reviewer", reviewers: ["octocat", "github"], pull_request_number: 123 }] })),
          (process.env.GH_AW_AGENT_OUTPUT = outputFile),
          (process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"),
          await import("./add_reviewer.cjs"),
          expect(mockCore.summary.addRaw).toHaveBeenCalled(),
          expect(mockCore.summary.write).toHaveBeenCalled(),
          expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled());
      }),
      it("should filter by allowed reviewers", async () => {
        (fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "add_reviewer", reviewers: ["octocat", "github", "unauthorized"] }] })),
          (process.env.GH_AW_AGENT_OUTPUT = outputFile),
          (process.env.GH_AW_REVIEWERS_ALLOWED = "octocat,github"),
          (process.env.GH_AW_REVIEWERS_MAX_COUNT = "3"),
          await import("./add_reviewer.cjs"),
          expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", pull_number: 123, reviewers: ["octocat", "github"] }));
      }),
      it("should enforce max count limit", async () => {
        (fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "add_reviewer", reviewers: ["user1", "user2", "user3", "user4", "user5"] }] })),
          (process.env.GH_AW_AGENT_OUTPUT = outputFile),
          (process.env.GH_AW_REVIEWERS_MAX_COUNT = "2"),
          await import("./add_reviewer.cjs"),
          expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", pull_number: 123, reviewers: ["user1", "user2"] }));
      }),
      it("should handle non-PR context gracefully", async () => {
        (fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "add_reviewer", reviewers: ["octocat"] }] })),
          (process.env.GH_AW_AGENT_OUTPUT = outputFile),
          (global.context = { ...mockContext, eventName: "issues", payload: { issue: { number: 123 } } }),
          await import("./add_reviewer.cjs"),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining('Target is "triggering" but not running in pull request context')),
          expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled());
      }),
      it("should handle explicit PR number with * target", async () => {
        (fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "add_reviewer", reviewers: ["octocat"], pull_request_number: 456 }] })),
          (process.env.GH_AW_AGENT_OUTPUT = outputFile),
          (process.env.GH_AW_REVIEWERS_TARGET = "*"),
          await import("./add_reviewer.cjs"),
          expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", pull_number: 456, reviewers: ["octocat"] }));
      }),
      it("should handle API errors gracefully", async () => {
        (fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "add_reviewer", reviewers: ["octocat"] }] })),
          (process.env.GH_AW_AGENT_OUTPUT = outputFile),
          mockGithub.rest.pulls.requestReviewers.mockRejectedValueOnce(new Error("API Error")),
          await import("./add_reviewer.cjs"),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to add reviewers")),
          expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to add reviewers")));
      }),
      it("should deduplicate reviewers", async () => {
        (fs.writeFileSync(outputFile, JSON.stringify({ items: [{ type: "add_reviewer", reviewers: ["octocat", "github", "octocat", "github"] }] })),
          (process.env.GH_AW_AGENT_OUTPUT = outputFile),
          (process.env.GH_AW_REVIEWERS_MAX_COUNT = "10"),
          await import("./add_reviewer.cjs"),
          expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", pull_number: 123, reviewers: ["octocat", "github"] }));
      }));
  }));
