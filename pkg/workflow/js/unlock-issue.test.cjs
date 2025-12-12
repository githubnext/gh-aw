import { describe, it, expect, beforeEach, vi } from "vitest";
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
};

const mockGithub = {
  rest: {
    issues: {
      unlock: vi.fn(),
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
  issue: {
    number: 42,
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

describe("unlock-issue", () => {
  let unlockIssueScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.issue = { number: 42 };
    global.context.payload.issue = { number: 42 };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "unlock-issue.cjs");
    unlockIssueScript = fs.readFileSync(scriptPath, "utf8");
  });

  it("should unlock issue successfully", async () => {
    // Mock successful unlock
    mockGithub.rest.issues.unlock.mockResolvedValue({
      status: 204,
    });

    // Execute the script
    await eval(`(async () => { ${unlockIssueScript} })()`);

    expect(mockGithub.rest.issues.unlock).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 42,
    });

    expect(mockCore.info).toHaveBeenCalledWith("Unlocking issue #42 after agent workflow execution");
    expect(mockCore.info).toHaveBeenCalledWith("✅ Successfully unlocked issue #42");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should fail when issue number is not found in context", async () => {
    // Remove issue number from context
    global.context.issue = {};
    delete global.context.payload.issue;

    // Execute the script
    await eval(`(async () => { ${unlockIssueScript} })()`);

    expect(mockGithub.rest.issues.unlock).not.toHaveBeenCalled();
    expect(mockCore.setFailed).toHaveBeenCalledWith("Issue number not found in context");
  });

  it("should handle API errors gracefully", async () => {
    // Mock API error
    const apiError = new Error("Issue was not locked");
    mockGithub.rest.issues.unlock.mockRejectedValue(apiError);

    // Execute the script
    await eval(`(async () => { ${unlockIssueScript} })()`);

    expect(mockGithub.rest.issues.unlock).toHaveBeenCalled();
    expect(mockCore.error).toHaveBeenCalledWith("Failed to unlock issue: Issue was not locked");
    expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to unlock issue #42: Issue was not locked");
  });

  it("should handle non-Error exceptions", async () => {
    // Mock non-Error exception
    mockGithub.rest.issues.unlock.mockRejectedValue("String error");

    // Execute the script
    await eval(`(async () => { ${unlockIssueScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith("Failed to unlock issue: String error");
    expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to unlock issue #42: String error");
  });

  it("should work with different issue numbers", async () => {
    // Change issue number
    global.context.issue = { number: 200 };
    global.context.payload.issue = { number: 200 };

    mockGithub.rest.issues.unlock.mockResolvedValue({
      status: 204,
    });

    // Execute the script
    await eval(`(async () => { ${unlockIssueScript} })()`);

    expect(mockGithub.rest.issues.unlock).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 200,
    });

    expect(mockCore.info).toHaveBeenCalledWith("Unlocking issue #200 after agent workflow execution");
    expect(mockCore.info).toHaveBeenCalledWith("✅ Successfully unlocked issue #200");
  });

  it("should handle permission errors", async () => {
    // Mock permission error
    const permissionError = new Error("Resource not accessible by integration");
    mockGithub.rest.issues.unlock.mockRejectedValue(permissionError);

    // Execute the script
    await eval(`(async () => { ${unlockIssueScript} })()`);

    expect(mockGithub.rest.issues.unlock).toHaveBeenCalled();
    expect(mockCore.error).toHaveBeenCalledWith("Failed to unlock issue: Resource not accessible by integration");
    expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to unlock issue #42: Resource not accessible by integration");
  });

  it("should work even when issue is already unlocked", async () => {
    // Some APIs return success even if already unlocked
    mockGithub.rest.issues.unlock.mockResolvedValue({
      status: 204,
    });

    // Execute the script
    await eval(`(async () => { ${unlockIssueScript} })()`);

    expect(mockGithub.rest.issues.unlock).toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith("✅ Successfully unlocked issue #42");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });
});
