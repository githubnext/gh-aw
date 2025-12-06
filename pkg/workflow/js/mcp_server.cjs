// @ts-check
/// <reference types="@actions/github-script" />

/**
 * MCP Server Implementation
 *
 * This module provides the MCPServer class for handling MCP (Model Context Protocol)
 * tool registration and JSON-RPC 2.0 protocol handling.
 *
 * Features:
 * - Tool registration with schema validation
 * - JSON-RPC 2.0 protocol handling (initialize, tools/list, tools/call)
 * - Transport-agnostic design (works with HTTP, stdio, etc.)
 *
 * References:
 * - MCP Specification: https://spec.modelcontextprotocol.io
 * - JSON-RPC 2.0: https://www.jsonrpc.org/specification
 */

/**
 * Simple MCP Server implementation that provides tool registration and protocol handling
 */
class MCPServer {
  /**
   * @param {Object} serverInfo - Server metadata
   * @param {string} serverInfo.name - Server name
   * @param {string} serverInfo.version - Server version
   * @param {Object} [options] - Server options
   * @param {Object} [options.capabilities] - Server capabilities
   */
  constructor(serverInfo, options = {}) {
    this.serverInfo = serverInfo;
    this.capabilities = options.capabilities || { tools: {} };
    this.tools = new Map();
    this.transport = null;
    this.initialized = false;
  }

  /**
   * Register a tool with the server
   * @param {string} name - Tool name
   * @param {string} description - Tool description
   * @param {Object} inputSchema - JSON Schema for tool input
   * @param {Function} handler - Async function that handles tool calls
   */
  tool(name, description, inputSchema, handler) {
    this.tools.set(name, {
      name,
      description,
      inputSchema,
      handler,
    });
  }

  /**
   * Connect to a transport
   * @param {any} transport - Transport instance (must have setServer and start methods)
   */
  async connect(transport) {
    this.transport = transport;
    transport.setServer(this);
    await transport.start();
  }

  /**
   * Handle initialize request
   * @param {Object} params - Initialize parameters
   * @returns {Object} Initialize result
   */
  handleInitialize(params) {
    this.initialized = true;
    return {
      protocolVersion: params.protocolVersion || "2024-11-05",
      serverInfo: this.serverInfo,
      capabilities: this.capabilities,
    };
  }

  /**
   * Handle tools/list request
   * @returns {Object} Tools list result
   */
  handleToolsList() {
    const tools = Array.from(this.tools.values()).map(tool => ({
      name: tool.name,
      description: tool.description,
      inputSchema: tool.inputSchema,
    }));
    return { tools };
  }

  /**
   * Handle tools/call request
   * @param {Object} params - Call parameters
   * @param {string} params.name - Tool name
   * @param {Object} params.arguments - Tool arguments
   * @returns {Promise<Object>} Tool call result
   */
  async handleToolsCall(params) {
    const tool = this.tools.get(params.name);
    if (!tool) {
      throw {
        code: -32602,
        message: `Tool '${params.name}' not found`,
      };
    }

    try {
      const result = await tool.handler(params.arguments || {});
      return result;
    } catch (error) {
      throw {
        code: -32603,
        message: error instanceof Error ? error.message : String(error),
      };
    }
  }

  /**
   * Handle ping request
   * @returns {Object} Empty ping result
   */
  handlePing() {
    return {};
  }

  /**
   * Handle an incoming JSON-RPC request
   * @param {Object} request - JSON-RPC request
   * @returns {Promise<Object>} JSON-RPC response
   */
  async handleRequest(request) {
    const { id, method, params } = request;

    try {
      // Handle notifications per JSON-RPC 2.0 spec:
      // 1. Requests without id field (undefined) are notifications (no response)
      // 2. MCP convention: methods starting with "notifications/" are also notifications
      // Note: id can be null for valid requests, so we only check for undefined
      if (!("id" in request) || (method && method.startsWith("notifications/"))) {
        // Notifications don't require a response
        return null;
      }

      let result;

      switch (method) {
        case "initialize":
          result = this.handleInitialize(params || {});
          break;

        case "ping":
          result = this.handlePing();
          break;

        case "tools/list":
          result = this.handleToolsList();
          break;

        case "tools/call":
          result = await this.handleToolsCall(params || {});
          break;

        default:
          throw {
            code: -32601,
            message: `Method '${method}' not found`,
          };
      }

      return {
        jsonrpc: "2.0",
        id,
        result,
      };
    } catch (error) {
      return {
        jsonrpc: "2.0",
        id,
        error: {
          code: error.code || -32603,
          message: error.message || "Internal error",
        },
      };
    }
  }
}

module.exports = {
  MCPServer,
};
