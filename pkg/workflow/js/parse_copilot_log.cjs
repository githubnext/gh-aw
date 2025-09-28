function main() {
  const fs = require("fs");

  try {
    const logFile = process.env.AGENT_LOG_FILE;
    if (!logFile) {
      console.log("No agent log file specified");
      return;
    }

    if (!fs.existsSync(logFile)) {
      console.log(`Log file not found: ${logFile}`);
      return;
    }

    const content = fs.readFileSync(logFile, "utf8");
    const parsedLog = parseCopilotLog(content);

    if (parsedLog) {
      core.summary.addRaw(parsedLog).write();
      console.log("Copilot log parsed successfully");
    } else {
      console.log("Failed to parse Copilot log");
    }
  } catch (error) {
    core.setFailed(error.message);
  }
}

function parseCopilotLog(logContent) {
  try {
    const lines = logContent.split("\n");
    let markdown = "## 🤖 GitHub Copilot CLI Execution\n\n";

    let hasOutput = false;
    let inCodeBlock = false;
    let currentCodeBlock = "";
    let currentLanguage = "";

    for (const line of lines) {
      // Look for code block markers
      if (line.trim().startsWith("```")) {
        if (!inCodeBlock) {
          // Starting a code block
          inCodeBlock = true;
          currentLanguage = line.trim().substring(3);
          currentCodeBlock = "";
        } else {
          // Ending a code block
          inCodeBlock = false;
          if (currentCodeBlock.trim()) {
            markdown += `\`\`\`${currentLanguage}\n${currentCodeBlock}\`\`\`\n\n`;
            hasOutput = true;
          }
          currentCodeBlock = "";
          currentLanguage = "";
        }
        continue;
      }

      if (inCodeBlock) {
        currentCodeBlock += line + "\n";
        continue;
      }

      // Look for copilot CLI specific patterns
      if (line.includes("copilot -p") || line.includes("github copilot")) {
        markdown += `**Command:** \`${line.trim()}\`\n\n`;
        hasOutput = true;
      }

      // Look for responses or suggestions
      if (line.includes("Suggestion:") || line.includes("Response:")) {
        markdown += `**${line.trim()}**\n\n`;
        hasOutput = true;
      }

      // Look for errors or warnings
      if (line.toLowerCase().includes("error:")) {
        markdown += `❌ **Error:** ${line.trim()}\n\n`;
        hasOutput = true;
      } else if (line.toLowerCase().includes("warning:")) {
        markdown += `⚠️ **Warning:** ${line.trim()}\n\n`;
        hasOutput = true;
      }

      // Capture general output that looks important
      const trimmedLine = line.trim();
      if (
        trimmedLine &&
        !trimmedLine.startsWith("$") &&
        !trimmedLine.startsWith("#") &&
        !trimmedLine.match(/^\d{4}-\d{2}-\d{2}/) && // Skip timestamps
        trimmedLine.length > 10
      ) {
        // Only include lines that look like actual copilot output
        if (
          trimmedLine.includes("copilot") ||
          trimmedLine.includes("suggestion") ||
          trimmedLine.includes("generate") ||
          trimmedLine.includes("explain")
        ) {
          markdown += `${trimmedLine}\n\n`;
          hasOutput = true;
        }
      }
    }

    // Handle any remaining code block
    if (inCodeBlock && currentCodeBlock.trim()) {
      markdown += `\`\`\`${currentLanguage}\n${currentCodeBlock}\`\`\`\n\n`;
      hasOutput = true;
    }

    if (!hasOutput) {
      markdown += "*No significant output captured from Copilot CLI execution.*\n";
    }

    // Process permission errors and create missing-tool entries
    const permissionErrors = extractPermissionErrorsFromCopilotLog(logContent);
    if (permissionErrors.length > 0) {
      outputMissingToolsForPermissionErrors(permissionErrors);
    }

    return markdown;
  } catch (error) {
    console.error("Error parsing Copilot log:", error);
    return `## 🤖 GitHub Copilot CLI Execution\n\n*Error parsing log: ${error.message}*\n`;
  }
}

/**
 * Extracts permission errors from Copilot CLI log content
 * @param {string} logContent - The raw log content
 * @returns {Array<{tool: string, reason: string, content: string}>} Array of permission errors found
 */
function extractPermissionErrorsFromCopilotLog(logContent) {
  const permissionErrors = [];
  const lines = logContent.split("\n");
  
  // Permission error patterns to look for in Copilot CLI logs
  const permissionPatterns = [
    /access denied.*only authorized.*can trigger.*workflow/i,
    /access denied.*user.*not authorized/i,
    /repository permission check failed/i,
    /configuration error.*required permissions not specified/i,
    /permission.*denied/i,
    /unauthorized/i,
    /forbidden/i,
    /access.*restricted/i,
    /insufficient.*permission/i,
    /authentication failed/i,
    /token.*invalid/i,
    /not authorized.*copilot/i,
  ];

  for (const line of lines) {
    const trimmedLine = line.trim();
    if (!trimmedLine) continue;
    
    // Check if the line matches permission error patterns
    for (const pattern of permissionPatterns) {
      if (pattern.test(trimmedLine)) {
        permissionErrors.push({
          tool: 'github-copilot-cli',
          reason: `Permission denied: ${trimmedLine.substring(0, 200)}${trimmedLine.length > 200 ? '...' : ''}`,
          content: trimmedLine,
        });
        break; // Found a match, no need to check other patterns
      }
    }
  }

  return permissionErrors;
}

/**
 * Outputs missing-tool entries for permission errors to the safe outputs file
 * @param {Array<{tool: string, reason: string, content: string}>} permissionErrors - Array of permission errors
 */
function outputMissingToolsForPermissionErrors(permissionErrors) {
  const fs = require("fs");
  
  // Get the safe outputs file path from environment
  const safeOutputsFile = process.env.GITHUB_AW_SAFE_OUTPUTS;
  if (!safeOutputsFile) {
    console.log("GITHUB_AW_SAFE_OUTPUTS not set, cannot write permission error missing-tool entries");
    return;
  }

  try {
    // Create missing-tool entries for each permission error
    for (const error of permissionErrors) {
      const missingToolEntry = {
        type: "missing-tool",
        tool: error.tool,
        reason: error.reason,
        alternatives: "Check repository permissions and access controls",
        timestamp: new Date().toISOString(),
      };
      
      // Append to the safe outputs file as NDJSON
      fs.appendFileSync(safeOutputsFile, JSON.stringify(missingToolEntry) + "\n");
      console.log(`Recorded permission error as missing tool: ${error.tool}`);
    }
  } catch (writeError) {
    console.log(`Failed to write permission error missing-tool entries: ${writeError instanceof Error ? writeError.message : String(writeError)}`);
  }
}

main();
