// @ts-check
/// <reference types="@actions/github-script" />

// generate_codex_config.cjs
// Generates Codex MCP configuration in TOML format from JSON input
// This script runs in GitHub Actions and receives MCP config via environment variable

const fs = require("fs");
const path = require("path");

/**
 * Escapes a string for use in TOML basic strings (values)
 * @param {string} str - The string to escape
 * @returns {string} - The escaped string
 */
function escapeTOMLString(str) {
  return str.replace(/\\/g, "\\\\").replace(/"/g, '\\"').replace(/\n/g, "\\n").replace(/\r/g, "\\r").replace(/\t/g, "\\t");
}

/**
 * Escapes a key or section name for use in TOML
 * Keys with special characters must be quoted
 * @param {string} key - The key to escape
 * @returns {string} - The escaped key (quoted if necessary)
 */
function escapeTOMLKey(key) {
  // Check if key needs quoting (contains special chars or starts with number)
  const needsQuoting = /[^a-zA-Z0-9_-]/.test(key) || /^[0-9]/.test(key);
  
  if (needsQuoting) {
    // Escape the key and wrap in quotes
    return `"${escapeTOMLString(key)}"`;
  }
  
  // Simple keys don't need quotes
  return key;
}

/**
 * Converts a value to TOML format
 * @param {any} value - The value to convert
 * @param {number} indent - The indentation level
 * @returns {string} - The TOML representation
 */
function toTOMLValue(value, indent = 0) {
  const indentStr = "  ".repeat(indent);

  if (typeof value === "string") {
    return `"${escapeTOMLString(value)}"`;
  } else if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  } else if (Array.isArray(value)) {
    if (value.length === 0) {
      return "[]";
    }
    // Arrays in TOML are formatted on multiple lines for readability
    const items = value.map((item) => `${indentStr}  ${toTOMLValue(item, indent + 1)}`).join(",\n");
    return `[\n${items}\n${indentStr}]`;
  } else if (typeof value === "object" && value !== null) {
    // Objects are rendered as inline tables in TOML
    const pairs = Object.entries(value)
      .map(([k, v]) => `${escapeTOMLKey(k)} = ${toTOMLValue(v, indent)}`)
      .join(", ");
    return `{ ${pairs} }`;
  }
  return "null";
}

/**
 * Renders an MCP server configuration block in TOML format
 * @param {string} serverName - The name of the MCP server
 * @param {any} serverConfig - The server configuration object
 * @returns {string} - The TOML configuration block
 */
function renderMCPServer(serverName, serverConfig) {
  // Escape the server name for use in section headers
  const escapedServerName = escapeTOMLKey(serverName);
  let toml = `\n[mcp_servers.${escapedServerName}]\n`;

  // Handle different server types
  if (serverConfig.type === "http") {
    // HTTP server configuration
    if (serverConfig.url) {
      toml += `url = ${toTOMLValue(serverConfig.url)}\n`;
    }
    if (serverConfig.bearer_token_env_var) {
      toml += `bearer_token_env_var = ${toTOMLValue(serverConfig.bearer_token_env_var)}\n`;
    }
  } else {
    // stdio/local server configuration
    if (serverConfig.command) {
      toml += `command = ${toTOMLValue(serverConfig.command)}\n`;
    }
    if (serverConfig.args && Array.isArray(serverConfig.args)) {
      toml += `args = ${toTOMLValue(serverConfig.args)}\n`;
    }
  }

  // Add optional fields
  if (serverConfig.user_agent) {
    toml += `user_agent = ${toTOMLValue(serverConfig.user_agent)}\n`;
  }
  if (serverConfig.startup_timeout_sec !== undefined) {
    toml += `startup_timeout_sec = ${serverConfig.startup_timeout_sec}\n`;
  }
  if (serverConfig.tool_timeout_sec !== undefined) {
    toml += `tool_timeout_sec = ${serverConfig.tool_timeout_sec}\n`;
  }

  // Add environment variables as a TOML section if present
  if (serverConfig.env && typeof serverConfig.env === "object" && Object.keys(serverConfig.env).length > 0) {
    toml += `\n[mcp_servers.${escapedServerName}.env]\n`;
    for (const [key, value] of Object.entries(serverConfig.env)) {
      // Escape environment variable keys
      const escapedKey = escapeTOMLKey(key);
      toml += `${escapedKey} = ${toTOMLValue(value)}\n`;
    }
  }

  return toml;
}

/**
 * Generates the complete Codex MCP configuration in TOML format
 * @param {any} mcpConfig - The MCP configuration object
 * @returns {string} - The complete TOML configuration
 */
function generateCodexConfig(mcpConfig) {
  let toml = "";

  // Add history configuration if present
  if (mcpConfig.history) {
    toml += "[history]\n";
    for (const [key, value] of Object.entries(mcpConfig.history)) {
      // Escape history configuration keys
      const escapedKey = escapeTOMLKey(key);
      toml += `${escapedKey} = ${toTOMLValue(value)}\n`;
    }
  }

  // Add MCP servers
  if (mcpConfig.mcp_servers && typeof mcpConfig.mcp_servers === "object") {
    for (const [serverName, serverConfig] of Object.entries(mcpConfig.mcp_servers)) {
      toml += renderMCPServer(serverName, serverConfig);
    }
  }

  // Add custom configuration if present
  if (mcpConfig.custom_config) {
    toml += "\n# Custom configuration\n";
    toml += mcpConfig.custom_config;
    if (!mcpConfig.custom_config.endsWith("\n")) {
      toml += "\n";
    }
  }

  return toml;
}

/**
 * Main function for Codex config generation in GitHub Actions
 */
function main() {
  try {
    // Read MCP configuration from environment variable
    const mcpConfigJSON = process.env.GH_AW_MCP_CONFIG_JSON;
    if (!mcpConfigJSON) {
      core.setFailed("GH_AW_MCP_CONFIG_JSON environment variable is not set");
      process.exit(1);
    }

    // Parse the JSON configuration
    let mcpConfig;
    try {
      // Trim whitespace that might come from YAML block scalar
      mcpConfig = JSON.parse(mcpConfigJSON.trim());
    } catch (error) {
      core.setFailed(`Failed to parse MCP configuration JSON: ${error instanceof Error ? error.message : String(error)}`);
      process.exit(1);
    }

    // Generate TOML configuration
    const tomlConfig = generateCodexConfig(mcpConfig);

    // Get output path from environment or use default
    const outputPath = process.env.GH_AW_MCP_CONFIG || "/tmp/gh-aw/mcp-config/config.toml";

    // Ensure directory exists
    const outputDir = path.dirname(outputPath);
    if (!fs.existsSync(outputDir)) {
      fs.mkdirSync(outputDir, { recursive: true });
    }

    // Write TOML configuration to file
    fs.writeFileSync(outputPath, tomlConfig, "utf8");

    core.info(`Codex MCP configuration written to ${outputPath}`);
    core.info(`Configuration size: ${tomlConfig.length} bytes`);

    // Add summary
    core.summary
      .addHeading("Codex MCP Configuration Generated", 3)
      .addRaw("\n")
      .addRaw(`Configuration file: \`${outputPath}\`\n`)
      .addRaw(`Total servers configured: ${mcpConfig.mcp_servers ? Object.keys(mcpConfig.mcp_servers).length : 0}\n`)
      .write();
  } catch (error) {
    core.setFailed(error instanceof Error ? error.message : String(error));
  }
}

// Export functions for testing
if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    escapeTOMLString,
    escapeTOMLKey,
    toTOMLValue,
    renderMCPServer,
    generateCodexConfig,
    main,
  };
}

// Execute main function only when run directly (not when required)
if (require.main === module) {
  main();
}
