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

const http = require("http");
const { randomUUID } = require("crypto");
const { MCPServer, MCPHTTPTransport } = require("./mcp_http_transport.cjs");
const { validateRequiredFields } = require("./safe_inputs_validation.cjs");
const { generateEnhancedErrorMessage } = require("./mcp_enhanced_errors.cjs");
const { createLogger } = require("./mcp_logger.cjs");
const { bootstrapSafeInputsServer, cleanupConfigFile } = require("./safe_inputs_bootstrap.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Create and configure the MCP server with tools
 * @param {string} configPath - Path to the configuration JSON file
 * @param {Object} [options] - Additional options
 * @param {string} [options.logDir] - Override log directory from config
 * @returns {Object} Server instance and configuration
 */
function createMCPServer(configPath, options = {}) {
  // Create logger early
  const logger = createLogger("safeinputs");

  logger.debug(`=== Creating MCP Server ===`);
  logger.debug(`Configuration file: ${configPath}`);

  // Bootstrap: load configuration and tools using shared logic
  const { config, tools } = bootstrapSafeInputsServer(configPath, logger);

  // Create server with configuration
  const serverName = config.serverName || "safeinputs";
  const version = config.version || "1.0.0";

  logger.debug(`Server name: ${serverName}`);
  logger.debug(`Server version: ${version}`);

  // Create MCP Server instance
  const server = new MCPServer(
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

  // Register all tools with the MCP SDK server using the tool() method
  logger.debug(`Registering tools with MCP server...`);
  let registeredCount = 0;
  let skippedCount = 0;

  for (const tool of tools) {
    if (!tool.handler) {
      logger.debug(`Skipping tool ${tool.name} - no handler loaded`);
      skippedCount++;
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
        throw new Error(generateEnhancedErrorMessage(missing, tool.name, tool.inputSchema));
      }

      // Call the handler
      const result = await Promise.resolve(tool.handler(args));
      logger.debug(`Handler returned for tool: ${tool.name}`);

      // Normalize result to MCP format
      const content = result && result.content ? result.content : [];
      return { content, isError: false };
    });

    registeredCount++;
  }

  logger.debug(`Tool registration complete: ${registeredCount} registered, ${skippedCount} skipped`);
  logger.debug(`=== MCP Server Creation Complete ===`);

  // Cleanup: delete the configuration file after loading
  cleanupConfigFile(configPath, logger);

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

  const logger = createLogger("safe-inputs-startup");

  logger.debug(`=== Starting Safe Inputs MCP HTTP Server ===`);
  logger.debug(`Configuration file: ${configPath}`);
  logger.debug(`Port: ${port}`);
  logger.debug(`Mode: ${stateless ? "stateless" : "stateful"}`);
  logger.debug(`Environment: NODE_VERSION=${process.version}, PLATFORM=${process.platform}`);

  // Create the MCP server
  try {
    const { server, config, logger: mcpLogger } = createMCPServer(configPath, { logDir: options.logDir });

    // Use the MCP logger for subsequent messages
    Object.assign(logger, mcpLogger);

    logger.debug(`MCP server created successfully`);
    logger.debug(`Server name: ${config.serverName || "safeinputs"}`);
    logger.debug(`Server version: ${config.version || "1.0.0"}`);
    logger.debug(`Tools configured: ${config.tools.length}`);

    logger.debug(`Creating HTTP transport...`);
    // Create the HTTP transport
    const transport = new MCPHTTPTransport({
      sessionIdGenerator: stateless ? undefined : () => randomUUID(),
      enableJsonResponse: true,
      enableDnsRebindingProtection: false, // Disable for local development
    });
    logger.debug(`HTTP transport created`);

    // Connect server to transport
    logger.debug(`Connecting server to transport...`);
    await server.connect(transport);
    logger.debug(`Server connected to transport successfully`);

    // Create HTTP server
    logger.debug(`Creating HTTP server...`);
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

      // Handle GET /health endpoint for health checks
      if (req.method === "GET" && req.url === "/health") {
        res.writeHead(200, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            status: "ok",
            server: config.serverName || "safeinputs",
            version: config.version || "1.0.0",
            tools: config.tools.length,
          })
        );
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
        // Log the full error with stack trace on the server for debugging
        logger.debugError("Error handling request: ", error);

        if (!res.headersSent) {
          res.writeHead(500, { "Content-Type": "application/json" });
          res.end(
            JSON.stringify({
              jsonrpc: "2.0",
              error: {
                code: -32603,
                message: "Internal server error",
              },
              id: null,
            })
          );
        }
      }
    });

    // Start listening
    logger.debug(`Attempting to bind to port ${port}...`);
    httpServer.listen(port, () => {
      logger.debug(`=== Safe Inputs MCP HTTP Server Started Successfully ===`);
      logger.debug(`HTTP server listening on http://localhost:${port}`);
      logger.debug(`MCP endpoint: POST http://localhost:${port}/`);
      logger.debug(`Server name: ${config.serverName || "safeinputs"}`);
      logger.debug(`Server version: ${config.version || "1.0.0"}`);
      logger.debug(`Tools available: ${config.tools.length}`);
      logger.debug(`Server is ready to accept requests`);
    });

    // Handle bind errors
    httpServer.on("error", error => {
      /** @type {NodeJS.ErrnoException} */
      const errnoError = error;
      if (errnoError.code === "EADDRINUSE") {
        logger.debugError(`ERROR: Port ${port} is already in use. `, error);
      } else if (errnoError.code === "EACCES") {
        logger.debugError(`ERROR: Permission denied to bind to port ${port}. `, error);
      } else {
        logger.debugError(`ERROR: Failed to start HTTP server: `, error);
      }
      process.exit(1);
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
  } catch (error) {
    // Log detailed error information for startup failures
    const errorLogger = createLogger("safe-inputs-startup-error");
    errorLogger.debug(`=== FATAL ERROR: Failed to start Safe Inputs MCP HTTP Server ===`);
    if (error && typeof error === "object") {
      if ("constructor" in error && error.constructor) {
        errorLogger.debug(`Error type: ${error.constructor.name}`);
      }
      if ("message" in error) {
        errorLogger.debug(`Error message: ${error.message}`);
      }
      if ("stack" in error && error.stack) {
        errorLogger.debug(`Stack trace:\n${error.stack}`);
      }
      if ("code" in error && error.code) {
        errorLogger.debug(`Error code: ${error.code}`);
      }
    }
    errorLogger.debug(`Configuration file: ${configPath}`);
    errorLogger.debug(`Port: ${port}`);

    // Re-throw the error to be caught by the caller
    throw error;
  }
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
    /** @type {string | undefined} */
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
    console.error(`Error starting HTTP server: ${getErrorMessage(error)}`);
    process.exit(1);
  });
}

module.exports = {
  startHttpServer,
  createMCPServer,
};
