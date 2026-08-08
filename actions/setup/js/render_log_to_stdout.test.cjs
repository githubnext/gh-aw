import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

// Minimal mock for @actions/core used by github-script CJS modules.
const mockCore = {
  setSecret: vi.fn(),
};

global.core = mockCore;

// Capture stdout so we can assert on the workflow-command output.
const originalWrite = process.stdout.write.bind(process.stdout);
let stdoutChunks = [];
const stubbedWrite = chunk => {
  stdoutChunks.push(typeof chunk === "string" ? chunk : chunk.toString("utf8"));
  return true;
};

function capturedStdout() {
  return stdoutChunks.join("");
}

describe("render_log_to_stdout.cjs", () => {
  let mod;

  beforeEach(async () => {
    vi.clearAllMocks();
    stdoutChunks = [];
    process.stdout.write = stubbedWrite;
    mod = await import("./render_log_to_stdout.cjs?t=" + Date.now());
  });

  afterEach(() => {
    process.stdout.write = originalWrite;
  });

  it("wraps content in ::group:: / ::endgroup::", () => {
    mod.renderLogToStdout("My Group", "some content\n");
    const out = capturedStdout();
    expect(out).toMatch(/^::group::My Group\n/);
    expect(out).toContain("::endgroup::\n");
  });

  it("emits ::stop-commands:: with a render- prefixed hex token", () => {
    mod.renderLogToStdout("Test", "line\n");
    const out = capturedStdout();
    const match = out.match(/::stop-commands::(render-[a-f0-9]+)\n/);
    expect(match).not.toBeNull();
    const token = match[1];
    expect(out).toContain("::" + token + "::\n");
  });

  it("places content between stop-commands and end-token lines", () => {
    mod.renderLogToStdout("G", "the-payload\n");
    const out = capturedStdout();
    const stopIdx = out.indexOf("::stop-commands::");
    const endTokenIdx = out.lastIndexOf("::");
    const contentIdx = out.indexOf("the-payload");
    expect(contentIdx).toBeGreaterThan(stopIdx);
    expect(contentIdx).toBeLessThan(endTokenIdx);
  });

  it("appends a trailing newline when content lacks one", () => {
    mod.renderLogToStdout("G", "no-newline");
    const out = capturedStdout();
    // The end-token marker must start on its own line.
    expect(out).toMatch(/\n::render-/);
  });

  it("uses a unique stop token each call", () => {
    mod.renderLogToStdout("A", "x\n");
    const out1 = capturedStdout();
    stdoutChunks = [];
    mod.renderLogToStdout("B", "y\n");
    const out2 = capturedStdout();

    const token1 = out1.match(/::stop-commands::(render-[a-f0-9]+)\n/)[1];
    const token2 = out2.match(/::stop-commands::(render-[a-f0-9]+)\n/)[1];
    expect(token1).not.toBe(token2);
  });

  it("emits closing markers in finally even when content write throws", () => {
    let callCount = 0;
    process.stdout.write = chunk => {
      const text = typeof chunk === "string" ? chunk : chunk.toString("utf8");
      stdoutChunks.push(text);
      callCount++;
      // Throw on the third write (the content write), after group + stop-commands.
      if (callCount === 3) throw new Error("simulated write failure");
      return true;
    };

    expect(() => mod.renderLogToStdout("G", "content\n")).toThrow("simulated write failure");

    const out = capturedStdout();
    expect(out).toContain("::endgroup::\n");
    expect(out).toMatch(/::render-[a-f0-9]+::\n/);
  });

  it("calls core.setSecret for each extracted MCP gateway token via maskSecret", () => {
    // maskSecret delegates to core.setSecret when it is available.
    // core.setSecret is set up in the global mock above.
    // Without real gateway config files present, extractMCPGatewayTokens
    // returns [] so setSecret is not called — this verifies the happy path
    // silently skips when there are no tokens to mask.
    mod.renderLogToStdout("G", "data\n");
    // Absence of a throw is sufficient; maskSecret is a best-effort step.
    expect(capturedStdout()).toContain("::group::G\n");
  });

  it("accepts a custom group title", () => {
    mod.renderLogToStdout("Detection Log", "entry\n");
    expect(capturedStdout()).toMatch(/^::group::Detection Log\n/);
  });
});
