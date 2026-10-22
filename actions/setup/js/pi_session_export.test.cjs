import { afterEach, describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import childProcess from "node:child_process";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { sessionIDFromStream } = require("./pi_session_export.cjs");

describe("Pi session export", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
    delete require.cache[require.resolve("./pi_session_export.cjs")];
  });

  it("finds the real session header in mixed diagnostics and JSONL", () => {
    expect(sessionIDFromStream('diagnostic\n{"type":"session","id":"session-1"}\n{"type":"agent_settled"}')).toBe("session-1");
  });
  it("reports missing session identity", () => {
    expect(() => sessionIDFromStream('{"type":"agent_settled"}')).toThrow("session header");
  });

  it.each([undefined, "/runner/temp/gh-aw/sandbox/agent/logs/pi-streaming.jsonl"])("exports from the compiler-selected streaming log (%s)", streamFile => {
    vi.stubEnv("GH_AW_PI_CONFIG", '{"session":{"enabled":true,"export":true}}');
    const execute = vi.spyOn(childProcess, "execFileSync").mockReturnValue("");
    delete require.cache[require.resolve("./pi_session_export.cjs")];
    const { main } = require("./pi_session_export.cjs");
    const read = vi.spyOn(fs, "readFileSync").mockReturnValue('{"type":"session","id":"session-1"}\n');
    vi.spyOn(fs, "readdirSync").mockReturnValue(["timestamp_session-1.jsonl"]);
    main(streamFile);
    expect(read).toHaveBeenCalledWith(streamFile || "/tmp/gh-aw/sandbox/agent/logs/pi-streaming.jsonl", "utf8");
    expect(execute).toHaveBeenCalledWith(expect.any(String), ["--export", "/tmp/gh-aw/agent/pi-sessions/timestamp_session-1.jsonl", expect.stringContaining("/session.html")], expect.any(Object));
  });
});
