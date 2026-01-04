// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { main } from "./dispatch_workflow.cjs";
import fs from "fs";

// Mock dependencies
global.core = {
  info: vi.fn(),
  setOutput: vi.fn(),
  setFailed: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
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
    },
  },
};

describe("dispatch_workflow", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_DISPATCH_WORKFLOW_ALLOWED;
    delete process.env.GH_AW_DISPATCH_WORKFLOW_MAX_COUNT;
  });

  it("should handle no agent output environment variable", async () => {
    await main();
    expect(core.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
    expect(core.setOutput).toHaveBeenCalledWith("count", "");
  });

  it("should handle empty agent output", async () => {
    process.env.GH_AW_AGENT_OUTPUT = "/tmp/test-empty.json";
    vi.spyOn(fs, "readFileSync").mockReturnValue("");

    await main();
    expect(core.info).toHaveBeenCalledWith("Agent output content is empty");
  });

  it("should dispatch workflows in non-staged mode", async () => {
    const agentOutput = {
      items: [
        {
          type: "dispatch_workflow",
          workflow_name: "test-workflow",
          inputs: {
            param1: "value1",
            param2: 42,
          },
        },
      ],
      errors: [],
    };

    process.env.GH_AW_AGENT_OUTPUT = "/tmp/test-dispatch.json";
    process.env.GH_AW_DISPATCH_WORKFLOW_ALLOWED = '["test-workflow"]';
    process.env.GH_AW_DISPATCH_WORKFLOW_MAX_COUNT = "5";
    vi.spyOn(fs, "readFileSync").mockReturnValue(JSON.stringify(agentOutput));

    await main();

    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: expect.any(String), // Accept any string ref
      inputs: {
        param1: "value1",
        param2: "42",
      },
    });

    expect(core.setOutput).toHaveBeenCalledWith("count", "1");
    expect(core.info).toHaveBeenCalledWith("✓ Successfully dispatched workflow: test-workflow");
  });

  it("should preview workflows in staged mode", async () => {
    const agentOutput = {
      items: [
        {
          type: "dispatch_workflow",
          workflow_name: "test-workflow",
          inputs: {
            param1: "value1",
          },
        },
      ],
      errors: [],
    };

    process.env.GH_AW_AGENT_OUTPUT = "/tmp/test-staged.json";
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
    vi.spyOn(fs, "readFileSync").mockReturnValue(JSON.stringify(agentOutput));

    await main();

    expect(github.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
    expect(core.summary.addRaw).toHaveBeenCalled();
    expect(core.summary.write).toHaveBeenCalled();
  });

  it("should reject workflows not in allowed list", async () => {
    const agentOutput = {
      items: [
        {
          type: "dispatch_workflow",
          workflow_name: "unauthorized-workflow",
          inputs: {},
        },
      ],
      errors: [],
    };

    process.env.GH_AW_AGENT_OUTPUT = "/tmp/test-unauthorized.json";
    process.env.GH_AW_DISPATCH_WORKFLOW_ALLOWED = '["test-workflow", "other-workflow"]';
    vi.spyOn(fs, "readFileSync").mockReturnValue(JSON.stringify(agentOutput));

    await main();

    expect(core.setFailed).toHaveBeenCalledWith(expect.stringContaining('Workflow "unauthorized-workflow" is not in the allowed workflows list'));
    expect(github.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
  });

  it("should enforce max count", async () => {
    const agentOutput = {
      items: [
        { type: "dispatch_workflow", workflow_name: "workflow1", inputs: {} },
        { type: "dispatch_workflow", workflow_name: "workflow2", inputs: {} },
        { type: "dispatch_workflow", workflow_name: "workflow3", inputs: {} },
      ],
      errors: [],
    };

    process.env.GH_AW_AGENT_OUTPUT = "/tmp/test-max-count.json";
    process.env.GH_AW_DISPATCH_WORKFLOW_MAX_COUNT = "2";
    vi.spyOn(fs, "readFileSync").mockReturnValue(JSON.stringify(agentOutput));

    await main();

    expect(core.setFailed).toHaveBeenCalledWith("Too many dispatch-workflow items: 3 (max: 2)");
    expect(github.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
  });

  it("should handle dispatch errors", async () => {
    const agentOutput = {
      items: [
        {
          type: "dispatch_workflow",
          workflow_name: "missing-workflow",
          inputs: {},
        },
      ],
      errors: [],
    };

    process.env.GH_AW_AGENT_OUTPUT = "/tmp/test-error.json";
    process.env.GH_AW_DISPATCH_WORKFLOW_ALLOWED = '["missing-workflow"]';
    vi.spyOn(fs, "readFileSync").mockReturnValue(JSON.stringify(agentOutput));

    github.rest.actions.createWorkflowDispatch.mockRejectedValueOnce(new Error("Request failed with status code 404"));

    await main();

    expect(core.setFailed).toHaveBeenCalledWith(expect.stringContaining('Workflow "missing-workflow.lock.yml" not found'));
  });
});
