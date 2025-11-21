import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

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

const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  eventName: "issues",
  payload: {
    issue: {
      number: 123,
    },
  },
};

const mockGithub = {
  rest: {
    issues: {
      get: vi.fn(),
      createComment: vi.fn(),
    },
  },
};

const mockExec = {
  exec: vi.fn().mockResolvedValue(0),
};

// Set up global mocks before importing the module
global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;
global.exec = mockExec;

describe("assign_to_agent", () => {
  let assignToAgentScript;
  let tempFilePath;

  // Helper function to set agent output via file
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
    delete process.env.GH_AW_AGENT_DEFAULT;
    delete process.env.GH_AW_AGENT_MAX_COUNT;
    delete process.env.GH_AW_TARGET_REPO;
    
    // Set up GitHub token for gh CLI
    process.env.GH_TOKEN = "test-token";

    // Reset exec mock to successful state
    mockExec.exec.mockResolvedValue(0);

    // Read the script content
    const scriptPath = path.join(process.cwd(), "assign_to_agent.cjs");
    assignToAgentScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up temp file
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
    }
  });

  it("should handle empty agent output", async () => {
    setAgentOutput({
      items: [],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No assign_to_agent items found"));
  });

  it("should handle missing agent output", async () => {
    delete process.env.GH_AW_AGENT_OUTPUT;

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
  });

  it("should assign agent successfully with default agent", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
        },
      ],
      errors: [],
    });

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        title: "Test Issue",
        number: 42,
      },
    });

    mockGithub.rest.issues.createComment.mockResolvedValue({});

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 42,
    });

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 42,
      body: "@copilot has been assigned to this issue.",
    });

    expect(mockCore.info).toHaveBeenCalledWith('Successfully assigned agent "copilot" to issue #42');
    expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_agents", "42:copilot");
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
  });

  it("should assign agent successfully with custom agent", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
          agent: "my-custom-agent",
        },
      ],
      errors: [],
    });

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        title: "Test Issue",
        number: 42,
      },
    });

    mockGithub.rest.issues.createComment.mockResolvedValue({});

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 42,
      body: "@my-custom-agent has been assigned to this issue.",
    });

    expect(mockCore.info).toHaveBeenCalledWith('Successfully assigned agent "my-custom-agent" to issue #42');
    expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_agents", "42:my-custom-agent");
  });

  it("should handle staged mode correctly", async () => {
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
          agent: "copilot",
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript} })()`);

    // In staged mode, should not call the API
    expect(mockGithub.rest.issues.get).not.toHaveBeenCalled();
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();

    // Should generate preview
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryCall).toContain("🎭 Staged Mode");
    expect(summaryCall).toContain("Issue:** #42");
    expect(summaryCall).toContain("Agent:** copilot");
  });

  it("should respect max count configuration", async () => {
    process.env.GH_AW_AGENT_MAX_COUNT = "2";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 1,
        },
        {
          type: "assign_to_agent",
          issue_number: 2,
        },
        {
          type: "assign_to_agent",
          issue_number: 3,
        },
      ],
      errors: [],
    });

    mockGithub.rest.issues.get.mockResolvedValue({
      data: { title: "Test Issue" },
    });
    mockGithub.rest.issues.createComment.mockResolvedValue({});

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Found 3 agent assignments, but max is 2"));
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledTimes(2);
  });

  it("should use default agent from environment", async () => {
    process.env.GH_AW_AGENT_DEFAULT = "my-default-agent";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
        },
      ],
      errors: [],
    });

    mockGithub.rest.issues.get.mockResolvedValue({
      data: { title: "Test Issue" },
    });
    mockGithub.rest.issues.createComment.mockResolvedValue({});

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Default agent: my-default-agent");
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 42,
      body: "@my-default-agent has been assigned to this issue.",
    });
  });

  it("should handle target repository configuration", async () => {
    process.env.GH_AW_TARGET_REPO = "other-owner/other-repo";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
        },
      ],
      errors: [],
    });

    mockGithub.rest.issues.get.mockResolvedValue({
      data: { title: "Test Issue" },
    });
    mockGithub.rest.issues.createComment.mockResolvedValue({});

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Using target repository: other-owner/other-repo");
    expect(mockGithub.rest.issues.get).toHaveBeenCalledWith({
      owner: "other-owner",
      repo: "other-repo",
      issue_number: 42,
    });
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "other-owner",
      repo: "other-repo",
      issue_number: 42,
      body: "@copilot has been assigned to this issue.",
    });
  });

  it("should handle API errors gracefully", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
        },
      ],
      errors: [],
    });

    mockGithub.rest.issues.get.mockRejectedValue(new Error("Issue not found"));

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to assign agent"));
    expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to assign 1 agent(s)");
  });



  it("should handle invalid issue numbers", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: "invalid",
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith("Invalid issue_number: invalid");
  });

  it("should handle multiple successful assignments", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 1,
          agent: "agent-1",
        },
        {
          type: "assign_to_agent",
          issue_number: 2,
          agent: "agent-2",
        },
      ],
      errors: [],
    });

    process.env.GH_AW_AGENT_MAX_COUNT = "2"; // Allow processing multiple items

    mockGithub.rest.issues.get.mockResolvedValue({
      data: { title: "Test Issue" },
    });
    mockGithub.rest.issues.createComment.mockResolvedValue({});

    await eval(`(async () => { ${assignToAgentScript} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledTimes(2);
    expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_agents", "1:agent-1\n2:agent-2");
  });
});
