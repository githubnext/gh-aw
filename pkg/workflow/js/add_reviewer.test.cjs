import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

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

describe("add_reviewer", () => {
  let tempDir;
  let outputFile;

  beforeEach(() => {
    // Create a temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "add-reviewer-test-"));
    outputFile = path.join(tempDir, "agent-output.json");

    // Reset all mocks before each test
    vi.clearAllMocks();
    vi.resetModules(); // Reset module cache to allow fresh imports

    // Clear environment variables
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_REVIEWERS_ALLOWED;
    delete process.env.GH_AW_REVIEWERS_MAX_COUNT;
    delete process.env.GH_AW_REVIEWERS_TARGET;

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

  afterEach(() => {
    // Clean up temporary files
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  it("should handle missing GH_AW_AGENT_OUTPUT", async () => {
    delete process.env.GH_AW_AGENT_OUTPUT;

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
  });

  it("should handle missing add_reviewer item", async () => {
    const agentOutput = {
      items: [{ type: "create_issue", title: "Test", body: "Body" }],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockCore.warning).toHaveBeenCalledWith("No add-reviewer item found in agent output");
  });

  it("should add reviewers to PR in non-staged mode", async () => {
    const agentOutput = {
      items: [
        {
          type: "add_reviewer",
          reviewers: ["octocat", "github"],
        },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;
    process.env.GH_AW_REVIEWERS_MAX_COUNT = "3";
    process.env.GH_AW_REVIEWERS_TARGET = "triggering";

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      reviewers: ["octocat", "github"],
    });
    expect(mockCore.setOutput).toHaveBeenCalledWith("reviewers_added", "octocat\ngithub");
  });

  it("should generate staged preview in staged mode", async () => {
    const agentOutput = {
      items: [
        {
          type: "add_reviewer",
          reviewers: ["octocat", "github"],
          pull_request_number: 123,
        },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
    expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });

  it("should filter by allowed reviewers", async () => {
    const agentOutput = {
      items: [
        {
          type: "add_reviewer",
          reviewers: ["octocat", "github", "unauthorized"],
        },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;
    process.env.GH_AW_REVIEWERS_ALLOWED = "octocat,github";
    process.env.GH_AW_REVIEWERS_MAX_COUNT = "3";

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      reviewers: ["octocat", "github"],
    });
  });

  it("should enforce max count limit", async () => {
    const agentOutput = {
      items: [
        {
          type: "add_reviewer",
          reviewers: ["user1", "user2", "user3", "user4", "user5"],
        },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;
    process.env.GH_AW_REVIEWERS_MAX_COUNT = "2";

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      reviewers: ["user1", "user2"],
    });
  });

  it("should handle non-PR context gracefully", async () => {
    const agentOutput = {
      items: [
        {
          type: "add_reviewer",
          reviewers: ["octocat"],
        },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;
    global.context = {
      ...mockContext,
      eventName: "issues", // Not a PR context
      payload: {
        issue: {
          number: 123,
        },
      },
    };

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining('Target is "triggering" but not running in pull request context'));
    expect(mockGithub.rest.pulls.requestReviewers).not.toHaveBeenCalled();
  });

  it("should handle explicit PR number with * target", async () => {
    const agentOutput = {
      items: [
        {
          type: "add_reviewer",
          reviewers: ["octocat"],
          pull_request_number: 456,
        },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;
    process.env.GH_AW_REVIEWERS_TARGET = "*";

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 456,
      reviewers: ["octocat"],
    });
  });

  it("should handle API errors gracefully", async () => {
    const agentOutput = {
      items: [
        {
          type: "add_reviewer",
          reviewers: ["octocat"],
        },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;
    mockGithub.rest.pulls.requestReviewers.mockRejectedValueOnce(new Error("API Error"));

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to add reviewers"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to add reviewers"));
  });

  it("should deduplicate reviewers", async () => {
    const agentOutput = {
      items: [
        {
          type: "add_reviewer",
          reviewers: ["octocat", "github", "octocat", "github"],
        },
      ],
    };
    fs.writeFileSync(outputFile, JSON.stringify(agentOutput));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;
    process.env.GH_AW_REVIEWERS_MAX_COUNT = "10"; // Set high max to test deduplication

    const { main } = await import("./add_reviewer.cjs"); await main();

    expect(mockGithub.rest.pulls.requestReviewers).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      reviewers: ["octocat", "github"],
    });
  });
});
