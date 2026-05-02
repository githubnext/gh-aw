/**
 * Loader for compiler-generated harness inputs.
 *
 * Reads config.json and prompt.txt produced by the gh-aw compiler and
 * assembles the final prompt string (imports prepended to prompt body).
 *
 * The harness MUST NOT read or parse raw workflow Markdown files directly.
 * All configuration and prompt content MUST come from these pre-processed inputs.
 */

import { readFileSync } from "node:fs";
import type { HarnessConfig } from "./types.js";

// ─── Defaults ────────────────────────────────────────────────────────────────

/** Default values applied when optional harness config keys are absent. */
export const CONFIG_DEFAULTS = {
  timeoutMinutes: 60,
  context: {
    compaction: "none" as const,
    compactionThreshold: 0.75,
  },
  steering: {
    timeWarningMinutes: 5,
    timeCriticalMinutes: 2,
    budgetWarnPercent: 75,
    budgetCriticalPercent: 90,
  },
};

// ─── parseConfig ─────────────────────────────────────────────────────────────

/**
 * Parse and validate the compiler-generated config.json object, applying defaults.
 *
 * @param raw - Parsed JSON object from config.json
 * @throws {Error} if required fields are missing or invalid
 */
export function parseConfig(raw: unknown): HarnessConfig {
  if (!raw || typeof raw !== "object") {
    throw new Error("config.json must be a JSON object");
  }

  const r = raw as Record<string, unknown>;

  if (typeof r["model"] !== "string" || r["model"].trim() === "") {
    throw new Error('config.json must include a non-empty "model" string');
  }

  const timeoutMinutes =
    typeof r["timeoutMinutes"] === "number" && r["timeoutMinutes"] > 0
      ? r["timeoutMinutes"]
      : CONFIG_DEFAULTS.timeoutMinutes;

  // ── budget ──
  let budget: HarnessConfig["budget"];
  if (r["budget"] && typeof r["budget"] === "object") {
    const b = r["budget"] as Record<string, unknown>;
    if (typeof b["maxEffectiveTokens"] === "number" && b["maxEffectiveTokens"] > 0) {
      budget = { maxEffectiveTokens: b["maxEffectiveTokens"] };
    } else {
      // Log a warning so workflow authors understand why budget was ignored
      process.stderr.write(
        `[aw-harness] ⚠️ Ignoring invalid budget configuration: ` +
          `maxEffectiveTokens must be a positive number (got: ${JSON.stringify(b["maxEffectiveTokens"])})\n`,
      );
    }
  }

  // ── context ──
  const rawCtx =
    r["context"] && typeof r["context"] === "object" ? (r["context"] as Record<string, unknown>) : {};
  const compactionRaw = rawCtx["compaction"];
  const compaction =
    compactionRaw === "none" || compactionRaw === "sliding-window" || compactionRaw === "summarize"
      ? compactionRaw
      : CONFIG_DEFAULTS.context.compaction;
  const compactionThreshold =
    typeof rawCtx["compactionThreshold"] === "number"
      ? Math.max(0, Math.min(1, rawCtx["compactionThreshold"]))
      : CONFIG_DEFAULTS.context.compactionThreshold;

  // ── steering ──
  const rawSt =
    r["steering"] && typeof r["steering"] === "object" ? (r["steering"] as Record<string, unknown>) : {};
  const steering = {
    timeWarningMinutes:
      typeof rawSt["timeWarningMinutes"] === "number"
        ? rawSt["timeWarningMinutes"]
        : CONFIG_DEFAULTS.steering.timeWarningMinutes,
    timeCriticalMinutes:
      typeof rawSt["timeCriticalMinutes"] === "number"
        ? rawSt["timeCriticalMinutes"]
        : CONFIG_DEFAULTS.steering.timeCriticalMinutes,
    budgetWarnPercent:
      typeof rawSt["budgetWarnPercent"] === "number"
        ? rawSt["budgetWarnPercent"]
        : CONFIG_DEFAULTS.steering.budgetWarnPercent,
    budgetCriticalPercent:
      typeof rawSt["budgetCriticalPercent"] === "number"
        ? rawSt["budgetCriticalPercent"]
        : CONFIG_DEFAULTS.steering.budgetCriticalPercent,
  };

  // ── observability ──
  let observability: HarnessConfig["observability"];
  if (r["observability"] && typeof r["observability"] === "object") {
    const obs = r["observability"] as Record<string, unknown>;
    if (obs["otlp"] && typeof obs["otlp"] === "object") {
      const otlp = obs["otlp"] as Record<string, unknown>;
      if (typeof otlp["endpoint"] === "string") {
        observability = {
          otlp: {
            endpoint: otlp["endpoint"],
            headers:
              otlp["headers"] && typeof otlp["headers"] === "object"
                ? (otlp["headers"] as Record<string, string>)
                : undefined,
          },
        };
      }
    }
  }

  // ── extensions ──
  const extensions = Array.isArray(r["extensions"])
    ? (r["extensions"] as unknown[]).filter((e): e is string => typeof e === "string")
    : undefined;

  const extensionsRequired =
    typeof r["extensionsRequired"] === "boolean" ? r["extensionsRequired"] : false;

  // ── imports ──
  const imports = Array.isArray(r["imports"])
    ? (r["imports"] as unknown[]).filter(
        (e): e is { path: string; content: string } =>
          typeof e === "object" &&
          e !== null &&
          typeof (e as Record<string, unknown>)["path"] === "string" &&
          typeof (e as Record<string, unknown>)["content"] === "string",
      )
    : undefined;

  return {
    model: r["model"],
    timeoutMinutes,
    budget,
    context: { compaction, compactionThreshold },
    steering,
    observability,
    extensions,
    extensionsRequired,
    imports,
  };
}

// ─── loadInputs ──────────────────────────────────────────────────────────────

/**
 * Load and parse the compiler-generated config.json and prompt.txt.
 *
 * @param configPath - Path to config.json
 * @param promptPath - Path to prompt.txt
 * @returns Parsed config and raw prompt body
 * @throws {Error} on I/O or parse errors
 */
export function loadInputs(
  configPath: string,
  promptPath: string,
): { config: HarnessConfig; promptBody: string } {
  let rawJson: unknown;
  try {
    rawJson = JSON.parse(readFileSync(configPath, "utf8"));
  } catch (err) {
    throw new Error(`Failed to read config.json at '${configPath}': ${(err as Error).message}`);
  }

  let promptBody: string;
  try {
    promptBody = readFileSync(promptPath, "utf8");
  } catch (err) {
    throw new Error(`Failed to read prompt.txt at '${promptPath}': ${(err as Error).message}`);
  }

  const config = parseConfig(rawJson);
  return { config, promptBody };
}

// ─── parseArgs ───────────────────────────────────────────────────────────────

/**
 * Parse CLI arguments for the harness entry point.
 * Expects: --config <path> --prompt <path>
 *
 * @param argv - process.argv slice (from index 2)
 * @throws {Error} if required flags are missing
 */
export function parseArgs(argv: string[]): { configPath: string; promptPath: string } {
  let configPath: string | undefined;
  let promptPath: string | undefined;

  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === "--config" && argv[i + 1]) {
      configPath = argv[++i];
    } else if (argv[i] === "--prompt" && argv[i + 1]) {
      promptPath = argv[++i];
    }
  }

  if (!configPath) {
    throw new Error("Missing required argument: --config <path>");
  }
  if (!promptPath) {
    throw new Error("Missing required argument: --prompt <path>");
  }

  return { configPath, promptPath };
}
