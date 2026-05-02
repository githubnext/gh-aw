// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { main } from "./call_workflow.cjs";

// Mock the core GitHub Actions toolkit
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setOutput: vi.fn(),
};

// Mock GitHub Actions context required by buildAwContext (imported via aw_context.cjs)
global.context = {
  repo: { owner: "test-owner", repo: "test-repo" },
  runId: 1,
  actor: "test-actor",
  eventName: "issues",
  ref: "refs/heads/main",
  payload: {},
};

describe("call_workflow handler factory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should create a handler function", async () => {
    const handler = await main({});
    expect(typeof handler).toBe("function");
  });

  it("should select a workflow and set outputs", async () => {
    const config = {
      workflows: ["spring-boot-bugfix", "frontend-dep-upgrade"],
      max: 1,
    };
    const handler = await main(config);

    const message = {
      type: "call_workflow",
      workflow_name: "spring-boot-bugfix",
      inputs: {
        environment: "staging",
        version: "1.2.3",
      },
    };

    const result = await handler(message);

    expect(result.success).toBe(true);
    expect(result.workflow_name).toBe("spring-boot-bugfix");
    expect(core.setOutput).toHaveBeenCalledWith("call_workflow_name", "spring-boot-bugfix");

    // payload should contain the inputs AND the embedded aw_context
    const payloadCall = core.setOutput.mock.calls.find(([name]) => name === "call_workflow_payload");
    expect(payloadCall).toBeDefined();
    const payload = JSON.parse(payloadCall[1]);
    expect(payload.environment).toBe("staging");
    expect(payload.version).toBe("1.2.3");
    expect(payload).toHaveProperty("aw_context");
  });

  it("should reject unknown workflow names", async () => {
    const config = {
      workflows: ["worker-a", "worker-b"],
      max: 1,
    };
    const handler = await main(config);

    const message = {
      type: "call_workflow",
      workflow_name: "unauthorized-worker",
      inputs: {},
    };

    const result = await handler(message);

    expect(result.success).toBe(false);
    expect(result.error).toContain("not in the allowed workflows list");
    expect(core.setOutput).not.toHaveBeenCalled();
  });

  it("should reject empty workflow names", async () => {
    const config = {
      workflows: ["worker-a"],
      max: 1,
    };
    const handler = await main(config);

    const message = {
      type: "call_workflow",
      workflow_name: "",
      inputs: {},
    };

    const result = await handler(message);

    expect(result.success).toBe(false);
    expect(result.error).toContain("empty");
    expect(core.setOutput).not.toHaveBeenCalled();
  });

  it("should enforce max count limit", async () => {
    const config = {
      workflows: ["worker-a", "worker-b"],
      max: 1,
    };
    const handler = await main(config);

    // First call should succeed
    const result1 = await handler({ workflow_name: "worker-a", inputs: {} });
    expect(result1.success).toBe(true);

    // Second call should fail because max is 1
    const result2 = await handler({ workflow_name: "worker-b", inputs: {} });
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("Max count");
  });

  it("should serialise inputs as JSON payload including embedded aw_context", async () => {
    const config = {
      workflows: ["worker-a"],
      max: 1,
    };
    const handler = await main(config);

    const inputs = {
      package_manager: "npm",
      dry_run: true,
      count: 42,
    };

    await handler({ workflow_name: "worker-a", inputs });

    const payloadCall = core.setOutput.mock.calls.find(([name]) => name === "call_workflow_payload");
    expect(payloadCall).toBeDefined();
    const payload = JSON.parse(payloadCall[1]);
    expect(payload.package_manager).toBe("npm");
    expect(payload.dry_run).toBe(true);
    expect(payload.count).toBe(42);
    // aw_context is always embedded in the payload
    expect(payload).toHaveProperty("aw_context");
  });

  it("should allow any workflow when allowed list is empty", async () => {
    // An empty workflows array is treated as permissive (no restriction).
    // In practice, the compiler always populates this list from frontmatter,
    // so this case should not occur during normal usage.
    const config = {
      workflows: [],
      max: 5,
    };
    const handler = await main(config);

    // When no allowed list, any workflow should pass
    const result = await handler({ workflow_name: "any-workflow", inputs: {} });
    expect(result.success).toBe(true);
  });

  it("should handle missing inputs gracefully", async () => {
    const config = {
      workflows: ["worker-a"],
      max: 1,
    };
    const handler = await main(config);

    const result = await handler({ workflow_name: "worker-a" });

    expect(result.success).toBe(true);
    // Even with no inputs, payload should still have embedded aw_context
    const payloadCall = core.setOutput.mock.calls.find(([name]) => name === "call_workflow_payload");
    expect(payloadCall).toBeDefined();
    const payload = JSON.parse(payloadCall[1]);
    expect(payload).toHaveProperty("aw_context");
  });

  it("should embed aw_context in payload including standard fields", async () => {
    const config = { workflows: ["worker-a"], max: 1 };
    const handler = await main(config);

    const result = await handler({ workflow_name: "worker-a", inputs: {} });

    expect(result.success).toBe(true);
    const payloadCall = core.setOutput.mock.calls.find(([name]) => name === "call_workflow_payload");
    expect(payloadCall).toBeDefined();
    const payload = JSON.parse(payloadCall[1]);

    // aw_context is a JSON string embedded in the payload
    expect(typeof payload.aw_context).toBe("string");
    const awContext = JSON.parse(payload.aw_context);
    expect(awContext).toHaveProperty("repo");
    expect(awContext).toHaveProperty("run_id");
    expect(awContext).toHaveProperty("workflow_id");
    expect(awContext).toHaveProperty("time");
    expect(awContext).toHaveProperty("experiments");
  });

  it("should include experiment assignments in embedded aw_context when GH_AW_EXPERIMENTS_JSON is set", async () => {
    process.env.GH_AW_EXPERIMENTS_JSON = '{"feature1":"A","style":"concise"}';
    const config = { workflows: ["worker-a"], max: 1 };
    const handler = await main(config);

    const result = await handler({ workflow_name: "worker-a", inputs: {} });

    expect(result.success).toBe(true);
    const payloadCall = core.setOutput.mock.calls.find(([name]) => name === "call_workflow_payload");
    expect(payloadCall).toBeDefined();
    const payload = JSON.parse(payloadCall[1]);
    const awContext = JSON.parse(payload.aw_context);
    expect(awContext.experiments).toBe('{"feature1":"A","style":"concise"}');

    delete process.env.GH_AW_EXPERIMENTS_JSON;
  });
});
