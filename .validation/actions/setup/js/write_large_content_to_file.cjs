// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

const { generateCompactSchema } = require("./generate_compact_schema.cjs");

/**
 * Writes large content to a file and returns metadata
 * @param {string} content - The content to write
 * @returns {Object} Object with filename and description
 */
function writeLargeContentToFile(content) {
  const logsDir = "/tmp/gh-aw/safeoutputs";

  // Ensure directory exists
  if (!fs.existsSync(logsDir)) {
    fs.mkdirSync(logsDir, { recursive: true });
  }

  // Generate SHA256 hash of content
  const hash = crypto.createHash("sha256").update(content).digest("hex");

  // MCP tools return JSON, so always use .json extension
  const filename = `${hash}.json`;
  const filepath = path.join(logsDir, filename);

  // Write content to file
  fs.writeFileSync(filepath, content, "utf8");

  // Generate compact schema description for jq/agent
  const description = generateCompactSchema(content);

  return {
    filename: filename,
    description: description,
  };
}

module.exports = {
  writeLargeContentToFile,
};
