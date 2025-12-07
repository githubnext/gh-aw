// @ts-check
/// <reference types="@actions/github-script" />

/**
 * MCP HTTP Transport Implementation
 *
 * This module provides the HTTP transport layer for the MCP (Model Context Protocol),
 * removing the dependency on @modelcontextprotocol/sdk.
 *
 * Features:
 * - HTTP request/response handling
 * - Session management (stateful and stateless modes)
 * - CORS support for development
 * - JSON-RPC 2.0 compatible
 *
 * References:
 * - MCP Specification: https://spec.modelcontextprotocol.io
 * - JSON-RPC 2.0: https://www.jsonrpc.org/specification
 */

const http = require("http");
const { randomUUID } = require("crypto");
const { MCPServer } = require("./mcp_server.cjs");
const { createLogger } = require("./mcp_logger.cjs");

/**
 * MCP HTTP Transport implementation
 * Handles HTTP requests and converts them to MCP protocol messages
 */
class MCPHTTPTransport {
  /**
   * @param {Object} options - Transport options
   * @param {Function} [options.sessionIdGenerator] - Function that generates session IDs (undefined for stateless)
   * @param {boolean} [options.enableJsonResponse] - Enable JSON responses instead of SSE (default: true for simplicity)
   * @param {boolean} [options.enableDnsRebindingProtection] - Enable DNS rebinding protection (default: false)
   */
  constructor(options = {}) {
    this.sessionIdGenerator = options.sessionIdGenerator;
    this.enableJsonResponse = options.enableJsonResponse !== false; // Default to true
    this.enableDnsRebindingProtection = options.enableDnsRebindingProtection || false;
    this.server = null;
    this.sessionId = null;
    this.started = false;
    this.logger = createLogger("mcp-http-transport");
  }

  /**
   * Set the MCP server instance
   * @param {MCPServer} server - MCP server instance
   */
  setServer(server) {
    this.server = server;
  }

  /**
   * Start the transport
   */
  async start() {
    if (this.started) {
      throw new Error("Transport already started");
    }
    this.started = true;
  }

  /**
   * Handle an incoming HTTP request
   * @param {http.IncomingMessage} req - HTTP request
   * @param {http.ServerResponse} res - HTTP response
   * @param {Object} [parsedBody] - Pre-parsed request body
   */
  async handleRequest(req, res, parsedBody) {
    // Log all incoming requests
    this.logger.debug(`Incoming ${req.method} request to ${req.url}`);
    this.logger.debug(`Headers: ${JSON.stringify(req.headers)}`);

    // Set CORS headers
    res.setHeader("Access-Control-Allow-Origin", "*");
    res.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
    res.setHeader("Access-Control-Allow-Headers", "Content-Type, Accept, Mcp-Session-Id");

    // Handle OPTIONS preflight
    if (req.method === "OPTIONS") {
      this.logger.debug("Handling OPTIONS preflight request");
      res.writeHead(200);
      res.end();
      return;
    }

    // Only handle POST requests for MCP protocol
    if (req.method !== "POST") {
      this.logger.debug(`Rejecting non-POST request: ${req.method}`);
      res.writeHead(405, { "Content-Type": "application/json" });
      res.end(JSON.stringify({ error: "Method not allowed" }));
      return;
    }

    try {
      // Parse request body if not already parsed
      let body = parsedBody;
      if (!body) {
        this.logger.debug("Parsing request body from stream");
        const chunks = [];
        for await (const chunk of req) {
          chunks.push(chunk);
        }
        const bodyStr = Buffer.concat(chunks).toString();
        this.logger.debug(`Request body length: ${bodyStr.length} bytes`);
        try {
          body = bodyStr ? JSON.parse(bodyStr) : null;
          this.logger.debug(`Parsed JSON body: ${JSON.stringify(body)}`);
        } catch (parseError) {
          this.logger.debug(`JSON parse error: ${parseError instanceof Error ? parseError.message : String(parseError)}`);
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
      } else {
        this.logger.debug(`Using pre-parsed body: ${JSON.stringify(body)}`);
      }

      if (!body) {
        this.logger.debug("Empty request body");
        res.writeHead(400, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            jsonrpc: "2.0",
            error: {
              code: -32600,
              message: "Invalid Request: Empty request body",
            },
            id: null,
          })
        );
        return;
      }

      // Validate JSON-RPC structure
      if (!body.jsonrpc || body.jsonrpc !== "2.0") {
        this.logger.debug(`Invalid JSON-RPC version: ${body.jsonrpc}`);
        res.writeHead(400, { "Content-Type": "application/json" });
        res.end(
          JSON.stringify({
            jsonrpc: "2.0",
            error: {
              code: -32600,
              message: "Invalid Request: jsonrpc must be '2.0'",
            },
            id: body.id || null,
          })
        );
        return;
      }

      this.logger.debug(`Processing JSON-RPC method: ${body.method}, id: ${body.id}`);

      // Handle session management for stateful mode
      if (this.sessionIdGenerator) {
        // For initialize, generate a new session ID
        if (body.method === "initialize") {
          this.sessionId = this.sessionIdGenerator();
          this.logger.debug(`Generated new session ID: ${this.sessionId}`);
        } else {
          // For other methods, validate session ID
          const requestSessionId = req.headers["mcp-session-id"];
          this.logger.debug(`Validating session ID from header: ${requestSessionId}`);
          if (!requestSessionId) {
            this.logger.debug("Missing Mcp-Session-Id header");
            res.writeHead(400, { "Content-Type": "application/json" });
            res.end(
              JSON.stringify({
                jsonrpc: "2.0",
                error: {
                  code: -32600,
                  message: "Invalid Request: Missing Mcp-Session-Id header",
                },
                id: body.id || null,
              })
            );
            return;
          }

          if (requestSessionId !== this.sessionId) {
            this.logger.debug(`Session not found: ${requestSessionId} (expected: ${this.sessionId})`);
            res.writeHead(404, { "Content-Type": "application/json" });
            res.end(
              JSON.stringify({
                jsonrpc: "2.0",
                error: {
                  code: -32001,
                  message: "Session not found",
                },
                id: body.id || null,
              })
            );
            return;
          }
          this.logger.debug("Session ID validated successfully");
        }
      }

      // Process the request through the MCP server
      this.logger.debug("Forwarding request to MCP server");
      const response = await this.server.handleRequest(body);
      this.logger.debug(`MCP server response: ${JSON.stringify(response)}`);

      // Handle notifications (null response means no reply needed)
      if (response === null) {
        this.logger.debug("Notification handled (no response)");
        res.writeHead(204); // No Content
        res.end();
        return;
      }

      // Set response headers
      const headers = { "Content-Type": "application/json" };
      if (this.sessionId) {
        headers["mcp-session-id"] = this.sessionId;
      }

      this.logger.debug(`Sending response with headers: ${JSON.stringify(headers)}`);
      res.writeHead(200, headers);
      res.end(JSON.stringify(response));
    } catch (error) {
      this.logger.debugError("Error handling request: ", error);
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
  }
}

module.exports = {
  MCPServer,
  MCPHTTPTransport,
};
