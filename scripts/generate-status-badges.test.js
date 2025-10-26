#!/usr/bin/env node

/**
 * Test for Status Badges Generator
 *
 * Validates that the status badges generator correctly:
 * - Extracts workflow information from lock files
 * - Extracts engine types from markdown files
 * - Generates a properly formatted table
 * - Links to workflow markdown files
 */

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Paths
const OUTPUT_PATH = path.join(__dirname, "../docs/src/content/docs/status.mdx");

/**
 * Test helper to check if output contains expected content
 */
function assertContains(content, expected, testName) {
  if (!content.includes(expected)) {
    console.error(`❌ FAIL: ${testName}`);
    console.error(`   Expected to find: "${expected}"`);
    return false;
  }
  console.log(`✓ PASS: ${testName}`);
  return true;
}

/**
 * Test helper to check if output does NOT contain unexpected content
 */
function assertNotContains(content, unexpected, testName) {
  if (content.includes(unexpected)) {
    console.error(`❌ FAIL: ${testName}`);
    console.error(`   Expected NOT to find: "${unexpected}"`);
    return false;
  }
  console.log(`✓ PASS: ${testName}`);
  return true;
}

/**
 * Test helper to count occurrences of a pattern
 */
function countOccurrences(content, pattern) {
  const matches = content.match(new RegExp(pattern, "g"));
  return matches ? matches.length : 0;
}

// Run the status badges generator
console.log("Running status badges generator...");
import("./generate-status-badges.js");

// Wait a bit for the file to be written
await new Promise(resolve => setTimeout(resolve, 500));

// Read the generated output
const output = fs.readFileSync(OUTPUT_PATH, "utf-8");

// Test suite
let allPassed = true;

console.log("\nRunning tests...\n");

// Test 1: Table format
allPassed &= assertContains(output, "| Workflow | Agent | Status | Workflow Link |", "Table header is present with correct columns");

allPassed &= assertContains(output, "|----------|-------|--------|---------------|", "Table separator is present");

// Test 2: Engine detection (copilot)
allPassed &= assertContains(output, "| copilot |", "Copilot engine detected in at least one workflow");

// Test 3: Engine detection (claude)
allPassed &= assertContains(output, "| claude |", "Claude engine detected in at least one workflow");

// Test 4: Engine detection (codex)
allPassed &= assertContains(output, "| codex |", "Codex engine detected in at least one workflow");

// Test 5: Workflow links are present
allPassed &= assertContains(output, ".github/workflows/", "Workflow links to .github/workflows directory");

allPassed &= assertContains(output, ".md)", "Workflow links point to .md files");

// Test 6: Status badges are present
allPassed &= assertContains(output, "badge.svg", "Status badges are present");

allPassed &= assertContains(output, "https://github.com/githubnext/gh-aw/actions/workflows/", "Status badges link to workflow runs");

// Test 7: No "unknown" engine values
allPassed &= assertNotContains(output, "| unknown |", "No workflows with unknown engine (should default to copilot)");

// Test 8: Frontmatter is correct
allPassed &= assertContains(output, "title: Workflow Status", "Frontmatter title is present");

allPassed &= assertContains(
  output,
  "description: Status badges for all GitHub Actions workflows in the repository.",
  "Frontmatter description is present"
);

// Test 9: Introduction text is present
allPassed &= assertContains(
  output,
  "This page shows the current status of all agentic workflows in the repository.",
  "Introduction text is present"
);

// Test 10: Note section is present
allPassed &= assertContains(output, ":::note", "Note section is present");

allPassed &= assertContains(output, "Click on a workflow link to view the source markdown file.", "Note mentions workflow links");

// Test 11: Verify table rows match workflow count
const tableRowCount = countOccurrences(output, "\\| \\[!\\[");
console.log(`Found ${tableRowCount} table rows with workflows`);
if (tableRowCount >= 50) {
  // We expect at least 50 workflows
  console.log("✓ PASS: Table contains workflow rows");
} else {
  console.error(`❌ FAIL: Table should contain at least 50 workflow rows, found ${tableRowCount}`);
  allPassed = false;
}

// Test 12: Verify no CardGrid remnants
allPassed &= assertNotContains(output, "<CardGrid>", "No CardGrid component (should be table now)");

allPassed &= assertNotContains(output, "<Card>", "No Card component (should be table now)");

// Summary
console.log("\n" + "=".repeat(50));
if (allPassed) {
  console.log("✅ All tests passed!");
  process.exit(0);
} else {
  console.log("❌ Some tests failed!");
  process.exit(1);
}
