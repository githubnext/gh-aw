/**
 * Tests for generate_mcp_config.cjs
 */

import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import fs from 'fs';
import path from 'path';
import { exec } from 'child_process';
import { promisify } from 'util';

const execAsync = promisify(exec);

// Mock @actions/core
global.core = {
  info: vi.fn(),
  error: vi.fn(),
  warning: vi.fn(),
  debug: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  getInput: vi.fn()
};

describe('generate_mcp_config.cjs', () => {
  const testConfigDir = '/tmp/test-mcp-config';
  
  beforeEach(() => {
    // Clean up any existing test directory
    if (fs.existsSync(testConfigDir)) {
      fs.rmSync(testConfigDir, { recursive: true });
    }
    
    // Reset all mocks
    vi.clearAllMocks();
    
    // Reset environment variables
    delete process.env.MCP_CONFIG_FORMAT;
    delete process.env.MCP_SAFE_OUTPUTS_CONFIG;
    delete process.env.MCP_GITHUB_CONFIG;
    delete process.env.MCP_PLAYWRIGHT_CONFIG;
    delete process.env.MCP_CUSTOM_TOOLS_CONFIG;
  });

  afterEach(() => {
    // Clean up test directory
    if (fs.existsSync(testConfigDir)) {
      fs.rmSync(testConfigDir, { recursive: true });
    }
  });

  async function runScript(env = {}) {
    const scriptPath = path.join(__dirname, 'generate_mcp_config.cjs');
    
    // Create a wrapper script that provides the core mock
    const wrapperScript = `
// Mock @actions/core
global.core = {
  info: () => {},
  error: () => {},
  warning: () => {},
  debug: () => {},
  setFailed: (message) => { throw new Error(message); },
  setOutput: () => {},
  exportVariable: () => {},
  getInput: () => {}
};

// Load the actual script
${fs.readFileSync(scriptPath, 'utf8').replace(/\/tmp\/mcp-config/g, testConfigDir)}
`;
    
    const tempScriptPath = path.join('/tmp', `test-script-${Date.now()}.cjs`);
    fs.writeFileSync(tempScriptPath, wrapperScript);

    const testEnv = {
      ...process.env,
      ...env
    };

    try {
      const result = await execAsync(`node ${tempScriptPath}`, { env: testEnv });
      return { success: true, stdout: result.stdout, stderr: result.stderr };
    } catch (error) {
      return { success: false, error: error.message, stdout: error.stdout, stderr: error.stderr };
    } finally {
      if (fs.existsSync(tempScriptPath)) {
        fs.unlinkSync(tempScriptPath);
      }
    }
  }

  describe('JSON format generation (Claude)', () => {
    test('should generate basic JSON config with safe-outputs only', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_SAFE_OUTPUTS_CONFIG: JSON.stringify({ enabled: true })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      expect(fs.existsSync(configPath)).toBe(true);

      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      expect(config).toEqual({
        mcpServers: {
          safe_outputs: {
            command: 'node',
            args: ['/tmp/safe-outputs/mcp-server.cjs'],
            env: {
              GITHUB_AW_SAFE_OUTPUTS: '${{ env.GITHUB_AW_SAFE_OUTPUTS }}',
              GITHUB_AW_SAFE_OUTPUTS_CONFIG: '${{ env.GITHUB_AW_SAFE_OUTPUTS_CONFIG }}'
            }
          }
        }
      });
    });

    test('should generate JSON config with GitHub tool', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_GITHUB_CONFIG: JSON.stringify({ dockerImageVersion: 'v1.0.0' })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      
      expect(config.mcpServers.github).toEqual({
        command: 'docker',
        args: [
          'run',
          '-i',
          '--rm',
          '-e', 'GITHUB_TOKEN',
          'ghcr.io/modelcontextprotocol/servers/github:v1.0.0'
        ]
      });
    });

    test('should generate JSON config with Playwright tool', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_PLAYWRIGHT_CONFIG: JSON.stringify({ 
          dockerImageVersion: 'v1.41.0',
          allowedDomains: ['github.com', '*.github.com']
        })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      
      expect(config.mcpServers.playwright).toEqual({
        command: 'docker',
        args: [
          'compose',
          '-f', 'docker-compose-playwright.yml',
          'run',
          '--rm',
          'playwright'
        ],
        env: {
          PLAYWRIGHT_ALLOWED_DOMAINS: 'github.com,*.github.com'
        }
      });
    });

    test('should generate JSON config with custom MCP tools', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_CUSTOM_TOOLS_CONFIG: JSON.stringify({
          'custom-tool': {
            command: 'custom-command',
            args: ['arg1', 'arg2'],
            env: { 'CUSTOM_VAR': 'value' }
          },
          'http-tool': {
            url: 'https://example.com/mcp',
            headers: { 'Authorization': 'Bearer token' }
          }
        })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      
      expect(config.mcpServers['custom-tool']).toEqual({
        command: 'custom-command',
        args: ['arg1', 'arg2'],
        env: { 'CUSTOM_VAR': 'value' }
      });

      expect(config.mcpServers['http-tool']).toEqual({
        url: 'https://example.com/mcp',
        headers: { 'Authorization': 'Bearer token' }
      });
    });

    test('should generate complete JSON config with all tools', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_SAFE_OUTPUTS_CONFIG: JSON.stringify({ enabled: true }),
        MCP_GITHUB_CONFIG: JSON.stringify({ dockerImageVersion: 'latest' }),
        MCP_PLAYWRIGHT_CONFIG: JSON.stringify({ allowedDomains: ['example.com'] }),
        MCP_CUSTOM_TOOLS_CONFIG: JSON.stringify({
          'my-tool': { command: 'node', args: ['script.js'] }
        })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      
      expect(Object.keys(config.mcpServers)).toEqual(['safe_outputs', 'github', 'playwright', 'my-tool']);
    });
  });

  describe('TOML format generation (Codex)', () => {
    test('should generate basic TOML config with safe-outputs only', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'toml',
        MCP_SAFE_OUTPUTS_CONFIG: JSON.stringify({ enabled: true })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'config.toml');
      expect(fs.existsSync(configPath)).toBe(true);

      const content = fs.readFileSync(configPath, 'utf8');
      expect(content).toContain('[history]');
      expect(content).toContain('persistence = "none"');
      expect(content).toContain('[mcp_servers.safe_outputs]');
      expect(content).toContain('command = "node"');
      expect(content).toContain('"/tmp/safe-outputs/mcp-server.cjs"');
    });

    test('should generate TOML config with GitHub tool', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'toml',
        MCP_GITHUB_CONFIG: JSON.stringify({ dockerImageVersion: 'v1.0.0' })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'config.toml');
      const content = fs.readFileSync(configPath, 'utf8');
      
      expect(content).toContain('[mcp_servers.github]');
      expect(content).toContain('command = "docker"');
      expect(content).toContain('ghcr.io/modelcontextprotocol/servers/github:v1.0.0');
    });

    test('should generate TOML config with Playwright tool', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'toml',
        MCP_PLAYWRIGHT_CONFIG: JSON.stringify({ 
          allowedDomains: ['github.com', 'example.com']
        })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'config.toml');
      const content = fs.readFileSync(configPath, 'utf8');
      
      expect(content).toContain('[mcp_servers.playwright]');
      expect(content).toContain('docker-compose-playwright.yml');
      expect(content).toContain('PLAYWRIGHT_ALLOWED_DOMAINS');
      expect(content).toContain('github.com,example.com');
    });

    test('should generate TOML config with custom MCP tools', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'toml',
        MCP_CUSTOM_TOOLS_CONFIG: JSON.stringify({
          'custom-tool': {
            command: 'python',
            args: ['script.py', '--arg'],
            env: { 'VAR1': 'value1', 'VAR2': 'value2' }
          }
        })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'config.toml');
      const content = fs.readFileSync(configPath, 'utf8');
      
      expect(content).toContain('[mcp_servers.custom-tool]');
      expect(content).toContain('command = "python"');
      expect(content).toContain('"script.py"');
      expect(content).toContain('"--arg"');
      expect(content).toContain('"VAR1" = "value1"');
      expect(content).toContain('"VAR2" = "value2"');
    });
  });

  describe('Environment variable parsing', () => {
    test('should handle missing environment variables gracefully', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json'
        // No other config variables set
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      
      expect(config).toEqual({
        mcpServers: {}
      });
    });

    test('should handle invalid JSON in environment variables', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_SAFE_OUTPUTS_CONFIG: 'invalid-json'
      };

      const result = await runScript(env);
      expect(result.success).toBe(false);
      expect(result.error).toContain('Unexpected token');
    });

    test('should default to JSON format when format not specified', async () => {
      const env = {
        MCP_SAFE_OUTPUTS_CONFIG: JSON.stringify({ enabled: true })
        // MCP_CONFIG_FORMAT not set
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      expect(fs.existsSync(configPath)).toBe(true);
    });

    test('should fail with unsupported format', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'xml'
      };

      const result = await runScript(env);
      expect(result.success).toBe(false);
      expect(result.error).toContain('Unsupported format: xml');
    });
  });

  describe('Directory creation', () => {
    test('should create config directory if it does not exist', async () => {
      // Ensure directory doesn't exist
      expect(fs.existsSync(testConfigDir)).toBe(false);

      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_SAFE_OUTPUTS_CONFIG: JSON.stringify({ enabled: true })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);
      expect(fs.existsSync(testConfigDir)).toBe(true);
    });
  });

  describe('Edge cases', () => {
    test('should handle safe-outputs config with enabled=false', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_SAFE_OUTPUTS_CONFIG: JSON.stringify({ enabled: false })
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      
      expect(config.mcpServers.safe_outputs).toBeUndefined();
    });

    test('should handle GitHub config with default image version', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_GITHUB_CONFIG: JSON.stringify({}) // No dockerImageVersion specified
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      
      expect(config.mcpServers.github.args).toContain('ghcr.io/modelcontextprotocol/servers/github:latest');
    });

    test('should handle Playwright config without allowed domains', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_PLAYWRIGHT_CONFIG: JSON.stringify({}) // No allowedDomains specified
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      
      expect(config.mcpServers.playwright.env).toBeUndefined();
    });

    test('should handle empty custom tools config', async () => {
      const env = {
        MCP_CONFIG_FORMAT: 'json',
        MCP_CUSTOM_TOOLS_CONFIG: JSON.stringify({}) // Empty object
      };

      const result = await runScript(env);
      expect(result.success).toBe(true);

      const configPath = path.join(testConfigDir, 'mcp-servers.json');
      const config = JSON.parse(fs.readFileSync(configPath, 'utf8'));
      
      expect(config).toEqual({
        mcpServers: {}
      });
    });
  });
});