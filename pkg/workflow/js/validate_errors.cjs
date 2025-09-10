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
    const result = validateErrors(content, patterns);

    if (result) {
      core.summary.addRaw(result).write();
      console.log("Error validation completed successfully");
    } else {
      console.log("No errors or warnings found in log");
    }
  } catch (error) {
    core.setFailed(`Error validating log: ${error.message}`);
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
    throw new Error(`Failed to parse GITHUB_AW_ERROR_PATTERNS as JSON: ${e.message}`);
  }
}

function validateErrors(logContent, patterns) {
  const errors = [];
  const warnings = [];
  const lines = logContent.split("\n");

  for (const pattern of patterns) {
    try {
      const regex = new RegExp(pattern.pattern, "g");

      for (let lineIndex = 0; lineIndex < lines.length; lineIndex++) {
        const line = lines[lineIndex];
        let match;

        while ((match = regex.exec(line)) !== null) {
          const level = extractLevel(match, pattern);
          const message = extractMessage(match, pattern, line);

          const errorInfo = {
            line: lineIndex + 1,
            level: level,
            message: message,
            pattern: pattern.description || "Unknown pattern",
            rawLine: line.trim(),
          };

          if (level.toLowerCase() === "error") {
            errors.push(errorInfo);
          } else if (level.toLowerCase().includes("warn")) {
            warnings.push(errorInfo);
          } else {
            // Default to warning for unknown levels
            warnings.push(errorInfo);
          }
        }
      }
    } catch (e) {
      console.warn(
        `Error processing pattern '${pattern.description}': ${e.message}`
      );
    }
  }

  return generateValidationSummary(errors, warnings);
}

function extractLevel(match, pattern) {
  if (
    pattern.level_group &&
    pattern.level_group > 0 &&
    match[pattern.level_group]
  ) {
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

function extractMessage(match, pattern, fullLine) {
  if (
    pattern.message_group &&
    pattern.message_group > 0 &&
    match[pattern.message_group]
  ) {
    return match[pattern.message_group].trim();
  }

  // Fallback to the full match or line
  return match[0] || fullLine.trim();
}

function generateValidationSummary(errors, warnings) {
  if (errors.length === 0 && warnings.length === 0) {
    return null; // No issues found
  }

  let markdown = "## 🔍 Log Validation Results\n\n";

  // Summary
  const totalIssues = errors.length + warnings.length;
  markdown += `Found **${totalIssues}** issue(s) in the agent log:\n`;
  if (errors.length > 0) {
    markdown += `- 🚨 **${errors.length}** error(s)\n`;
  }
  if (warnings.length > 0) {
    markdown += `- ⚠️ **${warnings.length}** warning(s)\n`;
  }
  markdown += "\n";

  // Errors section
  if (errors.length > 0) {
    markdown += "### 🚨 Errors\n\n";
    for (const error of errors) {
      markdown += `**Line ${error.line}**: ${error.message}\n`;
      markdown += `- *Pattern*: ${error.pattern}\n`;
      markdown += `- *Raw log*: \`${truncateString(error.rawLine, 120)}\`\n\n`;
    }
  }

  // Warnings section
  if (warnings.length > 0) {
    markdown += "### ⚠️ Warnings\n\n";
    for (const warning of warnings) {
      markdown += `**Line ${warning.line}**: ${warning.message}\n`;
      markdown += `- *Pattern*: ${warning.pattern}\n`;
      markdown += `- *Raw log*: \`${truncateString(warning.rawLine, 120)}\`\n\n`;
    }
  }

  // Add recommendations
  if (errors.length > 0) {
    markdown += "### 💡 Recommendations\n\n";
    markdown +=
      "- Review the errors above and check if they indicate problems with the agent execution\n";
    markdown +=
      "- Consider updating the workflow configuration if errors are recurring\n";
    markdown +=
      "- Check the full logs for additional context around these errors\n\n";
  }

  return markdown;
}

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
    generateValidationSummary,
    getErrorPatternsFromEnv,
    truncateString,
  };
}

main();
