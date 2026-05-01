// @ts-check

/**
 * Claude Code CLI Harness with Retry Logic
 *
 * Wraps the Claude Code CLI command with retry logic for failures that occur after the session
 * has been partially executed.  Passes all arguments to the claude subprocess, transparently
 * forwarding stdin/stdout/stderr.
 *
 * Retry policy:
 *   - If the process produced any output (hasOutput) and exits with a non-zero code, the
 *     session is considered partially executed.  The driver retries with --continue so the
 *     Claude Code CLI can continue from where it left off.
 *   - Overloaded API errors (HTTP 529 / "overloaded_error") and rate-limit errors (HTTP 429 /
 *     "rate_limit_error") are well-known transient failure modes and are logged explicitly, but
 *     any partial-execution failure is retried — not just those specific errors.
 *   - If the process produced no output (failed to start / auth error before any work), the
 *     driver does not retry because there is nothing to resume.
 *   - On a `--continue` retry the initial prompt is omitted: Claude Code resumes the session
 *     from its on-disk state rather than re-processing the original instructions.
 *   - Retries use exponential backoff: 5s → 10s → 20s (capped at 60s).
 *   - Maximum 3 retry attempts after the initial run.
 *
 * Prompt handling:
 *   - The harness expects a `--prompt-file <path>` argument in the args list.
 *   - For the initial run it reads the file and appends the content as the last positional arg.
 *   - For `--continue` retries the prompt is omitted (Claude resumes from session state).
 *
 * Usage: node claude_harness.cjs <command> [args...]
 * Example: node claude_harness.cjs claude --print --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt
 */

"use strict";

const { spawn } = require("child_process");
const fs = require("fs");
const path = require("path");

// Maximum number of retry attempts after the initial run
const MAX_RETRIES = 3;
// Initial delay in milliseconds before the first retry
const INITIAL_DELAY_MS = 5000;
// Multiplier applied to delay after each retry
const BACKOFF_MULTIPLIER = 2;
// Maximum delay cap in milliseconds
const MAX_DELAY_MS = 60000;

// AWF API proxy management endpoint for discovering configured LLM providers and available models.
const AWF_API_PROXY_REFLECT_URL = "http://api-proxy:10000/reflect";
// Path inside the agent container where the reflect payload is persisted.
const AWF_REFLECT_OUTPUT_PATH = "/tmp/gh-aw/sandbox/firewall/awf-reflect.json";
// Milliseconds to wait for the /reflect endpoint before giving up.
const AWF_REFLECT_TIMEOUT_MS = 5000;
// Milliseconds to wait for each models_url fallback fetch.
const AWF_MODELS_URL_TIMEOUT_MS = 3000;
// Gemini model name prefix stripped from model IDs in the Gemini models API response.
const GEMINI_MODEL_NAME_PREFIX = "models/";

// Pattern to detect Anthropic API overload errors (HTTP 529).
// Matches "overloaded_error" from the Anthropic error type field, and the
// "Overloaded" human-readable message that Claude Code emits in its stream-json output.
const OVERLOADED_ERROR_PATTERN = /overloaded_error|"overloaded"/i;

// Pattern to detect Anthropic rate-limit errors (HTTP 429).
const RATE_LIMIT_ERROR_PATTERN = /rate_limit_error|429 Too Many Requests/i;

/**
 * Emit a timestamped diagnostic log line to stderr.
 * All driver messages are prefixed with "[claude-harness]" so they are easy to
 * grep out of the combined agent-stdio.log.
 * @param {string} message
 */
function log(message) {
  const ts = new Date().toISOString();
  process.stderr.write(`[claude-harness] ${ts} ${message}\n`);
}

/**
 * Determines if the collected output contains an Anthropic overload error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isOverloadedError(output) {
  return OVERLOADED_ERROR_PATTERN.test(output);
}

/**
 * Determines if the collected output contains an Anthropic rate-limit error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isRateLimitError(output) {
  return RATE_LIMIT_ERROR_PATTERN.test(output);
}

/**
 * Sleep for a specified duration
 * @param {number} ms - Duration in milliseconds
 * @returns {Promise<void>}
 */
function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

/**
 * Format elapsed milliseconds as a human-readable string (e.g. "3m 12s").
 * @param {number} ms
 * @returns {string}
 */
function formatDuration(ms) {
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes > 0) {
    return `${minutes}m ${seconds}s`;
  }
  return `${seconds}s`;
}

/**
 * Extract model IDs from a provider API response body.
 *
 * Handles:
 *   - OpenAI / Anthropic / Copilot format: { data: [{ id: "..." }, ...] }
 *   - Gemini format: { models: [{ name: "models/gemini-1.5-pro" }, ...] }
 *
 * @param {object|null} json - Parsed API response
 * @returns {string[]|null} Sorted array of model IDs, or null if unavailable
 */
function extractModelIds(json) {
  if (!json || typeof json !== "object") return null;

  // OpenAI / Anthropic / Copilot format: { data: [{ id: "..." }, ...] }
  if (Array.isArray(json.data)) {
    const ids = json.data.map(m => m && (m.id || m.name)).filter(Boolean);
    return ids.length > 0 ? ids.sort() : null;
  }

  // Gemini format: { models: [{ name: "models/gemini-1.5-pro", ... }, ...] }
  if (Array.isArray(json.models)) {
    const ids = json.models
      .map(m => {
        if (!m) return null;
        const name = m.name || null;
        if (!name) return null;
        return name.startsWith(GEMINI_MODEL_NAME_PREFIX) ? name.slice(GEMINI_MODEL_NAME_PREFIX.length) : name;
      })
      .filter(Boolean);
    return ids.length > 0 ? ids.sort() : null;
  }

  return null;
}

/**
 * Fetch model IDs from a single models_url endpoint via HTTP GET.
 * @param {string} modelsUrl - URL of the models endpoint on the api-proxy
 * @param {number} timeoutMs - Request timeout in milliseconds
 * @param {(msg: string) => void} logger
 * @returns {Promise<string[]|null>}
 */
async function fetchModelsFromUrl(modelsUrl, timeoutMs, logger) {
  const ac = new AbortController();
  const timer = setTimeout(() => {
    logger(`awf-reflect: models fetch timed out for ${modelsUrl}`);
    ac.abort();
  }, timeoutMs);
  try {
    const res = await fetch(modelsUrl, { signal: ac.signal });
    if (!res.ok) {
      logger(`awf-reflect: models fetch returned ${res.status} for ${modelsUrl}`);
      return null;
    }
    const json = await res.json();
    const models = extractModelIds(json);
    if (models) {
      logger(`awf-reflect: fetched ${models.length} model(s) from ${modelsUrl}`);
    }
    return models;
  } catch (err) {
    const e = /** @type {Error} */ err;
    if (e.name === "AbortError") {
      return null;
    }
    logger(`awf-reflect: models fetch error for ${modelsUrl}: ${e.message}`);
    return null;
  } finally {
    clearTimeout(timer);
  }
}

/**
 * Enrich a reflect response by fetching models for configured endpoints where
 * the api-proxy's startup fetch left models as null.
 * @param {object} reflectData - Parsed /reflect response (mutated in-place)
 * @param {number} timeoutMs - Per-request timeout for models_url fetches
 * @param {(msg: string) => void} logger
 * @returns {Promise<void>}
 */
async function enrichReflectModels(reflectData, timeoutMs, logger) {
  const endpoints = Array.isArray(reflectData.endpoints) ? reflectData.endpoints : [];
  const fetches = endpoints
    .filter(ep => ep && ep.configured && ep.models === null && ep.models_url)
    .map(async ep => {
      const models = await fetchModelsFromUrl(ep.models_url, timeoutMs, logger);
      if (models) {
        ep.models = models;
      }
    });
  if (fetches.length > 0) {
    await Promise.allSettled(fetches);
  }
}

/**
 * Fetch the AWF API proxy /reflect endpoint and persist the response to disk.
 * This is best-effort: failures are logged but do not affect the agent exit code.
 * @param {{
 *   reflectUrl?: string,
 *   outputPath?: string,
 *   timeoutMs?: number,
 *   modelsTimeoutMs?: number,
 *   logger?: (msg: string) => void,
 *   writeFileSync?: (path: string, data: string, options: object) => void,
 * }=} options
 * @returns {Promise<void>}
 */
async function fetchAWFReflect(options) {
  const reflectUrl = (options && options.reflectUrl) || AWF_API_PROXY_REFLECT_URL;
  const outputPath = (options && options.outputPath) || AWF_REFLECT_OUTPUT_PATH;
  const timeoutMs = options && options.timeoutMs != null ? options.timeoutMs : AWF_REFLECT_TIMEOUT_MS;
  const modelsTimeoutMs = options && options.modelsTimeoutMs != null ? options.modelsTimeoutMs : AWF_MODELS_URL_TIMEOUT_MS;
  const logger = (options && options.logger) || log;
  const writeFile = (options && options.writeFileSync) || fs.writeFileSync;

  logger(`awf-reflect: fetching ${reflectUrl} (timeout=${timeoutMs}ms)`);

  const ac = new AbortController();
  const timer = setTimeout(() => {
    logger(`awf-reflect: request timed out after ${timeoutMs}ms`);
    ac.abort();
  }, timeoutMs);

  try {
    const res = await fetch(reflectUrl, { signal: ac.signal });
    if (!res.ok) {
      logger(`awf-reflect: unexpected status ${res.status}, skipping`);
      return;
    }
    const reflectData = await res.json();
    await enrichReflectModels(reflectData, modelsTimeoutMs, logger);
    const enrichedBody = JSON.stringify(reflectData);
    fs.mkdirSync(path.dirname(outputPath), { recursive: true });
    writeFile(outputPath, enrichedBody, { encoding: "utf8" });
    logger(`awf-reflect: saved ${enrichedBody.length}B to ${outputPath}`);
  } catch (err) {
    const e = /** @type {Error} */ err;
    if (e.name === "AbortError") {
      return;
    }
    logger(`awf-reflect: request failed: ${e.message}`);
  } finally {
    clearTimeout(timer);
  }
}

/**
 * Run a command with the given arguments, transparently forwarding stdin/stdout/stderr.
 * Also collects output for error pattern detection.
 *
 * @param {string} command - The executable to run
 * @param {string[]} args - Arguments to pass to the command
 * @param {number} attempt - Current attempt index (0-based), used for logging
 * @returns {Promise<{exitCode: number, output: string, hasOutput: boolean, durationMs: number}>}
 */
function runProcess(command, args, attempt) {
  return new Promise(resolve => {
    const startTime = Date.now();
    log(`attempt ${attempt + 1}: spawning: ${command} ${args.join(" ").substring(0, 200)}`);

    const child = spawn(command, args, {
      stdio: ["inherit", "pipe", "pipe"],
      env: process.env,
    });

    log(`attempt ${attempt + 1}: process started (pid=${child.pid ?? "unknown"})`);

    let collectedOutput = "";
    let hasOutput = false;
    let stdoutBytes = 0;
    let stderrBytes = 0;

    child.stdout.on(
      "data",
      /** @param {Buffer} data */ data => {
        hasOutput = true;
        stdoutBytes += data.length;
        collectedOutput += data.toString();
        process.stdout.write(data);
      }
    );

    child.stderr.on(
      "data",
      /** @param {Buffer} data */ data => {
        hasOutput = true;
        stderrBytes += data.length;
        collectedOutput += data.toString();
        process.stderr.write(data);
      }
    );

    child.on("exit", (code, signal) => {
      log(`attempt ${attempt + 1}: process exit event` + ` exitCode=${code ?? 1}` + (signal ? ` signal=${signal}` : ""));
    });

    // Resolve on 'close', not 'exit', to ensure stdio streams are fully drained.
    child.on("close", (code, signal) => {
      const durationMs = Date.now() - startTime;
      const exitCode = code ?? 1;
      log(`attempt ${attempt + 1}: process closed` + ` exitCode=${exitCode}` + (signal ? ` signal=${signal}` : "") + ` duration=${formatDuration(durationMs)}` + ` stdout=${stdoutBytes}B stderr=${stderrBytes}B hasOutput=${hasOutput}`);
      resolve({ exitCode, output: collectedOutput, hasOutput, durationMs });
    });

    child.on("error", err => {
      const durationMs = Date.now() - startTime;
      const errno = /** @type {NodeJS.ErrnoException} */ err;
      const errCode = errno.code ?? "unknown";
      const errSyscall = errno.syscall ?? "unknown";
      log(`attempt ${attempt + 1}: failed to start process '${command}': ${err.message}` + ` (code=${errCode} syscall=${errSyscall})`);
      resolve({
        exitCode: 1,
        output: collectedOutput,
        hasOutput,
        durationMs,
      });
    });
  });
}

/**
 * Resolve --prompt-file arguments for the initial Claude run.
 * Strips the --prompt-file <path> pair from args and appends the file content
 * as the last positional argument, which is where Claude Code expects the prompt.
 *
 * For --continue retries the prompt should be omitted entirely (Claude resumes
 * from its on-disk session state).  Call this function only for the initial run.
 *
 * @param {string[]} args
 * @returns {string[]} Args with --prompt-file resolved to inline prompt content
 */
function resolveClaudePromptFileArgs(args) {
  /** @type {string[]} */
  const filteredArgs = [];
  /** @type {string|null} */
  let promptContent = null;

  for (let i = 0; i < args.length; i++) {
    if (args[i] !== "--prompt-file") {
      filteredArgs.push(args[i]);
      continue;
    }

    if (i + 1 >= args.length) {
      log("warning: --prompt-file provided without a path; leaving arguments unchanged");
      filteredArgs.push(args[i]);
      continue;
    }

    const promptFile = args[i + 1];
    try {
      const stat = fs.statSync(promptFile);
      log(`resolved --prompt-file: path=${promptFile} size=${stat.size}B`);
      promptContent = fs.readFileSync(promptFile, "utf8");
    } catch (error) {
      const err = /** @type {Error} */ error;
      log(`warning: failed to read --prompt-file ${promptFile}: ${err.message}; leaving arguments unchanged`);
      filteredArgs.push(args[i], promptFile);
    }
    i++; // Skip the prompt-file path argument
  }

  // Append the prompt content as the last positional argument (Claude Code convention).
  if (promptContent !== null) {
    filteredArgs.push(promptContent);
  }

  return filteredArgs;
}

/**
 * Strip --prompt-file and its path argument from args.
 * Used for --continue retries where Claude resumes from on-disk session state
 * and should not be given the original prompt again.
 *
 * @param {string[]} args
 * @returns {string[]} Args with --prompt-file pair removed
 */
function stripPromptFileArgs(args) {
  /** @type {string[]} */
  const filteredArgs = [];
  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--prompt-file" && i + 1 < args.length) {
      i++; // Skip path too
      continue;
    }
    filteredArgs.push(args[i]);
  }
  return filteredArgs;
}

/**
 * Main entry point: run claude with retry logic for transient API failures.
 */
async function main() {
  const [, , command, ...args] = process.argv;

  if (!command) {
    process.stderr.write("claude-harness: Usage: node claude_harness.cjs <command> [args...]\n");
    process.exit(1);
  }

  log(`starting: command=${command} maxRetries=${MAX_RETRIES} initialDelayMs=${INITIAL_DELAY_MS}` + ` backoffMultiplier=${BACKOFF_MULTIPLIER} maxDelayMs=${MAX_DELAY_MS}` + ` nodeVersion=${process.version} platform=${process.platform}`);

  // Resolve the prompt for the initial run (reads --prompt-file content).
  const initialArgs = resolveClaudePromptFileArgs(args);
  // Args without --prompt-file, used as the base for --continue retries.
  const continueBaseArgs = stripPromptFileArgs(args);

  let delay = INITIAL_DELAY_MS;
  let lastExitCode = 1;
  let useContinueOnRetry = false;
  const driverStartTime = Date.now();

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    // For --continue retries: omit the original prompt and add --continue.
    // Claude Code resumes the session from on-disk state; re-sending the original
    // instructions would re-execute the full task from scratch.
    let currentArgs;
    if (attempt > 0 && useContinueOnRetry) {
      currentArgs = [...continueBaseArgs, "--continue"];
    } else {
      currentArgs = initialArgs;
    }

    if (attempt > 0) {
      const retryMode = useContinueOnRetry ? "--continue" : "fresh run";
      log(`retry ${attempt}/${MAX_RETRIES}: sleeping ${delay}ms before next attempt (${retryMode})`);
      await sleep(delay);
      delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
      log(`retry ${attempt}/${MAX_RETRIES}: woke up, next delay cap will be ${Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS)}ms`);
    }

    const result = await runProcess(command, currentArgs, attempt);
    lastExitCode = result.exitCode;

    // Success — stop retrying
    if (result.exitCode === 0) {
      log(`success on attempt ${attempt + 1}: totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
      lastExitCode = 0;
      break;
    }

    const isOverloaded = isOverloadedError(result.output);
    const isRateLimit = isRateLimitError(result.output);
    log(`attempt ${attempt + 1} failed:` + ` exitCode=${result.exitCode}` + ` isOverloadedError=${isOverloaded}` + ` isRateLimitError=${isRateLimit}` + ` hasOutput=${result.hasOutput}` + ` retriesRemaining=${MAX_RETRIES - attempt}`);

    // Retry when the session was partially executed (has output).
    // Use --continue so Claude Code can resume from its saved session state.
    if (attempt < MAX_RETRIES && result.hasOutput) {
      const reason = isOverloaded ? "overloaded_error (transient)" : isRateLimit ? "rate_limit_error (transient)" : "partial execution";
      useContinueOnRetry = true;
      log(`attempt ${attempt + 1}: ${reason} — will retry with --continue (attempt ${attempt + 2}/${MAX_RETRIES + 1})`);
      continue;
    }

    if (attempt >= MAX_RETRIES) {
      log(`all ${MAX_RETRIES} retries exhausted — giving up (exitCode=${lastExitCode})`);
    } else {
      log(`attempt ${attempt + 1}: no output produced — not retrying` + ` (possible causes: binary not found, permission denied, auth failure, or silent startup crash)`);
    }

    break;
  }

  // Fetch AWF API proxy reflection data and persist to disk for post-run step summary.
  await fetchAWFReflect();

  log(`done: exitCode=${lastExitCode} totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
  process.exit(lastExitCode);
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    AWF_API_PROXY_REFLECT_URL,
    AWF_REFLECT_OUTPUT_PATH,
    AWF_REFLECT_TIMEOUT_MS,
    AWF_MODELS_URL_TIMEOUT_MS,
    GEMINI_MODEL_NAME_PREFIX,
    enrichReflectModels,
    extractModelIds,
    fetchAWFReflect,
    fetchModelsFromUrl,
    resolveClaudePromptFileArgs,
    stripPromptFileArgs,
  };
}

if (require.main === module) {
  main().catch(err => {
    log(`unexpected error: ${err.message}`);
    process.exit(1);
  });
}
