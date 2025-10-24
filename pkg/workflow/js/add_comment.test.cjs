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
    issues: {
      createComment: vi.fn(),
    },
  },
};

const mockContext = {
  eventName: "issues",
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    issue: {
      number: 123,
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

describe("add_comment.cjs", () => {
  let createCommentScript;

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

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "add_comment.cjs");
    createCommentScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && require("fs").existsSync(tempFilePath)) {
      require("fs").unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  it("should skip when no agent output is provided", async () => {
    // Remove the output content environment variable
    delete process.env.GH_AW_AGENT_OUTPUT;

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("should skip when agent output is empty", async () => {
    setAgentOutput("");

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("should skip when not in issue or PR context and no branch can be resolved", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment content",
        },
      ],
    });
    global.context.eventName = "push"; // Not an issue or PR event

    // Ensure no GITHUB_REF or GITHUB_HEAD_REF is set
    delete process.env.GITHUB_REF;
    delete process.env.GITHUB_HEAD_REF;

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Could not resolve current branch. Exiting gracefully without commenting.");
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("should create comment on issue successfully", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment content",
        },
      ],
    });
    global.context.eventName = "issues";

    const mockComment = {
      id: 456,
      html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 123,
      body: expect.stringContaining("Test comment content"),
    });

    expect(mockCore.setOutput).toHaveBeenCalledWith("comment_id", 456);
    expect(mockCore.setOutput).toHaveBeenCalledWith("comment_url", mockComment.html_url);
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should create comment on pull request successfully", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test PR comment content",
        },
      ],
    });
    global.context.eventName = "pull_request";
    global.context.payload.pull_request = { number: 789 };
    delete global.context.payload.issue; // Remove issue from payload

    const mockComment = {
      id: 789,
      html_url: "https://github.com/testowner/testrepo/issues/789#issuecomment-789",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 789,
      body: expect.stringContaining("Test PR comment content"),
    });
  });

  it("should include run information in comment body", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test content",
        },
      ],
    });
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 }; // Make sure issue context is properly set

    const mockComment = {
      id: 456,
      html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalled();
    expect(mockGithub.rest.issues.createComment.mock.calls).toHaveLength(1);

    const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];
    expect(callArgs.body).toContain("Test content");
    expect(callArgs.body).toContain("AI generated by");
    expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345");
  });

  it("should include workflow source in footer when GH_AW_WORKFLOW_SOURCE is provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test content with source",
        },
      ],
    });
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_WORKFLOW_SOURCE = "githubnext/agentics/workflows/ci-doctor.md@v1.0.0";
    process.env.GH_AW_WORKFLOW_SOURCE_URL = "https://github.com/githubnext/agentics/tree/v1.0.0/workflows/ci-doctor.md";
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };

    const mockComment = {
      id: 456,
      html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalled();
    const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];

    // Check that the footer contains the expected elements
    expect(callArgs.body).toContain("Test content with source");
    expect(callArgs.body).toContain("AI generated by [Test Workflow]");
    expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345");
    expect(callArgs.body).toContain("gh aw add githubnext/agentics/workflows/ci-doctor.md@v1.0.0");
    expect(callArgs.body).toContain("usage guide");
  });

  it("should not include workflow source footer when GH_AW_WORKFLOW_SOURCE is not provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test content without source",
        },
      ],
    });
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    delete process.env.GH_AW_WORKFLOW_SOURCE; // Ensure it's not set
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };

    const mockComment = {
      id: 456,
      html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalled();
    const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];

    // Check that the footer does NOT contain the workflow source
    expect(callArgs.body).toContain("Test content without source");
    expect(callArgs.body).toContain("AI generated by [Test Workflow]");
    expect(callArgs.body).not.toContain("gh aw add");
    expect(callArgs.body).not.toContain("usage guide");
  });

  it("should use GITHUB_SERVER_URL when repository context is not available", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test content with custom server",
        },
      ],
    });
    process.env.GITHUB_SERVER_URL = "https://github.enterprise.com";
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };
    // Remove repository context to force use of GITHUB_SERVER_URL
    delete global.context.payload.repository;

    const mockComment = {
      id: 456,
      html_url: "https://github.enterprise.com/testowner/testrepo/issues/123#issuecomment-456",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalled();
    const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];

    // Check that the footer uses the custom GitHub server URL
    expect(callArgs.body).toContain("Test content with custom server");
    expect(callArgs.body).toContain("https://github.enterprise.com/testowner/testrepo/actions/runs/12345");
    expect(callArgs.body).not.toContain("https://github.com/testowner/testrepo/actions/runs/12345");

    // Clean up
    delete process.env.GITHUB_SERVER_URL;
  });

  it("should fallback to https://github.com when GITHUB_SERVER_URL is not set and repository context is missing", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test content with fallback",
        },
      ],
    });
    delete process.env.GITHUB_SERVER_URL;
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };
    // Remove repository context to test fallback
    delete global.context.payload.repository;

    const mockComment = {
      id: 456,
      html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalled();
    const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];

    // Check that the footer uses the default https://github.com
    expect(callArgs.body).toContain("Test content with fallback");
    expect(callArgs.body).toContain("https://github.com/testowner/testrepo/actions/runs/12345");
  });

  it("should include triggering issue number in footer when in issue context", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Comment from issue context",
        },
      ],
    });
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";

    // Simulate issue context
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 42 };

    const mockComment = {
      id: 789,
      html_url: "https://github.com/testowner/testrepo/issues/42#issuecomment-789",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];

    // Check that the footer includes reference to triggering issue
    expect(callArgs.body).toContain("Comment from issue context");
    expect(callArgs.body).toContain("AI generated by [Test Workflow]");
    expect(callArgs.body).toContain("for #42");
  });

  it("should include triggering PR number in footer when in PR context", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Comment from PR context",
        },
      ],
    });
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";

    // Simulate PR context
    global.context.eventName = "pull_request";
    delete global.context.payload.issue;
    global.context.payload.pull_request = { number: 123 };

    const mockComment = {
      id: 890,
      html_url: "https://github.com/testowner/testrepo/pull/123#issuecomment-890",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];

    // Check that the footer includes reference to triggering PR
    expect(callArgs.body).toContain("Comment from PR context");
    expect(callArgs.body).toContain("AI generated by [Test Workflow]");
    expect(callArgs.body).toContain("for #123");

    // Clean up
    delete global.context.payload.pull_request;
  });

  it("should use header level 4 for related items in comments", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment with related items",
        },
      ],
    });
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };

    // Set environment variables for created items
    process.env.GH_AW_CREATED_ISSUE_URL = "https://github.com/testowner/testrepo/issues/456";
    process.env.GH_AW_CREATED_ISSUE_NUMBER = "456";
    process.env.GH_AW_CREATED_DISCUSSION_URL = "https://github.com/testowner/testrepo/discussions/789";
    process.env.GH_AW_CREATED_DISCUSSION_NUMBER = "789";
    process.env.GH_AW_CREATED_PULL_REQUEST_URL = "https://github.com/testowner/testrepo/pull/101";
    process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER = "101";

    const mockComment = {
      id: 890,
      html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-890",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({ data: mockComment });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    const callArgs = mockGithub.rest.issues.createComment.mock.calls[0][0];

    // Check that the related items section uses header level 4 (####)
    expect(callArgs.body).toContain("#### Related Items");
    // Check that it uses exactly 4 hashes, not 2
    expect(callArgs.body).toMatch(/####\s+Related Items/);
    expect(callArgs.body).not.toMatch(/^##\s+Related Items/m);
    expect(callArgs.body).not.toMatch(/\*\*Related Items:\*\*/);

    // Check that the references are included
    expect(callArgs.body).toContain("- Issue: [#456](https://github.com/testowner/testrepo/issues/456)");
    expect(callArgs.body).toContain("- Discussion: [#789](https://github.com/testowner/testrepo/discussions/789)");
    expect(callArgs.body).toContain("- Pull Request: [#101](https://github.com/testowner/testrepo/pull/101)");

    // Clean up
    delete process.env.GH_AW_CREATED_ISSUE_URL;
    delete process.env.GH_AW_CREATED_ISSUE_NUMBER;
    delete process.env.GH_AW_CREATED_DISCUSSION_URL;
    delete process.env.GH_AW_CREATED_DISCUSSION_NUMBER;
    delete process.env.GH_AW_CREATED_PULL_REQUEST_URL;
    delete process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER;
  });

  it("should use header level 4 for related items in staged mode preview", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment in staged mode",
        },
      ],
    });
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };

    // Enable staged mode
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    // Set environment variables for created items
    process.env.GH_AW_CREATED_ISSUE_URL = "https://github.com/testowner/testrepo/issues/456";
    process.env.GH_AW_CREATED_ISSUE_NUMBER = "456";
    process.env.GH_AW_CREATED_DISCUSSION_URL = "https://github.com/testowner/testrepo/discussions/789";
    process.env.GH_AW_CREATED_DISCUSSION_NUMBER = "789";
    process.env.GH_AW_CREATED_PULL_REQUEST_URL = "https://github.com/testowner/testrepo/pull/101";
    process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER = "101";

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Check that summary was written with correct header level 4
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    const summaryContent = mockCore.summary.addRaw.mock.calls[0][0];

    // Check that the related items section uses header level 4 (####)
    expect(summaryContent).toContain("#### Related Items");
    // Check that it uses exactly 4 hashes, not 2
    expect(summaryContent).toMatch(/####\s+Related Items/);
    expect(summaryContent).not.toMatch(/^##\s+Related Items/m);
    expect(summaryContent).not.toMatch(/\*\*Related Items:\*\*/);

    // Check that the references are included
    expect(summaryContent).toContain("- Issue: [#456](https://github.com/testowner/testrepo/issues/456)");
    expect(summaryContent).toContain("- Discussion: [#789](https://github.com/testowner/testrepo/discussions/789)");
    expect(summaryContent).toContain("- Pull Request: [#101](https://github.com/testowner/testrepo/pull/101)");

    // Clean up
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_CREATED_ISSUE_URL;
    delete process.env.GH_AW_CREATED_ISSUE_NUMBER;
    delete process.env.GH_AW_CREATED_DISCUSSION_URL;
    delete process.env.GH_AW_CREATED_DISCUSSION_NUMBER;
    delete process.env.GH_AW_CREATED_PULL_REQUEST_URL;
    delete process.env.GH_AW_CREATED_PULL_REQUEST_NUMBER;
  });

  it("should create comment on discussion using GraphQL when in discussion_comment context", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test discussion comment",
        },
      ],
    });

    // Simulate discussion_comment context
    global.context.eventName = "discussion_comment";
    global.context.payload.discussion = { number: 1993 };
    global.context.payload.comment = {
      id: 12345,
      node_id: "DC_kwDOABcD1M4AaBbC", // Node ID of the comment to reply to
    };
    delete global.context.payload.issue;
    delete global.context.payload.pull_request;

    // Mock GraphQL responses for discussion
    const mockGraphqlResponse = vi.fn();
    mockGraphqlResponse
      .mockResolvedValueOnce({
        // First call: get discussion ID
        repository: {
          discussion: {
            id: "D_kwDOPc1QR84BpqRs",
            url: "https://github.com/testowner/testrepo/discussions/1993",
          },
        },
      })
      .mockResolvedValueOnce({
        // Second call: create comment with replyToId
        addDiscussionComment: {
          comment: {
            id: "DC_kwDOPc1QR84BpqRt",
            body: "Test discussion comment",
            createdAt: "2025-10-19T22:00:00Z",
            url: "https://github.com/testowner/testrepo/discussions/1993#discussioncomment-123",
          },
        },
      });

    global.github.graphql = mockGraphqlResponse;

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Verify GraphQL was called with correct queries
    expect(mockGraphqlResponse).toHaveBeenCalledTimes(2);

    // First call should fetch discussion ID
    expect(mockGraphqlResponse.mock.calls[0][0]).toContain("query");
    expect(mockGraphqlResponse.mock.calls[0][0]).toContain("discussion(number: $num)");
    expect(mockGraphqlResponse.mock.calls[0][1]).toEqual({
      owner: "testowner",
      repo: "testrepo",
      num: 1993,
    });

    // Second call should create the comment with replyToId
    expect(mockGraphqlResponse.mock.calls[1][0]).toContain("mutation");
    expect(mockGraphqlResponse.mock.calls[1][0]).toContain("addDiscussionComment");
    expect(mockGraphqlResponse.mock.calls[1][0]).toContain("replyToId");
    expect(mockGraphqlResponse.mock.calls[1][1].body).toContain("Test discussion comment");
    expect(mockGraphqlResponse.mock.calls[1][1].replyToId).toBe("DC_kwDOABcD1M4AaBbC");

    // Verify REST API was NOT called
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();

    // Verify outputs were set
    expect(mockCore.setOutput).toHaveBeenCalledWith("comment_id", "DC_kwDOPc1QR84BpqRt");
    expect(mockCore.setOutput).toHaveBeenCalledWith(
      "comment_url",
      "https://github.com/testowner/testrepo/discussions/1993#discussioncomment-123"
    );

    // Clean up
    delete global.github.graphql;
    delete global.context.payload.discussion;
    delete global.context.payload.comment;
  });

  it("should create comment on discussion using GraphQL when GITHUB_AW_COMMENT_DISCUSSION is true (explicit discussion mode)", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test explicit discussion comment",
          item_number: 2001,
        },
      ],
    });

    // Set target configuration to use explicit number
    process.env.GH_AW_COMMENT_TARGET = "*";
    // Force discussion mode via environment variable
    process.env.GITHUB_AW_COMMENT_DISCUSSION = "true";

    // Use a non-discussion context (e.g., issues) to test explicit override
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };
    delete global.context.payload.discussion;
    delete global.context.payload.pull_request;

    // Mock GraphQL responses for discussion
    const mockGraphqlResponse = vi.fn();
    mockGraphqlResponse
      .mockResolvedValueOnce({
        // First call: get discussion ID
        repository: {
          discussion: {
            id: "D_kwDOPc1QR84BpqRu",
            url: "https://github.com/testowner/testrepo/discussions/2001",
          },
        },
      })
      .mockResolvedValueOnce({
        // Second call: create comment (no replyToId for non-comment context)
        addDiscussionComment: {
          comment: {
            id: "DC_kwDOPc1QR84BpqRv",
            body: "Test explicit discussion comment",
            createdAt: "2025-10-22T12:00:00Z",
            url: "https://github.com/testowner/testrepo/discussions/2001#discussioncomment-456",
          },
        },
      });

    global.github.graphql = mockGraphqlResponse;

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Verify GraphQL was called with correct queries
    expect(mockGraphqlResponse).toHaveBeenCalledTimes(2);

    // First call should fetch discussion ID for the explicit number
    expect(mockGraphqlResponse.mock.calls[0][0]).toContain("query");
    expect(mockGraphqlResponse.mock.calls[0][0]).toContain("discussion(number: $num)");
    expect(mockGraphqlResponse.mock.calls[0][1]).toEqual({
      owner: "testowner",
      repo: "testrepo",
      num: 2001, // Should use the item_number from the comment item
    });

    // Second call should create the comment (without replyToId since this is not discussion_comment context)
    expect(mockGraphqlResponse.mock.calls[1][0]).toContain("mutation");
    expect(mockGraphqlResponse.mock.calls[1][0]).toContain("addDiscussionComment");
    expect(mockGraphqlResponse.mock.calls[1][1].body).toContain("Test explicit discussion comment");
    // Should NOT have replyToId since we're not in discussion_comment context
    expect(mockGraphqlResponse.mock.calls[1][1].replyToId).toBeUndefined();

    // Verify REST API was NOT called (should use GraphQL for discussions)
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();

    // Verify outputs were set
    expect(mockCore.setOutput).toHaveBeenCalledWith("comment_id", "DC_kwDOPc1QR84BpqRv");
    expect(mockCore.setOutput).toHaveBeenCalledWith(
      "comment_url",
      "https://github.com/testowner/testrepo/discussions/2001#discussioncomment-456"
    );

    // Verify info logging shows it's targeting a discussion
    expect(mockCore.info).toHaveBeenCalledWith("Creating comment on discussion #2001");

    // Clean up
    delete process.env.GH_AW_COMMENT_TARGET;
    delete process.env.GITHUB_AW_COMMENT_DISCUSSION;
    delete global.github.graphql;
  });

  it("should find and comment on PR when triggered by workflow_dispatch with matching open PR", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment from workflow_dispatch",
        },
      ],
    });

    // Simulate workflow_dispatch event (non-commentable)
    global.context.eventName = "workflow_dispatch";
    delete global.context.payload.issue;
    delete global.context.payload.pull_request;
    delete global.context.payload.discussion;

    // Set environment variable for branch
    process.env.GITHUB_REF = "refs/heads/feature-branch";

    // Mock GitHub API response for searching PRs
    mockGithub.rest.pulls = {
      list: vi.fn().mockResolvedValue({
        data: [
          {
            number: 456,
            html_url: "https://github.com/testowner/testrepo/pull/456",
            head: { ref: "feature-branch" },
          },
        ],
      }),
    };

    const mockComment = {
      id: 789,
      html_url: "https://github.com/testowner/testrepo/pull/456#issuecomment-789",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Verify the branch was resolved
    expect(mockCore.info).toHaveBeenCalledWith("Resolved branch from GITHUB_REF: feature-branch");

    // Verify PR search was called
    expect(mockGithub.rest.pulls.list).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      state: "open",
      head: "testowner:feature-branch",
      per_page: 1,
    });

    // Verify comment was created on the PR
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 456,
      body: expect.stringContaining("Test comment from workflow_dispatch"),
    });

    // Clean up
    delete process.env.GITHUB_REF;
    delete mockGithub.rest.pulls;
  });

  it("should resolve branch from GITHUB_HEAD_REF when available", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment with GITHUB_HEAD_REF",
        },
      ],
    });

    // Simulate workflow_dispatch event
    global.context.eventName = "workflow_dispatch";
    delete global.context.payload.issue;
    delete global.context.payload.pull_request;

    // Set GITHUB_HEAD_REF (takes precedence over GITHUB_REF)
    process.env.GITHUB_HEAD_REF = "another-feature-branch";
    process.env.GITHUB_REF = "refs/heads/main"; // Should be ignored

    // Mock GitHub API response
    mockGithub.rest.pulls = {
      list: vi.fn().mockResolvedValue({
        data: [
          {
            number: 999,
            html_url: "https://github.com/testowner/testrepo/pull/999",
          },
        ],
      }),
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: {
        id: 111,
        html_url: "https://github.com/testowner/testrepo/pull/999#issuecomment-111",
      },
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Verify GITHUB_HEAD_REF was used
    expect(mockCore.info).toHaveBeenCalledWith("Resolved branch from GITHUB_HEAD_REF: another-feature-branch");

    // Verify correct branch was used in PR search
    expect(mockGithub.rest.pulls.list).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      state: "open",
      head: "testowner:another-feature-branch",
      per_page: 1,
    });

    // Clean up
    delete process.env.GITHUB_HEAD_REF;
    delete process.env.GITHUB_REF;
    delete mockGithub.rest.pulls;
  });

  it("should exit gracefully when no open PR found for branch in non-commentable context", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment with no PR",
        },
      ],
    });

    // Simulate workflow_dispatch event
    global.context.eventName = "workflow_dispatch";
    delete global.context.payload.issue;
    delete global.context.payload.pull_request;

    // Set environment variable for branch
    process.env.GITHUB_REF = "refs/heads/no-pr-branch";

    // Mock GitHub API response with no PRs
    mockGithub.rest.pulls = {
      list: vi.fn().mockResolvedValue({
        data: [], // No open PRs
      }),
    };

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Verify the graceful exit message
    expect(mockCore.info).toHaveBeenCalledWith(
      "No open pull request found for branch no-pr-branch. Exiting gracefully without commenting."
    );

    // Verify no comment was created
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();

    // Clean up
    delete process.env.GITHUB_REF;
    delete mockGithub.rest.pulls;
  });

  it("should exit gracefully when branch cannot be resolved in non-commentable context", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment without resolvable branch",
        },
      ],
    });

    // Simulate workflow_dispatch event
    global.context.eventName = "workflow_dispatch";
    delete global.context.payload.issue;
    delete global.context.payload.pull_request;

    // Don't set GITHUB_REF or GITHUB_HEAD_REF
    delete process.env.GITHUB_REF;
    delete process.env.GITHUB_HEAD_REF;

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Verify the graceful exit message
    expect(mockCore.info).toHaveBeenCalledWith("Could not resolve current branch. Exiting gracefully without commenting.");

    // Verify no comment was created
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("should handle target-repo when searching for PRs", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment with target-repo",
        },
      ],
    });

    // Simulate workflow_dispatch event
    global.context.eventName = "workflow_dispatch";
    delete global.context.payload.issue;
    delete global.context.payload.pull_request;

    // Set environment variables
    process.env.GITHUB_REF = "refs/heads/fork-branch";
    process.env.GH_AW_TARGET_REPO_SLUG = "forkowner/forkrepo";

    // Mock GitHub API response
    mockGithub.rest.pulls = {
      list: vi.fn().mockResolvedValue({
        data: [
          {
            number: 555,
            html_url: "https://github.com/testowner/testrepo/pull/555",
          },
        ],
      }),
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: {
        id: 666,
        html_url: "https://github.com/testowner/testrepo/pull/555#issuecomment-666",
      },
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Verify PR search was called with forkowner prefix
    expect(mockGithub.rest.pulls.list).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      state: "open",
      head: "forkowner:fork-branch",
      per_page: 1,
    });

    // Clean up
    delete process.env.GITHUB_REF;
    delete process.env.GH_AW_TARGET_REPO_SLUG;
    delete mockGithub.rest.pulls;
  });

  it("should preserve existing behavior for issue context", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment in issue context",
        },
      ],
    });

    // Ensure we're in issue context (default)
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 123 };

    const mockComment = {
      id: 456,
      html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-456",
    };

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: mockComment,
    });

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Verify comment was created on the issue
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 123,
      body: expect.stringContaining("Test comment in issue context"),
    });

    // Verify PR search was NOT called (existing behavior preserved)
    expect(mockGithub.rest.pulls?.list).toBeUndefined();
  });

  it("should handle API errors when searching for PRs gracefully", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment with API error",
        },
      ],
    });

    // Simulate workflow_dispatch event
    global.context.eventName = "workflow_dispatch";
    delete global.context.payload.issue;
    delete global.context.payload.pull_request;

    // Set environment variable for branch
    process.env.GITHUB_REF = "refs/heads/test-branch";

    // Mock GitHub API to throw an error
    mockGithub.rest.pulls = {
      list: vi.fn().mockRejectedValue(new Error("API rate limit exceeded")),
    };

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    // Verify warning was logged
    expect(mockCore.warning).toHaveBeenCalledWith("Error searching for pull requests: API rate limit exceeded");

    // Verify graceful exit
    expect(mockCore.info).toHaveBeenCalledWith("No open pull request found for branch test-branch. Exiting gracefully without commenting.");

    // Verify no comment was created
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();

    // Clean up
    delete process.env.GITHUB_REF;
    delete mockGithub.rest.pulls;
  });
});
