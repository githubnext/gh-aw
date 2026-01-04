// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Setup Threat Detection
 *
 * This module sets up the threat detection analysis by:
 * 1. Checking for existence of artifact files (prompt, agent output, patch)
 * 2. Creating a threat detection prompt from the embedded template
 * 3. Writing the prompt to a file for the AI engine to process
 * 4. Adding the rendered prompt to the workflow summary
 */

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * List all files recursively in a directory
 * @param {string} dirPath - The directory path to list
 * @param {string} [relativeTo] - Optional base path to show relative paths
 * @returns {string[]} Array of file paths
 */
function listFilesRecursively(dirPath, relativeTo) {
  const files = [];
  try {
    if (!fs.existsSync(dirPath)) {
      return files;
    }
    const entries = fs.readdirSync(dirPath, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(dirPath, entry.name);
      if (entry.isDirectory()) {
        files.push(...listFilesRecursively(fullPath, relativeTo));
      } else {
        const displayPath = relativeTo ? path.relative(relativeTo, fullPath) : fullPath;
        files.push(displayPath);
      }
    }
  } catch (error) {
    core.warning("Failed to list files in " + dirPath + ": " + getErrorMessage(error));
  }
  return files;
}

/**
 * Check if file exists and provide helpful error message with directory listing
 * @param {string} filePath - The file path to check
 * @param {string} artifactDir - The artifact directory to list if file not found
 * @param {string} fileDescription - Description of the file (e.g., "Prompt file", "Agent output file")
 * @param {boolean} required - Whether the file is required
 * @returns {boolean} True if file exists (or not required), false otherwise
 */
function checkFileExists(filePath, artifactDir, fileDescription, required) {
  if (fs.existsSync(filePath)) {
    try {
      const stats = fs.statSync(filePath);
      const fileInfo = filePath + " (" + stats.size + " bytes)";
      core.info(fileDescription + " found: " + fileInfo);
      return true;
    } catch (error) {
      core.warning("Failed to stat " + fileDescription.toLowerCase() + ": " + getErrorMessage(error));
      return false;
    }
  } else {
    if (required) {
      core.error("❌ " + fileDescription + " not found at: " + filePath);
      // List all files in artifact directory for debugging
      core.info("📁 Listing all files in artifact directory: " + artifactDir);
      const files = listFilesRecursively(artifactDir, artifactDir);
      if (files.length === 0) {
        core.warning("  No files found in " + artifactDir);
      } else {
        core.info("  Found " + files.length + " file(s):");
        files.forEach(file => core.info("    - " + file));
      }
      core.setFailed("❌ " + fileDescription + " not found at: " + filePath);
      return false;
    } else {
      core.info("No " + fileDescription.toLowerCase() + " found at: " + filePath);
      return true;
    }
  }
}

/**
 * Main entry point for setting up threat detection
 * @param {string} templateContent - The threat detection prompt template
 * @returns {Promise<void>}
 */
async function main(templateContent) {
  // Check if prompt file exists
  // Since agent-artifacts is downloaded to /tmp/gh-aw/threat-detection/agent-artifacts,
  // and the artifact contains files with full paths like /tmp/gh-aw/aw-prompts/prompt.txt,
  // the downloaded file will be at /tmp/gh-aw/threat-detection/agent-artifacts/tmp/gh-aw/aw-prompts/prompt.txt
  const artifactsDir = "/tmp/gh-aw/threat-detection/agent-artifacts";
  const promptPath = path.join(artifactsDir, "tmp/gh-aw/aw-prompts/prompt.txt");
  if (!checkFileExists(promptPath, artifactsDir, "Prompt file", true)) {
    return;
  }

  // Check if agent output file exists
  // Agent output is a separate artifact downloaded to /tmp/gh-aw/threat-detection/agent-output,
  // so it appears directly as /tmp/gh-aw/threat-detection/agent-output/agent_output.json
  const agentOutputDir = "/tmp/gh-aw/threat-detection/agent-output";
  const agentOutputPath = path.join(agentOutputDir, "agent_output.json");
  if (!checkFileExists(agentOutputPath, agentOutputDir, "Agent output file", true)) {
    return;
  }

  // Check if patch file exists
  // Since agent-artifacts is downloaded to /tmp/gh-aw/threat-detection/agent-artifacts,
  // and the artifact contains /tmp/gh-aw/aw.patch,
  // the downloaded file will be at /tmp/gh-aw/threat-detection/agent-artifacts/tmp/gh-aw/aw.patch
  const patchPath = path.join(artifactsDir, "tmp/gh-aw/aw.patch");
  const hasPatch = process.env.HAS_PATCH === "true";
  if (!checkFileExists(patchPath, artifactsDir, "Patch file", hasPatch)) {
    if (hasPatch) {
      return;
    }
  }

  // Get file info for template replacement
  const promptFileInfo = promptPath + " (" + fs.statSync(promptPath).size + " bytes)";
  const agentOutputFileInfo = agentOutputPath + " (" + fs.statSync(agentOutputPath).size + " bytes)";
  let patchFileInfo = "No patch file found";
  if (fs.existsSync(patchPath)) {
    patchFileInfo = patchPath + " (" + fs.statSync(patchPath).size + " bytes)";
  }

  // Create threat detection prompt with embedded template
  let promptContent = templateContent
    .replace(/{WORKFLOW_NAME}/g, process.env.WORKFLOW_NAME || "Unnamed Workflow")
    .replace(/{WORKFLOW_DESCRIPTION}/g, process.env.WORKFLOW_DESCRIPTION || "No description provided")
    .replace(/{WORKFLOW_PROMPT_FILE}/g, promptFileInfo)
    .replace(/{AGENT_OUTPUT_FILE}/g, agentOutputFileInfo)
    .replace(/{AGENT_PATCH_FILE}/g, patchFileInfo);

  // Append custom prompt instructions if provided
  const customPrompt = process.env.CUSTOM_PROMPT;
  if (customPrompt) {
    promptContent += "\n\n## Additional Instructions\n\n" + customPrompt;
  }

  // Write prompt file
  fs.mkdirSync("/tmp/gh-aw/aw-prompts", { recursive: true });
  fs.writeFileSync("/tmp/gh-aw/aw-prompts/prompt.txt", promptContent);
  core.exportVariable("GH_AW_PROMPT", "/tmp/gh-aw/aw-prompts/prompt.txt");

  // Note: creation of /tmp/gh-aw/threat-detection and detection.log is handled by a separate shell step

  // Write rendered prompt to step summary using HTML details/summary
  await core.summary.addRaw("<details>\n<summary>Threat Detection Prompt</summary>\n\n" + "``````markdown\n" + promptContent + "\n" + "``````\n\n</details>\n").write();

  core.info("Threat detection setup completed");
}

module.exports = { main };
