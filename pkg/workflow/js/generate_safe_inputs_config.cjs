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
  // Using 45 bytes gives us 360 bits of entropy and ensures at least 40 characters
  // after base64 encoding and removing special characters (base64 of 45 bytes = 60 chars)
  const apiKeyBuffer = crypto.randomBytes(45);
  const apiKey = apiKeyBuffer.toString("base64").replace(/[/+=]/g, "");

  // Choose a port for the HTTP server (default 3000)
  const port = 3000;

  // Set outputs with descriptive names to avoid conflicts
  core.setOutput("safe_inputs_api_key", apiKey);
  core.setOutput("safe_inputs_port", port.toString());

  core.info(`Safe Inputs MCP server will run on port ${port}`);

  return { apiKey, port };
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    generateSafeInputsConfig,
  };
}
