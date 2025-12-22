import { describe, it, expect, beforeEach, vi } from "vitest";
import { readFileSync } from "fs";
import path from "path";
const createTestableFunction = scriptContent => {
  const beforeMainCall = scriptContent.match(/^([\s\S]*?)\s*await main\(\);?\s*$/);
  if (!beforeMainCall) throw new Error("Could not extract script content before await main()");
  let scriptBody = beforeMainCall[1];
  return (
    (scriptBody = scriptBody.replace(/\/\*\* @type \{typeof import\("fs"\)\} \*\/\s*const fs = require\("fs"\);?\s*/g, "")),
    (scriptBody = scriptBody.replace(/\/\*\* @type \{typeof import\("crypto"\)\} \*\/\s*const crypto = require\("crypto"\);?\s*/g, "")),
    (scriptBody = scriptBody.replace(/const \{ updateActivationComment \} = require\("\.\/update_activation_comment\.cjs"\);?\s*/g, "")),
    (scriptBody = scriptBody.replace(/const \{ getTrackerID \} = require\("\.\/get_tracker_id\.cjs"\);?\s*/g, "")),
    (scriptBody = scriptBody.replace(/const \{ addExpirationComment \} = require\("\.\/expiration_helpers\.cjs"\);?\s*/g, "")),
    (scriptBody = scriptBody.replace(/const \{ removeDuplicateTitleFromDescription \} = require\("\.\/remove_duplicate_title\.cjs"\);?\s*/g, "")),
    new Function(
      `\n    const { fs, crypto, github, core, context, process, console, updateActivationComment, getTrackerID, addExpirationComment, removeDuplicateTitleFromDescription } = arguments[0];\n    \n    ${scriptBody}\n    \n    return main;\n  `
    )
  );
};
describe("create_pull_request.cjs", () => {
  let createMainFunction, tempFilePath;
  const mockPatchContent = (mockDeps, patchContent) => {
    mockDeps.fs.readFileSync.mockImplementation(filepath => (filepath === mockDeps.process.env.GH_AW_AGENT_OUTPUT ? mockDeps.process.env.GH_AW_AGENT_OUTPUT : patchContent));
  };
  let mockDependencies;
  (beforeEach(() => {
    const scriptPath = path.join(process.cwd(), "create_pull_request.cjs"),
      scriptContent = readFileSync(scriptPath, "utf8");
    ((createMainFunction = createTestableFunction(scriptContent)),
      (global.exec = { exec: vi.fn().mockResolvedValue(0), getExecOutput: vi.fn().mockResolvedValue({ exitCode: 0, stdout: "", stderr: "" }) }),
      (mockDependencies = {
        fs: {
          existsSync: vi.fn().mockReturnValue(!0),
          readFileSync: vi.fn().mockImplementation(filepath => (filepath === mockDependencies.process.env.GH_AW_AGENT_OUTPUT ? mockDependencies.process.env.GH_AW_AGENT_OUTPUT : "diff --git a/file.txt b/file.txt\n+new content")),
        },
        crypto: { randomBytes: vi.fn().mockReturnValue(Buffer.from("1234567890abcdef", "hex")) },
        execSync: vi.fn(),
        github: { rest: { pulls: { create: vi.fn() }, issues: { addLabels: vi.fn() } } },
        core: {
          debug: vi.fn(),
          info: vi.fn(),
          notice: vi.fn(),
          warning: vi.fn(),
          error: vi.fn(),
          setFailed: vi.fn(),
          setOutput: vi.fn(),
          exportVariable: vi.fn(),
          setSecret: vi.fn(),
          getInput: vi.fn(),
          getBooleanInput: vi.fn(),
          getMultilineInput: vi.fn(),
          getState: vi.fn(),
          saveState: vi.fn(),
          startGroup: vi.fn(),
          endGroup: vi.fn(),
          group: vi.fn(),
          addPath: vi.fn(),
          setCommandEcho: vi.fn(),
          isDebug: vi.fn().mockReturnValue(!1),
          getIDToken: vi.fn(),
          toPlatformPath: vi.fn(),
          toPosixPath: vi.fn(),
          toWin32Path: vi.fn(),
          summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
        },
        context: { runId: 12345, repo: { owner: "testowner", repo: "testrepo" }, payload: { repository: { html_url: "https://github.com/testowner/testrepo" } } },
        process: { env: {} },
        console: { log: vi.fn() },
        updateActivationComment: vi.fn(),
        getTrackerID: vi.fn(format => ""),
        addExpirationComment: vi.fn(),
        removeDuplicateTitleFromDescription: vi.fn((title, description) => description),
      }));
  }),
    afterEach(() => {
      (tempFilePath && require("fs").existsSync(tempFilePath) && (require("fs").unlinkSync(tempFilePath), (tempFilePath = void 0)),
        tempFilePath && require("fs").existsSync(tempFilePath) && (require("fs").unlinkSync(tempFilePath), (tempFilePath = void 0)),
        "undefined" != typeof global && delete global.exec);
    }),
    it("should throw error when GH_AW_WORKFLOW_ID is missing", async () => {
      const mainFunction = createMainFunction(mockDependencies);
      await expect(mainFunction()).rejects.toThrow("GH_AW_WORKFLOW_ID environment variable is required");
    }),
    it("should throw error when GH_AW_BASE_BRANCH is missing", async () => {
      mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow";
      const mainFunction = createMainFunction(mockDependencies);
      await expect(mainFunction()).rejects.toThrow("GH_AW_BASE_BRANCH environment variable is required");
    }),
    it("should handle missing patch file with default warn behavior", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"), (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"), mockDependencies.fs.existsSync.mockReturnValue(!1));
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(), expect(mockDependencies.core.warning).toHaveBeenCalledWith("No patch file found - cannot create pull request without changes"), expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
    }),
    it("should handle empty patch with default warn behavior when patch file is empty", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"), (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"), mockPatchContent(mockDependencies, "   "));
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(), expect(mockDependencies.core.warning).toHaveBeenCalledWith("Patch file is empty - no changes to apply (noop operation)"), expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
    }),
    it("should create pull request successfully with valid input", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "New Feature", body: "This adds a new feature to the codebase." }] })),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "git rev-parse HEAD" === command ? "abc123456" : "";
        }));
      const mockPullRequest = { number: 123, html_url: "https://github.com/testowner/testrepo/pull/123" };
      mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: mockPullRequest });
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(),
        expect(global.exec.exec).toHaveBeenCalledWith("git fetch origin main"),
        expect(global.exec.exec).toHaveBeenCalledWith("git checkout main"),
        expect(global.exec.exec).toHaveBeenCalledWith("git checkout -b test-workflow-1234567890abcdef"),
        expect(global.exec.exec).toHaveBeenCalledWith("git am /tmp/gh-aw/aw.patch"),
        expect(global.exec.exec).toHaveBeenCalledWith("git push origin test-workflow-1234567890abcdef"),
        expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          title: "New Feature",
          body: expect.stringContaining("This adds a new feature to the codebase."),
          head: "test-workflow-1234567890abcdef",
          base: "main",
          draft: !0,
        }),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("pull_request_number", 123),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("pull_request_url", mockPullRequest.html_url),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("branch_name", "test-workflow-1234567890abcdef"));
    }),
    it("should handle labels correctly", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "PR with labels", body: "PR with labels" }] })),
        (mockDependencies.process.env.GH_AW_PR_LABELS = "enhancement, automated, needs-review"),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }),
        mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 456, html_url: "https://github.com/testowner/testrepo/pull/456" } }),
        mockDependencies.github.rest.issues.addLabels.mockResolvedValue({}));
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(), expect(mockDependencies.github.rest.issues.addLabels).toHaveBeenCalledWith({ owner: "testowner", repo: "testrepo", issue_number: 456, labels: ["enhancement", "automated", "needs-review"] }));
    }),
    it("should respect draft setting from environment", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Non-draft PR", body: "Non-draft PR" }] })),
        (mockDependencies.process.env.GH_AW_PR_DRAFT = "false"),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }),
        mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 789, html_url: "https://github.com/testowner/testrepo/pull/789" } }));
      const mainFunction = createMainFunction(mockDependencies);
      await mainFunction();
      const callArgs = mockDependencies.github.rest.pulls.create.mock.calls[0][0];
      expect(callArgs.draft).toBe(!1);
    }),
    it("should include run information in PR body", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR Title", body: "Test PR content with detailed body information." }] })),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }),
        mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 202, html_url: "https://github.com/testowner/testrepo/pull/202" } }));
      const mainFunction = createMainFunction(mockDependencies);
      await mainFunction();
      const callArgs = mockDependencies.github.rest.pulls.create.mock.calls[0][0];
      (expect(callArgs.title).toBe("Test PR Title"),
        expect(callArgs.body).toContain("Test PR content with detailed body information."),
        expect(callArgs.body).toContain("AI generated by"),
        expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345"));
    }),
    it("should apply title prefix when provided", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Simple PR title", body: "Simple PR body content" }] })),
        (mockDependencies.process.env.GH_AW_PR_TITLE_PREFIX = "[BOT] "),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }),
        mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 987, html_url: "https://github.com/testowner/testrepo/pull/987" } }));
      const mainFunction = createMainFunction(mockDependencies);
      await mainFunction();
      const callArgs = mockDependencies.github.rest.pulls.create.mock.calls[0][0];
      expect(callArgs.title).toBe("[BOT] Simple PR title");
    }),
    it("should not duplicate title prefix when already present", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "[BOT] PR title already prefixed", body: "PR body content" }] })),
        (mockDependencies.process.env.GH_AW_PR_TITLE_PREFIX = "[BOT] "),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }),
        mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 988, html_url: "https://github.com/testowner/testrepo/pull/988" } }));
      const mainFunction = createMainFunction(mockDependencies);
      await mainFunction();
      const callArgs = mockDependencies.github.rest.pulls.create.mock.calls[0][0];
      expect(callArgs.title).toBe("[BOT] PR title already prefixed");
    }),
    it("should fallback to creating issue when PR creation fails", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "PR that will fail", body: "This PR creation will fail and fallback to an issue." }] })),
        (mockDependencies.process.env.GH_AW_PR_LABELS = "enhancement, automated"),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }));
      const prError = new Error("Pull request creation is disabled by organization policy");
      mockDependencies.github.rest.pulls.create.mockRejectedValue(prError);
      const mockIssue = { number: 456, html_url: "https://github.com/testowner/testrepo/issues/456" };
      mockDependencies.github.rest.issues = { ...mockDependencies.github.rest.issues, create: vi.fn().mockResolvedValue({ data: mockIssue }) };
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(),
        expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          title: "PR that will fail",
          body: expect.stringContaining("This PR creation will fail and fallback to an issue."),
          head: "test-workflow-1234567890abcdef",
          base: "main",
          draft: !0,
        }),
        expect(mockDependencies.github.rest.issues.create).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          title: "PR that will fail",
          body: expect.stringMatching(
            /This PR creation will fail and fallback to an issue\.[\s\S]*---[\s\S]*Note.*originally intended as a pull request[\s\S]*Original error.*Pull request creation is disabled by organization policy[\s\S]*You can manually create a pull request from the branch if needed/
          ),
          labels: ["enhancement", "automated"],
        }),
        expect(mockDependencies.core.warning).toHaveBeenCalledWith("Failed to create pull request: Pull request creation is disabled by organization policy"),
        expect(mockDependencies.core.info).toHaveBeenCalledWith("Falling back to creating an issue instead"),
        expect(mockDependencies.core.info).toHaveBeenCalledWith("Created fallback issue #456: https://github.com/testowner/testrepo/issues/456"),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_number", 456),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_url", mockIssue.html_url),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("branch_name", "test-workflow-1234567890abcdef"),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("fallback_used", "true"),
        expect(mockDependencies.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("## Fallback Issue Created")));
    }),
    it("should include patch preview in fallback issue when PR creation fails", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "PR with patch preview", body: "This PR will fail and create an issue with patch preview." }] })));
      const patchLines = ["diff --git a/file.txt b/file.txt", "--- a/file.txt", "+++ b/file.txt", "@@ -1,1 +1,1 @@"];
      for (let i = 0; i < 600; i++) patchLines.push(`+Line ${i}`);
      const largePatch = patchLines.join("\n");
      mockPatchContent(mockDependencies, largePatch);
      const prError = new Error("Pull request creation is disabled");
      (mockDependencies.github.rest.pulls.create.mockRejectedValue(prError),
        (mockDependencies.github.rest.issues = { ...mockDependencies.github.rest.issues, create: vi.fn().mockResolvedValue({ data: { number: 789, html_url: "https://github.com/testowner/testrepo/issues/789" } }) }));
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(), expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled());
      const issueCreateCall = mockDependencies.github.rest.issues.create.mock.calls[0][0];
      (expect(issueCreateCall.body).toMatch(/<details><summary>Show patch preview \(500 of 604 lines\)<\/summary>/),
        expect(issueCreateCall.body).toMatch(/```diff/),
        expect(issueCreateCall.body).toMatch(/\.\.\. \(truncated\)/),
        expect(issueCreateCall.body).toContain("+Line 0"),
        expect(issueCreateCall.body).not.toContain("+Line 550"));
    }),
    it("should truncate patch by character limit when it exceeds 2000 chars", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "PR with large patch", body: "This PR will fail and create an issue with char-limited patch." }] })));
      const patchLines = ["diff --git a/file.txt b/file.txt", "--- a/file.txt", "+++ b/file.txt", "@@ -1,1 +1,100 @@"];
      for (let i = 0; i < 100; i++) patchLines.push(`+This is a longer line ${i} with more content to trigger character limit truncation`);
      const largePatch = patchLines.join("\n");
      mockPatchContent(mockDependencies, largePatch);
      const prError = new Error("Pull request creation is disabled");
      (mockDependencies.github.rest.pulls.create.mockRejectedValue(prError),
        (mockDependencies.github.rest.issues = { ...mockDependencies.github.rest.issues, create: vi.fn().mockResolvedValue({ data: { number: 790, html_url: "https://github.com/testowner/testrepo/issues/790" } }) }));
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(), expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled());
      const issueCreateCall = mockDependencies.github.rest.issues.create.mock.calls[0][0];
      (expect(issueCreateCall.body).toMatch(/<details><summary>Show patch preview/), expect(issueCreateCall.body).toMatch(/```diff/), expect(issueCreateCall.body).toMatch(/\.\.\. \(truncated\)/));
      const patchMatch = issueCreateCall.body.match(/```diff\n([\s\S]*?)\n\.\.\. \(truncated\)/);
      if (patchMatch) {
        const patchInBody = patchMatch[1];
        expect(patchInBody.length).toBeLessThanOrEqual(2e3);
      }
    }),
    it("should include full patch when under 500 lines in fallback issue", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "PR with small patch", body: "This PR will fail and create an issue with full patch." }] })),
        mockPatchContent(mockDependencies, "diff --git a/file.txt b/file.txt\n--- a/file.txt\n+++ b/file.txt\n@@ -1,1 +1,1 @@\n+Small change"));
      const prError = new Error("Pull request creation is disabled");
      (mockDependencies.github.rest.pulls.create.mockRejectedValue(prError),
        (mockDependencies.github.rest.issues = { ...mockDependencies.github.rest.issues, create: vi.fn().mockResolvedValue({ data: { number: 790, html_url: "https://github.com/testowner/testrepo/issues/790" } }) }));
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(), expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled());
      const issueCreateCall = mockDependencies.github.rest.issues.create.mock.calls[0][0];
      (expect(issueCreateCall.body).toMatch(/<details><summary>Show patch \(5 lines\)<\/summary>/),
        expect(issueCreateCall.body).toMatch(/```diff/),
        expect(issueCreateCall.body).toContain("+Small change"),
        expect(issueCreateCall.body).not.toContain("... (truncated)"));
    }),
    it("should fail when both PR and fallback issue creation fail", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "PR that will fail", body: "Both PR and issue creation will fail." }] })),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }));
      const prError = new Error("Pull request creation failed"),
        issueError = new Error("Issue creation also failed");
      (mockDependencies.github.rest.pulls.create.mockRejectedValue(prError), (mockDependencies.github.rest.issues = { ...mockDependencies.github.rest.issues, create: vi.fn().mockRejectedValue(issueError) }));
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(),
        expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalled(),
        expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled(),
        expect(mockDependencies.core.setFailed).toHaveBeenCalledWith("Failed to create both pull request and fallback issue. PR error: Pull request creation failed. Issue error: Issue creation also failed"));
    }),
    it("should fallback to creating issue when git push fails", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Push will fail", body: "Git push will fail and fallback to an issue." }] })),
        (mockDependencies.process.env.GH_AW_PR_LABELS = "automation"),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }),
        (global.exec.exec = vi.fn().mockImplementation(async (cmd, args, options) => {
          if ("string" == typeof cmd && cmd.includes("git push")) throw new Error("Permission denied (publickey)");
          return ("string" == typeof cmd && cmd.includes("git ls-remote"), 0);
        })));
      const mockIssue = { number: 789, html_url: "https://github.com/testowner/testrepo/issues/789" };
      mockDependencies.github.rest.issues = { ...mockDependencies.github.rest.issues, create: vi.fn().mockResolvedValue({ data: mockIssue }) };
      const mainFunction = createMainFunction(mockDependencies);
      await mainFunction();
      const pushCallsForTest = global.exec.exec.mock.calls.filter(call => call[0] && call[0].includes("git push"));
      (expect(pushCallsForTest.length).toBeGreaterThan(0),
        expect(mockDependencies.github.rest.issues.create).toHaveBeenCalledWith({
          owner: "testowner",
          repo: "testrepo",
          title: "Push will fail",
          body: expect.stringMatching(/Git push will fail[\s\S]*\[!NOTE\][\s\S]*git push operation failed[\s\S]*gh run download[\s\S]*git am aw\.patch/),
          labels: ["automation"],
        }),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_number", 789),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_url", mockIssue.html_url),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("branch_name", "test-workflow-1234567890abcdef"),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("fallback_used", "true"),
        expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("push_failed", "true"),
        expect(mockDependencies.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("## Push Failure Fallback")),
        expect(mockDependencies.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Permission denied")),
        expect(mockDependencies.core.error).toHaveBeenCalledWith(expect.stringContaining("Git push failed")),
        expect(mockDependencies.core.warning).toHaveBeenCalledWith(expect.stringContaining("Git push operation failed")));
    }),
    it("should include patch preview in fallback issue when git push fails", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Push will fail with patch", body: "Git push will fail and create issue with patch preview." }] })));
      const patchLines = ["diff --git a/test.js b/test.js", "--- a/test.js", "+++ b/test.js", "@@ -1,1 +1,1 @@"];
      for (let i = 0; i < 100; i++) patchLines.push(`+Test line ${i}`);
      const testPatch = patchLines.join("\n");
      (mockPatchContent(mockDependencies, testPatch),
        (global.exec.exec = vi.fn().mockImplementation(async (cmd, args, options) => {
          if ("string" == typeof cmd && cmd.includes("git push")) throw new Error("Permission denied (publickey)");
          return ("string" == typeof cmd && cmd.includes("git ls-remote"), 0);
        })),
        (mockDependencies.github.rest.issues = { ...mockDependencies.github.rest.issues, create: vi.fn().mockResolvedValue({ data: { number: 890, html_url: "https://github.com/testowner/testrepo/issues/890" } }) }));
      const mainFunction = createMainFunction(mockDependencies);
      (await mainFunction(), expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled());
      const issueCreateCall = mockDependencies.github.rest.issues.create.mock.calls[0][0];
      (expect(issueCreateCall.body).toMatch(/<details><summary>Show patch \(104 lines\)<\/summary>/),
        expect(issueCreateCall.body).toMatch(/```diff/),
        expect(issueCreateCall.body).toContain("diff --git a/test.js b/test.js"),
        expect(issueCreateCall.body).toContain("+Test line 0"));
    }),
    it("should fail when both git push and fallback issue creation fail", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Push and issue will fail", body: "Both git push and issue creation will fail." }] })),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }),
        (global.exec.exec = vi.fn().mockImplementation(async (cmd, args, options) => {
          if ("string" == typeof cmd && cmd.includes("git push")) throw new Error("Network error: Connection timeout");
          return ("string" == typeof cmd && cmd.includes("git ls-remote"), 0);
        })));
      const issueError = new Error("GitHub API rate limit exceeded");
      mockDependencies.github.rest.issues = { ...mockDependencies.github.rest.issues, create: vi.fn().mockRejectedValue(issueError) };
      const mainFunction = createMainFunction(mockDependencies);
      await mainFunction();
      const pushCallsInFailTest = global.exec.exec.mock.calls.filter(call => call[0] && call[0].includes("git push"));
      (expect(pushCallsInFailTest.length).toBeGreaterThan(0),
        expect(mockDependencies.github.rest.issues.create).toHaveBeenCalled(),
        expect(mockDependencies.core.setFailed).toHaveBeenCalledWith(expect.stringMatching(/Failed to push and failed to create fallback issue.*Network error.*GitHub API rate limit/)));
    }),
    it("should handle remote branch collision by appending random suffix", async () => {
      ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
        (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
        (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR with branch collision", body: "This will handle remote branch collision." }] })),
        mockDependencies.execSync.mockImplementation(command => {
          if ("git diff --cached --exit-code" === command) {
            const error = new Error("Changes exist");
            throw ((error.status = 1), error);
          }
          return "";
        }));
      let randomBytesCallCount = 0;
      ((mockDependencies.crypto.randomBytes = vi.fn().mockImplementation(size => (randomBytesCallCount++, 1 === randomBytesCallCount ? Buffer.from("1234567890abcdef", "hex") : Buffer.from("fedcba09", "hex")))),
        (global.exec.getExecOutput = vi
          .fn()
          .mockImplementation(async cmd => ("string" == typeof cmd && cmd.includes("git ls-remote") ? { exitCode: 0, stdout: "abc123 refs/heads/test-workflow-1234567890abcdef\n", stderr: "" } : { exitCode: 0, stdout: "", stderr: "" }))),
        (global.exec.exec = vi.fn().mockImplementation(async (cmd, args, options) => (("string" == typeof cmd && cmd.includes("git branch -m")) || ("string" == typeof cmd && cmd.includes("git push")), 0))),
        mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 123, html_url: "https://github.com/testowner/testrepo/pull/123" } }));
      const mainFunction = createMainFunction(mockDependencies);
      await mainFunction();
      const branchRenameCalls = global.exec.exec.mock.calls.filter(call => call[0] && call[0].includes("git branch -m"));
      (expect(branchRenameCalls.length).toBeGreaterThan(0),
        expect(mockDependencies.core.warning).toHaveBeenCalledWith(expect.stringContaining("already exists")),
        expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalledWith(expect.objectContaining({ head: expect.stringMatching(/test-workflow-1234567890abcdef-fedcba09/) })));
    }),
    describe("if-no-changes configuration", () => {
      (beforeEach(() => {
        ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
          (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
          (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "Test PR body" }] })));
      }),
        it("should handle empty patch with warn (default) behavior", async () => {
          (mockPatchContent(mockDependencies, ""), (mockDependencies.process.env.GH_AW_PR_IF_NO_CHANGES = "warn"));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(), expect(mockDependencies.core.warning).toHaveBeenCalledWith("Patch file is empty - no changes to apply (noop operation)"), expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
        }),
        it("should handle empty patch with ignore behavior", async () => {
          (mockPatchContent(mockDependencies, ""), (mockDependencies.process.env.GH_AW_PR_IF_NO_CHANGES = "ignore"));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(), expect(mockDependencies.core.info).not.toHaveBeenCalledWith(expect.stringContaining("Patch file is empty")), expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
        }),
        it("should handle empty patch with error behavior", async () => {
          (mockPatchContent(mockDependencies, ""), (mockDependencies.process.env.GH_AW_PR_IF_NO_CHANGES = "error"));
          const mainFunction = createMainFunction(mockDependencies);
          (await expect(mainFunction()).rejects.toThrow("No changes to push - failing as configured by if-no-changes: error"), expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
        }),
        it("should handle missing patch file with warn behavior", async () => {
          (mockDependencies.fs.existsSync.mockReturnValue(!1), (mockDependencies.process.env.GH_AW_PR_IF_NO_CHANGES = "warn"));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(), expect(mockDependencies.core.warning).toHaveBeenCalledWith("No patch file found - cannot create pull request without changes"), expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
        }),
        it("should handle missing patch file with ignore behavior", async () => {
          (mockDependencies.fs.existsSync.mockReturnValue(!1), (mockDependencies.process.env.GH_AW_PR_IF_NO_CHANGES = "ignore"));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(), expect(mockDependencies.core.info).not.toHaveBeenCalledWith(expect.stringContaining("No patch file found")), expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
        }),
        it("should handle missing patch file with error behavior", async () => {
          (mockDependencies.fs.existsSync.mockReturnValue(!1), (mockDependencies.process.env.GH_AW_PR_IF_NO_CHANGES = "error"));
          const mainFunction = createMainFunction(mockDependencies);
          (await expect(mainFunction()).rejects.toThrow("No patch file found - cannot create pull request without changes"), expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
        }),
        it("should handle patch with error message with warn behavior", async () => {
          (mockPatchContent(mockDependencies, "Failed to generate patch: some error"), (mockDependencies.process.env.GH_AW_PR_IF_NO_CHANGES = "warn"));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(),
            expect(mockDependencies.core.warning).toHaveBeenCalledWith("Patch file contains error message - cannot create pull request without changes"),
            expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
        }),
        it("should default to warn when if-no-changes is not specified", async () => {
          mockPatchContent(mockDependencies, "");
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(), expect(mockDependencies.core.warning).toHaveBeenCalledWith("Patch file is empty - no changes to apply (noop operation)"), expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled());
        }));
    }),
    describe("staged mode functionality", () => {
      (beforeEach(() => {
        ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
          (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
          (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Staged Mode Test PR", body: "This is a test PR for staged mode functionality.", branch: "feature-test" }] })));
      }),
        it("should write step summary instead of creating PR when in staged mode", async () => {
          mockDependencies.process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(),
            expect(mockDependencies.core.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("## 🎭 Staged Mode: Create Pull Request Preview")),
            expect(mockDependencies.core.summary.write).toHaveBeenCalled(),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("📝 Pull request creation preview written to step summary"),
            expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled(),
            expect(mockDependencies.execSync).not.toHaveBeenCalled());
        }),
        it("should include patch information in staged mode summary", async () => {
          ((mockDependencies.process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"), mockPatchContent(mockDependencies, "diff --git a/test.txt b/test.txt\n+added line\n-removed line"));
          const mainFunction = createMainFunction(mockDependencies);
          await mainFunction();
          const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
          (expect(summaryCall).toContain("**Title:** Staged Mode Test PR"),
            expect(summaryCall).toContain("**Branch:** feature-test"),
            expect(summaryCall).toContain("**Base:** main"),
            expect(summaryCall).toContain("**Body:**"),
            expect(summaryCall).toContain("This is a test PR for staged mode functionality."),
            expect(summaryCall).toContain("**Changes:** Patch file exists with"),
            expect(summaryCall).toContain("Show patch preview"),
            expect(summaryCall).toContain("diff --git a/test.txt"));
        }),
        it("should handle empty patch in staged mode", async () => {
          ((mockDependencies.process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"), mockPatchContent(mockDependencies, ""));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(), expect(mockDependencies.core.summary.addRaw).toHaveBeenCalled());
          const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
          (expect(summaryCall).toContain("**Changes:** No changes (empty patch)"), expect(summaryCall).not.toContain("Show patch preview"));
        }),
        it("should use auto-generated branch when no branch specified in staged mode", async () => {
          ((mockDependencies.process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"),
            (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "PR without branch", body: "Test PR body" }] })));
          const mainFunction = createMainFunction(mockDependencies);
          await mainFunction();
          const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
          expect(summaryCall).toContain("**Branch:** auto-generated");
        }),
        it("should not execute git operations in staged mode", async () => {
          mockDependencies.process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(),
            expect(mockDependencies.execSync).not.toHaveBeenCalledWith(expect.stringContaining("git"), expect.anything()),
            expect(mockDependencies.github.rest.pulls.create).not.toHaveBeenCalled(),
            expect(mockDependencies.github.rest.issues.addLabels).not.toHaveBeenCalled(),
            expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("pull_request_number", ""),
            expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("pull_request_url", ""),
            expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_number", ""),
            expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("issue_url", ""),
            expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("branch_name", ""),
            expect(mockDependencies.core.setOutput).toHaveBeenCalledWith("fallback_used", ""));
        }),
        it("should handle missing patch file in staged mode", async () => {
          ((mockDependencies.process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"), mockDependencies.fs.existsSync.mockReturnValue(!1));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(), expect(mockDependencies.core.summary.addRaw).toHaveBeenCalled());
          const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
          (expect(summaryCall).toContain("⚠️ No patch file found"),
            expect(summaryCall).toContain("No patch file found - cannot create pull request without changes"),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("📝 Pull request creation preview written to step summary (no patch file)"));
        }),
        it("should handle patch error in staged mode", async () => {
          ((mockDependencies.process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"), mockPatchContent(mockDependencies, "Failed to generate patch: some error occurred"));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(), expect(mockDependencies.core.summary.addRaw).toHaveBeenCalled());
          const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
          (expect(summaryCall).toContain("⚠️ Patch file contains error"),
            expect(summaryCall).toContain("Patch file contains error message - cannot create pull request without changes"),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("📝 Pull request creation preview written to step summary (patch error)"));
        }),
        it("should validate patch size within limit", async () => {
          ((mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "This is a test PR", branch: "test-branch" }] })),
            (mockDependencies.process.env.GH_AW_MAX_PATCH_SIZE = "10"),
            mockDependencies.fs.existsSync.mockReturnValue(!0));
          const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
          (mockPatchContent(mockDependencies, patchContent), mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 123, html_url: "https://github.com/testowner/testrepo/pull/123" } }));
          const main = createMainFunction(mockDependencies);
          (await main(),
            expect(mockDependencies.core.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 10 KB\)/)),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("Patch size validation passed"));
        }),
        it("should fail when patch size exceeds limit", async () => {
          ((mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "This is a test PR", branch: "test-branch" }] })),
            (mockDependencies.process.env.GH_AW_MAX_PATCH_SIZE = "1"),
            mockDependencies.fs.existsSync.mockReturnValue(!0));
          const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
          mockPatchContent(mockDependencies, patchContent);
          const main = createMainFunction(mockDependencies);
          (await expect(main()).rejects.toThrow(/Patch size \(\d+ KB\) exceeds maximum allowed size \(1 KB\)/), expect(mockDependencies.core.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1 KB\)/)));
        }),
        it("should show staged preview when patch size exceeds limit in staged mode", async () => {
          ((mockDependencies.process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true"),
            (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "This is a test PR", branch: "test-branch" }] })),
            (mockDependencies.process.env.GH_AW_MAX_PATCH_SIZE = "1"),
            mockDependencies.fs.existsSync.mockReturnValue(!0));
          const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
          mockPatchContent(mockDependencies, patchContent);
          const main = createMainFunction(mockDependencies);
          (await main(), expect(mockDependencies.core.summary.addRaw).toHaveBeenCalled());
          const summaryCall = mockDependencies.core.summary.addRaw.mock.calls[0][0];
          (expect(summaryCall).toContain("❌ Patch size exceeded"),
            expect(summaryCall).toContain("exceeds maximum allowed size"),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("📝 Pull request creation preview written to step summary (patch size error)"));
        }),
        it("should use default 1024 KB limit when env var not set", async () => {
          ((mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "This is a test PR", branch: "test-branch" }] })),
            delete mockDependencies.process.env.GH_AW_MAX_PATCH_SIZE,
            mockDependencies.fs.existsSync.mockReturnValue(!0),
            mockPatchContent(mockDependencies, "diff --git a/file.txt b/file.txt\n+new content\n"),
            mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 123, html_url: "https://github.com/testowner/testrepo/pull/123" } }));
          const main = createMainFunction(mockDependencies);
          (await main(),
            expect(mockDependencies.core.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1024 KB\)/)),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("Patch size validation passed"));
        }),
        it("should skip patch size validation for empty patches", async () => {
          ((mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "This is a test PR", branch: "test-branch" }] })),
            (mockDependencies.process.env.GH_AW_MAX_PATCH_SIZE = "1"),
            mockDependencies.fs.existsSync.mockReturnValue(!0),
            mockPatchContent(mockDependencies, ""));
          const main = createMainFunction(mockDependencies);
          (await main(), expect(mockDependencies.core.info).not.toHaveBeenCalledWith(expect.stringMatching(/Patch size:/)));
        }));
    }),
    describe("Patch failure investigation", () => {
      (beforeEach(() => {
        ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
          (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
          (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "This is a test PR" }] })));
      }),
        it("should investigate patch failure by logging git status and failed patch", async () => {
          (mockDependencies.fs.existsSync.mockReturnValue(!0),
            mockPatchContent(mockDependencies, "diff --git a/file.txt b/file.txt\n+new content"),
            (global.exec.exec = vi.fn().mockImplementation(async cmd => {
              if ("string" == typeof cmd && cmd.includes("git am")) throw new Error("Patch does not apply");
              return 0;
            })),
            (global.exec.getExecOutput = vi
              .fn()
              .mockImplementation(async (command, args) =>
                "git" === command && args && "status" === args[0]
                  ? Promise.resolve({ exitCode: 0, stdout: "On branch test-branch\nYour branch is up to date\n", stderr: "" })
                  : "git" === command && args && "am" === args[0] && "--show-current-patch=diff" === args[1]
                    ? Promise.resolve({ exitCode: 0, stdout: "diff --git a/conflicting.txt b/conflicting.txt\n+conflicting line\n", stderr: "" })
                    : Promise.resolve({ exitCode: 0, stdout: "", stderr: "" })
              )));
          const mainFunction = createMainFunction(mockDependencies);
          await mainFunction();
          const gitAmCalls = global.exec.exec.mock.calls.filter(call => call[0] && call[0].includes("git am"));
          (expect(gitAmCalls.length).toBeGreaterThan(0),
            expect(global.exec.getExecOutput).toHaveBeenCalledWith("git", ["status"]),
            expect(global.exec.getExecOutput).toHaveBeenCalledWith("git", ["am", "--show-current-patch=diff"]),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("Investigating patch failure..."),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("Git status output:"),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("On branch test-branch\nYour branch is up to date\n"),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("Failed patch content:"),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("diff --git a/conflicting.txt b/conflicting.txt\n+conflicting line\n"),
            expect(mockDependencies.core.setFailed).toHaveBeenCalledWith("Failed to apply patch"));
        }),
        it("should handle investigation failure gracefully", async () => {
          (mockDependencies.fs.existsSync.mockReturnValue(!0),
            mockPatchContent(mockDependencies, "diff --git a/file.txt b/file.txt\n+new content"),
            (global.exec.exec = vi.fn().mockImplementation(async cmd => {
              if ("string" == typeof cmd && cmd.includes("git am")) throw new Error("Patch does not apply");
              return 0;
            })),
            (global.exec.getExecOutput = vi.fn().mockImplementation(async (command, args) => {
              if ("git" === command) throw new Error("Git command failed");
              return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
            })));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(),
            expect(mockDependencies.core.info).toHaveBeenCalledWith("Investigating patch failure..."),
            expect(mockDependencies.core.warning).toHaveBeenCalledWith("Failed to investigate patch failure: Git command failed"),
            expect(mockDependencies.core.setFailed).toHaveBeenCalledWith("Failed to apply patch"));
        }));
    }),
    describe("activation comment update", () => {
      (it("should update activation comment with PR link when comment_id is provided", async () => {
        ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
          (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
          (mockDependencies.process.env.GH_AW_COMMENT_ID = "123456"),
          (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"),
          (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "Test PR body" }] })),
          mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 42, html_url: "https://github.com/testowner/testrepo/pull/42" } }));
        const mainFunction = createMainFunction(mockDependencies);
        (await mainFunction(),
          expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalled(),
          expect(mockDependencies.updateActivationComment).toHaveBeenCalledWith(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42));
      }),
        it("should update discussion comment with PR link when comment_id starts with DC_", async () => {
          ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
            (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
            (mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF"),
            (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"),
            (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "Test PR body" }] })),
            mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 42, html_url: "https://github.com/testowner/testrepo/pull/42" } }));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(),
            expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalled(),
            expect(mockDependencies.updateActivationComment).toHaveBeenCalledWith(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42));
        }),
        it("should skip updating comment when GH_AW_COMMENT_ID is not set", async () => {
          ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
            (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
            (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "Test PR body" }] })),
            mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 42, html_url: "https://github.com/testowner/testrepo/pull/42" } }));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(),
            expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalled(),
            expect(mockDependencies.updateActivationComment).toHaveBeenCalledWith(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42));
        }),
        it("should not fail workflow if comment update fails", async () => {
          ((mockDependencies.process.env.GH_AW_WORKFLOW_ID = "test-workflow"),
            (mockDependencies.process.env.GH_AW_BASE_BRANCH = "main"),
            (mockDependencies.process.env.GH_AW_COMMENT_ID = "123456"),
            (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"),
            (mockDependencies.process.env.GH_AW_AGENT_OUTPUT = JSON.stringify({ items: [{ type: "create_pull_request", title: "Test PR", body: "Test PR body" }] })),
            mockDependencies.github.rest.pulls.create.mockResolvedValue({ data: { number: 42, html_url: "https://github.com/testowner/testrepo/pull/42" } }));
          const mainFunction = createMainFunction(mockDependencies);
          (await mainFunction(),
            expect(mockDependencies.github.rest.pulls.create).toHaveBeenCalled(),
            expect(mockDependencies.updateActivationComment).toHaveBeenCalledWith(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42));
        }));
    }));
});
