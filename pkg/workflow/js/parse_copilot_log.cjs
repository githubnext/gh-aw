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
 * Parse JSON blocks from grouped log content
 * @param {string} jsonContent - The JSON content to parse
 * @param {string} blockType - The type of JSON block
 * @param {object} configInfo - Configuration info to populate
 * @param {array} toolsAvailable - Tools array to populate
 */
function parseJsonBlock(jsonContent, blockType, configInfo, toolsAvailable) {
  try {
    // Try to parse as JSON
    const jsonData = JSON.parse(jsonContent);
    
    if (blockType.includes("configured settings")) {
      // Extract configuration information
      if (jsonData.api && jsonData.api.copilot) {
        configInfo.integrationId = jsonData.api.copilot.integrationId;
      }
      
      if (jsonData.github && jsonData.github.repo) {
        configInfo.repository = `${jsonData.github.owner.name}/${jsonData.github.repo.name}`;
      }
      
      if (jsonData.problem && jsonData.problem.statement) {
        configInfo.problemStatement = jsonData.problem.statement
          .replace(/\n+/g, ' ')
          .replace(/\s+/g, ' ')
          .trim();
      }
      
      if (jsonData.service && jsonData.service.agent) {
        configInfo.model = jsonData.service.agent.model;
      }
    } else if (blockType.includes("Tools")) {
      // Extract available tools
      if (Array.isArray(jsonData)) {
        jsonData.forEach(tool => {
          if (tool.function && tool.function.name) {
            toolsAvailable.push(tool.function.name);
          }
        });
      }
    }
  } catch (e) {
    // If JSON parsing fails, try to extract key-value pairs manually
    const lines = jsonContent.split('\n');
    lines.forEach(line => {
      const trimmed = line.trim();
      
      // Extract simple key-value patterns
      if (trimmed.includes('"name":') && trimmed.includes('"bash"')) {
        toolsAvailable.push('bash');
      } else if (trimmed.includes('"name":') && trimmed.includes('"str_replace_editor"')) {
        toolsAvailable.push('str_replace_editor');
      } else if (trimmed.includes('"name":') && trimmed.includes('"write_bash"')) {
        toolsAvailable.push('write_bash');
      } else if (trimmed.includes('"name":') && trimmed.includes('"read_bash"')) {
        toolsAvailable.push('read_bash');
      } else if (trimmed.includes('"name":') && trimmed.includes('"stop_bash"')) {
        toolsAvailable.push('stop_bash');
      }
    });
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
    let markdown = "## 🤖 GitHub Copilot CLI Execution\n\n";

    const configInfo = {};
    const toolsAvailable = [];
    const debugInfo = [];
    let requestInfo = null;
    let hasSignificantOutput = false;

    // Parse the structured log format
    let inJsonBlock = false;
    let currentJsonContent = "";
    let jsonBlockType = "";

    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const trimmedLine = line.trim();

      // Skip empty lines
      if (!trimmedLine) continue;

      // Extract timestamp and log level from lines
      const timestampMatch = line.match(/^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[(DEBUG|INFO|WARN|ERROR|START-GROUP|END-GROUP)\]\s*(.*)$/);
      
      if (timestampMatch) {
        const [, timestamp, level, content] = timestampMatch;

        // Handle group markers for JSON content
        if (level === "START-GROUP") {
          inJsonBlock = true;
          jsonBlockType = content;
          currentJsonContent = "";
          continue;
        } else if (level === "END-GROUP") {
          if (inJsonBlock && currentJsonContent.trim()) {
            parseJsonBlock(currentJsonContent, jsonBlockType, configInfo, toolsAvailable);
            hasSignificantOutput = true;
          }
          inJsonBlock = false;
          currentJsonContent = "";
          jsonBlockType = "";
          continue;
        }

        // Collect JSON content when in a group
        if (inJsonBlock) {
          currentJsonContent += content + "\n";
          continue;
        }

        // Extract key information from debug messages
        if (level === "DEBUG") {
          if (content.includes("Using model:")) {
            const modelMatch = content.match(/Using model:\s*(.+)$/);
            if (modelMatch) {
              configInfo.model = modelMatch[1];
              hasSignificantOutput = true;
            }
          } else if (content.includes("Successfully listed") && content.includes("models")) {
            debugInfo.push("Model catalog loaded successfully");
          } else if (content.includes("Got model info:")) {
            debugInfo.push("Model capabilities retrieved");
          } else if (content.includes("response (Request-ID")) {
            const requestMatch = content.match(/response \(Request-ID ([^)]+)\)/);
            if (requestMatch) {
              requestInfo = {
                requestId: requestMatch[1],
                timestamp: timestamp
              };
              hasSignificantOutput = true;
            }
          }
        }
      } else if (inJsonBlock) {
        // Continue collecting JSON content
        currentJsonContent += line + "\n";
      }
    }

    // Build summary
    if (Object.keys(configInfo).length > 0) {
      markdown += "### 🔧 Configuration\n\n";
      
      if (configInfo.model) {
        markdown += `**Model:** ${configInfo.model}\n\n`;
      }
      
      if (configInfo.integrationId) {
        markdown += `**Integration:** ${configInfo.integrationId}\n\n`;
      }
      
      if (configInfo.repository) {
        markdown += `**Repository:** ${configInfo.repository}\n\n`;
      }
      
      if (configInfo.problemStatement) {
        const truncatedProblem = configInfo.problemStatement.length > 200 
          ? configInfo.problemStatement.substring(0, 197) + "..."
          : configInfo.problemStatement;
        markdown += `**Task:** ${truncatedProblem}\n\n`;
      }
    }

    if (toolsAvailable.length > 0) {
      markdown += "### 🛠️ Available Tools\n\n";
      const uniqueTools = [...new Set(toolsAvailable)];
      uniqueTools.forEach(tool => {
        markdown += `* ${tool}\n`;
      });
      markdown += "\n";
    }

    if (requestInfo) {
      markdown += "### 📤 API Interaction\n\n";
      markdown += `**Request ID:** ${requestInfo.requestId}\n\n`;
      markdown += `**Response Time:** ${requestInfo.timestamp}\n\n`;
    }

    if (debugInfo.length > 0) {
      markdown += "### 🔍 Debug Information\n\n";
      debugInfo.forEach(info => {
        markdown += `* ${info}\n`;
      });
      markdown += "\n";
    }

    if (!hasSignificantOutput) {
      markdown += "*No significant execution details found in Copilot CLI log.*\n\n";
      markdown += "*This may be a partial log or the execution was interrupted.*\n";
    }

    return markdown;
  } catch (error) {
    console.error("Error parsing Copilot log:", error);
    return `## 🤖 GitHub Copilot CLI Execution\n\n*Error parsing log: ${error.message}*\n`;
  }
}

main();