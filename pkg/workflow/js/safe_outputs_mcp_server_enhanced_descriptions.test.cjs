import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import { spawn } from "child_process";

// Test enhanced tool descriptions based on configuration
describe("safe_outputs_mcp_server.cjs enhanced descriptions", () => {
  let originalEnv;
  let tempConfigFile;
  let tempOutputDir;

  beforeEach(() => {
    originalEnv = { ...process.env };

    // Create temporary directories for testing
    tempOutputDir = path.join("/tmp", `test_safe_outputs_enhanced_${Date.now()}`);
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

  const runServerWithConfig = (config, timeout = 5000) => {
    const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");

    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        child.kill();
        reject(new Error("Test timeout"));
      }, timeout);

      const child = spawn("node", [serverPath], {
        stdio: ["pipe", "pipe", "pipe"],
        env: {
          ...process.env,
          GITHUB_AW_SAFE_OUTPUTS_CONFIG: JSON.stringify(config),
          GITHUB_AW_SAFE_OUTPUTS: path.join(tempOutputDir, "outputs.jsonl"),
        },
      });

      let stderr = "";
      let stdout = "";
      let responses = [];

      child.stderr.on("data", data => {
        stderr += data.toString();
      });

      child.stdout.on("data", data => {
        stdout += data.toString();
        // Parse JSON-RPC responses
        const lines = stdout.split("\n").filter(line => line.trim());
        for (const line of lines) {
          try {
            const response = JSON.parse(line);
            responses.push(response);
          } catch (e) {
            // Ignore parse errors for partial lines
          }
        }
      });

      child.on("error", error => {
        clearTimeout(timer);
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

      // Wait a bit for init, then request tools list
      setTimeout(() => {
        const listToolsMessage =
          JSON.stringify({
            jsonrpc: "2.0",
            id: 2,
            method: "tools/list",
          }) + "\n";

        child.stdin.write(listToolsMessage);

        // Wait for response, then close
        setTimeout(() => {
          child.kill();
          clearTimeout(timer);

          // Find the tools/list response
          const toolsResponse = responses.find(r => r.id === 2);

          resolve({
            stderr,
            stdout,
            responses,
            toolsResponse,
          });
        }, 1000);
      }, 500);
    });
  };

  it("should enhance add_labels description when allowed labels are configured", async () => {
    const config = {
      "add-labels": {
        allowed: ["bug", "enhancement", "documentation"],
        max: 3,
      },
    };

    const { toolsResponse } = await runServerWithConfig(config);

    expect(toolsResponse).toBeDefined();
    expect(toolsResponse.result).toBeDefined();
    expect(toolsResponse.result.tools).toBeDefined();

    const addLabelsTool = toolsResponse.result.tools.find(t => t.name === "add_labels");
    expect(addLabelsTool).toBeDefined();
    expect(addLabelsTool.description).toContain("bug");
    expect(addLabelsTool.description).toContain("enhancement");
    expect(addLabelsTool.description).toContain("documentation");
  });

  it("should enhance update_issue description when updateable fields are configured", async () => {
    const config = {
      "update-issue": {
        status: true,
        title: true,
        body: false,
      },
    };

    const { toolsResponse } = await runServerWithConfig(config);

    expect(toolsResponse).toBeDefined();
    expect(toolsResponse.result).toBeDefined();
    expect(toolsResponse.result.tools).toBeDefined();

    const updateIssueTool = toolsResponse.result.tools.find(t => t.name === "update_issue");
    expect(updateIssueTool).toBeDefined();
    expect(updateIssueTool.description).toContain("status");
    expect(updateIssueTool.description).toContain("title");
    expect(updateIssueTool.description).not.toContain("body");
  });

  it("should enhance upload_asset description with allowed extensions and max size", async () => {
    const config = {
      "upload-asset": {
        "allowed-exts": [".png", ".jpg", ".gif"],
        "max-size": 5120,
      },
    };

    const { toolsResponse } = await runServerWithConfig(config);

    expect(toolsResponse).toBeDefined();
    expect(toolsResponse.result).toBeDefined();
    expect(toolsResponse.result.tools).toBeDefined();

    const uploadAssetTool = toolsResponse.result.tools.find(t => t.name === "upload_asset");
    expect(uploadAssetTool).toBeDefined();
    expect(uploadAssetTool.description).toContain(".png");
    expect(uploadAssetTool.description).toContain(".jpg");
    expect(uploadAssetTool.description).toContain(".gif");
    expect(uploadAssetTool.description).toContain("5120KB");
  });

  it("should enhance push_to_pull_request_branch description with labels and title prefix", async () => {
    const config = {
      "push-to-pull-request-branch": {
        labels: ["ai-generated", "automated"],
        "title-prefix": "[bot] ",
      },
    };

    const { toolsResponse } = await runServerWithConfig(config);

    expect(toolsResponse).toBeDefined();
    expect(toolsResponse.result).toBeDefined();
    expect(toolsResponse.result.tools).toBeDefined();

    const pushToBranchTool = toolsResponse.result.tools.find(t => t.name === "push_to_pull_request_branch");
    expect(pushToBranchTool).toBeDefined();
    expect(pushToBranchTool.description).toContain("ai-generated");
    expect(pushToBranchTool.description).toContain("automated");
    expect(pushToBranchTool.description).toContain("[bot]");
  });

  it("should not modify description when no special configuration is present", async () => {
    const config = {
      "create-issue": {
        max: 1,
      },
    };

    const { toolsResponse } = await runServerWithConfig(config);

    expect(toolsResponse).toBeDefined();
    expect(toolsResponse.result).toBeDefined();
    expect(toolsResponse.result.tools).toBeDefined();

    const createIssueTool = toolsResponse.result.tools.find(t => t.name === "create_issue");
    expect(createIssueTool).toBeDefined();
    // Should have the default description
    expect(createIssueTool.description).toBe("Create a new GitHub issue");
  });
});
