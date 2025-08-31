function main() {
  const fs = require('fs');
  const core = require('@actions/core');
  
  try {
    const logFile = process.env.AGENT_LOG_FILE;
    if (!logFile) {
      console.log('No agent log file specified');
      return;
    }
    
    if (!fs.existsSync(logFile)) {
      console.log(`Log file not found: ${logFile}`);
      return;
    }
    
    const content = fs.readFileSync(logFile, 'utf8');
    const parsedLog = parseCodexLog(content);
    
    if (parsedLog) {
      core.summary.addRaw(parsedLog).write();
      console.log('Codex log parsed successfully');
    } else {
      console.log('Failed to parse Codex log');
    }
  } catch (error) {
    core.setFailed(error.message);
  }
}

function parseCodexLog(logContent) {
  try {
    const lines = logContent.split('\n');
    let markdown = '## 🤖 Commands and Tools\n\n';
    
    const commandSummary = [];
    
    // First pass: collect commands for summary
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      
      // Detect tool usage and exec commands
      if (line.includes('] tool ') && line.includes('(')) {
        // Extract tool name
        const toolMatch = line.match(/\] tool ([^(]+)\(/);
        if (toolMatch) {
          const toolName = toolMatch[1];
          
          // Look ahead to find the result status
          let statusIcon = '❓'; // Unknown by default
          for (let j = i + 1; j < Math.min(i + 5, lines.length); j++) {
            const nextLine = lines[j];
            if (nextLine.includes('success in')) {
              statusIcon = '✅';
              break;
            } else if (nextLine.includes('failure in') || nextLine.includes('error in') || nextLine.includes('failed in')) {
              statusIcon = '❌';
              break;
            }
          }
          
          if (toolName.includes('.')) {
            // Format as provider::method
            const parts = toolName.split('.');
            const provider = parts[0];
            const method = parts.slice(1).join('_');
            commandSummary.push(`* ${statusIcon} \`${provider}::${method}(...)\``);
          } else {
            commandSummary.push(`* ${statusIcon} \`${toolName}(...)\``);
          }
        }
      } else if (line.includes('] exec ')) {
        // Extract exec command
        const execMatch = line.match(/exec (.+?) in/);
        if (execMatch) {
          const formattedCommand = formatBashCommand(execMatch[1]);
          
          // Look ahead to find the result status
          let statusIcon = '❓'; // Unknown by default
          for (let j = i + 1; j < Math.min(i + 5, lines.length); j++) {
            const nextLine = lines[j];
            if (nextLine.includes('succeeded in')) {
              statusIcon = '✅';
              break;
            } else if (nextLine.includes('failed in') || nextLine.includes('error')) {
              statusIcon = '❌';
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
      markdown += 'No commands or tools used.\n';
    }
    
    // Add Information section
    markdown += '\n## 📊 Information\n\n';
    
    // Extract metadata from Codex logs
    let totalTokens = 0;
    const tokenMatches = logContent.match(/tokens used: (\d+)/g);
    if (tokenMatches) {
      for (const match of tokenMatches) {
        const tokens = parseInt(match.match(/(\d+)/)[1]);
        totalTokens += tokens;
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
    
    markdown += '\n## 🤖 Reasoning\n\n';
    
    // Second pass: process full conversation flow with interleaved reasoning, tools, and commands
    let inThinkingSection = false;
    
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      
      // Skip metadata lines
      if (line.includes('OpenAI Codex') || line.startsWith('--------') || 
          line.includes('workdir:') || line.includes('model:') || 
          line.includes('provider:') || line.includes('approval:') || 
          line.includes('sandbox:') || line.includes('reasoning effort:') || 
          line.includes('reasoning summaries:') || line.includes('tokens used:')) {
        continue;
      }
      
      // Process thinking sections
      if (line.includes('] thinking')) {
        inThinkingSection = true;
        continue;
      }
      
      // Process tool calls
      if (line.includes('] tool ') && line.includes('(')) {
        inThinkingSection = false;
        const toolMatch = line.match(/\] tool ([^(]+)\(/);
        if (toolMatch) {
          const toolName = toolMatch[1];
          
          // Look ahead to find the result status
          let statusIcon = '❓'; // Unknown by default
          for (let j = i + 1; j < Math.min(i + 5, lines.length); j++) {
            const nextLine = lines[j];
            if (nextLine.includes('success in')) {
              statusIcon = '✅';
              break;
            } else if (nextLine.includes('failure in') || nextLine.includes('error in') || nextLine.includes('failed in')) {
              statusIcon = '❌';
              break;
            }
          }
          
          if (toolName.includes('.')) {
            const parts = toolName.split('.');
            const provider = parts[0];
            const method = parts.slice(1).join('_');
            markdown += `${statusIcon} ${provider}::${method}(...)\n\n`;
          } else {
            markdown += `${statusIcon} ${toolName}(...)\n\n`;
          }
        }
        continue;
      }
      
      // Process exec commands
      if (line.includes('] exec ')) {
        inThinkingSection = false;
        const execMatch = line.match(/exec (.+?) in/);
        if (execMatch) {
          const formattedCommand = formatBashCommand(execMatch[1]);
          
          // Look ahead to find the result status
          let statusIcon = '❓'; // Unknown by default
          for (let j = i + 1; j < Math.min(i + 5, lines.length); j++) {
            const nextLine = lines[j];
            if (nextLine.includes('succeeded in')) {
              statusIcon = '✅';
              break;
            } else if (nextLine.includes('failed in') || nextLine.includes('error')) {
              statusIcon = '❌';
              break;
            }
          }
          
          markdown += `${statusIcon} \`${formattedCommand}\`\n\n`;
        }
        continue;
      }
      
      // Process thinking content
      if (inThinkingSection && line.trim().length > 20 && !line.startsWith('[2025-')) {
        const trimmed = line.trim();
        // Add thinking content directly
        markdown += `${trimmed}\n\n`;
      }
    }
    
    return markdown;
  } catch (error) {
    console.error('Error parsing Codex log:', error);
    return '## 🤖 Commands and Tools\n\nError parsing log content.\n\n## 🤖 Reasoning\n\nUnable to parse reasoning from log.\n\n';
  }
}

function formatBashCommand(command) {
  if (!command) return '';
  
  // Convert multi-line commands to single line by replacing newlines with spaces
  // and collapsing multiple spaces
  let formatted = command
    .replace(/\n/g, ' ')           // Replace newlines with spaces
    .replace(/\r/g, ' ')           // Replace carriage returns with spaces
    .replace(/\t/g, ' ')           // Replace tabs with spaces
    .replace(/\s+/g, ' ')          // Collapse multiple spaces into one
    .trim();                       // Remove leading/trailing whitespace
  
  // Escape backticks to prevent markdown issues
  formatted = formatted.replace(/`/g, '\\`');
  
  // Truncate if too long (keep reasonable length for summary)
  const maxLength = 80;
  if (formatted.length > maxLength) {
    formatted = formatted.substring(0, maxLength) + '...';
  }
  
  return formatted;
}

function truncateString(str, maxLength) {
  if (!str) return '';
  if (str.length <= maxLength) return str;
  return str.substring(0, maxLength) + '...';
}

// Export for testing
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { parseCodexLog, formatBashCommand, truncateString };
}

main();
