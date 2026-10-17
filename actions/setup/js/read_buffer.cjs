// @ts-check
/// <reference types="@actions/github-script" />

const { JsonlParser } = require("./jsonl_helpers.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_PARSE } = require("./error_codes.cjs");

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
    this._parser = new JsonlParser();
    /** @type {Generator<import("./jsonl_helpers.cjs").JsonlEvent>[]} */
    this._lines = [];
    this._offset = 0;
  }

  /**
   * Append data to the buffer
   * @param {Buffer} chunk - Data chunk to append
   */
  append(chunk) {
    this._lines.push(this._parser.push(chunk));
  }

  /**
   * Read a complete message from the buffer
   * @returns {Object|null} Parsed JSON message or null if no complete message
   */
  readMessage() {
    while (this._offset < this._lines.length) {
      const next = this._lines[this._offset].next();
      if (next.done) {
        delete this._lines[this._offset++];
        if (this._offset === this._lines.length) {
          this._lines = [];
          this._offset = 0;
        } else if (this._offset >= 1024 && this._offset * 2 >= this._lines.length) {
          this._lines = this._lines.slice(this._offset);
          this._offset = 0;
        }
        continue;
      }
      const event = next.value;
      if (event.type === "invalid") {
        if (event.reason === "blank") continue;
        const error = event.error;
        throw new Error(`${ERR_PARSE}: Parse error: ${getErrorMessage(error)}`, { cause: error });
      }
      return event.value;
    }
    return null;
  }
}

module.exports = {
  ReadBuffer,
};
