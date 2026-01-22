// @ts-check

const { getErrorMessage } = require("./error_helpers.cjs");

const fs = require("fs");
const path = require("path");

/**
 * Load and process safe outputs configuration
 * @param {Object} server - The MCP server instance for logging
 * @returns {Object} An object containing the processed config and output file path
 */
function loadConfig(server) {
  // Read configuration from file
  const configPath = process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH || "/opt/gh-aw/safeoutputs/config.json";
  let safeOutputsConfigRaw;

  server.debug(`Reading config from file: ${configPath}`);

  try {
    if (fs.existsSync(configPath)) {
      server.debug(`Config file exists at: ${configPath}`);
      const configFileContent = fs.readFileSync(configPath, "utf8");
      server.debug(`Config file content length: ${configFileContent.length} characters`);
      // Don't log raw content to avoid exposing sensitive configuration data
      server.debug(`Config file read successfully, attempting to parse JSON`);
      safeOutputsConfigRaw = JSON.parse(configFileContent);
      server.debug(`Successfully parsed config from file with ${Object.keys(safeOutputsConfigRaw).length} configuration keys`);
    } else {
      server.debug(`Config file does not exist at: ${configPath}`);
      server.debug(`Using minimal default configuration`);
      safeOutputsConfigRaw = {};
    }
  } catch (error) {
    server.debug(`Error reading config file: ${getErrorMessage(error)}`);
    server.debug(`Falling back to empty configuration`);
    safeOutputsConfigRaw = {};
  }

  const safeOutputsConfig = Object.fromEntries(Object.entries(safeOutputsConfigRaw).map(([k, v]) => [k.replace(/-/g, "_"), v]));
  server.debug(`Final processed config: ${JSON.stringify(safeOutputsConfig)}`);

  // Handle GH_AW_SAFE_OUTPUTS with default fallback
  // Default is /opt (read-only mount for agent container)
  const outputFile = process.env.GH_AW_SAFE_OUTPUTS || "/opt/gh-aw/safeoutputs/outputs.jsonl";
  if (!process.env.GH_AW_SAFE_OUTPUTS) {
    server.debug(`GH_AW_SAFE_OUTPUTS not set, using default: ${outputFile}`);
  }
  // Always ensure the directory exists, regardless of whether env var is set
  const outputDir = path.dirname(outputFile);
  if (!fs.existsSync(outputDir)) {
    server.debug(`Creating output directory: ${outputDir}`);
    fs.mkdirSync(outputDir, { recursive: true });
  }

  return {
    config: safeOutputsConfig,
    outputFile: outputFile,
  };
}

module.exports = { loadConfig };
