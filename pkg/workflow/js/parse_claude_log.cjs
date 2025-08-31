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
    const markdown = parseClaudeLog(logContent);
    
    // Append to GitHub step summary
    core.summary.addRaw(markdown).write();
    
  } catch (error) {
    console.error('Error parsing Claude log:', error.message);
    core.setFailed(error.message);
  }
}

function parseClaudeLog(logContent) {
  try {
    const logEntries = JSON.parse(logContent);
    if (!Array.isArray(logEntries)) {
      return '## Agent Log Summary\n\nLog format not recognized as Claude JSON array.\n';
    }
    
    let markdown = '## 🤖 Commands and Tools\n\n';
    const toolUsePairs = new Map(); // Map tool_use_id to tool_result
    const commandSummary = []; // For the succinct summary
    
    // First pass: collect tool results by tool_use_id
    for (const entry of logEntries) {
      if (entry.type === 'user' && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === 'tool_result' && content.tool_use_id) {
            toolUsePairs.set(content.tool_use_id, content);
          }
        }
      }
    }
    
    // Collect all tool uses for summary
    for (const entry of logEntries) {
      if (entry.type === 'assistant' && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === 'tool_use') {
            const toolName = content.name;
            const input = content.input || {};
            
            // Skip internal tools - only show external commands and API calls
            if (['Read', 'Write', 'Edit', 'MultiEdit', 'LS', 'Grep', 'Glob', 'TodoWrite'].includes(toolName)) {
              continue; // Skip internal file operations and searches
            }
            
            // Find the corresponding tool result to get status
            const toolResult = toolUsePairs.get(content.id);
            let statusIcon = '❓';
            if (toolResult) {
              statusIcon = toolResult.is_error === true ? '❌' : '✅';
            }
            
            // Add to command summary (only external tools)
            if (toolName === 'Bash') {
              const formattedCommand = formatBashCommand(input.command || '');
              commandSummary.push(`* ${statusIcon} \`${formattedCommand}\``);
            } else if (toolName.startsWith('mcp__')) {
              const mcpName = formatMcpName(toolName);
              commandSummary.push(`* ${statusIcon} \`${mcpName}(...)\``);
            } else {
              // Handle other external tools (if any)
              commandSummary.push(`* ${statusIcon} ${toolName}`);
            }
          }
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
    
    // Add Information section from the last entry with result metadata
    markdown += '\n## 📊 Information\n\n';
    
    // Find the last entry with metadata
    const lastEntry = logEntries[logEntries.length - 1];
    if (lastEntry && (lastEntry.num_turns || lastEntry.duration_ms || lastEntry.total_cost_usd || lastEntry.usage)) {
      if (lastEntry.num_turns) {
        markdown += `**Turns:** ${lastEntry.num_turns}\n\n`;
      }
      
      if (lastEntry.duration_ms) {
        const durationSec = Math.round(lastEntry.duration_ms / 1000);
        const minutes = Math.floor(durationSec / 60);
        const seconds = durationSec % 60;
        markdown += `**Duration:** ${minutes}m ${seconds}s\n\n`;
      }
      
      if (lastEntry.total_cost_usd) {
        markdown += `**Total Cost:** $${lastEntry.total_cost_usd.toFixed(4)}\n\n`;
      }
      
      if (lastEntry.usage) {
        const usage = lastEntry.usage;
        if (usage.input_tokens || usage.output_tokens) {
          markdown += `**Token Usage:**\n`;
          if (usage.input_tokens) markdown += `- Input: ${usage.input_tokens.toLocaleString()}\n`;
          if (usage.cache_creation_input_tokens) markdown += `- Cache Creation: ${usage.cache_creation_input_tokens.toLocaleString()}\n`;
          if (usage.cache_read_input_tokens) markdown += `- Cache Read: ${usage.cache_read_input_tokens.toLocaleString()}\n`;
          if (usage.output_tokens) markdown += `- Output: ${usage.output_tokens.toLocaleString()}\n`;
          markdown += '\n';
        }
      }
      
      if (lastEntry.permission_denials && lastEntry.permission_denials.length > 0) {
        markdown += `**Permission Denials:** ${lastEntry.permission_denials.length}\n\n`;
      }
    }
    
    markdown += '\n## 🤖 Reasoning\n\n';
    
    // Second pass: process assistant messages in sequence
    for (const entry of logEntries) {
      if (entry.type === 'assistant' && entry.message?.content) {
        for (const content of entry.message.content) {
          if (content.type === 'text' && content.text) {
            // Add reasoning text directly (no header)
            const text = content.text.trim();
            if (text && text.length > 0) {
              markdown += text + '\n\n';
            }
          } else if (content.type === 'tool_use') {
            // Process tool use with its result
            const toolResult = toolUsePairs.get(content.id);
            const toolMarkdown = formatToolUse(content, toolResult);
            if (toolMarkdown) {
              markdown += toolMarkdown;
            }
          }
        }
      }
    }
    
    return markdown;
    
  } catch (error) {
    return `## Agent Log Summary\n\nError parsing Claude log: ${error.message}\n`;
  }
}

function formatToolUse(toolUse, toolResult) {
  const toolName = toolUse.name;
  const input = toolUse.input || {};
  
  // Skip TodoWrite except the very last one (we'll handle this separately)
  if (toolName === 'TodoWrite') {
    return ''; // Skip for now, would need global context to find the last one
  }
  
  // Helper function to determine status icon
  function getStatusIcon() {
    if (toolResult) {
      return toolResult.is_error === true ? '❌' : '✅';
    }
    return '❓'; // Unknown by default
  }
  
  let markdown = '';
  const statusIcon = getStatusIcon();
  
  switch (toolName) {
    case 'Bash':
      const command = input.command || '';
      const description = input.description || '';
      
      // Format the command to be single line
      const formattedCommand = formatBashCommand(command);
      
      if (description) {
        markdown += `${description}:\n\n`;
      }
      markdown += `${statusIcon} \`${formattedCommand}\`\n\n`;
      break;

    case 'Read':
      const filePath = input.file_path || input.path || '';
      const relativePath = filePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, ''); // Remove /home/runner/work/repo/repo/ prefix
      markdown += `${statusIcon} Read \`${relativePath}\`\n\n`;
      break;

    case 'Write':
    case 'Edit':
    case 'MultiEdit':
      const writeFilePath = input.file_path || input.path || '';
      const writeRelativePath = writeFilePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, '');
      
      markdown += `${statusIcon} Write \`${writeRelativePath}\`\n\n`;
      break;

    case 'Grep':
    case 'Glob':
      const query = input.query || input.pattern || '';
      markdown += `${statusIcon} Search for \`${truncateString(query, 80)}\`\n\n`;
      break;

    case 'LS':
      const lsPath = input.path || '';
      const lsRelativePath = lsPath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, '');
      markdown += `${statusIcon} LS: ${lsRelativePath || lsPath}\n\n`;
      break;

    default:
      // Handle MCP calls and other tools
      if (toolName.startsWith('mcp__')) {
        const mcpName = formatMcpName(toolName);
        const params = formatMcpParameters(input);
        markdown += `${statusIcon} ${mcpName}(${params})\n\n`;
      } else {
        // Generic tool formatting - show the tool name and main parameters
        const keys = Object.keys(input);
        if (keys.length > 0) {
          // Try to find the most important parameter
          const mainParam = keys.find(k => ['query', 'command', 'path', 'file_path', 'content'].includes(k)) || keys[0];
          const value = String(input[mainParam] || '');
          
          if (value) {
            markdown += `${statusIcon} ${toolName}: ${truncateString(value, 100)}\n\n`;
          } else {
            markdown += `${statusIcon} ${toolName}\n\n`;
          }
        } else {
          markdown += `${statusIcon} ${toolName}\n\n`;
        }
      }
  }
  
  return markdown;
}

function formatMcpName(toolName) {
  // Convert mcp__github__search_issues to github::search_issues
  if (toolName.startsWith('mcp__')) {
    const parts = toolName.split('__');
    if (parts.length >= 3) {
      const provider = parts[1]; // github, etc.
      const method = parts.slice(2).join('_'); // search_issues, etc.
      return `${provider}::${method}`;
    }
  }
  return toolName;
}

function formatMcpParameters(input) {
  const keys = Object.keys(input);
  if (keys.length === 0) return '';
  
  const paramStrs = [];
  for (const key of keys.slice(0, 4)) { // Show up to 4 parameters
    const value = String(input[key] || '');
    paramStrs.push(`${key}: ${truncateString(value, 40)}`);
  }
  
  if (keys.length > 4) {
    paramStrs.push('...');
  }
  
  return paramStrs.join(', ');
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
  module.exports = { parseClaudeLog, formatToolUse, formatBashCommand, truncateString };
}

main();
