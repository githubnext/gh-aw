// @ts-check
"use strict";

const crypto = require("crypto");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Escape a workflow command message.
 *
 * @param {string} value
 * @returns {string}
 */
function escapeWorkflowCommandMessage(value) {
  return value.replace(/%/g, "%25").replace(/\r/g, "%0D").replace(/\n/g, "%0A");
}

/**
 * Write process output inside a collapsible Actions log group while workflow
 * command processing is disabled.
 *
 * @param {string} title
 * @param {NodeJS.WritableStream} [output]
 * @returns {{write: (chunk: string | Uint8Array) => boolean, end: () => void}}
 */
function createActionLogGroup(title, output = process.stdout) {
  const token = `gh-aw-${crypto.randomUUID()}`;
  let ended = false;
  let endsWithNewline = true;

  output.write(`::group::${escapeWorkflowCommandMessage(title)}\n`);
  output.write(`::stop-commands::${token}\n`);

  return {
    write(chunk) {
      if (ended) {
        throw new Error("Cannot write to a closed action log group");
      }
      const value = Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk);
      if (value.length === 0) return true;
      endsWithNewline = value[value.length - 1] === 0x0a;
      return output.write(value);
    },
    end() {
      if (ended) return;
      ended = true;
      if (!endsWithNewline) output.write("\n");
      output.write(`::${token}::\n`);
      output.write("::endgroup::\n");
    },
  };
}

/**
 * Default cap on the total number of bytes buffered in memory by
 * captureProcessOutput(). This bounds worst-case memory use when a captured
 * process (or a misbehaving handler) produces unexpectedly large or
 * unbounded output. Once the cap is reached, further bytes are counted
 * towards `totalBytes` but are not retained, and `truncated` is set so
 * callers can surface a clear, non-user-controlled notice.
 */
const DEFAULT_CAPTURE_MAX_BYTES = 5 * 1024 * 1024;

/**
 * Capture writes made through process.stdout/process.stderr in memory.
 *
 * @param {{maxBytes?: number}} [options]
 * @returns {{entries: Array<{stream: "stdout" | "stderr", chunk: Buffer}>, maxBytes: number, capturedBytes: number, totalBytes: number, truncated: boolean, restore: () => void}}
 */
function captureProcessOutput({ maxBytes = DEFAULT_CAPTURE_MAX_BYTES } = {}) {
  /** @type {Array<{stream: "stdout" | "stderr", chunk: Buffer}>} */
  const entries = [];
  const originalStdoutWrite = process.stdout.write;
  const originalStderrWrite = process.stderr.write;
  let restored = false;
  let capturedBytes = 0;
  let totalBytes = 0;
  let truncated = false;

  /**
   * @param {"stdout" | "stderr"} stream
   * @returns {typeof process.stdout.write}
   */
  function createCaptureWrite(stream) {
    return (
      /** @type {typeof process.stdout.write} */ /**
       * @param {string | Uint8Array} chunk
       * @param {BufferEncoding | (() => void)} [encoding]
       * @param {() => void} [callback]
       */
      (chunk, encoding, callback) => {
        const resolvedEncoding = typeof encoding === "string" ? encoding : undefined;
        let bufferedChunk;
        if (Buffer.isBuffer(chunk)) {
          bufferedChunk = Buffer.from(chunk);
        } else if (typeof chunk === "string") {
          bufferedChunk = Buffer.from(chunk, resolvedEncoding);
        } else {
          bufferedChunk = Buffer.from(chunk);
        }
        totalBytes += bufferedChunk.length;
        if (!truncated && bufferedChunk.length > 0) {
          const remaining = maxBytes - capturedBytes;
          if (bufferedChunk.length <= remaining) {
            entries.push({ stream, chunk: bufferedChunk });
            capturedBytes += bufferedChunk.length;
          } else {
            if (remaining > 0) {
              entries.push({ stream, chunk: bufferedChunk.subarray(0, remaining) });
              capturedBytes += remaining;
            }
            truncated = true;
          }
        }
        const resolvedCallback = typeof encoding === "function" ? encoding : callback;
        resolvedCallback?.();
        return true;
      }
    );
  }

  process.stdout.write = createCaptureWrite("stdout");
  process.stderr.write = createCaptureWrite("stderr");

  return {
    entries,
    maxBytes,
    get capturedBytes() {
      return capturedBytes;
    },
    get totalBytes() {
      return totalBytes;
    },
    get truncated() {
      return truncated;
    },
    restore() {
      if (restored) return;
      restored = true;
      process.stdout.write = originalStdoutWrite;
      process.stderr.write = originalStderrWrite;
    },
  };
}

/**
 * Send captured stdout/stderr entries to the Actions log.
 *
 * Accepts either a plain array of entries (legacy shape) or the full object
 * returned by captureProcessOutput(), which additionally carries truncation
 * metadata. When the capture was truncated, a static, non-user-controlled
 * notice is appended inside the same protected (stop-commands-guarded) group
 * so it cannot be mistaken for an interpretable workflow command.
 *
 * @param {string} title
 * @param {Array<{stream: "stdout" | "stderr", chunk: Buffer}> | {entries: Array<{stream: "stdout" | "stderr", chunk: Buffer}>, truncated?: boolean, capturedBytes?: number, totalBytes?: number, maxBytes?: number}} capture
 */
function logCapturedProcessOutput(title, capture) {
  const entries = Array.isArray(capture) ? capture : capture.entries;
  const truncated = !Array.isArray(capture) && Boolean(capture.truncated);
  if (entries.length === 0 && !truncated) return;
  const group = createActionLogGroup(title);
  for (const entry of entries) {
    group.write(entry.chunk);
  }
  if (truncated && !Array.isArray(capture)) {
    group.write(`\n[gh-aw] Captured output truncated: kept ${capture.capturedBytes} of ${capture.totalBytes} bytes (limit ${capture.maxBytes} bytes). Additional output was discarded to bound memory use.\n`);
  }
  group.end();
}

/**
 * Maximum number of bytes read from a process output file before it is logged.
 * When a file exceeds this size, only the trailing portion (most recent output,
 * where the actionable error is usually found) is emitted, with a note indicating
 * how much was omitted. This keeps a runaway or verbose process from flooding the
 * Actions log with megabytes of output.
 */
const DEFAULT_MAX_LOG_FILE_BYTES = 64 * 1024;

/**
 * Reads up to `maxBytes` from the end of a file.
 *
 * @param {string} filePath
 * @param {number} maxBytes
 * @returns {{contents: string, truncated: boolean, totalBytes: number}}
 */
function readFileTail(filePath, maxBytes) {
  const fs = require("fs");
  try {
    const stat = fs.statSync(filePath);
    if (stat.size <= maxBytes) {
      return { contents: fs.readFileSync(filePath, "utf8"), truncated: false, totalBytes: stat.size };
    }
    const fd = fs.openSync(filePath, "r");
    try {
      const buffer = Buffer.alloc(maxBytes);
      const start = stat.size - maxBytes;
      fs.readSync(fd, buffer, 0, maxBytes, start);
      return { contents: buffer.toString("utf8"), truncated: true, totalBytes: stat.size };
    } finally {
      fs.closeSync(fd);
    }
  } catch (error) {
    throw new Error(`Failed to read process output file '${filePath}': ${getErrorMessage(error)}`, { cause: error });
  }
}

/**
 * Log a process output file after it has been redacted.
 *
 * @param {string} title
 * @param {string[]} filePaths
 * @param {string[]} [redactions]
 * @param {number} [maxBytes] - Maximum number of trailing bytes to emit per file.
 */
function logProcessOutputFiles(title, filePaths, redactions = [], maxBytes = DEFAULT_MAX_LOG_FILE_BYTES) {
  const fs = require("fs");
  const path = require("path");
  const existingPaths = filePaths.filter(filePath => {
    if (!fs.existsSync(filePath)) return false;
    try {
      return fs.statSync(filePath).size > 0;
    } catch (error) {
      throw new Error(`Failed to inspect process output file '${filePath}': ${getErrorMessage(error)}`, { cause: error });
    }
  });
  if (existingPaths.length === 0) return;

  const group = createActionLogGroup(title);
  for (const filePath of existingPaths) {
    group.write(`--- ${path.basename(filePath)} ---\n`);
    const { contents: raw, truncated, totalBytes } = readFileTail(filePath, maxBytes);
    let contents = raw;
    for (const redaction of redactions) {
      if (redaction) contents = contents.replaceAll(redaction, "***");
    }
    if (truncated) {
      group.write(`[truncated: showing last ${maxBytes} of ${totalBytes} bytes]\n`);
    }
    group.write(contents);
  }
  group.end();
}

if (require.main === module) {
  const title = process.argv[2] || "Process output";
  const group = createActionLogGroup(title);
  process.stdin.on("data", chunk => {
    if (!group.write(chunk)) {
      process.stdin.pause();
      process.stdout.once("drain", () => process.stdin.resume());
    }
  });
  process.stdin.on("end", () => group.end());
  process.stdin.on("error", error => {
    group.write(`Failed to read process output: ${error.message}\n`);
    group.end();
    process.exitCode = 1;
  });
  process.stdin.resume();
}

module.exports = {
  DEFAULT_CAPTURE_MAX_BYTES,
  DEFAULT_MAX_LOG_FILE_BYTES,
  captureProcessOutput,
  createActionLogGroup,
  escapeWorkflowCommandMessage,
  logCapturedProcessOutput,
  logProcessOutputFiles,
  readFileTail,
};
