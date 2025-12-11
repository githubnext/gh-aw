import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
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
    issues: {
      get: vi.fn(),
      createComment: vi.fn(),
      lock: vi.fn(),
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
      number: 42,
    },
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("lock_issue", () => {
  let lockIssueScript;
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
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_LOCK_ISSUE_REQUIRED_LABELS;
    delete process.env.GH_AW_LOCK_ISSUE_REQUIRED_TITLE_PREFIX;
    delete process.env.GH_AW_LOCK_ISSUE_TARGET;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 42 };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "lock_issue.cjs");
    lockIssueScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up temp files
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  it("should handle empty agent output", async () => {
    setAgentOutput({ items: [], errors: [] });

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No lock-issue items found in agent output");
  });

  it("should handle missing agent output", async () => {
    // Don't set GH_AW_AGENT_OUTPUT

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
  });

  it("should lock issue with comment in triggering context", async () => {
    setAgentOutput({
      items: [
        {
          type: "lock_issue",
          body: "Locking this issue as discussion has become unproductive.",
        },
      ],
    });

    // Mock the issue get response
    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 42,
        title: "[test] Test Issue",
        labels: [{ name: "bug" }],
        locked: false,
        html_url: "https://github.com/testowner/testrepo/issues/42",
      },
    });

    // Mock the comment creation
    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: {
        id: 123,
        html_url: "https://github.com/testowner/testrepo/issues/42#issuecomment-123",
      },
    });

    // Mock the issue lock
    mockGithub.rest.issues.lock.mockResolvedValue({});

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 42,
    });

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalled();
    expect(mockGithub.rest.issues.lock).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 42,
      lock_reason: undefined,
    });

    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", 42);
    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_url", "https://github.com/testowner/testrepo/issues/42");
  });

  it("should lock specific issue when target is * with lock reason", async () => {
    setAgentOutput({
      items: [
        {
          type: "lock_issue",
          issue_number: 100,
          body: "Locking this issue due to spam.",
          lock_reason: "spam",
        },
      ],
    });

    process.env.GH_AW_LOCK_ISSUE_TARGET = "*";

    // Mock the issue get response
    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 100,
        title: "[spam] Spam Issue",
        labels: [],
        locked: false,
        html_url: "https://github.com/testowner/testrepo/issues/100",
      },
    });

    // Mock the comment creation
    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: {
        id: 456,
        html_url: "https://github.com/testowner/testrepo/issues/100#issuecomment-456",
      },
    });

    // Mock the issue lock
    mockGithub.rest.issues.lock.mockResolvedValue({});

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 100,
    });

    expect(mockGithub.rest.issues.lock).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 100,
      lock_reason: "spam",
    });

    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", 100);
  });

  it("should filter by required title prefix", async () => {
    setAgentOutput({
      items: [
        {
          type: "lock_issue",
          issue_number: 50,
          body: "Locking this issue.",
        },
      ],
    });

    process.env.GH_AW_LOCK_ISSUE_TARGET = "*";
    process.env.GH_AW_LOCK_ISSUE_REQUIRED_TITLE_PREFIX = "[spam] ";

    // Mock the issue get response with wrong prefix
    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 50,
        title: "[bug] Bug Fix",
        labels: [],
        locked: false,
        html_url: "https://github.com/testowner/testrepo/issues/50",
      },
    });

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    expect(mockGithub.rest.issues.get).toHaveBeenCalled();
    expect(mockGithub.rest.issues.lock).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("does not have required title prefix"));
  });

  it("should filter by required labels", async () => {
    setAgentOutput({
      items: [
        {
          type: "lock_issue",
          issue_number: 60,
          body: "Locking this issue.",
        },
      ],
    });

    process.env.GH_AW_LOCK_ISSUE_TARGET = "*";
    process.env.GH_AW_LOCK_ISSUE_REQUIRED_LABELS = "spam,off-topic";

    // Mock the issue get response without required labels
    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 60,
        title: "Test Issue",
        labels: [{ name: "bug" }],
        locked: false,
        html_url: "https://github.com/testowner/testrepo/issues/60",
      },
    });

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    expect(mockGithub.rest.issues.get).toHaveBeenCalled();
    expect(mockGithub.rest.issues.lock).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("does not have required labels"));
  });

  it("should skip already locked issues", async () => {
    setAgentOutput({
      items: [
        {
          type: "lock_issue",
          issue_number: 70,
          body: "Locking this issue.",
        },
      ],
    });

    process.env.GH_AW_LOCK_ISSUE_TARGET = "*";

    // Mock the issue get response as already locked
    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 70,
        title: "Already Locked",
        labels: [],
        locked: true,
        html_url: "https://github.com/testowner/testrepo/issues/70",
      },
    });

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    expect(mockGithub.rest.issues.get).toHaveBeenCalled();
    expect(mockGithub.rest.issues.lock).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith("Issue #70 is already locked, skipping");
  });

  it("should work in staged mode", async () => {
    setAgentOutput({
      items: [
        {
          type: "lock_issue",
          issue_number: 80,
          body: "This is a test lock.",
          lock_reason: "resolved",
        },
      ],
    });

    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    // Should not call GitHub API in staged mode
    expect(mockGithub.rest.issues.get).not.toHaveBeenCalled();
    expect(mockGithub.rest.issues.lock).not.toHaveBeenCalled();

    // Should write staged preview to summary
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("🎭 Staged Mode: Lock Issues Preview"));
    expect(mockCore.info).toHaveBeenCalledWith("📝 Issue lock preview written to step summary");
  });

  it("should handle multiple issues in batch", async () => {
    setAgentOutput({
      items: [
        {
          type: "lock_issue",
          issue_number: 91,
          body: "Locking issue 91.",
        },
        {
          type: "lock_issue",
          issue_number: 92,
          body: "Locking issue 92.",
          lock_reason: "off-topic",
        },
      ],
    });

    process.env.GH_AW_LOCK_ISSUE_TARGET = "*";

    // Mock responses for both issues
    mockGithub.rest.issues.get
      .mockResolvedValueOnce({
        data: {
          number: 91,
          title: "Issue 91",
          labels: [],
          locked: false,
          html_url: "https://github.com/testowner/testrepo/issues/91",
        },
      })
      .mockResolvedValueOnce({
        data: {
          number: 92,
          title: "Issue 92",
          labels: [],
          locked: false,
          html_url: "https://github.com/testowner/testrepo/issues/92",
        },
      });

    mockGithub.rest.issues.createComment.mockResolvedValue({
      data: {
        id: 999,
        html_url: "https://github.com/testowner/testrepo/issues/91#issuecomment-999",
      },
    });

    mockGithub.rest.issues.lock.mockResolvedValue({});

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    expect(mockGithub.rest.issues.get).toHaveBeenCalledTimes(2);
    expect(mockGithub.rest.issues.lock).toHaveBeenCalledTimes(2);
    expect(mockCore.info).toHaveBeenCalledWith("Successfully locked 2 issue(s)");
  });

  it("should skip when not in issue context and target is triggering", async () => {
    setAgentOutput({
      items: [
        {
          type: "lock_issue",
          body: "Locking issue.",
        },
      ],
    });

    // Change context to non-issue event
    global.context.eventName = "push";
    delete global.context.payload.issue;

    // Execute the script
    await eval(`(async () => { ${lockIssueScript} })()`);

    expect(mockGithub.rest.issues.lock).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith('Target is "triggering" but not running in issue context, skipping issue lock');
  });
});
