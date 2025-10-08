import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

describe("parse_codex_log.cjs", () => {
  let mockCore;
  let parseCodexLog;
  let originalConsole;

  beforeEach(() => {
    // Save originals before mocking
    originalConsole = global.console;

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

      // Summary object with chainable methods
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(),
      },
    };
    global.core = mockCore;

    // Load the module
    const modulePath = path.join(__dirname, "parse_codex_log.cjs");
    const module = require(modulePath);
    parseCodexLog = module.parseCodexLog;
  });

  afterEach(() => {
    // Restore originals
    global.console = originalConsole;
    vi.clearAllMocks();
  });

  describe("parseCodexLog function", () => {
    it("should parse codex log with thinking sections correctly", () => {
      const sampleLogPath = path.join(__dirname, "..", "test_data", "sample_codex_log.txt");
      const logContent = fs.readFileSync(sampleLogPath, "utf8");

      const result = parseCodexLog(logContent);

      // Verify structure
      expect(result).toContain("## 🤖 Commands and Tools");
      expect(result).toContain("## 📊 Information");
      expect(result).toContain("## 🤖 Reasoning");

      // Verify commands section has tool calls
      const commandsSection = result.split("## 📊 Information")[0];
      expect(commandsSection).toContain("time::get_current_time");
      expect(commandsSection).toContain("github::list_pull_requests");
      expect(commandsSection).toContain("github::search_pull_requests");
      expect(commandsSection).toContain("bash -lc 'git remote -v'");

      // Verify information section
      const infoSection = result.split("## 📊 Information")[1].split("## 🤖 Reasoning")[0];
      expect(infoSection).toContain("Total Tokens Used:");
      expect(infoSection).toContain("Tool Calls:");
      expect(infoSection).toContain("Commands Executed:");

      // Verify reasoning section contains only reasoning text (no tool markers)
      const reasoningSection = result.split("## 🤖 Reasoning")[1];
      expect(reasoningSection).toContain("Planning diff analysis");
      expect(reasoningSection).toContain("Summarizing PR search results");
      expect(reasoningSection).toContain("Communicating PR status");

      // Critical: Reasoning section should NOT contain tool call markers
      expect(reasoningSection).not.toMatch(/✅.*get_current_time/);
      expect(reasoningSection).not.toMatch(/✅.*list_pull_requests/);
      expect(reasoningSection).not.toMatch(/✅.*bash -lc/);
      expect(reasoningSection).not.toMatch(/\] tool /);
      expect(reasoningSection).not.toMatch(/\] exec /);
    });

    it("should handle logs with multiple thinking sections", () => {
      const logWithMultipleThinking = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0 (research preview)
--------
workdir: /test
model: o4-mini
--------
[2025-08-31T12:37:47] thinking

**First thought**

This is the first reasoning section.

[2025-08-31T12:37:49] tool test.do_something({"arg": "value"})
[2025-08-31T12:37:50] test.do_something({"arg": "value"}) success in 10ms:
[2025-08-31T12:37:50] tokens used: 1000

[2025-08-31T12:38:35] thinking

**Second thought**

This is the second reasoning section.

[2025-08-31T12:38:40] exec bash -lc 'echo test' in /test
[2025-08-31T12:38:40] bash -lc 'echo test' succeeded in 5ms:
test
[2025-08-31T12:38:40] tokens used: 500
`;

      const result = parseCodexLog(logWithMultipleThinking);

      const reasoningSection = result.split("## 🤖 Reasoning")[1];

      // Should contain both thinking sections
      expect(reasoningSection).toContain("First thought");
      expect(reasoningSection).toContain("This is the first reasoning section");
      expect(reasoningSection).toContain("Second thought");
      expect(reasoningSection).toContain("This is the second reasoning section");

      // Should NOT contain tool or exec markers in reasoning
      expect(reasoningSection).not.toMatch(/test::do_something/);
      expect(reasoningSection).not.toMatch(/bash -lc 'echo test'/);
    });

    it("should extract token usage correctly", () => {
      const logWithTokens = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
[2025-08-31T12:37:50] tokens used: 1000
[2025-08-31T12:38:40] tokens used: 500
[2025-08-31T12:39:20] tokens used: 2500
`;

      const result = parseCodexLog(logWithTokens);

      // Should sum all token usages
      expect(result).toContain("**Total Tokens Used:** 4,000");
    });

    it("should count tool calls and exec commands", () => {
      const logWithToolsAndExecs = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
[2025-08-31T12:37:49] tool test.tool1({})
[2025-08-31T12:37:50] test.tool1({}) success in 10ms:
[2025-08-31T12:37:55] tool test.tool2({})
[2025-08-31T12:37:56] test.tool2({}) success in 10ms:
[2025-08-31T12:38:40] exec bash -lc 'echo test' in /test
[2025-08-31T12:38:40] bash -lc 'echo test' succeeded in 5ms:
[2025-08-31T12:38:45] exec bash -lc 'ls' in /test
[2025-08-31T12:38:45] bash -lc 'ls' succeeded in 5ms:
`;

      const result = parseCodexLog(logWithToolsAndExecs);

      expect(result).toContain("**Tool Calls:** 2");
      expect(result).toContain("**Commands Executed:** 2");
    });

    it("should handle empty logs gracefully", () => {
      const result = parseCodexLog("");

      expect(result).toContain("## 🤖 Commands and Tools");
      expect(result).toContain("No commands or tools used");
      expect(result).toContain("## 📊 Information");
      expect(result).toContain("## 🤖 Reasoning");
    });

    it("should skip metadata lines in reasoning section", () => {
      const logWithMetadata = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
--------
workdir: /test
model: o4-mini
provider: openai
approval: never
sandbox: workspace-write
reasoning effort: medium
reasoning summaries: auto
--------
[2025-08-31T12:37:47] thinking

**Planning something**

This is reasoning content.

[2025-08-31T12:37:50] tokens used: 1000
`;

      const result = parseCodexLog(logWithMetadata);

      const reasoningSection = result.split("## 🤖 Reasoning")[1];

      // Should contain reasoning
      expect(reasoningSection).toContain("Planning something");
      expect(reasoningSection).toContain("This is reasoning content");

      // Should NOT contain metadata
      expect(reasoningSection).not.toContain("OpenAI Codex");
      expect(reasoningSection).not.toContain("workdir:");
      expect(reasoningSection).not.toContain("model:");
      expect(reasoningSection).not.toContain("reasoning effort:");
      expect(reasoningSection).not.toContain("tokens used:");
    });

    it("should format tool names with dots correctly", () => {
      const logWithDottedTools = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
[2025-08-31T12:37:49] tool github.get_repository({"owner": "test", "repo": "repo"})
[2025-08-31T12:37:50] github.get_repository({"owner": "test", "repo": "repo"}) success in 100ms:
`;

      const result = parseCodexLog(logWithDottedTools);

      // Should format as provider::method
      expect(result).toContain("github::get_repository");
    });

    it("should handle tools with multiple dots correctly", () => {
      const logWithMultipleDots = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
[2025-08-31T12:37:49] tool some.nested.tool.name({"arg": "value"})
[2025-08-31T12:37:50] some.nested.tool.name({"arg": "value"}) success in 10ms:
`;

      const result = parseCodexLog(logWithMultipleDots);

      // Should join with underscores after first dot
      expect(result).toContain("some::nested_tool_name");
    });

    it("should detect tool/exec success and failure status", () => {
      const logWithStatuses = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0
[2025-08-31T12:37:49] tool test.success({})
[2025-08-31T12:37:50] test.success({}) success in 10ms:
[2025-08-31T12:37:55] tool test.failure({})
[2025-08-31T12:37:56] test.failure({}) failure in 10ms:
[2025-08-31T12:38:00] exec bash -lc 'echo ok' in /test
[2025-08-31T12:38:00] bash -lc 'echo ok' succeeded in 5ms:
[2025-08-31T12:38:05] exec bash -lc 'false' in /test
[2025-08-31T12:38:05] bash -lc 'false' failed in 5ms:
`;

      const result = parseCodexLog(logWithStatuses);

      const commandsSection = result.split("## 📊 Information")[0];

      // Check for success markers
      expect(commandsSection).toContain("✅");
      // Check for failure markers
      expect(commandsSection).toContain("❌");
    });
  });
});
