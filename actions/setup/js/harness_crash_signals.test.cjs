// @ts-check

import { describe, expect, it } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { CRASH_SIGNAL_EXIT_CODES, isCrashSignalExitCode, crashSignalNameForExitCode } = require("./harness_crash_signals.cjs");

describe("harness_crash_signals.cjs", () => {
  it("classifies known fatal-signal exit codes as crashes", () => {
    for (const [exitCode, signalName] of CRASH_SIGNAL_EXIT_CODES) {
      expect(isCrashSignalExitCode(exitCode)).toBe(true);
      expect(crashSignalNameForExitCode(exitCode)).toBe(signalName);
    }
  });

  it("does not classify normal exit codes as crashes", () => {
    expect(isCrashSignalExitCode(0)).toBe(false);
    expect(isCrashSignalExitCode(1)).toBe(false);
    expect(isCrashSignalExitCode(2)).toBe(false);
    expect(crashSignalNameForExitCode(0)).toBeNull();
    expect(crashSignalNameForExitCode(1)).toBeNull();
  });

  it("does not classify expected timeout/cancellation signals (SIGKILL/SIGTERM) as crashes", () => {
    expect(isCrashSignalExitCode(137)).toBe(false); // SIGKILL
    expect(isCrashSignalExitCode(143)).toBe(false); // SIGTERM
    expect(crashSignalNameForExitCode(137)).toBeNull();
    expect(crashSignalNameForExitCode(143)).toBeNull();
  });

  it("maps exit codes to the expected signal names", () => {
    expect(crashSignalNameForExitCode(132)).toBe("SIGILL");
    expect(crashSignalNameForExitCode(134)).toBe("SIGABRT");
    expect(crashSignalNameForExitCode(135)).toBe("SIGBUS");
    expect(crashSignalNameForExitCode(136)).toBe("SIGFPE");
    expect(crashSignalNameForExitCode(139)).toBe("SIGSEGV");
    expect(crashSignalNameForExitCode(159)).toBe("SIGSYS");
  });
});
