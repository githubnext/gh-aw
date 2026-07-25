// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const safeOutputHandlerManager = require("./safe_output_handler_manager.cjs");

const LOGS_DIR = "/tmp/gh-aw";
const STDOUT_LOG = "process-safe-outputs.stdout.log";
const STDERR_LOG = "process-safe-outputs.stderr.log";

/**
 * End a writable stream, rejecting if an error occurs during close.
 * @param {import("fs").WriteStream} stream
 * @returns {Promise<void>}
 */
function endStream(stream) {
  return new Promise((resolve, reject) => {
    stream.once("error", reject);
    stream.end(() => resolve());
  });
}

/**
 * Capture process output while safe-output handlers execute.
 */
async function main() {
  try {
    fs.mkdirSync(LOGS_DIR, { recursive: true });
  } catch (err) {
    throw new Error(`Failed to create logs directory: ${getErrorMessage(err)}`, { cause: err });
  }

  const stdoutStream = fs.createWriteStream(path.join(LOGS_DIR, STDOUT_LOG), { flags: "w" });
  const stderrStream = fs.createWriteStream(path.join(LOGS_DIR, STDERR_LOG), { flags: "w" });
  stdoutStream.on("error", err => core.warning(`stdout log write error: ${getErrorMessage(err)}`));
  stderrStream.on("error", err => core.warning(`stderr log write error: ${getErrorMessage(err)}`));

  const originalStdoutWrite = process.stdout.write.bind(process.stdout);
  const originalStderrWrite = process.stderr.write.bind(process.stderr);

  // TypeScript cannot verify that a single arrow function satisfies a multi-overload
  // interface. Cast via any so the assignment passes the type checker; the runtime
  // behaviour is correct (both 2-arg and 3-arg Node.js write() forms are forwarded).
  /** @type {any} */
  const hookedStdoutWrite = (/** @type {string | Uint8Array} */ chunk, /** @type {BufferEncoding | undefined} */ encoding, /** @type {((err?: Error | null) => void) | undefined} */ callback) => {
    if (typeof encoding === "string") {
      stdoutStream.write(chunk, encoding);
    } else {
      stdoutStream.write(chunk);
    }
    return originalStdoutWrite(chunk, encoding, callback);
  };

  /** @type {any} */
  const hookedStderrWrite = (/** @type {string | Uint8Array} */ chunk, /** @type {BufferEncoding | undefined} */ encoding, /** @type {((err?: Error | null) => void) | undefined} */ callback) => {
    if (typeof encoding === "string") {
      stderrStream.write(chunk, encoding);
    } else {
      stderrStream.write(chunk);
    }
    return originalStderrWrite(chunk, encoding, callback);
  };

  process.stdout.write = hookedStdoutWrite;
  process.stderr.write = hookedStderrWrite;

  try {
    await safeOutputHandlerManager.main();
  } finally {
    process.stdout.write = originalStdoutWrite;
    process.stderr.write = originalStderrWrite;
    await Promise.all([endStream(stdoutStream), endStream(stderrStream)]);
  }
}

module.exports = { main };
