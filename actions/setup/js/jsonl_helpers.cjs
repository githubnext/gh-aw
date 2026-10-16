// @ts-check

const fs = require("node:fs");
const { StringDecoder } = require("node:string_decoder");

/**
 * @typedef {object} JsonlOptions
 * @property {(line: string, lineNumber: number) => boolean} [lineFilter] Runs before JSON.parse.
 * @property {(error: Error, lineNumber: number) => void} [onError] Reports rejected nonblank lines without exposing their contents.
 * @property {number} [maxLineLength] Optional limit in UTF-16 code units; unlimited by default.
 */

/** @param {string} line */
function isJsonCandidate(line) {
  if (line[0] === "{") return /^\{\s*["}]/.test(line);
  if (line[0] === "[") return /^\[\s*(?:[{\["\d\]-]|(?:true|false|null)(?:[\s,\]]|$))/.test(line);
  return /^(?:["\d-]|(?:true|false|null)$)/.test(line);
}

/**
 * Strict single-record parser for protocols and callers with their own error policy.
 * @param {string} line
 * @returns {any}
 */
function parseJsonlLine(line) {
  const trimmed = line.trim();
  if (!isJsonCandidate(trimmed)) {
    throw new SyntaxError("Invalid JSONL record");
  }
  try {
    return JSON.parse(trimmed);
  } catch {
    throw new SyntaxError("Invalid JSONL record");
  }
}

/**
 * Lazily split existing text without allocating an array of every line.
 * LF and CRLF are supported; a final unterminated line is included.
 * @param {string} content
 */
function* iterateLines(content) {
  let start = 0;
  while (start < content.length) {
    const end = content.indexOf("\n", start);
    if (end === -1) {
      yield content.slice(start).replace(/\r$/, "");
      return;
    }
    yield content.slice(start, end).replace(/\r$/, "");
    start = end + 1;
  }
  yield "";
}

/**
 * Chunk framing retains only the current line, joining fragments once rather than
 * repeatedly copying a growing multi-megabyte string.
 */
class LineFramer {
  constructor() {
    /** @type {string[]} */
    this.fragments = [];
  }

  /** @param {string} chunk */
  *push(chunk) {
    let start = 0;
    let end;
    while ((end = chunk.indexOf("\n", start)) !== -1) {
      const part = chunk.slice(start, end);
      const line = this.fragments.length ? this.fragments.join("") + part : part;
      this.fragments = [];
      start = end + 1;
      yield line.replace(/\r$/, "");
    }
    if (start < chunk.length) this.fragments.push(chunk.slice(start));
  }

  *finish() {
    if (this.fragments.length) {
      const line = this.fragments.join("");
      this.fragments = [];
      yield line.replace(/\r$/, "");
    }
  }
}

/**
 * Synchronous, bounded-read file iterator. Opens on first next(), closes on
 * completion, errors, and early return (including a consumer's break).
 * @param {string} filePath
 */
function* readLines(filePath) {
  const fd = fs.openSync(filePath, "r");
  try {
    const buffer = Buffer.allocUnsafe(64 * 1024);
    const decoder = new StringDecoder("utf8");
    const framer = new LineFramer();
    let count;
    while ((count = fs.readSync(fd, buffer, 0, buffer.length, null)) !== 0) {
      yield* framer.push(decoder.write(buffer.subarray(0, count)));
    }
    yield* framer.push(decoder.end());
    yield* framer.finish();
  } finally {
    fs.closeSync(fd);
  }
}

/**
 * Lazily frame byte or UTF-8 text chunks with backpressure. Byte chunks may split
 * UTF-8 characters. Closing the iterator also closes the source iterator.
 * @param {AsyncIterable<Buffer|string>|Iterable<Buffer|string>} source
 */
async function* streamLines(source) {
  const decoder = new StringDecoder("utf8");
  const framer = new LineFramer();
  for await (const chunk of source) {
    // Flush pending bytes before switching to already-decoded text.
    yield* framer.push(typeof chunk === "string" ? decoder.end() + chunk : decoder.write(chunk));
  }
  yield* framer.push(decoder.end());
  yield* framer.finish();
}

/** @param {JsonlOptions} options */
function validateOptions(options) {
  if (options.maxLineLength !== undefined && (!Number.isSafeInteger(options.maxLineLength) || options.maxLineLength < 1)) {
    throw new RangeError("maxLineLength must be a positive safe integer");
  }
}

/**
 * @param {string} rawLine
 * @param {number} lineNumber
 * @param {JsonlOptions} options
 * @returns {Generator<any>}
 */
function* parseLine(rawLine, lineNumber, options) {
  const line = rawLine.trim();
  if (!line || (options.lineFilter && !options.lineFilter(line, lineNumber))) return;
  if (options.maxLineLength !== undefined && rawLine.length > options.maxLineLength) {
    options.onError?.(new RangeError("JSONL record exceeds maxLineLength"), lineNumber);
    return;
  }
  if (!isJsonCandidate(line)) {
    options.onError?.(new SyntaxError("Invalid JSONL record"), lineNumber);
    return;
  }
  let value;
  try {
    value = JSON.parse(line);
  } catch {
    // Do not forward JSON.parse's message: newer Node versions include input data.
    options.onError?.(new SyntaxError("Invalid JSONL record"), lineNumber);
    return;
  }
  yield value;
}

/**
 * Tolerant, lazy JSONL parser over text or an iterable of complete lines.
 * All JSON values (including null, false, and zero) are preserved. Corrupted
 * records are skipped; onError can count/report them. Filter/callback failures
 * and input I/O errors propagate, rather than being mistaken for corrupt JSON.
 * @param {string|Iterable<string>} source
 * @param {JsonlOptions} [options]
 * @returns {Generator<any>}
 */
function* iterateJsonl(source, options = {}) {
  validateOptions(options);
  let lineNumber = 0;
  for (const line of typeof source === "string" ? iterateLines(source) : source) {
    yield* parseLine(line, ++lineNumber, options);
  }
}

/**
 * Read massive files without loading them into memory. Memory is proportional
 * to the largest line, not the file. Results are not retained by the iterator.
 * @param {string} filePath
 * @param {JsonlOptions} [options]
 * @returns {Generator<any>}
 */
function* readJsonlFile(filePath, options = {}) {
  yield* iterateJsonl(readLines(filePath), options);
}

/**
 * Async streaming JSONL parser for fs.createReadStream(), network responses,
 * and other chunk iterables. No reads occur until iteration starts.
 * @param {AsyncIterable<Buffer|string>|Iterable<Buffer|string>} source
 * @param {JsonlOptions} [options]
 * @returns {AsyncGenerator<any>}
 */
async function* streamJsonl(source, options = {}) {
  validateOptions(options);
  let lineNumber = 0;
  for await (const line of streamLines(source)) {
    yield* parseLine(line, ++lineNumber, options);
  }
}

/**
 * Parse JSONL content into an array of entries.
 * Malformed lines are ignored so callers can safely consume partially written logs.
 *
 * @param {string} content
 * @param {(line: string) => boolean} [lineFilter] Optional predicate to pre-filter lines before JSON parsing.
 * @returns {unknown[]}
 */
function parseJsonlContent(content, lineFilter) {
  if (typeof content !== "string" || content.length === 0) {
    return [];
  }

  return Array.from(iterateJsonl(content, { lineFilter: typeof lineFilter === "function" ? lineFilter : undefined }));
}

module.exports = {
  parseJsonlContent,
  parseJsonlLine,
  iterateLines,
  readLines,
  streamLines,
  iterateJsonl,
  readJsonlFile,
  streamJsonl,
  LineFramer,
};
