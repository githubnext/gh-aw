// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { main } from "./dispatch_workflow.cjs";

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
};

global.github = {
  rest: {
    actions: {
      createWorkflowDispatch: vi.fn().mockResolvedValue({}),
      listRepoWorkflows: vi.fn().mockResolvedValue({
        data: {
          workflows: [
            { path: ".github/workflows/test-workflow.lock.yml" },
            { path: ".github/workflows/other-workflow.yml" },
          ],
        },
      }),
    },
  },
};

describe("dispatch_workflow handler factory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    process.env.GITHUB_REF = "refs/heads/main";
  });

  it("should create a handler function", async () => {
    const handler = await main({});
    expect(typeof handler).toBe("function");
  });

  it("should dispatch workflows with valid configuration", async () => {
    const config = {
      workflows: ["test-workflow"],
      max: 5,
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "test-workflow",
      inputs: {
        param1: "value1",
        param2: 42,
      },
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.workflow_name).toBe("test-workflow");
    // Should resolve workflow file first
    expect(github.rest.actions.listRepoWorkflows).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
    });
    // Then dispatch to the resolved file
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: expect.any(String),
      inputs: {
        param1: "value1",
        param2: "42",
      },
    });
  });

  it("should reject workflows not in allowed list", async () => {
    const config = {
      workflows: ["allowed-workflow"],
      max: 5,
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "unauthorized-workflow",
      inputs: {},
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not in the allowed workflows list");
    expect(github.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
  });

  it("should enforce max count", async () => {
    const config = {
      workflows: ["workflow1", "workflow2"],
      max: 1,
    };
    const handler = await main(config);

    // First message should succeed
    const message1 = {
      type: "dispatch_workflow",
      workflow_name: "workflow1",
      inputs: {},
    };
    const result1 = await handler(message1, {});
    expect(result1.success).toBe(true);

    // Second message should be rejected due to max count
    const message2 = {
      type: "dispatch_workflow",
      workflow_name: "workflow2",
      inputs: {},
    };
    const result2 = await handler(message2, {});
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("max count");
  });

  it("should handle empty workflow name", async () => {
    const handler = await main({});

    const message = {
      type: "dispatch_workflow",
      workflow_name: "",
      inputs: {},
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("empty");
    expect(github.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
  });

  it("should handle dispatch errors", async () => {
    const handler = await main({ workflows: ["missing-workflow"] });

    // Mock listRepoWorkflows to return workflows without missing-workflow
    github.rest.actions.listRepoWorkflows.mockResolvedValueOnce({
      data: {
        workflows: [
          { path: ".github/workflows/other-workflow.yml" },
        ],
      },
    });

    const message = {
      type: "dispatch_workflow",
      workflow_name: "missing-workflow",
      inputs: {},
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should convert input values to strings", async () => {
    const handler = await main({ workflows: ["test-workflow"] });

    const message = {
      type: "dispatch_workflow",
      workflow_name: "test-workflow",
      inputs: {
        string: "hello",
        number: 42,
        boolean: true,
        object: { key: "value" },
        null: null,
        undefined: undefined,
      },
    };

    await handler(message, {});

    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        inputs: {
          string: "hello",
          number: "42",
          boolean: "true",
          object: '{"key":"value"}',
          null: "",
          undefined: "",
        },
      })
    );
  });
});
