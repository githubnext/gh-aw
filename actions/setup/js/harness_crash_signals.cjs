// @ts-check

"use strict";

// Exit codes (128 + signal number) that indicate an agentic CLI subprocess was
// killed by a fatal OS-level signal rather than exiting normally or being
// cancelled.  These signify a sandbox/runtime-level crash (e.g. a bad syscall
// trapped by seccomp, a segfault, or an illegal instruction) rather than an
// application-level error, so resuming the same on-disk session with
// --continue risks immediately reproducing the same crash.
// Map: SIGILL=4, SIGABRT=6, SIGBUS=7, SIGFPE=8, SIGSEGV=11, SIGSYS=31.
const CRASH_SIGNAL_EXIT_CODES = new Map([
  [132, "SIGILL"],
  [134, "SIGABRT"],
  [135, "SIGBUS"],
  [136, "SIGFPE"],
  [139, "SIGSEGV"],
  [159, "SIGSYS"],
]);

/**
 * Determines whether the exit code corresponds to a fatal-signal crash of a CLI
 * subprocess (e.g. SIGSEGV=139, SIGSYS=159) as opposed to a normal application
 * error or an expected timeout/cancellation signal (e.g. SIGKILL=137/SIGTERM=143).
 * @param {number} exitCode
 * @returns {boolean}
 */
function isCrashSignalExitCode(exitCode) {
  return CRASH_SIGNAL_EXIT_CODES.has(exitCode);
}

/**
 * Best-effort mapping of a fatal-signal exit code (128 + signal number) to its
 * signal name, for diagnostic logging. Returns null when the exit code is not a
 * recognized crash signal.
 * @param {number} exitCode
 * @returns {string | null}
 */
function crashSignalNameForExitCode(exitCode) {
  return CRASH_SIGNAL_EXIT_CODES.get(exitCode) ?? null;
}

module.exports = {
  CRASH_SIGNAL_EXIT_CODES,
  isCrashSignalExitCode,
  crashSignalNameForExitCode,
};
