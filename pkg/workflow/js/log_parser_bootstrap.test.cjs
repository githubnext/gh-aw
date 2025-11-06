import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe("log_parser_bootstrap.cjs", () => {
  let mockCore;
  let runLogParser;
  let originalProcess;

  beforeEach(() => {
    // Save originals before mocking
    originalProcess = { ...process };

    // Mock core actions methods
    mockCore = {
      debug: vi.fn(),
      info: vi.fn(),
      notice: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      exportVariable: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    global.core = mockCore;

    // Import the module after setting up global.core
    const module = require("./log_parser_bootstrap.cjs");
    runLogParser = module.runLogParser;
  });

  afterEach(() => {
    // Restore originals
    process.env = originalProcess.env;
    vi.restoreAllMocks();
    delete global.core;
  });

  describe("runLogParser", () => {
    it("should handle missing GH_AW_AGENT_OUTPUT environment variable", () => {
      delete process.env.GH_AW_AGENT_OUTPUT;

      const mockParseLog = vi.fn();
      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
      });

      expect(mockCore.info).toHaveBeenCalledWith("No agent log file specified");
      expect(mockParseLog).not.toHaveBeenCalled();
    });

    it("should handle non-existent log file", () => {
      process.env.GH_AW_AGENT_OUTPUT = "/non/existent/file.log";

      const mockParseLog = vi.fn();
      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
      });

      expect(mockCore.info).toHaveBeenCalledWith("Log path not found: /non/existent/file.log");
      expect(mockParseLog).not.toHaveBeenCalled();
    });

    it("should read and parse a single log file", () => {
      // Create a temporary log file
      const tmpDir = fs.mkdtempSync(path.join(__dirname, "test-"));
      const logFile = path.join(tmpDir, "test.log");
      const logContent = "Test log content";
      fs.writeFileSync(logFile, logContent);

      process.env.GH_AW_AGENT_OUTPUT = logFile;

      const mockParseLog = vi.fn().mockReturnValue("## Parsed Log\n\nSuccess!");
      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
      });

      expect(mockParseLog).toHaveBeenCalledWith(logContent);
      expect(mockCore.info).toHaveBeenCalledWith("## Parsed Log\n\nSuccess!");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith("## Parsed Log\n\nSuccess!");
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith("TestParser log parsed successfully");

      // Cleanup
      fs.unlinkSync(logFile);
      fs.rmdirSync(tmpDir);
    });

    it("should handle parser returning object with markdown", () => {
      const tmpDir = fs.mkdtempSync(path.join(__dirname, "test-"));
      const logFile = path.join(tmpDir, "test.log");
      fs.writeFileSync(logFile, "content");

      process.env.GH_AW_AGENT_OUTPUT = logFile;

      const mockParseLog = vi.fn().mockReturnValue({
        markdown: "## Result\n",
        mcpFailures: [],
        maxTurnsHit: false,
      });

      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
      });

      expect(mockCore.info).toHaveBeenCalledWith("## Result\n");
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith("## Result\n");
      expect(mockCore.setFailed).not.toHaveBeenCalled();

      fs.unlinkSync(logFile);
      fs.rmdirSync(tmpDir);
    });

    it("should handle MCP failures", () => {
      const tmpDir = fs.mkdtempSync(path.join(__dirname, "test-"));
      const logFile = path.join(tmpDir, "test.log");
      fs.writeFileSync(logFile, "content");

      process.env.GH_AW_AGENT_OUTPUT = logFile;

      const mockParseLog = vi.fn().mockReturnValue({
        markdown: "## Result\n",
        mcpFailures: ["server1", "server2"],
        maxTurnsHit: false,
      });

      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
      });

      expect(mockCore.setFailed).toHaveBeenCalledWith("MCP server(s) failed to launch: server1, server2");

      fs.unlinkSync(logFile);
      fs.rmdirSync(tmpDir);
    });

    it("should handle max-turns limit reached", () => {
      const tmpDir = fs.mkdtempSync(path.join(__dirname, "test-"));
      const logFile = path.join(tmpDir, "test.log");
      fs.writeFileSync(logFile, "content");

      process.env.GH_AW_AGENT_OUTPUT = logFile;

      const mockParseLog = vi.fn().mockReturnValue({
        markdown: "## Result\n",
        mcpFailures: [],
        maxTurnsHit: true,
      });

      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
      });

      expect(mockCore.setFailed).toHaveBeenCalledWith(
        "Agent execution stopped: max-turns limit reached. The agent did not complete its task successfully."
      );

      fs.unlinkSync(logFile);
      fs.rmdirSync(tmpDir);
    });

    it("should read and concatenate multiple log files from directory when supportsDirectories is true", () => {
      const tmpDir = fs.mkdtempSync(path.join(__dirname, "test-"));
      const logFile1 = path.join(tmpDir, "1.log");
      const logFile2 = path.join(tmpDir, "2.log");
      fs.writeFileSync(logFile1, "First log");
      fs.writeFileSync(logFile2, "Second log");

      process.env.GH_AW_AGENT_OUTPUT = tmpDir;

      const mockParseLog = vi.fn().mockReturnValue("## Parsed");
      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
        supportsDirectories: true,
      });

      expect(mockParseLog).toHaveBeenCalledWith("First log\nSecond log");

      fs.unlinkSync(logFile1);
      fs.unlinkSync(logFile2);
      fs.rmdirSync(tmpDir);
    });

    it("should reject directories when supportsDirectories is false", () => {
      const tmpDir = fs.mkdtempSync(path.join(__dirname, "test-"));
      fs.writeFileSync(path.join(tmpDir, "1.log"), "content");

      process.env.GH_AW_AGENT_OUTPUT = tmpDir;

      const mockParseLog = vi.fn();
      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
        supportsDirectories: false,
      });

      expect(mockCore.info).toHaveBeenCalledWith(`Log path is a directory but TestParser parser does not support directories: ${tmpDir}`);
      expect(mockParseLog).not.toHaveBeenCalled();

      fs.unlinkSync(path.join(tmpDir, "1.log"));
      fs.rmdirSync(tmpDir);
    });

    it("should handle empty directory gracefully", () => {
      const tmpDir = fs.mkdtempSync(path.join(__dirname, "test-"));

      process.env.GH_AW_AGENT_OUTPUT = tmpDir;

      const mockParseLog = vi.fn();
      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
        supportsDirectories: true,
      });

      expect(mockCore.info).toHaveBeenCalledWith(`No log files found in directory: ${tmpDir}`);
      expect(mockParseLog).not.toHaveBeenCalled();

      fs.rmdirSync(tmpDir);
    });

    it("should handle parser errors", () => {
      const tmpDir = fs.mkdtempSync(path.join(__dirname, "test-"));
      const logFile = path.join(tmpDir, "test.log");
      fs.writeFileSync(logFile, "content");

      process.env.GH_AW_AGENT_OUTPUT = logFile;

      const mockParseLog = vi.fn().mockImplementation(() => {
        throw new Error("Parser error");
      });

      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
      });

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.any(Error));

      fs.unlinkSync(logFile);
      fs.rmdirSync(tmpDir);
    });

    it("should handle failed parse (empty result)", () => {
      const tmpDir = fs.mkdtempSync(path.join(__dirname, "test-"));
      const logFile = path.join(tmpDir, "test.log");
      fs.writeFileSync(logFile, "content");

      process.env.GH_AW_AGENT_OUTPUT = logFile;

      const mockParseLog = vi.fn().mockReturnValue("");

      runLogParser({
        parseLog: mockParseLog,
        parserName: "TestParser",
      });

      expect(mockCore.error).toHaveBeenCalledWith("Failed to parse TestParser log");

      fs.unlinkSync(logFile);
      fs.rmdirSync(tmpDir);
    });
  });
});
