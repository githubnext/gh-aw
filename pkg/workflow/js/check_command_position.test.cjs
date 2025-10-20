import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
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

  // Input/state functions
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

const mockContext = {
  eventName: "issues",
  payload: {},
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
};

// Set up global variables
global.core = mockCore;
global.context = mockContext;

describe("check_command_position.cjs", () => {
  let checkCommandPositionScript;
  let originalEnv;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = {
      GH_AW_COMMAND: process.env.GH_AW_COMMAND,
    };

    // Load the script
    const scriptPath = path.join(__dirname, "check_command_position.cjs");
    checkCommandPositionScript = fs.readFileSync(scriptPath, "utf8");

    // Reset context
    mockContext.eventName = "issues";
    mockContext.payload = {};
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv.GH_AW_COMMAND !== undefined) {
      process.env.GH_AW_COMMAND = originalEnv.GH_AW_COMMAND;
    } else {
      delete process.env.GH_AW_COMMAND;
    }
  });

  it("should fail when GH_AW_COMMAND is not set", async () => {
    delete process.env.GH_AW_COMMAND;

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setFailed).toHaveBeenCalledWith("Configuration error: GH_AW_COMMAND not specified.");
  });

  it("should pass when command is the first word in issue body", async () => {
    process.env.GH_AW_COMMAND = "test-bot";
    mockContext.eventName = "issues";
    mockContext.payload = {
      issue: {
        body: "/test-bot please help with this issue",
      },
    };

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Command '/test-bot' is at the start"));
  });

  it("should fail when command is not the first word in issue body", async () => {
    process.env.GH_AW_COMMAND = "test-bot";
    mockContext.eventName = "issues";
    mockContext.payload = {
      issue: {
        body: "Please help with /test-bot this issue",
      },
    };

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "false");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Command '/test-bot' is not the first word"));
  });

  it("should pass when command is first word after whitespace", async () => {
    process.env.GH_AW_COMMAND = "helper";
    mockContext.eventName = "issue_comment";
    mockContext.payload = {
      comment: {
        body: "  \n  /helper analyze this code",
      },
    };

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true");
  });

  it("should pass for non-comment events", async () => {
    process.env.GH_AW_COMMAND = "test-bot";
    mockContext.eventName = "workflow_dispatch";
    mockContext.payload = {};

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("does not require command position check"));
  });

  it("should handle pull_request event with command at start", async () => {
    process.env.GH_AW_COMMAND = "review-bot";
    mockContext.eventName = "pull_request";
    mockContext.payload = {
      pull_request: {
        body: "/review-bot please review my changes",
      },
    };

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true");
  });

  it("should pass when text is empty", async () => {
    process.env.GH_AW_COMMAND = "test-bot";
    mockContext.eventName = "issues";
    mockContext.payload = {
      issue: {
        body: "",
      },
    };

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No command"));
  });

  it("should pass when text does not contain the command", async () => {
    process.env.GH_AW_COMMAND = "test-bot";
    mockContext.eventName = "issues";
    mockContext.payload = {
      issue: {
        body: "This is a regular issue without any command",
      },
    };

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No command"));
  });

  it("should handle discussion events", async () => {
    process.env.GH_AW_COMMAND = "discuss-bot";
    mockContext.eventName = "discussion";
    mockContext.payload = {
      discussion: {
        body: "/discuss-bot help needed here",
      },
    };

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true");
  });

  it("should handle discussion_comment events", async () => {
    process.env.GH_AW_COMMAND = "discuss-bot";
    mockContext.eventName = "discussion_comment";
    mockContext.payload = {
      comment: {
        body: "/discuss-bot analyze this",
      },
    };

    await eval(`(async () => { ${checkCommandPositionScript} })()`);

    expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true");
  });
});
