import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

describe("safe_inputs_config_loader.cjs", () => {
  let tempDir;

  beforeEach(() => {
    // Create a temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "config-loader-test-"));
  });

  afterEach(() => {
    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  });

  describe("loadConfig", () => {
    it("should load configuration from a valid JSON file", async () => {
      const { loadConfig } = await import("./safe_inputs_config_loader.cjs");

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

      const loadedConfig = loadConfig(configPath);

      expect(loadedConfig.serverName).toBe("test-server");
      expect(loadedConfig.version).toBe("1.0.0");
      expect(loadedConfig.tools).toHaveLength(1);
      expect(loadedConfig.tools[0].name).toBe("test_tool");
    });

    it("should throw error for non-existent file", async () => {
      const { loadConfig } = await import("./safe_inputs_config_loader.cjs");

      expect(() => loadConfig("/non/existent/config.json")).toThrow("Configuration file not found");
    });

    it("should throw error for invalid JSON", async () => {
      const { loadConfig } = await import("./safe_inputs_config_loader.cjs");

      const configPath = path.join(tempDir, "invalid.json");
      fs.writeFileSync(configPath, "not valid json");

      expect(() => loadConfig(configPath)).toThrow();
    });

    it("should throw error for missing tools array", async () => {
      const { loadConfig } = await import("./safe_inputs_config_loader.cjs");

      const configPath = path.join(tempDir, "no-tools.json");
      fs.writeFileSync(configPath, JSON.stringify({ serverName: "test" }));

      expect(() => loadConfig(configPath)).toThrow("Configuration must contain a 'tools' array");
    });

    it("should throw error for tools that is not an array", async () => {
      const { loadConfig } = await import("./safe_inputs_config_loader.cjs");

      const configPath = path.join(tempDir, "tools-not-array.json");
      fs.writeFileSync(configPath, JSON.stringify({ tools: "not an array" }));

      expect(() => loadConfig(configPath)).toThrow("Configuration must contain a 'tools' array");
    });

    it("should load configuration with minimal required fields", async () => {
      const { loadConfig } = await import("./safe_inputs_config_loader.cjs");

      const configPath = path.join(tempDir, "minimal.json");
      const config = {
        tools: [],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      const loadedConfig = loadConfig(configPath);

      expect(loadedConfig.tools).toHaveLength(0);
      expect(loadedConfig.serverName).toBeUndefined();
      expect(loadedConfig.version).toBeUndefined();
    });

    it("should load configuration with multiple tools", async () => {
      const { loadConfig } = await import("./safe_inputs_config_loader.cjs");

      const configPath = path.join(tempDir, "multiple-tools.json");
      const config = {
        tools: [
          {
            name: "tool1",
            description: "Tool 1",
            inputSchema: {},
            handler: "tool1.cjs",
          },
          {
            name: "tool2",
            description: "Tool 2",
            inputSchema: {},
            handler: "tool2.sh",
          },
          {
            name: "tool3",
            description: "Tool 3",
            inputSchema: {},
            handler: "tool3.py",
          },
        ],
      };
      fs.writeFileSync(configPath, JSON.stringify(config));

      const loadedConfig = loadConfig(configPath);

      expect(loadedConfig.tools).toHaveLength(3);
      expect(loadedConfig.tools[0].name).toBe("tool1");
      expect(loadedConfig.tools[1].name).toBe("tool2");
      expect(loadedConfig.tools[2].name).toBe("tool3");
    });
  });
});
