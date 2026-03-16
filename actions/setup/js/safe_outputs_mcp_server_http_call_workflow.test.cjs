// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import * as fs from "fs";
import * as path from "path";
import * as os from "os";
const { createMCPServer } = require("./safe_outputs_mcp_server_http.cjs");

/**
 * Regression tests for call_workflow tool registration in HTTP server.
 *
 * These tests verify that the HTTP server correctly registers call_workflow tools
 * that have _call_workflow_name metadata. Before the fix, these tools fell through
 * to the generic enabledTools.has(tool.name) check, which always failed because
 * the config key is "call_workflow" while the tool name is the workflow-specific
 * name (e.g. "generic_worker").
 *
 * Reference: Issue where call_workflow tools were not being registered by the HTTP
 * server even though the compiler generated them and safe_outputs_tools_loader.cjs
 * handled them correctly.
 */
describe("safe_outputs_mcp_server_http call_workflow registration", () => {
  /**
   * Helper that replicates the HTTP server predefined-tool registration logic
   * so we can test it in isolation.
   *
   * @param {Object} tool - Tool definition
   * @param {Object} config - Safe outputs configuration
   * @returns {boolean} Whether the tool should be registered
   */
  function shouldRegisterTool(tool, config) {
    const enabledTools = new Set();
    for (const [toolName, enabled] of Object.entries(config)) {
      if (enabled) {
        enabledTools.add(toolName);
      }
    }

    const isDispatchWorkflowTool = tool._workflow_name && typeof tool._workflow_name === "string" && tool._workflow_name.length > 0;
    const isCallWorkflowTool = tool._call_workflow_name && typeof tool._call_workflow_name === "string" && tool._call_workflow_name.length > 0;

    if (isDispatchWorkflowTool) {
      return !!config.dispatch_workflow;
    } else if (isCallWorkflowTool) {
      return !!config.call_workflow;
    } else {
      return enabledTools.has(tool.name);
    }
  }

  it("should register call_workflow tools when config.call_workflow exists", () => {
    const tool = {
      name: "generic_worker",
      _call_workflow_name: "generic-worker",
      description: "Call the 'generic-worker' workflow",
      inputSchema: {
        type: "object",
        properties: {
          task: {
            type: "string",
            description: "Task to perform",
          },
        },
        additionalProperties: false,
      },
    };

    const config = {
      call_workflow: {
        workflows: ["generic-worker"],
      },
      missing_tool: {},
      missing_data: {},
      noop: { max: 1 },
    };

    expect(shouldRegisterTool(tool, config)).toBe(true);
  });

  it("should NOT register call_workflow tools when config.call_workflow is absent", () => {
    const tool = {
      name: "generic_worker",
      _call_workflow_name: "generic-worker",
      description: "Call the 'generic-worker' workflow",
      inputSchema: { type: "object", properties: {} },
    };

    // Config WITHOUT call_workflow
    const config = {
      missing_tool: {},
      missing_data: {},
      noop: { max: 1 },
    };

    expect(shouldRegisterTool(tool, config)).toBe(false);
  });

  it("should register multiple call_workflow tools when config.call_workflow exists", () => {
    const tools = [
      {
        name: "generic_worker",
        _call_workflow_name: "generic-worker",
        description: "Call the 'generic-worker' workflow",
        inputSchema: { type: "object", properties: {} },
      },
      {
        name: "specialist_worker",
        _call_workflow_name: "specialist-worker",
        description: "Call the 'specialist-worker' workflow",
        inputSchema: { type: "object", properties: {} },
      },
    ];

    const config = {
      call_workflow: {
        workflows: ["generic-worker", "specialist-worker"],
      },
    };

    for (const tool of tools) {
      expect(shouldRegisterTool(tool, config)).toBe(true, `Tool ${tool.name} should be registered`);
    }
  });

  it("should NOT register call_workflow tools when config.call_workflow is falsy", () => {
    const tool = {
      name: "generic_worker",
      _call_workflow_name: "generic-worker",
      description: "Call the 'generic-worker' workflow",
      inputSchema: { type: "object", properties: {} },
    };

    // Config with falsy call_workflow value
    const config = {
      call_workflow: null,
      missing_tool: {},
    };

    expect(shouldRegisterTool(tool, config)).toBe(false);
  });

  it("should register both call_workflow and dispatch_workflow tools when both configs exist", () => {
    const callWorkflowTool = {
      name: "generic_worker",
      _call_workflow_name: "generic-worker",
      description: "Call the 'generic-worker' workflow",
      inputSchema: { type: "object", properties: {} },
    };

    const dispatchWorkflowTool = {
      name: "release_workflow",
      _workflow_name: "release-workflow",
      description: "Dispatch the 'release-workflow' workflow",
      inputSchema: { type: "object", properties: {} },
    };

    const config = {
      call_workflow: { workflows: ["generic-worker"] },
      dispatch_workflow: { workflows: ["release-workflow"] },
    };

    expect(shouldRegisterTool(callWorkflowTool, config)).toBe(true);
    expect(shouldRegisterTool(dispatchWorkflowTool, config)).toBe(true);
  });

  it("should register call_workflow tool but NOT dispatch_workflow tool when only call_workflow config exists", () => {
    const callWorkflowTool = {
      name: "generic_worker",
      _call_workflow_name: "generic-worker",
      description: "Call the 'generic-worker' workflow",
      inputSchema: { type: "object", properties: {} },
    };

    const dispatchWorkflowTool = {
      name: "release_workflow",
      _workflow_name: "release-workflow",
      description: "Dispatch the 'release-workflow' workflow",
      inputSchema: { type: "object", properties: {} },
    };

    const config = {
      call_workflow: { workflows: ["generic-worker"] },
    };

    expect(shouldRegisterTool(callWorkflowTool, config)).toBe(true);
    expect(shouldRegisterTool(dispatchWorkflowTool, config)).toBe(false);
  });

  it("should register dispatch_workflow tool but NOT call_workflow tool when only dispatch_workflow config exists", () => {
    const callWorkflowTool = {
      name: "generic_worker",
      _call_workflow_name: "generic-worker",
      description: "Call the 'generic-worker' workflow",
      inputSchema: { type: "object", properties: {} },
    };

    const dispatchWorkflowTool = {
      name: "release_workflow",
      _workflow_name: "release-workflow",
      description: "Dispatch the 'release-workflow' workflow",
      inputSchema: { type: "object", properties: {} },
    };

    const config = {
      dispatch_workflow: { workflows: ["release-workflow"] },
    };

    expect(shouldRegisterTool(callWorkflowTool, config)).toBe(false);
    expect(shouldRegisterTool(dispatchWorkflowTool, config)).toBe(true);
  });

  it("should include call_workflow tools in tools/list when config.call_workflow is set", () => {
    // Simulate the full tool list that would be registered on the server
    const allTools = [
      {
        name: "missing_tool",
        description: "Report a missing tool",
        inputSchema: { type: "object", properties: {} },
      },
      {
        name: "noop",
        description: "No-op tool",
        inputSchema: { type: "object", properties: {} },
      },
      {
        name: "generic_worker",
        _call_workflow_name: "generic-worker",
        description: "Call the 'generic-worker' workflow",
        inputSchema: { type: "object", properties: {} },
      },
    ];

    const config = {
      missing_tool: {},
      noop: { max: 1 },
      call_workflow: { workflows: ["generic-worker"] },
    };

    // Simulate registration by filtering tools using the registration logic
    const registeredToolNames = allTools.filter(tool => shouldRegisterTool(tool, config)).map(tool => tool.name);

    expect(registeredToolNames).toContain("missing_tool");
    expect(registeredToolNames).toContain("noop");
    expect(registeredToolNames).toContain("generic_worker");
    expect(registeredToolNames).toHaveLength(3);
  });

  it("should exclude call_workflow tools from tools/list when config.call_workflow is absent", () => {
    const allTools = [
      {
        name: "missing_tool",
        description: "Report a missing tool",
        inputSchema: { type: "object", properties: {} },
      },
      {
        name: "generic_worker",
        _call_workflow_name: "generic-worker",
        description: "Call the 'generic-worker' workflow",
        inputSchema: { type: "object", properties: {} },
      },
    ];

    // Config without call_workflow
    const config = {
      missing_tool: {},
    };

    const registeredToolNames = allTools.filter(tool => shouldRegisterTool(tool, config)).map(tool => tool.name);

    expect(registeredToolNames).toContain("missing_tool");
    expect(registeredToolNames).not.toContain("generic_worker");
    expect(registeredToolNames).toHaveLength(1);
  });
});

/**
 * Integration tests for call_workflow registration that exercise createMCPServer directly.
 *
 * These tests use real temp config/tools files (same pattern as safe_outputs_bootstrap.test.cjs)
 * and verify the actual server.tools Map populated by createMCPServer.
 */
describe("safe_outputs_mcp_server_http createMCPServer call_workflow integration", () => {
  let tempDir;

  beforeEach(() => {
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "test-http-call-workflow-"));
  });

  afterEach(() => {
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
    delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH;
    delete process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH;
    delete process.env.GH_AW_SAFE_OUTPUTS;
  });

  it("should register a call_workflow tool when config.call_workflow is present", () => {
    const configPath = path.join(tempDir, "config.json");
    const toolsPath = path.join(tempDir, "tools.json");
    const outputPath = path.join(tempDir, "output.jsonl");

    fs.writeFileSync(
      configPath,
      JSON.stringify({
        "call-workflow": { workflows: ["generic-worker"] },
      })
    );
    fs.writeFileSync(
      toolsPath,
      JSON.stringify([
        {
          name: "generic_worker",
          _call_workflow_name: "generic-worker",
          description: "Call the 'generic-worker' workflow",
          inputSchema: { type: "object", properties: {}, additionalProperties: false },
        },
      ])
    );

    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = toolsPath;
    process.env.GH_AW_SAFE_OUTPUTS = outputPath;

    const { server } = createMCPServer();

    expect(server.tools.has("generic_worker")).toBe(true);
  });

  it("should NOT register a call_workflow tool when config.call_workflow is absent", () => {
    const configPath = path.join(tempDir, "config.json");
    const toolsPath = path.join(tempDir, "tools.json");
    const outputPath = path.join(tempDir, "output.jsonl");

    // Config without call-workflow key
    fs.writeFileSync(configPath, JSON.stringify({ "missing-tool": {}, noop: { max: 1 } }));
    fs.writeFileSync(
      toolsPath,
      JSON.stringify([
        {
          name: "generic_worker",
          _call_workflow_name: "generic-worker",
          description: "Call the 'generic-worker' workflow",
          inputSchema: { type: "object", properties: {} },
        },
      ])
    );

    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = toolsPath;
    process.env.GH_AW_SAFE_OUTPUTS = outputPath;

    const { server } = createMCPServer();

    expect(server.tools.has("generic_worker")).toBe(false);
  });

  it("should register all call_workflow tools for multiple configured workers", () => {
    const configPath = path.join(tempDir, "config.json");
    const toolsPath = path.join(tempDir, "tools.json");
    const outputPath = path.join(tempDir, "output.jsonl");

    fs.writeFileSync(
      configPath,
      JSON.stringify({
        "call-workflow": { workflows: ["generic-worker", "specialist-worker"] },
      })
    );
    fs.writeFileSync(
      toolsPath,
      JSON.stringify([
        {
          name: "generic_worker",
          _call_workflow_name: "generic-worker",
          description: "Call the 'generic-worker' workflow",
          inputSchema: { type: "object", properties: {} },
        },
        {
          name: "specialist_worker",
          _call_workflow_name: "specialist-worker",
          description: "Call the 'specialist-worker' workflow",
          inputSchema: { type: "object", properties: {} },
        },
      ])
    );

    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = toolsPath;
    process.env.GH_AW_SAFE_OUTPUTS = outputPath;

    const { server } = createMCPServer();

    expect(server.tools.has("generic_worker")).toBe(true);
    expect(server.tools.has("specialist_worker")).toBe(true);
  });

  it("should register call_workflow and regular tools together from config", () => {
    const configPath = path.join(tempDir, "config.json");
    const toolsPath = path.join(tempDir, "tools.json");
    const outputPath = path.join(tempDir, "output.jsonl");

    fs.writeFileSync(
      configPath,
      JSON.stringify({
        noop: { max: 1 },
        "call-workflow": { workflows: ["generic-worker"] },
      })
    );
    fs.writeFileSync(
      toolsPath,
      JSON.stringify([
        {
          name: "noop",
          description: "No-op tool",
          inputSchema: { type: "object", properties: {} },
        },
        {
          name: "generic_worker",
          _call_workflow_name: "generic-worker",
          description: "Call the 'generic-worker' workflow",
          inputSchema: { type: "object", properties: {} },
        },
      ])
    );

    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = toolsPath;
    process.env.GH_AW_SAFE_OUTPUTS = outputPath;

    const { server } = createMCPServer();

    expect(server.tools.has("noop")).toBe(true);
    expect(server.tools.has("generic_worker")).toBe(true);
  });

  it("should register call_workflow and dispatch_workflow tools independently", () => {
    const configPath = path.join(tempDir, "config.json");
    const toolsPath = path.join(tempDir, "tools.json");
    const outputPath = path.join(tempDir, "output.jsonl");

    fs.writeFileSync(
      configPath,
      JSON.stringify({
        "call-workflow": { workflows: ["generic-worker"] },
        dispatch_workflow: { workflows: ["release-workflow"] },
      })
    );
    fs.writeFileSync(
      toolsPath,
      JSON.stringify([
        {
          name: "generic_worker",
          _call_workflow_name: "generic-worker",
          description: "Call the 'generic-worker' workflow",
          inputSchema: { type: "object", properties: {} },
        },
        {
          name: "release_workflow",
          _workflow_name: "release-workflow",
          description: "Dispatch the 'release-workflow' workflow",
          inputSchema: { type: "object", properties: {} },
        },
      ])
    );

    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = toolsPath;
    process.env.GH_AW_SAFE_OUTPUTS = outputPath;

    const { server } = createMCPServer();

    expect(server.tools.has("generic_worker")).toBe(true);
    expect(server.tools.has("release_workflow")).toBe(true);
  });

  it("should return a call_workflow tool with a callable handler", async () => {
    const configPath = path.join(tempDir, "config.json");
    const toolsPath = path.join(tempDir, "tools.json");
    const outputPath = path.join(tempDir, "output.jsonl");

    fs.writeFileSync(
      configPath,
      JSON.stringify({
        "call-workflow": { workflows: ["generic-worker"] },
      })
    );
    fs.writeFileSync(
      toolsPath,
      JSON.stringify([
        {
          name: "generic_worker",
          _call_workflow_name: "generic-worker",
          description: "Call the 'generic-worker' workflow",
          inputSchema: {
            type: "object",
            properties: { task: { type: "string", description: "Task to perform" } },
            additionalProperties: false,
          },
        },
      ])
    );

    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = configPath;
    process.env.GH_AW_SAFE_OUTPUTS_TOOLS_PATH = toolsPath;
    process.env.GH_AW_SAFE_OUTPUTS = outputPath;

    const { server } = createMCPServer();
    const tool = server.tools.get("generic_worker");

    expect(tool).toBeDefined();
    expect(typeof tool.handler).toBe("function");

    // Calling the handler should produce a valid MCP result and write to the output file
    const result = await tool.handler({ task: "do something" });

    expect(result).toBeDefined();
    expect(result.content).toBeDefined();
    expect(result.isError).toBe(false);

    // Output file should have been written with call_workflow type
    const written = fs.readFileSync(outputPath, "utf8");
    const entry = JSON.parse(written.trim());
    expect(entry.type).toBe("call_workflow");
    expect(entry.workflow_name).toBe("generic-worker");
  });
});
