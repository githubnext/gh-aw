import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    pulls: {
      requestReviewers: vi.fn().mockResolvedValue({}),
    },
  },
};

const mockContext = {
  eventName: "pull_request",
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    pull_request: {
      number: 123,
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("add_copilot_reviewer", () => {
  beforeEach(() => {
    // Reset all mocks before each test
    vi.clearAllMocks();
    vi.resetModules(); // Reset module cache to allow fresh imports

    // Clear environment variables
    delete process.env.PR_NUMBER;

    // Reset context to default
    global.context = {
      eventName: "pull_request",
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      payload: {
        pull_request: {
          number: 123,
        },
      },
    };
  });

  // Helper function to run the script with main() call
  async function runScript() {
    const { main } = await import("./add_copilot_reviewer.cjs?" + Date.now());
    await main();
  }

  it("should fail when PR_NUMBER is not set", async () => {
    delete process.env.PR_NUMBER;

    await runScript();

    expect(mockCore.setFailed).toHaveBeenCalledWith("PR_NUMBER environment variable is required but not set");
    expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });

  it("should fail when PR_NUMBER is empty", async () => {
    process.env.PR_NUMBER = "   ";

    await runScript();

    expect(mockCore.setFailed).toHaveBeenCalledWith("PR_NUMBER environment variable is required but not set");
    expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });

  it("should fail when PR_NUMBER is not a valid number", async () => {
    process.env.PR_NUMBER = "not-a-number";

    await runScript();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Invalid PR_NUMBER"));
    expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });

  it("should fail when PR_NUMBER is zero", async () => {
    process.env.PR_NUMBER = "0";

    await runScript();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Invalid PR_NUMBER"));
    expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });

  it("should fail when PR_NUMBER is negative", async () => {
    process.env.PR_NUMBER = "-1";

    await runScript();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Invalid PR_NUMBER"));
    expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });

  it("should add copilot as reviewer when PR_NUMBER is valid", async () => {
    process.env.PR_NUMBER = "456";

    await runScript();

    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 456,
      reviewers: ["copilot-pull-request-reviewer[bot]"],
    });
    expect(mockCore.info).toHaveBeenCalledWith("Successfully added Copilot as reviewer to PR #456");
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should handle API errors gracefully", async () => {
    process.env.PR_NUMBER = "123";
    mockGithub.rest.pulls.requestReviewers.mockRejectedValueOnce(new Error("API Error"));

    await runScript();

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to add Copilot as reviewer"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to add Copilot as reviewer"));
  });

  it("should trim whitespace from PR_NUMBER", async () => {
    process.env.PR_NUMBER = "  789  ";

    await runScript();

    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 789,
      reviewers: ["copilot-pull-request-reviewer[bot]"],
    });
  });
});
