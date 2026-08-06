// @ts-check
/// <reference types="@actions/github-script" />

const safeOutputHandlerManager = require("./safe_output_handler_manager.cjs");
const { captureProcessOutput, logCapturedProcessOutput } = require("./action_log.cjs");

/**
 * Temporarily intercepts core.setFailed so that a failure signaled while
 * process output is being captured can be replayed for real once capture is
 * restored, instead of being buffered and later replayed inertly inside the
 * stop-commands-guarded output group (where it would never be an
 * interpretable/live annotation).
 *
 * This only defers the *call* to core.setFailed -- it never inspects or
 * rewrites raw captured stdout/stderr bytes, so the protected transcript
 * remains unchanged. Replaying is safe regardless of message content because
 * @actions/core escapes `%`, `\r` and `\n` in the message before writing the
 * workflow command, so it can never be used to fabricate additional log
 * lines or commands.
 *
 * @returns {{restore: () => void, replay: () => void}}
 */
function deferFinalFailureAnnotation() {
  const coreRef = /** @type {any} */ global.core;
  const originalSetFailed = coreRef && coreRef.setFailed;
  /** @type {any[]} */
  const deferredMessages = [];
  let restored = false;

  if (typeof originalSetFailed === "function") {
    coreRef.setFailed = (/** @type {any} */ message) => {
      deferredMessages.push(message);
    };
  }

  return {
    restore() {
      if (restored) return;
      restored = true;
      if (typeof originalSetFailed === "function") {
        coreRef.setFailed = originalSetFailed;
      }
    },
    replay() {
      if (typeof originalSetFailed !== "function") return;
      for (const message of deferredMessages) {
        originalSetFailed.call(coreRef, message);
      }
    },
  };
}

/**
 * Run safe-output handlers.
 *
 * Process stdout/stderr logs are captured in memory and sent directly to the
 * Actions log. They are never written to disk or packaged into artifacts.
 */
async function main() {
  const capture = captureProcessOutput();
  const failureAnnotation = deferFinalFailureAnnotation();
  try {
    await safeOutputHandlerManager.main();
  } finally {
    capture.restore();
    failureAnnotation.restore();
    try {
      logCapturedProcessOutput("Safe output processing logs", capture);
    } finally {
      failureAnnotation.replay();
    }
  }
}

module.exports = { main };
