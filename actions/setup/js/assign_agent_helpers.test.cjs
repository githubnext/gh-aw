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
};

// Set up global mocks before importing the module
globalThis.core = mockCore;
globalThis.github = mockGithub;

const { AGENT_LOGIN_NAMES, getAgentName, getAvailableAgentLogins, findAgent, getIssueDetails, assignAgentToIssue, generatePermissionErrorSummary, assignAgentToIssueByName } = await import("./assign_agent_helpers.cjs");

describe("assign_agent_helpers.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
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
              nodes: [
                { id: "USER_1", login: "user1" },
                { id: "USER_2", login: "user2" },
              ],
            },
          },
        },
      });

      const result = await getIssueDetails("owner", "repo", 123);

      expect(result).toEqual({
        issueId: "ISSUE_123",
        currentAssignees: [
          { id: "USER_1", login: "user1" },
          { id: "USER_2", login: "user2" },
        ],
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

      await expect(getIssueDetails("owner", "repo", 123)).rejects.toThrow("GraphQL error");
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
    it("should successfully assign agent using mutation", async () => {
      // Mock the global github.graphql
      mockGithub.graphql.mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

      const result = await assignAgentToIssue("ISSUE_123", "AGENT_456", [{ id: "USER_1", login: "user1" }], "copilot", null);

      expect(result).toBe(true);
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("replaceActorsForAssignable"),
        expect.objectContaining({
          assignableId: "ISSUE_123",
          actorIds: ["AGENT_456", "USER_1"],
        })
      );
    });

    it("should preserve existing assignees when adding agent", async () => {
      mockGithub.graphql.mockResolvedValueOnce({
        replaceActorsForAssignable: {
          __typename: "ReplaceActorsForAssignablePayload",
        },
      });

      await assignAgentToIssue(
        "ISSUE_123",
        "AGENT_456",
        [
          { id: "USER_1", login: "user1" },
          { id: "USER_2", login: "user2" },
        ],
        "copilot",
        null
      );

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

  describe("generatePermissionErrorSummary", () => {
    it("should return markdown content with permission requirements", () => {
      const summary = generatePermissionErrorSummary();

      expect(summary).toContain("### ⚠️ Permission Requirements");
      expect(summary).toContain("actions: write");
      expect(summary).toContain("contents: write");
      expect(summary).toContain("issues: write");
      expect(summary).toContain("pull-requests: write");
      expect(summary).toContain("replaceActorsForAssignable");
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
