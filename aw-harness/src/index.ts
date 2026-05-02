/**
 * ⚠️  EXPERIMENTAL — AW Harness Entry Point
 *
 * This file is the entry point for the `engine: aw` execution engine.
 * It is EXPERIMENTAL and subject to breaking changes without notice.
 * Do not use in production workflows without understanding the risks.
 *
 * See specs/aw-harness.md for the full specification.
 *
 * Usage:
 *   node aw_harness.cjs --config <config-path> --prompt <prompt-path>
 *
 * Exit codes:
 *   0 — Prompt completed successfully
 *   1 — Session failed or budget exceeded
 *   2 — Invocation error (missing arguments, unreadable files)
 */

import {
  createAgentSession,
  DefaultResourceLoader,
  SessionManager,
  getAgentDir,
} from "@mariozechner/pi-coding-agent";

import { assemblePrompt } from "./context.js";
import { loadInputs, parseArgs } from "./loader.js";
import { createSharedState } from "./types.js";
import { loadUserExtensions } from "./user-extensions.js";
import { createCostTrackerExtension } from "./extensions/cost-tracker.js";
import { createObservabilityExtension } from "./extensions/observability.js";
import { providerSetupExtension } from "./extensions/provider-setup.js";
import { createRepairExtension } from "./extensions/repair.js";
import { createSteeringExtension } from "./extensions/steering.js";

// ─── main ────────────────────────────────────────────────────────────────────

async function main(): Promise<void> {
  // Emit experimental warning on every invocation
  process.stderr.write(
    "[aw-harness] ⚠️  EXPERIMENTAL: engine: aw is experimental and subject to breaking changes.\n",
  );

  // ── Parse CLI arguments ──
  let configPath: string;
  let promptPath: string;
  try {
    ({ configPath, promptPath } = parseArgs(process.argv.slice(2)));
  } catch (err) {
    process.stderr.write(`[aw-harness] ✗ ${(err as Error).message}\n`);
    process.exit(2);
  }

  // ── Load config.json + prompt.txt ──
  let config: Awaited<ReturnType<typeof loadInputs>>["config"];
  let promptBody: string;
  try {
    ({ config, promptBody } = loadInputs(configPath, promptPath));
  } catch (err) {
    process.stderr.write(`[aw-harness] ✗ ${(err as Error).message}\n`);
    process.exit(2);
  }

  process.stderr.write(
    `[aw-harness] ✓ Loaded config (model: ${config.model}, timeout: ${config.timeoutMinutes}min)\n`,
  );

  // ── Assemble prompt (imports prepended to prompt body) ──
  const prompt = assemblePrompt(config.imports ?? [], promptBody);

  // ── Create shared runtime state ──
  const sharedState = createSharedState(config.model);

  // ── Build built-in extension factories ──
  const builtinExtensions = [
    providerSetupExtension,
    createCostTrackerExtension(config.budget, config.steering, sharedState),
    createSteeringExtension(config.timeoutMinutes, config.steering, sharedState),
    createRepairExtension(),
    createObservabilityExtension(
      config.imports ?? [],
      config.observability,
      sharedState,
      promptBody,
    ),
  ];

  // ── Load user-declared extensions (harness.extensions) ──
  let userExtensions: Awaited<ReturnType<typeof loadUserExtensions>> = [];
  try {
    userExtensions = await loadUserExtensions(
      config.extensions ?? [],
      process.cwd(),
      config.extensionsRequired ?? false,
    );
  } catch (err) {
    // loadUserExtensions only throws if extensionsRequired: true
    process.stderr.write(`[aw-harness] ✗ User extension load failed: ${(err as Error).message}\n`);
    process.exit(1);
  }

  const allExtensions = [...builtinExtensions, ...userExtensions];

  // ── Create Pi AgentSession ──
  const agentDir = getAgentDir();
  const resourceLoader = new DefaultResourceLoader({
    cwd: process.cwd(),
    agentDir,
    extensionFactories: allExtensions,
    // No auto-discovery of skills/context files per spec §6.4
    noContextFiles: true,
    noSkills: true,
    noPromptTemplates: true,
    noThemes: true,
  });
  await resourceLoader.reload();

  let session: Awaited<ReturnType<typeof createAgentSession>>["session"];
  try {
    ({ session } = await createAgentSession({
      sessionManager: SessionManager.inMemory(),
      resourceLoader,
      // Disable built-in interactive tools; bash is available via cli-proxy on PATH
      noTools: "builtin",
    }));
  } catch (err) {
    process.stderr.write(
      `[aw-harness] ✗ Failed to create agent session: ${(err as Error).message}\n`,
    );
    process.exit(1);
  }

  // ── Run the session ──
  process.stderr.write(`[aw-harness] → Starting session (model: ${config.model})\n`);

  try {
    await session.prompt(prompt, { source: "rpc" });
  } catch (err) {
    process.stderr.write(
      `[aw-harness] ✗ Session failed: ${(err as Error).message}\n`,
    );
    session.dispose();
    process.exit(1);
  } finally {
    session.dispose();
  }

  // ── Check post-session exit conditions ──
  if (sharedState.budgetAborted) {
    process.stderr.write("[aw-harness] ✗ Session aborted: budget limit exceeded.\n");
    process.exit(1);
  }

  process.stderr.write("[aw-harness] ✓ Session completed successfully.\n");
  process.exit(0);
}

main().catch((err: unknown) => {
  const msg = err instanceof Error ? err.message : String(err);
  process.stderr.write(`[aw-harness] ✗ Unhandled error: ${msg}\n`);
  process.exit(1);
});
