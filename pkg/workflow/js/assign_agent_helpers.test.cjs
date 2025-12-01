import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

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

// Mock Octokit graphql function for custom token scenarios
const mockOctokitGraphql = vi.fn();

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
  generatePermissionErrorSummary,
  assignAgentToIssueByName,
  setGetOctokitFactory,
} = await import("./assign_agent_helpers.cjs");

// Mock Octokit instance with graphql method
const mockOctokitInstance = {
  graphql: mockOctokitGraphql,
};

describe("assign_agent_helpers.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Set up the mock getOctokit factory
    setGetOctokitFactory(() => mockOctokitInstance);
  });

  afterEach(() => {
    // Reset to null so real getOctokit is used if not mocked
    setGetOctokitFactory(null);
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

    it("should return available agent logins using Octokit when token is provided", async () => {
      mockOctokitGraphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [
              { login: "copilot-swe-agent", __typename: "Bot" },
              { login: "some-other-bot", __typename: "Bot" },
            ],
          },
        },
      });

      const result = await getAvailableAgentLogins("owner", "repo", "test-token");

      expect(result).toEqual(["copilot-swe-agent"]);
      expect(mockOctokitGraphql).toHaveBeenCalledTimes(1);
      expect(mockGithub.graphql).not.toHaveBeenCalled();
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

    it("should handle Octokit errors gracefully when token is provided", async () => {
      mockOctokitGraphql.mockRejectedValueOnce(new Error("Octokit error"));

      const result = await getAvailableAgentLogins("owner", "repo", "test-token");

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
    it("should find copilot agent and return its ID using github.graphql when no token provided", async () => {
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

    it("should find copilot agent and return its ID using Octokit when token is provided", async () => {
      mockOctokitGraphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [
              { id: "BOT_12345", login: "copilot-swe-agent", __typename: "Bot" },
              { id: "BOT_67890", login: "other-bot", __typename: "Bot" },
            ],
          },
        },
      });

      const result = await findAgent("owner", "repo", "copilot", "test-token");

      expect(result).toBe("BOT_12345");
      expect(mockOctokitGraphql).toHaveBeenCalledTimes(1);
      expect(mockGithub.graphql).not.toHaveBeenCalled();
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

    it("should handle fetch errors when token is provided", async () => {
      const mockFetch = vi.fn().mockRejectedValue(new Error("Network error"));
      global.fetch = mockFetch;

      const result = await findAgent("owner", "repo", "copilot", "test-token");

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
    it("should successfully assign agent using mutation", async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        json: () =>
          Promise.resolve({
            data: {
              replaceActorsForAssignable: {
                __typename: "ReplaceActorsForAssignablePayload",
              },
            },
          }),
      });
      global.fetch = mockFetch;

      const result = await assignAgentToIssue("ISSUE_123", "AGENT_456", ["USER_1"], "copilot", "test-token");

      expect(result).toBe(true);
      expect(mockFetch).toHaveBeenCalledWith(
        "https://api.github.com/graphql",
        expect.objectContaining({
          method: "POST",
          headers: {
            Authorization: "Bearer test-token",
            "Content-Type": "application/json",
          },
        })
      );
    });

    it("should return false when token is not set", async () => {
      const result = await assignAgentToIssue(
        "ISSUE_123",
        "AGENT_456",
        [],
        "copilot",
        "" // Empty token
      );

      expect(result).toBe(false);
      expect(mockCore.error).toHaveBeenCalledWith("GitHub token is not set. Cannot perform assignment mutation.");
    });

    it("should preserve existing assignees when adding agent", async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        json: () =>
          Promise.resolve({
            data: {
              replaceActorsForAssignable: {
                __typename: "ReplaceActorsForAssignablePayload",
              },
            },
          }),
      });
      global.fetch = mockFetch;

      await assignAgentToIssue("ISSUE_123", "AGENT_456", ["USER_1", "USER_2"], "copilot", "test-token");

      const fetchCall = mockFetch.mock.calls[0];
      const body = JSON.parse(fetchCall[1].body);
      // The mutation should use GraphQL variables - check that variables are passed correctly
      expect(body.variables.assignableId).toBe("ISSUE_123");
      expect(body.variables.actorIds).toContain("AGENT_456");
      expect(body.variables.actorIds).toContain("USER_1");
      expect(body.variables.actorIds).toContain("USER_2");
    });

    it("should not duplicate agent if already in assignees", async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        json: () =>
          Promise.resolve({
            data: {
              replaceActorsForAssignable: {
                __typename: "ReplaceActorsForAssignablePayload",
              },
            },
          }),
      });
      global.fetch = mockFetch;

      await assignAgentToIssue(
        "ISSUE_123",
        "AGENT_456",
        ["AGENT_456", "USER_1"], // Agent already in list
        "copilot",
        "test-token"
      );

      const fetchCall = mockFetch.mock.calls[0];
      const body = JSON.parse(fetchCall[1].body);
      // Agent should only appear once in the variables.actorIds array
      const agentMatches = body.variables.actorIds.filter(id => id === "AGENT_456");
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
      // Mock findAgent (uses Octokit because token is provided)
      mockOctokitGraphql.mockResolvedValueOnce({
        repository: {
          suggestedActors: {
            nodes: [{ id: "AGENT_456", login: "copilot-swe-agent", __typename: "Bot" }],
          },
        },
      });

      // Mock getIssueDetails (uses github.graphql because no token parameter is passed to it)
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

      // Mock assignAgentToIssue mutation (still uses fetch for the mutation)
      const mockFetch = vi.fn().mockResolvedValue({
        json: () =>
          Promise.resolve({
            data: {
              replaceActorsForAssignable: {
                __typename: "ReplaceActorsForAssignablePayload",
              },
            },
          }),
      });
      global.fetch = mockFetch;

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot", "test-token");

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("Looking for copilot coding agent...");
      expect(mockCore.info).toHaveBeenCalledWith("Found copilot coding agent (ID: AGENT_456)");
    });

    it("should return error for unsupported agent", async () => {
      const result = await assignAgentToIssueByName("owner", "repo", 123, "unknown", "test-token");

      expect(result.success).toBe(false);
      expect(result.error).toContain("not supported");
      expect(mockCore.warning).toHaveBeenCalled();
    });

    it("should return error when agent is not available", async () => {
      // Mock findAgent and getAvailableAgentLogins (both use Octokit with token)
      // Both calls return empty nodes
      mockOctokitGraphql.mockResolvedValue({
        repository: {
          suggestedActors: {
            nodes: [], // No agents
          },
        },
      });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot", "test-token");

      expect(result.success).toBe(false);
      expect(result.error).toContain("not available");
    });

    it("should report already assigned when agent is in assignees", async () => {
      const agentId = "AGENT_456";

      // Mock findAgent (uses Octokit with token)
      mockOctokitGraphql.mockResolvedValueOnce({
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

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot", "test-token");

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("copilot is already assigned to issue #123");
    });
  });
});
