import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

describe("safe_inputs_bootstrap.cjs", () => {
  let tempDir;

  beforeEach(() => {
    vi.resetModules();
    // Create a temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "safe-inputs-bootstrap-test-"));
  });

  afterEach(() => {
    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  });

  describe("bootstrapSafeInputsServer", () => {
    it("should load configuration and return bootstrap result", async () => {
      const { bootstrapSafeInputsServer } = await import("./safe_inputs_bootstrap.cjs");

      // Create a test configuration file
      const configPath = path.join(tempDir, "config.json");
      const config = {
        serverName: "test-server",
        version: "1.0.0",
        tools: [
          {
            name: "test_tool",
            description: "A test tool",
            inputSchema: { type: "object", properties: {} },
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      // Create mock logger
      const logger = {
        debug: vi.fn(),
        debugError: vi.fn(),
      };

      const result = bootstrapSafeInputsServer(configPath, logger);

      expect(result.config.serverName).toBe("test-server");
      expect(result.config.version).toBe("1.0.0");
      expect(result.config.tools).toHaveLength(1);
      expect(result.basePath).toBe(tempDir);
      expect(result.tools).toBeInstanceOf(Array);
      expect(logger.debug).toHaveBeenCalled();
    });

    it("should load tools with handlers", async () => {
      const { bootstrapSafeInputsServer } = await import("./safe_inputs_bootstrap.cjs");

      // Create a test handler file
      const handlerPath = path.join(tempDir, "test_handler.cjs");
      fs.writeFileSync(handlerPath, `module.exports = function(args) { return { content: [{ type: "text", text: "test" }] }; };`);

      // Create a test configuration file with handler
      const configPath = path.join(tempDir, "config.json");
      const config = {
        serverName: "test-server",
        version: "1.0.0",
        tools: [
          {
            name: "test_tool",
            description: "A test tool",
            inputSchema: { type: "object", properties: {} },
            handler: "test_handler.cjs",
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      // Create mock logger
      const logger = {
        debug: vi.fn(),
        debugError: vi.fn(),
      };

      const result = bootstrapSafeInputsServer(configPath, logger);

      expect(result.tools).toHaveLength(1);
      expect(result.tools[0].name).toBe("test_tool");
      expect(result.tools[0].handler).toBeInstanceOf(Function);
    });

    it("should throw error for non-existent config file", async () => {
      const { bootstrapSafeInputsServer } = await import("./safe_inputs_bootstrap.cjs");

      const logger = {
        debug: vi.fn(),
        debugError: vi.fn(),
      };

      expect(() => bootstrapSafeInputsServer("/non/existent/config.json", logger)).toThrow("Configuration file not found");
    });
  });

  describe("cleanupConfigFile", () => {
    it("should delete configuration file", async () => {
      const { cleanupConfigFile } = await import("./safe_inputs_bootstrap.cjs");

      // Create a test configuration file
      const configPath = path.join(tempDir, "config.json");
      fs.writeFileSync(configPath, JSON.stringify({ tools: [] }));

      // Create mock logger
      const logger = {
        debug: vi.fn(),
        debugError: vi.fn(),
      };

      expect(fs.existsSync(configPath)).toBe(true);

      cleanupConfigFile(configPath, logger);

      expect(fs.existsSync(configPath)).toBe(false);
      expect(logger.debug).toHaveBeenCalledWith(expect.stringContaining("Deleted configuration file"));
    });

    it("should handle non-existent file gracefully", async () => {
      const { cleanupConfigFile } = await import("./safe_inputs_bootstrap.cjs");

      const logger = {
        debug: vi.fn(),
        debugError: vi.fn(),
      };

      // Should not throw error
      cleanupConfigFile("/non/existent/config.json", logger);

      expect(logger.debugError).not.toHaveBeenCalled();
    });

    it("should handle deletion errors gracefully", async () => {
      const { cleanupConfigFile } = await import("./safe_inputs_bootstrap.cjs");

      // Create a test configuration file
      const configPath = path.join(tempDir, "config.json");
      fs.writeFileSync(configPath, JSON.stringify({ tools: [] }));

      // Mock fs.unlinkSync to throw an error
      const originalUnlink = fs.unlinkSync;
      fs.unlinkSync = vi.fn(() => {
        throw new Error("Permission denied");
      });

      const logger = {
        debug: vi.fn(),
        debugError: vi.fn(),
      };

      // Should not throw error, just log warning
      cleanupConfigFile(configPath, logger);

      expect(logger.debugError).toHaveBeenCalledWith(expect.stringContaining("Warning: Could not delete configuration file"), expect.any(Error));

      // Restore original function
      fs.unlinkSync = originalUnlink;
    });
  });
});
