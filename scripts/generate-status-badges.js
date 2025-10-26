#!/usr/bin/env node

/**
 * Status Badges Generator
 *
 * Generates a markdown documentation page with GitHub Actions status badges
 * for all workflows in the repository (only from .lock.yml files).
 * Displays workflows in a table with columns for name, agent, status, and workflow link.
 *
 * Usage:
 *   node scripts/generate-status-badges.js
 */

import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Paths
const WORKFLOWS_DIR = path.join(__dirname, "../.github/workflows");
const OUTPUT_PATH = path.join(__dirname, "../docs/src/content/docs/status.mdx");

// Repository owner and name
const REPO_OWNER = "githubnext";
const REPO_NAME = "gh-aw";

/**
 * Extract workflow name and filename from a lock file
 * Uses simple regex parsing instead of YAML parser
 */
function extractWorkflowInfo(filePath) {
  try {
    const content = fs.readFileSync(filePath, "utf-8");

    // Match the name field in YAML format
    // Handles both quoted and unquoted values:
    // name: "My Workflow"
    // name: 'My Workflow'
    // name: My Workflow
    const nameMatch = content.match(/^name:\s*["']?([^"'\n]+?)["']?\s*$/m);

    if (!nameMatch) {
      return null;
    }

    const workflowName = nameMatch[1].trim();
    const filename = path.basename(filePath);

    return {
      name: workflowName,
      filename: filename,
      badgeUrl: `https://github.com/${REPO_OWNER}/${REPO_NAME}/actions/workflows/${filename}/badge.svg`,
      workflowUrl: `https://github.com/${REPO_OWNER}/${REPO_NAME}/actions/workflows/${filename}`,
    };
  } catch (error) {
    console.error(`Error parsing ${filePath}:`, error.message);
    return null;
  }
}

/**
 * Extract engine type from a markdown workflow file
 * Returns 'copilot', 'claude', 'codex', 'custom', or 'copilot' (default)
 */
function extractEngineFromMarkdown(mdFilePath) {
  try {
    if (!fs.existsSync(mdFilePath)) {
      return "copilot"; // Default engine
    }

    const content = fs.readFileSync(mdFilePath, "utf-8");

    // Look for engine field in frontmatter
    // Handles both simple string format and object format:
    // engine: copilot
    // engine: "claude"
    // engine:
    //   id: codex
    const engineStringMatch = content.match(/^engine:\s*["']?(\w+)["']?\s*$/m);
    if (engineStringMatch) {
      return engineStringMatch[1].toLowerCase();
    }

    // Check for object format with 'id' field
    const engineObjectMatch = content.match(/^engine:\s*\n\s+id:\s*["']?(\w+)["']?\s*$/m);
    if (engineObjectMatch) {
      return engineObjectMatch[1].toLowerCase();
    }

    return "copilot"; // Default engine
  } catch (error) {
    console.error(`Error extracting engine from ${mdFilePath}:`, error.message);
    return "copilot"; // Default engine
  }
}

/**
 * Generate the markdown documentation
 */
function generateMarkdown(workflows) {
  const lines = [];

  // Frontmatter
  lines.push("---");
  lines.push("title: Workflow Status");
  lines.push("description: Status badges for all GitHub Actions workflows in the repository.");
  lines.push("sidebar:");
  lines.push("  order: 1000");
  lines.push("---");
  lines.push("");

  // Introduction
  lines.push("This page shows the current status of all agentic workflows in the repository.");
  lines.push("");

  // Sort workflows alphabetically by name
  workflows.sort((a, b) => a.name.localeCompare(b.name));

  // Generate table header
  lines.push("| Workflow | Agent | Status | Workflow Link |");
  lines.push("|----------|-------|--------|---------------|");

  // Generate table rows
  for (const workflow of workflows) {
    const agent = workflow.engine || "copilot";
    const statusBadge = `[![${workflow.name}](${workflow.badgeUrl})](${workflow.workflowUrl})`;
    const workflowLink = workflow.mdFilename
      ? `[${workflow.mdFilename}](https://github.com/${REPO_OWNER}/${REPO_NAME}/blob/main/.github/workflows/${workflow.mdFilename})`
      : "-";

    lines.push(`| ${workflow.name} | ${agent} | ${statusBadge} | ${workflowLink} |`);
  }

  lines.push("");
  lines.push(":::note");
  lines.push(
    "Status badges update automatically based on the latest workflow runs. Click on a badge to view the workflow details and run history. Click on a workflow link to view the source markdown file."
  );
  lines.push(":::");
  lines.push("");

  return lines.join("\n");
}

// Main execution
console.log("Generating status badges documentation...");

// Read all .lock.yml files
const lockFiles = fs
  .readdirSync(WORKFLOWS_DIR)
  .filter(file => file.endsWith(".lock.yml"))
  .map(file => path.join(WORKFLOWS_DIR, file));

console.log(`Found ${lockFiles.length} lock files`);

// Extract workflow information and match with markdown files
const workflows = lockFiles
  .map(lockFilePath => {
    const workflowInfo = extractWorkflowInfo(lockFilePath);
    if (!workflowInfo) {
      return null;
    }

    // Try to find corresponding .md file
    // Convert "workflow-name.lock.yml" to "workflow-name.md"
    const mdFilename = workflowInfo.filename.replace(".lock.yml", ".md");
    const mdFilePath = path.join(WORKFLOWS_DIR, mdFilename);

    // Extract engine from markdown file
    const engine = extractEngineFromMarkdown(mdFilePath);

    return {
      ...workflowInfo,
      engine: engine,
      mdFilename: fs.existsSync(mdFilePath) ? mdFilename : null,
    };
  })
  .filter(info => info !== null);

console.log(`Extracted ${workflows.length} workflows with valid names`);

// Generate the markdown
const markdown = generateMarkdown(workflows);

// Ensure output directory exists
const outputDir = path.dirname(OUTPUT_PATH);
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

// Write the output
fs.writeFileSync(OUTPUT_PATH, markdown, "utf-8");
console.log(`✓ Generated status badges documentation: ${OUTPUT_PATH}`);
console.log(`✓ Total workflows: ${workflows.length}`);
