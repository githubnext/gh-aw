function main() {
  const fs = require("fs");
  const path = require("path");

  try {
    const logFile = process.env.GITHUB_AW_AGENT_OUTPUT;
    const declaredOutputFiles = process.env.GITHUB_AW_DECLARED_OUTPUT_FILES;

    // Build list of paths to check - start with declared output files, then add log file
    const outputPaths = [];
    if (declaredOutputFiles) {
      outputPaths.push(...declaredOutputFiles.split("\n").filter(p => p.trim()));
    }
    if (logFile) {
      outputPaths.push(logFile);
    }

    if (outputPaths.length === 0) {
      core.info("No agent log file or output files specified");
      return;
    }

    // Try to find and parse agentic output from any of the paths
    const agenticOutput = findAgenticOutputInPaths(outputPaths);

    if (agenticOutput) {
      const parsedLog = parseCodexLog(agenticOutput);
      if (parsedLog) {
        core.info(parsedLog);
        core.summary.addRaw(parsedLog).write();
        core.info("Codex log parsed successfully");
      } else {
        core.error("Failed to parse Codex log");
      }
    } else {
      core.info("No agent log content found");
    }
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
}

/**
 * Search for agentic output files in the declared output paths
 * @param {string[]} outputPaths - Array of file or directory paths
 * @returns {string|null} The content of the first agentic output file found, or null
 */
function findAgenticOutputInPaths(outputPaths) {
  const fs = require("fs");
  const path = require("path");

  // Common agentic output file names to search for
  const outputFileNames = ["agentic-output.md", "output.md", "agentic-output.txt", "output.txt"];
  // Valid file extensions for agentic output
  const validExtensions = [".txt", ".log", ".jsonl", ".json", ".md"];

  for (const outputPath of outputPaths) {
    if (!outputPath || !outputPath.trim()) {
      continue;
    }

    const trimmedPath = outputPath.trim();

    try {
      const stats = fs.statSync(trimmedPath);

      if (stats.isDirectory()) {
        // Search for any file with valid extensions in the directory
        const files = fs.readdirSync(trimmedPath);
        for (const file of files) {
          const ext = path.extname(file).toLowerCase();
          if (validExtensions.includes(ext)) {
            const filePath = path.join(trimmedPath, file);
            core.info(`Found agentic output file: ${filePath}`);
            return fs.readFileSync(filePath, "utf8");
          }
        }
      } else if (stats.isFile()) {
        // If it's a file, check if it has a valid extension
        const ext = path.extname(trimmedPath).toLowerCase();
        if (validExtensions.includes(ext)) {
          core.info(`Found agentic output file: ${trimmedPath}`);
          return fs.readFileSync(trimmedPath, "utf8");
        }
      }
    } catch (error) {
      // Path doesn't exist or can't be accessed, continue to next path
      continue;
    }
  }

  return null;
}

/**
 * Parse codex log content and format as markdown
 * @param {string} logContent - The raw log content to parse
 * @returns {string} Formatted markdown content
 */
function parseCodexLog(logContent) {
  try {
    const lines = logContent.split("\n");
    let markdown = "## 🤖 Commands and Tools\n\n";

    const commandSummary = [];

    // First pass: collect commands for summary
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Detect tool usage and exec commands
      if (line.includes("] tool ") && line.includes("(")) {
        // Extract tool name
        const toolMatch = line.match(/\] tool ([^(]+)\(/);
        if (toolMatch) {
          const toolName = toolMatch[1];

          // Look ahead to find the result status
          let statusIcon = "❓"; // Unknown by default
          for (let j = i + 1; j < Math.min(i + 5, lines.length); j++) {
            const nextLine = lines[j];
            if (nextLine.includes("success in")) {
              statusIcon = "✅";
              break;
            } else if (nextLine.includes("failure in") || nextLine.includes("error in") || nextLine.includes("failed in")) {
              statusIcon = "❌";
              break;
            }
          }

          if (toolName.includes(".")) {
            // Format as provider::method
            const parts = toolName.split(".");
            const provider = parts[0];
            const method = parts.slice(1).join("_");
            commandSummary.push(`* ${statusIcon} \`${provider}::${method}(...)\``);
          } else {
            commandSummary.push(`* ${statusIcon} \`${toolName}(...)\``);
          }
        }
      } else if (line.includes("] exec ")) {
        // Extract exec command
        const execMatch = line.match(/exec (.+?) in/);
        if (execMatch) {
          const formattedCommand = formatBashCommand(execMatch[1]);

          // Look ahead to find the result status
          let statusIcon = "❓"; // Unknown by default
          for (let j = i + 1; j < Math.min(i + 5, lines.length); j++) {
            const nextLine = lines[j];
            if (nextLine.includes("succeeded in")) {
              statusIcon = "✅";
              break;
            } else if (nextLine.includes("failed in") || nextLine.includes("error")) {
              statusIcon = "❌";
              break;
            }
          }

          commandSummary.push(`* ${statusIcon} \`${formattedCommand}\``);
        }
      }
    }

    // Add command summary
    if (commandSummary.length > 0) {
      for (const cmd of commandSummary) {
        markdown += `${cmd}\n`;
      }
    } else {
      markdown += "No commands or tools used.\n";
    }

    // Add Information section
    markdown += "\n## 📊 Information\n\n";

    // Extract metadata from Codex logs
    let totalTokens = 0;
    const tokenMatches = logContent.match(/tokens used: (\d+)/g);
    if (tokenMatches) {
      for (const match of tokenMatches) {
        const numberMatch = match.match(/(\d+)/);
        if (numberMatch) {
          const tokens = parseInt(numberMatch[1]);
          totalTokens += tokens;
        }
      }
    }

    if (totalTokens > 0) {
      markdown += `**Total Tokens Used:** ${totalTokens.toLocaleString()}\n\n`;
    }

    // Count tool calls and exec commands
    const toolCalls = (logContent.match(/\] tool /g) || []).length;
    const execCommands = (logContent.match(/\] exec /g) || []).length;

    if (toolCalls > 0) {
      markdown += `**Tool Calls:** ${toolCalls}\n\n`;
    }

    if (execCommands > 0) {
      markdown += `**Commands Executed:** ${execCommands}\n\n`;
    }

    markdown += "\n## 🤖 Reasoning\n\n";

    // Second pass: process full conversation flow with interleaved reasoning, tools, and commands
    let inThinkingSection = false;

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Skip metadata lines
      if (
        line.includes("OpenAI Codex") ||
        line.startsWith("--------") ||
        line.includes("workdir:") ||
        line.includes("model:") ||
        line.includes("provider:") ||
        line.includes("approval:") ||
        line.includes("sandbox:") ||
        line.includes("reasoning effort:") ||
        line.includes("reasoning summaries:") ||
        line.includes("tokens used:")
      ) {
        continue;
      }

      // Process thinking sections
      if (line.includes("] thinking")) {
        inThinkingSection = true;
        continue;
      }

      // Process tool calls
      if (line.includes("] tool ") && line.includes("(")) {
        inThinkingSection = false;
        const toolMatch = line.match(/\] tool ([^(]+)\(/);
        if (toolMatch) {
          const toolName = toolMatch[1];

          // Look ahead to find the result status
          let statusIcon = "❓"; // Unknown by default
          for (let j = i + 1; j < Math.min(i + 5, lines.length); j++) {
            const nextLine = lines[j];
            if (nextLine.includes("success in")) {
              statusIcon = "✅";
              break;
            } else if (nextLine.includes("failure in") || nextLine.includes("error in") || nextLine.includes("failed in")) {
              statusIcon = "❌";
              break;
            }
          }

          if (toolName.includes(".")) {
            const parts = toolName.split(".");
            const provider = parts[0];
            const method = parts.slice(1).join("_");
            markdown += `${statusIcon} ${provider}::${method}(...)\n\n`;
          } else {
            markdown += `${statusIcon} ${toolName}(...)\n\n`;
          }
        }
        continue;
      }

      // Process exec commands
      if (line.includes("] exec ")) {
        inThinkingSection = false;
        const execMatch = line.match(/exec (.+?) in/);
        if (execMatch) {
          const formattedCommand = formatBashCommand(execMatch[1]);

          // Look ahead to find the result status
          let statusIcon = "❓"; // Unknown by default
          for (let j = i + 1; j < Math.min(i + 5, lines.length); j++) {
            const nextLine = lines[j];
            if (nextLine.includes("succeeded in")) {
              statusIcon = "✅";
              break;
            } else if (nextLine.includes("failed in") || nextLine.includes("error")) {
              statusIcon = "❌";
              break;
            }
          }

          markdown += `${statusIcon} \`${formattedCommand}\`\n\n`;
        }
        continue;
      }

      // Process thinking content
      if (inThinkingSection && line.trim().length > 20 && !line.startsWith("[2025-")) {
        const trimmed = line.trim();
        // Add thinking content directly
        markdown += `${trimmed}\n\n`;
      }
    }

    return markdown;
  } catch (error) {
    core.error(`Error parsing Codex log: ${error}`);
    return "## 🤖 Commands and Tools\n\nError parsing log content.\n\n## 🤖 Reasoning\n\nUnable to parse reasoning from log.\n\n";
  }
}

/**
 * Format bash command for display
 * @param {string} command - The command to format
 * @returns {string} Formatted command string
 */
function formatBashCommand(command) {
  if (!command) return "";

  // Convert multi-line commands to single line by replacing newlines with spaces
  // and collapsing multiple spaces
  let formatted = command
    .replace(/\n/g, " ") // Replace newlines with spaces
    .replace(/\r/g, " ") // Replace carriage returns with spaces
    .replace(/\t/g, " ") // Replace tabs with spaces
    .replace(/\s+/g, " ") // Collapse multiple spaces into one
    .trim(); // Remove leading/trailing whitespace

  // Escape backticks to prevent markdown issues
  formatted = formatted.replace(/`/g, "\\`");

  // Truncate if too long (keep reasonable length for summary)
  const maxLength = 80;
  if (formatted.length > maxLength) {
    formatted = formatted.substring(0, maxLength) + "...";
  }

  return formatted;
}

/**
 * Truncate string to maximum length
 * @param {string} str - The string to truncate
 * @param {number} maxLength - Maximum length allowed
 * @returns {string} Truncated string
 */
function truncateString(str, maxLength) {
  if (!str) return "";
  if (str.length <= maxLength) return str;
  return str.substring(0, maxLength) + "...";
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = { parseCodexLog, formatBashCommand, truncateString };
}

main();
