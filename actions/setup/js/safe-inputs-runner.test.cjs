import { describe, it, expect, vi, afterEach } from "vitest";

describe("safe-inputs-runner.cjs", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should call execute with parsed JSON inputs from stdin", async () => {
    const runSafeInput = (await import("./safe-inputs-runner.cjs")).default;

    const inputJson = JSON.stringify({ query: "hello" });
    const chunks = [inputJson];
    let chunkIdx = 0;

    const stdinHandlers = {};
    vi.spyOn(process.stdin, "setEncoding").mockImplementation(() => process.stdin);
    vi.spyOn(process.stdin, "on").mockImplementation((event, handler) => {
      stdinHandlers[event] = handler;
      return process.stdin;
    });

    const stdoutSpy = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    const execute = vi.fn().mockResolvedValue({ result: "ok" });
    runSafeInput(execute);

    // Simulate data then end
    stdinHandlers["data"](chunks[chunkIdx++]);
    await stdinHandlers["end"]();

    expect(execute).toHaveBeenCalledWith({ query: "hello" });
    expect(stdoutSpy).toHaveBeenCalledWith(JSON.stringify({ result: "ok" }));
  });

  it("should call execute with empty object when stdin is empty", async () => {
    const runSafeInput = (await import("./safe-inputs-runner.cjs")).default;

    const stdinHandlers = {};
    vi.spyOn(process.stdin, "setEncoding").mockImplementation(() => process.stdin);
    vi.spyOn(process.stdin, "on").mockImplementation((event, handler) => {
      stdinHandlers[event] = handler;
      return process.stdin;
    });

    const stdoutSpy = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    const execute = vi.fn().mockResolvedValue({ result: "empty" });
    runSafeInput(execute);

    await stdinHandlers["end"]();

    expect(execute).toHaveBeenCalledWith({});
    expect(stdoutSpy).toHaveBeenCalledWith(JSON.stringify({ result: "empty" }));
  });

  it("should write warning to stderr and continue when JSON parsing fails", async () => {
    const runSafeInput = (await import("./safe-inputs-runner.cjs")).default;

    const stdinHandlers = {};
    vi.spyOn(process.stdin, "setEncoding").mockImplementation(() => process.stdin);
    vi.spyOn(process.stdin, "on").mockImplementation((event, handler) => {
      stdinHandlers[event] = handler;
      return process.stdin;
    });

    const stderrSpy = vi.spyOn(process.stderr, "write").mockImplementation(() => true);
    const stdoutSpy = vi.spyOn(process.stdout, "write").mockImplementation(() => true);

    const execute = vi.fn().mockResolvedValue({ result: "fallback" });
    runSafeInput(execute);

    stdinHandlers["data"]("not-valid-json");
    await stdinHandlers["end"]();

    expect(stderrSpy).toHaveBeenCalledWith(expect.stringContaining("Warning: Failed to parse inputs:"));
    expect(execute).toHaveBeenCalledWith({});
    expect(stdoutSpy).toHaveBeenCalledWith(JSON.stringify({ result: "fallback" }));
  });

  it("should write error to stderr and exit(1) when execute throws", async () => {
    const runSafeInput = (await import("./safe-inputs-runner.cjs")).default;

    const stdinHandlers = {};
    vi.spyOn(process.stdin, "setEncoding").mockImplementation(() => process.stdin);
    vi.spyOn(process.stdin, "on").mockImplementation((event, handler) => {
      stdinHandlers[event] = handler;
      return process.stdin;
    });

    const stderrSpy = vi.spyOn(process.stderr, "write").mockImplementation(() => true);
    const exitSpy = vi.spyOn(process, "exit").mockImplementation(() => {
      throw new Error("process.exit called");
    });

    const execute = vi.fn().mockRejectedValue(new Error("tool failed"));
    runSafeInput(execute);

    await expect(stdinHandlers["end"]()).rejects.toThrow("process.exit called");

    expect(stderrSpy).toHaveBeenCalledWith("Error: tool failed");
    expect(exitSpy).toHaveBeenCalledWith(1);
  });
});
