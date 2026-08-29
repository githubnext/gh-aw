import { describe, expect, it, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { parseCopilotSDKToolConfig, buildCopilotSDKSessionToolConfig, isReservedSDKPermission } = require("./copilot_sdk_tool_config.cjs");
const { runWithCopilotSDK } = require("./copilot_sdk_session.cjs");

function validToolConfig(overrides = {}) {
  return {
    version: overrides.version ?? 1,
    capabilities: {
      bash: false,
      edit: false,
      webFetch: true,
      webSearch: false,
      mcp: true,
      cliProxy: false,
      ...(overrides.capabilities ?? {}),
    },
    permissions: {
      allowedTools: ["read", "safeoutputs", "web_fetch"],
      ...(overrides.permissions ?? {}),
    },
    explicitlyDisabledTools: overrides.explicitlyDisabledTools ?? ["bash", "cli-proxy", "edit"],
  };
}

class FakeToolSet {
  items = [];

  addBuiltIn(names) {
    for (const name of Array.isArray(names) ? names : [names]) this.items.push(`builtin:${name}`);
    return this;
  }

  addCustom(name) {
    this.items.push(`custom:${name}`);
    return this;
  }

  addMcp(name) {
    this.items.push(`mcp:${name}`);
    return this;
  }

  toArray() {
    return [...this.items];
  }
}

const fakeSDKTools = {
  ToolSet: FakeToolSet,
  BuiltInTools: {
    Isolated: ["ask_user", "task_complete", "exit_plan_mode", "task", "read_agent", "write_agent", "list_agents", "skill"],
  },
  defineTool: (name, config) => ({ name, ...config }),
};

describe("parseCopilotSDKToolConfig", () => {
  it("fails closed when the compiler contract is absent", () => {
    expect(() => parseCopilotSDKToolConfig(undefined)).toThrow("is required");
    expect(() => parseCopilotSDKToolConfig("")).toThrow("is required");
  });

  it("normalizes a valid compiler contract", () => {
    expect(parseCopilotSDKToolConfig(JSON.stringify(validToolConfig()))).toEqual(validToolConfig());
  });

  it("treats null explicitlyDisabledTools as absent", () => {
    expect(parseCopilotSDKToolConfig(JSON.stringify({ ...validToolConfig(), explicitlyDisabledTools: null })).explicitlyDisabledTools).toEqual([]);
  });

  it.each([
    ["invalid JSON", "{", "must be valid JSON"],
    ["unsupported version", JSON.stringify({ ...validToolConfig(), version: 2 }), "unsupported"],
    ["missing capability", JSON.stringify({ ...validToolConfig(), capabilities: { bash: false } }), "capabilities.edit"],
    ["duplicate permission", JSON.stringify(validToolConfig({ permissions: { allowedTools: ["read", "read"] } })), "duplicate"],
    ["empty permissions", JSON.stringify(validToolConfig({ permissions: { allowedTools: [] } })), "must not be empty"],
  ])("fails closed for %s", (_name, value, message) => {
    expect(() => parseCopilotSDKToolConfig(value)).toThrow(message);
  });

  it.each([
    ["bash", { capabilities: { bash: true }, permissions: { allowedTools: ["read", "safeoutputs", "web_fetch"] } }, "bash visibility"],
    ["edit", { capabilities: { edit: true }, permissions: { allowedTools: ["read", "safeoutputs", "web_fetch"] } }, "edit visibility"],
    ["web_fetch", { capabilities: { webFetch: false }, permissions: { allowedTools: ["read", "safeoutputs", "web_fetch"] } }, "web_fetch visibility"],
    ["MCP", { capabilities: { mcp: false }, permissions: { allowedTools: ["read", "safeoutputs", "web_fetch"] } }, "MCP permissions"],
  ])("rejects %s catalog/permission drift", (_name, partial, message) => {
    const value = validToolConfig({
      capabilities: { ...validToolConfig().capabilities, ...partial.capabilities },
      permissions: partial.permissions,
    });
    expect(() => parseCopilotSDKToolConfig(JSON.stringify(value))).toThrow(message);
  });

  it("rejects an explicitly disabled tool that resolves visible", () => {
    const value = validToolConfig({
      capabilities: { ...validToolConfig().capabilities, bash: true },
      permissions: { allowedTools: ["read", "safeoutputs", "shell", "web_fetch"] },
    });
    expect(() => parseCopilotSDKToolConfig(JSON.stringify(value))).toThrow("explicitly disabled bash");
  });
});

describe("isReservedSDKPermission", () => {
  it("distinguishes built-in SDK permissions from MCP server grants", () => {
    expect(isReservedSDKPermission("read")).toBe(true);
    expect(isReservedSDKPermission("read(pkg/**)")).toBe(true);
    expect(isReservedSDKPermission("shell(git:*)")).toBe(true);
    expect(isReservedSDKPermission("web_fetch")).toBe(true);
    expect(isReservedSDKPermission("web_fetch(get)")).toBe(false);
    expect(isReservedSDKPermission("github")).toBe(false);
  });
});

describe("buildCopilotSDKSessionToolConfig", () => {
  it("keeps neutral and isolated controls while excluding ask_user and the bash family", () => {
    const config = buildCopilotSDKSessionToolConfig(validToolConfig(), fakeSDKTools);
    expect(config.availableTools.toArray()).toEqual([
      "builtin:task_complete",
      "builtin:exit_plan_mode",
      "builtin:task",
      "builtin:read_agent",
      "builtin:write_agent",
      "builtin:list_agents",
      "builtin:skill",
      "builtin:view",
      "builtin:rg",
      "builtin:glob",
      "builtin:sql",
      "mcp:*",
      "custom:web_fetch",
    ]);
    for (const forbiddenTool of ["builtin:bash", "builtin:read_bash", "builtin:stop_bash", "builtin:list_bash", "builtin:apply_patch"]) {
      expect(config.availableTools.toArray()).not.toContain(forbiddenTool);
    }
    expect(config.tools).toHaveLength(1);
    expect(config.tools[0]).toMatchObject({
      name: "web_fetch",
      overridesBuiltInTool: true,
      defer: "never",
    });
  });

  it("admits all bash lifecycle tools only when shell permission is enabled", () => {
    const toolConfig = validToolConfig({
      capabilities: { ...validToolConfig().capabilities, bash: true },
      permissions: { allowedTools: ["read", "safeoutputs", "shell(git:*)", "web_fetch"] },
      explicitlyDisabledTools: ["cli-proxy", "edit"],
    });
    const config = buildCopilotSDKSessionToolConfig(toolConfig, fakeSDKTools);
    expect(config.availableTools.toArray()).toEqual(expect.arrayContaining(["builtin:bash", "builtin:read_bash", "builtin:stop_bash", "builtin:list_bash"]));
  });

  it("preserves legacy SDK behavior only when the compiler contract is absent", () => {
    expect(buildCopilotSDKSessionToolConfig(null, {})).toEqual({});
  });

  it("fails closed when required SDK filtering APIs are missing", () => {
    expect(() => buildCopilotSDKSessionToolConfig(validToolConfig(), {})).toThrow("ToolSet and BuiltInTools.Isolated");
  });

  it("fails closed when the SDK does not preserve the web_fetch override contract", () => {
    expect(() =>
      buildCopilotSDKSessionToolConfig(validToolConfig(), {
        ...fakeSDKTools,
        defineTool: (name, config) => ({ name, ...config, overridesBuiltInTool: false }),
      })
    ).toThrow("web_fetch override contract");
  });
});

describe("runWithCopilotSDK compiler-owned catalog", () => {
  it("passes one filtered catalog to the parent session for inherited subagent enforcement", async () => {
    const createSession = vi.fn().mockResolvedValue({
      sessionId: "session-tool-contract",
      on: () => {},
      sendAndWait: vi.fn().mockResolvedValue({ data: { content: "ok" } }),
      disconnect: vi.fn().mockResolvedValue(undefined),
    });
    class FakeCopilotClient {
      start = vi.fn().mockResolvedValue(undefined);
      createSession = createSession;
      stop = vi.fn().mockResolvedValue(undefined);
    }
    const sdkModule = {
      ...fakeSDKTools,
      CopilotClient: FakeCopilotClient,
      RuntimeConnection: { forUri: vi.fn(() => ({})) },
      approveAll: () => ({ kind: "approve-once" }),
    };

    const result = await runWithCopilotSDK({
      sdkUri: "http://127.0.0.1:3002",
      prompt: "test prompt",
      logger: () => {},
      permissionConfig: validToolConfig().permissions,
      toolConfig: validToolConfig(),
      sdkModule,
    });

    expect(result.exitCode).toBe(0);
    const sessionConfig = createSession.mock.calls[0][0];
    for (const forbiddenTool of ["builtin:bash", "builtin:read_bash", "builtin:stop_bash", "builtin:list_bash", "builtin:apply_patch"]) {
      expect(sessionConfig.availableTools.toArray()).not.toContain(forbiddenTool);
    }
    expect(sessionConfig.tools.map(tool => tool.name)).toEqual(["web_fetch"]);
  });
});
