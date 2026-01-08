// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Safe Outputs MCP Server with HTTP Transport
 *
 * This module extends the safe-outputs MCP server to support HTTP transport
 * using the StreamableHTTPServerTransport from the MCP SDK.
 *
 * It provides both stateful and stateless HTTP modes, as well as SSE streaming.
 *
 * Usage:
 *   node safe_outputs_mcp_server_http.cjs [--port 3000] [--stateless]
 *
 * Options:
 *   --port <number>    Port to listen on (default: 3000)
 *   --stateless        Run in stateless mode (no session management)
 *   --log-dir <path>   Directory for log files
 */

const http = require("http");
const { randomUUID } = require("crypto");
const { MCPServer, MCPHTTPTransport } = require("./mcp_http_transport.cjs");
const { createLogger } = require("./mcp_logger.cjs");
const { bootstrapSafeOutputsServer, cleanupConfigFile } = require("./safe_outputs_bootstrap.cjs");
const { createAppendFunction } = require("./safe_outputs_append.cjs");
const { createHandlers } = require("./safe_outputs_handlers.cjs");
const { attachHandlers, registerPredefinedTools, registerDynamicTools } = require("./safe_outputs_tools_loader.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Create and configure the MCP server with tools
 * @param {Object} [options] - Additional options
 * @param {string} [options.logDir] - Override log directory from config
 * @returns {Object} Server instance and configuration
 */
function createMCPServer(options = {}) {
  // Create logger early
  const MCP_LOG_DIR = options.logDir || process.env.GH_AW_MCP_LOG_DIR;
  const logger = createLogger("safeoutputs", MCP_LOG_DIR);

  logger.debug(`=== Creating MCP Server ===`);

  // Bootstrap: load configuration and tools using shared logic
  const { config: safeOutputsConfig, outputFile, tools: ALL_TOOLS } = bootstrapSafeOutputsServer(logger);

  // Create server with configuration
  const serverName = "safeoutputs";
  const version = "1.0.0";

  logger.debug(`Server name: ${serverName}`);
  logger.debug(`Server version: ${version}`);
  logger.debug(`Output file: ${outputFile}`);
  logger.debug(`Config: ${JSON.stringify(safeOutputsConfig)}`);

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

  // Create append function
  const appendSafeOutput = createAppendFunction(outputFile);

  // Create handlers with configuration
  const handlers = createHandlers(logger, appendSafeOutput, safeOutputsConfig);
  const { defaultHandler } = handlers;

  // Attach handlers to tools
  const toolsWithHandlers = attachHandlers(ALL_TOOLS, handlers);

  logger.debug(`Registering tools with MCP server...`);
  let registeredCount = 0;
  let skippedCount = 0;

  // Helper to register a single tool
  const registerTool = tool => {
    if (!tool.handler) {
      logger.debug(`Skipping tool ${tool.name} - no handler loaded`);
      skippedCount++;
      return;
    }

    logger.debug(`Registering tool: ${tool.name}`);

    // Register the tool with the MCP SDK using the high-level API
    // The callback receives the arguments directly as the first parameter
    server.tool(tool.name, tool.description || "", tool.inputSchema || { type: "object", properties: {} }, async args => {
      logger.debug(`Calling handler for tool: ${tool.name}`);

      // Call the handler
      const result = await Promise.resolve(tool.handler(args));
      logger.debug(`Handler returned for tool: ${tool.name}`);

      // Normalize result to MCP format
      const content = result && result.content ? result.content : [];
      return { content, isError: false };
    });

    registeredCount++;
  };

  // Helper to normalize tool structure
  const normalizeTool = tool => {
    return {
      name: tool.name,
      description: tool.description || "",
      inputSchema: tool.inputSchema || { type: "object", properties: {} },
      handler: tool.handler,
    };
  };

  // Register predefined tools that are enabled in configuration
  // Note: We're using inline implementation rather than calling the loader function
  // because we need to register with MCP SDK instead of mcp_server_core
  const PREDEFINED_TOOLS = ["create_issue", "update_issue", "add_comment", "add_labels", "create_pull_request", "create_discussion", "update_discussion", "create_code_scanning_alert", "create_pr_review_comment", "close_discussion"];

  for (const toolName of PREDEFINED_TOOLS) {
    if (safeOutputsConfig[toolName] === undefined) {
      logger.debug(`  skipping ${toolName} (not in config)`);
      continue;
    }

    // Find the tool in the tools with handlers
    const tool = toolsWithHandlers[toolName];
    if (!tool) {
      logger.debug(`  warning: ${toolName} in config but not in available tools`);
      continue;
    }

    registerTool(normalizeTool(tool));
  }

  // Add safe-jobs as dynamic tools
  if (safeOutputsConfig.safe_jobs) {
    const safeJobsTools = safeOutputsConfig.safe_jobs;
    logger.debug(`Adding ${Object.keys(safeJobsTools).length} safe-jobs as dynamic tools`);

    for (const [jobName, jobConfig] of Object.entries(safeJobsTools)) {
      // Create a tool definition for this safe-job
      const tool = toolsWithHandlers.safe_jobs;
      if (!tool) {
        logger.debug(`  warning: safe_jobs tool not found in available tools`);
        continue;
      }

      // Create a wrapped handler that passes the job name
      const wrappedHandler = async args => {
        return tool.handler({
          ...args,
          job: jobName,
        });
      };

      registerTool(
        normalizeTool({
          name: jobName,
          description: jobConfig.description || `Execute safe job: ${jobName}`,
          inputSchema: {
            type: "object",
            properties: {},
            additionalProperties: true,
          },
          handler: wrappedHandler,
        })
      );
    }
  }

  logger.debug(`Tool registration complete: ${registeredCount} registered, ${skippedCount} skipped`);

  if (!registeredCount) {
    logger.debug(`WARNING: No tools enabled in configuration - server will start but no tools will be available`);
    logger.debug(`This may indicate an empty safe-outputs configuration or missing tool definitions`);
  }

  logger.debug(`=== MCP Server Creation Complete ===`);

  // Note: We do NOT cleanup the config file here because it's needed by the ingestion
  // phase (collect_ndjson_output.cjs) that runs after the MCP server completes.
  // The config file only contains schema information (no secrets), so it's safe to leave.

  return { server, config: safeOutputsConfig, logger, registeredCount };
}

/**
 * Start the HTTP server with MCP protocol support
 * @param {Object} options - Server options
 * @param {number} [options.port] - Port to listen on (default: 3000)
 * @param {boolean} [options.stateless] - Run in stateless mode (default: false)
 * @param {string} [options.logDir] - Override log directory from config
 */
async function startHttpServer(options = {}) {
  const port = options.port || 3000;
  const stateless = options.stateless || false;

  const logger = createLogger("safe-outputs-startup");

  logger.debug(`=== Starting Safe Outputs MCP HTTP Server ===`);
  logger.debug(`Port: ${port}`);
  logger.debug(`Mode: ${stateless ? "stateless" : "stateful"}`);
  logger.debug(`Environment: NODE_VERSION=${process.version}, PLATFORM=${process.platform}`);

  // Create the MCP server
  try {
    const { server, config, logger: mcpLogger, registeredCount } = createMCPServer({ logDir: options.logDir });

    // Use the MCP logger for subsequent messages
    Object.assign(logger, mcpLogger);

    logger.debug(`MCP server created successfully`);
    logger.debug(`Server name: safeoutputs`);
    logger.debug(`Server version: 1.0.0`);
    logger.debug(`Configuration items: ${Object.keys(config).length}`);
    logger.debug(`Registered tools: ${registeredCount}`);

    if (registeredCount === 0) {
      logger.debug(`WARNING: No tools registered - server will run but no tools will be available to the MCP client`);
    }

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
            server: "safeoutputs",
            version: "1.0.0",
            tools: registeredCount,
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
      logger.debug(`=== Safe Outputs MCP HTTP Server Started Successfully ===`);
      logger.debug(`HTTP server listening on http://localhost:${port}`);
      logger.debug(`MCP endpoint: POST http://localhost:${port}/`);
      logger.debug(`Server name: safeoutputs`);
      logger.debug(`Server version: 1.0.0`);
      logger.debug(`Tools available: ${Object.keys(config).length}`);
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
    const errorLogger = createLogger("safe-outputs-startup-error");
    errorLogger.debug(`=== FATAL ERROR: Failed to start Safe Outputs MCP HTTP Server ===`);
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
    errorLogger.debug(`Port: ${port}`);

    // Re-throw the error to be caught by the caller
    throw error;
  }
}

// If run directly, start the HTTP server with command-line arguments
if (require.main === module) {
  const args = process.argv.slice(2);

  const options = {
    port: 3000,
    stateless: false,
    /** @type {string | undefined} */
    logDir: undefined,
  };

  // Parse optional arguments
  for (let i = 0; i < args.length; i++) {
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

  startHttpServer(options).catch(error => {
    console.error(`Error starting HTTP server: ${getErrorMessage(error)}`);
    process.exit(1);
  });
}

module.exports = {
  startHttpServer,
  createMCPServer,
};
