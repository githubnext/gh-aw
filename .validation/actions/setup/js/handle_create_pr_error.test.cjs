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
  },
  mockGithub = {
    rest: {
      search: {
        issuesAndPullRequests: vi.fn(),
      },
      issues: {
        createComment: vi.fn(),
        create: vi.fn(),
      },
    },
  },
  mockContext = { repo: { owner: "testowner", repo: "testrepo" } };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("handle_create_pr_error.cjs", () => {
    let scriptContent, originalEnv;
    (beforeEach(() => {
      (vi.clearAllMocks(),
        (originalEnv = {
          CREATE_PR_ERROR_MESSAGE: process.env.CREATE_PR_ERROR_MESSAGE,
          GH_AW_WORKFLOW_NAME: process.env.GH_AW_WORKFLOW_NAME,
          GH_AW_RUN_URL: process.env.GH_AW_RUN_URL,
          GH_AW_WORKFLOW_SOURCE: process.env.GH_AW_WORKFLOW_SOURCE,
          GH_AW_WORKFLOW_SOURCE_URL: process.env.GH_AW_WORKFLOW_SOURCE_URL,
        }));
      const scriptPath = path.join(process.cwd(), "handle_create_pr_error.cjs");
      scriptContent = fs.readFileSync(scriptPath, "utf8");
    }),
      afterEach(() => {
        Object.keys(originalEnv).forEach(key => {
          void 0 !== originalEnv[key] ? (process.env[key] = originalEnv[key]) : delete process.env[key];
        });
      }),
      describe("when no error message is set", () => {
        it("should skip and not call any API", async () => {
          (delete process.env.CREATE_PR_ERROR_MESSAGE,
            await eval(`(async () => { ${scriptContent}; await main(); })()`),
            expect(mockCore.info).toHaveBeenCalledWith("No create_pull_request error message - skipping"),
            expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled());
        });
      }),
      describe("when error is not a permission error", () => {
        it("should skip and not call any API", async () => {
          ((process.env.CREATE_PR_ERROR_MESSAGE = "Some unrelated error"),
            await eval(`(async () => { ${scriptContent}; await main(); })()`),
            expect(mockCore.info).toHaveBeenCalledWith("Not a permission error - skipping"),
            expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled());
        });
      }),
      describe("when it is the permission error", () => {
        beforeEach(() => {
          ((process.env.CREATE_PR_ERROR_MESSAGE = "GitHub Actions is not permitted to create or approve pull requests"),
            (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
            (process.env.GH_AW_RUN_URL = "https://github.com/owner/repo/actions/runs/123"));
        });

        it("should create a new issue when none exists", async () => {
          (mockGithub.rest.search.issuesAndPullRequests.mockResolvedValueOnce({ data: { total_count: 0, items: [] } }),
            mockGithub.rest.issues.create.mockResolvedValueOnce({ data: { number: 42, html_url: "https://github.com/owner/repo/issues/42" } }),
            await eval(`(async () => { ${scriptContent}; await main(); })()`),
            expect(mockGithub.rest.issues.create).toHaveBeenCalledWith(
              expect.objectContaining({
                owner: "testowner",
                repo: "testrepo",
                title: "[aw] GitHub Actions needs permission to create pull requests",
                labels: ["agentic-workflows", "configuration"],
              })
            ),
            expect(mockCore.info).toHaveBeenCalledWith("Created issue #42: https://github.com/owner/repo/issues/42"),
            expect(mockCore.setFailed).not.toHaveBeenCalled());
        });

        it("should add a comment to an existing issue", async () => {
          (mockGithub.rest.search.issuesAndPullRequests.mockResolvedValueOnce({
            data: { total_count: 1, items: [{ number: 10, html_url: "https://github.com/owner/repo/issues/10" }] },
          }),
            mockGithub.rest.issues.createComment.mockResolvedValueOnce({ data: {} }),
            await eval(`(async () => { ${scriptContent}; await main(); })()`),
            expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith(
              expect.objectContaining({
                owner: "testowner",
                repo: "testrepo",
                issue_number: 10,
                body: expect.stringContaining("https://github.com/owner/repo/actions/runs/123"),
              })
            ),
            expect(mockCore.info).toHaveBeenCalledWith("Added comment to existing issue #10"),
            expect(mockCore.setFailed).not.toHaveBeenCalled());
        });

        describe("error handling", () => {
          it("should warn but not fail when search API throws", async () => {
            (mockGithub.rest.search.issuesAndPullRequests.mockRejectedValueOnce(new Error("Rate limit exceeded")),
              await eval(`(async () => { ${scriptContent}; await main(); })()`),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to create or update permission error issue")),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Rate limit exceeded")),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          });

          it("should warn but not fail when issue creation throws", async () => {
            (mockGithub.rest.search.issuesAndPullRequests.mockResolvedValueOnce({ data: { total_count: 0, items: [] } }),
              mockGithub.rest.issues.create.mockRejectedValueOnce(new Error("Forbidden")),
              await eval(`(async () => { ${scriptContent}; await main(); })()`),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to create or update permission error issue")),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Forbidden")),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          });

          it("should warn but not fail when createComment throws", async () => {
            (mockGithub.rest.search.issuesAndPullRequests.mockResolvedValueOnce({
              data: { total_count: 1, items: [{ number: 10 }] },
            }),
              mockGithub.rest.issues.createComment.mockRejectedValueOnce(new Error("Network error")),
              await eval(`(async () => { ${scriptContent}; await main(); })()`),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to create or update permission error issue")),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Network error")),
              expect(mockCore.setFailed).not.toHaveBeenCalled());
          });
        });
      }));
  }));
