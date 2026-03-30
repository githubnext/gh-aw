// @ts-check
import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";

describe("check_version_updates", () => {
  let mockCore;
  let mockFetch;
  let checkVersionUpdates;

  beforeEach(async () => {
    vi.useFakeTimers();

    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      debug: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    global.core = mockCore;

    mockFetch = vi.fn();
    vi.stubGlobal("fetch", mockFetch);

    delete process.env.GH_AW_COMPILED_VERSION;

    vi.resetModules();

    checkVersionUpdates = await import("./check_version_updates.cjs");
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.clearAllMocks();
  });

  /**
   * Helper to mock a successful fetch response with the given body.
   * @param {string} body
   */
  function mockFetchSuccess(body) {
    mockFetch.mockResolvedValue({
      ok: true,
      status: 200,
      text: () => Promise.resolve(body),
    });
  }

  /**
   * Helper to mock a failed fetch request (network error).
   * @param {Error} err
   */
  function mockFetchError(err) {
    mockFetch.mockRejectedValue(err);
  }

  /**
   * Run main() and advance all pending timers to process retry delays.
   * @returns {Promise<void>}
   */
  async function runMain() {
    const promise = checkVersionUpdates.main();
    await vi.runAllTimersAsync();
    return promise;
  }

  it("should skip check when version is 'dev'", async () => {
    process.env.GH_AW_COMPILED_VERSION = "dev";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("dev"));
  });

  it("should skip check when version is empty", async () => {
    process.env.GH_AW_COMPILED_VERSION = "";
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).not.toHaveBeenCalled();
  });

  it("should skip check when all fetch attempts fail", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchError(new Error("ECONNREFUSED"));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  it("should retry and succeed when fetch fails transiently", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0";
    // First call fails transiently; second call succeeds
    mockFetch.mockRejectedValueOnce(new Error("ECONNRESET")).mockResolvedValueOnce({
      ok: true,
      status: 200,
      text: () => Promise.resolve(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" })),
    });
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockFetch).toHaveBeenCalledTimes(2);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should skip check when server returns non-200 status", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetch.mockResolvedValue({
      ok: false,
      status: 404,
      text: () => Promise.resolve(""),
    });
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  it("should pass when version is not blocked and meets minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v0.9.0"], minimumVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should fail when version is in blocked list", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.1.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.1.0", "v1.2.0"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("v1.1.0"));
  });

  it("should fail when version is below minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.8.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Outdated compile-agentic version"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("v0.8.0"));
  });

  it("should pass when version exactly equals minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should skip minimum check when minimumVersion is empty", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.1.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should handle config with no blockedVersions field", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ minimumVersion: "v0.5.0" }));
    await runMain();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should block version when config uses version without 'v' prefix", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["1.0.0"], minimumVersion: "" }));
    // "v1.0.0" should be blocked by "1.0.0" after normalization
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
  });

  it("should block version regardless of 'v' prefix in compiled version", async () => {
    process.env.GH_AW_COMPILED_VERSION = "1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.0.0"], minimumVersion: "" }));
    // "1.0.0" should be blocked by "v1.0.0" after normalization
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
  });

  it("should fail when version is blocked with exact string match", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockFetchSuccess(JSON.stringify({ blockedVersions: ["v1.0.0"], minimumVersion: "" }));
    await runMain();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
  });
});
