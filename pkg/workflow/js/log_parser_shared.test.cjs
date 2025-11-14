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
});
