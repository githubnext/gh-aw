/**
 * User extension loader.
 *
 * Loads user-declared Pi extensions from harness.extensions paths in config.json.
 * Each entry is either a repo-relative path (./…) or an npm package name.
 *
 * Per spec §6.1.4:
 * - Emit a warning to stderr if an extension fails to load.
 * - Only abort if harness.extensions-required: true is set.
 * - User extensions MUST NOT override or replace built-in gh-aw extensions.
 */

import { resolve } from "node:path";
import type { ExtensionFactory } from "@mariozechner/pi-coding-agent";

// ─── loadUserExtensions ──────────────────────────────────────────────────────

/**
 * Load and validate user-declared Pi extensions.
 *
 * @param refs - Extension references from harness.extensions (paths or npm names)
 * @param cwd - Working directory for resolving relative paths
 * @param extensionsRequired - Whether to throw on load failure (default: false)
 * @returns Array of loaded ExtensionFactory functions
 */
export async function loadUserExtensions(
  refs: string[],
  cwd: string,
  extensionsRequired: boolean,
): Promise<ExtensionFactory[]> {
  const loaded: ExtensionFactory[] = [];

  for (const ref of refs) {
    try {
      const factory = await loadOneExtension(ref, cwd);
      loaded.push(factory);
    } catch (err) {
      const msg = `[aw-harness] ⚠️ Failed to load user extension '${ref}': ${(err as Error).message}`;
      process.stderr.write(msg + "\n");
      if (extensionsRequired) {
        throw new Error(msg);
      }
    }
  }

  return loaded;
}

// ─── loadOneExtension ────────────────────────────────────────────────────────

/**
 * Load a single Pi extension from a path or package name.
 *
 * @param ref - Repo-relative path (./…) or npm package name
 * @param cwd - Working directory for resolving relative paths
 */
async function loadOneExtension(ref: string, cwd: string): Promise<ExtensionFactory> {
  // Resolve repo-relative paths; leave npm package names unchanged
  const resolved =
    ref.startsWith("./") || ref.startsWith("../")
      ? resolve(cwd, ref)
      : ref;

  // Use CommonJS require() — compatible with the bundled CJS output and with
  // compiled Pi extension modules, which are also CJS.
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const mod = require(resolved) as Record<string, unknown>;

  // Normalise: support both module.exports = fn and module.exports.default = fn
  const factory: unknown = typeof mod?.["default"] === "function" ? mod["default"] : mod;

  if (typeof factory !== "function") {
    throw new Error(`Extension '${ref}' does not export a default function`);
  }

  return factory as ExtensionFactory;
}
