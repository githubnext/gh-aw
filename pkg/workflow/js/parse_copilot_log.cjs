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

/**
 * Parse copilot log content and format as markdown
 * @param {string} logContent - The raw log content to parse
 * @returns {string} Formatted markdown content
 */
function parseCopilotLog(logContent) {
  try {
    const lines = logContent.split("\n");
    let markdown = "## 🤖 Commands and Tools\n\n";

    const commandSummary = [];
    const errors = [];
    const warnings = [];
    let executionTime = null;
    let toolsUsed = [];
    let hasSignificantOutput = false;

    // First pass: collect commands, tools, and metadata
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const trimmedLine = line.trim();

      // Extract copilot CLI command execution
      if (trimmedLine.includes("copilot --") || trimmedLine.includes("github copilot")) {
        commandSummary.push(`* 🚀 **Command:** \`${trimmedLine}\``);
        hasSignificantOutput = true;
      }

      // Extract shell command executions
      if (trimmedLine.includes("[DEBUG] Executing shell command:") || 
          trimmedLine.includes("[INFO] Shell command executed")) {
        const cmdMatch = line.match(/Executing shell command:\s*(.+)$/);
        if (cmdMatch) {
          const command = formatBashCommand(cmdMatch[1]);
          
          // Look ahead for success/failure status
          let statusIcon = "❓";
          for (let j = i + 1; j < Math.min(i + 3, lines.length); j++) {
            const nextLine = lines[j];
            if (nextLine.includes("executed successfully") || nextLine.includes("[INFO] Shell command executed")) {
              statusIcon = "✅";
              break;
            } else if (nextLine.includes("failed") || nextLine.includes("error")) {
              statusIcon = "❌";
              break;
            }
          }
          
          commandSummary.push(`* ${statusIcon} \`${command}\``);
          toolsUsed.push("shell");
          hasSignificantOutput = true;
        }
      }

      // Extract MCP server connections and tools
      if (trimmedLine.includes("Connected to") && trimmedLine.includes("MCP server")) {
        const serverMatch = line.match(/Connected to (\w+) MCP server/);
        if (serverMatch) {
          commandSummary.push(`* 🔗 **MCP Server Connected:** ${serverMatch[1]}`);
          hasSignificantOutput = true;
        }
      }

      // Extract available tools
      if (trimmedLine.includes("Available tools:")) {
        const toolsMatch = line.match(/Available tools:\s*(.+)$/);
        if (toolsMatch) {
          const tools = toolsMatch[1].split(',').map(t => t.trim());
          toolsUsed.push(...tools);
          commandSummary.push(`* 🛠️ **Available Tools:** ${tools.join(', ')}`);
          hasSignificantOutput = true;
        }
      }

      // Extract execution time
      if (trimmedLine.includes("Total execution time:")) {
        const timeMatch = line.match(/Total execution time:\s*([\d.]+)\s*seconds/);
        if (timeMatch) {
          executionTime = timeMatch[1];
        }
      }

      // Extract errors with timestamps
      if (trimmedLine.match(/\[(ERROR|CRITICAL)\]/) || 
          trimmedLine.includes("copilot: error:") ||
          trimmedLine.includes("Fatal error:") ||
          trimmedLine.includes("npm ERR!")) {
        
        let errorMsg = trimmedLine;
        // Clean up timestamp and log level prefixes
        errorMsg = errorMsg.replace(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z\s*\[(ERROR|CRITICAL)\]\s*/, '');
        errorMsg = errorMsg.replace(/^copilot:\s*error:\s*/, '');
        errorMsg = errorMsg.replace(/^Fatal error:\s*/, '');
        errorMsg = errorMsg.replace(/^npm ERR!\s*/, '');
        
        if (errorMsg) {
          errors.push(errorMsg);
          hasSignificantOutput = true;
        }
      }

      // Extract warnings with timestamps
      if (trimmedLine.match(/\[(WARN|WARNING)\]/) || 
          trimmedLine.includes("Warning:")) {
        
        let warnMsg = trimmedLine;
        // Clean up timestamp and log level prefixes
        warnMsg = warnMsg.replace(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z\s*\[(WARN|WARNING)\]\s*/, '');
        warnMsg = warnMsg.replace(/^Warning:\s*/, '');
        
        if (warnMsg) {
          warnings.push(warnMsg);
          hasSignificantOutput = true;
        }
      }
    }

    // Add command summary
    if (commandSummary.length > 0) {
      for (const cmd of commandSummary) {
        markdown += `${cmd}\n`;
      }
      markdown += "\n";
    }

    // Second pass: extract code blocks and significant output
    let inCodeBlock = false;
    let currentCodeBlock = "";
    let currentLanguage = "";
    let inSuggestionOutput = false;

    markdown += "## 📋 Execution Output\n\n";

    for (const line of lines) {
      const trimmedLine = line.trim();

      // Handle code block markers in output
      if (trimmedLine.startsWith("```")) {
        if (!inCodeBlock) {
          // Starting a code block
          inCodeBlock = true;
          currentLanguage = trimmedLine.substring(3);
          currentCodeBlock = "";
        } else {
          // Ending a code block
          inCodeBlock = false;
          if (currentCodeBlock.trim()) {
            markdown += `\`\`\`${currentLanguage}\n${currentCodeBlock}\`\`\`\n\n`;
            hasSignificantOutput = true;
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

      // Capture suggestions and responses
      if (trimmedLine.startsWith("Suggestion:") || trimmedLine.startsWith("Response:")) {
        markdown += `**${trimmedLine}**\n\n`;
        inSuggestionOutput = true;
        hasSignificantOutput = true;
        continue;
      }

      // Capture significant copilot output (exclude debug timestamps)
      if (inSuggestionOutput || 
          (trimmedLine.length > 20 && 
           !trimmedLine.match(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/) &&
           !trimmedLine.startsWith("[DEBUG]") &&
           !trimmedLine.startsWith("[INFO]") &&
           !trimmedLine.startsWith("$") &&
           !trimmedLine.startsWith("#"))) {
        
        // Reset suggestion mode on empty line or new section  
        if (trimmedLine === "" || trimmedLine.includes("execution completed")) {
          inSuggestionOutput = false;
        }
        
        if (trimmedLine !== "") {
          markdown += `${trimmedLine}\n\n`;
          hasSignificantOutput = true;
        }
      }
    }

    // Handle any remaining code block
    if (inCodeBlock && currentCodeBlock.trim()) {
      markdown += `\`\`\`${currentLanguage}\n${currentCodeBlock}\`\`\`\n\n`;
      hasSignificantOutput = true;
    }

    // Add information section
    markdown += "## 📊 Information\n\n";

    if (executionTime) {
      markdown += `**Execution Time:** ${executionTime} seconds\n\n`;
    }

    if (toolsUsed.length > 0) {
      const uniqueTools = [...new Set(toolsUsed)];
      markdown += `**Tools Used:** ${uniqueTools.join(', ')}\n\n`;
    }

    if (commandSummary.length > 0) {
      markdown += `**Commands Executed:** ${commandSummary.filter(cmd => cmd.includes('`')).length}\n\n`;
    }

    // Add errors section if any
    if (errors.length > 0) {
      markdown += "## ❌ Errors\n\n";
      for (const error of errors) {
        markdown += `* ${error}\n`;
      }
      markdown += "\n";
    }

    // Add warnings section if any  
    if (warnings.length > 0) {
      markdown += "## ⚠️ Warnings\n\n";
      for (const warning of warnings) {
        markdown += `* ${warning}\n`;
      }
      markdown += "\n";
    }

    if (!hasSignificantOutput) {
      markdown += "*No significant output captured from GitHub Copilot CLI execution.*\n";
    }

    return markdown;
  } catch (error) {
    console.error("Error parsing Copilot log:", error);
    return `## 🤖 GitHub Copilot CLI Execution\n\n*Error parsing log: ${error.message}*\n`;
  }
}

/**
 * Format bash command for display (normalize whitespace and line breaks)
 * @param {string} command - Raw command string
 * @returns {string} Formatted command
 */
function formatBashCommand(command) {
  if (!command) return "";
  
  // Normalize whitespace and remove extra line breaks
  let formatted = command
    .replace(/\n\s*/g, " ")  // Replace line breaks with spaces
    .replace(/\s+/g, " ")   // Normalize multiple spaces
    .trim();
  
  // Truncate very long commands
  if (formatted.length > 80) {
    formatted = formatted.substring(0, 77) + "...";
  }
  
  return formatted;
}

main();
