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

const mockGithub = {
  rest: {
    pulls: {
      createReviewComment: vi.fn(),
    },
  },
};

const mockContext = {
  eventName: "pull_request",
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    pull_request: {
      number: 123,
      head: {
        sha: "abc123def456",
      },
    },
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("create_pr_review_comment.cjs", () => {
  let createPRReviewCommentScript;

  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GITHUB_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Read the script file
    const scriptPath = path.join(__dirname, "create_pr_review_comment.cjs");
    createPRReviewCommentScript = fs.readFileSync(scriptPath, "utf8");

    // Reset environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_AW_PR_REVIEW_COMMENT_SIDE;
    delete process.env.GITHUB_AW_PR_REVIEW_COMMENT_TARGET;

    // Reset global context to default PR context
    global.context = mockContext;
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && require("fs").existsSync(tempFilePath)) {
      require("fs").unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  it("should create a single PR review comment with basic configuration", async () => {
    // Mock the API response
    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({
      data: {
        id: 456,
        html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r456",
      },
    });

    // Set up environment
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "Consider using const instead of let here.",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify the API was called correctly
    expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      body: expect.stringContaining("Consider using const instead of let here."),
      path: "src/main.js",
      commit_id: "abc123def456",
      line: 10,
      side: "RIGHT",
    });

    // Verify outputs were set
    expect(mockCore.setOutput).toHaveBeenCalledWith("review_comment_id", 456);
    expect(mockCore.setOutput).toHaveBeenCalledWith("review_comment_url", "https://github.com/testowner/testrepo/pull/123#discussion_r456");
  });

  it("should create a multi-line PR review comment", async () => {
    // Mock the API response
    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({
      data: {
        id: 789,
        html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r789",
      },
    });

    // Set up environment with multi-line comment
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/utils.js",
          line: 25,
          start_line: 20,
          side: "LEFT",
          body: "This entire function could be simplified using modern JS features.",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify the API was called with multi-line parameters
    expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 123,
      body: expect.stringContaining("This entire function could be simplified using modern JS features."),
      path: "src/utils.js",
      commit_id: "abc123def456",
      line: 25,
      start_line: 20,
      side: "LEFT",
      start_side: "LEFT",
    });
  });

  it("should handle multiple review comments", async () => {
    // Mock multiple API responses
    mockGithub.rest.pulls.createReviewComment
      .mockResolvedValueOnce({
        data: {
          id: 111,
          html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r111",
        },
      })
      .mockResolvedValueOnce({
        data: {
          id: 222,
          html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r222",
        },
      });

    // Set up environment with multiple comments
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "First comment",
        },
        {
          type: "create_pull_request_review_comment",
          path: "src/utils.js",
          line: 25,
          body: "Second comment",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify both API calls were made
    expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledTimes(2);

    // Verify outputs were set for the last comment
    expect(mockCore.setOutput).toHaveBeenCalledWith("review_comment_id", 222);
    expect(mockCore.setOutput).toHaveBeenCalledWith("review_comment_url", "https://github.com/testowner/testrepo/pull/123#discussion_r222");
  });

  it("should use configured side from environment variable", async () => {
    // Mock the API response
    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({
      data: {
        id: 333,
        html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r333",
      },
    });

    // Set up environment with custom side
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "Comment on left side",
        },
      ],
    });
    process.env.GITHUB_AW_PR_REVIEW_COMMENT_SIDE = "LEFT";

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify the configured side was used
    expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith(
      expect.objectContaining({
        side: "LEFT",
      })
    );
  });

  it("should skip when not in pull request context", async () => {
    // Change context to non-PR event
    global.context = {
      ...mockContext,
      eventName: "issues",
      payload: {
        issue: { number: 123 },
        repository: mockContext.payload.repository,
      },
    };

    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "This should not be created",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify no API calls were made
    expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });

  it("should validate required fields and skip invalid items", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          // Missing path
          line: 10,
          body: "Missing path",
        },
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          // Missing line
          body: "Missing line",
        },
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          // Missing body
        },
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: "invalid",
          body: "Invalid line number",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify no API calls were made due to validation failures
    expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });

  it("should validate start_line is not greater than line", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          start_line: 15, // Invalid: start_line > line
          body: "Invalid range",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify no API calls were made due to validation failure
    expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
  });

  it("should validate side values", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          side: "INVALID_SIDE",
          body: "Invalid side value",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify no API calls were made due to validation failure
    expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
  });

  it("should include AI disclaimer in comment body", async () => {
    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({
      data: {
        id: 999,
        html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r999",
      },
    });

    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "Original comment",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify the body includes the AI disclaimer
    expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith(
      expect.objectContaining({
        body: expect.stringMatching(/Original comment[\s\S]*AI generated by/),
      })
    );
  });

  it("should respect target configuration for specific PR number", async () => {
    // Set target to specific PR number
    process.env.GITHUB_AW_PR_REVIEW_COMMENT_TARGET = "456";

    // Mock the API response for fetching PR details
    mockGithub.rest.pulls.get = vi.fn().mockResolvedValue({
      data: {
        number: 456,
        head: {
          sha: "def456abc789",
        },
      },
    });
    mockGithub.rest.pulls.createReviewComment = vi.fn().mockResolvedValue({
      data: {
        id: 999,
        html_url: "https://github.com/testowner/testrepo/pull/456#discussion_r999",
      },
    });

    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "Review comment on specific PR",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify PR details were fetched for the specified PR number
    expect(mockGithub.rest.pulls.get).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 456,
    });

    // Verify comment was created on the specified PR
    expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith(
      expect.objectContaining({
        pull_number: 456,
        path: "src/main.js",
        line: 10,
        commit_id: "def456abc789",
      })
    );
  });

  it('should respect target "*" configuration with pull_request_number in item', async () => {
    // Set target to "*"
    process.env.GITHUB_AW_PR_REVIEW_COMMENT_TARGET = "*";

    // Mock the API response for fetching PR details
    mockGithub.rest.pulls.get = vi.fn().mockResolvedValue({
      data: {
        number: 789,
        head: {
          sha: "xyz789abc456",
        },
      },
    });
    mockGithub.rest.pulls.createReviewComment = vi.fn().mockResolvedValue({
      data: {
        id: 888,
        html_url: "https://github.com/testowner/testrepo/pull/789#discussion_r888",
      },
    });

    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          pull_request_number: 789,
          path: "src/utils.js",
          line: 20,
          body: "Review comment on any PR",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify PR details were fetched for the specified PR number
    expect(mockGithub.rest.pulls.get).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      pull_number: 789,
    });

    // Verify comment was created on the specified PR
    expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalledWith(
      expect.objectContaining({
        pull_number: 789,
        path: "src/utils.js",
        line: 20,
        commit_id: "xyz789abc456",
      })
    );
  });

  it('should skip item when target is "*" but no pull_request_number specified', async () => {
    // Set target to "*"
    process.env.GITHUB_AW_PR_REVIEW_COMMENT_TARGET = "*";

    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "Review comment without PR number",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify no API calls were made
    expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
  });

  it("should skip comment creation when target is triggering but not in PR context", async () => {
    // Set target to "triggering" (default)
    process.env.GITHUB_AW_PR_REVIEW_COMMENT_TARGET = "triggering";

    // Change context to non-PR event
    global.context = {
      eventName: "issues",
      runId: 12345,
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      payload: {
        issue: {
          number: 10,
        },
        repository: {
          html_url: "https://github.com/testowner/testrepo",
        },
      },
    };

    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "This should not be created",
        },
      ],
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    // Verify no API calls were made
    expect(mockGithub.rest.pulls.createReviewComment).not.toHaveBeenCalled();
  });

  it("should include workflow source in footer when GITHUB_AW_WORKFLOW_SOURCE is provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "Test review comment with source",
        },
      ],
    });
    process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GITHUB_AW_WORKFLOW_SOURCE = "githubnext/agentics/workflows/ci-doctor.md@v1.0.0";
    process.env.GITHUB_AW_WORKFLOW_SOURCE_URL = "https://github.com/githubnext/agentics/tree/v1.0.0/workflows/ci-doctor.md";

    // Reset context to default PR context
    global.context = {
      eventName: "pull_request",
      runId: 12345,
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      payload: {
        pull_request: {
          number: 10,
          head: {
            sha: "abc123",
          },
        },
        repository: {
          html_url: "https://github.com/testowner/testrepo",
        },
      },
    };

    const mockComment = {
      id: 456,
      html_url: "https://github.com/testowner/testrepo/pull/10#discussion_r456",
    };

    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalled();
    const callArgs = mockGithub.rest.pulls.createReviewComment.mock.calls[0][0];

    // Check that the body contains the expected elements
    expect(callArgs.body).toContain("Test review comment with source");
    expect(callArgs.body).toContain("AI generated by [Test Workflow]");
    expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345");
    expect(callArgs.body).toContain("gh aw add githubnext/agentics/workflows/ci-doctor.md@v1.0.0");
    expect(callArgs.body).toContain("usage guide");
  });

  it("should not include workflow source footer when GITHUB_AW_WORKFLOW_SOURCE is not provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          path: "src/main.js",
          line: 10,
          body: "Test review comment without source",
        },
      ],
    });
    process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";
    delete process.env.GITHUB_AW_WORKFLOW_SOURCE; // Ensure it's not set

    // Reset context to default PR context
    global.context = {
      eventName: "pull_request",
      runId: 12345,
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      payload: {
        pull_request: {
          number: 10,
          head: {
            sha: "abc123",
          },
        },
        repository: {
          html_url: "https://github.com/testowner/testrepo",
        },
      },
    };

    const mockComment = {
      id: 457,
      html_url: "https://github.com/testowner/testrepo/pull/10#discussion_r457",
    };

    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    expect(mockGithub.rest.pulls.createReviewComment).toHaveBeenCalled();
    const callArgs = mockGithub.rest.pulls.createReviewComment.mock.calls[0][0];

    // Check that the body does NOT contain the workflow source
    expect(callArgs.body).toContain("Test review comment without source");
    expect(callArgs.body).toContain("AI generated by [Test Workflow]");
    expect(callArgs.body).not.toContain("gh aw add");
    expect(callArgs.body).not.toContain("usage guide");
  });

  it("should include triggering PR number in footer when in PR context", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_pull_request_review_comment",
          body: "Review comment from PR context",
          path: "test.js",
          line: 10,
        },
      ],
    });
    process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";

    // Simulate PR context
    global.context.eventName = "pull_request";
    global.context.payload.pull_request = {
      number: 123,
      head: { sha: "abc123" },
    };

    const mockComment = {
      id: 999,
      html_url: "https://github.com/testowner/testrepo/pull/123#discussion_r999",
    };

    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({ data: mockComment });

    // Execute the script
    await eval(`(async () => { ${createPRReviewCommentScript} })()`);

    const callArgs = mockGithub.rest.pulls.createReviewComment.mock.calls[0][0];

    // Check that the footer includes reference to triggering PR
    expect(callArgs.body).toContain("Review comment from PR context");
    expect(callArgs.body).toContain("AI generated by [Test Workflow]");
    expect(callArgs.body).toContain("for #123");
  });
});
