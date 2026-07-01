// @ts-check

/**
 * Copilot SDK Driver (entry point)
 *
 * Minimal standalone program launched by copilot_harness.cjs.  Reads
 * configuration from environment variables, delegates the full session
 * lifecycle to copilot_sdk_session.cjs, and exits with the session's exit code.
 *
 *   GH_AW_PROMPT                            — path to the prompt file
 *   COPILOT_SDK_URI                          — SDK server URI (set by the harness)
 *   COPILOT_CONNECTION_TOKEN                 — shared secret for the SDK session (set by the harness)
 *   COPILOT_MODEL                            — model override (optional)
 *   GH_AW_COPILOT_SDK_PROVIDER_BASE_URL     — BYOK provider base URL (single-provider, set by the harness)
 *   GH_AW_COPILOT_SDK_PROVIDER_TYPE         — BYOK provider type: "openai" | "azure" | "anthropic" (single-provider)
 *   GH_AW_COPILOT_SDK_PROVIDER_WIRE_API     — BYOK provider wire API: "completions" | "responses" (single-provider)
 *   GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON   — JSON-encoded multi-provider config (takes precedence over
 *                                             single-provider vars when set).  Shape:
 *                                             { model, providers: NamedProviderConfig[], models: ProviderModelConfig[] }
 *   GH_AW_COPILOT_SDK_SERVER_ARGS           — JSON-encoded allow-tool sidecar args (set by the engine)
 *
 * Provider selection priority:
 *   1. GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON — experimental multi-provider BYOK surface
 *   2. GH_AW_COPILOT_SDK_PROVIDER_BASE_URL   — single whole-session BYOK provider (legacy)
 *
 * The sidecar is started and stopped by the harness; the driver only opens a
 * client connection, runs the session, and exits.
 *
 * Reusable helpers live in:
 *   copilot_sdk_permissions.cjs — permission config parsing and handler builder
 *   copilot_sdk_session.cjs     — session runner and JSONL event serialization
 */

"use strict";

const fs = require("fs");
const { runWithCopilotSDK, extractPromptFromArgs } = require("./copilot_sdk_session.cjs");
const { parsePermissionConfigFromServerArgs } = require("./copilot_sdk_permissions.cjs");

// Re-export the session and permission helpers so that existing callers that
// require("./copilot_sdk_driver.cjs") (e.g. copilot_harness.cjs) continue to work.
module.exports = { extractPromptFromArgs, runWithCopilotSDK, parsePermissionConfigFromServerArgs, parseWireApiEnv, parseMultiProviderJson };

// ---------------------------------------------------------------------------
// Standalone entry point
// ---------------------------------------------------------------------------

/**
 * Log a message prefixed with [copilot-sdk-driver] to stderr.
 * @param {string} msg
 */
function log(msg) {
  process.stderr.write(`[copilot-sdk-driver] ${msg}\n`);
}

/**
 * Normalize the optional provider wire API env var.
 *
 * @param {string | undefined} raw
 * @returns {"completions" | "responses" | undefined}
 */
function parseWireApiEnv(raw) {
  const normalized = String(raw || "")
    .toLowerCase()
    .trim();
  return normalized === "responses" || normalized === "completions" ? normalized : undefined;
}

/**
 * Parse the GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON env var.
 *
 * Returns `null` when the env var is unset or contains invalid JSON.
 * On success returns `{ model, providers, models }` where the shapes match the
 * Copilot SDK `NamedProviderConfig` / `ProviderModelConfig` types.
 *
 * @param {string | undefined} raw
 * @returns {{
 *   model: string,
 *   providers: import("@github/copilot-sdk").NamedProviderConfig[],
 *   models: import("@github/copilot-sdk").ProviderModelConfig[],
 * } | null}
 */
function parseMultiProviderJson(raw) {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") return null;
    if (!Array.isArray(parsed.providers) || parsed.providers.length < 2) return null;
    if (!Array.isArray(parsed.models)) return null;
    const model = typeof parsed.model === "string" ? parsed.model.trim() : "";
    return { model, providers: parsed.providers, models: parsed.models };
  } catch {
    return null;
  }
}

/**
 * Entry point when the driver is run directly with Node:
 *   node copilot_sdk_driver.cjs
 *
 * Reads configuration from environment variables and connects to the headless
 * Copilot CLI sidecar that has already been started by copilot_harness.cjs.
 * Runs a single SDK session and exits with the session's exit code.
 * Any unhandled error causes a non-zero exit.
 */
async function main() {
  // --- Read configuration from environment ---------------------

  const promptFile = process.env.GH_AW_PROMPT;
  if (!promptFile) {
    process.stderr.write("[copilot-sdk-driver] error: GH_AW_PROMPT is not set\n");
    process.exit(1);
  }

  const sdkUri = process.env.COPILOT_SDK_URI;
  if (!sdkUri) {
    process.stderr.write("[copilot-sdk-driver] error: COPILOT_SDK_URI is not set\n");
    process.exit(1);
  }

  const connectionToken = process.env.COPILOT_CONNECTION_TOKEN;
  if (!connectionToken) {
    process.stderr.write("[copilot-sdk-driver] error: COPILOT_CONNECTION_TOKEN is required. This token is generated by copilot_harness.cjs and must be passed to the driver environment\n");
    process.exit(1);
  }

  // --- Read the prompt -------------------------------------------------

  let prompt;
  try {
    prompt = fs.readFileSync(promptFile, "utf8");
  } catch (err) {
    process.stderr.write(`[copilot-sdk-driver] error: failed to read prompt file ${promptFile}: ${err}\n`);
    process.exit(1);
  }

  log(`connecting to sidecar at ${sdkUri}`);

  // --- Resolve provider configuration from environment -----------------
  // The harness injects provider config before launching this driver.
  //
  // Priority order:
  //   1. GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON — experimental multi-provider BYOK surface:
  //      supports different wireApi per model by routing each model to a named provider.
  //   2. GH_AW_COPILOT_SDK_PROVIDER_BASE_URL   — single whole-session BYOK provider (legacy).
  //
  // BYOK is the only supported mode — fail immediately when neither is present.

  /** @type {import("@github/copilot-sdk").ProviderConfig | undefined} */
  let provider;
  /** @type {import("@github/copilot-sdk").NamedProviderConfig[] | undefined} */
  let providers;
  /** @type {import("@github/copilot-sdk").ProviderModelConfig[] | undefined} */
  let sdkModels;
  let model = process.env.COPILOT_MODEL || undefined;

  const multiProviderConfig = parseMultiProviderJson(process.env.GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON);
  if (multiProviderConfig) {
    providers = multiProviderConfig.providers;
    sdkModels = multiProviderConfig.models;
    // Prefer the model from the multi-provider config when COPILOT_MODEL is unset.
    if (!model && multiProviderConfig.model) {
      model = multiProviderConfig.model;
    }
    log(`multi-provider mode: ${providers.length} providers, ${sdkModels.length} models, model=${model ?? "(env)"}`);
    for (const p of providers) {
      log(`  provider: name=${p.name} type=${p.type} baseUrl=${p.baseUrl}${p.wireApi ? ` wireApi=${p.wireApi}` : ""}`);
    }
  } else {
    // Fall back to legacy single whole-session BYOK provider.
    const providerBaseUrl = process.env.GH_AW_COPILOT_SDK_PROVIDER_BASE_URL;
    if (!providerBaseUrl) {
      process.stderr.write(
        "[copilot-sdk-driver] error: no BYOK provider configured — " +
          "set GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON for multi-provider mode or " +
          "GH_AW_COPILOT_SDK_PROVIDER_BASE_URL for single-provider mode; " +
          "ensure the harness resolved a custom provider from awf-reflect data\n"
      );
      process.exit(1);
    }
    const rawProviderType = process.env.GH_AW_COPILOT_SDK_PROVIDER_TYPE || "openai";
    /** @type {"openai" | "azure" | "anthropic"} */
    const providerType = rawProviderType === "anthropic" || rawProviderType === "azure" ? rawProviderType : "openai";
    log(`single-provider mode: type=${providerType} baseUrl=${providerBaseUrl}`);
    const wireApi = parseWireApiEnv(process.env.GH_AW_COPILOT_SDK_PROVIDER_WIRE_API);
    provider = { type: providerType, baseUrl: providerBaseUrl, ...(wireApi ? { wireApi } : {}) };
  }

  // --- Build permission config from sidecar server args ----------------
  // GH_AW_COPILOT_SDK_SERVER_ARGS holds the JSON-encoded --allow-tool flags
  // that the Go engine passed to the sidecar. Mirror those same rules in the
  // SDK session so the driver's onPermissionRequest handler aligns with the
  // sidecar's pre-configured allow list (e.g. shell(safeoutputs:*) for
  // workflows with safe-outputs enabled and a restricted bash allowlist).
  const permissionConfig = parsePermissionConfigFromServerArgs(process.env.GH_AW_COPILOT_SDK_SERVER_ARGS);
  if (permissionConfig) {
    if (permissionConfig.allowAllTools) {
      log("permission config: allow-all-tools (sidecar launched with --allow-all-tools)");
    } else {
      log(`permission config: ${(permissionConfig.allowedTools ?? []).length} allow-tool entries from GH_AW_COPILOT_SDK_SERVER_ARGS`);
    }
  } else {
    log("permission config: none (onPermissionRequest will use unrestricted behavior)");
  }

  // --- Run SDK session -------------------------------------------------

  const result = await runWithCopilotSDK({
    sdkUri,
    prompt,
    logger: log,
    model,
    connectionToken,
    provider,
    providers,
    models: sdkModels,
    permissionConfig,
  });

  process.exit(result.exitCode);
}

if (require.main === module) {
  main().catch(err => {
    process.stderr.write(`[copilot-sdk-driver] unhandled error: ${err instanceof Error ? err.stack : String(err)}\n`);
    process.exit(1);
  });
}
