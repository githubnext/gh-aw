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

describe('parse_codex_log.cjs', () => {
  let parseCodexLogScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();
    
    // Reset environment variables
    delete process.env.AGENT_LOG_FILE;
    
    // Load the script content
    const scriptPath = path.join(__dirname, 'parse_codex_log.cjs');
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
    parseCodexLogScript = mockModule.exports;
  });

  describe('parseCodexLog', () => {
    it('should handle empty input', () => {
      const result = parseCodexLogScript.parseCodexLog('');
      expect(result).toContain('## 🤖 Agent Reasoning Sequence');
      expect(result).toContain('Log parsing in progress');
    });

    it('should parse basic Codex log format', () => {
      const logContent = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0 (research preview)
--------
workdir: /home/runner/work/gh-aw/gh-aw
model: o4-mini
provider: openai
approval: never
--------
[2025-08-31T12:37:08] User instructions:
# Test Codex

This is a test workflow.

[2025-08-31T12:37:15] Starting analysis...
I need to analyze the pull request and provide feedback.

[2025-08-31T12:37:20] Fetching PR data
Using GitHub API to get pull request information.

[2025-08-31T12:37:25] Analysis complete
Generated comprehensive review with key findings.`;

      const result = parseCodexLogScript.parseCodexLog(logContent);
      
      expect(result).toContain('## 🤖 Agent Reasoning Sequence');
      expect(result).toContain('🔧 Starting analysis...');
      expect(result).toContain('🔧 Fetching PR data');
      expect(result).toContain('🔧 Analysis complete');
      expect(result).toContain('analyze the pull request');
      expect(result).toContain('GitHub API');
    });

    it('should skip metadata lines', () => {
      const logContent = `[2025-08-31T12:37:08] OpenAI Codex v0.27.0 (research preview)
--------
workdir: /home/runner/work/gh-aw/gh-aw
model: o4-mini
provider: openai
approval: never
sandbox: workspace-write [workdir, /tmp, $TMPDIR]
reasoning effort: medium
reasoning summaries: auto
--------
[2025-08-31T12:37:15] Actual reasoning content
This is the important content that should be included.`;

      const result = parseCodexLogScript.parseCodexLog(logContent);
      
      expect(result).not.toContain('workdir:');
      expect(result).not.toContain('model:');
      expect(result).not.toContain('provider:');
      expect(result).not.toContain('sandbox:');
      expect(result).toContain('Actual reasoning content');
      expect(result).toContain('important content');
    });

    it('should skip user instructions section', () => {
      const logContent = `[2025-08-31T12:37:08] User instructions:
# Very Long Instructions

This is a very long instruction section that should be skipped
because it's usually quite lengthy and not part of the reasoning.

Multiple paragraphs of instructions...

More instructions...

[2025-08-31T12:37:15] Agent reasoning starts
This is the actual agent reasoning that should be included.`;

      const result = parseCodexLogScript.parseCodexLog(logContent);
      
      expect(result).not.toContain('Very Long Instructions');
      expect(result).not.toContain('lengthy and not part');
      expect(result).toContain('Agent reasoning starts');
      expect(result).toContain('actual agent reasoning');
    });

    it('should handle minimal output', () => {
      const logContent = `[2025-08-31T12:37:08] Short
OK`;

      const result = parseCodexLogScript.parseCodexLog(logContent);
      
      expect(result).toContain('Log parsing in progress or minimal output detected');
    });
  });

  describe('formatCodexSection', () => {
    it('should format sections correctly', () => {
      const title = 'Analyzing code';
      const content = ['I need to examine', 'the code structure', 'and identify issues'];
      
      const result = parseCodexLogScript.formatCodexSection(title, content);
      
      expect(result).toContain('### 🔧 Analyzing code');
      expect(result).toContain('I need to examine the code structure and identify issues');
    });

    it('should handle empty content', () => {
      const result = parseCodexLogScript.formatCodexSection('Title', []);
      expect(result).toBe('');
    });

    it('should handle empty title', () => {
      const result = parseCodexLogScript.formatCodexSection('', ['content']);
      expect(result).toBe('');
    });

    it('should truncate long content', () => {
      const title = 'Long section';
      const longContent = ['a'.repeat(200), 'b'.repeat(200)]; // Very long content
      
      const result = parseCodexLogScript.formatCodexSection(title, longContent);
      
      expect(result).toContain('### 🔧 Long section');
      expect(result).toContain('...');
      expect(result.length).toBeLessThan(500); // Should be truncated
    });
  });

  describe('cleanCodexMarkdown', () => {
    it('should remove excessive whitespace', () => {
      const input = 'Line 1\n\n\n\n\nLine 2\n\n\n\nLine 3';
      const result = parseCodexLogScript.cleanCodexMarkdown(input);
      
      expect(result).toBe('Line 1\n\nLine 2\n\nLine 3');
    });

    it('should limit header depth', () => {
      const input = '# H1\n## H2\n### H3\n#### H4\n##### H5\n###### H6\n####### H7';
      const result = parseCodexLogScript.cleanCodexMarkdown(input);
      
      expect(result).toContain('# H1');
      expect(result).toContain('## H2');
      expect(result).toContain('### H3');
      expect(result).toContain('#### H4');
      expect(result).toContain('#### H5'); // Limited to H4
      expect(result).toContain('#### H6'); // Limited to H4
      expect(result).toContain('#### H7'); // Limited to H4
    });
  });

  describe('truncateString', () => {
    it('should truncate long strings', () => {
      const longString = 'x'.repeat(400);
      const result = parseCodexLogScript.truncateString(longString, 300);
      expect(result).toHaveLength(303); // 300 + "..."
      expect(result).toEndWith('...');
    });

    it('should not truncate short strings', () => {
      const shortString = 'hello world';
      const result = parseCodexLogScript.truncateString(shortString, 300);
      expect(result).toBe(shortString);
    });

    it('should handle empty strings', () => {
      const result = parseCodexLogScript.truncateString('', 300);
      expect(result).toBe('');
    });

    it('should handle null/undefined', () => {
      expect(parseCodexLogScript.truncateString(null, 300)).toBe('');
      expect(parseCodexLogScript.truncateString(undefined, 300)).toBe('');
    });
  });

  describe('integration with sample log', () => {
    it('should process the sample Codex log from test data', () => {
      // Read the actual sample log file
      const sampleLogPath = path.join(__dirname, '../../test_data/sample_codex_log.txt');
      
      if (fs.existsSync(sampleLogPath)) {
        const sampleLogContent = fs.readFileSync(sampleLogPath, 'utf8');
        const result = parseCodexLogScript.parseCodexLog(sampleLogContent);
        
        // Basic assertions about the parsed content
        expect(result).toContain('## 🤖 Agent Reasoning Sequence');
        expect(result.length).toBeGreaterThan(50); // Should produce some output
        
        // Should not contain metadata
        expect(result).not.toContain('workdir:');
        expect(result).not.toContain('model:');
        expect(result).not.toContain('provider:');
        
        // Should contain some reasoning sections or minimal output message
        const hasContent = result.includes('🔧') || result.includes('Log parsing in progress');
        expect(hasContent).toBe(true);
      } else {
        console.log('Sample Codex log not found, skipping integration test');
      }
    });
  });
});
