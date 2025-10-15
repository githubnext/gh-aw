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
  setCancelled: vi.fn(),
  setError: vi.fn(),

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

const mockGithub = {
  rest: {
    actions: {
      listRepoWorkflows: vi.fn(),
      disableWorkflow: vi.fn(),
    },
  },
};

const mockContext = {
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("check_stop_time.cjs", () => {
  let checkStopTimeScript;
  let originalEnv;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Store original environment
    originalEnv = {
      GITHUB_AW_STOP_TIME: process.env.GITHUB_AW_STOP_TIME,
      GITHUB_AW_WORKFLOW_NAME: process.env.GITHUB_AW_WORKFLOW_NAME,
    };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "check_stop_time.cjs");
    checkStopTimeScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Restore original environment
    if (originalEnv.GITHUB_AW_STOP_TIME !== undefined) {
      process.env.GITHUB_AW_STOP_TIME = originalEnv.GITHUB_AW_STOP_TIME;
    } else {
      delete process.env.GITHUB_AW_STOP_TIME;
    }
    if (originalEnv.GITHUB_AW_WORKFLOW_NAME !== undefined) {
      process.env.GITHUB_AW_WORKFLOW_NAME = originalEnv.GITHUB_AW_WORKFLOW_NAME;
    } else {
      delete process.env.GITHUB_AW_WORKFLOW_NAME;
    }
  });

  describe("when stop time is not configured", () => {
    it("should allow execution if GITHUB_AW_STOP_TIME is not set", async () => {
      delete process.env.GITHUB_AW_STOP_TIME;
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";

      await eval(`(async () => { ${checkStopTimeScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("GITHUB_AW_STOP_TIME not specified"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("stop_time_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "config_error");
    });

    it("should allow execution if GITHUB_AW_WORKFLOW_NAME is not set", async () => {
      process.env.GITHUB_AW_STOP_TIME = "2025-12-31 23:59:59";
      delete process.env.GITHUB_AW_WORKFLOW_NAME;

      await eval(`(async () => { ${checkStopTimeScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("GITHUB_AW_WORKFLOW_NAME not specified"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("stop_time_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "config_error");
    });
  });

  describe("when stop time format is invalid", () => {
    it("should allow execution with warning for invalid format", async () => {
      process.env.GITHUB_AW_STOP_TIME = "invalid-date";
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";

      await eval(`(async () => { ${checkStopTimeScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Invalid stop-time format"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("stop_time_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "invalid_format");
    });
  });

  describe("when stop time is in the future", () => {
    it("should allow execution", async () => {
      // Set stop time to 1 year in the future
      const futureDate = new Date();
      futureDate.setFullYear(futureDate.getFullYear() + 1);
      const stopTime = futureDate.toISOString().replace("T", " ").substring(0, 19);

      process.env.GITHUB_AW_STOP_TIME = stopTime;
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";

      await eval(`(async () => { ${checkStopTimeScript} })()`);

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("All safety checks passed"));
      expect(mockCore.setOutput).toHaveBeenCalledWith("stop_time_ok", "true");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "ok");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });

  describe("when stop time has been reached", () => {
    it("should set stop_time_ok to false without attempting to disable workflow", async () => {
      // Set stop time to 1 year in the past
      const pastDate = new Date();
      pastDate.setFullYear(pastDate.getFullYear() - 1);
      const stopTime = pastDate.toISOString().replace("T", " ").substring(0, 19);

      process.env.GITHUB_AW_STOP_TIME = stopTime;
      process.env.GITHUB_AW_WORKFLOW_NAME = "test-workflow";

      await eval(`(async () => { ${checkStopTimeScript} })()`);

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Stop time reached"));
      // Should NOT attempt to disable workflow
      expect(mockGithub.rest.actions.listRepoWorkflows).not.toHaveBeenCalled();
      expect(mockGithub.rest.actions.disableWorkflow).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("stop_time_ok", "false");
      expect(mockCore.setOutput).toHaveBeenCalledWith("result", "stop_time_reached");
      // Should NOT call setFailed - let the activation job handle it
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });
  });
});
