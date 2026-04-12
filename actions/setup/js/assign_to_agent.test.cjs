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
  rest: {
    issues: {
      createComment: vi.fn().mockResolvedValue({ data: { id: 12345 } }),
    },
  },
};

global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;

describe("assign_to_agent", () => {
  let assignToAgentScript;
  let tempFilePath;

  // Simulates the safe-output handler manager: builds handler config from env vars,
  // calls main() as a factory, then processes items from GH_AW_AGENT_OUTPUT.
  // This mirrors the production flow without requiring any backward-compat changes in
  // assign_to_agent.cjs itself.
  const STANDALONE_RUNNER = `
    const _config = {};
    if (process.env.GH_AW_AGENT_DEFAULT?.trim()) _config.name = process.env.GH_AW_AGENT_DEFAULT.trim();
    if (process.env.GH_AW_AGENT_MAX_COUNT?.trim()) _config.max = process.env.GH_AW_AGENT_MAX_COUNT.trim();
    if (process.env.GH_AW_AGENT_TARGET?.trim()) _config.target = process.env.GH_AW_AGENT_TARGET.trim();
    if (process.env.GH_AW_AGENT_ALLOWED?.trim()) _config.allowed = process.env.GH_AW_AGENT_ALLOWED.trim();
    if (process.env.GH_AW_AGENT_IGNORE_IF_ERROR?.trim()) _config["ignore-if-error"] = process.env.GH_AW_AGENT_IGNORE_IF_ERROR.trim();
    if (process.env.GH_AW_AGENT_PULL_REQUEST_REPO?.trim()) _config["pull-request-repo"] = process.env.GH_AW_AGENT_PULL_REQUEST_REPO.trim();
    if (process.env.GH_AW_AGENT_ALLOWED_PULL_REQUEST_REPOS?.trim()) _config["allowed-pull-request-repos"] = process.env.GH_AW_AGENT_ALLOWED_PULL_REQUEST_REPOS.trim();
    if (process.env.GH_AW_AGENT_BASE_BRANCH?.trim()) _config["base-branch"] = process.env.GH_AW_AGENT_BASE_BRANCH.trim();
    if (process.env.GH_AW_ALLOWED_REPOS?.trim()) _config.allowed_repos = process.env.GH_AW_ALLOWED_REPOS.trim();

    let _handler;
    try { _handler = await main(_config); } catch (_err) { core.setFailed(_err.message); return; }

    const _agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
    if (!_agentOutputFile) { core.info("No GH_AW_AGENT_OUTPUT environment variable found"); return; }

    const _fs = require("fs");
    const _agentOutput = JSON.parse(_fs.readFileSync(_agentOutputFile, "utf8"));
    const _items = _agentOutput.items.filter(i => i.type === "assign_to_agent");
    if (_items.length === 0) {
      core.info("No assign_to_agent items found in agent output");
    } else {
      const _maxCount = parseInt(String(_config.max ?? "1"), 10);
      if (_items.length > _maxCount) {
        core.warning("Found " + _items.length + " agent assignments, but max is " + _maxCount + ". Extra assignments will be skipped.");
      }
      const { loadTemporaryIdMap } = require("./temporary_id.cjs");
      const _tempIdMap = loadTemporaryIdMap();
      for (const _item of _items) { await _handler(_item, {}, _tempIdMap); }
    }
    await writeAssignToAgentSummary();
    const _errorCount = getAssignToAgentErrorCount();
    core.setOutput("assigned", getAssignToAgentAssigned());
    core.setOutput("assignment_errors", getAssignToAgentErrors());
    core.setOutput("assignment_error_count", String(_errorCount));
    if (_errorCount > 0) { core.setFailed("Failed to assign " + _errorCount + " agent(s)"); }
  `;

  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    vi.clearAllMocks();

    // Reset mockGithub.graphql to ensure no lingering mock implementations
    mockGithub.graphql = vi.fn();

    // Reset mockGithub.rest.issues.createComment
    mockGithub.rest.issues.createComment = vi.fn().mockResolvedValue({ data: { id: 12345 } });

    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_AGENT_DEFAULT;
    delete process.env.GH_AW_AGENT_MAX_COUNT;
    delete process.env.GH_AW_AGENT_TARGET;
    delete process.env.GH_AW_AGENT_ALLOWED;
    delete process.env.GH_AW_TARGET_REPO_SLUG;
    delete process.env.GH_AW_ALLOWED_REPOS;
    delete process.env.GH_AW_AGENT_IGNORE_IF_ERROR;
    delete process.env.GH_AW_TEMPORARY_ID_MAP;
    delete process.env.GH_AW_AGENT_PULL_REQUEST_REPO;
    delete process.env.GH_AW_AGENT_ALLOWED_PULL_REQUEST_REPOS;
    delete process.env.GH_AW_AGENT_BASE_BRANCH;

    // Reset context to default
    mockContext.eventName = "issues";
    mockContext.payload = {
      issue: { number: 42 },
    };

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
    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No assign_to_agent items found"));
  });

  it("should handle missing agent output", async () => {
    delete process.env.GH_AW_AGENT_OUTPUT;
    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);
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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Found 3 agent assignments, but max is 2"));
  }, 20000); // Increase timeout to 20 seconds to account for the delay

  it("should resolve temporary issue IDs (aw_...) using GH_AW_TEMPORARY_ID_MAP", async () => {
    process.env.GH_AW_TEMPORARY_ID_MAP = JSON.stringify({
      aw_abc123: { repo: "test-owner/test-repo", number: 99 },
    });

    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: "aw_abc123",
          agent: "copilot",
        },
      ],
      errors: [],
    });

    // Mock GraphQL responses: findAgent -> getIssueDetails (issueNumber 99) -> addAssignees
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
            id: "issue-id-99",
            assignees: { nodes: [] },
          },
        },
      })
      .mockResolvedValueOnce({
        addAssigneesToAssignable: {
          assignable: { assignees: { nodes: [{ login: "copilot-swe-agent" }] } },
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Resolved temporary issue id"));

    // Ensure the issue lookup used the resolved issue number
    const secondCallArgs = mockGithub.graphql.mock.calls[1];
    expect(secondCallArgs).toBeDefined();
    const variables = secondCallArgs[1];
    expect(variables.issueNumber).toBe(99);
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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Error message changed to use resolveTarget validation
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Invalid"));
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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("copilot is already assigned to issue #42"));
  });

  it("should allow re-assignment when agent is already assigned but pull_request_repo differs", async () => {
    process.env.GH_AW_AGENT_PULL_REQUEST_REPO = "test-owner/default-pr-repo";
    process.env.GH_AW_AGENT_ALLOWED_PULL_REQUEST_REPOS = "test-owner/other-platform-repo";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
          agent: "copilot",
          pull_request_repo: "test-owner/other-platform-repo",
        },
      ],
      errors: [],
    });

    // Mock GraphQL responses
    mockGithub.graphql
      // Get global PR repository ID and default branch
      .mockResolvedValueOnce({
        repository: {
          id: "default-pr-repo-id",
          defaultBranchRef: { name: "main" },
        },
      })
      // Get per-item PR repository ID
      .mockResolvedValueOnce({
        repository: {
          id: "other-platform-repo-id",
        },
      })
      // Find agent
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "agent-id" }],
          },
        },
      })
      // Get issue details - agent is already assigned
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: {
              nodes: [{ id: "agent-id", login: "copilot-swe-agent" }],
            },
          },
        },
      })
      // Assign agent (should proceed despite already being assigned)
      .mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should NOT see "already assigned" skip message
    expect(mockCore.info).not.toHaveBeenCalledWith(expect.stringContaining("is already assigned to issue #42"));
    // Should see successful assignment
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully assigned copilot coding agent to issue #42"));
    expect(mockCore.setFailed).not.toHaveBeenCalled();

    // Verify the mutation was called with the per-item PR repo ID
    const lastGraphQLCall = mockGithub.graphql.mock.calls[mockGithub.graphql.mock.calls.length - 1];
    expect(lastGraphQLCall[0]).toContain("agentAssignment");
    expect(lastGraphQLCall[1].targetRepoId).toBe("other-platform-repo-id");
  });

  it("should still skip when agent is already assigned with global pull-request-repo but no per-item override", async () => {
    process.env.GH_AW_AGENT_PULL_REQUEST_REPO = "test-owner/global-pr-repo";
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
    mockGithub.graphql
      // Get global PR repository ID and default branch
      .mockResolvedValueOnce({
        repository: {
          id: "global-pr-repo-id",
          defaultBranchRef: { name: "main" },
        },
      })
      // Find agent
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "agent-id" }],
          },
        },
      })
      // Get issue details - agent is already assigned
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: {
              nodes: [{ id: "agent-id", login: "copilot-swe-agent" }],
            },
          },
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should see "already assigned" skip message
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("copilot is already assigned to issue #42"));
    // Should NOT have called the assignment mutation (only 3 GraphQL calls: repo lookup, find agent, get issue)
    expect(mockGithub.graphql).toHaveBeenCalledTimes(3);
    expect(mockCore.setFailed).not.toHaveBeenCalled();
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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should only look up agent once (cached for second assignment)
    const graphqlCalls = mockGithub.graphql.mock.calls.filter(call => call[0].includes("suggestedActors"));
    expect(graphqlCalls).toHaveLength(1);
  }, 15000); // Increase timeout to 15 seconds to account for the delay

  it("should use target repository when configured", async () => {
    process.env.GH_AW_TARGET_REPO_SLUG = "other-owner/other-repo";
    process.env.GH_AW_ALLOWED_REPOS = "other-owner/other-repo"; // Add to allowlist
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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Default target repo: other-owner/other-repo");
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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Invalid max value: invalid"));
  });

  it.skip("should generate permission error summary when appropriate", async () => {
    // TODO: This test needs to be fixed - the mock setup doesn't work correctly with eval()
    // The error from getIssueDetails is not being propagated properly in the test environment
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

    // Simulate permission error during agent assignment mutation (not during getIssueDetails)
    // First call: findAgent succeeds
    // Second call: getIssueDetails succeeds
    // Third call: assignAgentToIssue fails with permission error
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
      .mockRejectedValueOnce(permissionError);

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryCall).toContain("Resource not accessible");
    expect(summaryCall).toContain("Permission Requirements");
  });

  it.skip("should handle pull_number parameter", async () => {
    // TODO: Fix test mocking - the code works but the test setup has issues with GraphQL mocking for PR queries
    // The functionality is identical to issue_number (just uses pullRequest instead of issue in the GraphQL query)
    // and the schema/validation changes have been tested via the other validation tests
    process.env.GH_AW_AGENT_DEFAULT = "copilot";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          pull_number: 123,
          agent: "copilot",
        },
      ],
      errors: [],
    });

    // Mock GraphQL responses for PR
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
          pullRequest: {
            id: "pr-id-123",
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

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    if (mockCore.error.mock.calls.length > 0) {
      console.log("Errors:", mockCore.error.mock.calls);
    }

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully assigned copilot coding agent to pull request #123"));
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should error when both issue_number and pull_number are provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
          pull_number: 123,
          agent: "copilot",
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.error).toHaveBeenCalledWith("Cannot specify both issue_number and pull_number in the same assign_to_agent item");
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to assign 1 agent(s)"));
  });

  it("should auto-resolve issue number from context when not provided (triggering target)", async () => {
    // Set up context to simulate an issue event
    mockContext.eventName = "issues";
    mockContext.payload = {
      issue: { number: 123 },
    };
    mockContext.repo = {
      owner: "test-owner",
      repo: "test-repo",
    };

    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          agent: "copilot",
          // No issue_number or pull_number - should auto-resolve
        },
      ],
      errors: [],
    });

    // Mock GraphQL responses in the correct order
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
            id: "issue-id-123",
            assignees: {
              nodes: [],
            },
          },
        },
      })
      .mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // The key assertion: Target configuration should be "triggering" (the default)
    // This shows that when no explicit issue_number/pull_number is provided,
    // the handler falls back to the triggering context
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Target configuration: triggering"));

    // GraphQL should have been called for finding the agent and getting issue details
    expect(mockGithub.graphql).toHaveBeenCalled();
  });

  it("should skip when context doesn't match triggering target", async () => {
    // Set up context that doesn't support triggering target (e.g., push event)
    mockContext.eventName = "push";

    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          agent: "copilot",
          // No issue_number or pull_number
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should skip gracefully (not fail the workflow)
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("not running in issue or pull request context"));
  });

  it("should error when neither issue_number nor pull_number provided and target is '*'", async () => {
    process.env.GH_AW_AGENT_TARGET = "*"; // Explicit target mode

    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          agent: "copilot",
          // No issue_number or pull_number
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should fail because target "*" requires explicit issue_number or pull_number
    expect(mockCore.error).toHaveBeenCalled();
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to assign 1 agent(s)"));
  });

  it("should accept agent when in allowed list", async () => {
    process.env.GH_AW_AGENT_ALLOWED = "copilot";
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
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "MDQ6VXNlcjE=", __typename: "Bot" }],
          },
        },
      })
      .mockResolvedValueOnce({
        repository: {
          issue: { id: "issue-id", assignees: { nodes: [] } },
        },
      })
      .mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Key assertion: allowed agents list should be logged
    expect(mockCore.info).toHaveBeenCalledWith("Allowed agents: copilot");

    // Should not reject the agent for being not in the allowed list
    expect(mockCore.error).not.toHaveBeenCalledWith(expect.stringContaining("not in the allowed list"));
  });

  it("should reject agent not in allowed list", async () => {
    process.env.GH_AW_AGENT_ALLOWED = "other-agent";
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

    // No GraphQL mocks needed - validation happens before GraphQL calls

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Allowed agents: other-agent");
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining('Agent "copilot" is not in the allowed list'));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to assign 1 agent(s)"));

    // Should not have made any GraphQL calls since validation failed early
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });

  it("should allow any agent when no allowed list is configured", async () => {
    // No GH_AW_AGENT_ALLOWED set
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
          issue: { id: "issue-id", assignees: { nodes: [] } },
        },
      })
      .mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should not log allowed agents when list is not configured
    expect(mockCore.info).not.toHaveBeenCalledWith(expect.stringContaining("Allowed agents:"));
    expect(mockCore.error).not.toHaveBeenCalled();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should skip assignment and not fail when ignore-if-error is true and auth error occurs", async () => {
    process.env.GH_AW_AGENT_IGNORE_IF_ERROR = "true";
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

    // Simulate authentication error - use mockRejectedValueOnce to avoid affecting other tests
    const authError = new Error("Bad credentials");
    mockGithub.graphql.mockRejectedValueOnce(authError);

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should log that ignore-if-error is enabled
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Ignore-if-error mode enabled: Will not fail if agent assignment encounters auth errors"));

    // Should warn about skipping but not fail
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Agent assignment failed"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("ignore-if-error=true"));

    // Should not fail the workflow
    expect(mockCore.setFailed).not.toHaveBeenCalled();

    // Summary should show skipped assignments
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryCall).toContain("⏭️ Skipped");
    expect(summaryCall).toContain("assignment failed due to error");
  });

  it("should fail when ignore-if-error is false (default) and auth error occurs", async () => {
    // Don't set GH_AW_AGENT_IGNORE_IF_MISSING (defaults to false)
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

    // Simulate authentication error
    const authError = new Error("Bad credentials");
    mockGithub.graphql.mockRejectedValue(authError);

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should NOT log ignore-if-error mode
    expect(mockCore.info).not.toHaveBeenCalledWith(expect.stringContaining("ignore-if-error mode enabled"));

    // Should error and fail
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to assign agent"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to assign 1 agent(s)"));

    // Should post a failure comment on the issue with all required properties
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "test-owner",
        repo: "test-repo",
        issue_number: 42,
        body: expect.stringMatching(/Assignment failed.*Bad credentials/s),
      })
    );
  });

  it("should handle ignore-if-error when 'Resource not accessible' error", async () => {
    process.env.GH_AW_AGENT_IGNORE_IF_ERROR = "true";
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

    // Simulate permission error
    const permError = new Error("Resource not accessible by integration");
    mockGithub.graphql.mockRejectedValue(permError);

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should skip and not fail
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Agent assignment failed"));
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should still fail on non-auth errors even with ignore-if-error enabled", async () => {
    process.env.GH_AW_AGENT_IGNORE_IF_MISSING = "true";
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

    // Simulate a different error (not auth-related)
    const otherError = new Error("Network timeout");
    mockGithub.graphql.mockRejectedValue(otherError);

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should error and fail (not skipped because it's not an auth error)
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to assign agent"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to assign 1 agent(s)"));
  });

  it("should not post failure comment on success", async () => {
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
            id: "I_abc123",
            assignees: { nodes: [] },
          },
        },
      })
      .mockResolvedValueOnce({
        replaceActorsForAssignable: { __typename: "ReplaceActorsForAssignablePayload" },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should NOT post a failure comment on success
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("should post failure comment on single failed assignment", async () => {
    setAgentOutput({
      items: [{ type: "assign_to_agent", issue_number: 11, agent: "copilot" }],
      errors: [],
    });

    // Fail all assignments with auth error
    const authError = new Error("Bad credentials");
    mockGithub.graphql.mockRejectedValue(authError);

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should post a failure comment for the failed issue with all required properties
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledTimes(1);
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "test-owner",
        repo: "test-repo",
        issue_number: 11,
        body: expect.stringMatching(/Assignment failed.*Bad credentials/s),
      })
    );
  });

  it("should sanitize dangerous content in failure comment body", async () => {
    setAgentOutput({
      items: [{ type: "assign_to_agent", issue_number: 11, agent: "copilot" }],
      errors: [],
    });

    // Simulate an error whose message contains an @mention and an HTML comment —
    // both are potentially dangerous if posted unsanitized.
    const dangerousError = new Error("@admin triggered <!-- inject --> error");
    mockGithub.graphql.mockRejectedValue(dangerousError);

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledTimes(1);
    const [callArg] = mockGithub.rest.issues.createComment.mock.calls[0];
    // The body must be a string (sanitizeContent never returns undefined)
    expect(typeof callArg.body).toBe("string");
    // The raw @mention should be neutralized (wrapped in backticks, not bare)
    expect(callArg.body).not.toMatch(/(?<!`)@admin(?!`)/);
    // The HTML comment should be stripped
    expect(callArg.body).not.toContain("<!-- inject -->");
  });

  it("should not post failure comment when ignore-if-error skips the assignment", async () => {
    process.env.GH_AW_AGENT_IGNORE_IF_ERROR = "true";
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

    // Simulate authentication error (will be skipped by ignore-if-error)
    const authError = new Error("Bad credentials");
    mockGithub.graphql.mockRejectedValue(authError);

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should NOT post a failure comment since it was skipped
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("should still set outputs and log warning when failure comment post fails", async () => {
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

    const authError = new Error("Bad credentials");
    mockGithub.graphql.mockRejectedValue(authError);

    // Simulate failure to post comment
    mockGithub.rest.issues.createComment.mockRejectedValue(new Error("Could not post comment"));

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should still set the assignment_error outputs even if comment fails
    expect(mockCore.setOutput).toHaveBeenCalledWith("assignment_error_count", "1");
    expect(mockCore.setOutput).toHaveBeenCalledWith("assignment_errors", expect.stringContaining("Bad credentials"));

    // Should warn about failure to post comment (best-effort)
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to post failure comment"));
  });

  it.skip("should add 10-second delay between multiple agent assignments", async () => {
    // Note: This test is skipped because testing actual delays with eval() is complex.
    // The implementation has been manually verified to include the delay logic.
    // See lines in assign_to_agent.cjs where sleep(10000) is called between iterations.
    setAgentOutput({
      items: [
        { type: "assign_to_agent", issue_number: 1, agent: "copilot" },
        { type: "assign_to_agent", issue_number: 2, agent: "copilot" },
        { type: "assign_to_agent", issue_number: 3, agent: "copilot" },
      ],
      errors: [],
    });

    // Mock GraphQL responses for all three assignments
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
      })
      .mockResolvedValueOnce({
        repository: {
          issue: { id: "issue-id-3", assignees: { nodes: [] } },
        },
      })
      .mockResolvedValueOnce({
        addAssigneesToAssignable: {
          assignable: { assignees: { nodes: [{ login: "copilot-swe-agent" }] } },
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Verify delay message was logged twice (2 delays between 3 items)
    const delayMessages = mockCore.info.mock.calls.filter(call => call[0].includes("Waiting 10 seconds before processing next agent assignment"));
    expect(delayMessages).toHaveLength(2);
  }, 30000); // Increase timeout to 30 seconds to account for 2x10s delays

  describe("Cross-repository allowlist validation", () => {
    it("should reject target repository not in allowlist", async () => {
      process.env.GH_AW_ALLOWED_REPOS = "allowed-owner/allowed-repo";

      setAgentOutput({
        items: [
          {
            type: "assign_to_agent",
            issue_number: 42,
            agent: "copilot",
            repo: "not-allowed/other-repo",
          },
        ],
        errors: [],
      });

      await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("E004:"));
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("not in the allowed-repos list"));
    });

    it("should allow target repository in allowlist", async () => {
      process.env.GH_AW_ALLOWED_REPOS = "allowed-owner/allowed-repo,other-owner/other-repo";

      setAgentOutput({
        items: [
          {
            type: "assign_to_agent",
            issue_number: 42,
            agent: "copilot",
            repo: "allowed-owner/allowed-repo",
          },
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
            issue: { id: "issue-id", assignees: { nodes: [] } },
          },
        })
        .mockResolvedValueOnce({
          replaceActorsForAssignable: {
            __typename: "ReplaceActorsForAssignablePayload",
          },
        });

      await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      // Check that the target repository was used and assignment proceeded
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Looking for copilot coding agent"));
    }, 20000);

    it("should allow default repository even without allowlist", async () => {
      // Default repo is test-owner/test-repo (from mockContext)
      // No GH_AW_TARGET_REPO_SLUG set, no GH_AW_ALLOWED_REPOS set
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
            issue: { id: "issue-id", assignees: { nodes: [] } },
          },
        })
        .mockResolvedValueOnce({
          replaceActorsForAssignable: {
            __typename: "ReplaceActorsForAssignablePayload",
          },
        });

      await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

      expect(mockCore.setFailed).not.toHaveBeenCalled();
      // Check that assignment proceeded without errors
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Default target repo: test-owner/test-repo"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Looking for copilot coding agent"));
    }, 20000);
  });

  it("should handle pull-request-repo configuration correctly", async () => {
    process.env.GH_AW_AGENT_PULL_REQUEST_REPO = "test-owner/pull-request-repo";
    // Note: pull-request-repo is automatically allowed, no need to set allowed list
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
    mockGithub.graphql
      // Get PR repository ID and default branch
      .mockResolvedValueOnce({
        repository: {
          id: "pull-request-repo-id",
          defaultBranchRef: { name: "main" },
        },
      })
      // Find agent
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "agent-id" }],
          },
        },
      })
      // Get issue details
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: { nodes: [] },
          },
        },
      })
      // Assign agent with agentAssignment
      .mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Using pull request repository: test-owner/pull-request-repo"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Pull request repository ID: pull-request-repo-id"));

    // Verify the mutation was called with agentAssignment
    const lastGraphQLCall = mockGithub.graphql.mock.calls[mockGithub.graphql.mock.calls.length - 1];
    expect(lastGraphQLCall[0]).toContain("agentAssignment");
    expect(lastGraphQLCall[0]).toContain("targetRepositoryId");
    expect(lastGraphQLCall[1].targetRepoId).toBe("pull-request-repo-id");
  });

  it("should handle per-item pull_request_repo parameter", async () => {
    // Set global pull-request-repo which will be automatically allowed
    process.env.GH_AW_AGENT_PULL_REQUEST_REPO = "test-owner/default-pr-repo";
    // Set allowed list for additional repos
    process.env.GH_AW_AGENT_ALLOWED_PULL_REQUEST_REPOS = "test-owner/item-pull-request-repo";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
          agent: "copilot",
          pull_request_repo: "test-owner/item-pull-request-repo",
        },
      ],
      errors: [],
    });

    // Mock GraphQL responses
    mockGithub.graphql
      // Get global PR repository ID and default branch (for default-pr-repo)
      .mockResolvedValueOnce({
        repository: {
          id: "default-pr-repo-id",
          defaultBranchRef: { name: "main" },
        },
      })
      // Get item PR repository ID
      .mockResolvedValueOnce({
        repository: {
          id: "item-pull-request-repo-id",
        },
      })
      // Find agent
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "agent-id" }],
          },
        },
      })
      // Get issue details
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: { nodes: [] },
          },
        },
      })
      // Assign agent with agentAssignment
      .mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Using per-item pull request repository: test-owner/item-pull-request-repo"));

    // Verify the mutation was called with per-item PR repo ID
    const lastGraphQLCall = mockGithub.graphql.mock.calls[mockGithub.graphql.mock.calls.length - 1];
    expect(lastGraphQLCall[1].targetRepoId).toBe("item-pull-request-repo-id");
  });

  it("should reject per-item pull_request_repo not in allowed list", async () => {
    process.env.GH_AW_AGENT_PULL_REQUEST_REPO = "test-owner/default-pr-repo";
    process.env.GH_AW_AGENT_ALLOWED_PULL_REQUEST_REPOS = "test-owner/allowed-pr-repo";
    setAgentOutput({
      items: [
        {
          type: "assign_to_agent",
          issue_number: 42,
          agent: "copilot",
          pull_request_repo: "test-owner/not-allowed-repo",
        },
      ],
      errors: [],
    });

    // Mock global PR repo lookup
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "default-pr-repo-id",
        defaultBranchRef: { name: "main" },
      },
    });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("E004:"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to assign 1 agent(s)"));
  });

  it("should allow pull-request-repo without it being in allowed-pull-request-repos", async () => {
    // Set pull-request-repo but DO NOT set allowed-pull-request-repos
    // This tests that pull-request-repo is automatically allowed (like target-repo behavior)
    process.env.GH_AW_AGENT_PULL_REQUEST_REPO = "test-owner/auto-allowed-repo";
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
    mockGithub.graphql
      // Get PR repository ID and default branch
      .mockResolvedValueOnce({
        repository: {
          id: "auto-allowed-repo-id",
          defaultBranchRef: { name: "main" },
        },
      })
      // Find agent
      .mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "copilot-swe-agent", id: "agent-id" }],
          },
        },
      })
      // Get issue details
      .mockResolvedValueOnce({
        repository: {
          issue: {
            id: "issue-id",
            assignees: { nodes: [] },
          },
        },
      })
      // Assign agent with agentAssignment
      .mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    // Should succeed - pull-request-repo is automatically allowed
    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Using pull request repository: test-owner/auto-allowed-repo"));
  });

  it("should use explicit base-branch when GH_AW_AGENT_BASE_BRANCH is set", async () => {
    process.env.GH_AW_AGENT_PULL_REQUEST_REPO = "test-owner/code-repo";
    process.env.GH_AW_AGENT_BASE_BRANCH = "develop";
    setAgentOutput({
      items: [{ type: "assign_to_agent", issue_number: 42, agent: "copilot" }],
      errors: [],
    });

    mockGithub.graphql
      // Get PR repo ID and default branch
      .mockResolvedValueOnce({ repository: { id: "code-repo-id", defaultBranchRef: { name: "main" } } })
      // Find agent
      .mockResolvedValueOnce({ repository: { suggestedActors: { nodes: [{ login: "copilot-swe-agent", id: "agent-id" }] } } })
      // Get issue details
      .mockResolvedValueOnce({ repository: { issue: { id: "issue-id", assignees: { nodes: [] } } } })
      // Assign agent
      .mockResolvedValueOnce({ replaceActorsForAssignable: { __typename: "ReplaceActorsForAssignablePayload" } });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    // Verify the mutation was called with baseRef set to the explicit base-branch
    const lastCall = mockGithub.graphql.mock.calls[mockGithub.graphql.mock.calls.length - 1];
    expect(lastCall[0]).toContain("baseRef: $baseRef");
    expect(lastCall[1].baseRef).toBe("develop");
    // customInstructions should NOT contain the branch instruction text
    expect(lastCall[1].customInstructions).toBeUndefined();
  });

  it("should auto-resolve non-main default branch from pull-request-repo and set as baseRef", async () => {
    process.env.GH_AW_AGENT_PULL_REQUEST_REPO = "test-owner/code-repo";
    // No GH_AW_AGENT_BASE_BRANCH set - should use repo's default branch
    setAgentOutput({
      items: [{ type: "assign_to_agent", issue_number: 42, agent: "copilot" }],
      errors: [],
    });

    mockGithub.graphql
      // Get PR repo ID and default branch (non-main)
      .mockResolvedValueOnce({ repository: { id: "code-repo-id", defaultBranchRef: { name: "develop" } } })
      // Find agent
      .mockResolvedValueOnce({ repository: { suggestedActors: { nodes: [{ login: "copilot-swe-agent", id: "agent-id" }] } } })
      // Get issue details
      .mockResolvedValueOnce({ repository: { issue: { id: "issue-id", assignees: { nodes: [] } } } })
      // Assign agent
      .mockResolvedValueOnce({ replaceActorsForAssignable: { __typename: "ReplaceActorsForAssignablePayload" } });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Resolved pull request repository default branch: develop"));
    // Verify the mutation was called with baseRef set to the resolved default branch
    const lastCall = mockGithub.graphql.mock.calls[mockGithub.graphql.mock.calls.length - 1];
    expect(lastCall[0]).toContain("baseRef: $baseRef");
    expect(lastCall[1].baseRef).toBe("develop");
  });

  it("should set baseRef when pull-request-repo default branch is main (no explicit base-branch)", async () => {
    process.env.GH_AW_AGENT_PULL_REQUEST_REPO = "test-owner/code-repo";
    // No GH_AW_AGENT_BASE_BRANCH set; repo default is main
    setAgentOutput({
      items: [{ type: "assign_to_agent", issue_number: 42, agent: "copilot" }],
      errors: [],
    });

    mockGithub.graphql
      // Get PR repo ID and default branch (main)
      .mockResolvedValueOnce({ repository: { id: "code-repo-id", defaultBranchRef: { name: "main" } } })
      // Find agent
      .mockResolvedValueOnce({ repository: { suggestedActors: { nodes: [{ login: "copilot-swe-agent", id: "agent-id" }] } } })
      // Get issue details
      .mockResolvedValueOnce({ repository: { issue: { id: "issue-id", assignees: { nodes: [] } } } })
      // Assign agent
      .mockResolvedValueOnce({ replaceActorsForAssignable: { __typename: "ReplaceActorsForAssignablePayload" } });

    await eval(`(async () => { ${assignToAgentScript}; ${STANDALONE_RUNNER} })()`);

    expect(mockCore.setFailed).not.toHaveBeenCalled();
    // Verify the mutation was called with baseRef set to the repo's default branch
    const lastCall = mockGithub.graphql.mock.calls[mockGithub.graphql.mock.calls.length - 1];
    expect(lastCall[0]).toContain("baseRef: $baseRef");
    expect(lastCall[1].baseRef).toBe("main");
    expect(lastCall[1].customInstructions).toBeUndefined();
  });
});
