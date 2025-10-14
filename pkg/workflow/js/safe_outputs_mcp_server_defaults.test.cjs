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
    const defaultOutputDir = "/tmp/gh-aw/safe-outputs";
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
        expect(stderr).toContain("GITHUB_AW_SAFE_OUTPUTS not set, using default: /tmp/gh-aw/safe-outputs/outputs.jsonl");
        expect(stderr).toContain(
          "GITHUB_AW_SAFE_OUTPUTS_CONFIG not set, attempting to read from default path: /tmp/gh-aw/safe-outputs/config.json"
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
    const defaultConfigDir = "/tmp/gh-aw/safe-outputs";
    const defaultConfigFile = path.join(defaultConfigDir, "config.json");

    if (!fs.existsSync(defaultConfigDir)) {
      fs.mkdirSync(defaultConfigDir, { recursive: true });
    }

    const testConfig = {
      create_issue: { enabled: true },
      add_comment: { enabled: true, max: 3 },
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
        expect(stderr).toContain("Reading config from file: /tmp/gh-aw/safe-outputs/config.json");
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
    const defaultConfigFile = "/tmp/gh-aw/safe-outputs/config.json";
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
        expect(stderr).toContain("Config file does not exist at: /tmp/gh-aw/safe-outputs/config.json");
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
      add_labels: {
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
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl",
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
      add_labels: {
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
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl",
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

// Test that update_issue tool description is patched with allowed operations
describe("safe_outputs_mcp_server.cjs update_issue tool patching", () => {
  it("should patch update_issue tool description with allowed operations when some are restricted", async () => {
    const config = {
      update_issue: {
        status: true,
        title: false,
        body: true,
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
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl",
        },
      });

      let receivedMessages = [];

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

        // Find the update_issue tool
        const updateIssueTool = listResponse.result.tools.find(t => t.name === "update_issue");
        expect(updateIssueTool).toBeDefined();

        // Check that the description includes the allowed operations
        expect(updateIssueTool.description).toContain("status");
        expect(updateIssueTool.description).toContain("body");
        expect(updateIssueTool.description).not.toContain("title");
        expect(updateIssueTool.description).toContain("Allowed updates:");

        resolve();
      }, 500);
    });
  });

  it("should not patch update_issue tool description when all operations are allowed", async () => {
    const config = {
      update_issue: {
        status: true,
        title: true,
        body: true,
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
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl",
        },
      });

      let receivedMessages = [];

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

        // Find the update_issue tool
        const updateIssueTool = listResponse.result.tools.find(t => t.name === "update_issue");
        expect(updateIssueTool).toBeDefined();

        // Check that the description is the default (no "Allowed updates:" text)
        expect(updateIssueTool.description).toBe("Update a GitHub issue");
        expect(updateIssueTool.description).not.toContain("Allowed updates:");

        resolve();
      }, 500);
    });
  });

  it("should not patch update_issue tool description when config is not present", async () => {
    const config = {
      update_issue: {
        // No status, title, body fields - defaults to allowing all
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
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl",
        },
      });

      let receivedMessages = [];

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

        // Find the update_issue tool
        const updateIssueTool = listResponse.result.tools.find(t => t.name === "update_issue");
        expect(updateIssueTool).toBeDefined();

        // Check that the description is the default (no "Allowed updates:" text)
        expect(updateIssueTool.description).toBe("Update a GitHub issue");
        expect(updateIssueTool.description).not.toContain("Allowed updates:");

        resolve();
      }, 500);
    });
  });
});

// Test that upload_asset tool description is patched with constraints from environment
describe("safe_outputs_mcp_server.cjs upload_asset tool patching", () => {
  it("should patch upload_asset tool description with max size and allowed extensions", async () => {
    const config = {
      upload_asset: {},
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
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl",
          GITHUB_AW_ASSETS_MAX_SIZE_KB: "5120",
          GITHUB_AW_ASSETS_ALLOWED_EXTS: ".pdf,.txt,.md",
        },
      });

      let receivedMessages = [];

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

        // Find the upload_asset tool
        const uploadAssetTool = listResponse.result.tools.find(t => t.name === "upload_asset");
        expect(uploadAssetTool).toBeDefined();

        // Check that the description includes the constraints
        expect(uploadAssetTool.description).toContain("5120 KB");
        expect(uploadAssetTool.description).toContain(".pdf");
        expect(uploadAssetTool.description).toContain(".txt");
        expect(uploadAssetTool.description).toContain(".md");
        expect(uploadAssetTool.description).toContain("Maximum file size:");
        expect(uploadAssetTool.description).toContain("Allowed extensions:");

        resolve();
      }, 500);
    });
  });

  it("should patch upload_asset tool description with defaults when env vars not set", async () => {
    const config = {
      upload_asset: {},
    };

    const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        child.kill();
        reject(new Error("Test timeout"));
      }, 5000);

      // Remove environment variables
      const envWithoutAssetVars = { ...process.env };
      delete envWithoutAssetVars.GITHUB_AW_ASSETS_MAX_SIZE_KB;
      delete envWithoutAssetVars.GITHUB_AW_ASSETS_ALLOWED_EXTS;

      const child = spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...envWithoutAssetVars,
          GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify(config),
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl",
        },
      });

      let receivedMessages = [];

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

        // Find the upload_asset tool
        const uploadAssetTool = listResponse.result.tools.find(t => t.name === "upload_asset");
        expect(uploadAssetTool).toBeDefined();

        // Check that the description includes the default constraints
        expect(uploadAssetTool.description).toContain("10240 KB");
        expect(uploadAssetTool.description).toContain(".png");
        expect(uploadAssetTool.description).toContain(".jpg");
        expect(uploadAssetTool.description).toContain(".jpeg");
        expect(uploadAssetTool.description).toContain("Maximum file size:");
        expect(uploadAssetTool.description).toContain("Allowed extensions:");

        resolve();
      }, 500);
    });
  });
});

// Test that create_pull_request and push_to_pull_request_branch tools have optional branch parameter
describe("safe_outputs_mcp_server.cjs branch parameter handling", () => {
  it("should have optional branch parameter for create_pull_request", async () => {
    const config = {
      create_pull_request: {},
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
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl",
        },
      });

      let receivedMessages = [];

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

        // Find the create_pull_request tool
        const createPrTool = listResponse.result.tools.find(t => t.name === "create_pull_request");
        expect(createPrTool).toBeDefined();

        // Check that branch is NOT in required fields
        expect(createPrTool.inputSchema.required).toEqual(["title", "body"]);
        expect(createPrTool.inputSchema.required).not.toContain("branch");

        // Check that branch property exists and has the correct description
        expect(createPrTool.inputSchema.properties.branch).toBeDefined();
        expect(createPrTool.inputSchema.properties.branch.description).toContain("Optional");
        expect(createPrTool.inputSchema.properties.branch.description).toContain("current branch");

        resolve();
      }, 500);
    });
  });

  it("should have optional branch parameter for push_to_pull_request_branch", async () => {
    const config = {
      push_to_pull_request_branch: {},
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
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl",
        },
      });

      let receivedMessages = [];

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

        // Find the push_to_pull_request_branch tool
        const pushTool = listResponse.result.tools.find(t => t.name === "push_to_pull_request_branch");
        expect(pushTool).toBeDefined();

        // Check that branch is NOT in required fields (only message is required)
        expect(pushTool.inputSchema.required).toEqual(["message"]);
        expect(pushTool.inputSchema.required).not.toContain("branch");

        // Check that branch property exists and has the correct description
        expect(pushTool.inputSchema.properties.branch).toBeDefined();
        expect(pushTool.inputSchema.properties.branch.description).toContain("Optional");
        expect(pushTool.inputSchema.properties.branch.description).toContain("current branch");

        resolve();
      }, 500);
    });
  });
});

// Test that tool call responses include the isError field
describe("safe_outputs_mcp_server.cjs tool call response format", () => {
  it("should include isError field in tool call responses", async () => {
    const config = {
      create_issue: {},
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
          GITHUB_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs-iserror.jsonl",
        },
      });

      let receivedMessages = [];

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

      // Send tools/call request after initialization
      setTimeout(() => {
        const toolCallMessage =
          JSON.stringify({
            jsonrpc: "2.0",
            id: 2,
            method: "tools/call",
            params: {
              name: "create_issue",
              arguments: {
                title: "Test Issue",
                body: "Test body",
              },
            },
          }) + "\n";
        child.stdin.write(toolCallMessage);
      }, 200);

      // Check results after a delay
      setTimeout(() => {
        clearTimeout(timeout);
        child.kill();

        // Find the tools/call response
        const toolCallResponse = receivedMessages.find(m => m.id === 2);
        expect(toolCallResponse).toBeDefined();
        expect(toolCallResponse.result).toBeDefined();

        // Verify the response includes isError field
        expect(toolCallResponse.result.isError).toBeDefined();
        expect(toolCallResponse.result.isError).toBe(false);

        // Verify the response includes content
        expect(toolCallResponse.result.content).toBeDefined();
        expect(Array.isArray(toolCallResponse.result.content)).toBe(true);

        resolve();
      }, 500);
    });
  });
});
