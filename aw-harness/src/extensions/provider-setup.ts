/**
 * Extension 1: Provider Setup
 *
 * Registers LLM providers with Pi SDK using credentials injected by AWF into
 * the container environment. The harness MUST NOT hard-code any provider URL
 * or API key; all credentials MUST come from environment variables.
 *
 * Per spec §8.1.
 */

import type { ExtensionFactory } from "@mariozechner/pi-coding-agent";

// ─── Supported providers ─────────────────────────────────────────────────────

interface ProviderSpec {
  envKey: string;
  baseUrlEnv?: string;
  api: "anthropic-messages" | "openai-completions" | "openai-responses";
}

const PROVIDERS: Record<string, ProviderSpec> = {
  anthropic: {
    envKey: "ANTHROPIC_API_KEY",
    baseUrlEnv: "ANTHROPIC_BASE_URL",
    api: "anthropic-messages",
  },
  openai: {
    envKey: "OPENAI_API_KEY",
    baseUrlEnv: "OPENAI_BASE_URL",
    api: "openai-completions",
  },
  // GitHub Copilot uses GITHUB_TOKEN for model routing via the Copilot API
  copilot: {
    envKey: "GITHUB_TOKEN",
    baseUrlEnv: "COPILOT_BASE_URL",
    api: "openai-completions",
  },
};

// ─── providerSetupExtension ──────────────────────────────────────────────────

/**
 * Pi extension factory: register available LLM providers from environment variables.
 */
export const providerSetupExtension: ExtensionFactory = (pi) => {
  let registered = 0;

  for (const [name, spec] of Object.entries(PROVIDERS)) {
    const apiKey = process.env[spec.envKey];
    if (!apiKey) {
      continue;
    }

    const baseUrl = spec.baseUrlEnv ? process.env[spec.baseUrlEnv] : undefined;

    pi.registerProvider(name, {
      apiKey,
      api: spec.api,
      ...(baseUrl ? { baseUrl } : {}),
    });

    registered++;
    process.stderr.write(
      `[aw-harness] ✓ Registered provider '${name}'` +
        (baseUrl ? ` (base URL: ${baseUrl})` : "") +
        "\n",
    );
  }

  if (registered === 0) {
    throw new Error(
      "[aw-harness] No LLM provider credentials found in environment. " +
        "Set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GITHUB_TOKEN before running.",
    );
  }
};
