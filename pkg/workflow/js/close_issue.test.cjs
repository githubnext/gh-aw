import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
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
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    issues: {
      get: vi.fn(),
      update: vi.fn(),
      createComment: vi.fn(),
    },
  },
};

const mockContext = {
  eventName: "issues",
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    issue: {
      number: 123,
    },
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("close_issue.cjs", () => {
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_CLOSE_ISSUE_LABELS;
    delete process.env.GH_AW_CLOSE_ISSUE_TITLE_PREFIX;

    // Clean up temp file if it exists
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
    }

    // Reset module cache
    vi.resetModules();
  });

  it("should close an issue successfully", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 42,
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    // Mock GitHub API responses
    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 42,
        title: "Test Issue",
        state: "open",
        labels: [],
      },
    });

    mockGithub.rest.issues.update.mockResolvedValue({
      data: {
        number: 42,
        title: "Test Issue",
        html_url: "https://github.com/testowner/testrepo/issues/42",
        state: "closed",
      },
    });

    // Import and run the script
    await import("./close_issue.cjs");

    // Verify issue was closed
    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 42,
      state: "closed",
    });

    // Verify outputs were set
    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", 42);
    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_url", expect.any(String));
  });

  it("should close an issue with comment and state_reason", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 42,
          comment: "This issue has been resolved.",
          state_reason: "completed",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 42,
        title: "Test Issue",
        state: "open",
        labels: [],
      },
    });

    mockGithub.rest.issues.createComment.mockResolvedValue({});
    mockGithub.rest.issues.update.mockResolvedValue({
      data: {
        number: 42,
        title: "Test Issue",
        html_url: "https://github.com/testowner/testrepo/issues/42",
        state: "closed",
        state_reason: "completed",
      },
    });

    await import("./close_issue.cjs");

    // Verify comment was added
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 42,
      body: "This issue has been resolved.",
    });

    // Verify issue was closed with state_reason
    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 42,
      state: "closed",
      state_reason: "completed",
    });
  });

  it("should skip already closed issues", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 42,
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 42,
        title: "Test Issue",
        state: "closed",
        labels: [],
      },
    });

    await import("./close_issue.cjs");

    // Verify issue was not closed again
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("should filter issues by labels", async () => {
    process.env.GH_AW_CLOSE_ISSUE_LABELS = "bug,enhancement";

    const validatedOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 42,
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 42,
        title: "Test Issue",
        state: "open",
        labels: [{ name: "documentation" }],
      },
    });

    await import("./close_issue.cjs");

    // Verify issue was not closed (doesn't have required labels)
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("should close issue with matching label", async () => {
    process.env.GH_AW_CLOSE_ISSUE_LABELS = "bug,enhancement";

    const validatedOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 42,
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 42,
        title: "Test Issue",
        state: "open",
        labels: [{ name: "bug" }],
      },
    });

    mockGithub.rest.issues.update.mockResolvedValue({
      data: {
        number: 42,
        title: "Test Issue",
        html_url: "https://github.com/testowner/testrepo/issues/42",
        state: "closed",
      },
    });

    await import("./close_issue.cjs");

    // Verify issue was closed
    expect(mockGithub.rest.issues.update).toHaveBeenCalled();
  });

  it("should filter issues by title prefix", async () => {
    process.env.GH_AW_CLOSE_ISSUE_TITLE_PREFIX = "[bot]";

    const validatedOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 42,
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 42,
        title: "Test Issue",
        state: "open",
        labels: [],
      },
    });

    await import("./close_issue.cjs");

    // Verify issue was not closed (doesn't have required title prefix)
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("should close issue with matching title prefix", async () => {
    process.env.GH_AW_CLOSE_ISSUE_TITLE_PREFIX = "[bot]";

    const validatedOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 42,
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 42,
        title: "[bot] Test Issue",
        state: "open",
        labels: [],
      },
    });

    mockGithub.rest.issues.update.mockResolvedValue({
      data: {
        number: 42,
        title: "[bot] Test Issue",
        html_url: "https://github.com/testowner/testrepo/issues/42",
        state: "closed",
      },
    });

    await import("./close_issue.cjs");

    // Verify issue was closed
    expect(mockGithub.rest.issues.update).toHaveBeenCalled();
  });

  it("should show preview in staged mode", async () => {
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    const validatedOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 42,
          comment: "Closing this issue",
          state_reason: "completed",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    await import("./close_issue.cjs");

    // Verify no API calls were made
    expect(mockGithub.rest.issues.get).not.toHaveBeenCalled();
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();

    // Verify summary was written
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should handle no close-issue items", async () => {
    const validatedOutput = {
      items: [
        {
          type: "create_issue",
          title: "Test",
          body: "Test body",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    await import("./close_issue.cjs");

    // Verify no API calls were made
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("should handle API errors gracefully", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 42,
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    mockGithub.rest.issues.get.mockRejectedValue(new Error("API Error"));

    await import("./close_issue.cjs");

    // Verify error was logged but script didn't fail
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to close issue"));
  });
});
