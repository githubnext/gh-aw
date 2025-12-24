// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Sanitizes content for safe output in GitHub Actions
 * @param {string} content - The content to sanitize
 * @returns {string} The sanitized content
 */
const { sanitizeContent, writeRedactedDomainsLog } = require("./sanitize_content.cjs");

async function main() {
  const fs = require("fs");
  const outputFile = process.env.GH_AW_SAFE_OUTPUTS;
  if (!outputFile) {
    core.info("GH_AW_SAFE_OUTPUTS not set, no output to collect");
    core.setOutput("output", "");
    return;
  }

  if (!fs.existsSync(outputFile)) {
    core.info(`Output file does not exist: ${outputFile}`);
    core.setOutput("output", "");
    return;
  }

  const outputContent = fs.readFileSync(outputFile, "utf8");
  if (outputContent.trim() === "") {
    core.info("Output file is empty");
    core.setOutput("output", "");
  } else {
    const sanitizedContent = sanitizeContent(outputContent);
    core.info(`Collected agentic output (sanitized): ${sanitizedContent.substring(0, 200)}${sanitizedContent.length > 200 ? "..." : ""}`);
    core.setOutput("output", sanitizedContent);
  }

  // Write redacted URL domains to log file if any were collected
  const logPath = writeRedactedDomainsLog();
  if (logPath) {
    core.info(`Redacted URL domains written to: ${logPath}`);
  }
}

module.exports = { main };
