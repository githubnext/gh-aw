// @ts-check
/// <reference types="@actions/github-script" />

/**
 * MCP STDIO Transport Module
 *
 * This module provides the stdio transport implementation for MCP servers.
 * It handles reading from stdin and writing to stdout using JSON-RPC 2.0 over newline-delimited JSON.
 *
 * Usage:
 *   const { StdioTransport } = require("./mcp_stdio_transport.cjs");
 *
 *   const transport = new StdioTransport();
 *   transport.onMessage(message => handleMessage(message));
 *   transport.start();
 *   transport.send({ jsonrpc: "2.0", id: 1, result: { ... } });
 */

const fs = require("fs");

const { ReadBuffer } = require("./read_buffer.cjs");

const encoder = new TextEncoder();

/**
 * @typedef {Object} StdioTransportOptions
 * @property {Function} [onDebug] - Debug logging function
 */

/**
 * STDIO Transport for MCP servers
 * Handles reading from stdin and writing to stdout using newline-delimited JSON
 */
class StdioTransport {
  /**
   * Create a new STDIO transport
   * @param {StdioTransportOptions} [options] - Transport options
   */
  constructor(options = {}) {
    /** @type {ReadBuffer} */
    this._readBuffer = new ReadBuffer();

    /** @type {Function|null} */
    this._messageHandler = null;

    /** @type {Function} */
    this._debug = options.onDebug || (() => {});

    /** @type {boolean} */
    this._started = false;
  }

  /**
   * Set the message handler callback
   * @param {Function} handler - Function to call when a message is received
   */
  onMessage(handler) {
    this._messageHandler = handler;
  }

  /**
   * Send a message over the transport
   * @param {Object} message - The JSON-RPC message to send
   */
  send(message) {
    const json = JSON.stringify(message);
    this._debug(`send: ${json}`);
    const messageStr = json + "\n";
    const bytes = encoder.encode(messageStr);
    fs.writeSync(1, bytes);
  }

  /**
   * Process the read buffer and dispatch messages to the handler
   * @private
   */
  _processBuffer() {
    while (true) {
      try {
        const message = this._readBuffer.readMessage();
        if (!message) {
          break;
        }
        this._debug(`recv: ${JSON.stringify(message)}`);
        if (this._messageHandler) {
          this._messageHandler(message);
        }
      } catch (error) {
        // For parse errors, we can't know the request id, so we shouldn't send a response
        // according to JSON-RPC spec. Just log the error.
        this._debug(`Parse error: ${error instanceof Error ? error.message : String(error)}`);
      }
    }
  }

  /**
   * Start listening on stdin
   */
  start() {
    if (this._started) {
      return;
    }
    this._started = true;

    this._debug(`stdio transport starting...`);

    const onData = chunk => {
      this._readBuffer.append(chunk);
      this._processBuffer();
    };

    process.stdin.on("data", onData);
    process.stdin.on("error", err => this._debug(`stdin error: ${err}`));
    process.stdin.resume();
    this._debug(`stdio transport listening...`);
  }

  /**
   * Close the transport
   */
  close() {
    if (!this._started) {
      return;
    }
    this._debug(`stdio transport closing...`);
    // Note: We don't pause stdin here as it might affect other parts of the app
    // The transport is effectively closed by not processing any more messages
    this._started = false;
    this._messageHandler = null;
  }

  /**
   * Check if the transport is running
   * @returns {boolean}
   */
  isRunning() {
    return this._started;
  }
}

/**
 * Create a new STDIO transport instance
 * @param {StdioTransportOptions} [options] - Transport options
 * @returns {StdioTransport}
 */
function createStdioTransport(options = {}) {
  return new StdioTransport(options);
}

module.exports = {
  StdioTransport,
  createStdioTransport,
};
