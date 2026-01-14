import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

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
};

const mockGithub = {
  graphql: vi.fn(),
};

global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;

describe("assign_to_agent", () => {
  let assignToAgentScript;
  let tempFilePath;

  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_AGENT_DEFAULT;
    delete process.env.GH_AW_AGENT_MAX_COUNT;
    delete process.env.GH_AW_TARGET_REPO;

    // Clear module cache to ensure we get the latest version of assign_agent_helpers
    const helpersPath = require.resolve("./assign_agent_helpers.cjs");
    delete require.cache[helpersPath];

    const scriptPath = path.join(process.cwd(), "assign_to_agent.cjs");
    assignToAgentScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
    }
  });

  it("should handle empty agent output", async () => {
    setAgentOutput({ items: [], errors: [] });
    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No assign_to_agent items found"));
  });

  it("should handle missing agent output", async () => {
    delete process.env.GH_AW_AGENT_OUTPUT;
    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);
    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
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

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    expect(mockGithub.graphql).not.toHaveBeenCalled();
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryCall).toContain("🎭 Staged Mode");
    expect(summaryCall).toContain("Issue:** #42");
    expect(summaryCall).toContain("Agent:** copilot");
  });

  it("should use default agent when not specified", async () => {
    process.env.GH_AW_AGENT_DEFAULT = "copilot";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
        },
      ],
      errors: [],
    });

    // Mock GraphQL responses
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [
              {
                login: "copilot-swe-agent",
                id: "MDQ6VXNlcjE=",
              },
            ],
          },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: {
              nodes: [],
            },
          },
        },
      })
      .mockResolvedValueOnce({
        addAssigneesToAssignable: {
          assignable: {
            assignees: {
              nodes: [{ login: "copilot-swe-agent" }],
            },
          },
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Default agent: copilot");
  });

  it("should respect max count configuration", async () => {
    process.env.GH_AW_AGENT_MAX_COUNT = "2";
    setAgentOutput({
      items: [
        { type: "assign_to_agent", issue_number: 1, agent: "copilot" },
        { type: "assign_to_agent", issue_number: 2, agent: "copilot" },
        { type: "assign_to_agent", issue_number: 3, agent: "copilot" },
      ],
      errors: [],
    });

    // Mock GraphQL responses for 2 assignments
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "MDQ6VXNlcjE=" }],
          },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: { id: "issue-id-1", assignees: { nodes: [] } },
        },
      })
      .mockResolvedValueOnce({
        addAssigneesToAssignable: {
          assignable: { assignees: { nodes: [{ login: "copilot-swe-agent" }] } },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: { id: "issue-id-2", assignees: { nodes: [] } },
        },
      })
      .mockResolvedValueOnce({
        addAssigneesToAssignable: {
          assignable: { assignees: { nodes: [{ login: "copilot-swe-agent" }] } },
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Found 3 agent assignments, but max is 2"));
  });

  it("should reject unsupported agents", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
          agent: "unsupported-agent",
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining('Agent "unsupported-agent" is not supported'));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to assign 1 agent(s)"));
  });

  it("should handle invalid issue numbers", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: -1,
          agent: "copilot",
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Invalid issue_number"));
  });

  it("should handle agent already assigned", async () => {
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

    // Mock GraphQL responses - agent already assigned
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "MDQ6VXNlcjE=" }],
          },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: {
              nodes: [{ id: "MDQ6VXNlcjE=" }],
            },
          },
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("copilot is already assigned to issue #42"));
  });

  it("should handle API errors gracefully", async () => {
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

    const apiError = new Error("API rate limit exceeded");
    mockGithub.graphql.mockRejectedValue(apiError);

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to assign agent"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to assign 1 agent(s)"));
  });

  it("should handle 502 errors as success", async () => {
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

    // Mock successful agent lookup and issue details
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "MDQ6VXNlcjE=" }],
          },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: { nodes: [] },
          },
        },
      })
      .mockRejectedValueOnce({
        response: {
          status: 502,
          url: "https://api.github.com/graphql",
          headers: { "content-type": "text/html" },
          data: "<html>\n<head><title>502 Bad Gateway</title></head>\n<body>\n<center><h1>502 Bad Gateway</h1></center>\n<hr><center>nginx</center>\n</body>\n</html>\n",
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    // Should warn about 502 but treat as success
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Received 502 error from cloud gateway"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Treating 502 error as success"));
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryCall).toContain("Successfully assigned 1 agent(s)");
  });

  it("should handle 502 errors in message as success", async () => {
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

    // Mock successful agent lookup and issue details
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "MDQ6VXNlcjE=" }],
          },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: { nodes: [] },
          },
        },
      })
      .mockRejectedValueOnce(new Error("502 Bad Gateway"));

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    // Should warn about 502 but treat as success
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Received 502 error from cloud gateway"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Treating 502 error as success"));
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should cache agent IDs for multiple assignments", async () => {
    setAgentOutput({
      items: [
        { type: "assign_to_agent", issue_number: 1, agent: "copilot" },
        { type: "assign_to_agent", issue_number: 2, agent: "copilot" },
      ],
      errors: [],
    });

    // Mock GraphQL responses
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "MDQ6VXNlcjE=" }],
          },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: { id: "issue-id-1", assignees: { nodes: [] } },
        },
      })
      .mockResolvedValueOnce({
        addAssigneesToAssignable: {
          assignable: { assignees: { nodes: [{ login: "copilot-swe-agent" }] } },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: { id: "issue-id-2", assignees: { nodes: [] } },
        },
      })
      .mockResolvedValueOnce({
        addAssigneesToAssignable: {
          assignable: { assignees: { nodes: [{ login: "copilot-swe-agent" }] } },
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    // Should only look up agent once (cached for second assignment)
    const graphqlCalls = mockGithub.graphql.mock.calls.filter(call => call[0].includes("suggestedActors"));
    expect(graphqlCalls).toHaveLength(1);
  });

  it("should use target repository when configured", async () => {
    process.env.GH_AW_TARGET_REPO = "other-owner/other-repo";
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

    // Mock GraphQL responses
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        suggestedActors: {
          nodes: [{ login: "copilot-swe-agent", id: "MDQ6VXNlcjE=" }],
        },
      },
    });

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Using target repository: other-owner/other-repo");
  });

  it("should handle invalid max count configuration", async () => {
    process.env.GH_AW_AGENT_MAX_COUNT = "invalid";
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

    await eval(`(async () => { ${assignToAgentScript}; await main(); })()`);

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Invalid max value: invalid"));
  });

  it("should generate permission error summary when appropriate", async () => {
    // Explicitly reset mocks to ensure clean state
    vi.clearAllMocks();
    mockGithub.graphql.mockReset();
    
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

    // Clear module cache for all dependencies AND re-read the script file
    // This ensures we have the latest code when eval executes
    const modulesToClear = [
      "./assign_agent_helpers.cjs",
      "./load_agent_output.cjs",
      "./staged_preview.cjs",
      "./error_helpers.cjs",
    ];
    for (const mod of modulesToClear) {
      const modPath = require.resolve(mod);
      delete require.cache[modPath];
    }
    
    // Re-read the script file to ensure we have the latest version
    const scriptPath = path.join(process.cwd(), "assign_to_agent.cjs");
    const freshScript = fs.readFileSync(scriptPath, "utf8");

    // Simulate permission error during agent assignment mutation (not during getIssueDetails)
    // First call: findAgent succeeds
    // Second call: getIssueDetails succeeds
    // Third call: assignAgentToIssue fails with permission error
    // Fourth call: fallback mutation also fails with permission error
    const permissionError = new Error("Resource not accessible by integration");
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "MDQ6VXNlcjE=" }],
          },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: {
              nodes: [],
            },
          },
        },
      })
      .mockRejectedValueOnce(permissionError)
      .mockRejectedValueOnce(permissionError);

    await eval(`(async () => { ${freshScript}; await main(); })()`);

    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryCall).toContain("Resource not accessible");
    expect(summaryCall).toContain("Permission Requirements");
  });
});
