// @ts-check

/**
 * OpenAI Codex CLI Harness with Retry Logic
 *
 * Wraps the OpenAI Codex CLI command with retry logic for failures that occur after the
 * session has been partially executed.  Passes all arguments to the codex subprocess,
 * transparently forwarding stdin/stdout/stderr.
 *
 * Retry policy:
 *   - If the process produced any output (hasOutput) and exits with a non-zero code, the
 *     session is considered partially executed.  The driver retries with a fresh run
 *     because Codex does not support a --continue-style session resumption.
 *   - Rate-limit errors (HTTP 429 / "rate_limit_exceeded") and server errors (HTTP 500,
 *     503) are well-known transient failure modes and are logged explicitly, but
 *     any partial-execution failure is retried — not just those specific errors.
 *   - If the process produced no output (failed to start / auth error before any work), the
 *     driver does not retry because there is nothing to resume.
 *   - Retries use exponential backoff: 5s → 10s → 20s (capped at 60s).
 *   - Maximum 3 retry attempts after the initial run.
 *
 * Prompt handling:
 *   - The harness expects a `--prompt-file <path>` argument in the args list.
 *   - It reads the file and appends the content as the last positional argument, which is
 *     where the Codex CLI (`codex exec`) expects the prompt.
 *   - The `--prompt-file` flag is a harness-only argument and is not forwarded to codex.
 *
 * Usage: node codex_harness.cjs <command> [args...]
 * Example: node codex_harness.cjs codex exec --dangerously-bypass-approvals-and-sandbox --skip-git-repo-check --prompt-file /tmp/gh-aw/aw-prompts/prompt.txt
 */

"use strict";

const fs = require("fs");
const { runProcess, formatDuration, sleep } = require("./process_runner.cjs");
const {
  AWF_API_PROXY_REFLECT_URL,
  AWF_REFLECT_OUTPUT_PATH,
  AWF_REFLECT_TIMEOUT_MS,
  AWF_MODELS_URL_TIMEOUT_MS,
  GEMINI_MODEL_NAME_PREFIX,
  enrichReflectModels,
  extractModelIds,
  fetchAWFReflect,
  fetchModelsFromUrl,
} = require("./awf_reflect.cjs");

// Maximum number of retry attempts after the initial run
const MAX_RETRIES = 3;
// Initial delay in milliseconds before the first retry
const INITIAL_DELAY_MS = 5000;
// Multiplier applied to delay after each retry
const BACKOFF_MULTIPLIER = 2;
// Maximum delay cap in milliseconds
const MAX_DELAY_MS = 60000;

// Pattern to detect OpenAI rate-limit errors (HTTP 429).
// Matches "rate_limit_exceeded" from the OpenAI error type field and the "429" status code
// that Codex emits when the API rate limit is hit.
const RATE_LIMIT_ERROR_PATTERN = /rate_limit_exceeded|429 Too Many Requests|RateLimitError/i;
const AUTHENTICATION_FAILED_PATTERN = /Authentication failed(?:\s*\(Request ID:[^)]+\))?/i;

// Pattern to detect a missing API key at startup — Codex emits this before making any API
// calls when neither CODEX_API_KEY nor OPENAI_API_KEY is available in the environment.
// Example: "ERROR: Missing environment variable: `OPENAI_API_KEY`"
const MISSING_API_KEY_PATTERN = /Missing environment variable:\s*`?(?:CODEX_API_KEY|OPENAI_API_KEY)\b`?/i;

// Pattern to detect OpenAI server-side errors (HTTP 500, 503).
// These are transient infrastructure failures that may resolve on retry.
const SERVER_ERROR_PATTERN = /InternalServerError|ServiceUnavailableError|500 Internal Server Error|503 Service Unavailable/i;
const PERMISSION_DENIED_PATTERN = /\b(?:permission denied|permissions denied|EACCES|EPERM)\b/gi;
const NUMEROUS_PERMISSION_DENIED_THRESHOLD = 3;

/**
 * Emit a timestamped diagnostic log line to stderr.
 * All driver messages are prefixed with "[codex-harness]" so they are easy to
 * grep out of the combined agent-stdio.log.
 * @param {string} message
 */
function log(message) {
  const ts = new Date().toISOString();
  process.stderr.write(`[codex-harness] ${ts} ${message}\n`);
}

/**
 * Determines if the collected output contains an OpenAI rate-limit error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isRateLimitError(output) {
  return RATE_LIMIT_ERROR_PATTERN.test(output);
}

/**
 * Determines if the collected output contains an authentication failed error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isAuthenticationFailedError(output) {
  return AUTHENTICATION_FAILED_PATTERN.test(output);
}

/**
 * Determines if the collected output indicates a missing API key at startup.
 * Codex exits before producing any agent output in this case, so retrying is futile.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isMissingApiKeyError(output) {
  return MISSING_API_KEY_PATTERN.test(output);
}

/**
 * Determines if the collected output contains an OpenAI server error.
 * @param {string} output - Collected stdout+stderr from the process
 * @returns {boolean}
 */
function isServerError(output) {
  return SERVER_ERROR_PATTERN.test(output);
}

/**
 * Extract the commands that were denied from process output.
 * Scans for lines using the CLI pipe marker (│) that appear
 * within three lines before each "permission denied" occurrence.
 * Returns a deduplicated array of command strings (may be empty if
 * the output format does not contain extractable commands).
 * @param {string} output
 * @returns {string[]}
 */
function extractDeniedCommands(output) {
  if (!output) return [];
  const lines = output.split("\n");
  const deniedCommands = new Set();
  for (let i = 0; i < lines.length; i++) {
    if (/\bpermission denied\b/i.test(lines[i])) {
      // Look back up to 3 lines for a command displayed with the
      // CLI box-drawing pipe marker (│ U+2502) or plain pipe (|).
      for (let j = i - 1; j >= Math.max(0, i - 3); j--) {
        const cmdMatch = lines[j].match(/^\s*[\u2502|]\s+(.+)\s*$/);
        if (cmdMatch && cmdMatch[1].trim()) {
          deniedCommands.add(cmdMatch[1].trim());
          break;
        }
      }
    }
  }
  return [...deniedCommands];
}

/**
 * Count permission-denied indicators in process output.
 * @param {string} output
 * @returns {number}
 */
function countPermissionDeniedIssues(output) {
  if (!output) return 0;
  const matches = output.match(PERMISSION_DENIED_PATTERN);
  return matches ? matches.length : 0;
}

/**
 * Detect whether output contains numerous permission-denied issues.
 * @param {string} output
 * @returns {boolean}
 */
function hasNumerousPermissionDeniedIssues(output) {
  return countPermissionDeniedIssues(output) >= NUMEROUS_PERMISSION_DENIED_THRESHOLD;
}

/**
 * Build a structured missing_tool payload for repeated permission-denied failures.
 * @param {string[]} [deniedCommands] - Commands that were denied (may be empty)
 * @returns {string}
 */
function buildMissingToolPermissionIssuePayload(deniedCommands) {
  return JSON.stringify({
    type: "missing_tool",
    tool: "tool/permission",
    reason: "missing tool/permission issue: numerous permission denied errors detected",
    alternatives: "Verify token scopes, repository permissions, and MCP/tool access configuration.",
    denied_commands: deniedCommands && deniedCommands.length > 0 ? deniedCommands : [],
  });
}

/**
 * Emit a structured missing_tool signal for repeated permission-denied failures.
 * @param {{
 *   safeOutputsPath?: string,
 *   appendFileSync?: (path: import('node:fs').PathOrFileDescriptor, data: string, options?: import('node:fs').WriteFileOptions) => void,
 *   logger?: (message: string) => void,
 *   deniedCommands?: string[]
 * }=} options
 */
function emitMissingToolPermissionIssue(options) {
  const safeOutputsPath = options && typeof options.safeOutputsPath === "string" ? options.safeOutputsPath : process.env.GH_AW_SAFE_OUTPUTS || "";
  const appendFileSync = options && options.appendFileSync ? options.appendFileSync : fs.appendFileSync;
  const logger = options && options.logger ? options.logger : log;
  const deniedCommands = options && options.deniedCommands ? options.deniedCommands : [];

  if (!safeOutputsPath) {
    logger("missing_tool skipped: GH_AW_SAFE_OUTPUTS is not set");
    return;
  }
  try {
    appendFileSync(safeOutputsPath, buildMissingToolPermissionIssuePayload(deniedCommands) + "\n", { encoding: "utf8" });
    logger(`missing_tool emitted for permission issues: ${safeOutputsPath}`);
  } catch (error) {
    const err = /** @type {Error} */ error;
    logger(`missing_tool emission failed: ${err.message}`);
  }
}

/**
 * Resolve --prompt-file arguments for the Codex run.
 * Strips the --prompt-file <path> pair from args and appends the file content
 * as the last positional argument, which is where `codex exec` expects the prompt.
 *
 * @param {string[]} args
 * @returns {string[]} Args with --prompt-file resolved to inline prompt content
 */
function resolveCodexPromptFileArgs(args) {
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
      // An unreadable prompt file means no task instructions can be delivered to Codex.
      // Propagate as a fatal error rather than forwarding the harness-only flag to the
      // codex subprocess (which would fail with an "unknown option" error).
      throw new Error(`--prompt-file '${promptFile}' is not readable: ${err.message}`);
    }
    i++; // Skip the prompt-file path argument
  }

  // Append the prompt content as the last positional argument (codex exec convention).
  if (promptContent !== null) {
    filteredArgs.push(promptContent);
  }

  return filteredArgs;
}

/**
 * Inject `--json` after `exec` in the args list so that Codex streams structured
 * JSON Lines (JSONL) to stdout.  This enables machine-readable output for CI
 * pipelines without changing how stderr progress output works.
 *
 * No-op when the subcommand is not `exec` or when `--json` is already present.
 *
 * @param {string[]} args
 * @returns {string[]}
 */
function injectJsonFlag(args) {
  if (args.length === 0 || args[0] !== "exec") return args;
  if (args.includes("--json")) return args;
  return ["exec", "--json", ...args.slice(1)];
}

/**
 * Build child process environment for Codex execution.
 * Preserve API keys captured at harness startup, even if the parent environment
 * is sanitized later in the run.
 *
 * @param {NodeJS.ProcessEnv} baseEnv
 * @param {string|undefined} codexApiKey
 * @param {string|undefined} openaiApiKey
 * @returns {NodeJS.ProcessEnv}
 */
function buildCodexChildEnv(baseEnv, codexApiKey, openaiApiKey) {
  const childEnv = { ...baseEnv };
  if (codexApiKey) {
    childEnv.CODEX_API_KEY = codexApiKey;
  }
  if (openaiApiKey) {
    childEnv.OPENAI_API_KEY = openaiApiKey;
  }
  return childEnv;
}

/**
 * Extract numeric port from a URL string.
 * @param {string} urlString
 * @returns {number|null}
 */
function extractPortFromURL(urlString) {
  if (!urlString || typeof urlString !== "string") return null;
  try {
    const parsed = new URL(urlString);
    if (!parsed.port) return null;
    return Number(parsed.port);
  } catch {
    return null;
  }
}

/**
 * Read Codex openai-proxy base_url from config.toml content.
 * @param {string} tomlContent
 * @returns {string|null}
 */
function extractOpenAIProxyBaseURLFromToml(tomlContent) {
  if (!tomlContent || typeof tomlContent !== "string") return null;
  const providerSection = tomlContent.match(/\[model_providers\.openai-proxy\]([\s\S]*?)(?:\n\[|$)/);
  if (!providerSection) return null;
  const baseURLMatch = providerSection[1].match(/(?:^|\n)\s*base_url\s*=\s*"([^"]+)"/);
  return baseURLMatch && baseURLMatch[1] ? baseURLMatch[1].trim() : null;
}

/**
 * Determine configured OpenAI endpoint port from AWF /reflect payload.
 * @param {any} reflectData
 * @returns {number|null}
 */
function getConfiguredOpenAIPortFromReflect(reflectData) {
  const endpoints = reflectData && Array.isArray(reflectData.endpoints) ? reflectData.endpoints : [];
  const openAIEndpoint = endpoints.find(ep => {
    if (!ep || ep.configured !== true || typeof ep.provider !== "string") return false;
    return ep.provider.toLowerCase() === "openai";
  });
  if (!openAIEndpoint || openAIEndpoint.port == null) return null;
  const parsedPort = Number(openAIEndpoint.port);
  return Number.isNaN(parsedPort) ? null : parsedPort;
}

/**
 * Validate that Codex openai-proxy base_url matches the configured OpenAI endpoint from /reflect.
 * @param {{
 *   codexConfigPath: string,
 *   reflectPath: string,
 *   readFileSync?: (path: import("node:fs").PathOrFileDescriptor, options?: import("node:fs").ObjectEncodingOptions & { flag?: string } | BufferEncoding | null) => string
 * }} options
 * @returns {{ ok: boolean, reason?: string }}
 */
function validateCodexOpenAIBaseURLFromReflect(options) {
  const readFileSync = (options && options.readFileSync) || fs.readFileSync;
  const codexConfigPath = options && options.codexConfigPath;
  const reflectPath = options && options.reflectPath;
  if (!codexConfigPath || !reflectPath) return { ok: true };

  let tomlContent;
  let reflectContent;
  try {
    tomlContent = readFileSync(codexConfigPath, "utf8");
    reflectContent = readFileSync(reflectPath, "utf8");
  } catch {
    return { ok: true };
  }

  const baseURL = extractOpenAIProxyBaseURLFromToml(tomlContent);
  if (!baseURL) return { ok: true };
  const baseURLPort = extractPortFromURL(baseURL);
  if (baseURLPort == null) return { ok: true };

  let reflectData;
  try {
    reflectData = JSON.parse(reflectContent);
  } catch {
    return { ok: true };
  }
  const openAIPort = getConfiguredOpenAIPortFromReflect(reflectData);
  if (openAIPort == null) return { ok: true };
  if (openAIPort !== baseURLPort) {
    return {
      ok: false,
      reason: `Codex openai-proxy base_url port mismatch: config.toml uses ${baseURLPort}, but /reflect reports OpenAI on port ${openAIPort}`,
    };
  }
  return { ok: true };
}

/**
 * Main entry point: run codex with retry logic for transient API failures.
 * Codex does not support --continue session resumption, so all retries are fresh runs.
 */
async function main() {
  const [, , command, ...args] = process.argv;

  if (!command) {
    process.stderr.write("codex-harness: Usage: node codex_harness.cjs <command> [args...]\n");
    process.exit(1);
  }

  log(`starting: command=${command} maxRetries=${MAX_RETRIES} initialDelayMs=${INITIAL_DELAY_MS}` + ` backoffMultiplier=${BACKOFF_MULTIPLIER} maxDelayMs=${MAX_DELAY_MS}` + ` nodeVersion=${process.version} platform=${process.platform}`);

  // Diagnose API key presence so CI failures can be triaged without exposing secret values.
  const codexApiKey = process.env.CODEX_API_KEY;
  const openaiApiKey = process.env.OPENAI_API_KEY;
  const codexChildEnv = buildCodexChildEnv(process.env, codexApiKey, openaiApiKey);
  log(`secrets: CODEX_API_KEY=${codexApiKey ? `set (length=${codexApiKey.length})` : "not set"}` + ` OPENAI_API_KEY=${openaiApiKey ? `set (length=${openaiApiKey.length})` : "not set"}`);

  // Pre-flight: require at least one API key before spawning codex.
  // Without a key, codex exits immediately with "Missing environment variable" and every
  // retry attempt fails the same way. Failing here avoids burning the retry budget and
  // surfaces a clear, actionable message in CI logs.
  if (!codexApiKey && !openaiApiKey) {
    log("fatal: no API key available - set CODEX_API_KEY or OPENAI_API_KEY and retry");
    process.exit(1);
  }

  // Resolve the prompt for the initial run (reads --prompt-file content).
  // A missing or unreadable prompt file is treated as a fatal startup error.
  let resolvedArgs;
  try {
    resolvedArgs = resolveCodexPromptFileArgs(args);
  } catch (err) {
    const e = /** @type {Error} */ err;
    log(`fatal: ${e.message}`);
    process.exit(1);
  }

  // Safe arg list for logging: when --prompt-file was present, the last element of
  // resolvedArgs is the resolved prompt content. Replace it with a placeholder so that
  // task instructions are never written to stderr or captured in agent logs.
  const hadPromptFile = args.includes("--prompt-file");
  const safeArgs = hadPromptFile && resolvedArgs.length > 0 ? [...resolvedArgs.slice(0, -1), "<prompt omitted>"] : resolvedArgs;

  // Inject --json after `exec` to stream structured JSONL events to stdout, making
  // Codex output machine-readable in CI without affecting the stderr progress stream.
  resolvedArgs = injectJsonFlag(resolvedArgs);

  // Fetch AWF API proxy reflection data before running the agent to capture initial proxy state.
  // This is best-effort: failures are logged but do not affect the agent run.
  await fetchAWFReflect({ logger: log });
  const codexHome = process.env.CODEX_HOME || "";
  if (codexHome) {
    const validation = validateCodexOpenAIBaseURLFromReflect({
      codexConfigPath: `${codexHome}/config.toml`,
      reflectPath: AWF_REFLECT_OUTPUT_PATH,
    });
    if (!validation.ok) {
      log(`fatal: ${validation.reason}`);
      process.exit(1);
    }
  }

  let delay = INITIAL_DELAY_MS;
  let lastExitCode = 1;
  const driverStartTime = Date.now();

  for (let attempt = 0; attempt <= MAX_RETRIES; attempt++) {
    // Codex does not support --continue: every retry is a fresh run from scratch.
    // Context from the interrupted session is not recoverable, but transient API
    // failures (rate limits, server errors) may resolve on the next attempt.

    if (attempt > 0) {
      log(`retry ${attempt}/${MAX_RETRIES}: sleeping ${delay}ms before next attempt (fresh run)`);
      await sleep(delay);
      delay = Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS);
      log(`retry ${attempt}/${MAX_RETRIES}: woke up, next delay cap will be ${Math.min(delay * BACKOFF_MULTIPLIER, MAX_DELAY_MS)}ms`);
    }

    const result = await runProcess({ command, args: resolvedArgs, attempt, log, logArgs: safeArgs, env: codexChildEnv });
    lastExitCode = result.exitCode;

    // Success — stop retrying
    if (result.exitCode === 0) {
      log(`success on attempt ${attempt + 1}: totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
      lastExitCode = 0;
      break;
    }

    const isRateLimit = isRateLimitError(result.output);
    const isAuthenticationFailed = isAuthenticationFailedError(result.output);
    const isMissingApiKey = isMissingApiKeyError(result.output);
    const isServer = isServerError(result.output);
    const permissionDeniedCount = countPermissionDeniedIssues(result.output);
    const hasNumerousPermissionDenied = hasNumerousPermissionDeniedIssues(result.output);
    log(
      `attempt ${attempt + 1} failed:` +
        ` exitCode=${result.exitCode}` +
        ` isRateLimitError=${isRateLimit}` +
        ` isAuthenticationFailedError=${isAuthenticationFailed}` +
        ` isMissingApiKeyError=${isMissingApiKey}` +
        ` isServerError=${isServer}` +
        ` permissionDeniedCount=${permissionDeniedCount}` +
        ` hasNumerousPermissionDenied=${hasNumerousPermissionDenied}` +
        ` hasOutput=${result.hasOutput}` +
        ` retriesRemaining=${MAX_RETRIES - attempt}`
    );

    if (attempt === 0 && isAuthenticationFailed) {
      log(`attempt ${attempt + 1}: authentication failed — not retrying (first-attempt auth failure is non-retryable)`);
      break;
    }

    if (isMissingApiKey) {
      log(`attempt ${attempt + 1}: missing API key — not retrying (configure CODEX_API_KEY or OPENAI_API_KEY)`);
      break;
    }

    if (hasNumerousPermissionDenied) {
      const deniedCommands = extractDeniedCommands(result.output);
      emitMissingToolPermissionIssue({ deniedCommands });
      log(`attempt ${attempt + 1}: detected numerous permission-denied issues — not retrying (classified as missing tool/permission issue)`);
      break;
    }

    // Retry when the session was partially executed (has output) or on well-known
    // transient errors (rate limit, server error) even without output.
    const isTransient = isRateLimit || isServer;
    if (attempt < MAX_RETRIES && (result.hasOutput || isTransient)) {
      const reason = isRateLimit ? "rate_limit_exceeded (transient)" : isServer ? "server_error (transient)" : "partial execution";
      log(`attempt ${attempt + 1}: ${reason} — will retry as fresh run (attempt ${attempt + 2}/${MAX_RETRIES + 1})`);
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
  await fetchAWFReflect({ logger: log });

  log(`done: exitCode=${lastExitCode} totalDuration=${formatDuration(Date.now() - driverStartTime)}`);
  process.exit(lastExitCode);
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    resolveCodexPromptFileArgs,
    injectJsonFlag,
    isRateLimitError,
    isAuthenticationFailedError,
    isMissingApiKeyError,
    isServerError,
    countPermissionDeniedIssues,
    hasNumerousPermissionDeniedIssues,
    extractDeniedCommands,
    buildMissingToolPermissionIssuePayload,
    emitMissingToolPermissionIssue,
    buildCodexChildEnv,
    extractPortFromURL,
    extractOpenAIProxyBaseURLFromToml,
    getConfiguredOpenAIPortFromReflect,
    validateCodexOpenAIBaseURLFromReflect,
  };
}

if (require.main === module) {
  main().catch(err => {
    log(`unexpected error: ${err.message}`);
    process.exit(1);
  });
}
