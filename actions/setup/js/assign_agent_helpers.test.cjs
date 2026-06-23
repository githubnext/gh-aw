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
  request: vi.fn(),
  rest: {
    issues: {
      checkUserCanBeAssigned: vi.fn(),
      get: vi.fn(),
      listAssignees: vi.fn(),
    },
    users: {
      getByUsername: vi.fn(),
    },
    pulls: {
      get: vi.fn(),
    },
  },
};

// Set up global mocks before importing the module
globalThis.core = mockCore;
globalThis.github = mockGithub;

const { AGENT_LOGIN_NAMES, getAgentName, getAgentLogins, getAvailableAgentLogins, getAssignableBots, findAgent, getIssueDetails, getPullRequestDetails, assignAgentToIssue, generatePermissionErrorSummary, assignAgentToIssueByName } =
  await import("./assign_agent_helpers.cjs");

describe("assign_agent_helpers.cjs", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("AGENT_LOGIN_NAMES", () => {
    it("should have copilot mapped to known assignee aliases", () => {
      expect(AGENT_LOGIN_NAMES).toEqual({
        copilot: ["copilot-swe-agent", "github-copilot-enterprise", "github-copilot-enterprise[bot]", "github-copilot", "github-copilot[bot]"],
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

    it("should return copilot for known copilot assignee aliases", () => {
      expect(getAgentName("copilot-swe-agent")).toBe("copilot");
      expect(getAgentName("@github-copilot-enterprise")).toBe("copilot");
      expect(getAgentName("github-copilot-enterprise[bot]")).toBe("copilot");
      expect(getAgentName("github-copilot")).toBe("copilot");
      expect(getAgentName("github-copilot[bot]")).toBe("copilot");
    });

    it("should return null for empty string", () => {
      expect(getAgentName("")).toBeNull();
    });

    it("should return null for partial matches", () => {
      expect(getAgentName("copilot-agent")).toBeNull();
      expect(getAgentName("@copilot-agent")).toBeNull();
    });
  });

  describe("getAgentLogins", () => {
    it("should return all known copilot aliases", () => {
      expect(getAgentLogins("copilot")).toEqual(["copilot-swe-agent", "github-copilot-enterprise", "github-copilot-enterprise[bot]", "github-copilot", "github-copilot[bot]"]);
    });

    it("should return empty array for unknown agents", () => {
      expect(getAgentLogins("unknown")).toEqual([]);
    });
  });

  describe("getAvailableAgentLogins", () => {
    it("should return available agent logins when an alias is assignable", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.rest.issues.checkUserCanBeAssigned.mockRejectedValueOnce(err404).mockResolvedValueOnce({}).mockRejectedValue(err404);

      const result = await getAvailableAgentLogins("owner", "repo");

      expect(result).toEqual(["github-copilot-enterprise"]);
      expect(mockGithub.rest.issues.checkUserCanBeAssigned).toHaveBeenCalledWith({ owner: "owner", repo: "repo", assignee: "copilot-swe-agent" });
      expect(mockGithub.rest.issues.checkUserCanBeAssigned).toHaveBeenCalledWith({ owner: "owner", repo: "repo", assignee: "github-copilot-enterprise" });
      expect(mockGithub.rest.issues.checkUserCanBeAssigned).toHaveBeenCalledTimes(5);
    });

    it("should return empty array when no alias is assignable (404)", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.rest.issues.checkUserCanBeAssigned.mockRejectedValue(err404);

      const result = await getAvailableAgentLogins("owner", "repo");

      expect(result).toEqual([]);
      expect(mockGithub.rest.issues.checkUserCanBeAssigned).toHaveBeenCalledTimes(5);
    });

    it("should handle non-404 errors gracefully and return empty array", async () => {
      const err500 = Object.assign(new Error("Server error"), { status: 500 });
      mockGithub.rest.issues.checkUserCanBeAssigned.mockRejectedValue(err500);

      const result = await getAvailableAgentLogins("owner", "repo");

      expect(result).toEqual([]);
      expect(mockCore.debug).toHaveBeenCalledWith(expect.stringContaining("Failed to check assignability for copilot-swe-agent"));
      expect(mockGithub.rest.issues.checkUserCanBeAssigned).toHaveBeenCalledTimes(5);
    });
  });

  describe("getAssignableBots", () => {
    it("should return assignable bot logins from repository assignees", async () => {
      mockGithub.rest.issues.listAssignees.mockResolvedValue({
        data: [
          { login: "github-actions[bot]", type: "Bot" },
          { login: "octocat", type: "User" },
        ],
      });

      const result = await getAssignableBots("owner", "repo");

      expect(result).toEqual(["github-actions[bot]"]);
      expect(mockGithub.rest.issues.listAssignees).toHaveBeenCalledWith({ owner: "owner", repo: "repo", per_page: 100, page: 1 });
    });

    it("should filter bots by [bot] login suffix when type is not set", async () => {
      mockGithub.rest.issues.listAssignees.mockResolvedValue({
        data: [{ login: "dependabot[bot]" }],
      });

      const result = await getAssignableBots("owner", "repo");

      expect(result).toEqual(["dependabot[bot]"]);
    });

    it("should return [] and debug-log when listAssignees throws", async () => {
      mockGithub.rest.issues.listAssignees.mockRejectedValue(new Error("API down"));

      const result = await getAssignableBots("owner", "repo");

      expect(result).toEqual([]);
      expect(mockCore.debug).toHaveBeenCalledWith(expect.stringContaining("Failed to list assignable bots"));
    });
  });

  describe("findAgent", () => {
    it("should find copilot agent via a fallback alias and return its ID via REST", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.rest.issues.checkUserCanBeAssigned.mockRejectedValueOnce(err404).mockResolvedValueOnce({});
      mockGithub.rest.users.getByUsername.mockResolvedValueOnce({ data: { id: 12345 } });

      const result = await findAgent("owner", "repo", "copilot");

      expect(result).toBe("12345");
      expect(mockGithub.rest.issues.checkUserCanBeAssigned).toHaveBeenCalledWith({ owner: "owner", repo: "repo", assignee: "copilot-swe-agent" });
      expect(mockGithub.rest.issues.checkUserCanBeAssigned).toHaveBeenCalledWith({ owner: "owner", repo: "repo", assignee: "github-copilot-enterprise" });
      expect(mockGithub.rest.users.getByUsername).toHaveBeenCalledWith({ username: "github-copilot-enterprise" });
      expect(mockGithub.rest.issues.checkUserCanBeAssigned).toHaveBeenCalledTimes(2);
    });

    it("should return null for unknown agent name", async () => {
      const result = await findAgent("owner", "repo", "unknown-agent");

      expect(result).toBeNull();
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Unknown agent: unknown-agent"));
    });

    it("should return null and log bots when no copilot alias is available (404)", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.rest.issues.checkUserCanBeAssigned.mockRejectedValue(err404);
      mockGithub.rest.issues.listAssignees.mockResolvedValue({ data: [{ login: "github-actions[bot]", type: "Bot" }] });

      const result = await findAgent("owner", "repo", "copilot");

      expect(result).toBeNull();
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("copilot coding agent aliases are not available"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Assignable bots in this repository: github-actions[bot]"));
    });

    it("should handle REST errors and re-throw auth errors", async () => {
      const authError = new Error("Bad credentials");
      mockGithub.rest.issues.checkUserCanBeAssigned.mockRejectedValueOnce(authError);

      await expect(findAgent("owner", "repo", "copilot")).rejects.toThrow("Bad credentials");
    });

    it("should return null for non-auth errors", async () => {
      const err = new Error("Something unexpected");
      mockGithub.rest.issues.checkUserCanBeAssigned.mockRejectedValueOnce(err);
      mockGithub.rest.issues.checkUserCanBeAssigned.mockRejectedValue(Object.assign(new Error("Not Found"), { status: 404 }));
      mockGithub.rest.issues.listAssignees.mockResolvedValue({ data: [] });

      const result = await findAgent("owner", "repo", "copilot");

      expect(result).toBeNull();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Assignee alias copilot-swe-agent was not assignable"));
    });
  });

  describe("getIssueDetails", () => {
    it("should return issue ID, current assignees, and task context", async () => {
      mockGithub.rest.issues.get.mockResolvedValueOnce({
        data: {
          id: 67890,
          number: 123,
          assignees: [
            { id: 1, login: "user1" },
            { id: 2, login: "user2" },
          ],
          html_url: "https://github.com/owner/repo/issues/123",
          title: "Test issue",
          body: "Test body",
        },
      });

      const result = await getIssueDetails("owner", "repo", 123);

      expect(result).toEqual({
        issueId: "67890",
        currentAssignees: [
          { id: "1", login: "user1" },
          { id: "2", login: "user2" },
        ],
        htmlUrl: "https://github.com/owner/repo/issues/123",
        title: "Test issue",
        body: "Test body",
        taskContext: { owner: "owner", repo: "repo", type: "issue", number: 123 },
      });
    });

    it("should return null when issue is not found", async () => {
      mockGithub.rest.issues.get.mockResolvedValueOnce({ data: null });

      const result = await getIssueDetails("owner", "repo", 999);

      expect(result).toBeNull();
      expect(mockCore.error).toHaveBeenCalledWith("Could not get issue data");
    });

    it("should re-throw REST errors", async () => {
      mockGithub.rest.issues.get.mockRejectedValueOnce(new Error("API error"));

      await expect(getIssueDetails("owner", "repo", 123)).rejects.toThrow("API error");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to get issue details"));
    });

    it("should return empty assignees when none exist", async () => {
      mockGithub.rest.issues.get.mockResolvedValueOnce({
        data: {
          id: 67890,
          number: 123,
          assignees: [],
          html_url: "https://github.com/owner/repo/issues/123",
          title: "Test issue",
          body: "",
        },
      });

      const result = await getIssueDetails("owner", "repo", 123);

      expect(result).toEqual({
        issueId: "67890",
        currentAssignees: [],
        htmlUrl: "https://github.com/owner/repo/issues/123",
        title: "Test issue",
        body: "",
        taskContext: { owner: "owner", repo: "repo", type: "issue", number: 123 },
      });
    });
  });

  describe("assignAgentToIssue", () => {
    const taskContext = { owner: "myorg", repo: "myrepo", type: "issue", number: 42 };

    it("should start an agent task via REST", async () => {
      const mockRequest = vi.fn().mockResolvedValue({ data: { id: "task-123" } });
      const restClient = { request: mockRequest };

      const result = await assignAgentToIssue("ignored-id", "ignored-agent-id", [], "copilot", null, null, null, null, null, restClient, taskContext);

      expect(result).toBe(true);
      expect(mockRequest).toHaveBeenCalledWith(
        "POST /agents/repos/{owner}/{repo}/tasks",
        expect.objectContaining({
          owner: "myorg",
          repo: "myrepo",
          create_pull_request: true,
          prompt: expect.stringContaining("myorg/myrepo#42"),
        })
      );
    });

    it("should include model in REST request when provided", async () => {
      const mockRequest = vi.fn().mockResolvedValue({ data: { id: "task-123" } });
      const restClient = { request: mockRequest };

      await assignAgentToIssue("id", "agent", [], "copilot", null, "claude-opus-4.6", null, null, null, restClient, taskContext);

      expect(mockRequest).toHaveBeenCalledWith("POST /agents/repos/{owner}/{repo}/tasks", expect.objectContaining({ model: "claude-opus-4.6" }));
    });

    it("should include base_ref in REST request when baseBranch is provided", async () => {
      const mockRequest = vi.fn().mockResolvedValue({ data: { id: "task-123" } });
      const restClient = { request: mockRequest };

      await assignAgentToIssue("id", "agent", [], "copilot", null, null, null, null, "develop", restClient, taskContext);

      expect(mockRequest).toHaveBeenCalledWith("POST /agents/repos/{owner}/{repo}/tasks", expect.objectContaining({ base_ref: "develop" }));
    });

    it("should not include model or base_ref when not provided", async () => {
      const mockRequest = vi.fn().mockResolvedValue({ data: { id: "task-123" } });
      const restClient = { request: mockRequest };

      await assignAgentToIssue("id", "agent", [], "copilot", null, null, null, null, null, restClient, taskContext);

      const callArgs = mockRequest.mock.calls[0][1];
      expect(callArgs.model).toBeUndefined();
      expect(callArgs.base_ref).toBeUndefined();
    });

    it("should use pullRequestRepoSlug as target when provided", async () => {
      const mockRequest = vi.fn().mockResolvedValue({ data: { id: "task-123" } });
      const restClient = { request: mockRequest };

      await assignAgentToIssue("id", "agent", [], "copilot", null, null, null, null, null, restClient, taskContext, "otherorg/otherrepo");

      expect(mockRequest).toHaveBeenCalledWith("POST /agents/repos/{owner}/{repo}/tasks", expect.objectContaining({ owner: "otherorg", repo: "otherrepo" }));
    });

    it("should return false and not call request when taskContext is missing", async () => {
      const mockRequest = vi.fn();
      const restClient = { request: mockRequest };

      const result = await assignAgentToIssue("id", "agent", [], "copilot", null, null, null, null, null, restClient, null);

      expect(result).toBe(false);
      expect(mockRequest).not.toHaveBeenCalled();
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Invalid assignment context"));
    });

    it("should return false when client has neither request nor graphql", async () => {
      const emptyClient = {};

      const result = await assignAgentToIssue("id", "agent", [], "copilot", null, null, null, null, null, emptyClient, taskContext);

      expect(result).toBe(false);
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("does not support REST requests"));
    });

    it("should treat 502 REST errors as success", async () => {
      const err502 = Object.assign(new Error("502 Bad Gateway"), { response: { status: 502 } });
      const mockRequest = vi.fn().mockRejectedValue(err502);
      const restClient = { request: mockRequest };

      const result = await assignAgentToIssue("id", "agent", [], "copilot", null, null, null, null, null, restClient, taskContext);

      expect(result).toBe(true);
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("502"));
    });

    it("should call logPermissionError for Bad credentials on REST path", async () => {
      const errAuth = new Error("Bad credentials");
      const mockRequest = vi.fn().mockRejectedValue(errAuth);
      const restClient = { request: mockRequest };

      const result = await assignAgentToIssue("id", "agent", [], "copilot", null, null, null, null, null, restClient, taskContext);

      expect(result).toBe(false);
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Insufficient permissions"));
    });
  });

  describe("generatePermissionErrorSummary", () => {
    it("should return markdown content with permission requirements", () => {
      const summary = generatePermissionErrorSummary();

      expect(summary).toContain("### ⚠️ Permission Requirements");
      expect(summary).toContain("actions: write");
      expect(summary).toContain("contents: write");
      expect(summary).toContain("agent-tasks: write");
      expect(summary).toContain("POST /agents/repos/{owner}/{repo}/tasks");
    });
  });

  describe("assignAgentToIssueByName", () => {
    it("should successfully assign copilot agent", async () => {
      // findAgent: checkUserCanBeAssigned + getByUsername
      mockGithub.rest.issues.checkUserCanBeAssigned.mockResolvedValueOnce({});
      mockGithub.rest.users.getByUsername.mockResolvedValueOnce({ data: { id: 999 } });
      // getIssueDetails
      mockGithub.rest.issues.get.mockResolvedValueOnce({
        data: { id: 1111, number: 123, assignees: [], html_url: "", title: "", body: "" },
      });
      // assignAgentToIssue (REST)
      mockGithub.request.mockResolvedValueOnce({ data: { id: "task-1" } });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot");

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("Looking for copilot coding agent...");
      expect(mockCore.info).toHaveBeenCalledWith("Found copilot coding agent (ID: 999)");
    });

    it("should return error for unsupported agent", async () => {
      const result = await assignAgentToIssueByName("owner", "repo", 123, "unknown");

      expect(result.success).toBe(false);
      expect(result.error).toContain("not supported");
      expect(mockCore.warning).toHaveBeenCalled();
    });

    it("should return error when agent is not available", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.rest.issues.checkUserCanBeAssigned.mockRejectedValue(err404);
      mockGithub.rest.issues.listAssignees.mockResolvedValue({ data: [] });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot");

      expect(result.success).toBe(false);
      expect(result.error).toContain("not available");
    });

    it("should report already assigned when agent is in assignees", async () => {
      // findAgent
      mockGithub.rest.issues.checkUserCanBeAssigned.mockResolvedValueOnce({});
      mockGithub.rest.users.getByUsername.mockResolvedValueOnce({ data: { id: 999 } });
      // getIssueDetails - agent already assigned
      mockGithub.rest.issues.get.mockResolvedValueOnce({
        data: {
          id: 1111,
          number: 123,
          assignees: [{ id: 999, login: "copilot-swe-agent" }],
          html_url: "",
          title: "",
          body: "",
        },
      });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot");

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("copilot is already assigned to issue #123");
    });

    it("should skip assignment when a secondary copilot alias is already assigned", async () => {
      // findAgent resolves via primary alias with id 999
      mockGithub.rest.issues.checkUserCanBeAssigned.mockResolvedValueOnce({});
      mockGithub.rest.users.getByUsername.mockResolvedValueOnce({ data: { id: 999 } });
      // getIssueDetails - a secondary alias is the current assignee (different id, same agent)
      mockGithub.rest.issues.get.mockResolvedValueOnce({
        data: {
          id: 1111,
          number: 123,
          assignees: [{ id: 888, login: "github-copilot-enterprise" }],
          html_url: "",
          title: "",
          body: "",
        },
      });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot");

      expect(result.success).toBe(true);
      expect(mockGithub.request).not.toHaveBeenCalled();
      expect(mockCore.info).toHaveBeenCalledWith("copilot is already assigned to issue #123");
    });
  });

  describe("getPullRequestDetails", () => {
    it("should return pull request ID, current assignees, and task context", async () => {
      mockGithub.rest.pulls.get.mockResolvedValueOnce({
        data: {
          id: 67890,
          number: 42,
          assignees: [
            { id: 1, login: "user1" },
            { id: 2, login: "user2" },
          ],
          html_url: "https://github.com/owner/repo/pull/42",
          title: "Test PR",
          body: "Test body",
        },
      });

      const result = await getPullRequestDetails("owner", "repo", 42);

      expect(result).toEqual({
        pullRequestId: "67890",
        currentAssignees: [
          { id: "1", login: "user1" },
          { id: "2", login: "user2" },
        ],
        htmlUrl: "https://github.com/owner/repo/pull/42",
        title: "Test PR",
        body: "Test body",
        taskContext: { owner: "owner", repo: "repo", type: "pull", number: 42 },
      });
    });

    it("should return null when pull request is not found", async () => {
      mockGithub.rest.pulls.get.mockResolvedValueOnce({ data: null });

      const result = await getPullRequestDetails("owner", "repo", 999);

      expect(result).toBeNull();
      expect(mockCore.error).toHaveBeenCalledWith("Could not get pull request data");
    });

    it("should return empty assignees when none exist", async () => {
      mockGithub.rest.pulls.get.mockResolvedValueOnce({
        data: {
          id: 67890,
          number: 42,
          assignees: [],
          html_url: "https://github.com/owner/repo/pull/42",
          title: "Test PR",
          body: "",
        },
      });

      const result = await getPullRequestDetails("owner", "repo", 42);

      expect(result).toEqual({
        pullRequestId: "67890",
        currentAssignees: [],
        htmlUrl: "https://github.com/owner/repo/pull/42",
        title: "Test PR",
        body: "",
        taskContext: { owner: "owner", repo: "repo", type: "pull", number: 42 },
      });
    });

    it("should re-throw REST errors", async () => {
      mockGithub.rest.pulls.get.mockRejectedValueOnce(new Error("API error"));

      await expect(getPullRequestDetails("owner", "repo", 42)).rejects.toThrow("API error");
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to get pull request details"));
    });
  });
});
