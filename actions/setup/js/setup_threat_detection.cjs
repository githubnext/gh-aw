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

/**
 * Main entry point for setting up threat detection
 * @param {string} templateContent - The threat detection prompt template
 * @returns {Promise<void>}
 */
async function main(templateContent) {
  // Check if prompt file exists
  // Since agent-artifacts is downloaded to /tmp/gh-aw/threat-detection/agent-artifacts,
  // and the artifact contains files like aw-prompts/prompt.txt (GitHub Actions strips the common /tmp/gh-aw/ parent),
  // the downloaded file will be at /tmp/gh-aw/threat-detection/agent-artifacts/aw-prompts/prompt.txt
  const artifactsDir = "/tmp/gh-aw/threat-detection/agent-artifacts";
  const promptPath = path.join(artifactsDir, "aw-prompts/prompt.txt");
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
  // and the artifact contains aw.patch (GitHub Actions strips the common /tmp/gh-aw/ parent),
  // the downloaded file will be at /tmp/gh-aw/threat-detection/agent-artifacts/aw.patch
  const patchPath = path.join(artifactsDir, "aw.patch");
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
