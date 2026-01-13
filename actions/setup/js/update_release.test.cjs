import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn(), summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() } },
  mockGithub = { rest: { repos: { getReleaseByTag: vi.fn(), updateRelease: vi.fn() } } },
  mockContext = { repo: { owner: "test-owner", repo: "test-repo" }, serverUrl: "https://github.com", runId: 123456 };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("update_release", () => {
    let updateReleaseScript, tempFilePath;
    const setAgentOutput = data => {
      tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
      const content = "string" == typeof data ? data : JSON.stringify(data);
      (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
    };
    (beforeEach(() => {
      (vi.clearAllMocks(), delete process.env.GH_AW_SAFE_OUTPUTS_STAGED, delete process.env.GH_AW_AGENT_OUTPUT, delete process.env.GH_AW_WORKFLOW_NAME);
      const scriptPath = path.join(__dirname, "update_release.cjs");
      updateReleaseScript = fs.readFileSync(scriptPath, "utf8");
    }),
      afterEach(() => {
        tempFilePath && fs.existsSync(tempFilePath) && (fs.unlinkSync(tempFilePath), (tempFilePath = void 0));
      }),
      it("should handle empty message in staged mode", async () => {
        ((process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"),
          await eval(`(async () => { ${updateReleaseScript}; const handler = await main(); const result = await handler({}); return result; })()`),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Staged mode:")),
          expect(mockGithub.rest.repos.getReleaseByTag).not.toHaveBeenCalled());
      }),
      it("should handle replace operation", async () => {
        const mockRelease = { id: 1, tag_name: "v1.0.0", name: "Release v1.0.0", body: "Old release notes", html_url: "https://github.com/test-owner/test-repo/releases/tag/v1.0.0" },
          mockUpdatedRelease = { ...mockRelease, body: "New release notes" };
        const message = { type: "update_release", tag: "v1.0.0", operation: "replace", body: "New release notes" };
        (mockGithub.rest.repos.getReleaseByTag.mockResolvedValue({ data: mockRelease }), mockGithub.rest.repos.updateRelease.mockResolvedValue({ data: mockUpdatedRelease }), (process.env.GH_AW_WORKFLOW_NAME = "Test Workflow"));
        const result = await eval(`(async () => { ${updateReleaseScript}; const handler = await main(); return await handler(${JSON.stringify(message)}); })()`);
        expect(mockGithub.rest.repos.getReleaseByTag).toHaveBeenCalledWith({ owner: "test-owner", repo: "test-repo", tag: "v1.0.0" });
        // After the change, replace operation now adds footer
        const callArgs = mockGithub.rest.repos.updateRelease.mock.calls[0][0];
        expect(callArgs.owner).toBe("test-owner");
        expect(callArgs.repo).toBe("test-repo");
        expect(callArgs.release_id).toBe(1);
        expect(callArgs.body).toContain("New release notes");
        expect(callArgs.body).not.toContain("Old release notes");
        expect(callArgs.body).toContain("Test Workflow");
        expect(callArgs.body).toContain("https://github.com/test-owner/test-repo/actions/runs/123456");
        expect(result.tag).toBe("v1.0.0");
        expect(result.id).toBe(1);
      }),
      it("should handle append operation", async () => {
        const mockRelease = { id: 2, tag_name: "v2.0.0", name: "Release v2.0.0", body: "Original release notes", html_url: "https://github.com/test-owner/test-repo/releases/tag/v2.0.0" };
        const message = { type: "update_release", tag: "v2.0.0", operation: "append", body: "Additional notes" };
        (mockGithub.rest.repos.getReleaseByTag.mockResolvedValue({ data: mockRelease }),
          mockGithub.rest.repos.updateRelease.mockResolvedValue({ data: { ...mockRelease, body: "Updated body" } }),
          (process.env.GH_AW_WORKFLOW_NAME = "Test Workflow"));
        await eval(`(async () => { ${updateReleaseScript}; const handler = await main(); return await handler(${JSON.stringify(message)}); })()`);
        expect(mockGithub.rest.repos.updateRelease).toHaveBeenCalled();
        const callArgs = mockGithub.rest.repos.updateRelease.mock.calls[0][0];
        expect(callArgs.owner).toBe("test-owner");
        expect(callArgs.repo).toBe("test-repo");
        expect(callArgs.release_id).toBe(2);
        // Check structure: should contain original, separator, new content, and footer
        expect(callArgs.body).toContain("Original release notes");
        expect(callArgs.body).toContain("---");
        expect(callArgs.body).toContain("Additional notes");
        expect(callArgs.body).toContain("Test Workflow");
        expect(callArgs.body).toContain("https://github.com/test-owner/test-repo/actions/runs/123456");
      }),
      it("should handle prepend operation", async () => {
        const mockRelease = { id: 3, tag_name: "v3.0.0", name: "Release v3.0.0", body: "Existing release notes", html_url: "https://github.com/test-owner/test-repo/releases/tag/v3.0.0" };
        const message = { type: "update_release", tag: "v3.0.0", operation: "prepend", body: "Prepended notes" };
        (mockGithub.rest.repos.getReleaseByTag.mockResolvedValue({ data: mockRelease }),
          mockGithub.rest.repos.updateRelease.mockResolvedValue({ data: { ...mockRelease, body: "Updated body" } }),
          (process.env.GH_AW_WORKFLOW_NAME = "Test Workflow"));
        await eval(`(async () => { ${updateReleaseScript}; const handler = await main(); return await handler(${JSON.stringify(message)}); })()`);
        expect(mockGithub.rest.repos.updateRelease).toHaveBeenCalled();
        const callArgs = mockGithub.rest.repos.updateRelease.mock.calls[0][0];
        expect(callArgs.owner).toBe("test-owner");
        expect(callArgs.repo).toBe("test-repo");
        expect(callArgs.release_id).toBe(3);
        // Check structure: should contain new content, footer, separator, and original
        expect(callArgs.body).toContain("Prepended notes");
        expect(callArgs.body).toContain("Test Workflow");
        expect(callArgs.body).toContain("https://github.com/test-owner/test-repo/actions/runs/123456");
        expect(callArgs.body).toContain("---");
        expect(callArgs.body).toContain("Existing release notes");
        // Verify order: new content comes before original
        expect(callArgs.body.indexOf("Prepended notes")).toBeLessThan(callArgs.body.indexOf("Existing release notes"));
      }),
      it("should handle staged mode", async () => {
        process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
        const message = { type: "update_release", tag: "v1.0.0", operation: "replace", body: "New notes" };
        const result = await eval(`(async () => { ${updateReleaseScript}; const handler = await main(); return await handler(${JSON.stringify(message)}); })()`);
        expect(result.skipped).toBe(true);
        expect(result.reason).toBe("staged_mode");
        expect(mockGithub.rest.repos.getReleaseByTag).not.toHaveBeenCalled();
        expect(mockGithub.rest.repos.updateRelease).not.toHaveBeenCalled();
      }),
      it("should handle release not found error", async () => {
        mockGithub.rest.repos.getReleaseByTag.mockRejectedValue(new Error("Not Found"));
        const message = { type: "update_release", tag: "v99.99.99", operation: "replace", body: "New notes" };
        try {
          await eval(`(async () => { ${updateReleaseScript}; const handler = await main(); return await handler(${JSON.stringify(message)}); })()`);
          // Should not reach here
          expect(true).toBe(false);
        } catch (error) {
          expect(error.message).toContain("Release with tag 'v99.99.99' not found");
        }
      }),
      it("should handle multiple release updates", async () => {
        const mockRelease1 = { id: 1, tag_name: "v1.0.0", body: "Release 1", html_url: "https://github.com/test-owner/test-repo/releases/tag/v1.0.0" },
          mockRelease2 = { id: 2, tag_name: "v2.0.0", body: "Release 2", html_url: "https://github.com/test-owner/test-repo/releases/tag/v2.0.0" };
        const message1 = { type: "update_release", tag: "v1.0.0", operation: "replace", body: "Updated 1" };
        const message2 = { type: "update_release", tag: "v2.0.0", operation: "replace", body: "Updated 2" };
        (mockGithub.rest.repos.getReleaseByTag.mockResolvedValueOnce({ data: mockRelease1 }).mockResolvedValueOnce({ data: mockRelease2 }),
          mockGithub.rest.repos.updateRelease.mockResolvedValueOnce({ data: { ...mockRelease1, body: "Updated 1" } }).mockResolvedValueOnce({ data: { ...mockRelease2, body: "Updated 2" } }));
        const handler = await eval(`(async () => { ${updateReleaseScript}; return await main(); })()`);
        await handler(message1);
        await handler(message2);
        expect(mockGithub.rest.repos.getReleaseByTag).toHaveBeenCalledTimes(2);
        expect(mockGithub.rest.repos.updateRelease).toHaveBeenCalledTimes(2);
      }),
      it("should infer tag from release event context", async () => {
        ((mockContext.eventName = "release"), (mockContext.payload = { release: { tag_name: "v1.5.0", name: "Version 1.5.0", body: "Original release body" } }));
        const mockRelease = { id: 1, tag_name: "v1.5.0", body: "Original release body", html_url: "https://github.com/test-owner/test-repo/releases/tag/v1.5.0" };
        const message = { type: "update_release", operation: "replace", body: "Updated body" };
        (mockGithub.rest.repos.getReleaseByTag.mockResolvedValue({ data: mockRelease }), mockGithub.rest.repos.updateRelease.mockResolvedValue({ data: { ...mockRelease, body: "Updated body" } }));
        await eval(`(async () => { ${updateReleaseScript}; const handler = await main(); return await handler(${JSON.stringify(message)}); })()`);
        expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Inferred release tag from event context: v1.5.0"));
        expect(mockGithub.rest.repos.getReleaseByTag).toHaveBeenCalledWith({ owner: "test-owner", repo: "test-repo", tag: "v1.5.0" });
        // After the change, replace operation now adds footer
        const callArgs = mockGithub.rest.repos.updateRelease.mock.calls[0][0];
        expect(callArgs.owner).toBe("test-owner");
        expect(callArgs.repo).toBe("test-repo");
        expect(callArgs.release_id).toBe(1);
        expect(callArgs.body).toContain("Updated body");
        expect(callArgs.body).toContain("GitHub Agentic Workflow");
        expect(callArgs.body).toContain("https://github.com/test-owner/test-repo/actions/runs/123456");
        delete mockContext.eventName;
        delete mockContext.payload;
      }),
      it("should fail gracefully when tag is missing and cannot be inferred", async () => {
        ((mockContext.eventName = "push"), (mockContext.payload = {}));
        const message = { type: "update_release", operation: "replace", body: "Updated body" };
        try {
          await eval(`(async () => { ${updateReleaseScript}; const handler = await main(); return await handler(${JSON.stringify(message)}); })()`);
          // Should not reach here
          expect(true).toBe(false);
        } catch (error) {
          expect(error.message).toContain("Release tag is required");
        }
        delete mockContext.eventName;
        delete mockContext.payload;
      }));
  }));
