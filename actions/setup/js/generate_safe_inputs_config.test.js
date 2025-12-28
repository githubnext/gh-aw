import { describe, it, expect, vi, beforeEach } from "vitest";
import crypto from "crypto";

describe("generateSafeInputsConfig", () => {
  let mockCore;
  let generateSafeInputsConfig;

  beforeEach(async () => {
    // Reset module before each test
    vi.resetModules();

    // Create mock core
    mockCore = {
      setOutput: vi.fn(),
      info: vi.fn(),
    };

    // Import module
    const module = await import("./generate_safe_inputs_config.cjs");
    generateSafeInputsConfig = module.generateSafeInputsConfig;
  });

  it("should generate API key and port", () => {
    const result = generateSafeInputsConfig({ core: mockCore, crypto });

    // Verify API key was generated
    expect(result.apiKey).toBeDefined();
    expect(typeof result.apiKey).toBe("string");
    expect(result.apiKey.length).toBeGreaterThan(0);

    // Verify API key doesn't contain special characters
    expect(result.apiKey).not.toMatch(/[/+=]/);

    // Verify port is 3000
    expect(result.port).toBe(3000);

    // Verify outputs were set
    expect(mockCore.setOutput).toHaveBeenCalledWith("safe_inputs_api_key", result.apiKey);
    expect(mockCore.setOutput).toHaveBeenCalledWith("safe_inputs_port", "3000");

    // Verify info message was logged
    expect(mockCore.info).toHaveBeenCalledWith("Safe Inputs MCP server will run on port 3000");
  });

  it("should generate different API keys on each call", () => {
    const result1 = generateSafeInputsConfig({ core: mockCore, crypto });
    const result2 = generateSafeInputsConfig({ core: mockCore, crypto });

    expect(result1.apiKey).not.toBe(result2.apiKey);
  });

  it("should generate API keys with sufficient length", () => {
    const result = generateSafeInputsConfig({ core: mockCore, crypto });

    // 45 bytes of random data, base64 encoded without special chars
    // should give us at least 40 characters
    expect(result.apiKey.length).toBeGreaterThanOrEqual(40);
  });

  it("should return consistent port", () => {
    const result1 = generateSafeInputsConfig({ core: mockCore, crypto });
    const result2 = generateSafeInputsConfig({ core: mockCore, crypto });

    expect(result1.port).toBe(result2.port);
    expect(result1.port).toBe(3000);
  });

  it("should handle core.setOutput being called correctly", () => {
    generateSafeInputsConfig({ core: mockCore, crypto });

    expect(mockCore.setOutput).toHaveBeenCalledTimes(2);
    expect(mockCore.setOutput.mock.calls[0][0]).toBe("safe_inputs_api_key");
    expect(mockCore.setOutput.mock.calls[1][0]).toBe("safe_inputs_port");
    expect(mockCore.setOutput.mock.calls[1][1]).toBe("3000");
  });
});
