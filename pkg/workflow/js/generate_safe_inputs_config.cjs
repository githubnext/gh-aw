// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generates configuration for the Safe Inputs MCP HTTP server
 * @param {object} params - Parameters for config generation
 * @param {typeof import("@actions/core")} params.core - GitHub Actions core library
 * @param {typeof import("crypto")} params.crypto - Node.js crypto library
 * @returns {{apiKey: string, port: number}} Generated configuration
 */
function generateSafeInputsConfig({ core, crypto }) {
  // Generate a secure random API key for the MCP server
  // Using 32 bytes gives us 256 bits of entropy
  const apiKeyBuffer = crypto.randomBytes(32);
  const apiKey = apiKeyBuffer.toString("base64").replace(/[/+=]/g, "");

  // Choose a port for the HTTP server (default 3000)
  const port = 3000;

  // Set outputs
  core.setOutput("api_key", apiKey);
  core.setOutput("port", port.toString());

  core.info(`Safe Inputs MCP server will run on port ${port}`);

  return { apiKey, port };
}

module.exports = {
  generateSafeInputsConfig,
};
