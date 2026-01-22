import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { spawn } from "child_process";
(describe.sequential("safe_outputs_mcp_server.cjs defaults handling", () => {
  let originalEnv, tempConfigFile, tempOutputDir;
  (beforeEach(() => {
    ((originalEnv = { ...process.env }),
      (tempOutputDir = path.join("/tmp", `test_safe_outputs_defaults_${Date.now()}`)),
      fs.mkdirSync(tempOutputDir, { recursive: !0 }),
      (tempConfigFile = path.join(tempOutputDir, "config.json")),
      fs.existsSync("/opt/gh-aw/safeoutputs") || fs.mkdirSync("/opt/gh-aw/safeoutputs", { recursive: !0 }));
    const defaultConfigPath = path.join("/opt/gh-aw/safeoutputs", "config.json");
    fs.writeFileSync(defaultConfigPath, JSON.stringify({ create_issue: !0, missing_tool: !0 }));
    const toolsJsonPath = path.join("/opt/gh-aw/safeoutputs", "tools.json"),
      toolsJsonContent = fs.readFileSync(path.join(__dirname, "safe_outputs_tools.json"), "utf8");
    fs.writeFileSync(toolsJsonPath, toolsJsonContent);
  }),
    afterEach(() => {
      ((process.env = originalEnv), fs.existsSync(tempConfigFile) && fs.unlinkSync(tempConfigFile), fs.existsSync(tempOutputDir) && fs.rmSync(tempOutputDir, { recursive: !0, force: !0 }));
    }),
    it("should use default output file when GH_AW_SAFE_OUTPUTS is not set", async () => {
      (delete process.env.GH_AW_SAFE_OUTPUTS, delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH, fs.existsSync("/opt/gh-aw/safeoutputs") || fs.mkdirSync("/opt/gh-aw/safeoutputs", { recursive: !0 }));
      const defaultConfigPath = path.join("/opt/gh-aw/safeoutputs", "config.json");
      fs.writeFileSync(defaultConfigPath, JSON.stringify({ create_issue: !0, missing_tool: !0 }));
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");
      return new Promise((resolve, reject) => {
        const timeout = setTimeout(() => {
            (child.kill(), reject(new Error("Test timeout")));
          }, 5e3),
          child = spawn("node", [serverPath], { stdio: ["pipe", "pipe", "pipe"], env: { ...process.env } });
        let stderr = "",
          stdout = "";
        (child.stderr.on("data", data => {
          stderr += data.toString();
        }),
          child.stdout.on("data", data => {
            stdout += data.toString();
          }),
          child.on("error", error => {
            (clearTimeout(timeout), reject(error));
          }));
        const initMessage = JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test-client", version: "1.0.0" } } }) + "\n";
        (child.stdin.write(initMessage),
          setTimeout(() => {
            (child.kill(),
              clearTimeout(timeout),
              expect(stderr).toContain("GH_AW_SAFE_OUTPUTS not set, using default: /opt/gh-aw/safeoutputs/outputs.jsonl"),
              expect(stderr).toContain("Reading config from file: /opt/gh-aw/safeoutputs/config.json"),
              resolve());
          }, 2e3));
      });
    }),
    it("should read config from default file when config file exists", async () => {
      (delete process.env.GH_AW_SAFE_OUTPUTS, delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH);
      const defaultConfigFile = path.join("/opt/gh-aw/safeoutputs", "config.json");
      (fs.existsSync("/opt/gh-aw/safeoutputs") || fs.mkdirSync("/opt/gh-aw/safeoutputs", { recursive: !0 }), fs.writeFileSync(defaultConfigFile, JSON.stringify({ create_issue: { enabled: !0 }, add_comment: { enabled: !0, max: 3 } })));
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");
      return new Promise((resolve, reject) => {
        const timeout = setTimeout(() => {
            (child.kill(), reject(new Error("Test timeout")));
          }, 5e3),
          child = spawn("node", [serverPath], { stdio: ["pipe", "pipe", "pipe"], env: { ...process.env } });
        let stderr = "";
        (child.stderr.on("data", data => {
          stderr += data.toString();
        }),
          child.on("error", error => {
            (clearTimeout(timeout), reject(error));
          }));
        const initMessage = JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test-client", version: "1.0.0" } } }) + "\n";
        (child.stdin.write(initMessage),
          setTimeout(() => {
            (child.kill(),
              clearTimeout(timeout),
              fs.existsSync(defaultConfigFile) && fs.unlinkSync(defaultConfigFile),
              expect(stderr).toContain("Reading config from file: /opt/gh-aw/safeoutputs/config.json"),
              expect(stderr).toContain("Successfully parsed config from file with 2 configuration keys"),
              expect(stderr).toContain("Final processed config:"),
              expect(stderr).toContain("create_issue"),
              resolve());
          }, 2e3));
      });
    }),
    it("should use empty config when default file does not exist", async () => {
      (delete process.env.GH_AW_SAFE_OUTPUTS, delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH, fs.existsSync("/opt/gh-aw/safeoutputs/config.json") && fs.unlinkSync("/opt/gh-aw/safeoutputs/config.json"));
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");
      return new Promise((resolve, reject) => {
        const timeout = setTimeout(() => {
            (child.kill(), reject(new Error("Test timeout")));
          }, 5e3),
          child = spawn("node", [serverPath], { stdio: ["pipe", "pipe", "pipe"], env: { ...process.env } });
        let stderr = "";
        (child.stderr.on("data", data => {
          stderr += data.toString();
        }),
          child.on("error", error => {
            (clearTimeout(timeout), reject(error));
          }));
        const initMessage = JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test-client", version: "1.0.0" } } }) + "\n";
        (child.stdin.write(initMessage),
          setTimeout(() => {
            (child.kill(),
              clearTimeout(timeout),
              expect(stderr).toContain("Config file does not exist at: /opt/gh-aw/safeoutputs/config.json"),
              expect(stderr).toContain("Using minimal default configuration"),
              expect(stderr).toContain("Final processed config: {}"),
              resolve());
          }, 2e3));
      });
    }),
    it("should create output directory even when GH_AW_SAFE_OUTPUTS is set", async () => {
      const testOutputDir = path.join("/tmp", `test_safe_outputs_${Date.now()}_envset`),
        testOutputFile = path.join(testOutputDir, "outputs.jsonl");
      ((process.env.GH_AW_SAFE_OUTPUTS = testOutputFile),
        delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH,
        fs.existsSync(testOutputDir) && fs.rmSync(testOutputDir, { recursive: !0, force: !0 }),
        fs.existsSync("/tmp/gh-aw/safeoutputs") || fs.mkdirSync("/tmp/gh-aw/safeoutputs", { recursive: !0 }));
      const defaultConfigPath = path.join("/tmp/gh-aw/safeoutputs", "config.json");
      fs.writeFileSync(defaultConfigPath, JSON.stringify({ create_issue: !0, missing_tool: !0 }));
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");
      return new Promise((resolve, reject) => {
        const timeout = setTimeout(() => {
            (child.kill(), reject(new Error("Test timeout")));
          }, 5e3),
          child = spawn("node", [serverPath], { stdio: ["pipe", "pipe", "pipe"], env: { ...process.env } });
        let stderr = "";
        (child.stderr.on("data", data => {
          stderr += data.toString();
        }),
          child.on("error", error => {
            (clearTimeout(timeout), fs.existsSync(testOutputDir) && fs.rmSync(testOutputDir, { recursive: !0, force: !0 }), reject(error));
          }));
        const initMessage = JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test-client", version: "1.0.0" } } }) + "\n";
        (child.stdin.write(initMessage),
          setTimeout(() => {
            (child.kill(), clearTimeout(timeout), fs.existsSync(testOutputDir) && fs.rmSync(testOutputDir, { recursive: !0, force: !0 }), expect(stderr).toContain(`Creating output directory: ${testOutputDir}`), resolve());
          }, 2e3));
      });
    }));
}),
  describe.sequential("safe_outputs_mcp_server.cjs branch parameter handling", () => {
    (it("should have optional branch parameter for create_pull_request", async () => {
      const tempConfigPath = path.join("/tmp", `test-config-${Date.now()}-${Math.random().toString(36).substring(7)}.json`);
      fs.writeFileSync(tempConfigPath, JSON.stringify({ create_pull_request: {} }));
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");
      return new Promise((resolve, reject) => {
        const timeout = setTimeout(() => {
            (child.kill(), reject(new Error("Test timeout")));
          }, 5e3),
          child = spawn("node", [serverPath], { stdio: ["pipe", "pipe", "pipe"], env: { ...process.env, GH_AW_SAFE_OUTPUTS_CONFIG_PATH: tempConfigPath, GH_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs.jsonl" } });
        let receivedMessages = [];
        (child.stdout.on("data", data => {
          data
            .toString()
            .split("\n")
            .filter(l => l.trim())
            .forEach(line => {
              try {
                const msg = JSON.parse(line);
                receivedMessages.push(msg);
              } catch (e) {}
            });
        }),
          child.on("error", error => {
            (clearTimeout(timeout), reject(error));
          }),
          setTimeout(() => {
            const initMessage = JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test-client", version: "1.0.0" } } }) + "\n";
            child.stdin.write(initMessage);
          }, 100),
          setTimeout(() => {
            const listToolsMessage = JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} }) + "\n";
            child.stdin.write(listToolsMessage);
          }, 200),
          setTimeout(() => {
            (clearTimeout(timeout), child.kill());
            const listResponse = receivedMessages.find(m => 2 === m.id);
            (expect(listResponse).toBeDefined(), expect(listResponse.result).toBeDefined(), expect(listResponse.result.tools).toBeDefined());
            const createPrTool = listResponse.result.tools.find(t => "create_pull_request" === t.name);
            (expect(createPrTool).toBeDefined(),
              expect(createPrTool.inputSchema.required).toEqual(["title", "body"]),
              expect(createPrTool.inputSchema.required).not.toContain("branch"),
              expect(createPrTool.inputSchema.properties.branch).toBeDefined(),
              expect(createPrTool.inputSchema.properties.branch.description).toContain("If omitted"),
              expect(createPrTool.inputSchema.properties.branch.description).toContain("current"),
              resolve());
          }, 500));
      });
    }),
      it("should have optional branch parameter for push_to_pull_request_branch", async () => {
        const tempConfigPath = path.join("/tmp", `test-config-${Date.now()}-${Math.random().toString(36).substring(7)}.json`);
        fs.writeFileSync(tempConfigPath, JSON.stringify({ push_to_pull_request_branch: {} }));
        const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");
        return new Promise((resolve, reject) => {
          const timeout = setTimeout(() => {
              (child.kill(), reject(new Error("Test timeout")));
            }, 5e3),
            child = spawn("node", [serverPath], { stdio: ["pipe", "pipe", "pipe"], env: { ...process.env, GH_AW_SAFE_OUTPUTS_CONFIG_PATH: tempConfigPath, GH_AW_SAFE_OUTPUTS: `/tmp/gh-aw/test-outputs-push-${Date.now()}.jsonl` } });
          let receivedMessages = [];
          (child.stdout.on("data", data => {
            data
              .toString()
              .split("\n")
              .filter(l => l.trim())
              .forEach(line => {
                try {
                  const msg = JSON.parse(line);
                  receivedMessages.push(msg);
                } catch (e) {}
              });
          }),
            child.on("error", error => {
              (clearTimeout(timeout), reject(error));
            }),
            setTimeout(() => {
              const initMessage = JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test-client", version: "1.0.0" } } }) + "\n";
              child.stdin.write(initMessage);
            }, 100),
            setTimeout(() => {
              const listToolsMessage = JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/list", params: {} }) + "\n";
              child.stdin.write(listToolsMessage);
            }, 200),
            setTimeout(() => {
              (clearTimeout(timeout), child.kill());
              const listResponse = receivedMessages.find(m => 2 === m.id);
              (expect(listResponse).toBeDefined(), expect(listResponse.result).toBeDefined(), expect(listResponse.result.tools).toBeDefined());
              const pushTool = listResponse.result.tools.find(t => "push_to_pull_request_branch" === t.name);
              (expect(pushTool).toBeDefined(),
                expect(pushTool.inputSchema.required).toEqual(["message"]),
                expect(pushTool.inputSchema.required).not.toContain("branch"),
                expect(pushTool.inputSchema.properties.branch).toBeDefined(),
                expect(pushTool.inputSchema.properties.branch.description).toContain("If omitted"),
                expect(pushTool.inputSchema.properties.branch.description).toContain("current"),
                resolve());
            }, 500));
        });
      }));
  }),
  describe.sequential("safe_outputs_mcp_server.cjs tool call response format", () => {
    (it("should include isError field in tool call responses", async () => {
      const tempConfigPath = path.join("/tmp", `test-config-${Date.now()}-${Math.random().toString(36).substring(7)}.json`);
      fs.writeFileSync(tempConfigPath, JSON.stringify({ create_issue: {} }));
      const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");
      return new Promise((resolve, reject) => {
        const timeout = setTimeout(() => {
            (child.kill(), reject(new Error("Test timeout")));
          }, 5e3),
          child = spawn("node", [serverPath], { stdio: ["pipe", "pipe", "pipe"], env: { ...process.env, GH_AW_SAFE_OUTPUTS_CONFIG_PATH: tempConfigPath, GH_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs-iserror.jsonl" } });
        let receivedMessages = [];
        (child.stdout.on("data", data => {
          data
            .toString()
            .split("\n")
            .filter(l => l.trim())
            .forEach(line => {
              try {
                const msg = JSON.parse(line);
                receivedMessages.push(msg);
              } catch (e) {}
            });
        }),
          child.on("error", error => {
            (clearTimeout(timeout), reject(error));
          }),
          setTimeout(() => {
            const initMessage = JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test-client", version: "1.0.0" } } }) + "\n";
            child.stdin.write(initMessage);
          }, 100),
          setTimeout(() => {
            const toolCallMessage = JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/call", params: { name: "create_issue", arguments: { title: "Test Issue", body: "Test body" } } }) + "\n";
            child.stdin.write(toolCallMessage);
          }, 200),
          setTimeout(() => {
            (clearTimeout(timeout), child.kill());
            const toolCallResponse = receivedMessages.find(m => 2 === m.id);
            (expect(toolCallResponse).toBeDefined(),
              expect(toolCallResponse.result).toBeDefined(),
              expect(toolCallResponse.result.isError).toBeDefined(),
              expect(toolCallResponse.result.isError).toBe(!1),
              expect(toolCallResponse.result.content).toBeDefined(),
              expect(Array.isArray(toolCallResponse.result.content)).toBe(!0),
              resolve());
          }, 500));
      });
    }),
      it("should return stringified JSON in text content", async () => {
        const tempConfigPath = path.join("/tmp", `test-config-${Date.now()}-${Math.random().toString(36).substring(7)}.json`);
        fs.writeFileSync(tempConfigPath, JSON.stringify({ create_issue: {} }));
        const serverPath = path.join(__dirname, "safe_outputs_mcp_server.cjs");
        return new Promise((resolve, reject) => {
          const timeout = setTimeout(() => {
              (child.kill(), reject(new Error("Test timeout")));
            }, 5e3),
            child = spawn("node", [serverPath], { stdio: ["pipe", "pipe", "pipe"], env: { ...process.env, GH_AW_SAFE_OUTPUTS_CONFIG_PATH: tempConfigPath, GH_AW_SAFE_OUTPUTS: "/tmp/gh-aw/test-outputs-json-response.jsonl" } });
          let receivedMessages = [];
          (child.stdout.on("data", data => {
            data
              .toString()
              .split("\n")
              .filter(l => l.trim())
              .forEach(line => {
                try {
                  const msg = JSON.parse(line);
                  receivedMessages.push(msg);
                } catch (e) {}
              });
          }),
            child.on("error", error => {
              (clearTimeout(timeout), reject(error));
            }),
            setTimeout(() => {
              const initMessage = JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { protocolVersion: "2024-11-05", capabilities: {}, clientInfo: { name: "test-client", version: "1.0.0" } } }) + "\n";
              child.stdin.write(initMessage);
            }, 100),
            setTimeout(() => {
              const toolCallMessage = JSON.stringify({ jsonrpc: "2.0", id: 2, method: "tools/call", params: { name: "create_issue", arguments: { title: "Test Issue", body: "Test body" } } }) + "\n";
              child.stdin.write(toolCallMessage);
            }, 200),
            setTimeout(() => {
              (clearTimeout(timeout), child.kill());
              const toolCallResponse = receivedMessages.find(m => 2 === m.id);
              (expect(toolCallResponse).toBeDefined(),
                expect(toolCallResponse.result).toBeDefined(),
                expect(toolCallResponse.result.content).toBeDefined(),
                expect(Array.isArray(toolCallResponse.result.content)).toBe(!0),
                expect(toolCallResponse.result.content.length).toBeGreaterThan(0));
              const firstContent = toolCallResponse.result.content[0];
              (expect(firstContent.type).toBe("text"), expect(firstContent.text).toBeDefined(), expect(() => JSON.parse(firstContent.text)).not.toThrow());
              const parsedResult = JSON.parse(firstContent.text);
              (expect(parsedResult).toHaveProperty("result"), expect(parsedResult.result).toBe("success"), resolve());
            }, 500));
        });
      }));
  }));
