// @ts-check
/// <reference types="@actions/github-script" />

const { generateGatewayLogSummary } = require("./parse_mcp_gateway_log.cjs");

describe("parse_mcp_gateway_log", () => {
  describe("generateGatewayLogSummary", () => {
    test("generates summary with details and code fence", () => {
      const logContent = "Test log line 1\nTest log line 2\nTest log line 3";

      const summary = generateGatewayLogSummary(logContent);

      expect(summary).toContain("<details>");
      expect(summary).toContain("<summary>MCP Gateway Log</summary>");
      expect(summary).toContain("```");
      expect(summary).toContain("Test log line 1");
      expect(summary).toContain("Test log line 2");
      expect(summary).toContain("Test log line 3");
      expect(summary).toContain("</details>");
    });

    test("handles empty log content", () => {
      const logContent = "";

      const summary = generateGatewayLogSummary(logContent);

      expect(summary).toContain("<details>");
      expect(summary).toContain("```");
      expect(summary).toContain("</details>");
    });

    test("trims whitespace from log content", () => {
      const logContent = "\n\n  Test log with whitespace  \n\n";

      const summary = generateGatewayLogSummary(logContent);

      expect(summary).toContain("Test log with whitespace");
      expect(summary).not.toContain("\n\n  Test log");
    });

    test("preserves internal line breaks", () => {
      const logContent = "Line 1\nLine 2\nLine 3";

      const summary = generateGatewayLogSummary(logContent);

      const lines = summary.split("\n");
      const codeBlockStart = lines.findIndex((line) => line === "```");
      const codeBlockEnd = lines.findIndex(
        (line, index) => index > codeBlockStart && line === "```",
      );

      expect(codeBlockEnd - codeBlockStart).toBe(4); // Start + 3 lines + End
    });
  });
});
