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
    
    let markdown = '## 🤖 Agent Reasoning Sequence\n\n';
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
            
            // Add to command summary (only include relevant tools)
            if (toolName === 'Bash') {
              const formattedCommand = formatBashCommand(input.command || '');
              commandSummary.push(`\`${formattedCommand}\``);
            } else if (toolName.startsWith('mcp__')) {
              const mcpName = formatMcpName(toolName);
              commandSummary.push(`\`${mcpName}(...)\``);
            }
          }
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
  
  let markdown = '';
  
  switch (toolName) {
    case 'Bash':
      const command = input.command || '';
      const description = input.description || '';
      
      if (description) {
        markdown += `${description}:\n`;
      }
      markdown += '```bash\n';
      markdown += `> ${command}\n`;
      
      // Add result/output if available
      if (toolResult && toolResult.content) {
        const output = String(toolResult.content).trim();
        if (output) {
          markdown += truncateString(output, 200) + '\n';
        }
      }
      markdown += '```\n\n';
      break;

    case 'Read':
      const filePath = input.file_path || input.path || '';
      const relativePath = filePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, ''); // Remove /home/runner/work/repo/repo/ prefix
      markdown += `Read \`${relativePath}\`\n\n`;
      break;

    case 'Write':
    case 'Edit':
    case 'MultiEdit':
      const writeFilePath = input.file_path || input.path || '';
      const writeRelativePath = writeFilePath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, '');
      const writeContent = input.content || input.new_string || '';
      
      markdown += `Write \`${writeRelativePath}\`\n`;
      if (writeContent) {
        markdown += '```\n';
        markdown += truncateString(writeContent, 300) + '\n';
        markdown += '```\n';
      }
      markdown += '\n';
      break;

    case 'Grep':
    case 'Glob':
      const query = input.query || input.pattern || '';
      markdown += `Search for \`${truncateString(query, 80)}\`\n\n`;
      break;

    case 'LS':
      const lsPath = input.path || '';
      const lsRelativePath = lsPath.replace(/^\/[^\/]*\/[^\/]*\/[^\/]*\/[^\/]*\//, '');
      markdown += `LS: ${lsRelativePath || lsPath}\n\n`;
      break;

    default:
      // Handle MCP calls and other tools
      if (toolName.startsWith('mcp__')) {
        const mcpName = formatMcpName(toolName);
        const params = formatMcpParameters(input);
        markdown += `${mcpName}(${params})\n\n`;
      } else {
        // Generic tool formatting - show the tool name and main parameters
        const keys = Object.keys(input);
        if (keys.length > 0) {
          // Try to find the most important parameter
          const mainParam = keys.find(k => ['query', 'command', 'path', 'file_path', 'content'].includes(k)) || keys[0];
          const value = String(input[mainParam] || '');
          
          if (value) {
            markdown += `${toolName}: ${truncateString(value, 100)}\n\n`;
          } else {
            markdown += `${toolName}\n\n`;
          }
        } else {
          markdown += `${toolName}\n\n`;
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
