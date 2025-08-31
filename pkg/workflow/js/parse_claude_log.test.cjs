import { describe, it, expect, beforeEach, vi } from 'vitest';
import fs from 'fs';
import path from 'path';

// Mock the @actions/core module
const mockCore = {
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn()
  }
};

// Set up global variables
global.core = mockCore;

describe('parse_claude_log.cjs', () => {
  let parseClaudeLogScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();
    
    // Reset environment variables
    delete process.env.AGENT_LOG_FILE;
    
    // Load the script content
    const scriptPath = path.join(__dirname, 'parse_claude_log.cjs');
    const scriptContent = fs.readFileSync(scriptPath, 'utf8');
    
    // Execute the script to get the exports
    const scriptFunction = new Function('require', 'module', 'exports', 'process', 'console', 'global', 'core', scriptContent);
    const mockModule = { exports: {} };
    const mockRequire = (name) => {
      if (name === '@actions/core') return mockCore;
      if (name === 'fs') return fs;
      return {};
    };
    
    scriptFunction(mockRequire, mockModule, mockModule.exports, process, console, global, mockCore);
    parseClaudeLogScript = mockModule.exports;
  });

  describe('parseClaudeLog', () => {
    it('should handle empty input', () => {
      const result = parseClaudeLogScript.parseClaudeLog('');
      expect(result).toContain('## 🤖 Agent Reasoning Sequence');
      expect(result).toContain('Log format not recognized');
    });

    it('should handle malformed JSON', () => {
      const result = parseClaudeLogScript.parseClaudeLog('not json');
      expect(result).toContain('Error parsing Claude log');
    });

    it('should handle empty array', () => {
      const result = parseClaudeLogScript.parseClaudeLog('[]');
      expect(result).toContain('## 🤖 Agent Reasoning Sequence');
      expect(result).toContain('No tool usage found');
    });

    it('should parse basic Claude log with tool_use', () => {
      const logData = [
        {
          "type": "assistant",
          "message": {
            "role": "assistant",
            "content": [
              {
                "type": "text",
                "text": "I'll help you check the environment."
              },
              {
                "type": "tool_use",
                "id": "toolu_01ABC123",
                "name": "Bash",
                "input": {
                  "command": "echo $GITHUB_STEP_SUMMARY",
                  "description": "Get GitHub step summary file path"
                }
              }
            ]
          }
        },
        {
          "type": "user",
          "message": {
            "content": [
              {
                "tool_use_id": "toolu_01ABC123",
                "type": "tool_result",
                "content": "/home/runner/work/_temp/_runner_file_commands/step_summary_123.txt",
                "is_error": false
              }
            ]
          }
        }
      ];

      const result = parseClaudeLogScript.parseClaudeLog(JSON.stringify(logData));
      
      expect(result).toContain('## 🤖 Agent Reasoning Sequence');
      expect(result).toContain('💭 Reasoning');
      expect(result).toContain('I\'ll help you check the environment');
      expect(result).toContain('🔧 Bash');
      expect(result).toContain('echo $GITHUB_STEP_SUMMARY');
      expect(result).toContain('Get GitHub step summary file path');
      expect(result).toContain('✅ Success');
    });

    it('should parse Read tool usage', () => {
      const logData = [
        {
          "type": "assistant",
          "message": {
            "content": [
              {
                "type": "tool_use",
                "id": "toolu_01READ123",
                "name": "Read",
                "input": {
                  "file_path": "/home/runner/work/repo/repo/src/main.go"
                }
              }
            ]
          }
        },
        {
          "type": "user",
          "message": {
            "content": [
              {
                "tool_use_id": "toolu_01READ123",
                "type": "tool_result",
                "content": "package main\n\nfunc main() {\n    // code here\n}",
                "is_error": false
              }
            ]
          }
        }
      ];

      const result = parseClaudeLogScript.parseClaudeLog(JSON.stringify(logData));
      
      expect(result).toContain('🔧 Read');
      expect(result).toContain('src/main.go'); // Should show relative path
      expect(result).toContain('package main');
    });

    it('should handle tool errors', () => {
      const logData = [
        {
          "type": "assistant",
          "message": {
            "content": [
              {
                "type": "tool_use",
                "id": "toolu_01ERROR123",
                "name": "Bash",
                "input": {
                  "command": "invalid-command",
                  "description": "This will fail"
                }
              }
            ]
          }
        },
        {
          "type": "user",
          "message": {
            "content": [
              {
                "tool_use_id": "toolu_01ERROR123",
                "type": "tool_result",
                "content": "command not found: invalid-command",
                "is_error": true
              }
            ]
          }
        }
      ];

      const result = parseClaudeLogScript.parseClaudeLog(JSON.stringify(logData));
      
      expect(result).toContain('❌ Error');
      expect(result).toContain('command not found');
    });
  });

  describe('formatToolUse', () => {
    it('should format Bash tool correctly', () => {
      const toolUse = {
        name: 'Bash',
        input: {
          command: 'ls -la',
          description: 'List files'
        }
      };
      const toolResult = {
        is_error: false,
        content: 'total 4\ndrwxr-xr-x 2 user user 4096 Jan 1 12:00 .'
      };

      const result = parseClaudeLogScript.formatToolUse(toolUse, toolResult, 1);
      
      expect(result).toContain('1. 🔧 Bash');
      expect(result).toContain('**Command:** `ls -la`');
      expect(result).toContain('**Description:** List files');
      expect(result).toContain('✅ Success');
    });

    it('should skip TodoWrite tools', () => {
      const toolUse = {
        name: 'TodoWrite',
        input: { content: 'some todo content' }
      };

      const result = parseClaudeLogScript.formatToolUse(toolUse, null, 1);
      expect(result).toBe('');
    });
  });

  describe('truncateString', () => {
    it('should truncate long strings', () => {
      const longString = 'a'.repeat(100);
      const result = parseClaudeLogScript.truncateString(longString, 50);
      expect(result).toHaveLength(53); // 50 + "..."
      expect(result).toEndWith('...');
    });

    it('should not truncate short strings', () => {
      const shortString = 'hello world';
      const result = parseClaudeLogScript.truncateString(shortString, 50);
      expect(result).toBe(shortString);
    });

    it('should handle empty strings', () => {
      const result = parseClaudeLogScript.truncateString('', 50);
      expect(result).toBe('');
    });

    it('should handle null/undefined', () => {
      expect(parseClaudeLogScript.truncateString(null, 50)).toBe('');
      expect(parseClaudeLogScript.truncateString(undefined, 50)).toBe('');
    });
  });

  describe('integration with sample log', () => {
    it('should process the sample Claude log from test data', () => {
      // Read the actual sample log file
      const sampleLogPath = path.join(__dirname, '../../test_data/sample_claude_log.txt');
      
      if (fs.existsSync(sampleLogPath)) {
        const sampleLogContent = fs.readFileSync(sampleLogPath, 'utf8');
        const result = parseClaudeLogScript.parseClaudeLog(sampleLogContent);
        
        // Basic assertions about the parsed content
        expect(result).toContain('## 🤖 Agent Reasoning Sequence');
        expect(result.length).toBeGreaterThan(100); // Should produce meaningful output
        
        // Should contain some expected patterns from Claude logs
        // (We'll be flexible here since the exact content may vary)
        const hasToolUsage = result.includes('🔧') || result.includes('No tool usage found');
        expect(hasToolUsage).toBe(true);
      } else {
        console.log('Sample Claude log not found, skipping integration test');
      }
    });
  });
});
