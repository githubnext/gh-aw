import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),

  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  setCancelled: vi.fn(),
  setError: vi.fn(),

  // Input/state functions
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    search: {
      issuesAndPullRequests: vi.fn(),
    },
  },
};

const mockContext = {
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("check_skip_if_match.cjs", () => {
  let checkSkipIfMatchScript;
  let originalEnv;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = {
      GH_AW_SKIP_QUERY: process.env.GH_AW_SKIP_QUERY,
      GH_AW_WORKFLOW_NAME: process.env.GH_AW_WORKFLOW_NAME,
    };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "check_skip_if_match.cjs");
    checkSkipIfMatchScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv.GH_AW_SKIP_QUERY !== undefined) {
      process.env.GH_AW_SKIP_QUERY = originalEnv.GH_AW_SKIP_QUERY;
    } else {
      delete process.env.GH_AW_SKIP_QUERY;
    }
    if (originalEnv.GH_AW_WORKFLOW_NAME !== undefined) {
      process.env.GH_AW_WORKFLOW_NAME = originalEnv.GH_AW_WORKFLOW_NAME;
    } else {
      delete process.env.GH_AW_WORKFLOW_NAME;
    }
  });

  describe("when skip query is not configured", () => {
    it("should fail if GH_AW_SKIP_QUERY is not set", async () => {
      delete process.env.GH_AW_SKIP_QUERY;
      process.env.GH_AW_WORKFLOW_NAME = "test-workflow";

      await eval(`(async () => { ${checkSkipIfMatchScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_SKIP_QUERY not specified"));
      expect(mockCore.setOutput).not.toHaveBeenCalled();
    });

    it("should fail if GH_AW_WORKFLOW_NAME is not set", async () => {
      process.env.GH_AW_SKIP_QUERY = "is:issue is:open";
      delete process.env.GH_AW_WORKFLOW_NAME;

      await eval(`(async () => { ${checkSkipIfMatchScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_WORKFLOW_NAME not specified"));
      expect(mockCore.setOutput).not.toHaveBeenCalled();
    });
  });

  describe("when search returns no matches", () => {
    it("should allow execution", async () => {
      process.env.GH_AW_SKIP_QUERY = "is:issue is:open label:nonexistent";
      process.env.GH_AW_WORKFLOW_NAME = "test-workflow";

      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          total_count: 0,
          items: [],
        },
      });

      await eval(`(async () => { ${checkSkipIfMatchScript} })()`);

      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({
        q: "is:issue is:open label:nonexistent repo:testowner/testrepo",
        per_page: 1,
      });
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No matches found"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "true");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("when search returns matches", () => {
    it("should set skip_check_ok to false", async () => {
      process.env.GH_AW_SKIP_QUERY = "is:issue is:open label:bug";
      process.env.GH_AW_WORKFLOW_NAME = "test-workflow";

      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          total_count: 5,
          items: [{ id: 1, title: "Test Issue" }],
        },
      });

      await eval(`(async () => { ${checkSkipIfMatchScript} })()`);

      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({
        q: "is:issue is:open label:bug repo:testowner/testrepo",
        per_page: 1,
      });
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Skip condition matched"));
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("5 items found"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "false");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle single match", async () => {
      process.env.GH_AW_SKIP_QUERY = "is:pr is:open";
      process.env.GH_AW_WORKFLOW_NAME = "test-workflow";

      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          total_count: 1,
          items: [{ id: 1, title: "Test PR" }],
        },
      });

      await eval(`(async () => { ${checkSkipIfMatchScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("1 items found"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "false");
    });
  });

  describe("when search API fails", () => {
    it("should fail with error message", async () => {
      process.env.GH_AW_SKIP_QUERY = "is:issue";
      process.env.GH_AW_WORKFLOW_NAME = "test-workflow";

      const errorMessage = "API rate limit exceeded";
      mockGithub.rest.search.issuesAndPullRequests.mockRejectedValue(new Error(errorMessage));

      await eval(`(async () => { ${checkSkipIfMatchScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to execute search query"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining(errorMessage));
      expect(mockCore.setOutput).not.toHaveBeenCalled();
    });
  });

  describe("query scoping", () => {
    it("should automatically scope query to current repository", async () => {
      process.env.GH_AW_SKIP_QUERY = "is:issue label:enhancement";
      process.env.GH_AW_WORKFLOW_NAME = "test-workflow";

      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: { total_count: 0, items: [] },
      });

      await eval(`(async () => { ${checkSkipIfMatchScript} })()`);

      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({
        q: "is:issue label:enhancement repo:testowner/testrepo",
        per_page: 1,
      });
    });
  });
});
