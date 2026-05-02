/**
 * Extension 2: Cost Tracker
 *
 * Monitors token usage via Pi's turn_end events and enforces budget gates.
 *
 * - When tokens reach budget-warn-percent, injects a steering message.
 * - When tokens reach budget-critical-percent, injects a critical message
 *   and sets sharedState.budgetAborted so the main entry point can exit(1).
 *
 * Token counts are read from the AssistantMessage usage field in the
 * TurnEndEvent.message when available, falling back to context usage percent.
 *
 * Per spec §8.2.
 */

import type { ExtensionFactory } from "@mariozechner/pi-coding-agent";
import type { BudgetConfig, SharedHarnessState, SteeringConfig } from "../types.js";

// ─── createCostTrackerExtension ──────────────────────────────────────────────

/**
 * Create the cost-tracker Pi extension factory.
 *
 * @param budget - Budget configuration (undefined = no limit)
 * @param steering - Steering thresholds (percent values)
 * @param sharedState - Mutable runtime state shared with the entry point
 */
export function createCostTrackerExtension(
  budget: BudgetConfig | undefined,
  steering: SteeringConfig,
  sharedState: SharedHarnessState,
): ExtensionFactory {
  return (pi) => {
    // Track whether we have already sent warn/critical messages to avoid spam
    let warnSent = false;
    let criticalSent = false;

    pi.on("turn_end", (event, ctx) => {
      sharedState.turnCount++;

      // Extract token usage from the assistant message if available
      const msg = event.message as {
        role?: string;
        usage?: { input?: number; output?: number; totalTokens?: number; cost?: { total?: number } };
      };

      if (msg.role === "assistant" && msg.usage) {
        const turnTokens = msg.usage.totalTokens ?? (msg.usage.input ?? 0) + (msg.usage.output ?? 0);
        const turnCost = msg.usage.cost?.total ?? 0;
        sharedState.cumulativeTokens += turnTokens;
        sharedState.cumulativeCostUsd += turnCost;
      } else {
        // Fallback: use context window percentage as a proxy for token usage
        const ctxUsage = ctx.getContextUsage();
        if (ctxUsage?.tokens != null) {
          sharedState.cumulativeTokens = ctxUsage.tokens;
        }
      }

      if (!budget) {
        return; // No budget configured — nothing to gate
      }

      const { maxEffectiveTokens } = budget;
      const pct = (sharedState.cumulativeTokens / maxEffectiveTokens) * 100;

      if (!criticalSent && pct >= steering.budgetCriticalPercent) {
        criticalSent = true;
        sharedState.budgetAborted = true;
        process.stderr.write(
          `[aw-harness] 🚨 Budget critical: ${pct.toFixed(1)}% used ` +
            `(${sharedState.cumulativeTokens}/${maxEffectiveTokens} tokens). Aborting.\n`,
        );
        pi.sendUserMessage(
          `🚨 CRITICAL: Token budget exhausted (${pct.toFixed(0)}% used). ` +
            `You have ${maxEffectiveTokens - sharedState.cumulativeTokens} tokens remaining. ` +
            `Produce your final output NOW and stop immediately.`,
          { deliverAs: "steer" },
        );
        return;
      }

      if (!warnSent && pct >= steering.budgetWarnPercent) {
        warnSent = true;
        process.stderr.write(
          `[aw-harness] ⚠️ Budget warning: ${pct.toFixed(1)}% used ` +
            `(${sharedState.cumulativeTokens}/${maxEffectiveTokens} tokens).\n`,
        );
        pi.sendUserMessage(
          `⚠️ Token budget: ${pct.toFixed(0)}% used (${sharedState.cumulativeTokens}/${maxEffectiveTokens}). ` +
            `Be concise and wrap up soon.`,
          { deliverAs: "steer" },
        );
      }
    });
  };
}
