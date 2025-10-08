/**
 * Redacts secrets from files in /tmp directory before uploading artifacts
 * This script processes all .txt, .json, .log files under /tmp and redacts
 * any strings matching the secrets pattern provided via environment variable.
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
    core.warning(`Failed to scan directory ${dir}: ${error instanceof Error ? error.message : String(error)}`);
  }

  return results;
}

/**
 * Redacts secrets from file content
 * @param {string} content - File content to process
 * @param {RegExp} secretPattern - Regular expression pattern for secrets
 * @returns {{content: string, redactionCount: number}} Redacted content and count of redactions
 */
function redactSecrets(content, secretPattern) {
  let redactionCount = 0;

  const redacted = content.replace(secretPattern, match => {
    redactionCount++;
    core.debug(`Redacted secret occurrence #${redactionCount}`);
    return "***REDACTED***";
  });

  return { content: redacted, redactionCount };
}

/**
 * Process a single file for secret redaction
 * @param {string} filePath - Path to the file
 * @param {RegExp} secretPattern - Regular expression pattern for secrets
 * @returns {number} Number of redactions made
 */
function processFile(filePath, secretPattern) {
  try {
    const content = fs.readFileSync(filePath, "utf8");
    const { content: redactedContent, redactionCount } = redactSecrets(content, secretPattern);

    if (redactionCount > 0) {
      fs.writeFileSync(filePath, redactedContent, "utf8");
      core.debug(`Processed ${filePath}: ${redactionCount} redaction(s)`);
    }

    return redactionCount;
  } catch (error) {
    core.warning(`Failed to process file ${filePath}: ${error instanceof Error ? error.message : String(error)}`);
    return 0;
  }
}

/**
 * Main function
 */
async function main() {
  // Get the secrets pattern from environment variable
  const secretsPattern = process.env.GITHUB_AW_SECRETS_PATTERN;

  if (!secretsPattern) {
    core.info("GITHUB_AW_SECRETS_PATTERN not set, no redaction performed");
    return;
  }

  core.info("Starting secret redaction in /tmp directory");

  try {
    // Create regex pattern from the environment variable
    // The pattern should be a regex string that matches any of the secrets
    const secretRegex = new RegExp(secretsPattern, "g");

    // Find all target files in /tmp directory
    const targetExtensions = [".txt", ".json", ".log"];
    const files = findFiles("/tmp", targetExtensions);

    core.info(`Found ${files.length} file(s) to scan for secrets`);

    let totalRedactions = 0;
    let filesWithRedactions = 0;

    // Process each file
    for (const file of files) {
      const redactionCount = processFile(file, secretRegex);
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
    core.setFailed(`Secret redaction failed: ${error instanceof Error ? error.message : String(error)}`);
  }
}

await main();
