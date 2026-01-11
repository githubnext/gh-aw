// @ts-check
/// <reference types="@actions/github-script" />

const { generateGatewayLogSummary } = require("./parse_mcp_gateway_log.cjs");

describe("parse_mcp_gateway_log", () => {
  // Note: The main() function now checks for gateway.md first before falling back to log files.
  // If gateway.md exists, its content is written directly to the step summary.
  // These tests focus on the fallback generateGatewayLogSummary function used when gateway.md is not present.

  describe("generateGatewayLogSummary", () => {
    test("generates summary with both gateway.log and stderr.log", () => {
      const gatewayLogContent = "Gateway started\nServer listening on port 8080";
      const stderrLogContent = "Debug: connection accepted\nDebug: request processed";

      const summary = generateGatewayLogSummary(gatewayLogContent, stderrLogContent);

      // Check gateway.log section
      expect(summary).toContain("<summary>MCP Gateway Log (gateway.log)</summary>");
      expect(summary).toContain("Gateway started");
      expect(summary).toContain("Server listening on port 8080");

      // Check stderr.log section
      expect(summary).toContain("<summary>MCP Gateway Log (stderr.log)</summary>");
      expect(summary).toContain("Debug: connection accepted");
      expect(summary).toContain("Debug: request processed");

      // Check structure
      expect(summary).toContain("<details>");
      expect(summary).toContain("```");
      expect(summary).toContain("</details>");
    });

    test("generates summary with only gateway.log content", () => {
      const gatewayLogContent = "Gateway started\nServer ready";
      const stderrLogContent = "";

      const summary = generateGatewayLogSummary(gatewayLogContent, stderrLogContent);

      expect(summary).toContain("<summary>MCP Gateway Log (gateway.log)</summary>");
      expect(summary).toContain("Gateway started");
      expect(summary).not.toContain("<summary>MCP Gateway Log (stderr.log)</summary>");
    });

    test("generates summary with only stderr.log content", () => {
      const gatewayLogContent = "";
      const stderrLogContent = "Error: connection failed\nRetrying...";

      const summary = generateGatewayLogSummary(gatewayLogContent, stderrLogContent);

      expect(summary).not.toContain("<summary>MCP Gateway Log (gateway.log)</summary>");
      expect(summary).toContain("<summary>MCP Gateway Log (stderr.log)</summary>");
      expect(summary).toContain("Error: connection failed");
    });

    test("handles empty log content for both files", () => {
      const gatewayLogContent = "";
      const stderrLogContent = "";

      const summary = generateGatewayLogSummary(gatewayLogContent, stderrLogContent);

      expect(summary).toBe("");
    });

    test("trims whitespace from log content", () => {
      const gatewayLogContent = "\n\n  Gateway log with whitespace  \n\n";
      const stderrLogContent = "\n\n  Stderr log with whitespace  \n\n";

      const summary = generateGatewayLogSummary(gatewayLogContent, stderrLogContent);

      expect(summary).toContain("Gateway log with whitespace");
      expect(summary).toContain("Stderr log with whitespace");
      expect(summary).not.toContain("\n\n  Gateway log");
      expect(summary).not.toContain("\n\n  Stderr log");
    });

    test("preserves internal line breaks", () => {
      const gatewayLogContent = "Line 1\nLine 2\nLine 3";
      const stderrLogContent = "Error 1\nError 2";

      const summary = generateGatewayLogSummary(gatewayLogContent, stderrLogContent);

      const lines = summary.split("\n");

      // Find gateway.log code block - look for summary line with gateway.log
      const gatewaySummaryIndex = lines.findIndex(line => line.includes("gateway.log"));
      expect(gatewaySummaryIndex).toBeGreaterThan(-1);

      // Find the code block start after the gateway summary
      const gatewayCodeBlockIndex = lines.findIndex((line, index) => index > gatewaySummaryIndex && line === "```");
      expect(gatewayCodeBlockIndex).toBeGreaterThan(-1);

      // Find stderr.log code block - look for summary line with stderr.log
      const stderrSummaryIndex = lines.findIndex(line => line.includes("stderr.log"));
      expect(stderrSummaryIndex).toBeGreaterThan(-1);

      // Find the code block start after the stderr summary
      const stderrCodeBlockIndex = lines.findIndex((line, index) => index > stderrSummaryIndex && line === "```");
      expect(stderrCodeBlockIndex).toBeGreaterThan(-1);

      // Verify both sections exist and contain content
      expect(summary).toContain("Line 1");
      expect(summary).toContain("Line 2");
      expect(summary).toContain("Line 3");
      expect(summary).toContain("Error 1");
      expect(summary).toContain("Error 2");
    });
  });
});
