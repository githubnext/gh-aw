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
    expect(core.setOutput).toHaveBeenCalledWith("call_workflow_payload", JSON.stringify({ environment: "staging", version: "1.2.3" }));
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

  it("should serialise inputs as JSON payload", async () => {
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

    const expectedPayload = JSON.stringify(inputs);
    expect(core.setOutput).toHaveBeenCalledWith("call_workflow_payload", expectedPayload);
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
    expect(core.setOutput).toHaveBeenCalledWith("call_workflow_payload", "{}");
  });
});
