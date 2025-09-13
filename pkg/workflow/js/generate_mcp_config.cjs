/**
 * Generate MCP configuration file using actions/github-script
 * Reads configuration from environment variables and generates either JSON or TOML format
 */

const fs = require('fs');
const path = require('path');

try {
  // Get configuration format (json or toml)
  const format = process.env.MCP_CONFIG_FORMAT || 'json';
  
  // Parse configuration from environment variables
  const safeOutputsConfig = process.env.MCP_SAFE_OUTPUTS_CONFIG ? JSON.parse(process.env.MCP_SAFE_OUTPUTS_CONFIG) : null;
  const githubConfig = process.env.MCP_GITHUB_CONFIG ? JSON.parse(process.env.MCP_GITHUB_CONFIG) : null;
  const playwrightConfig = process.env.MCP_PLAYWRIGHT_CONFIG ? JSON.parse(process.env.MCP_PLAYWRIGHT_CONFIG) : null;
  const customToolsConfig = process.env.MCP_CUSTOM_TOOLS_CONFIG ? JSON.parse(process.env.MCP_CUSTOM_TOOLS_CONFIG) : null;

  core.info(`Generating MCP configuration in ${format} format`);

  // Ensure the directory exists
  const configDir = '/tmp/mcp-config';
  if (!fs.existsSync(configDir)) {
    fs.mkdirSync(configDir, { recursive: true });
  }

  if (format === 'json') {
    generateJSONConfig(configDir, safeOutputsConfig, githubConfig, playwrightConfig, customToolsConfig);
  } else if (format === 'toml') {
    generateTOMLConfig(configDir, safeOutputsConfig, githubConfig, playwrightConfig, customToolsConfig);
  } else {
    throw new Error(`Unsupported format: ${format}`);
  }

  core.info('MCP configuration generated successfully');

} catch (error) {
  core.setFailed(error instanceof Error ? error.message : String(error));
}

/**
 * Generate JSON format MCP configuration (Claude format)
 */
function generateJSONConfig(configDir, safeOutputsConfig, githubConfig, playwrightConfig, customToolsConfig) {
  const config = {
    mcpServers: {}
  };

  // Add safe-outputs server if configured
  if (safeOutputsConfig && safeOutputsConfig.enabled) {
    config.mcpServers.safe_outputs = {
      command: 'node',
      args: ['/tmp/safe-outputs/mcp-server.cjs'],
      env: {
        GITHUB_AW_SAFE_OUTPUTS: '${{ env.GITHUB_AW_SAFE_OUTPUTS }}',
        GITHUB_AW_SAFE_OUTPUTS_CONFIG: '${{ env.GITHUB_AW_SAFE_OUTPUTS_CONFIG }}'
      }
    };
  }

  // Add GitHub server if configured
  if (githubConfig) {
    config.mcpServers.github = generateGitHubJSONConfig(githubConfig);
  }

  // Add Playwright server if configured
  if (playwrightConfig) {
    config.mcpServers.playwright = generatePlaywrightJSONConfig(playwrightConfig);
  }

  // Add custom MCP tools
  if (customToolsConfig) {
    for (const [toolName, toolConfig] of Object.entries(customToolsConfig)) {
      config.mcpServers[toolName] = generateCustomToolJSONConfig(toolConfig);
    }
  }

  const configPath = path.join(configDir, 'mcp-servers.json');
  fs.writeFileSync(configPath, JSON.stringify(config, null, 2));
  core.info(`JSON MCP configuration written to ${configPath}`);
}

/**
 * Generate TOML format MCP configuration (Codex format)
 */
function generateTOMLConfig(configDir, safeOutputsConfig, githubConfig, playwrightConfig, customToolsConfig) {
  let tomlContent = '';

  // Add history configuration to disable persistence
  tomlContent += '[history]\n';
  tomlContent += 'persistence = "none"\n\n';

  // Add safe-outputs server if configured
  if (safeOutputsConfig && safeOutputsConfig.enabled) {
    tomlContent += '[mcp_servers.safe_outputs]\n';
    tomlContent += 'command = "node"\n';
    tomlContent += 'args = [\n';
    tomlContent += '  "/tmp/safe-outputs/mcp-server.cjs",\n';
    tomlContent += ']\n';
    tomlContent += 'env = { "GITHUB_AW_SAFE_OUTPUTS" = "${{ env.GITHUB_AW_SAFE_OUTPUTS }}", "GITHUB_AW_SAFE_OUTPUTS_CONFIG" = "${{ env.GITHUB_AW_SAFE_OUTPUTS_CONFIG }}" }\n\n';
  }

  // Add GitHub server if configured
  if (githubConfig) {
    tomlContent += generateGitHubTOMLConfig(githubConfig);
  }

  // Add Playwright server if configured
  if (playwrightConfig) {
    tomlContent += generatePlaywrightTOMLConfig(playwrightConfig);
  }

  // Add custom MCP tools
  if (customToolsConfig) {
    for (const [toolName, toolConfig] of Object.entries(customToolsConfig)) {
      tomlContent += generateCustomToolTOMLConfig(toolName, toolConfig);
    }
  }

  const configPath = path.join(configDir, 'config.toml');
  fs.writeFileSync(configPath, tomlContent);
  core.info(`TOML MCP configuration written to ${configPath}`);
}

/**
 * Generate GitHub MCP server configuration for JSON format
 */
function generateGitHubJSONConfig(githubConfig) {
  const dockerImageVersion = githubConfig.dockerImageVersion || 'latest';
  
  return {
    command: 'docker',
    args: [
      'run',
      '-i',
      '--rm',
      '-e', 'GITHUB_TOKEN',
      `ghcr.io/modelcontextprotocol/servers/github:${dockerImageVersion}`
    ]
  };
}

/**
 * Generate GitHub MCP server configuration for TOML format
 */
function generateGitHubTOMLConfig(githubConfig) {
  const dockerImageVersion = githubConfig.dockerImageVersion || 'latest';
  
  let tomlContent = '[mcp_servers.github]\n';
  tomlContent += 'command = "docker"\n';
  tomlContent += 'args = [\n';
  tomlContent += '  "run",\n';
  tomlContent += '  "-i",\n';
  tomlContent += '  "--rm",\n';
  tomlContent += '  "-e", "GITHUB_TOKEN",\n';
  tomlContent += `  "ghcr.io/modelcontextprotocol/servers/github:${dockerImageVersion}"\n`;
  tomlContent += ']\n\n';
  
  return tomlContent;
}

/**
 * Generate Playwright MCP server configuration for JSON format
 */
function generatePlaywrightJSONConfig(playwrightConfig) {
  const dockerImageVersion = playwrightConfig.dockerImageVersion || 'latest';
  const allowedDomains = playwrightConfig.allowedDomains || [];
  
  const config = {
    command: 'docker',
    args: [
      'compose',
      '-f', `docker-compose-playwright.yml`,
      'run',
      '--rm',
      'playwright'
    ]
  };

  if (allowedDomains.length > 0) {
    config.env = {
      PLAYWRIGHT_ALLOWED_DOMAINS: allowedDomains.join(',')
    };
  }

  return config;
}

/**
 * Generate Playwright MCP server configuration for TOML format
 */
function generatePlaywrightTOMLConfig(playwrightConfig) {
  const dockerImageVersion = playwrightConfig.dockerImageVersion || 'latest';
  const allowedDomains = playwrightConfig.allowedDomains || [];
  
  let tomlContent = '[mcp_servers.playwright]\n';
  tomlContent += 'command = "docker"\n';
  tomlContent += 'args = [\n';
  tomlContent += '  "compose",\n';
  tomlContent += '  "-f", "docker-compose-playwright.yml",\n';
  tomlContent += '  "run",\n';
  tomlContent += '  "--rm",\n';
  tomlContent += '  "playwright"\n';
  tomlContent += ']\n';

  if (allowedDomains.length > 0) {
    tomlContent += `env = { "PLAYWRIGHT_ALLOWED_DOMAINS" = "${allowedDomains.join(',')}" }\n`;
  }
  
  tomlContent += '\n';
  return tomlContent;
}

/**
 * Generate custom MCP tool configuration for JSON format
 */
function generateCustomToolJSONConfig(toolConfig) {
  const config = {};
  
  if (toolConfig.command) {
    config.command = toolConfig.command;
  }
  
  if (toolConfig.args) {
    config.args = toolConfig.args;
  }
  
  if (toolConfig.env) {
    config.env = toolConfig.env;
  }
  
  if (toolConfig.url) {
    config.url = toolConfig.url;
  }
  
  if (toolConfig.headers) {
    config.headers = toolConfig.headers;
  }
  
  return config;
}

/**
 * Generate custom MCP tool configuration for TOML format
 */
function generateCustomToolTOMLConfig(toolName, toolConfig) {
  let tomlContent = `[mcp_servers.${toolName}]\n`;
  
  if (toolConfig.command) {
    tomlContent += `command = "${toolConfig.command}"\n`;
  }
  
  if (toolConfig.args && Array.isArray(toolConfig.args)) {
    tomlContent += 'args = [\n';
    for (const arg of toolConfig.args) {
      tomlContent += `  "${arg}",\n`;
    }
    tomlContent += ']\n';
  }
  
  if (toolConfig.env && typeof toolConfig.env === 'object') {
    tomlContent += 'env = { ';
    const envEntries = Object.entries(toolConfig.env);
    for (let i = 0; i < envEntries.length; i++) {
      const [key, value] = envEntries[i];
      tomlContent += `"${key}" = "${value}"`;
      if (i < envEntries.length - 1) {
        tomlContent += ', ';
      }
    }
    tomlContent += ' }\n';
  }
  
  tomlContent += '\n';
  return tomlContent;
}