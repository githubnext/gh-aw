/**
 * Extension 5: Observability
 *
 * Emits structured JSONL event stream to stderr, writes a context provenance
 * file, renders a GitHub Actions step summary, and reports per-turn token
 * consumption.
 *
 * Per spec §8.5.
 */

import { appendFileSync, mkdirSync, writeFileSync } from "node:fs";
import { dirname } from "node:path";
import type { ExtensionFactory } from "@mariozechner/pi-coding-agent";
import type { ImportEntry, ObservabilityConfig, SharedHarnessState } from "../types.js";

// ─── Constants ───────────────────────────────────────────────────────────────

/** Default path for the context provenance JSONL file. */
export const CONTEXT_PROVENANCE_PATH = "/tmp/gh-aw/context-provenance.jsonl";

// ─── ProvenanceEntry ─────────────────────────────────────────────────────────

interface ProvenanceEntry {
  timestamp: string;
  source: "prompt" | "import";
  path?: string;
  tokens: number;
  cumulativeTokens: number;
  role: "user";
}

// ─── TurnRecord ──────────────────────────────────────────────────────────────

interface TurnRecord {
  turn: number;
  inputTokens: number;
  outputTokens: number;
  cumulativeTokens: number;
  estimatedCostUsd: number;
}

// ─── emitJsonl ───────────────────────────────────────────────────────────────

function emitJsonl(record: Record<string, unknown>): void {
  process.stderr.write(JSON.stringify(record) + "\n");
}

// ─── estimateTokens ──────────────────────────────────────────────────────────

/** Rough token estimate: ~4 chars per token. */
function estimateTokens(text: string): number {
  return Math.ceil(text.length / 4);
}

// ─── createObservabilityExtension ────────────────────────────────────────────

/**
 * Create the observability Pi extension factory.
 *
 * @param imports - Resolved import entries for context provenance tracking
 * @param observabilityConfig - Optional OTLP configuration
 * @param sharedState - Mutable runtime state shared with the entry point
 * @param promptBody - Raw prompt body text (for token estimation)
 */
export function createObservabilityExtension(
  imports: ImportEntry[],
  observabilityConfig: ObservabilityConfig | undefined,
  sharedState: SharedHarnessState,
  promptBody: string,
): ExtensionFactory {
  return (pi) => {
    const provenanceLog: ProvenanceEntry[] = [];
    const turnRecords: TurnRecord[] = [];
    let cumulativeProvenanceTokens = 0;

    // ── agent_start ──────────────────────────────────────────────────────────
    pi.on("agent_start", () => {
      const model = sharedState.model;
      emitJsonl({ event: "session_start", model, ts: new Date().toISOString() });

      if (observabilityConfig?.otlp?.endpoint) {
        process.stderr.write(
          `[aw-harness] ℹ OTel tracing to ${observabilityConfig.otlp.endpoint}\n`,
        );
        // OTel span creation would go here when an OTLP client is wired in
      }

      // Record context provenance for imports
      for (const entry of imports) {
        const tokens = estimateTokens(entry.content);
        cumulativeProvenanceTokens += tokens;
        const record: ProvenanceEntry = {
          timestamp: new Date().toISOString(),
          source: "import",
          path: entry.path,
          tokens,
          cumulativeTokens: cumulativeProvenanceTokens,
          role: "user",
        };
        provenanceLog.push(record);
      }

      // Record context provenance for the prompt body
      const promptTokens = estimateTokens(promptBody);
      cumulativeProvenanceTokens += promptTokens;
      provenanceLog.push({
        timestamp: new Date().toISOString(),
        source: "prompt",
        tokens: promptTokens,
        cumulativeTokens: cumulativeProvenanceTokens,
        role: "user",
      });
    });

    // ── model_select ─────────────────────────────────────────────────────────
    pi.on("model_select", (event) => {
      const modelId = (event as { model?: { id?: string } }).model?.id;
      if (modelId) {
        sharedState.model = modelId;
      }
    });

    // ── turn_end ─────────────────────────────────────────────────────────────
    pi.on("turn_end", (event) => {
      const msg = event.message as {
        role?: string;
        usage?: { input?: number; output?: number; totalTokens?: number; cost?: { total?: number } };
      };

      let inputTokens = 0;
      let outputTokens = 0;
      let turnCostUsd = 0;

      if (msg.role === "assistant" && msg.usage) {
        inputTokens = msg.usage.input ?? 0;
        outputTokens = msg.usage.output ?? 0;
        turnCostUsd = msg.usage.cost?.total ?? 0;
      }

      const turnRecord: TurnRecord = {
        turn: sharedState.turnCount,
        inputTokens,
        outputTokens,
        cumulativeTokens: sharedState.cumulativeTokens,
        estimatedCostUsd: sharedState.cumulativeCostUsd,
      };
      turnRecords.push(turnRecord);

      // Structured JSONL event
      emitJsonl({
        event: "turn_end",
        turn: sharedState.turnCount,
        input_tokens: inputTokens,
        output_tokens: outputTokens,
        cumulative_tokens: sharedState.cumulativeTokens,
        cumulative_cost_usd: sharedState.cumulativeCostUsd,
        model: sharedState.model,
        ts: new Date().toISOString(),
      });

      // Human-readable per-turn line (markdown blockquote format per spec §8.5.4)
      process.stderr.write(
        `> **Turn ${sharedState.turnCount}**: ${inputTokens.toLocaleString()} in / ` +
          `${outputTokens.toLocaleString()} out` +
          ` | cumulative ${sharedState.cumulativeTokens.toLocaleString()} tokens` +
          ` ($${sharedState.cumulativeCostUsd.toFixed(4)})\n`,
      );

      void turnCostUsd; // used indirectly via sharedState
    });

    // ── tool_execution_end ───────────────────────────────────────────────────
    pi.on("tool_execution_end", (event) => {
      const e = event as { toolName?: string; duration?: number };
      emitJsonl({
        event: "tool_end",
        tool: e.toolName ?? "unknown",
        duration_ms: e.duration,
        ts: new Date().toISOString(),
      });
    });

    // ── agent_end ────────────────────────────────────────────────────────────
    pi.on("agent_end", () => {
      const elapsedMs = sharedState.sessionStartMs > 0 ? Date.now() - sharedState.sessionStartMs : 0;

      emitJsonl({
        event: "session_end",
        tokens: sharedState.cumulativeTokens,
        cost_usd: sharedState.cumulativeCostUsd,
        elapsed_ms: elapsedMs,
        ts: new Date().toISOString(),
      });

      // Write context provenance file
      writeProvenanceFile(provenanceLog);

      // Write GitHub Actions step summary
      const stepSummaryPath = process.env["GITHUB_STEP_SUMMARY"];
      if (stepSummaryPath) {
        writeStepSummary(stepSummaryPath, sharedState, turnRecords, provenanceLog, elapsedMs);
      }
    });
  };
}

// ─── writeProvenanceFile ─────────────────────────────────────────────────────

function writeProvenanceFile(log: ProvenanceEntry[]): void {
  try {
    mkdirSync(dirname(CONTEXT_PROVENANCE_PATH), { recursive: true });
    const lines = log.map((e) => JSON.stringify(e)).join("\n") + (log.length > 0 ? "\n" : "");
    writeFileSync(CONTEXT_PROVENANCE_PATH, lines, "utf8");
  } catch (err) {
    process.stderr.write(
      `[aw-harness] ⚠️ Failed to write context provenance: ${(err as Error).message}\n`,
    );
  }
}

// ─── writeStepSummary ────────────────────────────────────────────────────────

function writeStepSummary(
  path: string,
  state: SharedHarnessState,
  turns: TurnRecord[],
  provenance: ProvenanceEntry[],
  elapsedMs: number,
): void {
  try {
    const elapsedSec = (elapsedMs / 1000).toFixed(1);
    const lines: string[] = [
      `## AW Harness Run — \`${state.model}\``,
      "",
      "> ⚠️ **EXPERIMENTAL** — engine: aw is experimental and subject to change.",
      "",
      "### Token Consumption",
      "",
      "| Turn | Input Tokens | Output Tokens | Cumulative | Est. Cost |",
      "|------|-------------|---------------|------------|-----------|",
    ];

    for (const t of turns) {
      lines.push(
        `| ${t.turn} | ${t.inputTokens.toLocaleString()} | ${t.outputTokens.toLocaleString()} | ${t.cumulativeTokens.toLocaleString()} | $${t.estimatedCostUsd.toFixed(4)} |`,
      );
    }

    lines.push(
      `| **Total** | | | **${state.cumulativeTokens.toLocaleString()}** | **$${state.cumulativeCostUsd.toFixed(4)}** |`,
      "",
      `**Elapsed:** ${elapsedSec}s`,
      "",
      "### Context Provenance",
      "",
      "| Source | Path | Tokens |",
      "|--------|------|--------|",
    );

    for (const p of provenance) {
      const src = p.source === "import" ? "import" : "prompt";
      const pathCol = p.path ? p.path : "_(prompt.txt)_";
      lines.push(`| ${src} | ${pathCol} | ${p.tokens.toLocaleString()} |`);
    }

    appendFileSync(path, lines.join("\n") + "\n", "utf8");
  } catch (err) {
    process.stderr.write(
      `[aw-harness] ⚠️ Failed to write step summary: ${(err as Error).message}\n`,
    );
  }
}
