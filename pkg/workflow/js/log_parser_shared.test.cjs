import { describe, it, expect } from "vitest";

describe("log_parser_shared.cjs", () => {
  describe("formatDuration", () => {
    it("should format duration less than 60 seconds", async () => {
      const { formatDuration } = await import("./log_parser_shared.cjs");

      expect(formatDuration(5000)).toBe("5s");
      expect(formatDuration(30000)).toBe("30s");
      expect(formatDuration(59499)).toBe("59s"); // Just under 60s
    });

    it("should format duration in minutes without seconds", async () => {
      const { formatDuration } = await import("./log_parser_shared.cjs");

      expect(formatDuration(60000)).toBe("1m");
      expect(formatDuration(120000)).toBe("2m");
      expect(formatDuration(300000)).toBe("5m");
    });

    it("should format duration in minutes with seconds", async () => {
      const { formatDuration } = await import("./log_parser_shared.cjs");

      expect(formatDuration(65000)).toBe("1m 5s");
      expect(formatDuration(90000)).toBe("1m 30s");
      expect(formatDuration(125000)).toBe("2m 5s");
    });

    it("should handle zero and negative durations", async () => {
      const { formatDuration } = await import("./log_parser_shared.cjs");

      expect(formatDuration(0)).toBe("");
      expect(formatDuration(-1000)).toBe("");
    });

    it("should handle null and undefined", async () => {
      const { formatDuration } = await import("./log_parser_shared.cjs");

      expect(formatDuration(null)).toBe("");
      expect(formatDuration(undefined)).toBe("");
    });
  });

  describe("formatBashCommand", () => {
    it("should normalize whitespace in commands", async () => {
      const { formatBashCommand } = await import("./log_parser_shared.cjs");

      const command = "echo    hello\n  world\t\tthere";
      const result = formatBashCommand(command);

      expect(result).toBe("echo hello world there");
    });

    it("should escape backticks", async () => {
      const { formatBashCommand } = await import("./log_parser_shared.cjs");

      const command = "echo `date`";
      const result = formatBashCommand(command);

      expect(result).toBe("echo \\`date\\`");
    });

    it("should truncate long commands", async () => {
      const { formatBashCommand } = await import("./log_parser_shared.cjs");

      const longCommand = "a".repeat(400);
      const result = formatBashCommand(longCommand);

      expect(result.length).toBe(303); // 300 chars + "..."
      expect(result.endsWith("...")).toBe(true);
    });

    it("should handle empty and null commands", async () => {
      const { formatBashCommand } = await import("./log_parser_shared.cjs");

      expect(formatBashCommand("")).toBe("");
      expect(formatBashCommand(null)).toBe("");
      expect(formatBashCommand(undefined)).toBe("");
    });

    it("should remove leading and trailing whitespace", async () => {
      const { formatBashCommand } = await import("./log_parser_shared.cjs");

      const command = "   echo hello   ";
      const result = formatBashCommand(command);

      expect(result).toBe("echo hello");
    });

    it("should handle multi-line commands", async () => {
      const { formatBashCommand } = await import("./log_parser_shared.cjs");

      const command = "echo line1\necho line2\necho line3";
      const result = formatBashCommand(command);

      expect(result).toBe("echo line1 echo line2 echo line3");
    });
  });

  describe("truncateString", () => {
    it("should truncate strings longer than max length", async () => {
      const { truncateString } = await import("./log_parser_shared.cjs");

      const longStr = "a".repeat(100);
      const result = truncateString(longStr, 50);

      expect(result.length).toBe(53); // 50 chars + "..."
      expect(result.endsWith("...")).toBe(true);
    });

    it("should not truncate strings at or below max length", async () => {
      const { truncateString } = await import("./log_parser_shared.cjs");

      expect(truncateString("hello", 10)).toBe("hello");
      expect(truncateString("1234567890", 10)).toBe("1234567890");
    });

    it("should handle empty and null strings", async () => {
      const { truncateString } = await import("./log_parser_shared.cjs");

      expect(truncateString("", 10)).toBe("");
      expect(truncateString(null, 10)).toBe("");
      expect(truncateString(undefined, 10)).toBe("");
    });

    it("should handle zero max length", async () => {
      const { truncateString } = await import("./log_parser_shared.cjs");

      const result = truncateString("hello", 0);
      expect(result).toBe("...");
    });
  });

  describe("estimateTokens", () => {
    it("should estimate tokens using 4 chars per token", async () => {
      const { estimateTokens } = await import("./log_parser_shared.cjs");

      expect(estimateTokens("test")).toBe(1); // 4 chars = 1 token
      expect(estimateTokens("hello world")).toBe(3); // 11 chars = 2.75 -> 3 tokens
      expect(estimateTokens("a".repeat(100))).toBe(25); // 100 chars = 25 tokens
    });

    it("should round up partial tokens", async () => {
      const { estimateTokens } = await import("./log_parser_shared.cjs");

      expect(estimateTokens("a")).toBe(1); // 1 char = 0.25 -> 1 token
      expect(estimateTokens("ab")).toBe(1); // 2 chars = 0.5 -> 1 token
      expect(estimateTokens("abc")).toBe(1); // 3 chars = 0.75 -> 1 token
    });

    it("should handle empty and null text", async () => {
      const { estimateTokens } = await import("./log_parser_shared.cjs");

      expect(estimateTokens("")).toBe(0);
      expect(estimateTokens(null)).toBe(0);
      expect(estimateTokens(undefined)).toBe(0);
    });

    it("should handle large text", async () => {
      const { estimateTokens } = await import("./log_parser_shared.cjs");

      const largeText = "a".repeat(10000);
      expect(estimateTokens(largeText)).toBe(2500); // 10000 chars = 2500 tokens
    });
  });

  describe("formatMcpName", () => {
    it("should format MCP tool names", async () => {
      const { formatMcpName } = await import("./log_parser_shared.cjs");

      expect(formatMcpName("mcp__github__search_issues")).toBe("github::search_issues");
      expect(formatMcpName("mcp__playwright__navigate")).toBe("playwright::navigate");
      expect(formatMcpName("mcp__server__tool_name")).toBe("server::tool_name");
    });

    it("should handle tool names with multiple underscores", async () => {
      const { formatMcpName } = await import("./log_parser_shared.cjs");

      expect(formatMcpName("mcp__github__get_pull_request_files")).toBe("github::get_pull_request_files");
    });

    it("should return non-MCP names unchanged", async () => {
      const { formatMcpName } = await import("./log_parser_shared.cjs");

      expect(formatMcpName("Bash")).toBe("Bash");
      expect(formatMcpName("Read")).toBe("Read");
      expect(formatMcpName("regular_tool")).toBe("regular_tool");
    });

    it("should handle malformed MCP names", async () => {
      const { formatMcpName } = await import("./log_parser_shared.cjs");

      expect(formatMcpName("mcp__")).toBe("mcp__");
      expect(formatMcpName("mcp__github")).toBe("mcp__github");
    });
  });

  describe("generateConversationMarkdown", () => {
    it("should generate markdown from log entries", async () => {
      const { generateConversationMarkdown } = await import("./log_parser_shared.cjs");

      const logEntries = [
        {
          type: "system",
          subtype: "init",
          model: "test-model",
        },
        {
          type: "assistant",
          message: {
            content: [
              { type: "text", text: "Let me help with that." },
              { type: "tool_use", id: "tool1", name: "Bash", input: { command: "echo hello" } },
            ],
          },
        },
        {
          type: "user",
          message: {
            content: [{ type: "tool_result", tool_use_id: "tool1", content: "hello", is_error: false }],
          },
        },
      ];

      const formatToolCallback = (content, toolResult) => {
        return `Tool: ${content.name}\n\n`;
      };

      const formatInitCallback = initEntry => {
        return `Model: ${initEntry.model}\n\n`;
      };

      const result = generateConversationMarkdown(logEntries, {
        formatToolCallback,
        formatInitCallback,
      });

      expect(result.markdown).toContain("## 🚀 Initialization");
      expect(result.markdown).toContain("Model: test-model");
      expect(result.markdown).toContain("## 🤖 Reasoning");
      expect(result.markdown).toContain("Let me help with that.");
      expect(result.markdown).toContain("Tool: Bash");
      expect(result.markdown).toContain("## 🤖 Commands and Tools");
      expect(result.commandSummary).toHaveLength(1);
      expect(result.commandSummary[0]).toContain("✅");
      expect(result.commandSummary[0]).toContain("echo hello");
    });

    it("should handle empty log entries", async () => {
      const { generateConversationMarkdown } = await import("./log_parser_shared.cjs");

      const result = generateConversationMarkdown([], {
        formatToolCallback: () => "",
        formatInitCallback: () => "",
      });

      expect(result.markdown).toContain("## 🤖 Reasoning");
      expect(result.markdown).toContain("## 🤖 Commands and Tools");
      expect(result.markdown).toContain("No commands or tools used.");
      expect(result.commandSummary).toHaveLength(0);
    });

    it("should skip internal tools in command summary", async () => {
      const { generateConversationMarkdown } = await import("./log_parser_shared.cjs");

      const logEntries = [
        {
          type: "assistant",
          message: {
            content: [
              { type: "tool_use", id: "tool1", name: "Read", input: { path: "/file.txt" } },
              { type: "tool_use", id: "tool2", name: "Bash", input: { command: "ls" } },
              { type: "tool_use", id: "tool3", name: "Edit", input: { path: "/file.txt" } },
            ],
          },
        },
      ];

      const result = generateConversationMarkdown(logEntries, {
        formatToolCallback: () => "",
        formatInitCallback: () => "",
      });

      // Should only include Bash, not Read or Edit
      expect(result.commandSummary).toHaveLength(1);
      expect(result.commandSummary[0]).toContain("ls");
    });

    it("should format MCP tool names in command summary", async () => {
      const { generateConversationMarkdown } = await import("./log_parser_shared.cjs");

      const logEntries = [
        {
          type: "assistant",
          message: {
            content: [{ type: "tool_use", id: "tool1", name: "mcp__github__search_issues", input: { query: "test" } }],
          },
        },
      ];

      const result = generateConversationMarkdown(logEntries, {
        formatToolCallback: () => "",
        formatInitCallback: () => "",
      });

      expect(result.commandSummary).toHaveLength(1);
      expect(result.commandSummary[0]).toContain("github::search_issues");
    });
  });

  describe("generateInformationSection", () => {
    it("should generate information section with metadata", async () => {
      const { generateInformationSection } = await import("./log_parser_shared.cjs");

      const lastEntry = {
        num_turns: 5,
        duration_ms: 125000,
        total_cost_usd: 0.0123,
        usage: {
          input_tokens: 1000,
          output_tokens: 500,
        },
      };

      const result = generateInformationSection(lastEntry);

      expect(result).toContain("## 📊 Information");
      expect(result).toContain("**Turns:** 5");
      expect(result).toContain("**Duration:** 2m 5s");
      expect(result).toContain("**Total Cost:** $0.0123");
      expect(result).toContain("**Token Usage:**");
      expect(result).toContain("- Input: 1,000");
      expect(result).toContain("- Output: 500");
    });

    it("should handle additional info callback", async () => {
      const { generateInformationSection } = await import("./log_parser_shared.cjs");

      const lastEntry = {
        num_turns: 3,
      };

      const result = generateInformationSection(lastEntry, {
        additionalInfoCallback: () => "**Custom Info:** test\n\n",
      });

      expect(result).toContain("**Turns:** 3");
      expect(result).toContain("**Custom Info:** test");
    });

    it("should handle cache tokens", async () => {
      const { generateInformationSection } = await import("./log_parser_shared.cjs");

      const lastEntry = {
        usage: {
          input_tokens: 1000,
          cache_creation_input_tokens: 500,
          cache_read_input_tokens: 200,
          output_tokens: 300,
        },
      };

      const result = generateInformationSection(lastEntry);

      expect(result).toContain("- Input: 1,000");
      expect(result).toContain("- Cache Creation: 500");
      expect(result).toContain("- Cache Read: 200");
      expect(result).toContain("- Output: 300");
    });

    it("should handle permission denials", async () => {
      const { generateInformationSection } = await import("./log_parser_shared.cjs");

      const lastEntry = {
        permission_denials: ["tool1", "tool2", "tool3"],
      };

      const result = generateInformationSection(lastEntry);

      expect(result).toContain("**Permission Denials:** 3");
    });

    it("should handle empty lastEntry", async () => {
      const { generateInformationSection } = await import("./log_parser_shared.cjs");

      const result = generateInformationSection(null);

      expect(result).toBe("\n## 📊 Information\n\n");
    });
  });
});
