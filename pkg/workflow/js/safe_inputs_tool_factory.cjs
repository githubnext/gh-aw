// @ts-check

/**
 * Safe Inputs Tool Factory
 *
 * This module provides factory functions for creating tool configuration objects
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
 * Create tool configuration for a JavaScript handler
 * @param {string} name - Tool name
 * @param {string} description - Tool description
 * @param {Object} inputSchema - JSON Schema for tool inputs
 * @param {string} handlerPath - Relative path to the .cjs handler file
 * @returns {SafeInputsToolConfig} Tool configuration object
 */
function createJsToolConfig(name, description, inputSchema, handlerPath) {
  return {
    name,
    description,
    inputSchema,
    handler: handlerPath,
  };
}

/**
 * Create tool configuration for a shell script handler
 * @param {string} name - Tool name
 * @param {string} description - Tool description
 * @param {Object} inputSchema - JSON Schema for tool inputs
 * @param {string} handlerPath - Relative path to the .sh handler file
 * @returns {SafeInputsToolConfig} Tool configuration object
 */
function createShellToolConfig(name, description, inputSchema, handlerPath) {
  return {
    name,
    description,
    inputSchema,
    handler: handlerPath,
  };
}

/**
 * Create tool configuration for a Python script handler
 * @param {string} name - Tool name
 * @param {string} description - Tool description
 * @param {Object} inputSchema - JSON Schema for tool inputs
 * @param {string} handlerPath - Relative path to the .py handler file
 * @returns {SafeInputsToolConfig} Tool configuration object
 */
function createPythonToolConfig(name, description, inputSchema, handlerPath) {
  return {
    name,
    description,
    inputSchema,
    handler: handlerPath,
  };
}

module.exports = {
  createJsToolConfig,
  createShellToolConfig,
  createPythonToolConfig,
};
