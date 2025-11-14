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

// Mock GitHub context and API
const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  sha: "abc123def456",
  ref: "refs/heads/main",
};

const mockGithub = {
  rest: {
    repos: {
      getContent: vi.fn(),
      listCommits: vi.fn(),
    },
  },
};

// Set up global variables
global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;

describe("check_workflow_timestamp.cjs", () => {
  let checkWorkflowTimestampScript;
  let originalEnv;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = {
      GH_AW_WORKFLOW_FILE: process.env.GH_AW_WORKFLOW_FILE,
    };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "check_workflow_timestamp.cjs");
    checkWorkflowTimestampScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv.GH_AW_WORKFLOW_FILE !== undefined) {
      process.env.GH_AW_WORKFLOW_FILE = originalEnv.GH_AW_WORKFLOW_FILE;
    } else {
      delete process.env.GH_AW_WORKFLOW_FILE;
    }
  });

  describe("when environment variables are missing", () => {
    it("should fail if GH_AW_WORKFLOW_FILE is not set", async () => {
      delete process.env.GH_AW_WORKFLOW_FILE;

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_WORKFLOW_FILE not available"));
    });
  });

  describe("when files do not exist in repository", () => {
    it("should skip check when source file does not exist", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";

      // Mock source file not found (404)
      mockGithub.rest.repos.getContent.mockImplementation(params => {
        if (params.path === ".github/workflows/test.md") {
          const error = new Error("Not Found");
          error.status = 404;
          throw error;
        }
        return Promise.resolve({ data: { sha: "lock123" } });
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Source file does not exist"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping timestamp check"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.error).not.toHaveBeenCalled();
    });

    it("should skip check when lock file does not exist", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";

      // Mock lock file not found (404)
      mockGithub.rest.repos.getContent.mockImplementation(params => {
        if (params.path === ".github/workflows/test.lock.yml") {
          const error = new Error("Not Found");
          error.status = 404;
          throw error;
        }
        return Promise.resolve({ data: { sha: "workflow123" } });
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Lock file does not exist"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping timestamp check"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.error).not.toHaveBeenCalled();
    });

    it("should skip check when both files do not exist", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";

      // Mock both files not found (404)
      mockGithub.rest.repos.getContent.mockImplementation(() => {
        const error = new Error("Not Found");
        error.status = 404;
        throw error;
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping timestamp check"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.error).not.toHaveBeenCalled();
    });
  });

  describe("when lock file is up to date", () => {
    it("should pass when lock file commit is newer than source file", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";

      // Mock both files exist
      mockGithub.rest.repos.getContent.mockResolvedValue({ data: { sha: "abc123" } });

      // Mock commits - source older, lock newer
      const oldDate = new Date("2024-01-01T10:00:00Z");
      const newDate = new Date("2024-01-02T10:00:00Z");

      mockGithub.rest.repos.listCommits.mockImplementation(params => {
        if (params.path === ".github/workflows/test.md") {
          return Promise.resolve({
            data: [
              {
                sha: "workflow123",
                commit: {
                  committer: { date: oldDate.toISOString() },
                },
              },
            ],
          });
        } else {
          return Promise.resolve({
            data: [
              {
                sha: "lock123",
                commit: {
                  committer: { date: newDate.toISOString() },
                },
              },
            ],
          });
        }
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Lock file is up to date"));
      expect(mockCore.error).not.toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    });

    it("should pass when lock file has same commit timestamp as source file", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";

      // Mock both files exist
      mockGithub.rest.repos.getContent.mockResolvedValue({ data: { sha: "abc123" } });

      // Mock commits - same timestamp
      const sameDate = new Date("2024-01-01T10:00:00Z");

      mockGithub.rest.repos.listCommits.mockResolvedValue({
        data: [
          {
            sha: "commit123",
            commit: {
              committer: { date: sameDate.toISOString() },
            },
          },
        ],
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Lock file is up to date"));
      expect(mockCore.error).not.toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    });
  });

  describe("when lock file is outdated", () => {
    it("should warn when source file commit is newer than lock file", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";

      // Mock both files exist
      mockGithub.rest.repos.getContent.mockResolvedValue({ data: { sha: "abc123" } });

      // Mock commits - source newer, lock older
      const oldDate = new Date("2024-01-01T10:00:00Z");
      const newDate = new Date("2024-01-02T10:00:00Z");

      mockGithub.rest.repos.listCommits.mockImplementation(params => {
        if (params.path === ".github/workflows/test.md") {
          return Promise.resolve({
            data: [
              {
                sha: "workflow123",
                commit: {
                  committer: { date: newDate.toISOString() },
                },
              },
            ],
          });
        } else {
          return Promise.resolve({
            data: [
              {
                sha: "lock123",
                commit: {
                  committer: { date: oldDate.toISOString() },
                },
              },
            ],
          });
        }
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("WARNING: Lock file"));
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("is outdated"));
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("gh aw compile"));
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should include file paths in warning message", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "my-workflow.lock.yml";

      // Mock both files exist
      mockGithub.rest.repos.getContent.mockResolvedValue({ data: { sha: "abc123" } });

      // Mock commits - source newer
      const oldDate = new Date("2024-01-01T10:00:00Z");
      const newDate = new Date("2024-01-02T10:00:00Z");

      mockGithub.rest.repos.listCommits.mockImplementation(params => {
        if (params.path === ".github/workflows/my-workflow.md") {
          return Promise.resolve({
            data: [
              {
                sha: "workflow123",
                commit: {
                  committer: { date: newDate.toISOString() },
                },
              },
            ],
          });
        } else {
          return Promise.resolve({
            data: [
              {
                sha: "lock123",
                commit: {
                  committer: { date: oldDate.toISOString() },
                },
              },
            ],
          });
        }
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringMatching(/WARNING.*my-workflow\.lock\.yml.*outdated/));
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringMatching(/my-workflow\.md/));
    });

    it("should add step summary with warning", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";

      // Mock both files exist
      mockGithub.rest.repos.getContent.mockResolvedValue({ data: { sha: "abc123" } });

      // Mock commits - source newer
      const oldDate = new Date("2024-01-01T10:00:00Z");
      const newDate = new Date("2024-01-02T10:00:00Z");

      mockGithub.rest.repos.listCommits.mockImplementation(params => {
        if (params.path === ".github/workflows/test.md") {
          return Promise.resolve({
            data: [
              {
                sha: "workflow123",
                commit: {
                  committer: { date: newDate.toISOString() },
                },
              },
            ],
          });
        } else {
          return Promise.resolve({
            data: [
              {
                sha: "lock123",
                commit: {
                  committer: { date: oldDate.toISOString() },
                },
              },
            ],
          });
        }
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Workflow Lock File Warning"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("WARNING"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("gh aw compile"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should include git SHA in summary", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";

      // Mock both files exist
      mockGithub.rest.repos.getContent.mockResolvedValue({ data: { sha: "abc123" } });

      // Mock commits - source newer
      const oldDate = new Date("2024-01-01T10:00:00Z");
      const newDate = new Date("2024-01-02T10:00:00Z");

      mockGithub.rest.repos.listCommits.mockImplementation(params => {
        if (params.path === ".github/workflows/test.md") {
          return Promise.resolve({
            data: [
              {
                sha: "workflow123",
                commit: {
                  committer: { date: newDate.toISOString() },
                },
              },
            ],
          });
        } else {
          return Promise.resolve({
            data: [
              {
                sha: "lock123",
                commit: {
                  committer: { date: oldDate.toISOString() },
                },
              },
            ],
          });
        }
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Git Commit"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("abc123def456"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should include file timestamps in summary", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";

      // Mock both files exist
      mockGithub.rest.repos.getContent.mockResolvedValue({ data: { sha: "abc123" } });

      // Mock commits - source newer
      const oldDate = new Date("2024-01-01T10:00:00Z");
      const newDate = new Date("2024-01-02T10:00:00Z");

      mockGithub.rest.repos.listCommits.mockImplementation(params => {
        if (params.path === ".github/workflows/test.md") {
          return Promise.resolve({
            data: [
              {
                sha: "workflow123",
                commit: {
                  committer: { date: newDate.toISOString() },
                },
              },
            ],
          });
        } else {
          return Promise.resolve({
            data: [
              {
                sha: "lock123",
                commit: {
                  committer: { date: oldDate.toISOString() },
                },
              },
            ],
          });
        }
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("last modified:"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringMatching(/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });

  describe("with different workflow names", () => {
    it("should handle workflow names with hyphens", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "my-test-workflow.lock.yml";

      // Mock both files exist
      mockGithub.rest.repos.getContent.mockResolvedValue({ data: { sha: "abc123" } });

      // Mock commits - same timestamp
      const sameDate = new Date("2024-01-01T10:00:00Z");

      mockGithub.rest.repos.listCommits.mockResolvedValue({
        data: [
          {
            sha: "commit123",
            commit: {
              committer: { date: sameDate.toISOString() },
            },
          },
        ],
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("my-test-workflow.md"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle workflow names with underscores", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "my_test_workflow.lock.yml";

      // Mock both files exist
      mockGithub.rest.repos.getContent.mockResolvedValue({ data: { sha: "abc123" } });

      // Mock commits - same timestamp
      const sameDate = new Date("2024-01-01T10:00:00Z");

      mockGithub.rest.repos.listCommits.mockResolvedValue({
        data: [
          {
            sha: "commit123",
            commit: {
              committer: { date: sameDate.toISOString() },
            },
          },
        ],
      });

      await eval(`(async () => { ${checkWorkflowTimestampScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("my_test_workflow.md"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });
});
