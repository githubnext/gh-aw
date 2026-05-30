// @ts-check
import { describe, it, expect, beforeEach, afterEach } from "vitest";
import fs from "fs";
import path from "path";
import { execSync } from "child_process";

const scriptPath = path.join(__dirname, "generate_safe_outputs_tools.cjs");

describe("generate_safe_outputs_tools", () => {
  /** @type {string} */
  let testDir;
  /** @type {string} */
  let toolsSourcePath;
  /** @type {string} */
  let configPath;
  /** @type {string} */
  let toolsMetaPath;
  /** @type {string} */
  let outputPath;

  const sampleSourceTools = [
    {
      name: "create_issue",
      description: "Creates a GitHub issue.",
      inputSchema: {
        type: "object",
        properties: {
          title: { type: "string", description: "Issue title" },
          body: { type: "string", description: "Issue body" },
        },
        required: ["title"],
      },
    },
    {
      name: "add_comment",
      description: "Adds a comment.",
      inputSchema: {
        type: "object",
        properties: {
          body: { type: "string", description: "Comment body" },
        },
        required: ["body"],
      },
    },
    {
      name: "missing_tool",
      description: "Reports a missing tool.",
      inputSchema: { type: "object", properties: {} },
    },
  ];

  beforeEach(() => {
    const testId = Math.random().toString(36).substring(7);
    testDir = `/tmp/test-generate-tools-${testId}`;
    fs.mkdirSync(testDir, { recursive: true });

    toolsSourcePath = path.join(testDir, "safe_outputs_tools.json");
    configPath = path.join(testDir, "config.json");
    toolsMetaPath = path.join(testDir, "tools_meta.json");
    outputPath = path.join(testDir, "tools.json");

    // Write source tools
    fs.writeFileSync(toolsSourcePath, JSON.stringify(sampleSourceTools));
  });

  afterEach(() => {
    fs.rmSync(testDir, { recursive: true, force: true });
  });

  /**
   * Run the generate script with the test env vars.
   * @param {Record<string, string>} [extraEnv] Additional env vars to set.
   * @returns {string} stdout output of the script.
   */
  function runScript(extraEnv = {}) {
    const env = {
      ...process.env,
      GH_AW_SAFE_OUTPUTS_TOOLS_SOURCE_PATH: toolsSourcePath,
      GH_AW_SAFE_OUTPUTS_CONFIG_PATH: configPath,
      GH_AW_SAFE_OUTPUTS_TOOLS_META_PATH: toolsMetaPath,
      GH_AW_SAFE_OUTPUTS_TOOLS_PATH: outputPath,
      ...extraEnv,
    };
    return execSync(`node ${scriptPath}`, { env, encoding: "utf8" });
  }

  it("filters tools based on config keys", () => {
    // Only create_issue and add_comment are enabled
    fs.writeFileSync(configPath, JSON.stringify({ create_issue: { max: 5 }, add_comment: { max: 10 } }));
    fs.writeFileSync(toolsMetaPath, JSON.stringify({ description_suffixes: {}, repo_params: {}, dynamic_tools: [] }));

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    expect(result).toHaveLength(2);
    expect(result.map((/** @type {{name: string}} */ t) => t.name)).toEqual(expect.arrayContaining(["create_issue", "add_comment"]));
    // missing_tool should NOT be included since it's not in config
    expect(result.map((/** @type {{name: string}} */ t) => t.name)).not.toContain("missing_tool");
  });

  it("applies description suffix from tools_meta", () => {
    fs.writeFileSync(configPath, JSON.stringify({ create_issue: { max: 5 } }));
    fs.writeFileSync(
      toolsMetaPath,
      JSON.stringify({
        description_suffixes: {
          create_issue: " CONSTRAINTS: Maximum 5 issue(s) can be created.",
        },
        repo_params: {},
        dynamic_tools: [],
      })
    );

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    const createIssueTool = result.find((/** @type {{name: string}} */ t) => t.name === "create_issue");
    expect(createIssueTool).toBeDefined();
    expect(createIssueTool.description).toContain("Creates a GitHub issue.");
    expect(createIssueTool.description).toContain("CONSTRAINTS: Maximum 5 issue(s) can be created.");
  });

  it("adds repo parameter when specified in tools_meta", () => {
    fs.writeFileSync(configPath, JSON.stringify({ create_issue: { max: 5 } }));
    fs.writeFileSync(
      toolsMetaPath,
      JSON.stringify({
        description_suffixes: {},
        repo_params: {
          create_issue: {
            type: "string",
            description: "Target repository in 'owner/repo' format.",
          },
        },
        dynamic_tools: [],
      })
    );

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    const createIssueTool = result.find((/** @type {{name: string}} */ t) => t.name === "create_issue");
    expect(createIssueTool).toBeDefined();
    expect(createIssueTool.inputSchema.properties.repo).toBeDefined();
    expect(createIssueTool.inputSchema.properties.repo.type).toBe("string");
  });

  it("appends dynamic tools from tools_meta", () => {
    fs.writeFileSync(configPath, JSON.stringify({ create_issue: { max: 1 } }));
    fs.writeFileSync(
      toolsMetaPath,
      JSON.stringify({
        description_suffixes: {},
        repo_params: {},
        dynamic_tools: [
          {
            name: "dispatch_deploy_workflow",
            description: "Dispatches the deploy workflow.",
            inputSchema: { type: "object", properties: { env: { type: "string" } } },
            _workflow_name: "deploy",
          },
        ],
      })
    );

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    expect(result).toHaveLength(2);
    expect(result.map((/** @type {{name: string}} */ t) => t.name)).toContain("dispatch_deploy_workflow");
    const dynamicTool = result.find((/** @type {{name: string, _workflow_name?: string}} */ t) => t._workflow_name === "deploy");
    expect(dynamicTool).toBeDefined();
  });

  it("handles empty config with no enabled tools", () => {
    fs.writeFileSync(configPath, JSON.stringify({}));
    fs.writeFileSync(toolsMetaPath, JSON.stringify({ description_suffixes: {}, repo_params: {}, dynamic_tools: [] }));

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    expect(result).toHaveLength(0);
  });

  it("ignores non-tool config keys when filtering", () => {
    // dispatch_workflow and max_bot_mentions are not tool names in source file
    fs.writeFileSync(
      configPath,
      JSON.stringify({
        create_issue: { max: 1 },
        dispatch_workflow: { workflows: ["deploy"] },
        max_bot_mentions: 5,
      })
    );
    fs.writeFileSync(toolsMetaPath, JSON.stringify({ description_suffixes: {}, repo_params: {}, dynamic_tools: [] }));

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    // Only create_issue should be in filtered static tools
    expect(result.map((/** @type {{name: string}} */ t) => t.name)).not.toContain("dispatch_workflow");
    expect(result.map((/** @type {{name: string}} */ t) => t.name)).not.toContain("max_bot_mentions");
    expect(result.map((/** @type {{name: string}} */ t) => t.name)).toContain("create_issue");
  });

  it("does not modify source tools in memory (deep copy)", () => {
    fs.writeFileSync(configPath, JSON.stringify({ create_issue: { max: 5 } }));
    fs.writeFileSync(
      toolsMetaPath,
      JSON.stringify({
        description_suffixes: { create_issue: " CONSTRAINTS: Maximum 5 issue(s)." },
        repo_params: {
          create_issue: { type: "string", description: "Target repo" },
        },
        dynamic_tools: [],
      })
    );

    // Run twice to ensure source tools are not modified between runs
    runScript();
    const result1 = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    runScript();
    const result2 = JSON.parse(fs.readFileSync(outputPath, "utf8"));

    expect(result1[0].description).toEqual(result2[0].description);
    expect(result1[0].inputSchema.properties.repo).toEqual(result2[0].inputSchema.properties.repo);
  });

  it("exits with error when source tools file is missing", () => {
    fs.writeFileSync(configPath, JSON.stringify({ create_issue: {} }));
    fs.writeFileSync(toolsMetaPath, JSON.stringify({ description_suffixes: {}, repo_params: {}, dynamic_tools: [] }));

    expect(() => runScript({ GH_AW_SAFE_OUTPUTS_TOOLS_SOURCE_PATH: "/nonexistent/path.json" })).toThrow();
  });

  it("exits with error when config file is missing", () => {
    fs.writeFileSync(toolsMetaPath, JSON.stringify({ description_suffixes: {}, repo_params: {}, dynamic_tools: [] }));

    expect(() => runScript({ GH_AW_SAFE_OUTPUTS_CONFIG_PATH: "/nonexistent/config.json" })).toThrow();
  });

  it("works when tools_meta file is missing (graceful fallback)", () => {
    fs.writeFileSync(configPath, JSON.stringify({ create_issue: { max: 1 } }));
    // No tools_meta.json - should still work with fallback to empty meta

    runScript({ GH_AW_SAFE_OUTPUTS_TOOLS_META_PATH: "/nonexistent/tools_meta.json" });

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    expect(result).toHaveLength(1);
    expect(result[0].name).toBe("create_issue");
    // Description should be unchanged (no suffix applied)
    expect(result[0].description).toBe("Creates a GitHub issue.");
  });
});
