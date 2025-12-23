import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
const mockCore = {
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
  mockContext = { eventName: "pull_request", payload: { pull_request: { number: 123 }, repository: { html_url: "https://github.com/testowner/testrepo" } }, repo: { owner: "testowner", repo: "testrepo" } },
  mockGithub = {
    graphql: vi.fn(),
    request: vi.fn(),
    rest: {
      pulls: {
        get: vi.fn().mockResolvedValue({
          data: {
            head: { ref: "feature-branch" },
            title: "Test PR Title",
            labels: [{ name: "bug" }, { name: "enhancement" }],
          },
        }),
      },
    },
  };
((global.core = mockCore),
  (global.context = mockContext),
  (global.github = mockGithub),
  describe("push_to_pull_request_branch.cjs", () => {
    let pushToPrBranchScript, mockFs, mockExec, tempFilePath;
    const setAgentOutput = data => {
        tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
        const content = "string" == typeof data ? data : JSON.stringify(data);
        (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
      },
      mockPatchContent = patchContent => {
        mockFs.readFileSync.mockImplementation((filepath, encoding) => {
          const agentOutputPath = process.env.GH_AW_AGENT_OUTPUT;
          return agentOutputPath && filepath === agentOutputPath ? fs.readFileSync(filepath, encoding || "utf8") : patchContent;
        });
      },
      executeScript = async () => ((global.core = mockCore), (global.context = mockContext), (global.github = mockGithub), (global.mockFs = mockFs), (global.exec = mockExec), await eval(`(async () => { ${pushToPrBranchScript}; await main(); })()`));
    (beforeEach(() => {
      (vi.clearAllMocks(),
        delete process.env.GH_AW_PUSH_TARGET,
        delete process.env.GH_AW_AGENT_OUTPUT,
        delete process.env.GH_AW_PUSH_IF_NO_CHANGES,
        delete process.env.GH_AW_PR_TITLE_PREFIX,
        delete process.env.GH_AW_PR_LABELS,
        (mockFs = {
          existsSync: vi.fn(),
          readFileSync: vi.fn().mockImplementation((filepath, encoding) => {
            const agentOutputPath = process.env.GH_AW_AGENT_OUTPUT;
            return agentOutputPath && filepath === agentOutputPath ? fs.readFileSync(filepath, encoding || "utf8") : "diff --git a/file.txt b/file.txt\n+new content";
          }),
        }),
        (mockExec = {
          exec: vi.fn().mockResolvedValue(0),
          getExecOutput: vi.fn().mockImplementation((command, args) => {
            return "git" === command && args && "rev-parse" === args[0] && "HEAD" === args[1] ? Promise.resolve({ exitCode: 0, stdout: "abc123def456\n", stderr: "" }) : Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
          }),
        }),
        mockCore.setFailed.mockReset(),
        mockCore.setOutput.mockReset(),
        mockCore.warning.mockReset(),
        mockCore.error.mockReset());
      const scriptPath = path.join(process.cwd(), "push_to_pull_request_branch.cjs");
      ((pushToPrBranchScript = fs.readFileSync(scriptPath, "utf8")),
        (pushToPrBranchScript = pushToPrBranchScript.replace(
          /\/\*\* @type \{typeof import\("fs"\)\} \*\/\nconst fs = require\("fs"\);/,
          "const core = global.core;\nconst context = global.context || {};\nconst fs = global.mockFs;\nconst exec = global.exec;"
        )));
    }),
      afterEach(() => {
        (tempFilePath && require("fs").existsSync(tempFilePath) && (require("fs").unlinkSync(tempFilePath), (tempFilePath = void 0)),
          "undefined" != typeof global && (delete global.core, delete global.context, delete global.mockFs, delete global.exec));
      }),
      describe("Script execution", () => {
        (it("should skip when no agent output is provided", async () => {
          (delete process.env.GH_AW_AGENT_OUTPUT, await executeScript(), expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty"), expect(mockCore.setFailed).not.toHaveBeenCalled());
        }),
          it("should skip when agent output is empty", async () => {
            (setAgentOutput(""), await executeScript(), expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty"), expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should handle missing patch file with default 'warn' behavior", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "test" }] }),
              mockFs.existsSync.mockReturnValue(!1),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith("No patch file found - cannot push without changes"),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should fail when patch file missing and if-no-changes is 'error'", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "test" }] }),
              (process.env.GH_AW_PUSH_IF_NO_CHANGES = "error"),
              mockFs.existsSync.mockReturnValue(!1),
              await executeScript(),
              expect(mockCore.setFailed).toHaveBeenCalledWith("No patch file found - cannot push without changes"));
          }),
          it("should silently succeed when patch file missing and if-no-changes is 'ignore'", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "test" }] }),
              (process.env.GH_AW_PUSH_IF_NO_CHANGES = "ignore"),
              mockFs.existsSync.mockReturnValue(!1),
              await executeScript(),
              expect(mockCore.info).not.toHaveBeenCalled(),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should fail when patch file contains error content", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "test" }] }),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("Failed to generate patch: some error"),
              await executeScript(),
              expect(mockCore.setFailed).toHaveBeenCalledWith("Patch file contains error message - cannot push without changes"),
              expect(mockCore.error).toHaveBeenCalledWith("Patch file generation failed - this is an error condition that requires investigation"),
              expect(mockCore.error).toHaveBeenCalledWith("Patch file location: /tmp/gh-aw/aw.patch"),
              expect(mockCore.error).toHaveBeenCalledWith(expect.stringMatching(/Patch file size: \d+ bytes/)),
              expect(mockCore.error).toHaveBeenCalledWith(expect.stringMatching(/Patch file preview \(first \d+ characters\):/)),
              expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to generate patch: some error")));
          }),
          it("should fail when patch file contains error content regardless of if-no-changes config", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "test" }] }),
              (process.env.GH_AW_PUSH_IF_NO_CHANGES = "ignore"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("Failed to generate patch: git diff failed"),
              await executeScript(),
              expect(mockCore.setFailed).toHaveBeenCalledWith("Patch file contains error message - cannot push without changes"),
              expect(mockCore.error).toHaveBeenCalledWith("Patch file generation failed - this is an error condition that requires investigation"));
          }),
          it("should handle empty patch file with default 'warn' behavior", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "test" }] }),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent(""),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith("Patch file is empty - no changes to apply (noop operation)"),
              expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Agent output content length: \d+/)),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should fail when empty patch and if-no-changes is 'error'", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "test" }] }),
              (process.env.GH_AW_PUSH_IF_NO_CHANGES = "error"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("   "),
              await executeScript(),
              expect(mockCore.setFailed).toHaveBeenCalledWith("No changes to push - failing as configured by if-no-changes: error"));
          }),
          it("should handle valid patch content and parse JSON output", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("diff --git a/file.txt b/file.txt\n+new content"),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Agent output content length: \d+/)),
              expect(mockCore.info).toHaveBeenCalledWith("Patch content validation passed"),
              expect(mockCore.info).toHaveBeenCalledWith("Target configuration: triggering"),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should handle invalid JSON in agent output", async () => {
            const invalidJsonPath = path.join("/tmp", `test_invalid_${Date.now()}.json`);
            (fs.writeFileSync(invalidJsonPath, "invalid json content"),
              (process.env.GH_AW_AGENT_OUTPUT = invalidJsonPath),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("some patch content"),
              await executeScript(),
              expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringMatching(/Error parsing agent output JSON:/)),
              fs.existsSync(invalidJsonPath) && fs.unlinkSync(invalidJsonPath));
          }),
          it("should handle agent output without valid items array", async () => {
            (setAgentOutput({ items: "not an array" }),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("some patch content"),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith("No valid items found in agent output"),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should use custom target configuration", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "test" }] }),
              (process.env.GH_AW_PUSH_TARGET = "custom-target"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("some patch content"),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith("Target configuration: custom-target"));
          }));
      }),
      describe("Script validation", () => {
        (it("should have valid JavaScript syntax", () => {
          const scriptPath = path.join(__dirname, "push_to_pull_request_branch.cjs"),
            scriptContent = fs.readFileSync(scriptPath, "utf8");
          (expect(scriptContent).toContain("async function main()"), expect(scriptContent).toContain("core.setFailed"), expect(scriptContent).toContain("/tmp/gh-aw/aw.patch"), expect(scriptContent).toContain("await main()"));
        }),
          it("should export a main function", () => {
            const scriptPath = path.join(__dirname, "push_to_pull_request_branch.cjs"),
              scriptContent = fs.readFileSync(scriptPath, "utf8");
            expect(scriptContent).toMatch(/async function main\(\) \{[\s\S]*\}/);
          }),
          it("should validate patch size within limit", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }), (process.env.GH_AW_MAX_PATCH_SIZE = "10"), mockFs.existsSync.mockReturnValue(!0));
            const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
            (mockPatchContent(patchContent),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 10 KB\)/)),
              expect(mockCore.info).toHaveBeenCalledWith("Patch size validation passed"),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should fail when patch size exceeds limit", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }), (process.env.GH_AW_MAX_PATCH_SIZE = "1"), mockFs.existsSync.mockReturnValue(!0));
            const patchContent = "diff --git a/file.txt b/file.txt\n+new content\n".repeat(100);
            (mockPatchContent(patchContent),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1 KB\)/)),
              expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringMatching(/Patch size \(\d+ KB\) exceeds maximum allowed size \(1 KB\)/)));
          }),
          it("should use default 1024 KB limit when env var not set", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              delete process.env.GH_AW_MAX_PATCH_SIZE,
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("diff --git a/file.txt b/file.txt\n+new content\n"),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith(expect.stringMatching(/Patch size: \d+ KB \(maximum allowed: 1024 KB\)/)),
              expect(mockCore.info).toHaveBeenCalledWith("Patch size validation passed"),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should skip patch size validation for empty patches", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_MAX_PATCH_SIZE = "1"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent(""),
              await executeScript(),
              expect(mockCore.info).not.toHaveBeenCalledWith(expect.stringMatching(/Patch size:/)),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should validate PR title prefix when specified", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_PR_TITLE_PREFIX = "[bot] "),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("diff --git a/file.txt b/file.txt\n+new content"),
              mockGithub.rest.pulls.get.mockResolvedValueOnce({
                data: {
                  head: { ref: "feature-branch" },
                  title: "[bot] Add new feature",
                  labels: [],
                },
              }),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith('✓ Title prefix validation passed: "[bot] "'),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should fail when PR title doesn't match required prefix", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_PR_TITLE_PREFIX = "[bot] "),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("diff --git a/file.txt b/file.txt\n+new content"),
              mockGithub.rest.pulls.get.mockResolvedValueOnce({
                data: {
                  head: { ref: "feature-branch" },
                  title: "Add new feature",
                  labels: [],
                },
              }),
              await executeScript(),
              expect(mockCore.setFailed).toHaveBeenCalledWith('Pull request title "Add new feature" does not start with required prefix "[bot] "'));
          }),
          it("should validate PR labels when specified", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_PR_LABELS = "automation,enhancement"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("diff --git a/file.txt b/file.txt\n+new content"),
              mockGithub.rest.pulls.get.mockResolvedValueOnce({
                data: {
                  head: { ref: "feature-branch" },
                  title: "Add new feature",
                  labels: [{ name: "automation" }, { name: "enhancement" }, { name: "feature" }],
                },
              }),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith("✓ Labels validation passed: automation,enhancement"),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }),
          it("should fail when PR is missing required labels", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_PR_LABELS = "automation,enhancement"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("diff --git a/file.txt b/file.txt\n+new content"),
              mockGithub.rest.pulls.get.mockResolvedValueOnce({
                data: {
                  head: { ref: "feature-branch" },
                  title: "Add new feature",
                  labels: [{ name: "feature" }],
                },
              }),
              await executeScript(),
              expect(mockCore.setFailed).toHaveBeenCalledWith("Pull request is missing required labels: automation, enhancement. Current labels: feature"));
          }),
          it("should validate both title prefix and labels when both are specified", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_PR_TITLE_PREFIX = "[automated] "),
              (process.env.GH_AW_PR_LABELS = "bot,feature"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("diff --git a/file.txt b/file.txt\n+new content"),
              mockGithub.rest.pulls.get.mockResolvedValueOnce({
                data: {
                  head: { ref: "feature-branch" },
                  title: "[automated] Add new feature",
                  labels: [{ name: "bot" }, { name: "feature" }, { name: "enhancement" }],
                },
              }),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith('✓ Title prefix validation passed: "[automated] "'),
              expect(mockCore.info).toHaveBeenCalledWith("✓ Labels validation passed: bot,feature"),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          }));
      }),
      describe("Commit title suffix", () => {
        (beforeEach(() => {
          mockFs.writeFileSync = vi.fn();
        }),
          it("should append bracketed suffix (for [skip-ci] support)", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_COMMIT_TITLE_SUFFIX = " [skip-ci]"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("From abc123 Mon Sep 17 00:00:00 2001\nFrom: Test User <test@example.com>\nDate: Mon, 1 Jan 2024 00:00:00 +0000\nSubject: [PATCH] Add new feature\n\n---\n file.txt | 1 +\n 1 file changed, 1 insertion(+)\n"),
              await executeScript(),
              expect(mockFs.writeFileSync).toHaveBeenCalled());
            const writtenPatch = mockFs.writeFileSync.mock.calls[0][1];
            (expect(writtenPatch).toContain("Subject: [PATCH] Add new feature [skip-ci]"), expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining('commit title suffix: " [skip-ci]"')));
          }),
          it("should append suffix without brackets as-is", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_COMMIT_TITLE_SUFFIX = " (automated)"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("From abc123 Mon Sep 17 00:00:00 2001\nFrom: Test User <test@example.com>\nDate: Mon, 1 Jan 2024 00:00:00 +0000\nSubject: [PATCH] Add new feature\n\n---\n file.txt | 1 +\n 1 file changed, 1 insertion(+)\n"),
              await executeScript(),
              expect(mockFs.writeFileSync).toHaveBeenCalled());
            const writtenPatch = mockFs.writeFileSync.mock.calls[0][1];
            (expect(writtenPatch).toContain("Subject: [PATCH] Add new feature (automated)"), expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining('commit title suffix: " (automated)"')));
          }),
          it("should append suffix with brackets", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_COMMIT_TITLE_SUFFIX = " [bot]"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("From abc123 Mon Sep 17 00:00:00 2001\nFrom: Test User <test@example.com>\nDate: Mon, 1 Jan 2024 00:00:00 +0000\nSubject: [PATCH] Add new feature\n\n---\n file.txt | 1 +\n 1 file changed, 1 insertion(+)\n"),
              await executeScript(),
              expect(mockFs.writeFileSync).toHaveBeenCalled());
            const writtenPatch = mockFs.writeFileSync.mock.calls[0][1];
            (expect(writtenPatch).toContain("Subject: [PATCH] Add new feature [bot]"), expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining('commit title suffix: " [bot]"')));
          }),
          it("should not modify patch when no commit title suffix is set", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              delete process.env.GH_AW_COMMIT_TITLE_SUFFIX,
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("From abc123 Mon Sep 17 00:00:00 2001\nFrom: Test User <test@example.com>\nDate: Mon, 1 Jan 2024 00:00:00 +0000\nSubject: [PATCH] Add new feature\n\n---\n file.txt | 1 +\n 1 file changed, 1 insertion(+)\n"),
              await executeScript(),
              expect(mockFs.writeFileSync).not.toHaveBeenCalled());
          }),
          it("should handle patch without [PATCH] prefix in Subject line", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_COMMIT_TITLE_SUFFIX = " [automated]"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("From abc123 Mon Sep 17 00:00:00 2001\nFrom: Test User <test@example.com>\nDate: Mon, 1 Jan 2024 00:00:00 +0000\nSubject: Add new feature\n\n---\n file.txt | 1 +\n 1 file changed, 1 insertion(+)\n"),
              await executeScript(),
              expect(mockFs.writeFileSync).toHaveBeenCalled());
            const writtenPatch = mockFs.writeFileSync.mock.calls[0][1];
            expect(writtenPatch).toContain("Subject: [PATCH] Add new feature [automated]");
          }),
          it("should use git am without --keep-non-patch flag", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              (process.env.GH_AW_COMMIT_TITLE_SUFFIX = " [skip-ci]"),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("diff --git a/file.txt b/file.txt\n+new content"),
              await executeScript(),
              expect(mockExec.exec).toHaveBeenCalledWith("git am /tmp/gh-aw/aw.patch"));
          }));
      }),
      describe("Patch failure investigation", () => {
        (beforeEach(() => {
          mockFs.writeFileSync = vi.fn();
        }),
          it("should investigate patch failure by logging git status and failed patch", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }), mockFs.existsSync.mockReturnValue(!0), mockPatchContent("diff --git a/file.txt b/file.txt\n+new content"));
            let gitAmCalled = !1;
            (mockExec.exec.mockImplementation(async (cmd, args) => {
              if ("string" == typeof cmd && cmd.includes("git am")) throw ((gitAmCalled = !0), new Error("Patch does not apply"));
              return 0;
            }),
              mockGithub.rest.pulls.get.mockResolvedValueOnce({
                data: {
                  head: { ref: "feature-branch" },
                  title: "Test PR Title",
                  labels: [{ name: "bug" }, { name: "enhancement" }],
                },
              }),
              mockExec.getExecOutput.mockImplementation(async (command, args) => {
                return "git" === command && args && "status" === args[0]
                  ? Promise.resolve({ exitCode: 0, stdout: "On branch feature-branch\nYour branch is up to date\n", stderr: "" })
                  : "git" === command && args && "log" === args[0] && "--oneline" === args[1] && "-5" === args[2]
                    ? Promise.resolve({ exitCode: 0, stdout: "abc123 Latest commit\ndef456 Previous commit\n", stderr: "" })
                    : "git" === command && args && "diff" === args[0] && "HEAD" === args[1]
                      ? Promise.resolve({ exitCode: 0, stdout: "diff --git a/modified.txt b/modified.txt\n+modified content\n", stderr: "" })
                      : "git" === command && args && "am" === args[0] && "--show-current-patch=diff" === args[1]
                        ? Promise.resolve({ exitCode: 0, stdout: "diff --git a/conflicting.txt b/conflicting.txt\n+conflicting line\n", stderr: "" })
                        : "git" === command && args && "am" === args[0] && "--show-current-patch" === args[1]
                          ? Promise.resolve({ exitCode: 0, stdout: "From abc123 Mon Sep 17 00:00:00 2001\nSubject: [PATCH] Add feature\n", stderr: "" })
                          : Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
              }),
              await executeScript(),
              expect(gitAmCalled).toBe(!0),
              expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["status"]),
              expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["log", "--oneline", "-5"]),
              expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["diff", "HEAD"]),
              expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["am", "--show-current-patch=diff"]),
              expect(mockExec.getExecOutput).toHaveBeenCalledWith("git", ["am", "--show-current-patch"]),
              expect(mockCore.info).toHaveBeenCalledWith("Investigating patch failure..."),
              expect(mockCore.info).toHaveBeenCalledWith("Git status output:"),
              expect(mockCore.info).toHaveBeenCalledWith("On branch feature-branch\nYour branch is up to date\n"),
              expect(mockCore.info).toHaveBeenCalledWith("Recent commits (last 5):"),
              expect(mockCore.info).toHaveBeenCalledWith("abc123 Latest commit\ndef456 Previous commit\n"),
              expect(mockCore.info).toHaveBeenCalledWith("Uncommitted changes:"),
              expect(mockCore.info).toHaveBeenCalledWith("diff --git a/modified.txt b/modified.txt\n+modified content\n"),
              expect(mockCore.info).toHaveBeenCalledWith("Failed patch diff:"),
              expect(mockCore.info).toHaveBeenCalledWith("diff --git a/conflicting.txt b/conflicting.txt\n+conflicting line\n"),
              expect(mockCore.info).toHaveBeenCalledWith("Failed patch (full):"),
              expect(mockCore.info).toHaveBeenCalledWith("From abc123 Mon Sep 17 00:00:00 2001\nSubject: [PATCH] Add feature\n"),
              expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to apply patch"));
          }),
          it("should handle investigation failure gracefully", async () => {
            (setAgentOutput({ items: [{ type: "push_to_pull_request_branch", content: "some changes to push" }] }),
              mockFs.existsSync.mockReturnValue(!0),
              mockPatchContent("diff --git a/file.txt b/file.txt\n+new content"),
              mockExec.exec.mockImplementation(async (cmd, args) => {
                if ("string" == typeof cmd && cmd.includes("git am")) throw new Error("Patch does not apply");
                return 0;
              }),
              mockGithub.rest.pulls.get.mockResolvedValueOnce({
                data: {
                  head: { ref: "feature-branch" },
                  title: "Test PR Title",
                  labels: [],
                },
              }),
              mockExec.getExecOutput.mockImplementation(async (command, args) => {
                if ("git" === command) throw new Error("Git command failed");
                return Promise.resolve({ exitCode: 0, stdout: "", stderr: "" });
              }),
              await executeScript(),
              expect(mockCore.info).toHaveBeenCalledWith("Investigating patch failure..."),
              expect(mockCore.warning).toHaveBeenCalledWith("Failed to investigate patch failure: Git command failed"),
              expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to apply patch"));
          }));
      }));
  }));
