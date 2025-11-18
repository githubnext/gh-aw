import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockContext = {
  runId: 12345,
  ref: "refs/heads/main",
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  payload: {
    repository: {
      html_url: "https://github.com/test-owner/test-repo",
    },
  },
};

const mockGithub = {
  rest: {
    actions: {
      createWorkflowDispatch: vi.fn(),
    },
  },
};

// Set up global mocks
global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;

describe("dispatch_workflow", () => {
  let dispatchWorkflowScript;
  let tempFilePath;

  // Helper function to set agent output
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset mocks before each test
    vi.clearAllMocks();
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_ALLOWED_WORKFLOWS;

    // Clean up temp file if it exists
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }

    // Read the script content
    const scriptPath = path.join(process.cwd(), "dispatch_workflow.cjs");
    dispatchWorkflowScript = fs.readFileSync(scriptPath, "utf8");
    dispatchWorkflowScript = dispatchWorkflowScript.replace("export {};", "");
  });

  it("should handle empty agent output", async () => {
    setAgentOutput("");

    await eval(`(async () => { ${dispatchWorkflowScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    expect(mockCore.setOutput).toHaveBeenCalledWith("workflow_name", "");
    expect(mockCore.setOutput).toHaveBeenCalledWith("workflow_ref", "");
  });

  it("should handle missing dispatch_workflow items", async () => {
    setAgentOutput({
      items: [{ type: "create_issue", title: "Test", body: "Test body" }],
      errors: [],
    });
    process.env.GH_AW_ALLOWED_WORKFLOWS = "workflow.yml";

    await eval(`(async () => { ${dispatchWorkflowScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No dispatch-workflow items found in agent output");
  });

  it("should fail when no allowed workflows configured", async () => {
    setAgentOutput({
      items: [{ type: "dispatch_workflow", workflow: "test-workflow.yml" }],
      errors: [],
    });

    await eval(`(async () => { ${dispatchWorkflowScript} })()`);

    expect(mockCore.setFailed).toHaveBeenCalledWith("No allowed workflows configured. Set GH_AW_ALLOWED_WORKFLOWS environment variable.");
  });

  it("should dispatch a valid workflow", async () => {
    setAgentOutput({
      items: [
        {
          type: "dispatch_workflow",
          workflow: "test-workflow.yml",
          ref: "main",
          inputs: { environment: "production", version: "1.0.0" },
        },
      ],
      errors: [],
    });
    process.env.GH_AW_ALLOWED_WORKFLOWS = "test-workflow.yml,other-workflow.yml";

    mockGithub.rest.actions.createWorkflowDispatch.mockResolvedValue({});

    await eval(`(async () => { ${dispatchWorkflowScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Found 1 dispatch-workflow item(s)");
    expect(mockGithub.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.yml",
      ref: "main",
      inputs: { environment: "production", version: "1.0.0" },
    });
    expect(mockCore.setOutput).toHaveBeenCalledWith("workflow_name", "test-workflow.yml");
    expect(mockCore.setOutput).toHaveBeenCalledWith("workflow_ref", "main");
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
  });

  it("should fail when workflow is not in allowlist", async () => {
    setAgentOutput({
      items: [
        {
          type: "dispatch_workflow",
          workflow: "unauthorized-workflow.yml",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_ALLOWED_WORKFLOWS = "test-workflow.yml,other-workflow.yml";

    await eval(`(async () => { ${dispatchWorkflowScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith("Workflow 'unauthorized-workflow.yml' is not in the allowed workflows list");
    expect(mockCore.setFailed).toHaveBeenCalledWith(
      "Workflow 'unauthorized-workflow.yml' is not allowed. Allowed workflows: test-workflow.yml, other-workflow.yml"
    );
    expect(mockGithub.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
  });

  it("should use default ref when not provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "dispatch_workflow",
          workflow: "test-workflow.yml",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_ALLOWED_WORKFLOWS = "test-workflow.yml";

    mockGithub.rest.actions.createWorkflowDispatch.mockResolvedValue({});

    await eval(`(async () => { ${dispatchWorkflowScript} })()`);

    expect(mockGithub.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.yml",
      ref: "refs/heads/main",
    });
  });

  it("should handle dispatch errors gracefully", async () => {
    setAgentOutput({
      items: [
        {
          type: "dispatch_workflow",
          workflow: "test-workflow.yml",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_ALLOWED_WORKFLOWS = "test-workflow.yml";

    const error = new Error("Workflow not found");
    mockGithub.rest.actions.createWorkflowDispatch.mockRejectedValue(error);

    await eval(`(async () => { ${dispatchWorkflowScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith("Failed to dispatch workflow 'test-workflow.yml': Workflow not found");
    expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to dispatch workflow 'test-workflow.yml': Workflow not found");
  });

  it("should show staged preview when staged mode enabled", async () => {
    setAgentOutput({
      items: [
        {
          type: "dispatch_workflow",
          workflow: "test-workflow.yml",
          ref: "develop",
          inputs: { environment: "staging" },
        },
      ],
      errors: [],
    });
    process.env.GH_AW_ALLOWED_WORKFLOWS = "test-workflow.yml";
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    await eval(`(async () => { ${dispatchWorkflowScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Found 1 dispatch-workflow item(s)");
    expect(mockGithub.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
  });

  it("should dispatch multiple workflows", async () => {
    setAgentOutput({
      items: [
        {
          type: "dispatch_workflow",
          workflow: "workflow1.yml",
          ref: "main",
        },
        {
          type: "dispatch_workflow",
          workflow: "workflow2.yml",
          ref: "develop",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_ALLOWED_WORKFLOWS = "workflow1.yml,workflow2.yml";

    mockGithub.rest.actions.createWorkflowDispatch.mockResolvedValue({});

    await eval(`(async () => { ${dispatchWorkflowScript} })()`);

    expect(mockGithub.rest.actions.createWorkflowDispatch).toHaveBeenCalledTimes(2);
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
  });
});
