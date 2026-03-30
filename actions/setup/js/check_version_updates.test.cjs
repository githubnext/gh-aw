// @ts-check
import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import https from "https";

// Mock the https module to avoid real network requests
vi.mock("https", () => {
  const mockGet = vi.fn();
  return { default: { get: mockGet }, get: mockGet };
});

describe("check_version_updates", () => {
  let mockCore;
  let mockGet;
  let checkVersionUpdates;

  beforeEach(async () => {
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setOutput: vi.fn(),
      setFailed: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    global.core = mockCore;

    delete process.env.GH_AW_COMPILED_VERSION;

    vi.resetModules();
    mockGet = vi.mocked(https.get);

    checkVersionUpdates = await import("./check_version_updates.cjs");
  });

  afterEach(() => {
    vi.clearAllMocks();
  });

  /**
   * Helper to mock a successful HTTPS response with the given body.
   * @param {string} body
   */
  function mockHttpsSuccess(body) {
    mockGet.mockImplementation((url, callback) => {
      const mockRes = {
        statusCode: 200,
        on: vi.fn((event, handler) => {
          if (event === "data") handler(Buffer.from(body));
          if (event === "end") handler();
          return mockRes;
        }),
        resume: vi.fn(),
      };
      callback(mockRes);
      return {
        on: vi.fn().mockReturnThis(),
        setTimeout: vi.fn().mockReturnThis(),
        destroy: vi.fn(),
      };
    });
  }

  /**
   * Helper to mock a failed HTTPS request.
   * @param {Error} err
   */
  function mockHttpsError(err) {
    mockGet.mockImplementation((url, callback) => {
      const req = {
        on: vi.fn((event, handler) => {
          if (event === "error") handler(err);
          return req;
        }),
        setTimeout: vi.fn().mockReturnThis(),
        destroy: vi.fn(),
      };
      return req;
    });
  }

  it("should skip check when version is 'dev'", async () => {
    process.env.GH_AW_COMPILED_VERSION = "dev";
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockGet).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("dev"));
  });

  it("should skip check when version is empty", async () => {
    process.env.GH_AW_COMPILED_VERSION = "";
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockGet).not.toHaveBeenCalled();
  });

  it("should skip check when download fails", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockHttpsError(new Error("ECONNREFUSED"));
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  it("should skip check when server returns non-200 status", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockGet.mockImplementation((url, callback) => {
      const mockRes = {
        statusCode: 404,
        on: vi.fn().mockReturnThis(),
        resume: vi.fn(),
      };
      callback(mockRes);
      return {
        on: vi.fn().mockReturnThis(),
        setTimeout: vi.fn().mockReturnThis(),
        destroy: vi.fn(),
      };
    });
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Could not fetch update configuration"));
  });

  it("should pass when version is not blocked and meets minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.2.0";
    mockHttpsSuccess(JSON.stringify({ blockedVersions: ["v0.9.0"], minimumVersion: "v1.0.0" }));
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Version check passed"));
  });

  it("should fail when version is in blocked list", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.1.0";
    mockHttpsSuccess(JSON.stringify({ blockedVersions: ["v1.1.0", "v1.2.0"], minimumVersion: "" }));
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("v1.1.0"));
  });

  it("should fail when version is below minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.8.0";
    mockHttpsSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" }));
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Outdated compile-agentic version"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("v0.8.0"));
  });

  it("should pass when version exactly equals minimum", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockHttpsSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "v1.0.0" }));
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should skip minimum check when minimumVersion is empty", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v0.1.0";
    mockHttpsSuccess(JSON.stringify({ blockedVersions: [], minimumVersion: "" }));
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should handle config with no blockedVersions field", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockHttpsSuccess(JSON.stringify({ minimumVersion: "v0.5.0" }));
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should block version when config uses version without 'v' prefix", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockHttpsSuccess(JSON.stringify({ blockedVersions: ["1.0.0"], minimumVersion: "" }));
    // "v1.0.0" should be blocked by "1.0.0" after normalization
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
  });

  it("should block version regardless of 'v' prefix in compiled version", async () => {
    process.env.GH_AW_COMPILED_VERSION = "1.0.0";
    mockHttpsSuccess(JSON.stringify({ blockedVersions: ["v1.0.0"], minimumVersion: "" }));
    // "1.0.0" should be blocked by "v1.0.0" after normalization
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
  });

  it("should fail when version is blocked with exact string match", async () => {
    process.env.GH_AW_COMPILED_VERSION = "v1.0.0";
    mockHttpsSuccess(JSON.stringify({ blockedVersions: ["v1.0.0"], minimumVersion: "" }));
    await checkVersionUpdates.main();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Blocked compile-agentic version"));
  });
});
