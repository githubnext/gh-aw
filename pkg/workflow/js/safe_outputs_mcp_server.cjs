// @ts-check
/// <reference types="@actions/github-script" />

const { createServer, registerTool, normalizeTool, start } = require("./mcp_server_core.cjs");
const { loadConfig } = require("./safe_outputs_config.cjs");
const { createAppendFunction } = require("./safe_outputs_append.cjs");
const { createHandlers } = require("./safe_outputs_handlers.cjs");
const { loadTools, attachHandlers, registerPredefinedTools, registerDynamicTools } = require("./safe_outputs_tools_loader.cjs");

// Server info for safe outputs MCP server
const SERVER_INFO = { name: "safeoutputs", version: "1.0.0" };

// Create the server instance with optional log directory
const MCP_LOG_DIR = process.env.GH_AW_MCP_LOG_DIR;
const server = createServer(SERVER_INFO, { logDir: MCP_LOG_DIR });

// Load configuration
const { config: safeOutputsConfig, outputFile } = loadConfig(server);

// Create append function
const appendSafeOutput = createAppendFunction(outputFile);

// Create handlers
const handlers = createHandlers(server, appendSafeOutput);
const { defaultHandler } = handlers;

// Load tools
let ALL_TOOLS = loadTools(server);

// Attach handlers to tools
ALL_TOOLS = attachHandlers(ALL_TOOLS, handlers);

server.debug(`  output file: ${outputFile}`);
server.debug(`  config: ${JSON.stringify(safeOutputsConfig)}`);

// Register predefined tools that are enabled in configuration
registerPredefinedTools(server, ALL_TOOLS, safeOutputsConfig, registerTool, normalizeTool);

// Add safe-jobs as dynamic tools
registerDynamicTools(server, ALL_TOOLS, safeOutputsConfig, outputFile, registerTool, normalizeTool);

server.debug(`  tools: ${Object.keys(server.tools).join(", ")}`);
if (!Object.keys(server.tools).length) throw new Error("No tools enabled in configuration");

// Start the server with the default handler
start(server, { defaultHandler });
