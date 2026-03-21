// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Shared HTTP MCP Server Runner
 *
 * This module provides the common HTTP server lifecycle implementation shared
 * between mcp_scripts_mcp_server_http.cjs and safe_outputs_mcp_server_http.cjs.
 *
 * It handles:
 * - CORS headers and OPTIONS preflight
 * - GET /health endpoint
 * - POST body parsing and JSON-RPC request routing
 * - Per-request error handling
 * - Bind error handling (EADDRINUSE, EACCES)
 * - Graceful shutdown on SIGINT / SIGTERM
 *
 * @module mcp_http_server_runner
 */

const http = require("http");
const { createLogger } = require("./mcp_logger.cjs");

/**
 * Start an HTTP server that routes MCP protocol requests to the given transport.
 *
 * @param {Object} options - Runner options
 * @param {import('./mcp_http_transport.cjs').MCPHTTPTransport} options.transport - Connected MCPHTTPTransport instance
 * @param {number} options.port - Port to listen on
 * @param {Function} options.getHealthPayload - Zero-argument function that returns the JSON object for GET /health responses
 * @param {Object} options.logger - Logger instance (from mcp_logger.cjs createLogger)
 * @param {string} options.serverLabel - Human-readable label used in log messages (e.g. "MCP Scripts")
 * @param {Function} [options.configureServer] - Optional callback called with the http.Server instance before listen, useful for setting timeouts
 * @returns {Promise<http.Server>} The started HTTP server
 */
async function runHttpServer(options) {
  const { transport, port, getHealthPayload, logger, serverLabel, configureServer } = options;

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
      res.end(JSON.stringify(getHealthPayload()));
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

  // Allow callers to configure the server before binding (e.g. set timeouts)
  if (configureServer) {
    configureServer(httpServer);
  }

  // Start listening
  logger.debug(`Attempting to bind to port ${port}...`);
  httpServer.listen(port, () => {
    logger.debug(`=== ${serverLabel} HTTP Server Started Successfully ===`);
    logger.debug(`HTTP server listening on http://localhost:${port}`);
    logger.debug(`MCP endpoint: POST http://localhost:${port}/`);
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
}

/**
 * Log detailed startup error information and re-throw.
 *
 * @param {unknown} error - The caught error
 * @param {string} namespace - Logger namespace for the error logger (e.g. "mcp-scripts-startup-error")
 * @param {Object} context - Key/value pairs to include in log output
 * @returns {never}
 */
function logStartupError(error, namespace, context) {
  const errorLogger = createLogger(namespace);
  errorLogger.debug(`=== FATAL ERROR: Failed to start HTTP Server ===`);
  if (error && typeof error === "object") {
    if ("constructor" in error && /** @type {any} */ error.constructor) {
      errorLogger.debug(`Error type: ${/** @type {any} */ error.constructor.name}`);
    }
    if ("message" in error) {
      errorLogger.debug(`Error message: ${/** @type {any} */ error.message}`);
    }
    if ("stack" in error && /** @type {any} */ error.stack) {
      errorLogger.debug(`Stack trace:\n${/** @type {any} */ error.stack}`);
    }
    if ("code" in error && /** @type {any} */ error.code) {
      errorLogger.debug(`Error code: ${/** @type {any} */ error.code}`);
    }
  }
  for (const [key, value] of Object.entries(context)) {
    errorLogger.debug(`${key}: ${value}`);
  }
  throw error;
}

module.exports = {
  runHttpServer,
  logStartupError,
};
