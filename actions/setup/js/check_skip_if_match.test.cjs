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
    setCancelled: vi.fn(),
    setError: vi.fn(),
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
  mockGithub = { rest: { search: { issuesAndPullRequests: vi.fn() } } },
  mockContext = { repo: { owner: "testowner", repo: "testrepo" } };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("check_skip_if_match.cjs", () => {
    let checkSkipIfMatchScript, originalEnv;
    (beforeEach(() => {
      (vi.clearAllMocks(), (originalEnv = { GH_AW_SKIP_QUERY: process.env.GH_AW_SKIP_QUERY, GH_AW_WORKFLOW_NAME: process.env.GH_AW_WORKFLOW_NAME, GH_AW_SKIP_MAX_MATCHES: process.env.GH_AW_SKIP_MAX_MATCHES }));
      const scriptPath = path.join(process.cwd(), "check_skip_if_match.cjs");
      checkSkipIfMatchScript = fs.readFileSync(scriptPath, "utf8");
    }),
      afterEach(() => {
        (void 0 !== originalEnv.GH_AW_SKIP_QUERY ? (process.env.GH_AW_SKIP_QUERY = originalEnv.GH_AW_SKIP_QUERY) : delete process.env.GH_AW_SKIP_QUERY,
          void 0 !== originalEnv.GH_AW_WORKFLOW_NAME ? (process.env.GH_AW_WORKFLOW_NAME = originalEnv.GH_AW_WORKFLOW_NAME) : delete process.env.GH_AW_WORKFLOW_NAME,
          void 0 !== originalEnv.GH_AW_SKIP_MAX_MATCHES ? (process.env.GH_AW_SKIP_MAX_MATCHES = originalEnv.GH_AW_SKIP_MAX_MATCHES) : delete process.env.GH_AW_SKIP_MAX_MATCHES);
      }),
      describe("when skip query is not configured", () => {
        (it("should fail if GH_AW_SKIP_QUERY is not set", async () => {
          (delete process.env.GH_AW_SKIP_QUERY,
            (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
            await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
            expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_SKIP_QUERY not specified")),
            expect(mockCore.setOutput).not.toHaveBeenCalled());
        }),
          it("should fail if GH_AW_WORKFLOW_NAME is not set", async () => {
            ((process.env.GH_AW_SKIP_QUERY = "is:issue is:open"),
              delete process.env.GH_AW_WORKFLOW_NAME,
              await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
              expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_WORKFLOW_NAME not specified")),
              expect(mockCore.setOutput).not.toHaveBeenCalled());
          }));
      }),
      describe("when search returns no matches", () => {
        it("should allow execution", async () => {
          ((process.env.GH_AW_SKIP_QUERY = "is:issue is:open label:nonexistent"),
            (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
            mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({ data: { total_count: 0, items: [] } }),
            await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
            expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({ q: "is:issue is:open label:nonexistent repo:testowner/testrepo", per_page: 1 }),
            expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("below threshold")),
            expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "true"),
            expect(mockCore.setFailed).not.toHaveBeenCalled());
        });
      }),
      describe("when search returns matches", () => {
        (it("should set skip_check_ok to false", async () => {
          ((process.env.GH_AW_SKIP_QUERY = "is:issue is:open label:bug"),
            (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
            mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({ data: { total_count: 5, items: [{ id: 1, title: "Test Issue" }] } }),
            await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
            expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({ q: "is:issue is:open label:bug repo:testowner/testrepo", per_page: 1 }),
            expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Skip condition matched")),
            expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("5 items found")),
            expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "false"),
            expect(mockCore.setFailed).not.toHaveBeenCalled());
        }),
          it("should handle single match", async () => {
            ((process.env.GH_AW_SKIP_QUERY = "is:pr is:open"),
              (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
              mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({ data: { total_count: 1, items: [{ id: 1, title: "Test PR" }] } }),
              await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("1 items found")),
              expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "false"));
          }));
      }),
      describe("when search API fails", () => {
        it("should fail with error message", async () => {
          ((process.env.GH_AW_SKIP_QUERY = "is:issue"), (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"));
          const errorMessage = "API rate limit exceeded";
          (mockGithub.rest.search.issuesAndPullRequests.mockRejectedValue(new Error(errorMessage)),
            await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
            expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to execute search query")),
            expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining(errorMessage)),
            expect(mockCore.setOutput).not.toHaveBeenCalled());
        });
      }),
      describe("query scoping", () => {
        it("should automatically scope query to current repository", async () => {
          ((process.env.GH_AW_SKIP_QUERY = "is:issue label:enhancement"),
            (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
            mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({ data: { total_count: 0, items: [] } }),
            await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
            expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({ q: "is:issue label:enhancement repo:testowner/testrepo", per_page: 1 }));
        });
      }),
      describe("max matches parameter", () => {
        (it("should default to 1 if GH_AW_SKIP_MAX_MATCHES is not set", async () => {
          ((process.env.GH_AW_SKIP_QUERY = "is:issue is:open"),
            (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
            delete process.env.GH_AW_SKIP_MAX_MATCHES,
            mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({ data: { total_count: 1, items: [{ id: 1 }] } }),
            await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
            expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("threshold: 1")),
            expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "false"));
        }),
          it("should skip when matches reach threshold", async () => {
            ((process.env.GH_AW_SKIP_QUERY = "is:pr is:open"),
              (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
              (process.env.GH_AW_SKIP_MAX_MATCHES = "3"),
              mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({ data: { total_count: 3, items: [{ id: 1 }] } }),
              await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("3 items found")),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("threshold: 3")),
              expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "false"));
          }),
          it("should skip when matches exceed threshold", async () => {
            ((process.env.GH_AW_SKIP_QUERY = "is:pr is:open"),
              (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
              (process.env.GH_AW_SKIP_MAX_MATCHES = "2"),
              mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({ data: { total_count: 5, items: [{ id: 1 }] } }),
              await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("5 items found")),
              expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("threshold: 2")),
              expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "false"));
          }),
          it("should allow execution when matches are below threshold", async () => {
            ((process.env.GH_AW_SKIP_QUERY = "is:issue is:open"),
              (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
              (process.env.GH_AW_SKIP_MAX_MATCHES = "5"),
              mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({ data: { total_count: 2, items: [{ id: 1 }] } }),
              await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
              expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("below threshold of 5")),
              expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "true"),
              expect(mockCore.warning).not.toHaveBeenCalled());
          }),
          it("should fail with invalid max matches value", async () => {
            ((process.env.GH_AW_SKIP_QUERY = "is:issue is:open"),
              (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
              (process.env.GH_AW_SKIP_MAX_MATCHES = "invalid"),
              await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
              expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("must be a positive integer")),
              expect(mockCore.setOutput).not.toHaveBeenCalled());
          }),
          it("should fail with zero or negative max matches", async () => {
            ((process.env.GH_AW_SKIP_QUERY = "is:issue is:open"),
              (process.env.GH_AW_WORKFLOW_NAME = "test-workflow"),
              (process.env.GH_AW_SKIP_MAX_MATCHES = "0"),
              await eval(`(async () => { ${checkSkipIfMatchScript}; await main(); })()`),
              expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("must be a positive integer")),
              expect(mockCore.setOutput).not.toHaveBeenCalled());
          }));
      }));
  }));
