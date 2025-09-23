function main() {
  const fs = require("fs");

  try {
    const logFile = process.env.GITHUB_AW_AGENT_OUTPUT;
    if (!logFile) {
      throw new Error("GITHUB_AW_AGENT_OUTPUT environment variable is required");
    }

    if (!fs.existsSync(logFile)) {
      throw new Error(`Log file not found: ${logFile}`);
    }

    // Get error patterns from environment variables
    const patterns = getErrorPatternsFromEnv();
    if (patterns.length === 0) {
      throw new Error("GITHUB_AW_ERROR_PATTERNS environment variable is required and must contain at least one pattern");
    }

    const content = fs.readFileSync(logFile, "utf8");
    const hasErrors = validateErrors(content, patterns);

    if (hasErrors) {
      core.setFailed("Errors detected in agent logs - failing workflow step");
    } else {
      core.info("Error validation completed successfully");
    }
  } catch (error) {
    console.debug(error);
    core.setFailed(`Error validating log: ${error instanceof Error ? error.message : String(error)}`);
  }
}

function getErrorPatternsFromEnv() {
  const patternsEnv = process.env.GITHUB_AW_ERROR_PATTERNS;
  if (!patternsEnv) {
    throw new Error("GITHUB_AW_ERROR_PATTERNS environment variable is required");
  }

  try {
    const patterns = JSON.parse(patternsEnv);
    if (!Array.isArray(patterns)) {
      throw new Error("GITHUB_AW_ERROR_PATTERNS must be a JSON array");
    }
    return patterns;
  } catch (e) {
    throw new Error(`Failed to parse GITHUB_AW_ERROR_PATTERNS as JSON: ${e instanceof Error ? e.message : String(e)}`);
  }
}

/**
 * @param {string} logContent
 * @param {any[]} patterns
 * @returns {boolean}
 */
function validateErrors(logContent, patterns) {
  const lines = logContent.split("\n");
  let hasErrors = false;

  for (const pattern of patterns) {
    const regex = new RegExp(pattern.pattern, "g");

    for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
      const line = lines[lineIndex];
      let match;

      while ((match = regex.exec(line)) !== null) {
        const level = extractLevel(match, pattern);
        const message = extractMessage(match, pattern, line);

        const errorMessage = `Line ${lineIndex + 1}: ${message} (Pattern: ${pattern.description || "Unknown pattern"}, Raw log: ${truncateString(line.trim(), 120)})`;

        if (level.toLowerCase() === "error") {
          core.error(errorMessage);
          hasErrors = true;
        } else {
          core.warning(errorMessage);
        }
      }
    }
  }

  return hasErrors;
}

/**
 * @param {any} match
 * @param {any} pattern
 * @returns {string}
 */
function extractLevel(match, pattern) {
  if (pattern.level_group && pattern.level_group > 0 && match[pattern.level_group]) {
    return match[pattern.level_group];
  }

  // Try to infer level from the match content
  const fullMatch = match[0];
  if (fullMatch.toLowerCase().includes("error")) {
    return "error";
  } else if (fullMatch.toLowerCase().includes("warn")) {
    return "warning";
  }

  return "unknown";
}

/**
 * @param {any} match
 * @param {any} pattern
 * @param {any} fullLine
 * @returns {string}
 */
function extractMessage(match, pattern, fullLine) {
  if (pattern.message_group && pattern.message_group > 0 && match[pattern.message_group]) {
    return match[pattern.message_group].trim();
  }

  // Fallback to the full match or line
  return match[0] || fullLine.trim();
}

/**
 * @param {any} str
 * @param {any} maxLength
 * @returns {string}
 */
function truncateString(str, maxLength) {
  if (!str) return "";
  if (str.length <= maxLength) return str;
  return str.substring(0, maxLength) + "...";
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    validateErrors,
    extractLevel,
    extractMessage,
    getErrorPatternsFromEnv,
    truncateString,
  };
}

// Only run main if this script is executed directly, not when imported for testing
if (typeof module === "undefined" || require.main === module) {
  main();
}
