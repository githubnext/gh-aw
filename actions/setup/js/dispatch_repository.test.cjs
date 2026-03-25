// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { main } from "./dispatch_repository.cjs";

// Mock dependencies
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

global.context = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  ref: "refs/heads/main",
  runId: 12345,
  actor: "octocat",
  eventName: "push",
  payload: {},
};

const mockCreateDispatchEvent = vi.fn().mockResolvedValue({});

global.github = {
  rest: {
    repos: {
      createDispatchEvent: mockCreateDispatchEvent,
      get: vi.fn().mockResolvedValue({ data: { default_branch: "main" } }),
    },
  },
};

describe("dispatch_repository handler factory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    process.env.GITHUB_WORKFLOW_REF = "test-owner/test-repo/.github/workflows/dispatcher.yml@refs/heads/main";
    process.env.GITHUB_RUN_ID = "12345";
    process.env.GITHUB_RUN_ATTEMPT = "1";
  });

  it("should create a handler function", async () => {
    const handler = await main({});
    expect(typeof handler).toBe("function");
  });

  it("should dispatch repository event with valid configuration", async () => {
    const config = {
      tools: {
        "notify-repo": {
          repository: "test-owner/other-repo",
          event_type: "workflow-trigger",
        },
      },
    };
    const handler = await main(config);

    const message = {
      tool_name: "notify-repo",
      inputs: { param1: "value1" },
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.tool_name).toBe("notify-repo");
    expect(result.repository).toBe("test-owner/other-repo");
    expect(result.event_type).toBe("workflow-trigger");
    expect(mockCreateDispatchEvent).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "test-owner",
        repo: "other-repo",
        event_type: "workflow-trigger",
      })
    );
  });

  it("should inject aw_context as a JSON string in client_payload", async () => {
    const config = {
      tools: {
        "notify-repo": {
          repository: "test-owner/other-repo",
          event_type: "workflow-trigger",
        },
      },
    };
    const handler = await main(config);

    const message = {
      tool_name: "notify-repo",
      inputs: {},
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    const clientPayload = result.client_payload;
    expect(clientPayload).toBeDefined();

    // aw_context must be a JSON string (not a plain object)
    const awContextRaw = clientPayload["aw_context"];
    expect(typeof awContextRaw).toBe("string");

    const awContext = JSON.parse(awContextRaw);
    expect(awContext).toHaveProperty("repo");
    expect(awContext).toHaveProperty("run_id");
    expect(awContext).toHaveProperty("workflow_id");
    expect(awContext).toHaveProperty("workflow_call_id");
    expect(awContext).toHaveProperty("time");
    expect(awContext).toHaveProperty("actor");
    expect(awContext).toHaveProperty("event_type");

    // Validate that it's consistent with buildAwContext structure
    expect(awContext.repo).toBe("test-owner/test-repo");
    expect(awContext.workflow_id).toBe("test-owner/test-repo/.github/workflows/dispatcher.yml@refs/heads/main");
  });

  it("should return error when tool_name is missing", async () => {
    const handler = await main({ tools: {} });
    const result = await handler({ tool_name: "" }, {});
    expect(result.success).toBe(false);
    expect(result.error).toContain("tool_name is required");
  });

  it("should return error for unknown tool", async () => {
    const handler = await main({ tools: {} });
    const result = await handler({ tool_name: "unknown-tool" }, {});
    expect(result.success).toBe(false);
    expect(result.error).toContain("not configured");
  });

  it("should return error when event_type is missing", async () => {
    const config = {
      tools: {
        "no-event-type": {
          repository: "test-owner/other-repo",
        },
      },
    };
    const handler = await main(config);
    const result = await handler({ tool_name: "no-event-type" }, {});
    expect(result.success).toBe(false);
    expect(result.error).toContain("event_type is required");
  });

  it("should enforce max count", async () => {
    const config = {
      tools: {
        "notify-repo": {
          repository: "test-owner/other-repo",
          event_type: "workflow-trigger",
          max: 1,
        },
      },
    };
    const handler = await main(config);
    const message = { tool_name: "notify-repo", inputs: {} };

    const result1 = await handler(message, {});
    expect(result1.success).toBe(true);

    const result2 = await handler(message, {});
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("Max count");
  });
});
