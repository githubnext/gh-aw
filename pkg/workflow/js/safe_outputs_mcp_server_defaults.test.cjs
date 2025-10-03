import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { spawn } from "child_process";

// Test defaults for safe outputs MCP server
describe("safe_outputs_mcp_server.cjs defaults handling", () => {
  let originalEnv;
  let tempConfigFile;
  let tempOutputDir;

  beforeEach(() => {
    originalEnv = { ...process.env };

    // Create temporary directories for testing
    tempOutputDir = path.join("/tmp", `test_safe_outputs_defaults_${Date.now()}`);
    fs.mkdirSync(tempOutputDir, { recursive: true });

    tempConfigFile = path.join(tempOutputDir, "config.json");
  });

  afterEach(() => {
    process.env = originalEnv;

    // Clean up temporary files
    if (fs.existsSync(tempConfigFile)) {
      fs.unlinkSync(tempConfigFile);
    }
    if (fs.existsSync(tempOutputDir)) {
      fs.rmSync(tempOutputDir, { recursive: true, force: true });
    }
  });

  it("should use default output file when GITHUB_AW_SAFE_OUTPUTS is not set", async () => {
    // Remove environment variables
    delete process.env.GITHUB_AW_SAFE_OUTPUTS;
    delete process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;

    // Create default directories
    const defaultOutputDir = "/tmp/safe-outputs";
    if (!fs.existsSync(defaultOutputDir)) {
      fs.mkdirSync(defaultOutputDir, { recursive: true });
    }

    const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        child.kill();
        reject(new Error("Test timeout"));
      }, 5000);

      const child = spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: { ...process.env },
      });

      let stderr = "";
      let stdout = "";

      child.stderr.on("data", data => {
        stderr += data.toString();
      });

      child.stdout.on("data", data => {
        stdout += data.toString();
      });

      child.on("error", error => {
        clearTimeout(timeout);
        reject(error);
      });

      // Send initialization message
      const initMessage =
        JSON.stringify({
          jsonrpc: "2.0",
          id: 1,
          method: "initialize",
          params: {
            protocolVersion: "2024-11-05",
            capabilities: {},
            clientInfo: { name: "test-client", version: "1.0.0" },
          },
        }) + "\n";

      child.stdin.write(initMessage);

      // Wait for response
      setTimeout(() => {
        child.kill();
        clearTimeout(timeout);

        // Check that default paths are mentioned in debug output
        expect(stderr).toContain("GITHUB_AW_SAFE_OUTPUTS not set, using default: /tmp/safe-outputs/outputs.jsonl");
        expect(stderr).toContain(
          "GITHUB_AW_SAFE_OUTPUTS_CONFIG not set, attempting to read from default path: /tmp/safe-outputs/config.json"
        );

        resolve();
      }, 2000);
    });
  });

  it("should read config from default file when config file exists", async () => {
    // Remove environment variables
    delete process.env.GITHUB_AW_SAFE_OUTPUTS;
    delete process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;

    // Create default config file
    const defaultConfigDir = "/tmp/safe-outputs";
    const defaultConfigFile = path.join(defaultConfigDir, "config.json");

    if (!fs.existsSync(defaultConfigDir)) {
      fs.mkdirSync(defaultConfigDir, { recursive: true });
    }

    const testConfig = {
      "create-issue": { enabled: true },
      "add-comment": { enabled: true, max: 3 },
    };

    fs.writeFileSync(defaultConfigFile, JSON.stringify(testConfig));

    const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        child.kill();
        reject(new Error("Test timeout"));
      }, 5000);

      const child = spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: { ...process.env },
      });

      let stderr = "";

      child.stderr.on("data", data => {
        stderr += data.toString();
      });

      child.on("error", error => {
        clearTimeout(timeout);
        reject(error);
      });

      // Send initialization message
      const initMessage =
        JSON.stringify({
          jsonrpc: "2.0",
          id: 1,
          method: "initialize",
          params: {
            protocolVersion: "2024-11-05",
            capabilities: {},
            clientInfo: { name: "test-client", version: "1.0.0" },
          },
        }) + "\n";

      child.stdin.write(initMessage);

      // Wait for response
      setTimeout(() => {
        child.kill();
        clearTimeout(timeout);

        // Clean up test config file
        if (fs.existsSync(defaultConfigFile)) {
          fs.unlinkSync(defaultConfigFile);
        }

        // Check that config was read from file
        expect(stderr).toContain("Reading config from file: /tmp/safe-outputs/config.json");
        expect(stderr).toContain("Successfully parsed config from file with 2 configuration keys");
        expect(stderr).toContain("Final processed config:");
        expect(stderr).toContain("create_issue");

        resolve();
      }, 2000);
    });
  });

  it("should use empty config when default file does not exist", async () => {
    // Remove environment variables
    delete process.env.GITHUB_AW_SAFE_OUTPUTS;
    delete process.env.GITHUB_AW_SAFE_OUTPUTS_CONFIG;

    // Ensure default config file does not exist
    const defaultConfigFile = "/tmp/safe-outputs/config.json";
    if (fs.existsSync(defaultConfigFile)) {
      fs.unlinkSync(defaultConfigFile);
    }

    const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        child.kill();
        reject(new Error("Test timeout"));
      }, 5000);

      const child = spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: { ...process.env },
      });

      let stderr = "";

      child.stderr.on("data", data => {
        stderr += data.toString();
      });

      child.on("error", error => {
        clearTimeout(timeout);
        reject(error);
      });

      // Send initialization message
      const initMessage =
        JSON.stringify({
          jsonrpc: "2.0",
          id: 1,
          method: "initialize",
          params: {
            protocolVersion: "2024-11-05",
            capabilities: {},
            clientInfo: { name: "test-client", version: "1.0.0" },
          },
        }) + "\n";

      child.stdin.write(initMessage);

      // Wait for response
      setTimeout(() => {
        child.kill();
        clearTimeout(timeout);

        // Check that empty config is used when file doesn't exist
        expect(stderr).toContain("Config file does not exist at: /tmp/safe-outputs/config.json");
        expect(stderr).toContain("Using minimal default configuration");
        expect(stderr).toContain("Final processed config: {}");

        resolve();
      }, 2000);
    });
  });
});

// Test that add_labels tool description is patched with allowed labels
describe("safe_outputs_mcp_server.cjs add_labels tool patching", () => {
  it("should patch add_labels tool description with allowed labels from config", async () => {
    const config = {
      "add-labels": {
        allowed: ["bug", "enhancement", "documentation"],
        max: 3,
      },
    };

    const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        child.kill();
        reject(new Error("Test timeout"));
      }, 5000);

      const child = spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...process.env,
          GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify(config),
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/test-outputs.jsonl",
        },
      });

      let stderr = "";
      let receivedMessages = [];

      child.stderr.on("data", data => {
        stderr += data.toString();
      });

      child.stdout.on("data", data => {
        const lines = data
          .toString()
          .split("\n")
          .filter(l => l.trim());
        lines.forEach(line => {
          try {
            const msg = JSON.parse(line);
            receivedMessages.push(msg);
          } catch (e) {
            // Ignore parse errors
          }
        });
      });

      child.on("error", error => {
        clearTimeout(timeout);
        reject(error);
      });

      // Send initialization message
      setTimeout(() => {
        const initMessage =
          JSON.stringify({
            jsonrpc: "2.0",
            id: 1,
            method: "initialize",
            params: {
              protocolVersion: "2024-11-05",
              capabilities: {},
              clientInfo: { name: "test-client", version: "1.0.0" },
            },
          }) + "\n";
        child.stdin.write(initMessage);
      }, 100);

      // Send tools/list request after initialization
      setTimeout(() => {
        const listToolsMessage =
          JSON.stringify({
            jsonrpc: "2.0",
            id: 2,
            method: "tools/list",
            params: {},
          }) + "\n";
        child.stdin.write(listToolsMessage);
      }, 200);

      // Check results after a delay
      setTimeout(() => {
        clearTimeout(timeout);
        child.kill();

        // Find the tools/list response
        const listResponse = receivedMessages.find(m => m.id === 2);
        expect(listResponse).toBeDefined();
        expect(listResponse.result).toBeDefined();
        expect(listResponse.result.tools).toBeDefined();

        // Find the add_labels tool
        const addLabelsTool = listResponse.result.tools.find(t => t.name === "add_labels");
        expect(addLabelsTool).toBeDefined();

        // Check that the description includes the allowed labels
        expect(addLabelsTool.description).toContain("bug");
        expect(addLabelsTool.description).toContain("enhancement");
        expect(addLabelsTool.description).toContain("documentation");
        expect(addLabelsTool.description).toContain("Allowed labels:");

        resolve();
      }, 500);
    });
  });

  it("should not patch add_labels tool description when no allowed labels configured", async () => {
    const config = {
      "add-labels": {
        max: 3,
      },
    };

    const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        child.kill();
        reject(new Error("Test timeout"));
      }, 5000);

      const child = spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...process.env,
          GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify(config),
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/test-outputs.jsonl",
        },
      });

      let stderr = "";
      let receivedMessages = [];

      child.stderr.on("data", data => {
        stderr += data.toString();
      });

      child.stdout.on("data", data => {
        const lines = data
          .toString()
          .split("\n")
          .filter(l => l.trim());
        lines.forEach(line => {
          try {
            const msg = JSON.parse(line);
            receivedMessages.push(msg);
          } catch (e) {
            // Ignore parse errors
          }
        });
      });

      child.on("error", error => {
        clearTimeout(timeout);
        reject(error);
      });

      // Send initialization message
      setTimeout(() => {
        const initMessage =
          JSON.stringify({
            jsonrpc: "2.0",
            id: 1,
            method: "initialize",
            params: {
              protocolVersion: "2024-11-05",
              capabilities: {},
              clientInfo: { name: "test-client", version: "1.0.0" },
            },
          }) + "\n";
        child.stdin.write(initMessage);
      }, 100);

      // Send tools/list request after initialization
      setTimeout(() => {
        const listToolsMessage =
          JSON.stringify({
            jsonrpc: "2.0",
            id: 2,
            method: "tools/list",
            params: {},
          }) + "\n";
        child.stdin.write(listToolsMessage);
      }, 200);

      // Check results after a delay
      setTimeout(() => {
        clearTimeout(timeout);
        child.kill();

        // Find the tools/list response
        const listResponse = receivedMessages.find(m => m.id === 2);
        expect(listResponse).toBeDefined();
        expect(listResponse.result).toBeDefined();
        expect(listResponse.result.tools).toBeDefined();

        // Find the add_labels tool
        const addLabelsTool = listResponse.result.tools.find(t => t.name === "add_labels");
        expect(addLabelsTool).toBeDefined();

        // Check that the description is the default (no "Allowed labels:" text)
        expect(addLabelsTool.description).toBe("Add labels to a GitHub issue or pull request");
        expect(addLabelsTool.description).not.toContain("Allowed labels:");

        resolve();
      }, 500);
    });
  });
});
