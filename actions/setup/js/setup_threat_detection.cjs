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
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Main entry point for setting up threat detection
 * @param {string} templateContent - The threat detection prompt template
 * @returns {Promise<void>}
 */
async function main(templateContent) {
  // Check if prompt file exists
  // Since agent-artifacts is downloaded to /tmp/gh-aw/artifacts,
  // and the artifact contains files with full paths like /tmp/gh-aw/aw-prompts/prompt.txt,
  // the downloaded file will be at /tmp/gh-aw/artifacts/tmp/gh-aw/aw-prompts/prompt.txt
  const promptPath = "/tmp/gh-aw/artifacts/tmp/gh-aw/aw-prompts/prompt.txt";
  let promptFileInfo = "No prompt file found";
  if (fs.existsSync(promptPath)) {
    try {
      const stats = fs.statSync(promptPath);
      promptFileInfo = promptPath + " (" + stats.size + " bytes)";
      core.info("Prompt file found: " + promptFileInfo);
    } catch (error) {
      core.warning("Failed to stat prompt file: " + getErrorMessage(error));
    }
  } else {
    core.setFailed("❌ Prompt file not found at: " + promptPath);
    return;
  }

  // Check if agent output file exists
  // Agent output is still a separate artifact downloaded to /tmp/gh-aw/artifacts,
  // so it appears directly as /tmp/gh-aw/artifacts/agent_output.json
  const agentOutputPath = "/tmp/gh-aw/artifacts/agent_output.json";
  let agentOutputFileInfo = "No agent output file found";
  if (fs.existsSync(agentOutputPath)) {
    try {
      const stats = fs.statSync(agentOutputPath);
      agentOutputFileInfo = agentOutputPath + " (" + stats.size + " bytes)";
      core.info("Agent output file found: " + agentOutputFileInfo);
    } catch (error) {
      core.warning("Failed to stat agent output file: " + getErrorMessage(error));
    }
  } else {
    core.setFailed("❌ Agent output file not found at: " + agentOutputPath);
    return;
  }

  // Check if patch file exists
  // Since agent-artifacts is downloaded to /tmp/gh-aw/artifacts,
  // and the artifact contains /tmp/gh-aw/aw.patch,
  // the downloaded file will be at /tmp/gh-aw/artifacts/tmp/gh-aw/aw.patch
  const patchPath = "/tmp/gh-aw/artifacts/tmp/gh-aw/aw.patch";
  let patchFileInfo = "No patch file found";
  const hasPatch = process.env.HAS_PATCH === "true";
  if (fs.existsSync(patchPath)) {
    try {
      const stats = fs.statSync(patchPath);
      patchFileInfo = patchPath + " (" + stats.size + " bytes)";
      core.info("Patch file found: " + patchFileInfo);
    } catch (error) {
      core.warning("Failed to stat patch file: " + getErrorMessage(error));
    }
  } else {
    if (hasPatch) {
      core.setFailed("❌ Patch file not found at: " + patchPath + " but was expected (has_patch=true)");
      return;
    } else {
      core.info("No patch file found at: " + patchPath);
    }
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
