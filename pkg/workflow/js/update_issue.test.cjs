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
      update: vi.fn(),
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

describe("update_issue.cjs", () => {
  let updateIssueScript;

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
    delete process.env.GH_AW_UPDATE_STATUS;
    delete process.env.GH_AW_UPDATE_TITLE;
    delete process.env.GH_AW_UPDATE_BODY;
    delete process.env.GH_AW_UPDATE_TARGET;

    // Set default values
    process.env.GH_AW_UPDATE_STATUS = "false";
    process.env.GH_AW_UPDATE_TITLE = "false";
    process.env.GH_AW_UPDATE_BODY = "false";

    // Read the script
    const scriptPath = path.join(__dirname, "update_issue.cjs");
    updateIssueScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && require("fs").existsSync(tempFilePath)) {
      require("fs").unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  it("should skip when no agent output is provided", async () => {
    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("should skip when agent output is empty", async () => {
    setAgentOutput("");

    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("should skip when not in issue context for triggering target", async () => {
    setAgentOutput({
      items: [
        {
          type: "update_issue",
          title: "Updated title",
        },
      ],
    });
    process.env.GH_AW_UPDATE_TITLE = "true";
    global.context.eventName = "push"; // Not an issue event

    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith('Target is "triggering" but not running in issue context, skipping issue update');
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("should update issue title successfully", async () => {
    setAgentOutput({
      items: [
        {
          type: "update_issue",
          title: "Updated issue title",
        },
      ],
    });
    process.env.GH_AW_UPDATE_TITLE = "true";
    global.context.eventName = "issues";

    const mockIssue = {
      number: 123,
      title: "Updated issue title",
      html_url: "https://github.com/testowner/testrepo/issues/123",
    };

    mockGithub.rest.issues.update.mockResolvedValue({ data: mockIssue });

    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);

    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 123,
      title: "Updated issue title",
    });

    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", 123);
    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_url", mockIssue.html_url);
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should update issue status successfully", async () => {
    setAgentOutput({
      items: [
        {
          type: "update_issue",
          status: "closed",
        },
      ],
    });
    process.env.GH_AW_UPDATE_STATUS = "true";
    global.context.eventName = "issues";

    const mockIssue = {
      number: 123,
      html_url: "https://github.com/testowner/testrepo/issues/123",
    };

    mockGithub.rest.issues.update.mockResolvedValue({ data: mockIssue });

    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);

    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 123,
      state: "closed",
    });
  });

  it("should update multiple fields successfully", async () => {
    setAgentOutput({
      items: [
        {
          type: "update_issue",
          title: "New title",
          body: "New body content",
          status: "open",
        },
      ],
    });
    process.env.GH_AW_UPDATE_TITLE = "true";
    process.env.GH_AW_UPDATE_BODY = "true";
    process.env.GH_AW_UPDATE_STATUS = "true";
    global.context.eventName = "issues";

    const mockIssue = {
      number: 123,
      html_url: "https://github.com/testowner/testrepo/issues/123",
    };

    mockGithub.rest.issues.update.mockResolvedValue({ data: mockIssue });

    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);

    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 123,
      title: "New title",
      body: "New body content",
      state: "open",
    });
  });

  it('should handle explicit issue number with target "*"', async () => {
    setAgentOutput({
      items: [
        {
          type: "update_issue",
          issue_number: 456,
          title: "Updated title",
        },
      ],
    });
    process.env.GH_AW_UPDATE_TITLE = "true";
    process.env.GH_AW_UPDATE_TARGET = "*";
    global.context.eventName = "push"; // Not an issue event, but should work with explicit target

    const mockIssue = {
      number: 456,
      html_url: "https://github.com/testowner/testrepo/issues/456",
    };

    mockGithub.rest.issues.update.mockResolvedValue({ data: mockIssue });

    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);

    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 456,
      title: "Updated title",
    });
  });

  it("should skip when no valid updates are provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "update_issue",
          title: "New title",
        },
      ],
    });
    // All update flags are false
    process.env.GH_AW_UPDATE_STATUS = "false";
    process.env.GH_AW_UPDATE_TITLE = "false";
    process.env.GH_AW_UPDATE_BODY = "false";
    global.context.eventName = "issues";

    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No valid updates to apply for this item");
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("should validate status values", async () => {
    setAgentOutput({
      items: [
        {
          type: "update_issue",
          status: "invalid",
        },
      ],
    });
    process.env.GH_AW_UPDATE_STATUS = "true";
    global.context.eventName = "issues";

    // Execute the script
    await eval(`(async () => { ${updateIssueScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Invalid status value: invalid. Must be 'open' or 'closed'");
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });
});
