// @ts-check

const fs = require("node:fs");
const { StringDecoder } = require("node:string_decoder");

/**
 * @typedef {object} JsonlOptions
 * @property {(line: string, lineNumber: number) => boolean} [lineFilter] Runs before JSON.parse.
 * @property {(error: Error, lineNumber: number) => void} [onError] Reports rejected nonblank lines without exposing their contents.
 * @property {number} [maxLineLength] Optional limit in UTF-16 code units; unlimited by default.
 */

/**
 * @typedef {JsonlOptions & {strict?: boolean}} JsonlValueOptions
 * @typedef {{type: "json", value: any, rawLine: string, lineNumber: number} |
 * {type: "invalid", error: Error, reason: "blank"|"syntax"|"too-long", rawLine: string, lineNumber: number}} JsonlEvent
 */

/** @param {string} line */
function isJsonCandidate(line) {
  if (line[0] === "{") return /^\{\s*["}]/.test(line);
  if (line[0] === "[") return /^\[\s*(?:[{\["\d\]-]|(?:true|false|null)(?:[\s,\]]|$))/.test(line);
  return /^(?:["\d-]|(?:true|false|null)$)/.test(line);
}

/**
 * Lazily split existing text without allocating an array of every line.
 * LF and CRLF are supported; a final unterminated line is included.
 * @param {string} content
 * @param {boolean} [includeNewline] Preserve original line endings in each yielded line.
 */
function* iterateLines(content, includeNewline = false) {
  let start = 0;
  while (start < content.length) {
    const end = content.indexOf("\n", start);
    if (end === -1) {
      yield includeNewline ? content.slice(start) : content.slice(start).replace(/\r$/, "");
      return;
    }
    yield includeNewline ? content.slice(start, end + 1) : content.slice(start, end).replace(/\r$/, "");
    start = end + 1;
  }
  if (!includeNewline) yield "";
}

/**
 * Chunk framing retains only the current line, joining fragments once rather than
 * repeatedly copying a growing multi-megabyte string.
 */
class LineFramer {
  /** @param {boolean} [includeNewline] */
  constructor(includeNewline = false) {
    this.includeNewline = includeNewline;
    /** @type {string[]} */
    this.fragments = [];
  }

  /** @param {string} chunk */
  *push(chunk) {
    let start = 0;
    let end;
    while ((end = chunk.indexOf("\n", start)) !== -1) {
      const part = chunk.slice(start, this.includeNewline ? end + 1 : end);
      const line = this.fragments.length ? this.fragments.join("") + part : part;
      this.fragments = [];
      start = end + 1;
      yield this.includeNewline ? line : line.replace(/\r$/, "");
    }
    if (start < chunk.length) this.fragments.push(chunk.slice(start));
  }

  *finish() {
    if (this.fragments.length) {
      const line = this.fragments.join("");
      this.fragments = [];
      yield this.includeNewline ? line : line.replace(/\r$/, "");
    }
  }
}

/**
 * Synchronous, bounded-read file iterator. Opens on first next(), closes on
 * completion, errors, and early return (including a consumer's break).
 * @param {string} filePath
 * @param {boolean} [includeNewline]
 */
function* readLines(filePath, includeNewline = false) {
  const fd = fs.openSync(filePath, "r");
  try {
    const buffer = Buffer.allocUnsafe(64 * 1024);
    const decoder = new StringDecoder("utf8");
    const framer = new LineFramer(includeNewline);
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
 * @param {boolean} [includeNewline]
 */
async function* streamLines(source, includeNewline = false) {
  const decoder = new StringDecoder("utf8");
  const framer = new LineFramer(includeNewline);
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
 * @param {"blank"|"syntax"|"too-long"} reason
 * @param {JsonlOptions} options
 * @returns {JsonlEvent}
 */
function invalidEvent(rawLine, lineNumber, reason, options) {
  /** @type {Error|undefined} */
  let error;
  /** @type {JsonlEvent} */
  const event = {
    type: "invalid",
    rawLine,
    lineNumber,
    reason,
    // Plain-text logs must not pay for an error stack unless it is consumed.
    get error() {
      return (error ??= reason === "too-long" ? new RangeError("JSONL record exceeds maxLineLength") : new SyntaxError(reason === "blank" ? "Blank JSONL record" : "Invalid JSONL record"));
    },
  };
  if (reason !== "blank" && options.onError) options.onError(event.error, lineNumber);
  return event;
}

/**
 * @param {string} rawLine
 * @param {number} lineNumber
 * @param {JsonlOptions} options
 * @returns {JsonlEvent|undefined}
 */
function parseEvent(rawLine, lineNumber, options) {
  const line = rawLine.trim();
  if (!line) {
    return invalidEvent(rawLine, lineNumber, "blank", options);
  }
  if (options.lineFilter && !options.lineFilter(line, lineNumber)) return;
  const length = rawLine.endsWith("\r\n") ? rawLine.length - 2 : rawLine.endsWith("\n") ? rawLine.length - 1 : rawLine.length;
  const tooLong = options.maxLineLength !== undefined && length > options.maxLineLength;
  if (!tooLong && isJsonCandidate(line)) {
    try {
      const value = JSON.parse(line);
      return { type: "json", value, rawLine, lineNumber };
    } catch {
      // Report a sanitized error below, not JSON.parse's input-bearing message.
    }
  }
  return invalidEvent(rawLine, lineNumber, tooLong ? "too-long" : "syntax", options);
}

/**
 * Emit valid and invalid record events without dropping raw input. rawLine
 * includes its original LF/CRLF terminator when present. Blank lines are invalid
 * events with reason "blank", but do not invoke onError. Input text is untrusted:
 * redact rawLine before logging it. break/return stops parsing immediately.
 * @param {string|Iterable<string>} source Text or complete raw lines.
 * @param {JsonlOptions} [options]
 * @returns {Generator<JsonlEvent>}
 */
function* iterateJsonlEvents(source, options = {}) {
  validateOptions(options);
  let lineNumber = 0;
  for (const rawLine of typeof source === "string" ? iterateLines(source, true) : source) {
    const event = parseEvent(rawLine, ++lineNumber, options);
    if (event) yield event;
  }
}

/**
 * Lazy file-backed record events, including malformed and blank lines.
 * The file closes on completion, errors, break, and return.
 * @param {string} filePath
 * @param {JsonlOptions} [options]
 * @returns {Generator<JsonlEvent>}
 */
function* readJsonlFileEvents(filePath, options = {}) {
  yield* iterateJsonlEvents(readLines(filePath, true), options);
}

/**
 * Async, backpressured record events over UTF-8 byte or text chunks.
 * @param {AsyncIterable<Buffer|string>|Iterable<Buffer|string>} source
 * @param {JsonlOptions} [options]
 * @returns {AsyncGenerator<JsonlEvent>}
 */
async function* streamJsonlEvents(source, options = {}) {
  validateOptions(options);
  let lineNumber = 0;
  for await (const rawLine of streamLines(source, true)) {
    const event = parseEvent(rawLine, ++lineNumber, options);
    if (event) yield event;
  }
}

/**
 * Tolerant, lazy JSONL parser over text or an iterable of complete lines.
 * All JSON values (including null, false, and zero) are preserved. Corrupted
 * records are skipped; onError can count/report them. Filter/callback failures
 * and input I/O errors propagate, rather than being mistaken for corrupt JSON.
 * @param {string|Iterable<string>} source
 * @param {JsonlValueOptions} [options] strict throws on any invalid record, including blanks.
 * @returns {Generator<any>}
 */
function* iterateJsonl(source, options = {}) {
  for (const event of iterateJsonlEvents(source, options)) {
    if (event.type === "json") yield event.value;
    else if (options.strict) throw event.error;
  }
}

/**
 * Read massive files without loading them into memory. Memory is proportional
 * to the largest line, not the file. Results are not retained by the iterator.
 * @param {string} filePath
 * @param {JsonlValueOptions} [options]
 * @returns {Generator<any>}
 */
function* readJsonlFile(filePath, options = {}) {
  yield* iterateJsonl(readLines(filePath, true), options);
}

/**
 * Async streaming JSONL parser for fs.createReadStream(), network responses,
 * and other chunk iterables. No reads occur until iteration starts.
 * @param {AsyncIterable<Buffer|string>|Iterable<Buffer|string>} source
 * @param {JsonlValueOptions} [options]
 * @returns {AsyncGenerator<any>}
 */
async function* streamJsonl(source, options = {}) {
  for await (const event of streamJsonlEvents(source, options)) {
    if (event.type === "json") yield event.value;
    else if (options.strict) throw event.error;
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
  iterateLines,
  readLines,
  streamLines,
  iterateJsonl,
  readJsonlFile,
  streamJsonl,
  iterateJsonlEvents,
  readJsonlFileEvents,
  streamJsonlEvents,
  LineFramer,
};
