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
    isDebug: vi.fn().mockReturnValue(!1),
    getIDToken: vi.fn(),
    toPlatformPath: vi.fn(),
    toPosixPath: vi.fn(),
    toWin32Path: vi.fn(),
    summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
  },
  mockContext = { eventName: "issues", payload: {}, runId: 12345, repo: { owner: "testowner", repo: "testrepo" } };
((global.core = mockCore),
  (global.context = mockContext),
  describe("check_command_position.cjs", () => {
    let checkCommandPositionScript, originalEnv;
    (beforeEach(() => {
      (vi.clearAllMocks(), (originalEnv = { GH_AW_COMMANDS: process.env.GH_AW_COMMANDS }));
      const scriptPath = path.join(__dirname, "check_command_position.cjs");
      ((checkCommandPositionScript = fs.readFileSync(scriptPath, "utf8")), (mockContext.eventName = "issues"), (mockContext.payload = {}));
    }),
      afterEach(() => {
        void 0 !== originalEnv.GH_AW_COMMANDS ? (process.env.GH_AW_COMMANDS = originalEnv.GH_AW_COMMANDS) : delete process.env.GH_AW_COMMANDS;
      }),
      it("should fail when GH_AW_COMMANDS is not set", async () => {
        (delete process.env.GH_AW_COMMANDS, await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`), expect(mockCore.setFailed).toHaveBeenCalledWith("Configuration error: GH_AW_COMMANDS not specified."));
      }),
      it("should pass when command is the first word in issue body", async () => {
        ((process.env.GH_AW_COMMANDS = JSON.stringify(["test-bot"])),
          (mockContext.eventName = "issues"),
          (mockContext.payload = { issue: { body: "/test-bot please help with this issue" } }),
          await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`),
          expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true"),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("matched at the start")));
      }),
      it("should fail when command is not the first word in issue body", async () => {
        ((process.env.GH_AW_COMMANDS = JSON.stringify(["test-bot"])),
          (mockContext.eventName = "issues"),
          (mockContext.payload = { issue: { body: "Please help with /test-bot this issue" } }),
          await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`),
          expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "false"),
          expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("None of the commands")));
      }),
      it("should pass when command is first word after whitespace", async () => {
        ((process.env.GH_AW_COMMANDS = JSON.stringify(["helper"])),
          (mockContext.eventName = "issue_comment"),
          (mockContext.payload = { comment: { body: "  \n  /helper analyze this code" } }),
          await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`),
          expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true"));
      }),
      it("should pass for non-comment events", async () => {
        ((process.env.GH_AW_COMMANDS = JSON.stringify(["test-bot"])),
          (mockContext.eventName = "workflow_dispatch"),
          (mockContext.payload = {}),
          await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`),
          expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true"),
          expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("does not require command position check")));
      }),
      it("should handle pull_request event with command at start", async () => {
        ((process.env.GH_AW_COMMANDS = JSON.stringify(["review-bot"])),
          (mockContext.eventName = "pull_request"),
          (mockContext.payload = { pull_request: { body: "/review-bot please review my changes" } }),
          await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`),
          expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true"));
      }),
      it("should pass when text is empty", async () => {
        ((process.env.GH_AW_COMMANDS = JSON.stringify(["test-bot"])),
          (mockContext.eventName = "issues"),
          (mockContext.payload = { issue: { body: "" } }),
          await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`),
          expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "false"),
          expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("None of the commands")));
      }),
      it("should pass when text does not contain the command", async () => {
        ((process.env.GH_AW_COMMANDS = JSON.stringify(["test-bot"])),
          (mockContext.eventName = "issues"),
          (mockContext.payload = { issue: { body: "This is a regular issue without any command" } }),
          await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`),
          expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "false"),
          expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("None of the commands")));
      }),
      it("should handle discussion events", async () => {
        ((process.env.GH_AW_COMMANDS = JSON.stringify(["discuss-bot"])),
          (mockContext.eventName = "discussion"),
          (mockContext.payload = { discussion: { body: "/discuss-bot help needed here" } }),
          await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`),
          expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true"));
      }),
      it("should handle discussion_comment events", async () => {
        ((process.env.GH_AW_COMMANDS = JSON.stringify(["discuss-bot"])),
          (mockContext.eventName = "discussion_comment"),
          (mockContext.payload = { comment: { body: "/discuss-bot analyze this" } }),
          await eval(`(async () => { ${checkCommandPositionScript}; await main(); })()`),
          expect(mockCore.setOutput).toHaveBeenCalledWith("command_position_ok", "true"));
      }));
  }));
