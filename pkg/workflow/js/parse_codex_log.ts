interface ToolCall {
  tool: string;
  count: number;
  details: string[];
}

interface CommandExecution {
  command: string;
  count: number;
  details: string[];
}

function parseCodexLogMain(): void {
  const fs = require("fs");

  try {
    const logFile = process.env.GITHUB_AW_AGENT_OUTPUT;
    if (!logFile) {
      core.info("No agent log file specified");
      return;
    }

    if (!fs.existsSync(logFile)) {
      core.info(`Log file not found: ${logFile}`);
      return;
    }

    const content = fs.readFileSync(logFile, "utf8");
    const parsedLog = parseCodexLog(content);

    if (parsedLog) {
      core.summary.addRaw(parsedLog).write();
      core.info("Codex log parsed successfully");
    } else {
      core.error("Failed to parse Codex log");
    }
  } catch (error) {
    core.setFailed(error instanceof Error ? error : String(error));
  }
}

/**
 * Parse codex log content and format as markdown
 * @param logContent - The raw log content to parse
 * @returns Formatted markdown content
 */
function parseCodexLog(logContent: string): string {
  try {
    const lines = logContent.split("\n");
    let markdown = "## 🤖 Commands and Tools\n\n";

    const commandSummary: CommandExecution[] = [];

    // First pass: collect commands for summary
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];

      // Detect tool usage and exec commands
      if (line.includes("] tool ") && line.includes("(")) {
        // Extract tool name
        const toolMatch = line.match(/\] tool ([^(]+)\(/);
        if (toolMatch) {
          const toolName = toolMatch[1].trim();
          let existing = commandSummary.find(cmd => cmd.command === toolName);
          if (!existing) {
            existing = { command: toolName, count: 0, details: [] };
            commandSummary.push(existing);
          }
          existing.count++;
          existing.details.push(line.trim());
        }
      }

      // Detect exec commands
      if (line.includes("exec:") || line.includes("$ ")) {
        const execMatch = line.match(/(?:exec:|[$])\s*(.+)/);
        if (execMatch) {
          const command = execMatch[1].trim();
          let existing = commandSummary.find(cmd => cmd.command === command);
          if (!existing) {
            existing = { command: command, count: 0, details: [] };
            commandSummary.push(existing);
          }
          existing.count++;
          existing.details.push(line.trim());
        }
      }
    }

    // Generate summary section
    if (commandSummary.length > 0) {
      markdown += "### Command Summary\n\n";
      
      // Sort by frequency
      commandSummary.sort((a, b) => b.count - a.count);
      
      for (const cmd of commandSummary) {
        markdown += `- **${cmd.command}** (${cmd.count} time${cmd.count !== 1 ? 's' : ''})\n`;
      }
      markdown += "\n";
    }

    // Generate detailed execution log
    markdown += "### Execution Log\n\n";
    
    let currentSection = "";
    let inCodeBlock = false;
    
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      
      // Skip empty lines
      if (line.trim() === "") {
        if (inCodeBlock) {
          markdown += "\n";
        }
        continue;
      }

      // Detect tool usage
      if (line.includes("] tool ") && line.includes("(")) {
        if (inCodeBlock) {
          markdown += "```\n\n";
          inCodeBlock = false;
        }
        
        const toolMatch = line.match(/\] tool ([^(]+)\(/);
        if (toolMatch) {
          const toolName = toolMatch[1].trim();
          if (currentSection !== toolName) {
            currentSection = toolName;
            markdown += `#### 🔧 Tool: ${toolName}\n\n`;
          }
          markdown += "```\n";
          markdown += line + "\n";
          inCodeBlock = true;
        }
      }
      // Detect exec commands
      else if (line.includes("exec:") || line.includes("$ ")) {
        if (inCodeBlock) {
          markdown += "```\n\n";
          inCodeBlock = false;
        }
        
        if (currentSection !== "shell") {
          currentSection = "shell";
          markdown += "#### 💻 Shell Commands\n\n";
        }
        markdown += "```bash\n";
        markdown += line + "\n";
        inCodeBlock = true;
      }
      // Continue existing code block
      else if (inCodeBlock) {
        markdown += line + "\n";
      }
      // Regular log line
      else {
        if (inCodeBlock) {
          markdown += "```\n\n";
          inCodeBlock = false;
        }
        markdown += `${line}\n\n`;
      }
    }

    // Close any open code block
    if (inCodeBlock) {
      markdown += "```\n\n";
    }

    // Add footer
    markdown += "---\n";
    markdown += "*Generated from Codex agent execution log*\n";

    return markdown;
  } catch (error) {
    core.error(`Error parsing log: ${error instanceof Error ? error.message : String(error)}`);
    return "";
  }
}

// Export for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    parseCodexLog,
  };
}

(async () => {
  parseCodexLogMain();
})();