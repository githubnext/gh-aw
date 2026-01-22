// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Redacts secrets from files in /tmp/gh-aw and /opt/gh-aw directories before uploading artifacts
 * This script processes all .txt, .json, .log, .md, .mdx, .yml, .jsonl files under /tmp/gh-aw and /opt/gh-aw
 * and redacts any strings matching the actual secret values provided via environment variables.
 */
const fs = require("fs");
const path = require("path");
/**
 * Recursively finds all files matching the specified extensions
 * @param {string} dir - Directory to search
 * @param {string[]} extensions - File extensions to match (e.g., ['.txt', '.json', '.log'])
 * @returns {string[]} Array of file paths
 */
function findFiles(dir, extensions) {
  const results = [];
  try {
    if (!fs.existsSync(dir)) {
      return results;
    }
    const entries = fs.readdirSync(dir, { withFileTypes: true });
    for (const entry of entries) {
      const fullPath = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        // Recursively search subdirectories
        results.push(...findFiles(fullPath, extensions));
      } else if (entry.isFile()) {
        // Check if file has one of the target extensions
        const ext = path.extname(entry.name).toLowerCase();
        if (extensions.includes(ext)) {
          results.push(fullPath);
        }
      }
    }
  } catch (error) {
    core.warning(`Failed to scan directory ${dir}: ${getErrorMessage(error)}`);
  }
  return results;
}

/**
 * Redacts secrets from file content using exact string matching
 * @param {string} content - File content to process
 * @param {string[]} secretValues - Array of secret values to redact
 * @returns {{content: string, redactionCount: number}} Redacted content and count of redactions
 */
function redactSecrets(content, secretValues) {
  let redactionCount = 0;
  let redacted = content;
  // Sort secret values by length (longest first) to handle overlapping secrets
  const sortedSecrets = secretValues.slice().sort((a, b) => b.length - a.length);
  for (const secretValue of sortedSecrets) {
    // Skip empty or very short values (likely not actual secrets)
    if (!secretValue || secretValue.length < 8) {
      continue;
    }
    // Count occurrences before replacement
    // Use split and join for exact string matching (not regex)
    // This is safer than regex as it doesn't interpret special characters
    // Show first 3 letters followed by asterisks for the remaining length
    const prefix = secretValue.substring(0, 3);
    const asterisks = "*".repeat(Math.max(0, secretValue.length - 3));
    const replacement = prefix + asterisks;
    const parts = redacted.split(secretValue);
    const occurrences = parts.length - 1;
    if (occurrences > 0) {
      redacted = parts.join(replacement);
      redactionCount += occurrences;
      core.info(`Redacted ${occurrences} occurrence(s) of a secret`);
    }
  }
  return { content: redacted, redactionCount };
}

/**
 * Process a single file for secret redaction
 * @param {string} filePath - Path to the file
 * @param {string[]} secretValues - Array of secret values to redact
 * @returns {number} Number of redactions made
 */
function processFile(filePath, secretValues) {
  try {
    const content = fs.readFileSync(filePath, "utf8");
    const { content: redactedContent, redactionCount } = redactSecrets(content, secretValues);
    if (redactionCount > 0) {
      fs.writeFileSync(filePath, redactedContent, "utf8");
      core.info(`Processed ${filePath}: ${redactionCount} redaction(s)`);
    }
    return redactionCount;
  } catch (error) {
    core.warning(`Failed to process file ${filePath}: ${getErrorMessage(error)}`);
    return 0;
  }
}

/**
 * Main function
 */
async function main() {
  // Get the list of secret names from environment variable
  const secretNames = process.env.GH_AW_SECRET_NAMES;
  if (!secretNames) {
    core.info("GH_AW_SECRET_NAMES not set, no redaction performed");
    return;
  }
  core.info("Starting secret redaction in /tmp/gh-aw directory");
  try {
    // Parse the comma-separated list of secret names
    const secretNameList = secretNames.split(",").filter(name => name.trim());
    // Collect the actual secret values from environment variables
    const secretValues = [];
    for (const secretName of secretNameList) {
      const envVarName = `SECRET_${secretName}`;
      const secretValue = process.env[envVarName];
      // Skip empty or undefined secrets
      if (!secretValue || secretValue.trim() === "") {
        continue;
      }
      secretValues.push(secretValue.trim());
    }
    if (secretValues.length === 0) {
      core.info("No secret values found to redact");
      return;
    }
    core.info(`Found ${secretValues.length} secret(s) to redact`);
    // Find all target files in /tmp/gh-aw and /opt/gh-aw directories
    const targetExtensions = [".txt", ".json", ".log", ".md", ".mdx", ".yml", ".jsonl"];
    const tmpFiles = findFiles("/tmp/gh-aw", targetExtensions);
    const optFiles = findFiles("/opt/gh-aw", targetExtensions);
    const files = [...tmpFiles, ...optFiles];
    core.info(`Found ${files.length} file(s) to scan for secrets (${tmpFiles.length} in /tmp/gh-aw, ${optFiles.length} in /opt/gh-aw)`);
    let totalRedactions = 0;
    let filesWithRedactions = 0;
    // Process each file
    for (const file of files) {
      const redactionCount = processFile(file, secretValues);
      if (redactionCount > 0) {
        filesWithRedactions++;
        totalRedactions += redactionCount;
      }
    }
    if (totalRedactions > 0) {
      core.info(`Secret redaction complete: ${totalRedactions} redaction(s) in ${filesWithRedactions} file(s)`);
    } else {
      core.info("Secret redaction complete: no secrets found");
    }
  } catch (error) {
    core.setFailed(`Secret redaction failed: ${getErrorMessage(error)}`);
  }
}

const { getErrorMessage } = require("./error_helpers.cjs");

module.exports = { main };
