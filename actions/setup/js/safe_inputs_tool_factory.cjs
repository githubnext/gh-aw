// @ts-check

/**
 * Safe Inputs Tool Factory
 *
 * This module provides a factory function for creating tool configuration objects
 * for different handler types (JavaScript, Shell, Python).
 */

/**
 * @typedef {Object} SafeInputsToolConfig
 * @property {string} name - Tool name
 * @property {string} description - Tool description
 * @property {Object} inputSchema - JSON Schema for tool inputs
 * @property {string} handler - Path to handler file (.cjs, .sh, or .py)
 */

/**
 * Create a tool configuration object
 * @param {string} name - Tool name
 * @param {string} description - Tool description
 * @param {Object} inputSchema - JSON Schema for tool inputs
 * @param {string} handlerPath - Path to the handler file (.cjs, .sh, or .py)
 * @returns {SafeInputsToolConfig} Tool configuration object
 */
function createToolConfig(name, description, inputSchema, handlerPath) {
  return {
    name,
    description,
    inputSchema,
    handler: handlerPath,
  };
}

module.exports = {
  createToolConfig,
};
