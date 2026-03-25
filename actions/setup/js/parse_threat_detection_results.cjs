// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Parse Threat Detection Results
 *
 * This module parses the threat detection results from the agent output file
 * and determines whether any security threats were detected (prompt injection,
 * secret leak, malicious patch). It sets the appropriate output and fails the
 * workflow if threats are detected.
 */

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { listFilesRecursively } = require("./file_helpers.cjs");
const { AGENT_OUTPUT_FILENAME } = require("./constants.cjs");
const { ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");

/**
 * Parse threat detection result from file content.
 * Scans lines for a `THREAT_DETECTION_RESULT:` prefix and merges the JSON
 * payload into the default verdict.
 * @param {string} content - The raw file content to parse
 * @returns {{ verdict: { prompt_injection: boolean, secret_leak: boolean, malicious_patch: boolean, reasons: string[] }, found: boolean }}
 */
function parseThreatDetectionResult(content) {
  const verdict = { prompt_injection: false, secret_leak: false, malicious_patch: false, reasons: [] };
  const lines = content.split("\n");
  for (const line of lines) {
    const trimmedLine = line.trim();
    if (trimmedLine.startsWith("THREAT_DETECTION_RESULT:")) {
      const jsonPart = trimmedLine.substring("THREAT_DETECTION_RESULT:".length);
      core.info("🔍 Found THREAT_DETECTION_RESULT line, parsing JSON payload");
      return { verdict: { ...verdict, ...JSON.parse(jsonPart) }, found: true };
    }
  }
  core.warning("⚠️ No THREAT_DETECTION_RESULT line found in agent output");
  return { verdict, found: false };
}

/**
 * Main entry point for parsing threat detection results
 * @returns {Promise<void>}
 */
async function main() {
  // Parse threat detection results
  let verdict = { prompt_injection: false, secret_leak: false, malicious_patch: false, reasons: [] };

  try {
    // Agent output artifact is downloaded to /tmp/gh-aw/threat-detection/
    // GitHub Actions places single-file artifacts directly in the target directory
    const threatDetectionDir = "/tmp/gh-aw/threat-detection";
    const outputPath = path.join(threatDetectionDir, AGENT_OUTPUT_FILENAME);
    if (!fs.existsSync(outputPath)) {
      core.error("❌ Agent output file not found at: " + outputPath);
      // List all files in artifact directory for debugging
      core.info("📁 Listing all files in artifact directory: " + threatDetectionDir);
      const files = listFilesRecursively(threatDetectionDir, threatDetectionDir);
      if (files.length === 0) {
        core.warning("  No files found in " + threatDetectionDir);
      } else {
        core.info("  Found " + files.length + " file(s):");
        files.forEach(file => core.info("    - " + file));
      }
      core.setFailed(`${ERR_SYSTEM}: ❌ Agent output file not found at: ${outputPath}`);
      return;
    }
    const outputContent = fs.readFileSync(outputPath, "utf8");
    const { verdict: parsed, found } = parseThreatDetectionResult(outputContent);
    if (!found) {
      core.setOutput("success", "false");
      core.setFailed(`${ERR_VALIDATION}: ❌ No THREAT_DETECTION_RESULT line found in agent output — threat detection result is missing`);
      return;
    }
    verdict = parsed;
  } catch (error) {
    core.warning("Failed to parse threat detection results: " + getErrorMessage(error));
  }

  core.info("Threat detection verdict: " + JSON.stringify(verdict));

  // Fail if threats detected
  if (verdict.prompt_injection || verdict.secret_leak || verdict.malicious_patch) {
    const threats = [];
    if (verdict.prompt_injection) threats.push("prompt injection");
    if (verdict.secret_leak) threats.push("secret leak");
    if (verdict.malicious_patch) threats.push("malicious patch");

    const reasonsText = verdict.reasons && verdict.reasons.length > 0 ? "\nReasons: " + verdict.reasons.join("; ") : "";

    // Set success output to false before failing
    core.setOutput("success", "false");
    core.setFailed(`${ERR_VALIDATION}: ❌ Security threats detected: ${threats.join(", ")}${reasonsText}`);
  } else {
    core.info("✅ No security threats detected. Safe outputs may proceed.");
    // Set success output to true when no threats detected
    core.setOutput("success", "true");
  }
}

module.exports = { main, parseThreatDetectionResult };
