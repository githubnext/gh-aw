import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(undefined) },
};

const mockContext = {
  eventName: "issues",
  payload: {},
  runId: 12345,
  repo: { owner: "testowner", repo: "testrepo" },
};

global.core = mockCore;
global.context = mockContext;

describe("get_trigger_label.cjs", () => {
  let scriptContent, originalEnv;

  beforeEach(() => {
    vi.clearAllMocks();
    originalEnv = { GH_AW_MATCHED_COMMAND: process.env.GH_AW_MATCHED_COMMAND };
    delete process.env.GH_AW_MATCHED_COMMAND;
    const scriptPath = path.join(__dirname, "get_trigger_label.cjs");
    scriptContent = fs.readFileSync(scriptPath, "utf8");
    mockContext.eventName = "issues";
    mockContext.payload = {};
  });

  afterEach(() => {
    if (originalEnv.GH_AW_MATCHED_COMMAND !== undefined) {
      process.env.GH_AW_MATCHED_COMMAND = originalEnv.GH_AW_MATCHED_COMMAND;
    } else {
      delete process.env.GH_AW_MATCHED_COMMAND;
    }
  });

  const run = () => eval(`(async () => { ${scriptContent}; await main(); })()`);

  // ── labeled events ──────────────────────────────────────────────────────────

  it("should output label name as command_name for labeled issues event", async () => {
    mockContext.eventName = "issues";
    mockContext.payload = { action: "labeled", label: { name: "deploy" } };
    await run();
    expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "deploy");
    expect(mockCore.setOutput).toHaveBeenCalledWith("command_name", "deploy");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should output label name as command_name for labeled pull_request event", async () => {
    mockContext.eventName = "pull_request";
    mockContext.payload = { action: "labeled", label: { name: "ship-it" } };
    await run();
    expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "ship-it");
    expect(mockCore.setOutput).toHaveBeenCalledWith("command_name", "ship-it");
  });

  it("should output label name as command_name for labeled discussion event", async () => {
    mockContext.eventName = "discussion";
    mockContext.payload = { action: "labeled", label: { name: "triage" } };
    await run();
    expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "triage");
    expect(mockCore.setOutput).toHaveBeenCalledWith("command_name", "triage");
  });

  // ── workflow_dispatch ────────────────────────────────────────────────────────

  it("should output empty strings for workflow_dispatch without GH_AW_MATCHED_COMMAND", async () => {
    mockContext.eventName = "workflow_dispatch";
    mockContext.payload = {};
    await run();
    expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "");
    expect(mockCore.setOutput).toHaveBeenCalledWith("command_name", "");
  });

  it("should output matched command for workflow_dispatch when GH_AW_MATCHED_COMMAND is set", async () => {
    process.env.GH_AW_MATCHED_COMMAND = "fix";
    mockContext.eventName = "workflow_dispatch";
    mockContext.payload = {};
    await run();
    expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "");
    expect(mockCore.setOutput).toHaveBeenCalledWith("command_name", "fix");
  });

  // ── non-labeled events (slash-command context) ───────────────────────────────

  it("should use GH_AW_MATCHED_COMMAND as command_name for issue_comment without label", async () => {
    process.env.GH_AW_MATCHED_COMMAND = "review";
    mockContext.eventName = "issue_comment";
    mockContext.payload = { comment: { body: "/review please" } };
    await run();
    expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "");
    expect(mockCore.setOutput).toHaveBeenCalledWith("command_name", "review");
  });

  it("should output empty strings for non-labeled event without GH_AW_MATCHED_COMMAND", async () => {
    mockContext.eventName = "issue_comment";
    mockContext.payload = { comment: { body: "just a comment" } };
    await run();
    expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "");
    expect(mockCore.setOutput).toHaveBeenCalledWith("command_name", "");
  });

  // ── label name takes precedence over GH_AW_MATCHED_COMMAND ──────────────────

  it("should prefer label name over GH_AW_MATCHED_COMMAND for labeled events", async () => {
    process.env.GH_AW_MATCHED_COMMAND = "some-command";
    mockContext.eventName = "issues";
    mockContext.payload = { action: "labeled", label: { name: "deploy" } };
    await run();
    expect(mockCore.setOutput).toHaveBeenCalledWith("label_name", "deploy");
    expect(mockCore.setOutput).toHaveBeenCalledWith("command_name", "deploy");
  });
});
