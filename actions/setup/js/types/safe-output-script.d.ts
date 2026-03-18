// TypeScript definitions for GitHub Agentic Workflows Safe Output Script Handlers
// This file describes the types available when writing a custom safe-output script
// (defined under safe-outputs.scripts in workflow frontmatter).
//
// Usage in a script body (only the body of main() is written by the user):
//
//   const { inputs } = config;
//   return async function handleMyScript(item) {
//     const channel = item.channel ?? inputs?.channel?.default ?? "#general";
//     core.info(`[SLACK] → ${channel}: ${item.message}`);
//     return { success: true };
//   };

import type { HandlerResult } from "./handler-factory";
export type { HandlerResult };

// ── Input-definition types ──────────────────────────────────────────────────

/**
 * The definition of a single user-declared input from the YAML `inputs:` section.
 * These definitions are available at runtime through `config.inputs`.
 */
export interface SafeOutputScriptInputDefinition {
  /** The declared type of this input ("string" | "boolean" | "number"). */
  type?: "string" | "boolean" | "number";
  /** Human-readable description shown in MCP tool registration. */
  description?: string;
  /** Whether the caller is required to supply a value for this input. */
  required?: boolean;
  /**
   * The default value to use when the caller omits the input.
   * `null` means no default was specified.
   */
  default?: string | boolean | number | null;
  /** Available options when `type` is "string" (choice constraint). */
  options?: string[];
}

// ── Config type ─────────────────────────────────────────────────────────────

/**
 * The `config` object passed to the `main()` factory function of a
 * custom safe-output script.
 *
 * This contains the **static** YAML configuration for the script — the
 * description and the input-definition metadata. The actual per-call input
 * values sent by the agent are exposed as direct properties on the `item`
 * object inside the handler function (not here).
 *
 * @example
 * ```javascript
 * // config.inputs.channel.required === true
 * // config.inputs.channel.type === "string"
 * const { inputs } = config;
 * return async function handleMyScript(item) {
 *   const ch = item.channel ?? inputs?.channel?.default ?? "#general";
 *   return { success: true };
 * };
 * ```
 */
export interface SafeOutputScriptConfig {
  /**
   * Human-readable description of this script (from `description:` in YAML).
   * Used in MCP tool registration.
   */
  description?: string;
  /**
   * Metadata for each declared input.
   * Keys are input names as declared in the YAML `inputs:` section.
   * **Note**: This is the *schema* for each input (type, description, required, default),
   * not the runtime values. Use `item.<inputName>` inside the handler to access values.
   */
  inputs?: Record<string, SafeOutputScriptInputDefinition>;
}

// ── Per-call message type ───────────────────────────────────────────────────

/**
 * The per-call message object passed to the handler function returned by `main()`.
 *
 * For custom safe-output scripts the agent sends a JSONL line like:
 * ```json
 * { "type": "post_slack_message", "channel": "#general", "message": "Hello" }
 * ```
 * All user-declared input values are properties at the **top level** of this
 * object (not nested under `.data`).
 *
 * @typeParam TInputs - The shape of the user-declared inputs.  When omitted the
 *   properties are typed as `unknown` and can be narrowed at runtime.
 *
 * @example
 * ```typescript
 * // With explicit input types:
 * type SlackInputs = { channel?: string; message?: string };
 * return async function handleSlack(item: SafeOutputScriptItem<SlackInputs>) {
 *   core.info(`channel: ${item.channel ?? "#general"}`);
 * };
 * ```
 */
export type SafeOutputScriptItem<TInputs extends Record<string, unknown> = Record<string, unknown>> = {
  /** The safe-output type identifier (normalized script name, e.g. "post_slack_message"). */
  type: string;
  /** Optional secrecy level of the message content (e.g. "public", "internal", "private"). */
  secrecy?: string;
  /** Optional integrity level of the message source (e.g. "low", "medium", "high"). */
  integrity?: string;
} & TInputs;

// ── Resolved temporary IDs ──────────────────────────────────────────────────

/**
 * Map of temporary IDs to their resolved GitHub issue/PR/discussion references.
 * Passed as the second argument to the handler function.
 */
export interface ResolvedTemporaryIds {
  [temporaryId: string]: {
    /** Repository in "owner/repo" format. */
    repo: string;
    /** Issue, PR, or discussion number. */
    number: number;
  };
}

// ── Handler and factory function types ─────────────────────────────────────

/**
 * The async message-handler function returned by `main()`.
 * Receives a single safe-output message and should return a `HandlerResult`.
 *
 * @typeParam TInputs - The shape of the user-declared inputs (defaults to
 *   `Record<string, unknown>`).
 */
export type SafeOutputScriptHandler<TInputs extends Record<string, unknown> = Record<string, unknown>> = (item: SafeOutputScriptItem<TInputs>, resolvedTemporaryIds: ResolvedTemporaryIds) => Promise<HandlerResult>;

/**
 * The type of the `main()` function generated by the compiler around the user's
 * script body.
 *
 * The compiler wraps the user-written body in:
 * ```javascript
 * async function main(config = {}) {
 *   // <user body here>
 * }
 * module.exports = { main };
 * ```
 *
 * The `main` function receives the static YAML configuration and returns an
 * async handler function that processes individual messages.
 *
 * @typeParam TInputs - The shape of the user-declared inputs.
 */
export type SafeOutputScriptMain<TInputs extends Record<string, unknown> = Record<string, unknown>> = (config: SafeOutputScriptConfig) => Promise<SafeOutputScriptHandler<TInputs>>;

/**
 * The `main` factory function exported by every auto-generated safe-output
 * script module (`module.exports = { main }`).
 *
 * This TypeScript declaration provides IDE type-checking support for the
 * CommonJS export (`module.exports = { main }`) that the compiler generates.
 */
export declare function main(config: SafeOutputScriptConfig): Promise<SafeOutputScriptHandler>;

// ── Globals available in the script body ────────────────────────────────────
// The globals below are injected by the `actions/github-script` environment
// that hosts the handler manager.  They are already declared in
// `github-script.d.ts`; this comment serves as a quick reference.
//
//   github   — authenticated Octokit instance
//   context  — GitHub Actions workflow run context
//   core     — @actions/core (setOutput, info, warning, error, …)
//   exec     — @actions/exec
//   glob     — @actions/glob
//   io       — @actions/io
//   require  — CommonJS require (supports relative paths and npm packages)
