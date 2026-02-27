// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { loadConfig } from "./safe_outputs_config.cjs";

describe("safe_outputs_config", () => {
  /** @type {{ debug: import("vitest").Mock }} */
  let mockServer;
  /** @type {string} */
  let testConfigPath;
  /** @type {string} */
  let testOutputPath;

  beforeEach(() => {
    // Create a mock server with debug function
    mockServer = {
      debug: vi.fn(),
    };

    // Use unique paths for each test
    const testId = Math.random().toString(36).substring(7);
    testConfigPath = `/tmp/test-safe-outputs-config-${testId}/config.json`;
    testOutputPath = `/tmp/test-safe-outputs-config-${testId}/outputs.jsonl`;

    // Set environment variables for test
    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = testConfigPath;
    process.env.GH_AW_SAFE_OUTPUTS = testOutputPath;
  });

  afterEach(() => {
    // Clean up test files
    try {
      if (fs.existsSync(testConfigPath)) {
        fs.unlinkSync(testConfigPath);
      }
      const testDir = path.dirname(testConfigPath);
      if (fs.existsSync(testDir)) {
        fs.rmSync(testDir, { recursive: true, force: true });
      }
    } catch (error) {
      // Ignore cleanup errors
    }

    // Clear environment variables
    delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH;
    delete process.env.GH_AW_SAFE_OUTPUTS;
  });

  describe("loadConfig", () => {
    it("should load and parse valid config file", () => {
      // Create config directory and file
      const configDir = path.dirname(testConfigPath);
      fs.mkdirSync(configDir, { recursive: true });

      const config = {
        "create-pull-request": true,
        "upload-assets": { maxSize: 1024 },
      };
      fs.writeFileSync(testConfigPath, JSON.stringify(config));

      /** @type {import("./safe_outputs_config.cjs").LoadConfigResult} */
      const result = loadConfig(mockServer);

      expect(result.config).toEqual({
        create_pull_request: true,
        upload_assets: { maxSize: 1024 },
      });
      expect(result.outputFile).toBe(testOutputPath);
      expect(mockServer.debug).toHaveBeenCalled();
    });

    it("should handle missing config file", () => {
      /** @type {import("./safe_outputs_config.cjs").LoadConfigResult} */
      const result = loadConfig(mockServer);

      expect(result.config).toEqual({});
      expect(result.outputFile).toBe(testOutputPath);
      expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("does not exist"));
    });

    it("should handle invalid JSON in config file", () => {
      // Create config directory and file with invalid JSON
      const configDir = path.dirname(testConfigPath);
      fs.mkdirSync(configDir, { recursive: true });
      fs.writeFileSync(testConfigPath, "{ invalid json }");

      /** @type {import("./safe_outputs_config.cjs").LoadConfigResult} */
      const result = loadConfig(mockServer);

      expect(result.config).toEqual({});
      expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Error reading config file"));
    });

    it("should normalize dashes to underscores in config keys", () => {
      const configDir = path.dirname(testConfigPath);
      fs.mkdirSync(configDir, { recursive: true });

      const config = {
        "create-pull-request": true,
        "push-to-pull-request-branch": true,
        "upload-assets": true,
      };
      fs.writeFileSync(testConfigPath, JSON.stringify(config));

      /** @type {import("./safe_outputs_config.cjs").LoadConfigResult} */
      const result = loadConfig(mockServer);

      expect(result.config).toEqual({
        create_pull_request: true,
        push_to_pull_request_branch: true,
        upload_assets: true,
      });
    });

    it("should use default output path when env var not set", () => {
      delete process.env.GH_AW_SAFE_OUTPUTS;

      const configDir = path.dirname(testConfigPath);
      fs.mkdirSync(configDir, { recursive: true });
      fs.writeFileSync(testConfigPath, JSON.stringify({}));

      // Mock fs.mkdirSync to avoid permission error on /opt/gh-aw/safeoutputs
      const mkdirSyncSpy = vi.spyOn(fs, "mkdirSync").mockImplementation(() => undefined);
      const existsSyncSpy = vi.spyOn(fs, "existsSync");
      const originalExistsSync = existsSyncSpy.getMockImplementation() || fs.existsSync.bind(fs);
      existsSyncSpy.mockImplementation(p => {
        // Pretend /opt/gh-aw/safeoutputs exists to skip mkdir
        if (String(p).startsWith("/opt/gh-aw/safeoutputs")) return true;
        return originalExistsSync(p);
      });

      try {
        /** @type {import("./safe_outputs_config.cjs").LoadConfigResult} */
        const result = loadConfig(mockServer);

        expect(result.outputFile).toBe("/opt/gh-aw/safeoutputs/outputs.jsonl");
        expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("GH_AW_SAFE_OUTPUTS not set"));
      } finally {
        mkdirSyncSpy.mockRestore();
        existsSyncSpy.mockRestore();
      }
    });

    it("should create output directory if it doesn't exist", () => {
      const customOutputPath = `/tmp/test-safe-outputs-config-${Date.now()}/custom/path/outputs.jsonl`;
      process.env.GH_AW_SAFE_OUTPUTS = customOutputPath;

      const configDir = path.dirname(testConfigPath);
      fs.mkdirSync(configDir, { recursive: true });
      fs.writeFileSync(testConfigPath, JSON.stringify({}));

      const outputDir = path.dirname(customOutputPath);
      expect(fs.existsSync(outputDir)).toBe(false);

      loadConfig(mockServer);

      expect(fs.existsSync(outputDir)).toBe(true);

      // Clean up
      fs.rmSync(outputDir, { recursive: true, force: true });
    });

    it("should handle empty config file", () => {
      const configDir = path.dirname(testConfigPath);
      fs.mkdirSync(configDir, { recursive: true });
      fs.writeFileSync(testConfigPath, JSON.stringify({}));

      /** @type {import("./safe_outputs_config.cjs").LoadConfigResult} */
      const result = loadConfig(mockServer);

      expect(result.config).toEqual({});
      expect(result.outputFile).toBe(testOutputPath);
    });

    it("should log config file details during loading", () => {
      const configDir = path.dirname(testConfigPath);
      fs.mkdirSync(configDir, { recursive: true });

      const config = { "test-tool": true };
      fs.writeFileSync(testConfigPath, JSON.stringify(config));

      loadConfig(mockServer);

      expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Reading config from file"));
      expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Config file exists"));
      expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Successfully parsed config"));
      expect(mockServer.debug).toHaveBeenCalledWith(expect.stringContaining("Final processed config"));
    });
  });
});
