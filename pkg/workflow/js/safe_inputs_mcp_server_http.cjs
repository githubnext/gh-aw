// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Inputs MCP Server with HTTP Transport
 *
 * This module extends the safe-inputs MCP server to support HTTP transport
 * using the StreamableHTTPServerTransport from the MCP SDK.
 *
 * It provides both stateful and stateless HTTP modes, as well as SSE streaming.
 *
 * Usage:
 *   node safe_inputs_mcp_server_http.cjs /path/to/tools.json [--port 3000] [--stateless]
 *
 * Options:
 *   --port <number>    Port to listen on (default: 3000)
 *   --stateless        Run in stateless mode (no session management)
 *   --log-dir <path>   Directory for log files
 */

const path = require("path");
const http = require("http");
const { randomUUID } = require("crypto");
const { McpServer } = require("@modelcontextprotocol/sdk/server/mcp.js");
const { StreamableHTTPServerTransport } = require("@modelcontextprotocol/sdk/server/streamableHttp.js");
const { loadConfig } = require("./safe_inputs_config_loader.cjs");
const { loadToolHandlers } = require("./mcp_server_core.cjs");
const { validateRequiredFields } = require("./safe_inputs_validation.cjs");

/**
 * Create and configure the MCP server with tools
 * @param {string} configPath - Path to the configuration JSON file
 * @param {Object} [options] - Additional options
 * @param {string} [options.logDir] - Override log directory from config
 * @returns {Object} Server instance and configuration
 */
function createMCPServer(configPath, options = {}) {
  // Load configuration
  const config = loadConfig(configPath);

  // Determine base path for resolving relative handler paths
  const basePath = path.dirname(configPath);

  // Create server with configuration
  const serverName = config.serverName || "safeinputs";
  const version = config.version || "1.0.0";

  // Create MCP SDK Server instance using McpServer
  const server = new McpServer(
    {
      name: serverName,
      version: version,
    },
    {
      capabilities: {
        tools: {},
      },
    }
  );

  // Create a simple logger that mimics mcp_server_core's debug function
  const logger = {
    debug: msg => {
      const timestamp = new Date().toISOString();
      process.stderr.write(`[${timestamp}] [${serverName}] ${msg}\n`);
    },
    debugError: (prefix, error) => {
      const errorMessage = error instanceof Error ? error.message : String(error);
      logger.debug(`${prefix}${errorMessage}`);
      if (error instanceof Error && error.stack) {
        logger.debug(`${prefix}Stack trace: ${error.stack}`);
      }
    },
  };

  logger.debug(`Loading safe-inputs configuration from: ${configPath}`);
  logger.debug(`Base path for handlers: ${basePath}`);
  logger.debug(`Tools to load: ${config.tools.length}`);

  // Load tool handlers from file paths
  // We'll use a temporary server object for loadToolHandlers compatibility
  const tempServer = { debug: logger.debug, debugError: logger.debugError };
  const tools = loadToolHandlers(tempServer, config.tools, basePath);

  // Register all tools with the MCP SDK server using the tool() method
  for (const tool of tools) {
    if (!tool.handler) {
      logger.debug(`Skipping tool ${tool.name} - no handler loaded`);
      continue;
    }

    logger.debug(`Registering tool: ${tool.name}`);

    // Register the tool with the MCP SDK using the high-level API
    // The callback receives the arguments directly as the first parameter
    server.tool(tool.name, tool.description || "", tool.inputSchema || { type: "object", properties: {} }, async args => {
      logger.debug(`Calling handler for tool: ${tool.name}`);

      // Validate required fields using helper
      const missing = validateRequiredFields(args, tool.inputSchema);
      if (missing.length) {
        throw new Error(`Invalid arguments: missing or empty ${missing.map(m => `'${m}'`).join(", ")}`);
      }

      // Call the handler
      const result = await Promise.resolve(tool.handler(args));
      logger.debug(`Handler returned for tool: ${tool.name}`);

      // Normalize result to MCP format
      const content = result && result.content ? result.content : [];
      return { content, isError: false };
    });
  }

  return { server, config, logger };
}

/**
 * Start the HTTP server with MCP protocol support
 * @param {string} configPath - Path to the configuration JSON file
 * @param {Object} options - Server options
 * @param {number} [options.port] - Port to listen on (default: 3000)
 * @param {boolean} [options.stateless] - Run in stateless mode (default: false)
 * @param {string} [options.logDir] - Override log directory from config
 */
async function startHttpServer(configPath, options = {}) {
  const port = options.port || 3000;
  const stateless = options.stateless || false;

  // Create the MCP server
  const { server, config, logger } = createMCPServer(configPath, { logDir: options.logDir });

  logger.debug(`Starting HTTP server on port ${port}`);
  logger.debug(`Mode: ${stateless ? "stateless" : "stateful"}`);

  // Create the HTTP transport
  const transport = new StreamableHTTPServerTransport({
    sessionIdGenerator: stateless ? undefined : () => randomUUID(),
    enableJsonResponse: true,
    enableDnsRebindingProtection: false, // Disable for local development
  });

  // Connect server to transport
  await server.connect(transport);

  // Create HTTP server
  const httpServer = http.createServer(async (req, res) => {
    // Set CORS headers for development
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
    res.setHeader("Access-Control-Allow-Headers", "Content-Type, Accept");

    // Handle OPTIONS preflight
    if (req.method === "OPTIONS") {
      res.writeHead(200);
      res.end();
      return;
    }

    // Only handle POST requests for MCP protocol
    if (req.method !== "POST") {
      res.writeHead(405, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "Method not allowed" }));
      return;
    }

    try {
      // Parse request body for POST requests
      let body = null;
      if (req.method === "POST") {
        const chunks = [];
        for await (const chunk of req) {
          chunks.push(chunk);
        }
        const bodyStr = Buffer.concat(chunks).toString();
        try {
          body = bodyStr ? JSON.parse(bodyStr) : null;
        } catch (parseError) {
          res.writeHead(400, { "Content-Type": "application/json" });
          res.end(
            JSON.stringify({
              jsonrpc: "2.0",
              error: {
                code: -32700,
                message: "Parse error: Invalid JSON in request body",
              },
              id: null,
            })
          );
          return;
        }
      }

      // Let the transport handle the request
      await transport.handleRequest(req, res, body);
    } catch (error) {
      logger.debugError("Error handling request: ", error);
      if (!res.headersSent) {
        res.writeHead(500, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            jsonrpc: "2.0",
            error: {
              code: -32603,
              message: error instanceof Error ? error.message : String(error),
            },
            id: null,
          })
        );
      }
    }
  });

  // Start listening
  httpServer.listen(port, () => {
    logger.debug(`HTTP server listening on http://localhost:${port}`);
    logger.debug(`MCP endpoint: POST http://localhost:${port}/`);
    logger.debug(`Server name: ${config.serverName || "safeinputs"}`);
    logger.debug(`Server version: ${config.version || "1.0.0"}`);
    logger.debug(`Tools available: ${config.tools.length}`);
  });

  // Handle shutdown gracefully
  process.on("SIGINT", () => {
    logger.debug("Received SIGINT, shutting down...");
    httpServer.close(() => {
      logger.debug("HTTP server closed");
      process.exit(0);
    });
  });

  process.on("SIGTERM", () => {
    logger.debug("Received SIGTERM, shutting down...");
    httpServer.close(() => {
      logger.debug("HTTP server closed");
      process.exit(0);
    });
  });

  return httpServer;
}

// If run directly, start the HTTP server with command-line arguments
if (require.main === module) {
  const args = process.argv.slice(2);

  if (args.length < 1) {
    console.error("Usage: node safe_inputs_mcp_server_http.cjs <config.json> [--port <number>] [--stateless] [--log-dir <path>]");
    process.exit(1);
  }

  const configPath = args[0];
  const options = {
    port: 3000,
    stateless: false,
    logDir: undefined,
  };

  // Parse optional arguments
  for (let i = 1; i < args.length; i++) {
    if (args[i] === "--port" && args[i + 1]) {
      options.port = parseInt(args[i + 1], 10);
      i++;
    } else if (args[i] === "--stateless") {
      options.stateless = true;
    } else if (args[i] === "--log-dir" && args[i + 1]) {
      options.logDir = args[i + 1];
      i++;
    }
  }

  startHttpServer(configPath, options).catch(error => {
    console.error(`Error starting HTTP server: ${error instanceof Error ? error.message : String(error)}`);
    process.exit(1);
  });
}

module.exports = {
  startHttpServer,
  createMCPServer,
};
