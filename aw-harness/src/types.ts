/**
 * Shared type definitions for the AW Harness.
 *
 * These types match the compiler-generated config.json schema consumed at runtime.
 * The harness MUST NOT re-parse raw Markdown frontmatter; all configuration comes
 * from the pre-processed inputs produced by the gh-aw compiler.
 */

// ─── Budget ─────────────────────────────────────────────────────────────────

/** Budget configuration from harness.budget in workflow frontmatter. */
export interface BudgetConfig {
  /** Maximum effective token count for the run. */
  maxEffectiveTokens: number;
}

// ─── Context ─────────────────────────────────────────────────────────────────

/** Compaction strategy. */
export type CompactionMode = "none" | "sliding-window" | "summarize";

/** Context configuration from harness.context. */
export interface ContextConfig {
  /** Compaction mode. Default: "none". */
  compaction: CompactionMode;
  /** Fraction of context window at which compaction triggers. Default: 0.75. */
  compactionThreshold: number;
}

// ─── Steering ────────────────────────────────────────────────────────────────

/** Steering configuration from harness.steering. */
export interface SteeringConfig {
  /** Minutes before timeout at which a warning is injected. Default: 5. */
  timeWarningMinutes: number;
  /** Minutes before timeout at which a critical message is injected. Default: 2. */
  timeCriticalMinutes: number;
  /** Budget percentage at which a warning is injected. Default: 75. */
  budgetWarnPercent: number;
  /** Budget percentage at which the session is aborted. Default: 90. */
  budgetCriticalPercent: number;
}

// ─── Observability ───────────────────────────────────────────────────────────

/** OTLP endpoint configuration. */
export interface OtlpConfig {
  endpoint: string;
  headers?: Record<string, string>;
}

/** Observability configuration from the workflow frontmatter. */
export interface ObservabilityConfig {
  otlp?: OtlpConfig;
}

// ─── Top-level HarnessConfig ─────────────────────────────────────────────────

/**
 * Compiler-generated harness configuration consumed from config.json.
 *
 * Corresponds to the `harness:` block in the workflow frontmatter, with
 * defaults already applied by the compiler.
 */
export interface HarnessConfig {
  /** Model alias or fully-qualified provider/model string. */
  model: string;

  /** Timeout for the workflow run, in minutes. */
  timeoutMinutes: number;

  /** Budget configuration. Absent means no budget limit. */
  budget?: BudgetConfig;

  /** Context/compaction configuration. */
  context: ContextConfig;

  /** Steering (time/budget pressure) configuration. */
  steering: SteeringConfig;

  /** Observability configuration. */
  observability?: ObservabilityConfig;

  /**
   * User-declared extension references from harness.extensions.
   * Each entry is a repo-relative path (./…) or npm package name.
   */
  extensions?: string[];

  /**
   * When true, abort the session if any user extension fails to load.
   * Default: false (emit warning only).
   */
  extensionsRequired?: boolean;

  /**
   * Resolved import entries from the imports: frontmatter key.
   * Each entry contains the file path and its full text content,
   * resolved by the compiler before the harness is invoked.
   */
  imports?: ImportEntry[];
}

/** A single resolved import file. */
export interface ImportEntry {
  /** Repository-relative path (e.g. "skills/reporting/SKILL.md"). */
  path: string;
  /** Full text content of the file. */
  content: string;
}

// ─── Loaded inputs ───────────────────────────────────────────────────────────

/** Fully loaded harness inputs returned by loadInputs(). */
export interface LoadedInputs {
  config: HarnessConfig;
  /** Assembled prompt string (imports + prompt.txt body). */
  prompt: string;
}

// ─── Shared runtime state ────────────────────────────────────────────────────

/**
 * Mutable state shared across extensions and the main entry point.
 * Extensions close over this object to communicate outcomes back to the harness.
 */
export interface SharedHarnessState {
  /**
   * Set by the cost-tracker extension when the budget critical threshold is hit.
   * The main entry point exits with code 1 after session.dispose() when this is set.
   */
  budgetAborted: boolean;

  /**
   * Running cumulative token total, updated by the cost-tracker extension on each turn.
   */
  cumulativeTokens: number;

  /**
   * Running cumulative estimated cost (USD), updated by the cost-tracker extension.
   */
  cumulativeCostUsd: number;

  /**
   * Turn index counter, incremented on each turn_end event.
   */
  turnCount: number;

  /**
   * Session start timestamp (ms since epoch), set on agent_start.
   */
  sessionStartMs: number;

  /**
   * Name of the model used by the session, set on agent_start or model_select.
   */
  model: string;
}

/** Create a fresh SharedHarnessState with default values. */
export function createSharedState(model: string): SharedHarnessState {
  return {
    budgetAborted: false,
    cumulativeTokens: 0,
    cumulativeCostUsd: 0,
    turnCount: 0,
    sessionStartMs: 0,
    model,
  };
}
