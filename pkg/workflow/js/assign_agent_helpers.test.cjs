import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
};

const mockGithub = {
  graphql: vi.fn(),
  rest: {
    issues: {
      get: vi.fn(),
      addAssignees: vi.fn(),
    },
  },
};

// Set up global mocks before importing the module
globalThis.core = mockCore;
globalThis.github = mockGithub;

const {
  AGENT_LOGIN_NAMES,
  getAgentName,
  getAvailableAgentLogins,
  findAgent,
  getIssueDetails,
  assignAgentToIssue,
  assignAgentViaRest,
  isAgentAlreadyAssigned,
  generatePermissionErrorSummary,
  assignAgentToIssueByName,
} = await import("./assign_agent_helpers.cjs");

describe("assign_agent_helpers.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset REST API mocks
    mockGithub.rest.issues.get.mockReset();
    mockGithub.rest.issues.addAssignees.mockReset();
  });

  describe("AGENT_LOGIN_NAMES", () => {
    it("should have copilot mapped to copilot-swe-agent", () => {
      expect(AGENT_LOGIN_NAMES).toEqual({
        copilot: "copilot-swe-agent",
      });
    });
  });

  describe("getAgentName", () => {
    it("should return copilot for @copilot", () => {
      expect(getAgentName("@copilot")).toBe("copilot");
    });

    it("should return copilot for copilot without @ prefix", () => {
      expect(getAgentName("copilot")).toBe("copilot");
    });

    it("should return null for unknown users", () => {
      expect(getAgentName("@some-user")).toBeNull();
      expect(getAgentName("some-user")).toBeNull();
    });

    it("should return null for empty string", () => {
      expect(getAgentName("")).toBeNull();
    });

    it("should return null for partial matches", () => {
      expect(getAgentName("copilot-agent")).toBeNull();
      expect(getAgentName("@copilot-agent")).toBeNull();
    });
  });

  describe("getAvailableAgentLogins", () => {
    it("should return available agent logins using github.graphql when no token provided", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [
              { login: "copilot-swe-agent", __typename: "Bot" },
              { login: "some-other-bot", __typename: "Bot" },
            ],
          },
        },
      });

      const result = await getAvailableAgentLogins("owner", "repo");

      expect(result).toEqual(["copilot-swe-agent"]);
      expect(mockGithub.graphql).toHaveBeenCalledTimes(1);
    });

    it("should return empty array when no agents are available", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ login: "some-random-bot", __typename: "Bot" }],
          },
        },
      });

      const result = await getAvailableAgentLogins("owner", "repo");

      expect(result).toEqual([]);
    });

    it("should handle GraphQL errors gracefully", async () => {
      mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error"));

      const result = await getAvailableAgentLogins("owner", "repo");

      expect(result).toEqual([]);
      expect(mockCore.debug).toHaveBeenCalledWith(expect.stringContaining("Failed to list available agent logins"));
    });

    it("should handle null suggestedActors", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: null,
        },
      });

      const result = await getAvailableAgentLogins("owner", "repo");

      expect(result).toEqual([]);
    });
  });

  describe("findAgent", () => {
    it("should find copilot agent and return its ID using github.graphql", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [
              { id: "BOT_12345", login: "copilot-swe-agent", __typename: "Bot" },
              { id: "BOT_67890", login: "other-bot", __typename: "Bot" },
            ],
          },
        },
      });

      const result = await findAgent("owner", "repo", "copilot");

      expect(result).toBe("BOT_12345");
      expect(mockGithub.graphql).toHaveBeenCalledTimes(1);
    });

    it("should return null for unknown agent name", async () => {
      // Need to mock GraphQL because the function calls it before checking agent name
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [],
          },
        },
      });

      const result = await findAgent("owner", "repo", "unknown-agent");

      expect(result).toBeNull();
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Unknown agent: unknown-agent"));
    });

    it("should return null when copilot is not available", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ id: "BOT_67890", login: "other-bot", __typename: "Bot" }],
          },
        },
      });

      const result = await findAgent("owner", "repo", "copilot");

      expect(result).toBeNull();
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("copilot coding agent (copilot-swe-agent) is not available"));
    });

    it("should handle GraphQL errors", async () => {
      mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error"));

      const result = await findAgent("owner", "repo", "copilot");

      expect(result).toBeNull();
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to find copilot agent"));
    });
  });

  describe("getIssueDetails", () => {
    it("should return issue ID and current assignees", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          issue: {
            id: "ISSUE_123",
            assignees: {
              nodes: [{ id: "USER_1" }, { id: "USER_2" }],
            },
          },
        },
      });

      const result = await getIssueDetails("owner", "repo", 123);

      expect(result).toEqual({
        issueId: "ISSUE_123",
        currentAssignees: ["USER_1", "USER_2"],
      });
    });

    it("should return null when issue is not found", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          issue: null,
        },
      });

      const result = await getIssueDetails("owner", "repo", 999);

      expect(result).toBeNull();
      expect(mockCore.error).toHaveBeenCalledWith("Could not get issue data");
    });

    it("should handle GraphQL errors", async () => {
      mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error"));

      const result = await getIssueDetails("owner", "repo", 123);

      expect(result).toBeNull();
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to get issue details"));
    });

    it("should return empty assignees when none exist", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          issue: {
            id: "ISSUE_123",
            assignees: {
              nodes: [],
            },
          },
        },
      });

      const result = await getIssueDetails("owner", "repo", 123);

      expect(result).toEqual({
        issueId: "ISSUE_123",
        currentAssignees: [],
      });
    });
  });

  describe("assignAgentToIssue", () => {
    it("should successfully assign agent using simple mutation (no options)", async () => {
      // Mock the global github.graphql
      mockGithub.graphql.mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

      const result = await assignAgentToIssue("ISSUE_123", "AGENT_456", ["USER_1"], "copilot");

      expect(result).toBe(true);
      // Simple mutation without options should not include headers
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("replaceActorsForAssignable"),
        expect.objectContaining({
          assignableId: "ISSUE_123",
          actorIds: ["AGENT_456", "USER_1"],
        })
      );
      // Verify no extra arguments (no headers) for simple mutation
      expect(mockGithub.graphql.mock.calls[0].length).toBe(2);
    });

    it("should use extended mutation with headers when Copilot options provided", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

      const options = {
        baseBranch: "main",
        customInstructions: "Test instructions",
      };

      const result = await assignAgentToIssue("ISSUE_123", "AGENT_456", ["USER_1"], "copilot", options);

      expect(result).toBe(true);
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("replaceActorsForAssignable"),
        expect.objectContaining({
          assignableId: "ISSUE_123",
          actorIds: ["AGENT_456", "USER_1"],
          copilotAssignmentOptions: expect.objectContaining({
            baseBranch: "main",
            customInstructions: "Test instructions",
          }),
        }),
        expect.objectContaining({
          headers: expect.objectContaining({
            "GraphQL-Features": "issues_copilot_assignment_api_support",
          }),
        })
      );
    });

    it("should preserve existing assignees when adding agent", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

      await assignAgentToIssue("ISSUE_123", "AGENT_456", ["USER_1", "USER_2"], "copilot");

      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("replaceActorsForAssignable"),
        expect.objectContaining({
          assignableId: "ISSUE_123",
          actorIds: expect.arrayContaining(["AGENT_456", "USER_1", "USER_2"]),
        })
      );
    });

    it("should not duplicate agent if already in assignees", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

      await assignAgentToIssue(
        "ISSUE_123",
        "AGENT_456",
        ["AGENT_456", "USER_1"], // Agent already in list
        "copilot"
      );

      const calledArgs = mockGithub.graphql.mock.calls[0][1];
      // Agent should only appear once in the actorIds array
      const agentMatches = calledArgs.actorIds.filter(id => id === "AGENT_456");
      expect(agentMatches.length).toBe(1);
    });
  });

  describe("isAgentAlreadyAssigned", () => {
    it("should return true if agent is already assigned", async () => {
      mockGithub.rest.issues.get.mockResolvedValueOnce({
        data: {
          assignees: [{ login: "copilot-swe-agent" }],
        },
      });

      const result = await isAgentAlreadyAssigned("owner", "repo", 123, "copilot");

      expect(result).toBe(true);
    });

    it("should return false if agent is not assigned", async () => {
      mockGithub.rest.issues.get.mockResolvedValueOnce({
        data: {
          assignees: [{ login: "other-user" }],
        },
      });

      const result = await isAgentAlreadyAssigned("owner", "repo", 123, "copilot");

      expect(result).toBe(false);
    });

    it("should return false on error", async () => {
      mockGithub.rest.issues.get.mockRejectedValueOnce(new Error("API error"));

      const result = await isAgentAlreadyAssigned("owner", "repo", 123, "copilot");

      expect(result).toBe(false);
    });
  });

  describe("assignAgentViaRest", () => {
    it("should successfully assign copilot agent via REST API", async () => {
      mockGithub.rest.issues.addAssignees.mockResolvedValueOnce({
        status: 201,
        data: {},
      });

      const result = await assignAgentViaRest("owner", "repo", 123, "copilot");

      expect(result.success).toBe(true);
      expect(mockGithub.rest.issues.addAssignees).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        issue_number: 123,
        assignees: ["copilot-swe-agent"],
      });
    });

    it("should return error for unknown agent", async () => {
      const result = await assignAgentViaRest("owner", "repo", 123, "unknown-agent");

      expect(result.success).toBe(false);
      expect(result.error).toContain("Unknown agent");
    });

    it("should handle 422 validation errors", async () => {
      mockGithub.rest.issues.addAssignees.mockRejectedValueOnce(new Error("422 Validation Failed"));

      const result = await assignAgentViaRest("owner", "repo", 123, "copilot");

      expect(result.success).toBe(false);
      expect(result.error).toContain("may not be available");
    });
  });

  describe("generatePermissionErrorSummary", () => {
    it("should return markdown content with permission requirements", () => {
      const summary = generatePermissionErrorSummary();

      expect(summary).toContain("### ⚠️ Permission Requirements");
      expect(summary).toContain("COPILOT_GITHUB_TOKEN");
      expect(summary).toContain("repo");
      expect(summary).toContain("GITHUB_TOKEN");
    });
  });

  describe("assignAgentToIssueByName", () => {
    it("should successfully assign copilot agent", async () => {
      // Mock findAgent (uses github.graphql)
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ id: "AGENT_456", login: "copilot-swe-agent", __typename: "Bot" }],
          },
        },
      });

      // Mock getIssueDetails (uses github.graphql)
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          issue: {
            id: "ISSUE_123",
            assignees: {
              nodes: [],
            },
          },
        },
      });

      // Mock assignAgentToIssue mutation (uses github.graphql)
      mockGithub.graphql.mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot");

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("Looking for copilot coding agent...");
      expect(mockCore.info).toHaveBeenCalledWith("Found copilot coding agent (ID: AGENT_456)");
    });

    it("should return error for unsupported agent", async () => {
      const result = await assignAgentToIssueByName("owner", "repo", 123, "unknown");

      expect(result.success).toBe(false);
      expect(result.error).toContain("not supported");
      expect(mockCore.warning).toHaveBeenCalled();
    });

    it("should return error when agent is not available", async () => {
      // Mock findAgent and getAvailableAgentLogins (both use github.graphql)
      // Both calls return empty nodes
      mockGithub.graphql.mockResolvedValue({
        repository: {
          suggestedActors: {
            nodes: [], // No agents
          },
        },
      });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot");

      expect(result.success).toBe(false);
      expect(result.error).toContain("not available");
    });

    it("should report already assigned when agent is in assignees", async () => {
      const agentId = "AGENT_456";

      // Mock findAgent (uses github.graphql)
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ id: agentId, login: "copilot-swe-agent", __typename: "Bot" }],
          },
        },
      });

      // Mock getIssueDetails (uses github.graphql)
      mockGithub.graphql.mockResolvedValueOnce({
        repository: {
          issue: {
            id: "ISSUE_123",
            assignees: {
              nodes: [{ id: agentId }], // Already assigned
            },
          },
        },
      });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot");

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("copilot is already assigned to issue #123");
    });
  });
});
