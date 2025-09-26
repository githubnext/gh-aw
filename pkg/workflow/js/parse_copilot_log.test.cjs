import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("parse_copilot_log.cjs", () => {
  let mockCore;
  let parseCopilotLogScript;
  let originalConsole;
  let originalProcess;

  beforeEach(() => {
    // Save originals before mocking
    originalConsole = global.console;
    originalProcess = { ...process };

    // Mock console methods
    global.console = {
      log: vi.fn(),
      error: vi.fn(),
    };

    // Mock core actions methods
    mockCore = {
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
    global.core = mockCore;

    // Mock require
    global.require = vi.fn().mockImplementation(module => {
      if (module === "fs") {
        return fs;
      }
      if (module === "@actions/core") {
        return mockCore;
      }
      throw new Error(`Module not found: ${module}`);
    });

    // Read the script file
    const scriptPath = path.join(__dirname, "parse_copilot_log.cjs");
    parseCopilotLogScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up environment variables
    delete process.env.AGENT_LOG_FILE;

    // Restore originals
    global.console = originalConsole;
    process.env = originalProcess.env;

    // Clean up globals
    delete global.core;
    delete global.require;
  });

  const runScript = async logContent => {
    // Create a temporary log file
    const tempFile = path.join(process.cwd(), `test_copilot_log_${Date.now()}.txt`);
    fs.writeFileSync(tempFile, logContent);
    process.env.AGENT_LOG_FILE = tempFile;

    try {
      // Create a new function context to execute the script
      const scriptWithExports = parseCopilotLogScript.replace(
        "main();",
        "global.testParseCopilotLog = parseCopilotLog; global.testMain = main; main();"
      );
      const scriptFunction = new Function(scriptWithExports);
      await scriptFunction();
    } finally {
      // Clean up temp file
      if (fs.existsSync(tempFile)) {
        fs.unlinkSync(tempFile);
      }
    }
  };

  const extractParseFunction = () => {
    // Extract just the parseCopilotLog function for unit testing
    const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testParseCopilotLog = parseCopilotLog;");
    const scriptFunction = new Function(scriptWithExport);
    scriptFunction();
    return global.testParseCopilotLog;
  };

  describe("parseCopilotLog function", () => {
    let parseCopilotLog;

    beforeEach(() => {
      parseCopilotLog = extractParseFunction();
    });

    it("should parse copilot CLI execution with commands and tools", () => {
      const copilotLog = `2024-09-16T18:10:29.123Z [INFO] Starting GitHub Copilot CLI
copilot --add-dir /tmp/ --log-level debug --log-dir /tmp/logs --prompt "Analyze this repository"
2024-09-16T18:10:30.456Z [INFO] Copilot CLI version: 1.0.0
2024-09-16T18:10:31.123Z [INFO] Connected to github MCP server successfully
2024-09-16T18:10:31.200Z [INFO] Connected to safe_outputs MCP server successfully
2024-09-16T18:10:31.201Z [DEBUG] Available tools: shell, write, github_create_issue
2024-09-16T18:10:33.456Z [DEBUG] Executing shell command: find /tmp -name "*.md" | head -5
2024-09-16T18:10:33.500Z [INFO] Shell command executed successfully
2024-09-16T18:10:35.791Z [DEBUG] Total execution time: 6.67 seconds
2024-09-16T18:10:35.800Z [INFO] GitHub Copilot CLI execution completed`;

      const result = parseCopilotLog(copilotLog);

      expect(result).toContain(
        '🚀 **Command:** `copilot --add-dir /tmp/ --log-level debug --log-dir /tmp/logs --prompt "Analyze this repository"`'
      );
      expect(result).toContain("🔗 **MCP Server Connected:** github");
      expect(result).toContain("🔗 **MCP Server Connected:** safe_outputs");
      expect(result).toContain("🛠️ **Available Tools:** shell, write, github_create_issue");
      expect(result).toContain('✅ `find /tmp -name "*.md" | head -5`');
      expect(result).toContain("**Execution Time:** 6.67 seconds");
      expect(result).toContain("**Tools Used:** shell, write, github_create_issue");
      expect(result).toContain("**Commands Executed:** 2");
    });

    it("should parse copilot suggestions and responses", () => {
      const copilotLog = `2024-09-16T18:10:32.000Z [INFO] Processing user prompt...

Suggestion: I'll analyze the repository structure and create a comprehensive summary.

Let me start by examining the directory structure:

\`\`\`bash
find /tmp -type f -name "*.md" | head -10
\`\`\`

2024-09-16T18:10:33.456Z [DEBUG] Executing shell command: find /tmp -type f -name "*.md" | head -10
README.md
docs/guide.md

Now let me create a summary document:

\`\`\`markdown
# Repository Analysis Summary

## Overview
This repository contains important documentation.
\`\`\`

2024-09-16T18:10:35.790Z [INFO] Generated summary document`;

      const result = parseCopilotLog(copilotLog);

      expect(result).toContain("**Suggestion: I'll analyze the repository structure and create a comprehensive summary.**");
      expect(result).toContain("Let me start by examining the directory structure:");
      expect(result).toContain('```bash\nfind /tmp -type f -name "*.md" | head -10\n```');
      expect(result).toContain(
        "```markdown\n# Repository Analysis Summary\n\n## Overview\nThis repository contains important documentation.\n```"
      );
      expect(result).toContain("Now let me create a summary document:");
    });

    it("should parse errors and warnings correctly", () => {
      const copilotLog = `2024-09-16T18:10:29.123Z [INFO] Starting GitHub Copilot CLI
2024-09-16T18:10:36.123Z [ERROR] Failed to save final output: Permission denied
Warning: Some tools may not be available in restricted mode
npm ERR! Could not install package @github/copilot
copilot: error: Invalid authentication token provided
Fatal error: Unable to complete workflow execution
[2024-09-16T18:10:36.500Z] [WARNING]: MCP server connection timeout`;

      const result = parseCopilotLog(copilotLog);

      expect(result).toContain("## ❌ Errors");
      expect(result).toContain("* Failed to save final output: Permission denied");
      expect(result).toContain("* Could not install package @github/copilot");
      expect(result).toContain("* Invalid authentication token provided");
      expect(result).toContain("* Unable to complete workflow execution");
      expect(result).toContain("## ⚠️ Warnings");
      expect(result).toContain("* Some tools may not be available in restricted mode");
      expect(result).toContain("* MCP server connection timeout");
    });

    it("should handle empty or minimal log content", () => {
      const emptyLog = "";
      const result = parseCopilotLog(emptyLog);

      expect(result).toContain("🤖 Commands and Tools");
      expect(result).toContain("📋 Execution Output");
      expect(result).toContain("📊 Information");
      expect(result).toContain("*No significant output captured from GitHub Copilot CLI execution.*");
    });

    it("should handle shell command execution with failure", () => {
      const copilotLog = `2024-09-16T18:10:33.456Z [DEBUG] Executing shell command: nonexistent-command --help
2024-09-16T18:10:33.500Z [ERROR] Shell command failed: command not found
2024-09-16T18:10:34.123Z [DEBUG] Executing shell command: echo "Hello World"
2024-09-16T18:10:34.200Z [INFO] Shell command executed successfully`;

      const result = parseCopilotLog(copilotLog);

      expect(result).toContain("❌ `nonexistent-command --help`");
      expect(result).toContain('✅ `echo "Hello World"`');
      expect(result).toContain("**Tools Used:** shell");
    });

    it("should truncate very long commands", () => {
      const longCommand = "echo " + "a".repeat(100);
      const copilotLog = `2024-09-16T18:10:33.456Z [DEBUG] Executing shell command: ${longCommand}
2024-09-16T18:10:33.500Z [INFO] Shell command executed successfully`;

      const result = parseCopilotLog(copilotLog);

      expect(result).toContain("✅ `echo " + "a".repeat(72) + "...`");
    });

    it("should handle invalid log content gracefully", () => {
      const invalidLog = "This is not a valid copilot log format\nJust some random text";

      const result = parseCopilotLog(invalidLog);

      expect(result).toContain("🤖 Commands and Tools");
      expect(result).toContain("*No significant output captured from GitHub Copilot CLI execution.*");
    });

    it("should parse mixed MCP server success and failure", () => {
      const copilotLog = `2024-09-16T18:10:31.123Z [INFO] Connected to github MCP server successfully
2024-09-16T18:10:31.200Z [ERROR] Failed to connect to broken_server MCP server
2024-09-16T18:10:31.300Z [INFO] Connected to safe_outputs MCP server successfully`;

      const result = parseCopilotLog(copilotLog);

      expect(result).toContain("🔗 **MCP Server Connected:** github");
      expect(result).toContain("🔗 **MCP Server Connected:** safe_outputs");
      expect(result).toContain("* Failed to connect to broken_server MCP server");
    });
  });

  describe("main function integration", () => {
    it("should handle valid copilot log file", async () => {
      const validLog = `2024-09-16T18:10:29.123Z [INFO] Starting GitHub Copilot CLI
copilot --add-dir /tmp/ --log-level debug --prompt "Test task"
2024-09-16T18:10:31.123Z [INFO] Connected to github MCP server successfully
2024-09-16T18:10:35.800Z [INFO] GitHub Copilot CLI execution completed`;

      await runScript(validLog);

      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(global.console.log).toHaveBeenCalledWith("Copilot log parsed successfully");

      // Check that markdown was added to summary
      const markdownCall = mockCore.summary.addRaw.mock.calls[0];
      expect(markdownCall[0]).toContain('🚀 **Command:** `copilot --add-dir /tmp/ --log-level debug --prompt "Test task"`');
      expect(markdownCall[0]).toContain("🔗 **MCP Server Connected:** github");
    });

    it("should handle missing log file", async () => {
      process.env.AGENT_LOG_FILE = "/nonexistent/file.log";

      // Extract main function and run it directly
      const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      expect(global.console.log).toHaveBeenCalledWith("Log file not found: /nonexistent/file.log");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle missing environment variable", async () => {
      delete process.env.AGENT_LOG_FILE;

      // Extract main function and run it directly
      const scriptWithExport = parseCopilotLogScript.replace("main();", "global.testMain = main;");
      const scriptFunction = new Function(scriptWithExport);
      scriptFunction();
      await global.testMain();

      expect(global.console.log).toHaveBeenCalledWith("No agent log file specified");
      expect(mockCore.setFailed).not.toHaveBeenCalled();
    });

    it("should handle script execution errors", async () => {
      // Create a log file that will cause parsing errors
      const invalidLog = "Invalid log content that causes errors";

      await runScript(invalidLog);

      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
      expect(global.console.log).toHaveBeenCalledWith("Copilot log parsed successfully");

      // Should still produce output even with invalid content
      const markdownCall = mockCore.summary.addRaw.mock.calls[0];
      expect(markdownCall[0]).toContain("🤖 Commands and Tools");
    });
  });

  describe("formatBashCommand helper function", () => {
    let parseCopilotLog;

    beforeEach(() => {
      parseCopilotLog = extractParseFunction();
    });

    it("should normalize multi-line commands", () => {
      const copilotLog = `2024-09-16T18:10:33.456Z [DEBUG] Executing shell command: echo "hello world"
  && ls -la
  && pwd
2024-09-16T18:10:33.500Z [INFO] Shell command executed successfully`;

      const result = parseCopilotLog(copilotLog);

      expect(result).toContain('✅ `echo "hello world" && ls -la && pwd`');
    });

    it("should handle empty commands", () => {
      const copilotLog = `2024-09-16T18:10:33.456Z [DEBUG] Executing shell command: 
2024-09-16T18:10:33.500Z [INFO] Shell command executed successfully`;

      const result = parseCopilotLog(copilotLog);

      expect(result).toContain("✅ ``");
    });
  });
});
