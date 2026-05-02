/**
 * Extension 3: Steering (Resource Pressure)
 *
 * Monitors elapsed time and injects user steering messages via Pi's sendUserMessage
 * when the session is approaching the workflow timeout.
 *
 * Per spec §8.3.
 */

import type { ExtensionFactory } from "@mariozechner/pi-coding-agent";
import type { SharedHarnessState, SteeringConfig } from "../types.js";

// ─── createSteeringExtension ─────────────────────────────────────────────────

/**
 * Create the steering Pi extension factory.
 *
 * @param timeoutMinutes - Total workflow timeout in minutes
 * @param steering - Steering thresholds (time-warning and time-critical minutes)
 * @param sharedState - Mutable runtime state shared with the entry point
 */
export function createSteeringExtension(
  timeoutMinutes: number,
  steering: SteeringConfig,
  sharedState: SharedHarnessState,
): ExtensionFactory {
  return (pi) => {
    let timeWarnSent = false;
    let timeCriticalSent = false;

    pi.on("agent_start", () => {
      sharedState.sessionStartMs = Date.now();
    });

    pi.on("turn_end", (_event, _ctx) => {
      if (sharedState.sessionStartMs === 0) {
        return;
      }

      const elapsedMs = Date.now() - sharedState.sessionStartMs;
      const elapsedMinutes = elapsedMs / 60_000;
      const remainingMinutes = timeoutMinutes - elapsedMinutes;

      if (!timeCriticalSent && remainingMinutes <= steering.timeCriticalMinutes) {
        timeCriticalSent = true;
        process.stderr.write(
          `[aw-harness] 🚨 Time critical: ${remainingMinutes.toFixed(1)} min remaining of ${timeoutMinutes}.\n`,
        );
        pi.sendUserMessage(
          `🚨 CRITICAL: Only ${remainingMinutes.toFixed(0)} minute(s) remaining before timeout. ` +
            `Write your final output NOW and stop.`,
          { deliverAs: "steer" },
        );
        return;
      }

      if (!timeWarnSent && remainingMinutes <= steering.timeWarningMinutes) {
        timeWarnSent = true;
        process.stderr.write(
          `[aw-harness] ⚠️ Time warning: ${remainingMinutes.toFixed(1)} min remaining of ${timeoutMinutes}.\n`,
        );
        pi.sendUserMessage(
          `⚠️ ${remainingMinutes.toFixed(0)} minute(s) remaining. Begin wrapping up your task.`,
          { deliverAs: "steer" },
        );
      }
    });
  };
}
