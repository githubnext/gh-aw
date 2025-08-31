function main() {
  const fs = require('fs');
  const core = require('@actions/core');
  
  try {
    // Get the log file path from environment
    const logFile = process.env.AGENT_LOG_FILE;
    if (!logFile) {
      console.log('No agent log file specified');
      return;
    }
    
    if (!fs.existsSync(logFile)) {
      console.log(`Log file not found: ${logFile}`);
      return;
    }
    
    const logContent = fs.readFileSync(logFile, 'utf8');
    const markdown = parseCodexLog(logContent);
    
    // Append to GitHub step summary
    core.summary.addRaw(markdown).write();
    
  } catch (error) {
    console.error('Error parsing Codex log:', error.message);
    core.setFailed(error.message);
  }
}

function parseCodexLog(logContent) {
  try {
    const lines = logContent.split('\n');
    let markdown = '## 🤖 Agent Reasoning Sequence\n\n';
    
    let stepNumber = 1;
    let inThinkingSection = false;
    let thinkingContent = [];
    let currentToolUse = null;
    const commandSummary = []; // For the succinct summary
    
    // First pass: collect commands for summary
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      
      // Detect tool usage and exec commands
      if (line.includes('] tool ') && line.includes('(')) {
        // Extract tool name
        const toolMatch = line.match(/\] tool ([^(]+)\(/);
        if (toolMatch) {
          const toolName = toolMatch[1];
          if (toolName.includes('.')) {
            // Format as provider::method
            const parts = toolName.split('.');
            const provider = parts[0];
            const method = parts.slice(1).join('_');
            commandSummary.push(`\`${provider}::${method}(...)\``);
          }
        }
      } else if (line.includes('] exec ')) {
        // Extract exec command
        const execMatch = line.match(/exec (.+?) in/);
        if (execMatch) {
          const formattedCommand = formatBashCommand(execMatch[1]);
          commandSummary.push(`\`${formattedCommand}\``);
        }
      }
    }
    
    // Add command summary at the top
    if (commandSummary.length > 0) {
      markdown += '### Commands and Tools\n\n';
      for (const cmd of commandSummary) {
        markdown += `* ${cmd}\n`;
      }
      markdown += '\n';
    }
    
    // Second pass: main parsing
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      
      // Skip metadata lines at the start
      if (line.includes('OpenAI Codex') || line.startsWith('--------') || 
          line.includes('workdir:') || line.includes('model:') || 
          line.includes('provider:') || line.includes('approval:') || 
          line.includes('sandbox:') || line.includes('reasoning effort:') || 
          line.includes('reasoning summaries:') || line.includes('tokens used:')) {
        continue;
      }
      
      // Skip user instructions section (too verbose for summary)
      if (line.includes('User instructions:')) {
        // Skip until we hit a thinking section or tool call
        while (i < lines.length && 
               !lines[i].includes('thinking') && 
               !lines[i].includes('] tool ') &&
               !lines[i].includes('] exec ')) {
          i++;
        }
        i--; // Back up one since the loop will increment
        continue;
      }
      
      // Detect thinking sections
      if (line.includes('] thinking')) {
        // Process any previous thinking section
        if (thinkingContent.length > 0) {
          markdown += formatThinkingSection(thinkingContent);
          thinkingContent = [];
        }
        inThinkingSection = true;
        continue;
      }
      
      // Detect tool usage (both tool calls and exec commands)
      if ((line.includes('] tool ') || line.includes('] exec ')) && line.includes('(')) {
        // Process any previous thinking section
        if (thinkingContent.length > 0) {
          markdown += formatThinkingSection(thinkingContent);
          thinkingContent = [];
        }
        inThinkingSection = false;
        
        // Extract tool information
        let toolName = '';
        if (line.includes('] tool ')) {
          const toolMatch = line.match(/\] tool ([^(]+)\(/);
          if (toolMatch) {
            toolName = toolMatch[1];
          }
        } else if (line.includes('] exec ')) {
          toolName = 'exec';
        }
        
        if (toolName) {
          currentToolUse = {
            name: toolName,
            line: line,
            stepNumber: stepNumber++
          };
        }
        continue;
      }
      
      // Detect tool results
      if (currentToolUse && (line.includes(') success in ') || line.includes(') failed in ') || line.includes(') succeeded in '))) {
        const isSuccess = line.includes(') success in ') || line.includes(') succeeded in ');
        markdown += formatToolSection(currentToolUse, isSuccess, line);
        currentToolUse = null;
        continue;
      }
      
      // Collect thinking content
      if (inThinkingSection && line.trim() !== '' && !line.startsWith('[2025-')) {
        thinkingContent.push(line.trim());
      }
    }
    
    // Process final thinking section
    if (thinkingContent.length > 0) {
      markdown += formatThinkingSection(thinkingContent);
    }
    
    // If no meaningful content was found, show a minimal message
    if (stepNumber === 1 && !markdown.includes('💭')) {
      markdown += '_Log parsing in progress or minimal output detected._\n';
    }
    
    return markdown;
  } catch (error) {
    return `## Agent Log Summary\n\nError parsing Codex log: ${error.message}\n`;
  }
}

function formatThinkingSection(content) {
  if (!content || content.length === 0) return '';
  
  let markdown = '### � Reasoning\n\n';
  
  // Join and clean the thinking content
  const cleanedContent = content
    .join(' ')
    .replace(/\s+/g, ' ') // Normalize whitespace
    .trim();
  
  if (cleanedContent) {
    markdown += cleanedContent + '\n\n';
  }
  
  return markdown;
}

function formatToolSection(toolUse, isSuccess, resultLine) {
  if (!toolUse) return '';
  
  const toolName = toolUse.name;
  let markdown = `### ${toolUse.stepNumber}. 🔧 ${toolName}\n\n`;
  
  // Extract parameters from the tool call line
  if (toolName === 'exec') {
    // Handle exec commands specially
    const execMatch = toolUse.line.match(/exec (.+) in/);
    if (execMatch) {
      markdown += `**Command:** \`${execMatch[1]}\`\n\n`;
    }
  } else {
    // Handle tool calls
    const paramMatch = toolUse.line.match(/\(([^)]*)\)/);
    if (paramMatch && paramMatch[1]) {
      try {
        const params = JSON.parse(paramMatch[1]);
        markdown += formatToolParameters(params);
      } catch (e) {
        // If JSON parsing fails, just show the raw parameters
        markdown += `**Parameters:** \`${paramMatch[1]}\`\n\n`;
      }
    }
  }
  
  // Add result
  if (isSuccess) {
    markdown += '**Result:** ✅ Success\n\n';
  } else {
    markdown += '**Result:** ❌ Error\n\n';
  }
  
  // Extract timing if available
  const timeMatch = resultLine.match(/in (\d+ms)/);
  if (timeMatch) {
    markdown += `**Duration:** ${timeMatch[1]}\n\n`;
  }
  
  return markdown;
}

function formatToolParameters(params) {
  let markdown = '';
  
  const keys = Object.keys(params);
  if (keys.length > 0) {
    markdown += '**Input:**\n';
    for (const key of keys.slice(0, 3)) { // Limit to first 3 keys
      const value = String(params[key] || '');
      markdown += `- **${key}:** ${truncateString(value, 60)}\n`;
    }
    if (keys.length > 3) {
      markdown += `- _(${keys.length - 3} more fields...)_\n`;
    }
    markdown += '\n';
  }
  
  return markdown;
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
  module.exports = { parseCodexLog, formatThinkingSection, formatToolSection, formatBashCommand, truncateString };
}

main();
