import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

// Set up global mocks before importing the module
global.core = mockCore;

const mockContext = {
  runId: 12345,
  runNumber: 42,
  sha: "abc123def456",
  ref: "refs/heads/main",
  actor: "octocat",
  eventName: "push",
  repo: { owner: "github", repo: "my-repo" },
};

describe("generate_aw_info.cjs", () => {
  let main;
  let awInfoPath;

  beforeEach(async () => {
    vi.clearAllMocks();

    // Create /tmp/gh-aw directory if it doesn't exist
    if (!fs.existsSync("/tmp/gh-aw")) {
      fs.mkdirSync("/tmp/gh-aw", { recursive: true });
    }
    awInfoPath = "/tmp/gh-aw/aw_info.json";

    // Set default env vars for compile-time values
    process.env.GH_AW_INFO_ENGINE_ID = "copilot";
    process.env.GH_AW_INFO_ENGINE_NAME = "GitHub Copilot CLI";
    process.env.GH_AW_INFO_MODEL = "gpt-4";
    process.env.GH_AW_INFO_VERSION = "";
    process.env.GH_AW_INFO_AGENT_VERSION = "0.0.419";
    process.env.GH_AW_INFO_CLI_VERSION = "";
    process.env.GH_AW_INFO_WORKFLOW_NAME = "my-workflow";
    process.env.GH_AW_INFO_EXPERIMENTAL = "false";
    process.env.GH_AW_INFO_SUPPORTS_TOOLS_ALLOWLIST = "true";
    process.env.GH_AW_INFO_STAGED = "false";
    process.env.GH_AW_INFO_ALLOWED_DOMAINS = "[]";
    process.env.GH_AW_INFO_FIREWALL_ENABLED = "false";
    process.env.GH_AW_INFO_AWF_VERSION = "";
    process.env.GH_AW_INFO_AWMG_VERSION = "";
    process.env.GH_AW_INFO_FIREWALL_TYPE = "";

    // Dynamic import to get fresh module state
    const module = await import("./generate_aw_info.cjs");
    main = module.main;
  });

  afterEach(() => {
    if (fs.existsSync(awInfoPath)) {
      fs.unlinkSync(awInfoPath);
    }
    // Clean up env vars
    const keysToDelete = Object.keys(process.env).filter(k => k.startsWith("GH_AW_INFO_"));
    for (const key of keysToDelete) {
      delete process.env[key];
    }
  });

  it("should write aw_info.json with values from env vars and context", async () => {
    await main(mockCore, mockContext);

    expect(fs.existsSync(awInfoPath)).toBe(true);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));

    expect(awInfo.engine_id).toBe("copilot");
    expect(awInfo.engine_name).toBe("GitHub Copilot CLI");
    expect(awInfo.model).toBe("gpt-4");
    expect(awInfo.workflow_name).toBe("my-workflow");
    expect(awInfo.experimental).toBe(false);
    expect(awInfo.supports_tools_allowlist).toBe(true);
    expect(awInfo.run_id).toBe(12345);
    expect(awInfo.run_number).toBe(42);
    expect(awInfo.sha).toBe("abc123def456");
    expect(awInfo.repository).toBe("github/my-repo");
    expect(awInfo.actor).toBe("octocat");
    expect(awInfo.event_name).toBe("push");
    expect(awInfo.staged).toBe(false);
    expect(awInfo.firewall_enabled).toBe(false);
    expect(awInfo.created_at).toBeTruthy();
  });

  it("should set model output", async () => {
    await main(mockCore, mockContext);

    expect(mockCore.setOutput).toHaveBeenCalledWith("model", "gpt-4");
  });

  it("should include cli_version only when GH_AW_INFO_CLI_VERSION is set", async () => {
    process.env.GH_AW_INFO_CLI_VERSION = "1.2.3";
    await main(mockCore, mockContext);

    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.cli_version).toBe("1.2.3");
  });

  it("should not include cli_version when GH_AW_INFO_CLI_VERSION is empty", async () => {
    process.env.GH_AW_INFO_CLI_VERSION = "";
    await main(mockCore, mockContext);

    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.cli_version).toBeUndefined();
  });

  it("should parse allowed domains from JSON env var", async () => {
    process.env.GH_AW_INFO_ALLOWED_DOMAINS = '["github.com","api.github.com"]';
    await main(mockCore, mockContext);

    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.allowed_domains).toEqual(["github.com", "api.github.com"]);
  });

  it("should warn and use empty array for invalid allowed_domains JSON", async () => {
    process.env.GH_AW_INFO_ALLOWED_DOMAINS = "not-valid-json";
    await main(mockCore, mockContext);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to parse GH_AW_INFO_ALLOWED_DOMAINS"));
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.allowed_domains).toEqual([]);
  });

  it("should warn for missing required context fields", async () => {
    const incompleteContext = { runId: 1 };
    await main(mockCore, incompleteContext);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("context.runNumber is not set"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("context.sha is not set"));
  });

  it("should set firewall info from env vars", async () => {
    process.env.GH_AW_INFO_FIREWALL_ENABLED = "true";
    process.env.GH_AW_INFO_AWF_VERSION = "v0.23.0";
    process.env.GH_AW_INFO_FIREWALL_TYPE = "squid";
    await main(mockCore, mockContext);

    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.firewall_enabled).toBe(true);
    expect(awInfo.awf_version).toBe("v0.23.0");
    expect(awInfo.steps.firewall).toBe("squid");
  });

  it("should fail when a numeric context field contains non-numeric data", async () => {
    const maliciousContext = {
      ...mockContext,
      payload: {
        issue: { number: "42; DROP TABLE users" },
      },
    };

    await expect(main(mockCore, maliciousContext)).rejects.toThrow();
    expect(mockCore.setFailed).toHaveBeenCalled();
  });

  it("should pass context validation when numeric fields are valid integers", async () => {
    const validContext = {
      ...mockContext,
      payload: {
        issue: { number: 42 },
        pull_request: { number: 100 },
      },
    };

    await main(mockCore, validContext);
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("✅ All context variables validated successfully"));
  });

  it("should call generateWorkflowOverview to write step summary", async () => {
    await main(mockCore, mockContext);

    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should reject aw_context that is not a plain object", async () => {
    const contextWithArrayInput = {
      ...mockContext,
      payload: { inputs: { aw_context: JSON.stringify([1, 2, 3]) } },
    };
    await main(mockCore, contextWithArrayInput);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.context).toBeUndefined();
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("must be a JSON object"));
  });

  it("should reject aw_context with nested objects", async () => {
    const contextWithNested = {
      ...mockContext,
      payload: {
        inputs: {
          aw_context: JSON.stringify({
            repo: "org/repo",
            run_id: "123",
            workflow_id: "my-workflow",
            nested: { bad: "field" },
          }),
        },
      },
    };
    await main(mockCore, contextWithNested);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.context).toBeUndefined();
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("nested objects"));
  });

  it("should reject aw_context missing required fields", async () => {
    const contextMissingFields = {
      ...mockContext,
      payload: {
        inputs: {
          aw_context: JSON.stringify({ actor: "octocat" }),
        },
      },
    };
    await main(mockCore, contextMissingFields);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.context).toBeUndefined();
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("missing required fields"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("run_id"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("repo"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("workflow_id"));
  });

  it("should accept valid aw_context and set awInfo.context", async () => {
    const validContext = {
      repo: "org/repo",
      run_id: "12345",
      workflow_id: "org/repo/.github/workflows/dispatcher.yml@refs/heads/main",
      workflow_call_id: "12345-1",
      time: new Date().toISOString(),
      actor: "octocat",
      event_type: "issues",
      item_type: "issue",
      item_number: "42",
      comment_id: "",
    };
    const contextWithValid = {
      ...mockContext,
      payload: { inputs: { aw_context: JSON.stringify(validContext) } },
    };
    await main(mockCore, contextWithValid);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.context).toEqual(validContext);
    expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("aw_context"));
  });

  it("should accept valid aw_context with comment_id and item_number for comment events", async () => {
    const validContext = {
      repo: "org/repo",
      run_id: "12345",
      workflow_id: "org/repo/.github/workflows/dispatcher.yml@refs/heads/main",
      workflow_call_id: "12345-1",
      time: new Date().toISOString(),
      actor: "octocat",
      event_type: "issue_comment",
      item_type: "issue",
      item_number: "7",
      comment_id: "99001122",
    };
    const contextWithComment = {
      ...mockContext,
      payload: { inputs: { aw_context: JSON.stringify(validContext) } },
    };
    await main(mockCore, contextWithComment);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.context).toEqual(validContext);
    expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("aw_context"));
  });

  it("should accept valid aw_context for pull_request_review events", async () => {
    const validContext = {
      repo: "org/repo",
      run_id: "12345",
      workflow_id: "org/repo/.github/workflows/dispatcher.yml@refs/heads/main",
      workflow_call_id: "12345-1",
      time: new Date().toISOString(),
      actor: "octocat",
      event_type: "pull_request_review",
      item_type: "pull_request",
      item_number: "100",
      comment_id: "55667788",
    };
    const contextWithReview = {
      ...mockContext,
      payload: { inputs: { aw_context: JSON.stringify(validContext) } },
    };
    await main(mockCore, contextWithReview);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.context).toEqual(validContext);
    expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("aw_context"));
  });

  it("should accept valid aw_context for check_run events", async () => {
    const validContext = {
      repo: "org/repo",
      run_id: "12345",
      workflow_id: "org/repo/.github/workflows/dispatcher.yml@refs/heads/main",
      workflow_call_id: "12345-1",
      time: new Date().toISOString(),
      actor: "octocat",
      event_type: "check_run",
      item_type: "check_run",
      item_number: "7654321",
      comment_id: "",
    };
    const contextWithCheckRun = {
      ...mockContext,
      payload: { inputs: { aw_context: JSON.stringify(validContext) } },
    };
    await main(mockCore, contextWithCheckRun);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.context).toEqual(validContext);
    expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("aw_context"));
  });

  it("should accept valid aw_context for check_suite events", async () => {
    const validContext = {
      repo: "org/repo",
      run_id: "12345",
      workflow_id: "org/repo/.github/workflows/dispatcher.yml@refs/heads/main",
      workflow_call_id: "12345-1",
      time: new Date().toISOString(),
      actor: "octocat",
      event_type: "check_suite",
      item_type: "check_suite",
      item_number: "9988776",
      comment_id: "",
    };
    const contextWithCheckSuite = {
      ...mockContext,
      payload: { inputs: { aw_context: JSON.stringify(validContext) } },
    };
    await main(mockCore, contextWithCheckSuite);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.context).toEqual(validContext);
    expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("aw_context"));
  });

  it("should accept valid aw_context with comment_node_id for discussion_comment events", async () => {
    const validContext = {
      repo: "org/repo",
      run_id: "12345",
      workflow_id: "org/repo/.github/workflows/dispatcher.yml@refs/heads/main",
      workflow_call_id: "12345-1",
      time: new Date().toISOString(),
      actor: "octocat",
      event_type: "discussion_comment",
      item_type: "discussion",
      item_number: "240",
      comment_id: "77889900",
      comment_node_id: "DC_kwDOParentComment456",
    };
    const contextWithDiscussion = {
      ...mockContext,
      payload: { inputs: { aw_context: JSON.stringify(validContext) } },
    };
    await main(mockCore, contextWithDiscussion);
    const awInfo = JSON.parse(fs.readFileSync(awInfoPath, "utf8"));
    expect(awInfo.context).toEqual(validContext);
    expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("aw_context"));
  });
});
