// @ts-check

const fs = require("fs");
const path = require("path");
const os = require("os");
const assert = require("assert");

const { bootstrapSafeOutputsServer, cleanupConfigFile } = require("./safe_outputs_bootstrap.cjs");

describe("safe_outputs_bootstrap", () => {
  let tempDir;

  beforeEach(() => {
    // Create a temporary directory for test files
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "test-safe-outputs-bootstrap-"));
  });

  afterEach(() => {
    // Clean up temporary directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    // Clean up environment variables
    delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH;
    delete process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    delete process.env.GH_AW_SAFE_OUTPUTS;
  });

  describe("bootstrapSafeOutputsServer", () => {
    it("should load configuration and tools successfully", () => {
      // Setup test configuration and tools files
      const configPath = path.join(tempDir, "config.json");
      const toolsPath = path.join(tempDir, "tools.json");
      const config = { "create-pull-request": { enabled: true } };
      const tools = [{ name: "create_pull_request", description: "Test tool" }];

      fs.writeFileSync(configPath, JSON.stringify(config));
      fs.writeFileSync(toolsPath, JSON.stringify(tools));

      process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;
      process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = toolsPath;
      process.env.GH_AW_SAFE_OUTPUTS = path.join(tempDir, "outputs.jsonl");

      // Create mock logger
      const logger = {
        debug: () => {},
        debugError: () => {},
      };

      // Call bootstrap
      const result = bootstrapSafeOutputsServer(logger);

      // Verify results
      assert.ok(result.config);
      assert.strictEqual(result.config.create_pull_request.enabled, true);
      assert.ok(result.tools);
      assert.strictEqual(result.tools.length, 1);
      assert.strictEqual(result.tools[0].name, "create_pull_request");
      assert.ok(result.outputFile);
      assert.strictEqual(result.outputFile, path.join(tempDir, "outputs.jsonl"));
    });

    it("should handle missing configuration file gracefully", () => {
      // Setup only tools file
      const toolsPath = path.join(tempDir, "tools.json");
      const tools = [];

      fs.writeFileSync(toolsPath, JSON.stringify(tools));

      process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = path.join(tempDir, "nonexistent.json");
      process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = toolsPath;
      process.env.GH_AW_SAFE_OUTPUTS = path.join(tempDir, "outputs.jsonl");

      // Create mock logger
      const logger = {
        debug: () => {},
        debugError: () => {},
      };

      // Call bootstrap
      const result = bootstrapSafeOutputsServer(logger);

      // Verify it uses empty config
      assert.ok(result.config);
      assert.strictEqual(Object.keys(result.config).length, 0);
    });

    it("should handle missing tools file gracefully", () => {
      // Setup only config file
      const configPath = path.join(tempDir, "config.json");
      const config = {};

      fs.writeFileSync(configPath, JSON.stringify(config));

      process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;
      process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = path.join(tempDir, "nonexistent.json");
      process.env.GH_AW_SAFE_OUTPUTS = path.join(tempDir, "outputs.jsonl");

      // Create mock logger
      const logger = {
        debug: () => {},
        debugError: () => {},
      };

      // Call bootstrap
      const result = bootstrapSafeOutputsServer(logger);

      // Verify it uses empty tools array
      assert.ok(result.tools);
      assert.strictEqual(result.tools.length, 0);
    });
  });

  describe("cleanupConfigFile", () => {
    it("should delete config file if it exists", () => {
      const configPath = path.join(tempDir, "config.json");
      fs.writeFileSync(configPath, JSON.stringify({}));

      process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;

      // Create mock logger
      const logger = {
        debug: () => {},
        debugError: () => {},
      };

      // Verify file exists before cleanup
      assert.ok(fs.existsSync(configPath));

      // Call cleanup
      cleanupConfigFile(logger);

      // Verify file is deleted
      assert.ok(!fs.existsSync(configPath));
    });

    it("should handle missing config file gracefully", () => {
      const configPath = path.join(tempDir, "nonexistent.json");
      process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;

      // Create mock logger
      const logger = {
        debug: () => {},
        debugError: () => {},
      };

      // Should not throw
      assert.doesNotThrow(() => {
        cleanupConfigFile(logger);
      });
    });
  });
});
