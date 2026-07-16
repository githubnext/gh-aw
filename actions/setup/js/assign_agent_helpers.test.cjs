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
        copilot: ["copilot-swe-agent[bot]", "github-copilot-enterprise[bot]", "github-copilot[bot]", "copilot-swe-agent", "github-copilot-enterprise", "github-copilot"],
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
      expect(getAgentLogins("copilot")).toEqual(["copilot-swe-agent[bot]", "github-copilot-enterprise[bot]", "github-copilot[bot]", "copilot-swe-agent", "github-copilot-enterprise", "github-copilot"]);
    });

    it("should return empty array for unknown agents", () => {
      expect(getAgentLogins("unknown")).toEqual([]);
    });
  });

  describe("getAvailableAgentLogins", () => {
    it("should return available agent logins when an alias is assignable", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.request.mockRejectedValueOnce(err404).mockResolvedValueOnce({ status: 204 }).mockRejectedValue(err404);

      const result = await getAvailableAgentLogins("owner", "repo", 123);

      expect(result).toEqual(["github-copilot-enterprise[bot]"]);
      expect(mockGithub.request).toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/{issue_number}/assignees/{assignee}", {
        owner: "owner",
        repo: "repo",
        issue_number: 123,
        assignee: "copilot-swe-agent[bot]",
      });
      expect(mockGithub.request).toHaveBeenCalledTimes(6);
    });

    it("should return empty array when no alias is assignable (404)", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.request.mockRejectedValue(err404);

      const result = await getAvailableAgentLogins("owner", "repo", 123);

      expect(result).toEqual([]);
      expect(mockGithub.request).toHaveBeenCalledTimes(6);
    });

    it("should handle non-404 errors gracefully and return empty array", async () => {
      const err500 = Object.assign(new Error("Server error"), { status: 500 });
      mockGithub.request.mockRejectedValue(err500);

      const result = await getAvailableAgentLogins("owner", "repo", 123);

      expect(result).toEqual([]);
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Failed to check assignability for copilot-swe-agent"));
      expect(mockGithub.request).toHaveBeenCalledTimes(6);
    });

    it("should use issue-scoped assignee checks when issue number is provided", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.request.mockRejectedValueOnce(err404).mockResolvedValueOnce({ status: 204 }).mockRejectedValue(err404);

      const result = await getAvailableAgentLogins("owner", "repo", 123);

      expect(result).toEqual(["github-copilot-enterprise[bot]"]);
      expect(mockGithub.request).toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/{issue_number}/assignees/{assignee}", {
        owner: "owner",
        repo: "repo",
        issue_number: 123,
        assignee: "copilot-swe-agent[bot]",
      });
      expect(mockGithub.request).toHaveBeenCalledTimes(6);
    });

    it("should return empty array when issue number is invalid", async () => {
      const result = await getAvailableAgentLogins("owner", "repo", "not-a-number");
      expect(result).toEqual([]);
      expect(mockGithub.request).not.toHaveBeenCalled();
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
    it("should find copilot agent via a fallback alias and return login via REST", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.request.mockRejectedValueOnce(err404).mockResolvedValueOnce({ status: 204 });

      const result = await findAgent("owner", "repo", "copilot", 123);

      expect(result).toBe("github-copilot-enterprise[bot]");
      expect(mockGithub.request).toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/{issue_number}/assignees/{assignee}", {
        owner: "owner",
        repo: "repo",
        issue_number: 123,
        assignee: "copilot-swe-agent[bot]",
      });
      expect(mockGithub.request).toHaveBeenCalledTimes(2);
    });

    it("should return null for unknown agent name", async () => {
      const result = await findAgent("owner", "repo", "unknown-agent");

      expect(result).toBeNull();
      expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Unknown agent: unknown-agent"));
    });

    it("should return null and log bots when no copilot alias is available (404)", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.request.mockRejectedValue(err404);
      mockGithub.rest.issues.listAssignees.mockResolvedValue({ data: [{ login: "github-actions[bot]", type: "Bot" }] });

      const result = await findAgent("owner", "repo", "copilot", 123);

      expect(result).toBeNull();
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("copilot coding agent aliases are not available"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Assignable bots in this repository: github-actions[bot]"));
    });

    it("should handle REST errors and re-throw auth errors", async () => {
      const authError = new Error("Bad credentials");
      mockGithub.request.mockRejectedValueOnce(authError);

      await expect(findAgent("owner", "repo", "copilot", 123)).rejects.toThrow("Bad credentials");
    });

    it("should return null for non-auth errors", async () => {
      const err = new Error("Something unexpected");
      mockGithub.request.mockRejectedValueOnce(err);
      mockGithub.request.mockRejectedValue(Object.assign(new Error("Not Found"), { status: 404 }));
      mockGithub.rest.issues.listAssignees.mockResolvedValue({ data: [] });

      const result = await findAgent("owner", "repo", "copilot", 123);

      expect(result).toBeNull();
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Assignee alias copilot-swe-agent was not assignable"));
    });

    it("should prefer issue-scoped assignee checks when issue number is provided", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.request.mockRejectedValueOnce(err404).mockResolvedValueOnce({ status: 204 });

      const result = await findAgent("owner", "repo", "copilot", 321);

      expect(result).toBe("github-copilot-enterprise[bot]");
      expect(mockGithub.request).toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/{issue_number}/assignees/{assignee}", {
        owner: "owner",
        repo: "repo",
        issue_number: 321,
        assignee: "copilot-swe-agent[bot]",
      });
      expect(mockGithub.request).toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/{issue_number}/assignees/{assignee}", {
        owner: "owner",
        repo: "repo",
        issue_number: 321,
        assignee: "github-copilot-enterprise[bot]",
      });
      expect(mockGithub.request).toHaveBeenCalledTimes(2);
    });

    it("should return null when issue number is invalid", async () => {
      const result = await findAgent("owner", "repo", "copilot", "not-a-number");

      expect(result).toBeNull();
      expect(mockGithub.request).not.toHaveBeenCalled();
    });

    it("should return null when github client has no request method", async () => {
      const clientWithoutRequest = {
        rest: {
          issues: {
            listAssignees: vi.fn().mockResolvedValue({ data: [] }),
          },
        },
      };

      const result = await findAgent("owner", "repo", "copilot", 123, clientWithoutRequest);

      expect(result).toBeNull();
      expect(clientWithoutRequest.rest.issues.listAssignees).toHaveBeenCalled();
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

    it("should assign via issues assignees REST endpoint", async () => {
      const mockRequest = vi.fn().mockResolvedValue({ status: 201 });
      const restClient = { request: mockRequest };

      const result = await assignAgentToIssue("ignored-id", "copilot-swe-agent[bot]", [], "copilot", null, null, null, null, null, restClient, taskContext);

      expect(result).toBe(true);
      expect(mockRequest).toHaveBeenCalledWith("POST /repos/{owner}/{repo}/issues/{issue_number}/assignees", {
        owner: "myorg",
        repo: "myrepo",
        issue_number: 42,
        assignees: ["copilot-swe-agent[bot]"],
      });
    });

    it("should use taskContext repository even when pullRequestRepoSlug is provided", async () => {
      const mockRequest = vi.fn().mockResolvedValue({ status: 201 });
      const restClient = { request: mockRequest };

      await assignAgentToIssue("id", "copilot-swe-agent[bot]", [], "copilot", null, null, null, null, null, restClient, taskContext, "otherorg/otherrepo");

      expect(mockRequest).toHaveBeenCalledWith("POST /repos/{owner}/{repo}/issues/{issue_number}/assignees", expect.objectContaining({ owner: "myorg", repo: "myrepo", issue_number: 42 }));
    });

    it("should include issue-intent metadata when enabled", async () => {
      const mockRequest = vi.fn().mockResolvedValue({ status: 201 });
      const restClient = { request: mockRequest };

      await assignAgentToIssue("id", "copilot-swe-agent[bot]", [], "copilot", null, null, null, null, null, restClient, taskContext, null, { rationale: "Agent owns the code path", confidence: "HIGH" }, true);

      expect(mockRequest).toHaveBeenCalledWith("POST /repos/{owner}/{repo}/issues/{issue_number}/assignees", {
        owner: "myorg",
        repo: "myrepo",
        issue_number: 42,
        assignees: ["copilot-swe-agent[bot]"],
        rationale: "Agent owns the code path",
        confidence: "HIGH",
      });
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

    it("should return false on REST errors", async () => {
      const err502 = Object.assign(new Error("502 Bad Gateway"), { response: { status: 502 } });
      const mockRequest = vi.fn().mockRejectedValue(err502);
      const restClient = { request: mockRequest };

      const result = await assignAgentToIssue("id", "agent", [], "copilot", null, null, null, null, null, restClient, taskContext);

      expect(result).toBe(false);
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
      expect(summary).toContain("Fine-grained personal access token");
      expect(summary).toContain("actions**, **contents**, **issues**");
      expect(summary).toContain("POST /repos/{owner}/{repo}/issues/{issue_number}/assignees");
      expect(summary).toContain("https://docs.github.com/en/copilot/how-tos/use-copilot-agents/cloud-agent/use-cloud-agent-via-the-api#using-the-issues-api");
    });
  });

  describe("assignAgentToIssueByName", () => {
    it("should successfully assign copilot agent", async () => {
      // findAgent: issue-scoped assignee check
      mockGithub.request.mockResolvedValueOnce({ status: 204 });
      // getIssueDetails
      mockGithub.rest.issues.get.mockResolvedValueOnce({
        data: { id: 1111, number: 123, assignees: [], html_url: "", title: "", body: "" },
      });
      // assignAgentToIssue (REST assignees)
      mockGithub.request.mockResolvedValueOnce({ status: 201 });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot");

      expect(result.success).toBe(true);
      expect(mockCore.info).toHaveBeenCalledWith("Looking for copilot coding agent...");
      expect(mockCore.info).toHaveBeenCalledWith("Found copilot coding agent (login: copilot-swe-agent[bot])");
    });

    it("should return error for unsupported agent", async () => {
      const result = await assignAgentToIssueByName("owner", "repo", 123, "unknown");

      expect(result.success).toBe(false);
      expect(result.error).toContain("not supported");
      expect(mockCore.warning).toHaveBeenCalled();
    });

    it("should return error when agent is not available", async () => {
      const err404 = Object.assign(new Error("Not Found"), { status: 404 });
      mockGithub.request.mockRejectedValue(err404);
      mockGithub.rest.issues.listAssignees.mockResolvedValue({ data: [] });

      const result = await assignAgentToIssueByName("owner", "repo", 123, "copilot");

      expect(result.success).toBe(false);
      expect(result.error).toContain("not available");
    });

    it("should report already assigned when agent is in assignees", async () => {
      // findAgent
      mockGithub.request.mockResolvedValueOnce({ status: 204 });
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
      // findAgent resolves via primary alias
      mockGithub.request.mockResolvedValueOnce({ status: 204 });
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
      expect(mockGithub.request).toHaveBeenCalledTimes(1);
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
