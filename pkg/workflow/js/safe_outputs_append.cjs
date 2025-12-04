// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");

/**
 * Create an append function for the safe outputs file
 * @param {string} outputFile - Path to the output file
 * @returns {Function} A function that appends entries to the safe outputs file
 */
function createAppendFunction(outputFile) {
  /**
   * Append an entry to the safe outputs file
   * @param {Object} entry - The entry to append
   */
  return function appendSafeOutput(entry) {
    if (!outputFile) throw new Error("No output file configured");
    // Normalize type to use underscores (convert any dashes to underscores)
    entry.type = entry.type.replace(/-/g, "_");
    const jsonLine = JSON.stringify(entry) + "\n";
    try {
      fs.appendFileSync(outputFile, jsonLine);
    } catch (error) {
      throw new Error(`Failed to write to output file: ${error instanceof Error ? error.message : String(error)}`);
    }
  };
}

module.exports = { createAppendFunction };
