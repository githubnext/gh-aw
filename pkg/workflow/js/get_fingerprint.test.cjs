import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock core
const mockCore = {
  info: vi.fn(),
};
global.core = mockCore;

describe("getFingerprint", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_FINGERPRINT;
  });

  it("should return empty string when fingerprint not set", async () => {
    const { getFingerprint } = await import("./get_fingerprint.cjs");

    const result = getFingerprint();

    expect(result).toBe("");
    expect(mockCore.info).not.toHaveBeenCalled();
  });

  it("should return fingerprint and log when set", async () => {
    process.env.GH_AW_FINGERPRINT = "test-fingerprint-123";
    const { getFingerprint } = await import("./get_fingerprint.cjs");

    const result = getFingerprint();

    expect(result).toBe("test-fingerprint-123");
    expect(mockCore.info).toHaveBeenCalledWith("Fingerprint: test-fingerprint-123");
  });

  it("should handle fingerprint with hyphens", async () => {
    process.env.GH_AW_FINGERPRINT = "project-alpha-2024";
    const { getFingerprint } = await import("./get_fingerprint.cjs");

    const result = getFingerprint();

    expect(result).toBe("project-alpha-2024");
    expect(mockCore.info).toHaveBeenCalledWith("Fingerprint: project-alpha-2024");
  });

  it("should handle fingerprint with underscores", async () => {
    process.env.GH_AW_FINGERPRINT = "project_alpha_2024";
    const { getFingerprint } = await import("./get_fingerprint.cjs");

    const result = getFingerprint();

    expect(result).toBe("project_alpha_2024");
    expect(mockCore.info).toHaveBeenCalledWith("Fingerprint: project_alpha_2024");
  });

  it("should handle mixed alphanumeric fingerprint", async () => {
    process.env.GH_AW_FINGERPRINT = "Test123_Project-v2";
    const { getFingerprint } = await import("./get_fingerprint.cjs");

    const result = getFingerprint();

    expect(result).toBe("Test123_Project-v2");
    expect(mockCore.info).toHaveBeenCalledWith("Fingerprint: Test123_Project-v2");
  });
});
