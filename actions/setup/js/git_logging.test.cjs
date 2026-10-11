import { afterEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { gitLoggingOptions } = require("./git_logging.cjs");
const originalCore = global.core;

afterEach(() => {
  global.core = originalCore;
});

describe("gitLoggingOptions", () => {
  it("keeps credential-query stdout silent without hiding failure diagnostics", () => {
    global.core = { info: vi.fn() };
    const options = gitLoggingOptions({}, false);
    options.listeners.stdout(Buffer.from("Authorization: basic credential"));
    options.listeners.stderr(Buffer.from("configuration read failed"));
    expect(global.core.info).toHaveBeenCalledExactlyOnceWith("configuration read failed");
  });

  it("logs complete stdout and stderr through core without writing automatic streams", async () => {
    global.core = { info: vi.fn() };
    const { getExecOutput } = await import("@actions/exec");
    const write = vi.fn();
    const result = await getExecOutput("git", ["-c", "test.output=full output without a newline", "config", "--get", "test.output"], gitLoggingOptions({ outStream: { write }, errStream: { write } }));
    expect(result.stdout).toBe("full output without a newline\n");
    expect(global.core.info.mock.calls.map(([text]) => text).join("")).toBe(result.stdout);
    expect(write).not.toHaveBeenCalled();
    global.core.info.mockClear();
    const failure = await getExecOutput("git", ["not-a-real-git-command"], gitLoggingOptions({ ignoreReturnCode: true, outStream: { write }, errStream: { write } }));
    expect(failure.exitCode).not.toBe(0);
    expect(global.core.info.mock.calls.map(([text]) => text).join("")).toBe(failure.stderr);
    expect(write).not.toHaveBeenCalled();
  });

  it("preserves options and listeners, including split UTF-8 and unterminated output", () => {
    global.core = { info: vi.fn() };
    const stdout = vi.fn();
    const stderr = vi.fn();
    const stdline = vi.fn();
    const options = gitLoggingOptions({ cwd: "/checkout", env: { TEST: "value" }, ignoreReturnCode: true, silent: false, listeners: { stdout, stderr, stdline } });
    expect(options).toMatchObject({ cwd: "/checkout", env: { TEST: "value" }, ignoreReturnCode: true, silent: true });
    expect(options.listeners.stdline).toBe(stdline);
    const text = Buffer.from("complete \u00e9 output");
    options.listeners.stdout(text.subarray(0, 10));
    options.listeners.stdout(text.subarray(10));
    options.listeners.stderr(Buffer.from("stderr without newline"));
    expect(global.core.info.mock.calls.map(([value]) => value).join("")).toBe("complete \u00e9 outputstderr without newline");
    expect(stdout).toHaveBeenCalledTimes(2);
    expect(stderr).toHaveBeenCalledWith(Buffer.from("stderr without newline"));
  });
});
