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
const { checkFileExists } = require("./file_helpers.cjs");
const { AGENT_OUTPUT_FILENAME } = require("./constants.cjs");

/**
 * Main entry point for setting up threat detection
 * @param {string} templateContent - The threat detection prompt template
 * @returns {Promise<void>}
 */
async function main(templateContent) {
  // Check if prompt file exists
  // The agent-artifacts artifact is downloaded to /tmp/gh-aw/threat-detection/
  // GitHub Actions preserves the directory structure from the uploaded artifact
  // (stripping the common /tmp/gh-aw/ prefix from the uploaded paths)
  // So /tmp/gh-aw/aw-prompts/prompt.txt becomes /tmp/gh-aw/threat-detection/aw-prompts/prompt.txt
  const threatDetectionDir = "/tmp/gh-aw/threat-detection";
  const promptPath = path.join(threatDetectionDir, "aw-prompts/prompt.txt");
  if (!checkFileExists(promptPath, threatDetectionDir, "Prompt file", true)) {
    return;
  }

  // Check if agent output file exists
  // The agent-output artifact is also downloaded to /tmp/gh-aw/threat-detection/
  // The artifact contains /tmp/gh-aw/agent_output.json which becomes /tmp/gh-aw/threat-detection/agent_output.json
  const agentOutputPath = path.join(threatDetectionDir, AGENT_OUTPUT_FILENAME);
  if (!checkFileExists(agentOutputPath, threatDetectionDir, "Agent output file", true)) {
    return;
  }

  // Check if patch file exists
  // The patch file is part of the agent-artifacts artifact
  // So /tmp/gh-aw/aw.patch becomes /tmp/gh-aw/threat-detection/aw.patch
  const patchPath = path.join(threatDetectionDir, "aw.patch");
  const hasPatch = process.env.HAS_PATCH === "true";
  if (!checkFileExists(patchPath, threatDetectionDir, "Patch file", hasPatch)) {
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
