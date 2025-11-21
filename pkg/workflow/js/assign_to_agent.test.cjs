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
  graphql: vi.fn(),
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

// Helper to flush pending promises
const flushPromises = () => new Promise(resolve => setImmediate(resolve));

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
    // Recreate mocks with fresh vi.fn() to ensure vitest tracking works
    mockCore.debug = vi.fn();
    mockCore.info = vi.fn();
    mockCore.warning = vi.fn();
    mockCore.error = vi.fn();
    mockCore.setFailed = vi.fn();
    mockCore.setOutput = vi.fn();
    mockCore.summary.addRaw = vi.fn().mockReturnValue({ write: vi.fn().mockResolvedValue() });
    mockGithub.graphql = vi.fn();
    mockExec.exec = vi.fn().mockResolvedValue(0);

    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_AGENT_DEFAULT;
    delete process.env.GH_AW_AGENT_MAX_COUNT;
    delete process.env.GH_AW_TARGET_REPO;

    // Set up GitHub token for gh CLI
    process.env.GH_TOKEN = "test-token";

    // Reassign to global to ensure eval sees updated mocks
    global.core = mockCore;
    global.github = mockGithub;
    global.exec = mockExec;

    // Set up default GraphQL mock implementation
    mockGithub.graphql.mockImplementation(query => {
      if (query.includes("suggestedActors")) {
        return Promise.resolve({
          repository: {
            suggestedActors: {
              nodes: [
                { login: "copilot-swe-agent", __typename: "Bot", id: "bot-copilot" },
                { login: "claude-swe-agent", __typename: "Bot", id: "bot-claude" },
                { login: "codex-swe-agent", __typename: "Bot", id: "bot-codex" },
              ],
            },
          },
        });
      }
      if (query.includes("issue(number:")) {
        return Promise.resolve({
          repository: {
            issue: {
              id: "issue-id-test",
              assignees: { nodes: [] },
            },
          },
        });
      }
      if (query.includes("replaceActorsForAssignable")) {
        return Promise.resolve({
          replaceActorsForAssignable: {
            __typename: "Issue",
          },
        });
      }
      return Promise.reject(new Error("Unexpected GraphQL query"));
    });

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
    await flushPromises();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No assign_to_agent items found"));
  });

  it("should handle missing agent output", async () => {
    delete process.env.GH_AW_AGENT_OUTPUT;

    await eval(`(async () => { ${assignToAgentScript} })()`);
    await flushPromises();

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

    await eval(`(async () => { ${assignToAgentScript} })()`);
    await flushPromises();

    expect(mockGithub.graphql).toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully assigned"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_agents", "42:copilot");
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
  });

  it("should assign agent successfully with custom agent", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
          agent: "claude",
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript} })()`);
    await flushPromises();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully assigned"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_agents", "42:claude");
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

    let graphqlCallCount = 0;
    mockGithub.graphql.mockImplementation(query => {
      graphqlCallCount++;
      if (query.includes("suggestedActors")) {
        return Promise.resolve({
          repository: {
            suggestedActors: {
              nodes: [{ login: "copilot-swe-agent", __typename: "Bot", id: "bot-id" }],
            },
          },
        });
      }
      if (query.includes("issue(number:")) {
        return Promise.resolve({
          repository: {
            issue: { id: `issue-id-${graphqlCallCount}`, assignees: { nodes: [] } },
          },
        });
      }
      if (query.includes("replaceActorsForAssignable")) {
        return Promise.resolve({
          replaceActorsForAssignable: {
            __typename: "Issue",
          },
        });
      }
      return Promise.reject(new Error("Unexpected GraphQL query"));
    });

    await eval(`(async () => { ${assignToAgentScript} })()`);
    await flushPromises();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Found 3 agent assignments, but max is 2"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully assigned"));
    const outputCall = mockCore.setOutput.mock.calls.find(call => call[0] === "assigned_agents");
    expect(outputCall).toBeDefined();
    expect(outputCall[1]).toContain("1:copilot");
    expect(outputCall[1]).toContain("2:copilot");
  });

  it("should use default agent from environment", async () => {
    process.env.GH_AW_AGENT_DEFAULT = "codex";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript} })()`);
    await flushPromises();

    expect(mockCore.info).toHaveBeenCalledWith("Default agent: codex");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully assigned"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_agents", "42:codex");
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

    // Override default mock to verify correct repo is used
    mockGithub.graphql.mockImplementation(query => {
      if (query.includes("suggestedActors")) {
        expect(query).toContain("other-owner");
        expect(query).toContain("other-repo");
        return Promise.resolve({
          repository: {
            suggestedActors: {
              nodes: [{ login: "copilot-swe-agent", __typename: "Bot", id: "bot-id" }],
            },
          },
        });
      }
      if (query.includes("issue(number:")) {
        expect(query).toContain("other-owner");
        expect(query).toContain("other-repo");
        return Promise.resolve({
          repository: {
            issue: { id: "issue-id-456", assignees: { nodes: [] } },
          },
        });
      }
      if (query.includes("replaceActorsForAssignable")) {
        return Promise.resolve({
          replaceActorsForAssignable: {
            __typename: "Issue",
          },
        });
      }
      return Promise.reject(new Error("Unexpected GraphQL query"));
    });

    await eval(`(async () => { ${assignToAgentScript} })()`);
    await flushPromises();

    expect(mockCore.info).toHaveBeenCalledWith("Using target repository: other-owner/other-repo");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully assigned"));
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

    mockGithub.graphql.mockRejectedValue(new Error("GraphQL API error"));

    await eval(`(async () => { ${assignToAgentScript} })()`);
    await flushPromises();

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to"));
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
    await flushPromises();

    expect(mockCore.error).toHaveBeenCalledWith("Invalid issue_number: invalid");
  });

  it("should handle multiple successful assignments", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 1,
          agent: "copilot",
        },
        {
          type: "assign_to_agent",
          issue_number: 2,
          agent: "claude",
        },
      ],
      errors: [],
    });

    process.env.GH_AW_AGENT_MAX_COUNT = "2"; // Allow processing multiple items

    await eval(`(async () => { ${assignToAgentScript} })()`);
    await flushPromises();

    expect(mockCore.setOutput).toHaveBeenCalledWith("assigned_agents", "1:copilot\n2:claude");
  });
});
