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

/**
 * Main entry point for setting up threat detection
 * @param {string} templateContent - The threat detection prompt template
 * @returns {Promise<void>}
 */
async function main(templateContent) {
  // Check if prompt file exists
  const promptPath = "/tmp/gh-aw/threat-detection/prompt.txt";
  let promptFileInfo = "No prompt file found";
  if (fs.existsSync(promptPath)) {
    try {
      const stats = fs.statSync(promptPath);
      promptFileInfo = promptPath + " (" + stats.size + " bytes)";
      core.info("Prompt file found: " + promptFileInfo);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      core.warning("Failed to stat prompt file: " + message);
    }
  } else {
    core.info("No prompt file found at: " + promptPath);
  }

  // Check if agent output file exists
  const agentOutputPath = "/tmp/gh-aw/threat-detection/agent_output.json";
  let agentOutputFileInfo = "No agent output file found";
  if (fs.existsSync(agentOutputPath)) {
    try {
      const stats = fs.statSync(agentOutputPath);
      agentOutputFileInfo = agentOutputPath + " (" + stats.size + " bytes)";
      core.info("Agent output file found: " + agentOutputFileInfo);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      core.warning("Failed to stat agent output file: " + message);
    }
  } else {
    core.info("No agent output file found at: " + agentOutputPath);
  }

  // Check if patch file exists
  const patchPath = "/tmp/gh-aw/threat-detection/aw.patch";
  let patchFileInfo = "No patch file found";
  if (fs.existsSync(patchPath)) {
    try {
      const stats = fs.statSync(patchPath);
      patchFileInfo = patchPath + " (" + stats.size + " bytes)";
      core.info("Patch file found: " + patchFileInfo);
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      core.warning("Failed to stat patch file: " + message);
    }
  } else {
    core.info("No patch file found at: " + patchPath);
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
