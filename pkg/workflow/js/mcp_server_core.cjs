// @ts-check
/// <reference types="@actions/github-script" />

/**
 * MCP Server Core Module
 *
 * This module provides a reusable API for creating MCP (Model Context Protocol) servers.
 * It handles JSON-RPC 2.0 message parsing, tool registration, and server lifecycle.
 *
 * The module supports different transport mechanisms:
 * - stdio: Uses stdin/stdout for communication (default via start())
 * - http: HTTP-based transport (to be implemented separately)
 *
 * Usage:
 *   const { createServer, registerTool, start } = require("./mcp_server_core.cjs");
 *
 *   const server = createServer({ name: "my-server", version: "1.0.0" });
 *   registerTool(server, {
 *     name: "my_tool",
 *     description: "A tool",
 *     inputSchema: { type: "object", properties: {} },
 *     handler: (args) => ({ content: [{ type: "text", text: "result" }] })
 *   });
 *   start(server);
 *
 * For direct transport access:
 *   const { createStdioTransport } = require("./mcp_stdio_transport.cjs");
 *   const transport = createStdioTransport({ onDebug: server.debug });
 *   transport.onMessage(msg => handleMessage(server, msg));
 *   transport.start();
 */

const fs = require("fs");
const path = require("path");

const { ReadBuffer } = require("./read_buffer.cjs");
const { createStdioTransport } = require("./mcp_stdio_transport.cjs");

const encoder = new TextEncoder();

/**
 * @typedef {Object} ServerInfo
 * @property {string} name - Server name
 * @property {string} version - Server version
 */

/**
 * @typedef {Object} Tool
 * @property {string} name - Tool name
 * @property {string} description - Tool description
 * @property {Object} inputSchema - JSON Schema for tool inputs
 * @property {Function} [handler] - Tool handler function
 */

/**
 * @typedef {Object} MCPServer
 * @property {ServerInfo} serverInfo - Server information
 * @property {Object<string, Tool>} tools - Registered tools
 * @property {Function} debug - Debug logging function
 * @property {Function} writeMessage - Write message to transport
 * @property {Function} replyResult - Send a result response
 * @property {Function} replyError - Send an error response
 * @property {ReadBuffer} readBuffer - Message buffer (legacy, kept for compatibility)
 * @property {string} [logDir] - Optional log directory
 * @property {string} [logFilePath] - Optional log file path
 * @property {boolean} logFileInitialized - Whether log file has been initialized
 * @property {Object} [transport] - Transport instance (stdio, http, etc.)
 */

/**
 * Initialize log file for the server
 * @param {MCPServer} server - The MCP server instance
 */
function initLogFile(server) {
  if (server.logFileInitialized || !server.logDir || !server.logFilePath) return;
  try {
    if (!fs.existsSync(server.logDir)) {
      fs.mkdirSync(server.logDir, { recursive: true });
    }
    // Initialize/truncate log file with header
    const timestamp = new Date().toISOString();
    fs.writeFileSync(
      server.logFilePath,
      `# ${server.serverInfo.name} MCP Server Log\n# Started: ${timestamp}\n# Version: ${server.serverInfo.version}\n\n`
    );
    server.logFileInitialized = true;
  } catch {
    // Silently ignore errors - logging to stderr will still work
  }
}

/**
 * Create a debug function for the server
 * @param {MCPServer} server - The MCP server instance
 * @returns {Function} Debug function
 */
function createDebugFunction(server) {
  return msg => {
    const timestamp = new Date().toISOString();
    const formattedMsg = `[${timestamp}] [${server.serverInfo.name}] ${msg}\n`;

    // Always write to stderr
    process.stderr.write(formattedMsg);

    // Also write to log file if log directory is set (initialize on first use)
    if (server.logDir && server.logFilePath) {
      if (!server.logFileInitialized) {
        initLogFile(server);
      }
      if (server.logFileInitialized) {
        try {
          fs.appendFileSync(server.logFilePath, formattedMsg);
        } catch {
          // Silently ignore file write errors - stderr logging still works
        }
      }
    }
  };
}

/**
 * Create a writeMessage function for the server
 * @param {MCPServer} server - The MCP server instance
 * @returns {Function} Write message function
 */
function createWriteMessageFunction(server) {
  return obj => {
    const json = JSON.stringify(obj);
    server.debug(`send: ${json}`);
    const message = json + "\n";
    const bytes = encoder.encode(message);
    fs.writeSync(1, bytes);
  };
}

/**
 * Create a replyResult function for the server
 * @param {MCPServer} server - The MCP server instance
 * @returns {Function} Reply result function
 */
function createReplyResultFunction(server) {
  return (id, result) => {
    if (id === undefined || id === null) return; // notification
    const res = { jsonrpc: "2.0", id, result };
    server.writeMessage(res);
  };
}

/**
 * Create a replyError function for the server
 * @param {MCPServer} server - The MCP server instance
 * @returns {Function} Reply error function
 */
function createReplyErrorFunction(server) {
  return (id, code, message) => {
    // Don't send error responses for notifications (id is null/undefined)
    if (id === undefined || id === null) {
      server.debug(`Error for notification: ${message}`);
      return;
    }

    const error = { code, message };
    const res = {
      jsonrpc: "2.0",
      id,
      error,
    };
    server.writeMessage(res);
  };
}

/**
 * Create a new MCP server instance
 * @param {ServerInfo} serverInfo - Server information (name and version)
 * @param {Object} [options] - Optional server configuration
 * @param {string} [options.logDir] - Directory for log file (optional)
 * @returns {MCPServer} The MCP server instance
 */
function createServer(serverInfo, options = {}) {
  const logDir = options.logDir || undefined;
  const logFilePath = logDir ? path.join(logDir, "server.log") : undefined;

  /** @type {MCPServer} */
  const server = {
    serverInfo,
    tools: {},
    debug: () => {}, // placeholder
    writeMessage: () => {}, // placeholder
    replyResult: () => {}, // placeholder
    replyError: () => {}, // placeholder
    readBuffer: new ReadBuffer(), // kept for backward compatibility
    logDir,
    logFilePath,
    logFileInitialized: false,
    transport: null, // will be set when start() is called
  };

  // Initialize functions with references to server
  server.debug = createDebugFunction(server);
  server.writeMessage = createWriteMessageFunction(server);
  server.replyResult = createReplyResultFunction(server);
  server.replyError = createReplyErrorFunction(server);

  return server;
}

/**
 * Register a tool with the server
 * @param {MCPServer} server - The MCP server instance
 * @param {Tool} tool - The tool to register
 */
function registerTool(server, tool) {
  const normalizedName = normalizeTool(tool.name);
  server.tools[normalizedName] = {
    ...tool,
    name: normalizedName,
  };
  server.debug(`Registered tool: ${normalizedName}`);
}

/**
 * Normalize a tool name (convert dashes to underscores, lowercase)
 * @param {string} name - The tool name to normalize
 * @returns {string} Normalized tool name
 */
function normalizeTool(name) {
  return name.replace(/-/g, "_").toLowerCase();
}

/**
 * Handle an incoming JSON-RPC message
 * @param {MCPServer} server - The MCP server instance
 * @param {Object} req - The incoming request
 * @param {Function} [defaultHandler] - Default handler for tools without a handler
 */
function handleMessage(server, req, defaultHandler) {
  // Validate basic JSON-RPC structure
  if (!req || typeof req !== "object") {
    server.debug(`Invalid message: not an object`);
    return;
  }

  if (req.jsonrpc !== "2.0") {
    server.debug(`Invalid message: missing or invalid jsonrpc field`);
    return;
  }

  const { id, method, params } = req;

  // Validate method field
  if (!method || typeof method !== "string") {
    server.replyError(id, -32600, "Invalid Request: method must be a string");
    return;
  }

  try {
    if (method === "initialize") {
      const clientInfo = params?.clientInfo ?? {};
      server.debug(`client info: ${JSON.stringify(clientInfo)}`);
      const protocolVersion = params?.protocolVersion ?? undefined;
      const result = {
        serverInfo: server.serverInfo,
        ...(protocolVersion ? { protocolVersion } : {}),
        capabilities: {
          tools: {},
        },
      };
      server.replyResult(id, result);
    } else if (method === "tools/list") {
      const list = [];
      Object.values(server.tools).forEach(tool => {
        const toolDef = {
          name: tool.name,
          description: tool.description,
          inputSchema: tool.inputSchema,
        };
        list.push(toolDef);
      });
      server.replyResult(id, { tools: list });
    } else if (method === "tools/call") {
      const name = params?.name;
      const args = params?.arguments ?? {};
      if (!name || typeof name !== "string") {
        server.replyError(id, -32602, "Invalid params: 'name' must be a string");
        return;
      }
      const tool = server.tools[normalizeTool(name)];
      if (!tool) {
        server.replyError(id, -32601, `Tool not found: ${name} (${normalizeTool(name)})`);
        return;
      }

      // Use tool handler, or default handler, or error
      let handler = tool.handler;
      if (!handler && defaultHandler) {
        handler = defaultHandler(tool.name);
      }
      if (!handler) {
        server.replyError(id, -32603, `No handler for tool: ${name}`);
        return;
      }

      const requiredFields = tool.inputSchema && Array.isArray(tool.inputSchema.required) ? tool.inputSchema.required : [];
      if (requiredFields.length) {
        const missing = requiredFields.filter(f => {
          const value = args[f];
          return value === undefined || value === null || (typeof value === "string" && value.trim() === "");
        });
        if (missing.length) {
          server.replyError(id, -32602, `Invalid arguments: missing or empty ${missing.map(m => `'${m}'`).join(", ")}`);
          return;
        }
      }
      const result = handler(args);
      const content = result && result.content ? result.content : [];
      server.replyResult(id, { content, isError: false });
    } else if (/^notifications\//.test(method)) {
      server.debug(`ignore ${method}`);
    } else {
      server.replyError(id, -32601, `Method not found: ${method}`);
    }
  } catch (e) {
    server.replyError(id, -32603, e instanceof Error ? e.message : String(e));
  }
}

/**
 * Process the read buffer and handle messages
 * @param {MCPServer} server - The MCP server instance
 * @param {Function} [defaultHandler] - Default handler for tools without a handler
 * @deprecated Use transport-based message handling instead
 */
function processReadBuffer(server, defaultHandler) {
  while (true) {
    try {
      const message = server.readBuffer.readMessage();
      if (!message) {
        break;
      }
      server.debug(`recv: ${JSON.stringify(message)}`);
      handleMessage(server, message, defaultHandler);
    } catch (error) {
      // For parse errors, we can't know the request id, so we shouldn't send a response
      // according to JSON-RPC spec. Just log the error.
      server.debug(`Parse error: ${error instanceof Error ? error.message : String(error)}`);
    }
  }
}

/**
 * Start the MCP server on stdio transport
 * @param {MCPServer} server - The MCP server instance
 * @param {Object} [options] - Start options
 * @param {Function} [options.defaultHandler] - Default handler for tools without a handler
 */
function start(server, options = {}) {
  const { defaultHandler } = options;

  server.debug(`v${server.serverInfo.version} ready on stdio`);
  server.debug(`  tools: ${Object.keys(server.tools).join(", ")}`);

  if (!Object.keys(server.tools).length) {
    throw new Error("No tools registered");
  }

  // Create and configure the stdio transport
  const transport = createStdioTransport({
    onDebug: server.debug,
  });

  // Update server to use transport for sending messages
  server.transport = transport;
  const originalWriteMessage = server.writeMessage;
  server.writeMessage = msg => {
    if (transport.isRunning()) {
      transport.send(msg);
    } else {
      // Fallback to original implementation if transport not running
      originalWriteMessage(msg);
    }
  };

  // Set up message handler
  transport.onMessage(message => {
    handleMessage(server, message, defaultHandler);
  });

  // Start the transport
  transport.start();
}

/**
 * Start the MCP server with a custom transport
 * This allows using different transports (stdio, http, etc.)
 * @param {MCPServer} server - The MCP server instance
 * @param {Object} transport - Transport instance with start(), send(), and onMessage() methods
 * @param {Object} [options] - Start options
 * @param {Function} [options.defaultHandler] - Default handler for tools without a handler
 */
function startWithTransport(server, transport, options = {}) {
  const { defaultHandler } = options;

  server.debug(`v${server.serverInfo.version} ready`);
  server.debug(`  tools: ${Object.keys(server.tools).join(", ")}`);

  if (!Object.keys(server.tools).length) {
    throw new Error("No tools registered");
  }

  // Store transport reference
  server.transport = transport;

  // Update server to use transport for sending messages
  server.writeMessage = msg => {
    transport.send(msg);
  };

  // Set up message handler
  transport.onMessage(message => {
    handleMessage(server, message, defaultHandler);
  });

  // Start the transport
  transport.start();
}

module.exports = {
  createServer,
  registerTool,
  normalizeTool,
  handleMessage,
  processReadBuffer,
  start,
  startWithTransport,
};
