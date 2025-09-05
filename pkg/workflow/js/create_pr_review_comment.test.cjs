import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn(),
  },
  warning: vi.fn(),
  error: vi.fn(),
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

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Read the script file
    const scriptPath = path.join(__dirname, "create_pr_review_comment.cjs");
    createPRReviewCommentScript = fs.readFileSync(scriptPath, "utf8");

    // Reset environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_AW_PR_REVIEW_COMMENT_SIDE;

    // Reset global context to default PR context
    global.context = mockContext;
  });

  it("should create a single PR review comment with basic configuration", async () => {
    // Mock the API response
    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({
      data: {
        id: 456,
        html_url:
          "https://github.com/testowner/testrepo/pull/123#discussion_r456",
      },
    });

    // Set up environment
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request-review-comment",
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
      body: expect.stringContaining(
        "Consider using const instead of let here."
      ),
      path: "src/main.js",
      commit_id: "abc123def456",
      line: 10,
      side: "RIGHT",
    });

    // Verify outputs were set
    expect(mockCore.setOutput).toHaveBeenCalledWith("review_comment_id", 456);
    expect(mockCore.setOutput).toHaveBeenCalledWith(
      "review_comment_url",
      "https://github.com/testowner/testrepo/pull/123#discussion_r456"
    );
  });

  it("should create a multi-line PR review comment", async () => {
    // Mock the API response
    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({
      data: {
        id: 789,
        html_url:
          "https://github.com/testowner/testrepo/pull/123#discussion_r789",
      },
    });

    // Set up environment with multi-line comment
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request-review-comment",
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
      body: expect.stringContaining(
        "This entire function could be simplified using modern JS features."
      ),
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
          html_url:
            "https://github.com/testowner/testrepo/pull/123#discussion_r111",
        },
      })
      .mockResolvedValueOnce({
        data: {
          id: 222,
          html_url:
            "https://github.com/testowner/testrepo/pull/123#discussion_r222",
        },
      });

    // Set up environment with multiple comments
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request-review-comment",
          path: "src/main.js",
          line: 10,
          body: "First comment",
        },
        {
          type: "create-pull-request-review-comment",
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
    expect(mockCore.setOutput).toHaveBeenCalledWith(
      "review_comment_url",
      "https://github.com/testowner/testrepo/pull/123#discussion_r222"
    );
  });

  it("should use configured side from environment variable", async () => {
    // Mock the API response
    mockGithub.rest.pulls.createReviewComment.mockResolvedValue({
      data: {
        id: 333,
        html_url:
          "https://github.com/testowner/testrepo/pull/123#discussion_r333",
      },
    });

    // Set up environment with custom side
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request-review-comment",
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

    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request-review-comment",
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
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request-review-comment",
          // Missing path
          line: 10,
          body: "Missing path",
        },
        {
          type: "create-pull-request-review-comment",
          path: "src/main.js",
          // Missing line
          body: "Missing line",
        },
        {
          type: "create-pull-request-review-comment",
          path: "src/main.js",
          line: 10,
          // Missing body
        },
        {
          type: "create-pull-request-review-comment",
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
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request-review-comment",
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
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request-review-comment",
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
        html_url:
          "https://github.com/testowner/testrepo/pull/123#discussion_r999",
      },
    });

    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify({
      items: [
        {
          type: "create-pull-request-review-comment",
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
        body: expect.stringMatching(
          /Original comment[\s\S]*Generated by Agentic Workflow Run/
        ),
      })
    );
  });
});
