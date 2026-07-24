// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { main: runSafeOutputHandlerManager } = require("./safe_output_handler_manager.cjs");

const LOGS_DIR = "/tmp/gh-aw";
const STDOUT_LOG = "process-safe-outputs.stdout.log";
const STDERR_LOG = "process-safe-outputs.stderr.log";

/**
 * Capture process output while safe-output handlers execute.
 */
async function main() {
  try {
    fs.mkdirSync(LOGS_DIR, { recursive: true });
  } catch (err) {
    throw new Error(`Failed to create logs directory: ${getErrorMessage(err)}`, { cause: err });
  }

  const stdoutStream = fs.createWriteStream(path.join(LOGS_DIR, STDOUT_LOG), { flags: "a" });
  const stderrStream = fs.createWriteStream(path.join(LOGS_DIR, STDERR_LOG), { flags: "a" });
  const originalStdoutWrite = process.stdout.write.bind(process.stdout);
  const originalStderrWrite = process.stderr.write.bind(process.stderr);

  process.stdout.write = (chunk, encoding, callback) => {
    stdoutStream.write(chunk);
    return originalStdoutWrite(chunk, encoding, callback);
  };

  process.stderr.write = (chunk, encoding, callback) => {
    stderrStream.write(chunk);
    return originalStderrWrite(chunk, encoding, callback);
  };

  try {
    await runSafeOutputHandlerManager();
  } finally {
    process.stdout.write = originalStdoutWrite;
    process.stderr.write = originalStderrWrite;
    await Promise.all([new Promise(resolve => stdoutStream.end(resolve)), new Promise(resolve => stderrStream.end(resolve))]);
  }
}

module.exports = { main };
