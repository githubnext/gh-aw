import { describe, it, expect } from "vitest";
import fs from "fs";
import path from "path";

/**
 * Test suite to validate that all safe output JavaScript files
 * use underscores in their type filtering, not dashes.
 *
 * Example:
 *   ✓ item.type === "create_issue"
 *   ✗ item.type === "create_issue"
 */
describe("Safe Output Type Validation", () => {
  // Map of file names to their expected type strings
  const typeFilters = {
    "create_issue.cjs": "create_issue",
    "add_comment.cjs": "add_comment",
    "update_issue.cjs": "update_issue",
    "create_pr_review_comment.cjs": "create_pull_request_review_comment",
    "add_labels.cjs": "add_labels",
    "create_code_scanning_alert.cjs": "create_code_scanning_alert",
    "upload_assets.cjs": "upload_asset",
    "create_discussion.cjs": "create_discussion",
    "push_to_pull_request_branch.cjs": "push_to_pull_request_branch",
    "create_pull_request.cjs": "create_pull_request",
  };

  // Test each file
  Object.entries(typeFilters).forEach(([fileName, expectedType]) => {
    it(`should use underscores in type filter for ${fileName}`, () => {
      const filePath = path.join(process.cwd(), fileName);
      const content = fs.readFileSync(filePath, "utf8");

      // Check that the expected underscore type is present
      const hasUnderscoreType = content.includes(`"${expectedType}"`);
      expect(hasUnderscoreType).toBe(true);

      // Create the dash version of the type
      const dashType = expectedType.replace(/_/g, "-");

      // Check that the dash version is NOT present (except in comments or strings)
      // We need to be careful not to match URLs or other non-type strings
      const dashTypePattern = new RegExp(`item\\.type\\s*===\\s*["']${dashType.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}["']`);
      const hasDashType = dashTypePattern.test(content);

      expect(hasDashType).toBe(false);
    });
  });

  it("should validate schema uses underscores", () => {
    const schemaPath = path.join(process.cwd(), "..", "..", "..", "schemas", "agent-output.json");
    const schemaContent = fs.readFileSync(schemaPath, "utf8");

    const expectedTypes = [
      "create_issue",
      "add_comment",
      "create_pull_request",
      "add_labels",
      "update_issue",
      "push_to_pull_request_branch",
      "create_pull_request_review_comment",
      "create_discussion",
      "missing_tool",
      "create_code_scanning_alert",
    ];

    // Check that each expected type is defined in the schema
    expectedTypes.forEach(type => {
      const hasType = schemaContent.includes(`"const": "${type}"`);
      expect(hasType).toBe(true);

      // Also verify that the dash version is NOT present
      const dashType = type.replace(/_/g, "-");
      const hasDashType = schemaContent.includes(`"const": "${dashType}"`);
      expect(hasDashType).toBe(false);
    });
  });

  it("should validate MCP server normalizes types to underscores", () => {
    const mcpServerPath = path.join(process.cwd(), "safe_outputs_mcp_server.cjs");
    const content = fs.readFileSync(mcpServerPath, "utf8");

    // Check that the MCP server normalizes type fields to underscores
    const hasNormalization = content.includes('entry.type = entry.type.replace(/-/g, "_")');
    expect(hasNormalization).toBe(true);

    // Check that all tool names use underscores
    const toolNames = [
      "create_issue",
      "create_discussion",
      "add_comment",
      "create_pull_request",
      "create_pull_request_review_comment",
      "create_code_scanning_alert",
      "add_labels",
      "update_issue",
      "push_to_pull_request_branch",
      "upload_asset",
    ];

    toolNames.forEach(toolName => {
      // Check for tool name definition (e.g., name: "create_issue")
      const hasToolName = content.includes(`name: "${toolName}"`);
      expect(hasToolName).toBe(true);
    });
  });
});
