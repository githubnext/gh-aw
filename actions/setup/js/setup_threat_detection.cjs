// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Setup Threat Detection
 *
 * This module sets up the threat detection analysis by:
 * 1. Checking for existence of artifact files (prompt, agent output, patch)
 * 2. Reading the threat detection prompt template from file
 * 3. Creating a threat detection prompt from the template
 * 4. Writing the prompt to a file for the AI engine to process
 * 5. Adding the rendered prompt to the workflow summary
 */

const fs = require("fs");
const path = require("path");
const { checkFileExists } = require("./file_helpers.cjs");
const { AGENT_OUTPUT_FILENAME } = require("./constants.cjs");

/**
 * Main entry point for setting up threat detection
 * @returns {Promise<void>}
 */
async function main() {
  // Read the threat detection template from file
  // At runtime, markdown files are copied to /opt/gh-aw/prompts/ by the setup action
  const templatePath = "/opt/gh-aw/prompts/threat_detection.md";
  if (!fs.existsSync(templatePath)) {
    core.setFailed(`Threat detection template not found at: ${templatePath}`);
    return;
  }
  const templateContent = fs.readFileSync(templatePath, "utf-8");
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

  // Check if patch file(s) exist
  // Patches are now named aw-{branch}.patch (one per branch)
  // The agent-artifacts artifact is downloaded to /tmp/gh-aw/threat-detection/
  const hasPatch = process.env.HAS_PATCH === "true";
  const patchFiles = [];
  try {
    const dirEntries = fs.readdirSync(threatDetectionDir);
    for (const entry of dirEntries) {
      if (/^aw-.+\.patch$/.test(entry)) {
        patchFiles.push(path.join(threatDetectionDir, entry));
      }
    }
  } catch {
    // Directory may not exist or be readable
  }

  if (patchFiles.length === 0 && hasPatch) {
    core.setFailed(`Patch file(s) expected but not found in: ${threatDetectionDir}`);
    return;
  }

  // Get file info for template replacement
  const promptFileInfo = promptPath + " (" + fs.statSync(promptPath).size + " bytes)";
  const agentOutputFileInfo = agentOutputPath + " (" + fs.statSync(agentOutputPath).size + " bytes)";

  // Build patch file info for template replacement
  let patchFileInfo = "No patch file found";
  if (patchFiles.length > 0) {
    patchFileInfo = patchFiles
      .map(p => {
        const size = fs.existsSync(p) ? fs.statSync(p).size : 0;
        return `${p} (${size} bytes)`;
      })
      .join("\n");
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
