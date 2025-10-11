function main() {
  const fs = require("fs");
  const path = require("path");

  core.debug("Starting validate_errors.cjs script");
  const startTime = Date.now();

  try {
    const logPath = process.env.GITHUB_AW_AGENT_OUTPUT;
    if (!logPath) {
      throw new Error("GITHUB_AW_AGENT_OUTPUT environment variable is required");
    }

    core.debug(`Log path: ${logPath}`);

    if (!fs.existsSync(logPath)) {
      throw new Error(`Log path not found: ${logPath}`);
    }

    // Get error patterns from environment variables
    const patterns = getErrorPatternsFromEnv();
    if (patterns.length === 0) {
      throw new Error("GITHUB_AW_ERROR_PATTERNS environment variable is required and must contain at least one pattern");
    }

    core.info(`Loaded ${patterns.length} error patterns`);
    core.debug(`Patterns: ${JSON.stringify(patterns.map(p => ({ description: p.description, pattern: p.pattern })))}`);

    let content = "";

    // Check if logPath is a directory or a file
    const stat = fs.statSync(logPath);
    if (stat.isDirectory()) {
      // Read all log files from the directory and concatenate them
      const files = fs.readdirSync(logPath);
      const logFiles = files.filter(file => file.endsWith(".log") || file.endsWith(".txt"));

      if (logFiles.length === 0) {
        core.info(`No log files found in directory: ${logPath}`);
        return;
      }

      core.info(`Found ${logFiles.length} log files in directory`);

      // Sort log files by name to ensure consistent ordering
      logFiles.sort();

      // Concatenate all log files
      for (const file of logFiles) {
        const filePath = path.join(logPath, file);
        const fileContent = fs.readFileSync(filePath, "utf8");
        core.debug(`Reading log file: ${file} (${fileContent.length} bytes)`);
        content += fileContent;
        // Add a newline between files if the previous file doesn't end with one
        if (content.length > 0 && !content.endsWith("\n")) {
          content += "\n";
        }
      }
    } else {
      // Read the single log file
      content = fs.readFileSync(logPath, "utf8");
      core.info(`Read single log file (${content.length} bytes)`);
    }

    core.info(`Total log content size: ${content.length} bytes, ${content.split("\n").length} lines`);

    const hasErrors = validateErrors(content, patterns);

    const elapsedTime = Date.now() - startTime;
    core.info(`Error validation completed in ${elapsedTime}ms`);

    if (hasErrors) {
      core.error("Errors detected in agent logs - continuing workflow step (not failing for now)");
      //core.setFailed("Errors detected in agent logs - failing workflow step");
    } else {
      core.info("Error validation completed successfully");
    }
  } catch (error) {
    console.debug(error);
    core.error(`Error validating log: ${error instanceof Error ? error.message : String(error)}`);
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
 * Determine if a log line should be skipped during error validation.
 * This prevents false positives from environment variable definitions and other metadata.
 * @param {string} line - The log line to check
 * @returns {boolean} - True if the line should be skipped
 */
function shouldSkipLine(line) {
  // Skip GitHub Actions environment variable declarations
  // Format: "2025-10-11T21:23:50.7459810Z   GITHUB_AW_ERROR_PATTERNS: [..."
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z\s+GITHUB_AW_ERROR_PATTERNS:/.test(line)) {
    return true;
  }

  // Skip lines that are showing environment variables in GitHub Actions format
  // Format: "   GITHUB_AW_ERROR_PATTERNS: [..."
  if (/^\s+GITHUB_AW_ERROR_PATTERNS:\s*\[/.test(line)) {
    return true;
  }

  // Skip lines showing env: section in GitHub Actions logs
  // Format: "2025-10-11T21:23:50.7453806Z env:"
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z\s+env:/.test(line)) {
    return true;
  }

  return false;
}

/**
 * @param {string} logContent
 * @param {any[]} patterns
 * @returns {boolean}
 */
function validateErrors(logContent, patterns) {
  const lines = logContent.split("\n");
  let hasErrors = false;

  // Configuration for infinite loop detection
  const MAX_ITERATIONS_PER_LINE = 10000; // Maximum regex matches per line
  const ITERATION_WARNING_THRESHOLD = 1000; // Warn if iterations exceed this

  core.debug(`Starting error validation with ${patterns.length} patterns and ${lines.length} lines`);

  for (let patternIndex = 0; patternIndex < patterns.length; patternIndex++) {
    const pattern = patterns[patternIndex];
    let regex;
    try {
      regex = new RegExp(pattern.pattern, "g");
      core.debug(`Pattern ${patternIndex + 1}/${patterns.length}: ${pattern.description || "Unknown"} - regex: ${pattern.pattern}`);
    } catch (e) {
      core.error(`invalid error regex pattern: ${pattern.pattern}`);
      continue;
    }

    for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
      const line = lines[lineIndex];

      // Skip lines that are environment variable definitions from GitHub Actions logs
      // These lines contain the error patterns themselves and create false positives
      if (shouldSkipLine(line)) {
        continue;
      }

      let match;
      let iterationCount = 0;
      let lastIndex = -1;

      while ((match = regex.exec(line)) !== null) {
        iterationCount++;

        // Detect potential infinite loop: regex.lastIndex not advancing
        if (regex.lastIndex === lastIndex) {
          core.error(`Infinite loop detected at line ${lineIndex + 1}! Pattern: ${pattern.pattern}, lastIndex stuck at ${lastIndex}`);
          core.error(`Line content (truncated): ${truncateString(line, 200)}`);
          break; // Exit the while loop to prevent hanging
        }
        lastIndex = regex.lastIndex;

        // Warn if iteration count is getting high
        if (iterationCount === ITERATION_WARNING_THRESHOLD) {
          core.warning(
            `High iteration count (${iterationCount}) on line ${lineIndex + 1} with pattern: ${pattern.description || pattern.pattern}`
          );
          core.warning(`Line content (truncated): ${truncateString(line, 200)}`);
        }

        // Hard limit to prevent actual infinite loops
        if (iterationCount > MAX_ITERATIONS_PER_LINE) {
          core.error(`Maximum iteration limit (${MAX_ITERATIONS_PER_LINE}) exceeded at line ${lineIndex + 1}! Pattern: ${pattern.pattern}`);
          core.error(`Line content (truncated): ${truncateString(line, 200)}`);
          core.error(`This likely indicates a problematic regex pattern. Skipping remaining matches on this line.`);
          break; // Exit the while loop
        }

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

      // Log if we had a significant number of matches on a line
      if (iterationCount > 100) {
        core.debug(`Line ${lineIndex + 1} had ${iterationCount} matches for pattern: ${pattern.description || pattern.pattern}`);
      }
    }
  }

  core.debug(`Error validation completed. Errors found: ${hasErrors}`);
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
    shouldSkipLine,
  };
}

// Only run main if this script is executed directly, not when imported for testing
if (typeof module === "undefined" || require.main === module) {
  main();
}
