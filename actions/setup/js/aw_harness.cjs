// @ts-check

/**
 * AW Harness — Single-Session Agentic Workflow Execution Engine
 *
 * Entry point for `engine: aw` workflows. Uses the Pi SDK
 * (@earendil-works/pi-coding-agent) to run a single AgentSession with the
 * compiled prompt, budget management, steering, and observability extensions.
 *
 * The gh-aw compiler pre-processes the workflow Markdown (frontmatter, prompt
 * body, imports) and provides the harness with two pre-built input files:
 *   config.json  — parsed harness configuration (model, budget, extensions …)
 *   prompt.txt   — extracted prompt body
 *
 * Invocation contract (section 5 of specs/aw-harness.md):
 *   node aw_harness.cjs --config <config-path> --prompt <prompt-path>
 *
 * Exit codes:
 *   0 — prompt completed successfully
 *   1 — session failed (non-recoverable error, budget exceeded, SDK missing)
 *   2 — invocation error (missing / unreadable config or prompt file)
 *
 * Standard streams:
 *   stderr — JSONL event stream, diagnostic messages
 *   stdout — reserved (not used by this harness)
 *   $GITHUB_STEP_SUMMARY — Markdown execution summary (when env var is set)
 *
 * The Pi SDK packages (@earendil-works/pi-coding-agent, @earendil-works/pi-ai)
 * are installed at runtime by AWF before this harness is invoked; they are NOT
 * bundled in the repository.  A missing SDK results in exit code 1.
 */

"use strict";

const fs = require("fs");
const crypto = require("crypto");

// ---------------------------------------------------------------------------
// Logging helpers (all diagnostic output goes to stderr)
// ---------------------------------------------------------------------------

/** @param {string} msg */
function log(msg) {
  process.stderr.write(`[aw-harness] ${msg}\n`);
}

/**
 * Emit a JSONL event to stderr.
 * @param {Record<string, unknown>} obj
 */
function emitJsonl(obj) {
  process.stderr.write(JSON.stringify(obj) + "\n");
}

// ---------------------------------------------------------------------------
// Step summary
// ---------------------------------------------------------------------------

/**
 * Append content to the GitHub Actions step summary file (if configured).
 * Errors writing the summary are logged but do not abort the harness.
 * @param {string} content
 */
function appendStepSummary(content) {
  const summaryFile = process.env.GITHUB_STEP_SUMMARY;
  if (!summaryFile) return;
  try {
    fs.appendFileSync(summaryFile, content, "utf8");
  } catch (err) {
    log(`warning: failed to write step summary to '${summaryFile}': ${err instanceof Error ? err.message : String(err)}`);
  }
}

// ---------------------------------------------------------------------------
// Argument parsing (section 5.1)
// ---------------------------------------------------------------------------

/**
 * Parse the --config and --prompt CLI arguments.
 * Returns null values when an argument is absent.
 *
 * @param {string[]} argv  process.argv
 * @returns {{ configPath: string|null, promptPath: string|null }}
 */
function parseArgs(argv) {
  const args = argv.slice(2);
  let configPath = null;
  let promptPath = null;

  for (let i = 0; i < args.length; i++) {
    if (args[i] === "--config" && i + 1 < args.length) {
      configPath = args[++i];
    } else if (args[i] === "--prompt" && i + 1 < args.length) {
      promptPath = args[++i];
    }
  }

  return { configPath, promptPath };
}

// ---------------------------------------------------------------------------
// Input loading (sections 5.1 / 6.3)
// ---------------------------------------------------------------------------

/**
 * Load config.json from disk and parse it.
 * Returns the parsed config object.
 * Throws an error with a descriptive message when the file cannot be read or parsed.
 *
 * @param {string} configPath
 * @returns {Record<string, unknown>}
 */
function loadConfig(configPath) {
  let raw;
  try {
    raw = fs.readFileSync(configPath, "utf8");
  } catch (err) {
    throw new Error(`failed to read config file '${configPath}': ${err instanceof Error ? err.message : String(err)}`);
  }
  try {
    return JSON.parse(raw);
  } catch (err) {
    throw new Error(`failed to parse config file '${configPath}': ${err instanceof Error ? err.message : String(err)}`);
  }
}

/**
 * Load prompt.txt from disk.
 * Returns the prompt string.
 * Throws an error with a descriptive message when the file cannot be read.
 *
 * @param {string} promptPath
 * @returns {string}
 */
function loadPrompt(promptPath) {
  try {
    return fs.readFileSync(promptPath, "utf8");
  } catch (err) {
    throw new Error(`failed to read prompt file '${promptPath}': ${err instanceof Error ? err.message : String(err)}`);
  }
}

// ---------------------------------------------------------------------------
// Provider setup (section 8.1)
// ---------------------------------------------------------------------------

/**
 * Build provider registrations from AWF-injected environment variables.
 * Returns an array of provider config objects recognised by pi.registerProvider().
 *
 * Supported provider environment variable pairs:
 *   ANTHROPIC_API_KEY  / ANTHROPIC_BASE_URL  → "anthropic" (api: "anthropic")
 *   OPENAI_API_KEY     / OPENAI_BASE_URL     → "openai"    (api: "openai-completions")
 *   CODEX_API_KEY      / CODEX_BASE_URL      → "openai"    (alias)
 *   GITHUB_TOKEN       / COPILOT_BASE_URL    → "copilot"   (api: "openai-completions")
 *   GEMINI_API_KEY     / GEMINI_BASE_URL     → "google"    (api: "google-generative-ai")
 *
 * @returns {Array<{name: string, apiKey: string, api: string, baseUrl?: string}>}
 */
function buildProviderConfigs() {
  /** @type {Array<{name: string, apiKey: string, api: string, baseUrl?: string}>} */
  const providers = [];

  const env = process.env;

  if (env.ANTHROPIC_API_KEY) {
    /** @type {{name: string, apiKey: string, api: string, baseUrl?: string}} */
    const p = { name: "anthropic", apiKey: env.ANTHROPIC_API_KEY, api: "anthropic" };
    if (env.ANTHROPIC_BASE_URL) p.baseUrl = env.ANTHROPIC_BASE_URL;
    providers.push(p);
  }

  const openaiKey = env.OPENAI_API_KEY || env.CODEX_API_KEY;
  if (openaiKey) {
    /** @type {{name: string, apiKey: string, api: string, baseUrl?: string}} */
    const p = { name: "openai", apiKey: openaiKey, api: "openai-completions" };
    const baseUrl = env.OPENAI_BASE_URL || env.CODEX_BASE_URL;
    if (baseUrl) p.baseUrl = baseUrl;
    providers.push(p);
  }

  const githubToken = env.COPILOT_GITHUB_TOKEN || env.GITHUB_TOKEN;
  if (githubToken) {
    /** @type {{name: string, apiKey: string, api: string, baseUrl?: string}} */
    const p = { name: "copilot", apiKey: githubToken, api: "openai-completions" };
    if (env.COPILOT_BASE_URL) p.baseUrl = env.COPILOT_BASE_URL;
    providers.push(p);
  }

  if (env.GEMINI_API_KEY) {
    /** @type {{name: string, apiKey: string, api: string, baseUrl?: string}} */
    const p = { name: "google", apiKey: env.GEMINI_API_KEY, api: "google-generative-ai" };
    if (env.GEMINI_BASE_URL) p.baseUrl = env.GEMINI_BASE_URL;
    providers.push(p);
  }

  return providers;
}

// ---------------------------------------------------------------------------
// Built-in provider setup extension (section 8.1)
// ---------------------------------------------------------------------------

/**
 * Returns the provider setup Pi extension function.
 * Registers all available LLM providers before the session begins.
 *
 * @returns {(pi: unknown) => void}
 */
function makeProviderSetupExtension() {
  return function providerSetupExtension(/** @type {any} */ pi) {
    const providers = buildProviderConfigs();
    if (providers.length === 0) {
      throw new Error(
        "no LLM provider credentials found in the environment. " +
        "At least one of the following must be set: ANTHROPIC_API_KEY, OPENAI_API_KEY, " +
        "CODEX_API_KEY, COPILOT_GITHUB_TOKEN, GITHUB_TOKEN, GEMINI_API_KEY.",
      );
    }
    for (const { name, apiKey, api, baseUrl } of providers) {
      /** @type {Record<string, unknown>} */
      const opts = { apiKey, api };
      if (baseUrl) opts.baseUrl = baseUrl;
      log(`registering provider: ${name} (api: ${api}${baseUrl ? `, baseUrl: ${baseUrl}` : ""})`);
      pi.registerProvider(name, opts);
    }
  };
}

// ---------------------------------------------------------------------------
// Built-in observability extension (section 8.5)
// ---------------------------------------------------------------------------

/**
 * Returns the observability Pi extension function.
 * Emits JSONL events to stderr and appends a Markdown summary to
 * $GITHUB_STEP_SUMMARY.
 *
 * @param {string} sessionId
 * @param {Record<string, unknown>} config
 * @returns {(pi: unknown) => void}
 */
function makeObservabilityExtension(sessionId, config) {
  return function observabilityExtension(/** @type {any} */ pi) {
    let turns = 0;
    let inputTokens = 0;
    let outputTokens = 0;
    const startMs = Date.now();

    pi.on("session_start", () => {
      emitJsonl({
        type: "session_start",
        session_id: sessionId,
        model: (config.model) || null,
        timestamp: new Date().toISOString(),
      });
    });

    pi.on("turn_end", (/** @type {any} */ event) => {
      turns++;
      const usage = event?.usage ?? {};
      inputTokens += usage.inputTokens ?? 0;
      outputTokens += usage.outputTokens ?? 0;
      process.stderr.write(
        `[aw-harness] turn ${turns}: input=${inputTokens} output=${outputTokens} tokens\n`,
      );
    });

    pi.on("session_end", (/** @type {any} */ event) => {
      const durationMs = Date.now() - startMs;
      emitJsonl({
        type: "session_end",
        session_id: sessionId,
        status: event?.error ? "error" : "success",
        turns,
        input_tokens: inputTokens,
        output_tokens: outputTokens,
        duration_ms: durationMs,
        timestamp: new Date().toISOString(),
      });
      const durationSec = (durationMs / 1000).toFixed(1);
      appendStepSummary(
        `### AW Harness Execution Summary\n\n` +
        `| Field | Value |\n` +
        `|---|---|\n` +
        `| Session ID | \`${sessionId}\` |\n` +
        `| Model | \`${config.model || "(default)"}\` |\n` +
        `| Turns | ${turns} |\n` +
        `| Input tokens | ${inputTokens} |\n` +
        `| Output tokens | ${outputTokens} |\n` +
        `| Duration | ${durationSec}s |\n` +
        `| Status | ${event?.error ? "❌ Error" : "✅ Success"} |\n\n`,
      );
    });
  };
}

// ---------------------------------------------------------------------------
// Main entry point
// ---------------------------------------------------------------------------

/**
 * Main entry point.  Parses CLI arguments, loads inputs, creates a Pi
 * AgentSession, runs the prompt to completion, and exits.
 */
async function main() {
  const { configPath, promptPath } = parseArgs(process.argv);

  // --- Invocation validation (exit code 2) ---
  if (!configPath) {
    process.stderr.write("[aw-harness] error: --config argument is required\n");
    process.exit(2);
  }
  if (!promptPath) {
    process.stderr.write("[aw-harness] error: --prompt argument is required\n");
    process.exit(2);
  }

  /** @type {Record<string, unknown>} */
  let config;
  try {
    config = loadConfig(configPath);
  } catch (err) {
    process.stderr.write(`[aw-harness] error: ${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(2);
  }

  let prompt;
  try {
    prompt = loadPrompt(promptPath);
  } catch (err) {
    process.stderr.write(`[aw-harness] error: ${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(2);
  }

  // --- Load Pi SDK (installed at runtime by AWF) ---
  /** @type {any} */
  let piSdk;
  try {
    // @ts-ignore — package installed at runtime; not bundled in repo
    piSdk = await import("@earendil-works/pi-coding-agent");
    // Importing @earendil-works/pi-ai registers all built-in API providers
    // @ts-ignore — package installed at runtime; not bundled in repo
    await import("@earendil-works/pi-ai");
  } catch (err) {
    process.stderr.write(
      "[aw-harness] error: @earendil-works/pi-coding-agent is not installed.\n" +
      "The Pi SDK packages are installed by AWF before this harness runs in the\n" +
      "GitHub Actions container. Check that the AWF engine setup step completed\n" +
      "successfully.\n",
    );
    process.exit(1);
  }

  const { createAgentSession, SessionManager } = piSdk;

  // --- Build session ---
  const sessionId = crypto.randomUUID();

  const extensions = [
    makeProviderSetupExtension(),
    makeObservabilityExtension(sessionId, config),
  ];

  // Load user-declared extensions from harness.extensions (section 6.1.4)
  const userExtensions = /** @type {unknown[]} */ (
    Array.isArray((/** @type {any} */ (config).harness?.extensions)
      ? (/** @type {any} */ (config)).harness.extensions
      : [])
  );
  const extensionsRequired = Boolean((/** @type {any} */ (config)).harness?.["extensions-required"]);

  for (const extEntry of userExtensions) {
    const extPath = String(extEntry);
    try {
      // @ts-ignore — dynamic user extension
      const extModule = require(extPath);
      const extFn = extModule?.default ?? extModule;
      if (typeof extFn !== "function") {
        throw new Error("extension module does not export a default function");
      }
      extensions.push(extFn);
      log(`loaded user extension: ${extPath}`);
    } catch (err) {
      const msg = `failed to load user extension '${extPath}': ${err instanceof Error ? err.message : String(err)}`;
      if (extensionsRequired) {
        process.stderr.write(`[aw-harness] error: ${msg}\n`);
        process.exit(1);
      } else {
        log(`warning: ${msg} (continuing without this extension)`);
      }
    }
  }

  // --- Run session (exit code 0 on success, 1 on failure) ---
  try {
    const { session } = await createAgentSession({
      sessionManager: SessionManager.inMemory(),
      extensions,
      model: (/** @type {any} */ (config)).model || undefined,
    });

    await session.prompt(prompt);
    session.dispose();
    process.exit(0);
  } catch (err) {
    const msg = err instanceof Error ? err.message : String(err);
    process.stderr.write(`[aw-harness] error: session failed: ${msg}\n`);
    process.exit(1);
  }
}

// Only run main() when invoked directly (not when required/imported by tests).
if (require.main === module) {
  main().catch(err => {
    process.stderr.write(`[aw-harness] fatal: ${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
  });
}

// ---------------------------------------------------------------------------
// Exported helpers (for testing)
// ---------------------------------------------------------------------------
module.exports = {
  parseArgs,
  loadConfig,
  loadPrompt,
  buildProviderConfigs,
  appendStepSummary,
};
