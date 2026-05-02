/**
 * Extension 4: Session Repair
 *
 * Detects broken tool calls and repairs the session by emitting a follow-up
 * message summarising progress so the agent can continue from where it left off.
 *
 * Currently implements the agent_end recovery path: if the session ends with a
 * recoverable error, inject a follow-up message so Pi can resume.
 *
 * Per spec §8.4.
 */

import type { ExtensionFactory } from "@mariozechner/pi-coding-agent";

// ─── Recoverable error patterns ──────────────────────────────────────────────

/** Patterns that indicate a transient / recoverable error. */
const RECOVERABLE_PATTERNS: RegExp[] = [
  /rate.?limit/i,
  /overloaded/i,
  /timeout/i,
  /network/i,
  /ECONNRESET/,
  /ETIMEDOUT/,
];

/** Patterns that indicate a corrupted tool result payload. */
const CORRUPTED_TOOL_PATTERNS: RegExp[] = [
  /null.*tool_call/i,
  /invalid type.*tool_calls/i,
  /tool_calls\[\d+\]\.type.*null/,
];

// ─── Helpers ─────────────────────────────────────────────────────────────────

function isRecoverableError(error: Error): boolean {
  return RECOVERABLE_PATTERNS.some((p) => p.test(error.message));
}

function isCorruptedToolContent(content: unknown): boolean {
  const str = typeof content === "string" ? content : JSON.stringify(content ?? "");
  return CORRUPTED_TOOL_PATTERNS.some((p) => p.test(str));
}

// ─── createRepairExtension ───────────────────────────────────────────────────

/**
 * Create the repair Pi extension factory.
 */
export function createRepairExtension(): ExtensionFactory {
  return (pi) => {
    // Detect corrupted tool results and log them for diagnostic purposes
    pi.on("tool_result", (event) => {
      const content = (event as { content?: unknown }).content;
      if (isCorruptedToolContent(content)) {
        process.stderr.write(
          `[aw-harness] ⚠️ Corrupted tool result detected for tool '` +
            `${(event as { toolName?: string }).toolName ?? "unknown"}'. ` +
            `The agent may produce degraded output for this turn.\n`,
        );
        // Emit a JSONL repair event for observability
        const record = {
          event: "repair",
          action: "corrupted_tool_result_detected",
          toolName: (event as { toolName?: string }).toolName ?? "unknown",
          ts: new Date().toISOString(),
        };
        process.stderr.write(JSON.stringify(record) + "\n");
      }
    });

    // On agent_end with a recoverable error, inject a follow-up so the agent can resume
    pi.on("agent_end", (event, _ctx) => {
      const endEvent = event as { error?: Error };
      if (!endEvent.error) {
        return;
      }

      if (isRecoverableError(endEvent.error)) {
        process.stderr.write(
          `[aw-harness] ⚠️ Recoverable error on agent_end: ${endEvent.error.message}. ` +
            `Injecting follow-up for continuation.\n`,
        );
        const record = {
          event: "repair",
          action: "follow_up_after_recoverable_error",
          error: endEvent.error.message,
          ts: new Date().toISOString(),
        };
        process.stderr.write(JSON.stringify(record) + "\n");

        pi.sendUserMessage(
          `The previous turn ended with a transient error: "${endEvent.error.message}". ` +
            `Please review your progress so far and continue from where you left off.`,
          { deliverAs: "followUp" },
        );
      } else {
        process.stderr.write(
          `[aw-harness] ✗ Non-recoverable error on agent_end: ${endEvent.error.message}\n`,
        );
      }
    });
  };
}
