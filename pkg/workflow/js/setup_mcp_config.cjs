/**
 * Setup MCP servers configuration for agentic workflows
 * This script generates the mcp-servers.json file based on workflow tool configuration
 */

function main() {
  const fs = require("fs");
  const path = require("path");

  try {
    // Get the tools configuration from environment variable
    const toolsConfigStr = process.env.GITHUB_AW_TOOLS_CONFIG;
    if (!toolsConfigStr) {
      core.setFailed("GITHUB_AW_TOOLS_CONFIG environment variable not set");
      return;
    }

    let toolsConfig;
    try {
      toolsConfig = JSON.parse(toolsConfigStr);
    } catch (error) {
      core.setFailed(`Failed to parse GITHUB_AW_TOOLS_CONFIG: ${error.message}`);
      return;
    }

    const mcpServersConfig = generateMCPConfig(toolsConfig);

    // Ensure the output directory exists
    const outputDir = "/tmp/mcp-config";
    fs.mkdirSync(outputDir, { recursive: true });

    // Write the MCP servers configuration
    const outputPath = path.join(outputDir, "mcp-servers.json");
    fs.writeFileSync(outputPath, JSON.stringify(mcpServersConfig, null, 2));

    core.info(`Generated MCP configuration at: ${outputPath}`);
    core.info(`Configuration: ${JSON.stringify(mcpServersConfig, null, 2)}`);

    // Set output for subsequent steps
    core.setOutput("mcp_config_path", outputPath);
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    core.setFailed(`Failed to setup MCP configuration: ${errorMessage}`);
  }
}

/**
 * Generate MCP server configuration from tools configuration
 * @param {Object} toolsConfig - The tools configuration object
 * @returns {Object} The MCP servers configuration
 */
function generateMCPConfig(toolsConfig) {
  const mcpServers = {};

  for (const [toolName, toolConfig] of Object.entries(toolsConfig)) {
    if (!toolConfig || typeof toolConfig !== "object") {
      continue;
    }

    // Handle built-in tools
    if (toolName === "github") {
      mcpServers.github = generateGitHubMCPConfig(toolConfig);
    } else if (toolName === "playwright") {
      mcpServers.playwright = generatePlaywrightMCPConfig(toolConfig);
    } else if (toolName === "cache-memory") {
      mcpServers["cache-memory"] = generateCacheMemoryMCPConfig(toolConfig);
    } else {
      // Handle custom MCP tools
      const customConfig = generateCustomMCPConfig(toolName, toolConfig);
      if (customConfig) {
        mcpServers[toolName] = customConfig;
      }
    }
  }

  return { mcpServers };
}

/**
 * Generate GitHub MCP server configuration
 * @param {Object} githubConfig - GitHub tool configuration
 * @returns {Object} GitHub MCP server configuration
 */
function generateGitHubMCPConfig(githubConfig) {
  const dockerImageVersion = githubConfig.docker_image_version || "sha-09deac4";

  return {
    command: "docker",
    args: [
      "run",
      "-i",
      "--rm",
      "-e",
      "GITHUB_PERSONAL_ACCESS_TOKEN",
      `ghcr.io/github/github-mcp-server:${dockerImageVersion}`
    ],
    env: {
      GITHUB_PERSONAL_ACCESS_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
    }
  };
}

/**
 * Generate Playwright MCP server configuration
 * @param {Object} playwrightConfig - Playwright tool configuration
 * @returns {Object} Playwright MCP server configuration
 */
function generatePlaywrightMCPConfig(playwrightConfig) {
  const args = ["@playwright/mcp@latest"];

  // Handle allowed domains
  if (playwrightConfig.allowed_domains) {
    let allowedDomains = [];
    if (Array.isArray(playwrightConfig.allowed_domains)) {
      allowedDomains = playwrightConfig.allowed_domains;
    } else if (typeof playwrightConfig.allowed_domains === "string") {
      allowedDomains = [playwrightConfig.allowed_domains];
    }

    // Ensure localhost domains are always included
    if (!allowedDomains.includes("localhost")) {
      allowedDomains.unshift("localhost");
    }
    if (!allowedDomains.includes("127.0.0.1")) {
      allowedDomains.unshift("127.0.0.1");
    }

    if (allowedDomains.length > 0) {
      args.push("--allowed-origins", allowedDomains.join(","));
    }
  }

  return {
    command: "npx",
    args: args
  };
}

/**
 * Generate cache-memory MCP server configuration
 * @param {Object} cacheConfig - Cache memory tool configuration
 * @returns {Object} Cache memory MCP server configuration
 */
function generateCacheMemoryMCPConfig(cacheConfig) {
  const dockerImage = cacheConfig["docker-image"] || "mcp/memory";

  return {
    command: "docker",
    args: [
      "run",
      "-i",
      "--rm",
      dockerImage
    ]
  };
}

/**
 * Generate custom MCP server configuration
 * @param {string} toolName - The tool name
 * @param {Object} toolConfig - The tool configuration
 * @returns {Object|null} Custom MCP server configuration or null if not applicable
 */
function generateCustomMCPConfig(toolName, toolConfig) {
  // Check for mcp-ref (VSCode import)
  if (toolConfig["mcp-ref"] === "vscode") {
    return loadVSCodeMCPConfig(toolName);
  }

  // Check for direct MCP configuration
  if (toolConfig.mcp) {
    return parseDirectMCPConfig(toolConfig.mcp);
  }

  return null;
}

/**
 * Load MCP configuration from VSCode settings
 * @param {string} serverName - The server name to load from VSCode
 * @returns {Object} MCP server configuration
 */
function loadVSCodeMCPConfig(serverName) {
  const fs = require("fs");
  const path = require("path");

  try {
    // Try to read .vscode/mcp.json from the workspace root
    const vscodeMCPPath = path.join(process.cwd(), ".vscode", "mcp.json");
    
    if (!fs.existsSync(vscodeMCPPath)) {
      throw new Error(`.vscode/mcp.json file not found at ${vscodeMCPPath}`);
    }

    const data = fs.readFileSync(vscodeMCPPath, "utf8");
    const config = JSON.parse(data);

    if (!config.servers || !config.servers[serverName]) {
      const availableServers = config.servers ? Object.keys(config.servers) : [];
      throw new Error(`Server '${serverName}' not found in .vscode/mcp.json. Available servers: [${availableServers.join(", ")}]`);
    }

    const server = config.servers[serverName];
    const result = {
      command: server.command
    };

    if (server.args && server.args.length > 0) {
      result.args = server.args;
    }

    if (server.env && Object.keys(server.env).length > 0) {
      result.env = server.env;
    }

    return result;
  } catch (error) {
    throw new Error(`Failed to load VSCode MCP config for '${serverName}': ${error.message}`);
  }
}

/**
 * Parse direct MCP configuration
 * @param {Object|string} mcpConfig - The MCP configuration (object or JSON string)
 * @returns {Object} Parsed MCP server configuration
 */
function parseDirectMCPConfig(mcpConfig) {
  let config = mcpConfig;
  
  // Handle JSON string format
  if (typeof mcpConfig === "string") {
    try {
      config = JSON.parse(mcpConfig);
    } catch (error) {
      throw new Error(`Invalid JSON in MCP configuration: ${error.message}`);
    }
  }

  if (!config.type) {
    throw new Error("MCP configuration missing required 'type' field");
  }

  const result = {};

  if (config.type === "stdio") {
    // Handle container field (simplified Docker run)
    if (config.container) {
      result.command = "docker";
      result.args = ["run", "--rm", "-i"];
      
      // Add environment variables
      if (config.env) {
        for (const [key, value] of Object.entries(config.env)) {
          result.args.push("-e", key);
        }
        result.env = config.env;
      }
      
      result.args.push(config.container);
    } else if (config.command) {
      result.command = config.command;
      if (config.args) {
        result.args = config.args;
      }
      if (config.env) {
        result.env = config.env;
      }
    } else {
      throw new Error("stdio type requires 'command' or 'container' field");
    }
  } else if (config.type === "http") {
    if (!config.url) {
      throw new Error("http type requires 'url' field");
    }
    // HTTP type MCP servers are not supported in local execution
    throw new Error("HTTP type MCP servers are not supported in local execution");
  } else {
    throw new Error(`Unsupported MCP type: ${config.type}`);
  }

  return result;
}

main();