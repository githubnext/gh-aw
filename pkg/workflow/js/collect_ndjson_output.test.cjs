import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import fs from 'fs';
import path from 'path';

describe('collect_ndjson_output.cjs', () => {
  let mockCore;
  let collectScript;

  beforeEach(() => {
    // Save original console before mocking
    global.originalConsole = global.console;
    
    // Mock console methods
    global.console = {
      log: vi.fn(),
      error: vi.fn()
    };

    // Mock core actions methods
    mockCore = {
      setOutput: vi.fn()
    };
    global.core = mockCore;

    // Read the script file
    const scriptPath = path.join(__dirname, 'collect_ndjson_output.cjs');
    collectScript = fs.readFileSync(scriptPath, 'utf8');

    // Make fs available globally for the evaluated script
    global.fs = fs;
  });

  afterEach(() => {
    // Clean up any test files
    const testFiles = ['/tmp/test-ndjson-output.txt'];
    testFiles.forEach(file => {
      try {
        if (fs.existsSync(file)) {
          fs.unlinkSync(file);
        }
      } catch (error) {
        // Ignore cleanup errors
      }
    });

    // Clean up globals safely - don't delete console as vitest may still need it
    if (typeof global !== 'undefined') {
      delete global.fs;
      delete global.core;
      // Restore original console instead of deleting
      if (global.originalConsole) {
        global.console = global.originalConsole;
        delete global.originalConsole;
      }
    }
  });

  it('should handle missing GITHUB_AW_SAFE_OUTPUTS environment variable', async () => {
    delete process.env.GITHUB_AW_SAFE_OUTPUTS;
    
    await eval(`(async () => { ${collectScript} })()`);
    
    expect(mockCore.setOutput).toHaveBeenCalledWith('output', '');
    expect(console.log).toHaveBeenCalledWith('GITHUB_AW_SAFE_OUTPUTS not set, no output to collect');
  });

  it('should handle missing output file', async () => {
    process.env.GITHUB_AW_SAFE_OUTPUTS = '/tmp/nonexistent-file.txt';
    
    await eval(`(async () => { ${collectScript} })()`);
    
    expect(mockCore.setOutput).toHaveBeenCalledWith('output', '');
    expect(console.log).toHaveBeenCalledWith('Output file does not exist:', '/tmp/nonexistent-file.txt');
  });

  it('should handle empty output file', async () => {
    const testFile = '/tmp/test-ndjson-output.txt';
    fs.writeFileSync(testFile, '');
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    
    await eval(`(async () => { ${collectScript} })()`);
    
    expect(mockCore.setOutput).toHaveBeenCalledWith('output', '');
    expect(console.log).toHaveBeenCalledWith('Output file is empty');
  });

  it('should validate and parse valid JSONL content', async () => {
    const testFile = '/tmp/test-ndjson-output.txt';
    const ndjsonContent = `{"type": "create-issue", "title": "Test Issue", "body": "Test body"}
{"type": "add-issue-comment", "body": "Test comment"}`;
    
    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-issue": true, "add-issue-comment": true}';
    
    await eval(`(async () => { ${collectScript} })()`);
    
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === 'output');
    expect(outputCall).toBeDefined();
    
    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(2);
    expect(parsedOutput.items[0].type).toBe('create-issue');
    expect(parsedOutput.items[1].type).toBe('add-issue-comment');
    expect(parsedOutput.errors).toHaveLength(0);
  });

  it('should reject items with unexpected output types', async () => {
    const testFile = '/tmp/test-ndjson-output.txt';
    const ndjsonContent = `{"type": "create-issue", "title": "Test Issue", "body": "Test body"}
{"type": "unexpected-type", "data": "some data"}`;
    
    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-issue": true}';
    
    await eval(`(async () => { ${collectScript} })()`);
    
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === 'output');
    expect(outputCall).toBeDefined();
    
    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1);
    expect(parsedOutput.items[0].type).toBe('create-issue');
    expect(parsedOutput.errors).toHaveLength(1);
    expect(parsedOutput.errors[0]).toContain('Unexpected output type');
  });

  it('should validate required fields for create-issue type', async () => {
    const testFile = '/tmp/test-ndjson-output.txt';
    const ndjsonContent = `{"type": "create-issue", "title": "Test Issue"}
{"type": "create-issue", "body": "Test body"}`;
    
    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-issue": true}';
    
    await eval(`(async () => { ${collectScript} })()`);
    
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === 'output');
    expect(outputCall).toBeDefined();
    
    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(0);
    expect(parsedOutput.errors).toHaveLength(2);
    expect(parsedOutput.errors[0]).toContain('requires a \'body\' string field');
    expect(parsedOutput.errors[1]).toContain('requires a \'title\' string field');
  });

  it('should validate required fields for add-issue-label type', async () => {
    const testFile = '/tmp/test-ndjson-output.txt';
    const ndjsonContent = `{"type": "add-issue-label", "labels": ["bug", "enhancement"]}
{"type": "add-issue-label", "labels": "not-an-array"}
{"type": "add-issue-label", "labels": [1, 2, 3]}`;
    
    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"add-issue-label": true}';
    
    await eval(`(async () => { ${collectScript} })()`);
    
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === 'output');
    expect(outputCall).toBeDefined();
    
    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1);
    expect(parsedOutput.items[0].labels).toEqual(['bug', 'enhancement']);
    expect(parsedOutput.errors).toHaveLength(2);
  });

  it('should handle invalid JSON lines', async () => {
    const testFile = '/tmp/test-ndjson-output.txt';
    const ndjsonContent = `{"type": "create-issue", "title": "Test Issue", "body": "Test body"}
{invalid json}
{"type": "add-issue-comment", "body": "Test comment"}`;
    
    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-issue": true, "add-issue-comment": true}';
    
    await eval(`(async () => { ${collectScript} })()`);
    
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === 'output');
    expect(outputCall).toBeDefined();
    
    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(2);
    expect(parsedOutput.errors).toHaveLength(1);
    expect(parsedOutput.errors[0]).toContain('Invalid JSON');
  });

  it('should allow multiple items of supported types up to limits', async () => {
    const testFile = '/tmp/test-ndjson-output.txt';
    const ndjsonContent = `{"type": "create-issue", "title": "First Issue", "body": "First body"}`;
    
    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-issue": true}';
    
    await eval(`(async () => { ${collectScript} })()`);
    
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === 'output');
    expect(outputCall).toBeDefined();
    
    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(1); // Both items should be allowed
    expect(parsedOutput.items[0].title).toBe('First Issue');
    expect(parsedOutput.errors).toHaveLength(0); // No errors for multiple items within limits
  });

  it('should respect max limits from config', async () => {
    const testFile = '/tmp/test-ndjson-output.txt';
    const ndjsonContent = `{"type": "create-issue", "title": "First Issue", "body": "First body"}
{"type": "create-issue", "title": "Second Issue", "body": "Second body"}
{"type": "create-issue", "title": "Third Issue", "body": "Third body"}`;
    
    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    // Set max to 2 for create-issue
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-issue": {"max": 2}}';
    
    await eval(`(async () => { ${collectScript} })()`);
    
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === 'output');
    expect(outputCall).toBeDefined();
    
    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(2); // Only first 2 items should be allowed
    expect(parsedOutput.items[0].title).toBe('First Issue');
    expect(parsedOutput.items[1].title).toBe('Second Issue');
    expect(parsedOutput.errors).toHaveLength(1); // Error for the third item exceeding max
    expect(parsedOutput.errors[0]).toContain('Too many items of type \'create-issue\'. Maximum allowed: 2');
  });

  it('should skip empty lines', async () => {
    const testFile = '/tmp/test-ndjson-output.txt';
    const ndjsonContent = `{"type": "create-issue", "title": "Test Issue", "body": "Test body"}

{"type": "add-issue-comment", "body": "Test comment"}
`;
    
    fs.writeFileSync(testFile, ndjsonContent);
    process.env.GITHUB_AW_SAFE_OUTPUTS = testFile;
    process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG = '{"create-issue": true, "add-issue-comment": true}';
    
    await eval(`(async () => { ${collectScript} })()`);
    
    const setOutputCalls = mockCore.setOutput.mock.calls;
    const outputCall = setOutputCalls.find(call => call[0] === 'output');
    expect(outputCall).toBeDefined();
    
    const parsedOutput = JSON.parse(outputCall[1]);
    expect(parsedOutput.items).toHaveLength(2);
    expect(parsedOutput.errors).toHaveLength(0);
  });
});
