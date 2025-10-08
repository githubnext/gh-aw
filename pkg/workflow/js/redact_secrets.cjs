const fs = require("fs");
const path = require("path");
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
    const prefix = secretValue.substring(0, 3);
    const asterisks = "*".repeat(Math.max(0, secretValue.length - 3));
    const replacement = prefix + asterisks;
    const parts = redacted.split(secretValue);
    const occurrences = parts.length - 1;
    if (occurrences > 0) {
      redacted = parts.join(replacement);
      redactionCount += occurrences;
      core.debug(`Redacted ${occurrences} occurrence(s) of a secret`);
    }
  }
  return { content: redacted, redactionCount };
}
function processFile(filePath, secretValues) {
  try {
    const content = fs.readFileSync(filePath, "utf8");
    const { content: redactedContent, redactionCount } = redactSecrets(content, secretValues);
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
async function main() {
  const secretNames = process.env.GITHUB_AW_SECRET_NAMES;
  if (!secretNames) {
    core.info("GITHUB_AW_SECRET_NAMES not set, no redaction performed");
    return;
  }
  core.info("Starting secret redaction in /tmp directory");
  try {
    const secretNameList = secretNames.split(",").filter(name => name.trim());
    const secretValues = [];
    for (const secretName of secretNameList) {
      const envVarName = `SECRET_${secretName}`;
      const secretValue = process.env[envVarName];
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
    const targetExtensions = [".txt", ".json", ".log"];
    const files = findFiles("/tmp", targetExtensions);
    core.info(`Found ${files.length} file(s) to scan for secrets`);
    let totalRedactions = 0;
    let filesWithRedactions = 0;
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
    core.setFailed(`Secret redaction failed: ${error instanceof Error ? error.message : String(error)}`);
  }
}
await main();
