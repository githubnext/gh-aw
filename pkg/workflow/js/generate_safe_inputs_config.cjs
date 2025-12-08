// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Generates configuration for the Safe Inputs MCP HTTP server
 * @param {object} params - Parameters for config generation
 * @param {typeof import("@actions/core")} params.core - GitHub Actions core library
 * @returns {{port: number}} Generated configuration
 */
function generateSafeInputsConfig({ core }) {
  // Use hardcoded port 52000 (similar to GitHub remote MCP configuration)
  const port = 52000;

  // Set outputs with descriptive names to avoid conflicts
  core.setOutput("safe_inputs_port", port.toString());

  core.info(`Safe Inputs MCP server will run on port ${port}`);

  return { port };
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    generateSafeInputsConfig,
  };
}
