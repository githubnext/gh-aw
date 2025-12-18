// @ts-check
/// <reference types="@actions/github-script" />

function main() {
  const fs = require("fs");
  const path = require("path");

  core.info("Starting validate_errors.cjs script");
  const startTime = Date.now();

  try {
    const logPath = process.env.GH_AW_AGENT_OUTPUT;
    if (!logPath) {
      throw new Error("GH_AW_AGENT_OUTPUT environment variable is required");
    }

    core.info(`Log path: ${logPath}`);

    if (!fs.existsSync(logPath)) {
      core.info(`Log path not found: ${logPath}`);
      core.info("No logs to validate - skipping error validation");
      return;
    }

    // Get error patterns from environment variables
    const patterns = getErrorPatternsFromEnv();
    if (patterns.length === 0) {
      throw new Error("GH_AW_ERROR_PATTERNS environment variable is required and must contain at least one pattern");
    }

    core.info(`Loaded ${patterns.length} error patterns`);
    core.info(`Patterns: ${JSON.stringify(patterns.map(p => ({ description: p.description, pattern: p.pattern })))}`);

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
        core.info(`Reading log file: ${file} (${fileContent.length} bytes)`);
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
  const patternsEnv = process.env.GH_AW_ERROR_PATTERNS;
  if (!patternsEnv) {
    throw new Error("GH_AW_ERROR_PATTERNS environment variable is required");
  }

  try {
    const patterns = JSON.parse(patternsEnv);
    if (!Array.isArray(patterns)) {
      throw new Error("GH_AW_ERROR_PATTERNS must be a JSON array");
    }
    return patterns;
  } catch (e) {
    throw new Error(`Failed to parse GH_AW_ERROR_PATTERNS as JSON: ${e instanceof Error ? e.message : String(e)}`);
  }
}

/**
 * Determine if a log line should be skipped during error validation.
 * This prevents false positives from environment variable definitions and other metadata.
 * @param {string} line - The log line to check
 * @returns {boolean} - True if the line should be skipped
 */
function shouldSkipLine(line) {
  // GitHub Actions timestamp format: YYYY-MM-DDTHH:MM:SS.MMMZ
  const GITHUB_ACTIONS_TIMESTAMP = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z\s+/;

  // Skip GitHub Actions environment variable declarations
  // Format: "2025-10-11T21:23:50.7459810Z   GH_AW_ERROR_PATTERNS: [..."
  if (new RegExp(GITHUB_ACTIONS_TIMESTAMP.source + "GH_AW_ERROR_PATTERNS:").test(line)) {
    return true;
  }

  // Skip lines that are showing environment variables in GitHub Actions format
  // Format: "   GH_AW_ERROR_PATTERNS: [..."
  if (/^\s+GH_AW_ERROR_PATTERNS:\s*\[/.test(line)) {
    return true;
  }

  // Skip lines showing env: section in GitHub Actions logs
  // Format: "2025-10-11T21:23:50.7453806Z env:"
  if (new RegExp(GITHUB_ACTIONS_TIMESTAMP.source + "env:").test(line)) {
    return true;
  }

  // Skip Copilot CLI DEBUG messages
  // Format: "2025-12-15T08:35:23.457Z [DEBUG] ..."
  // These are diagnostic messages that may contain error patterns but are not actual errors
  if (/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z\s+\[DEBUG\]/.test(line)) {
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

  // Configuration for infinite loop detection and performance
  const MAX_ITERATIONS_PER_LINE = 10000; // Maximum regex matches per line
  const ITERATION_WARNING_THRESHOLD = 1000; // Warn if iterations exceed this
  const MAX_TOTAL_ERRORS = 100; // Stop after finding this many errors (prevents excessive processing)
  const MAX_LINE_LENGTH = 10000; // Skip lines longer than this (likely JSON payloads)
  const TOP_SLOW_PATTERNS_COUNT = 5; // Number of slowest patterns to report

  core.info(`Starting error validation with ${patterns.length} patterns and ${lines.length} lines`);

  const validationStartTime = Date.now();
  let totalMatches = 0;
  let patternStats = [];

  for (let patternIndex = 0; patternIndex < patterns.length; patternIndex++) {
    const pattern = patterns[patternIndex];
    const patternStartTime = Date.now();
    let patternMatches = 0;

    let regex;
    try {
      regex = new RegExp(pattern.pattern, "g");
      core.info(`Pattern ${patternIndex + 1}/${patterns.length}: ${pattern.description || "Unknown"} - regex: ${pattern.pattern}`);
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

      // Skip very long lines that are likely JSON payloads or dumps
      // These rarely contain actionable error messages and are expensive to process
      if (line.length > MAX_LINE_LENGTH) {
        continue;
      }

      // Early termination if we've found too many errors
      if (totalMatches >= MAX_TOTAL_ERRORS) {
        core.warning(`Stopping error validation after finding ${totalMatches} matches (max: ${MAX_TOTAL_ERRORS})`);
        break;
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
          core.warning(`High iteration count (${iterationCount}) on line ${lineIndex + 1} with pattern: ${pattern.description || pattern.pattern}`);
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

        patternMatches++;
        totalMatches++;
      }

      // Log if we had a significant number of matches on a line
      if (iterationCount > 100) {
        core.info(`Line ${lineIndex + 1} had ${iterationCount} matches for pattern: ${pattern.description || pattern.pattern}`);
      }
    }

    // Track pattern performance
    const patternElapsed = Date.now() - patternStartTime;
    patternStats.push({
      description: pattern.description || "Unknown",
      pattern: pattern.pattern.substring(0, 50) + (pattern.pattern.length > 50 ? "..." : ""),
      matches: patternMatches,
      timeMs: patternElapsed,
    });

    // Log slow patterns (> 5 seconds)
    if (patternElapsed > 5000) {
      core.warning(`Pattern "${pattern.description}" took ${patternElapsed}ms to process (${patternMatches} matches)`);
    }

    // Early termination if we've found enough errors
    if (totalMatches >= MAX_TOTAL_ERRORS) {
      core.warning(`Stopping pattern processing after finding ${totalMatches} matches (max: ${MAX_TOTAL_ERRORS})`);
      break;
    }
  }

  // Log performance summary
  const validationElapsed = Date.now() - validationStartTime;
  core.info(`Validation summary: ${totalMatches} total matches found in ${validationElapsed}ms`);

  // Log top slowest patterns
  patternStats.sort((a, b) => b.timeMs - a.timeMs);
  const topSlow = patternStats.slice(0, TOP_SLOW_PATTERNS_COUNT);
  if (topSlow.length > 0 && topSlow[0].timeMs > 1000) {
    core.info(`Top ${TOP_SLOW_PATTERNS_COUNT} slowest patterns:`);
    topSlow.forEach((stat, idx) => {
      core.info(`  ${idx + 1}. "${stat.description}" - ${stat.timeMs}ms (${stat.matches} matches)`);
    });
  }

  core.info(`Error validation completed. Errors found: ${hasErrors}`);
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
