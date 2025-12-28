#!/usr/bin/env node
// @ts-check

/**
 * Demonstration of Enhanced MCP Error Messages
 *
 * This script demonstrates how the enhanced error messages provide
 * actionable guidance when tools are called with missing parameters.
 */

const { generateEnhancedErrorMessage } = require("./mcp_enhanced_errors.cjs");
const tools = require("./safe_outputs_tools.json");

console.log("=".repeat(80));
console.log("Enhanced MCP Error Messages - Demonstration");
console.log("=".repeat(80));
console.log();

// Demonstrate error for add_comment without item_number
const addCommentTool = tools.find(t => t.name === "add_comment");
if (addCommentTool) {
  console.log("Example 1: Missing 'item_number' parameter in add_comment");
  console.log("-".repeat(80));
  const error = generateEnhancedErrorMessage(["item_number"], addCommentTool.name, addCommentTool.inputSchema);
  console.log(error);
  console.log();
}

// Demonstrate error for create_issue without title and body
const createIssueTool = tools.find(t => t.name === "create_issue");
if (createIssueTool) {
  console.log("Example 2: Missing 'title' and 'body' parameters in create_issue");
  console.log("-".repeat(80));
  const error = generateEnhancedErrorMessage(["title", "body"], createIssueTool.name, createIssueTool.inputSchema);
  console.log(error);
  console.log();
}

// Demonstrate error for assign_milestone with missing parameters
const assignMilestoneTool = tools.find(t => t.name === "assign_milestone");
if (assignMilestoneTool) {
  console.log("Example 3: Missing required parameters in assign_milestone");
  console.log("-".repeat(80));
  const error = generateEnhancedErrorMessage(["issue_number", "milestone_number"], assignMilestoneTool.name, assignMilestoneTool.inputSchema);
  console.log(error);
  console.log();
}

console.log("=".repeat(80));
console.log("Benefits of Enhanced Error Messages:");
console.log("  1. Clear indication of which parameters are missing");
console.log("  2. Description of what each parameter represents");
console.log("  3. Example showing correct usage format");
console.log("  4. Helps agents self-correct without human intervention");
console.log("=".repeat(80));
