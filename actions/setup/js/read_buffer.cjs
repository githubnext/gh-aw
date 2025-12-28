// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * ReadBuffer Module
 *
 * This module provides a buffer class for parsing JSON-RPC messages from stdin.
 * It handles line-by-line reading and JSON parsing with support for both
 * Unix (\n) and Windows (\r\n) line endings.
 *
 * Usage:
 *   const { ReadBuffer } = require("./read_buffer.cjs");
 *
 *   const buffer = new ReadBuffer();
 *   buffer.append(chunk);
 *   const message = buffer.readMessage();
 */

/**
 * ReadBuffer class for parsing JSON-RPC messages from stdin
 */
class ReadBuffer {
  constructor() {
    /** @type {Buffer|null} */
    this._buffer = null;
  }

  /**
   * Append data to the buffer
   * @param {Buffer} chunk - Data chunk to append
   */
  append(chunk) {
    this._buffer = this._buffer ? Buffer.concat([this._buffer, chunk]) : chunk;
  }

  /**
   * Read a complete message from the buffer
   * @returns {Object|null} Parsed JSON message or null if no complete message
   */
  readMessage() {
    if (!this._buffer) {
      return null;
    }

    const index = this._buffer.indexOf("\n");
    if (index === -1) {
      return null;
    }

    const line = this._buffer.toString("utf8", 0, index).replace(/\r$/, "");
    this._buffer = this._buffer.subarray(index + 1);

    if (line.trim() === "") {
      return this.readMessage(); // Skip empty lines recursively
    }

    try {
      return JSON.parse(line);
    } catch (error) {
      throw new Error(`Parse error: ${getErrorMessage(error)}`);
    }
  }
}

module.exports = {
  ReadBuffer,
};
