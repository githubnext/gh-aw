import { describe, it, expect, beforeEach, vi } from "vitest";
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

  // Input/state functions (less commonly used but included for completeness)
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

const mockContext = {
  repo: { owner: "test-owner", repo: "test-repo" },
  runId: "12345",
  payload: {
    repository: {
      html_url: "https://github.com/test-owner/test-repo",
    },
  },
};

const mockGithub = {
  rest: {
    repos: {
      createCommitStatus: vi.fn(),
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;

describe("commit_status.cjs", () => {
  let commitStatusScript;
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = `/tmp/gh-aw-test-${Date.now()}.json`;
    fs.writeFileSync(tempFilePath, JSON.stringify(data), "utf8");
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset mocks before each test
    vi.clearAllMocks();
    // Reset environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_COMMIT_SHA;
    delete process.env.GH_AW_COMMIT_STATUS_CONTEXT;

    // Clean up any leftover temp file
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
    }

    // Read the script file
    const scriptPath = path.join(__dirname, "commit_status.cjs");
    commitStatusScript = fs.readFileSync(scriptPath, "utf8");

    // Make fs available globally for the evaluated script
    global.fs = fs;
  });

  afterEach(() => {
    // Clean up temp file
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
    }
    delete global.fs;
  });

  it("should handle missing agent output", async () => {
    // No GH_AW_AGENT_OUTPUT set
    await eval(`(async () => { ${commitStatusScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
  });

  it("should handle empty agent output", async () => {
    setAgentOutput({ items: [], errors: [] });

    await eval(`(async () => { ${commitStatusScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No commit_status items found in agent output");
  });

  it("should show staged preview when staged mode is enabled", async () => {
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
    setAgentOutput({
      items: [
        {
          type: "commit_status",
          state: "success",
          description: "All checks passed",
          context: "ci/test",
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${commitStatusScript} })()`);

    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
    expect(mockGithub.rest.repos.createCommitStatus).not.toHaveBeenCalled();
  });

  it("should skip gracefully when GH_AW_COMMIT_SHA is not set", async () => {
    setAgentOutput({
      items: [
        {
          type: "commit_status",
          state: "success",
          description: "Test passed",
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${commitStatusScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No commit SHA available - skipping commit status update");
    expect(mockGithub.rest.repos.createCommitStatus).not.toHaveBeenCalled();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should update commit status successfully", async () => {
    setAgentOutput({
      items: [
        {
          type: "commit_status",
          state: "success",
          description: "All tests passed",
          context: "ci/tests",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_COMMIT_SHA = "abc123def456";

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { id: 1 },
    });

    await eval(`(async () => { ${commitStatusScript} })()`);

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      sha: "abc123def456",
      state: "success",
      description: "All tests passed",
      context: "ci/tests",
      target_url: "https://github.com/test-owner/test-repo/actions/runs/12345",
    });

    expect(mockCore.info).toHaveBeenCalledWith("Successfully updated commit status to 'success'");
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should use default context when not specified", async () => {
    setAgentOutput({
      items: [
        {
          type: "commit_status",
          state: "failure",
          description: "Tests failed",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_COMMIT_SHA = "abc123def456";
    process.env.GH_AW_COMMIT_STATUS_CONTEXT = "custom-workflow";

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { id: 1 },
    });

    await eval(`(async () => { ${commitStatusScript} })()`);

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        context: "custom-workflow",
      })
    );
  });

  it("should handle API errors gracefully", async () => {
    setAgentOutput({
      items: [
        {
          type: "commit_status",
          state: "error",
          description: "Build error",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_COMMIT_SHA = "abc123def456";

    const apiError = new Error("API rate limit exceeded");
    mockGithub.rest.repos.createCommitStatus.mockRejectedValue(apiError);

    await eval(`(async () => { ${commitStatusScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith("Failed to update commit status: API rate limit exceeded");
    expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to update commit status: API rate limit exceeded");
  });

  it("should process multiple commit status items", async () => {
    setAgentOutput({
      items: [
        {
          type: "commit_status",
          state: "success",
          description: "Unit tests passed",
          context: "ci/unit-tests",
        },
        {
          type: "commit_status",
          state: "success",
          description: "Integration tests passed",
          context: "ci/integration-tests",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_COMMIT_SHA = "abc123def456";

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { id: 1 },
    });

    await eval(`(async () => { ${commitStatusScript} })()`);

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledTimes(2);
    expect(mockCore.info).toHaveBeenCalledWith("Successfully updated commit status to 'success'");
  });
});
