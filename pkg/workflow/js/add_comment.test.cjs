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
    process.env.GITHUB_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;

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
    delete process.env.GITHUB_AW_AGENT_OUTPUT;

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("should skip when agent output is empty", async () => {
    setAgentOutput("");

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("should skip when not in issue or PR context", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test comment content",
        },
      ],
    });
    global.context.eventName = "push"; // Not an issue or PR event

    // Execute the script
    await eval(`(async () => { ${createCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith(
      'Target is "triggering" but not running in issue, pull request, or discussion context, skipping comment creation'
    );
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

  it("should include workflow source in footer when GITHUB_AW_WORKFLOW_SOURCE is provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test content with source",
        },
      ],
    });
    process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GITHUB_AW_WORKFLOW_SOURCE = "githubnext/agentics/workflows/ci-doctor.md@v1.0.0";
    process.env.GITHUB_AW_WORKFLOW_SOURCE_URL = "https://github.com/githubnext/agentics/tree/v1.0.0/workflows/ci-doctor.md";
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

  it("should not include workflow source footer when GITHUB_AW_WORKFLOW_SOURCE is not provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_comment",
          body: "Test content without source",
        },
      ],
    });
    process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";
    delete process.env.GITHUB_AW_WORKFLOW_SOURCE; // Ensure it's not set
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
    process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";

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
    process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";

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
});
