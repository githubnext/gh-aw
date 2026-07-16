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

  it("adds required fields when specified in tools_meta", () => {
    fs.writeFileSync(configPath, JSON.stringify({ create_issue: { max: 5 } }));
    fs.writeFileSync(
      toolsMetaPath,
      JSON.stringify({
        description_suffixes: {},
        repo_params: {},
        dynamic_tools: [],
        required_field_additions: {
          create_issue: ["temporary_id"],
        },
      })
    );

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    const createIssueTool = result.find((/** @type {{name: string}} */ t) => t.name === "create_issue");
    expect(createIssueTool).toBeDefined();
    expect(createIssueTool.inputSchema.required).toEqual(expect.arrayContaining(["title", "temporary_id"]));
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

  it("dynamically marks add_comment discussion support as enabled when discussions:true", () => {
    fs.writeFileSync(configPath, JSON.stringify({ add_comment: { discussions: true } }));
    fs.writeFileSync(toolsMetaPath, JSON.stringify({ description_suffixes: {}, repo_params: {}, dynamic_tools: [] }));

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    const addCommentTool = result.find((/** @type {{name: string, description: string}} */ t) => t.name === "add_comment");
    expect(addCommentTool).toBeDefined();
    expect(addCommentTool.description).toContain("Discussion comments are enabled for this workflow");
    expect(addCommentTool.description).toContain("Supports reply_to_id for discussion threading.");
  });

  it("dynamically marks add_comment discussion support as disabled by default", () => {
    fs.writeFileSync(configPath, JSON.stringify({ add_comment: { max: 1 } }));
    fs.writeFileSync(
      toolsMetaPath,
      JSON.stringify({
        description_suffixes: { add_comment: " Supports reply_to_id for discussion threading." },
        repo_params: {},
        dynamic_tools: [],
      })
    );

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    const addCommentTool = result.find((/** @type {{name: string, description: string}} */ t) => t.name === "add_comment");
    expect(addCommentTool).toBeDefined();
    expect(addCommentTool.description).toContain("Discussion comments are disabled for this workflow");
    expect(addCommentTool.description).not.toContain("Supports reply_to_id for discussion threading.");
  });

  it("adds issue intent suffix for issue tools when explicitly enabled", () => {
    fs.writeFileSync(
      toolsSourcePath,
      JSON.stringify([
        { name: "set_issue_type", description: "Sets issue type.", inputSchema: { type: "object", properties: {} } },
        { name: "set_issue_field", description: "Sets issue field.", inputSchema: { type: "object", properties: {} } },
        { name: "add_labels", description: "Adds labels.", inputSchema: { type: "object", properties: {} } },
        { name: "close_issue", description: "Closes issue.", inputSchema: { type: "object", properties: {} } },
        { name: "assign_to_user", description: "Assigns users.", inputSchema: { type: "object", properties: {} } },
        { name: "assign_to_agent", description: "Assigns agent.", inputSchema: { type: "object", properties: {} } },
        { name: "create_issue", description: "Creates a GitHub issue.", inputSchema: { type: "object", properties: {} } },
      ])
    );
    fs.writeFileSync(
      configPath,
      JSON.stringify({
        set_issue_type: { issue_intent: true },
        set_issue_field: { issue_intent: true },
        add_labels: { issue_intent: true },
        close_issue: { issue_intent: true },
        assign_to_user: { issue_intent: true },
        assign_to_agent: { issue_intent: true },
        create_issue: {},
      })
    );
    fs.writeFileSync(toolsMetaPath, JSON.stringify({ description_suffixes: {}, repo_params: {}, dynamic_tools: [] }));

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    const intentSuffix = "INTENT: Include rationale (string, max 280 chars) and confidence (string, exactly one of: LOW, MEDIUM, HIGH) with each call.";
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "set_issue_type").description).toContain(intentSuffix);
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "set_issue_field").description).toContain(intentSuffix);
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "add_labels").description).toContain(intentSuffix);
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "close_issue").description).toContain(intentSuffix);
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "assign_to_user").description).toContain(intentSuffix);
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "assign_to_agent").description).toContain(intentSuffix);
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "create_issue").description).not.toContain(intentSuffix);
  });

  it("adds issue intent suffix even when unrelated runtime features are present", () => {
    fs.writeFileSync(
      toolsSourcePath,
      JSON.stringify([
        { name: "set_issue_type", description: "Sets issue type.", inputSchema: { type: "object", properties: {} } },
        { name: "set_issue_field", description: "Sets issue field.", inputSchema: { type: "object", properties: {} } },
        { name: "add_labels", description: "Adds labels.", inputSchema: { type: "object", properties: {} } },
      ])
    );
    fs.writeFileSync(configPath, JSON.stringify({ set_issue_type: { issue_intent: true }, set_issue_field: { issue_intent: true }, add_labels: { issue_intent: true } }));
    fs.writeFileSync(toolsMetaPath, JSON.stringify({ description_suffixes: {}, repo_params: {}, dynamic_tools: [] }));

    runScript({ GH_AW_RUNTIME_FEATURES: "other\nanother=true" });

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    const intentSuffix = "INTENT: Include rationale (string, max 280 chars) and confidence (string, exactly one of: LOW, MEDIUM, HIGH) with each call.";
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "set_issue_type").description).toContain(intentSuffix);
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "set_issue_field").description).toContain(intentSuffix);
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "add_labels").description).toContain(intentSuffix);
  });

  it("omits issue intent suffix by default and when explicitly disabled per tool", () => {
    fs.writeFileSync(
      toolsSourcePath,
      JSON.stringify([
        { name: "close_issue", description: "Closes issue.", inputSchema: { type: "object", properties: {} } },
        { name: "assign_to_user", description: "Assigns users.", inputSchema: { type: "object", properties: {} } },
      ])
    );
    fs.writeFileSync(configPath, JSON.stringify({ close_issue: { issue_intent: false }, assign_to_user: {} }));
    fs.writeFileSync(toolsMetaPath, JSON.stringify({ description_suffixes: {}, repo_params: {}, dynamic_tools: [] }));

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    const intentSuffix = "INTENT: Include rationale (string, max 280 chars) and confidence (string, exactly one of: LOW, MEDIUM, HIGH) with each call.";
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "close_issue").description).not.toContain(intentSuffix);
    expect(result.find((/** @type {{name: string, description: string}} */ t) => t.name === "assign_to_user").description).not.toContain(intentSuffix);
  });

  it("reflects required/optional/absent intent fields per tool configuration", () => {
    fs.writeFileSync(
      toolsSourcePath,
      JSON.stringify([
        {
          name: "close_issue",
          description: "Closes issue.",
          inputSchema: {
            type: "object",
            properties: { body: { type: "string" }, rationale: { type: "string" }, confidence: { type: "string" }, suggest: { type: "boolean" } },
            required: ["body"],
          },
        },
        {
          name: "assign_to_user",
          description: "Assigns users.",
          inputSchema: {
            type: "object",
            properties: { issue_number: { type: "number" }, rationale: { type: "string" }, confidence: { type: "string" }, suggest: { type: "boolean" } },
            required: ["issue_number"],
          },
        },
        {
          name: "assign_to_agent",
          description: "Assigns agent.",
          inputSchema: {
            type: "object",
            properties: { issue_number: { type: "number" }, rationale: { type: "string" }, confidence: { type: "string" }, suggest: { type: "boolean" } },
            required: ["issue_number"],
          },
        },
      ])
    );
    fs.writeFileSync(
      configPath,
      JSON.stringify({
        close_issue: { issue_intent: true },
        assign_to_user: {},
        assign_to_agent: { issue_intent: false },
      })
    );
    fs.writeFileSync(
      toolsMetaPath,
      JSON.stringify({
        description_suffixes: {},
        repo_params: {},
        dynamic_tools: [],
        required_field_additions: { close_issue: ["rationale", "confidence"] },
      })
    );

    runScript();

    const result = JSON.parse(fs.readFileSync(outputPath, "utf8"));
    const closeIssue = result.find((/** @type {{name: string, inputSchema: {properties: Record<string, unknown>, required: string[]}}} */ t) => t.name === "close_issue");
    const assignToUser = result.find((/** @type {{name: string, inputSchema: {properties: Record<string, unknown>, required: string[]}}} */ t) => t.name === "assign_to_user");
    const assignToAgent = result.find((/** @type {{name: string, inputSchema: {properties: Record<string, unknown>, required: string[]}}} */ t) => t.name === "assign_to_agent");

    expect(closeIssue.inputSchema.properties).toHaveProperty("rationale");
    expect(closeIssue.inputSchema.properties).toHaveProperty("confidence");
    expect(closeIssue.inputSchema.required).toEqual(expect.arrayContaining(["rationale", "confidence"]));

    expect(assignToUser.inputSchema.properties).toHaveProperty("rationale");
    expect(assignToUser.inputSchema.properties).toHaveProperty("confidence");
    expect(assignToUser.inputSchema.required).not.toContain("rationale");
    expect(assignToUser.inputSchema.required).not.toContain("confidence");

    expect(assignToAgent.inputSchema.properties).not.toHaveProperty("rationale");
    expect(assignToAgent.inputSchema.properties).not.toHaveProperty("confidence");
    expect(assignToAgent.inputSchema.properties).not.toHaveProperty("suggest");
    expect(assignToAgent.inputSchema.required).not.toContain("rationale");
    expect(assignToAgent.inputSchema.required).not.toContain("confidence");
  });
});
