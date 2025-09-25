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

    return markdown;
  } catch (error) {
    console.error("Error parsing Copilot log:", error);
    return `## 🤖 GitHub Copilot CLI Execution\n\n*Error parsing log: ${error.message}*\n`;
  }
}

main();
