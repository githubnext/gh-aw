import { describe, it, expect, beforeEach, afterEach } from "vitest";

// Set up global mocks before importing the module
const mockCore = {
  debug: () => {},
  info: () => {},
  warning: () => {},
  error: () => {},
};
global.core = mockCore;

describe("generate_history_link.cjs", () => {
  let generateHistoryUrl;
  let generateHistoryLink;

  let originalEnv;

  beforeEach(async () => {
    originalEnv = { ...process.env };

    const module = await import("./generate_history_link.cjs");
    generateHistoryUrl = module.generateHistoryUrl;
    generateHistoryLink = module.generateHistoryLink;
  });

  afterEach(() => {
    // Restore environment by mutating process.env in place
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    }
    Object.assign(process.env, originalEnv);
  });

  describe("generateHistoryUrl", () => {
    describe("item type filtering", () => {
      it("should include is:issue qualifier for issue type", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).toContain("is%3Aissue");
        expect(url).toContain("type=issues");
      });

      it("should NOT include is:pr qualifier for pull_request type", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "pull_request",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).not.toContain("is%3Apr");
        expect(url).toContain("type=pullrequests");
      });

      it("should NOT include is: qualifier for discussion type", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "discussion",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).not.toContain("is%3A");
        expect(url).toContain("type=discussions");
      });

      it("should NOT include is: qualifier for discussion_comment type", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "discussion_comment",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).not.toContain("is%3A");
        expect(url).toContain("type=discussions");
      });
    });

    describe("workflow ID selection", () => {
      it("should prefer workflowCallId over workflowId when both are provided", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowCallId: "caller/repo/WorkflowName",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).toContain("gh-aw-workflow-call-id");
        expect(url).not.toContain("gh-aw-workflow-id");
      });

      it("should use workflowId when workflowCallId is not provided", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).toContain("gh-aw-workflow-id");
        expect(url).not.toContain("gh-aw-workflow-call-id");
      });

      it("should use workflowId when workflowCallId is empty string", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowCallId: "",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).toContain("gh-aw-workflow-id");
        expect(url).not.toContain("gh-aw-workflow-call-id");
      });

      it("should return null when neither workflowCallId nor workflowId is provided", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          serverUrl: "https://github.com",
        });

        expect(url).toBeNull();
      });

      it("should return null when both workflowCallId and workflowId are empty strings", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowCallId: "",
          workflowId: "",
          serverUrl: "https://github.com",
        });

        expect(url).toBeNull();
      });
    });

    describe("search query content", () => {
      it("should include repo qualifier in search query", () => {
        const url = generateHistoryUrl({
          owner: "myowner",
          repo: "myrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).toContain("repo%3Amyowner%2Fmyrepo");
      });

      it("should include workflow-call-id marker in search query when provided", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowCallId: "caller/repo/WorkflowName",
          serverUrl: "https://github.com",
        });

        expect(url).toContain("gh-aw-workflow-call-id%3A+caller%2Frepo%2FWorkflowName");
      });

      it("should include workflow-id marker in search query when used", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).toContain("gh-aw-workflow-id%3A+my-workflow");
      });

      it("should NOT include open/closed state filter", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).not.toContain("is%3Aopen");
        expect(url).not.toContain("is%3Aclosed");
        expect(url).not.toContain("is:open");
        expect(url).not.toContain("is:closed");
      });
    });

    describe("enterprise deployment", () => {
      it("should use provided serverUrl for enterprise deployments", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.mycompany.com",
        });

        expect(url).toMatch(/^https:\/\/github\.mycompany\.com\/search/);
      });

      it("should use GITHUB_SERVER_URL env var when serverUrl is not provided", () => {
        process.env.GITHUB_SERVER_URL = "https://github.enterprise.example.com";

        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
        });

        expect(url).toMatch(/^https:\/\/github\.enterprise\.example\.com\/search/);
      });

      it("should default to https://github.com when no server URL is configured", () => {
        delete process.env.GITHUB_SERVER_URL;

        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
        });

        expect(url).toMatch(/^https:\/\/github\.com\/search/);
      });

      it("should prefer serverUrl param over GITHUB_SERVER_URL env var", () => {
        process.env.GITHUB_SERVER_URL = "https://github.enterprise.example.com";

        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.mycompany.com",
        });

        expect(url).toMatch(/^https:\/\/github\.mycompany\.com\/search/);
      });
    });

    describe("URL structure", () => {
      it("should generate a valid URL with /search path", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).toMatch(/^https:\/\/github\.com\/search\?/);
      });

      it("should include q parameter in URL", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).toContain("?q=");
      });

      it("should include type parameter in URL", () => {
        const url = generateHistoryUrl({
          owner: "testowner",
          repo: "testrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        expect(url).toContain("type=issues");
      });

      it("should generate a complete issue search URL", () => {
        const url = generateHistoryUrl({
          owner: "myowner",
          repo: "myrepo",
          itemType: "issue",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        const parsed = new URL(url);
        expect(parsed.hostname).toBe("github.com");
        expect(parsed.pathname).toBe("/search");
        expect(parsed.searchParams.get("type")).toBe("issues");

        const query = parsed.searchParams.get("q");
        expect(query).toContain("repo:myowner/myrepo");
        expect(query).toContain("is:issue");
        expect(query).toContain('"gh-aw-workflow-id: my-workflow"');
      });

      it("should generate a complete pull_request search URL", () => {
        const url = generateHistoryUrl({
          owner: "myowner",
          repo: "myrepo",
          itemType: "pull_request",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        const parsed = new URL(url);
        expect(parsed.searchParams.get("type")).toBe("pullrequests");

        const query = parsed.searchParams.get("q");
        expect(query).not.toContain("is:pr");
        expect(query).toContain('"gh-aw-workflow-id: my-workflow"');
      });

      it("should generate a complete discussion search URL", () => {
        const url = generateHistoryUrl({
          owner: "myowner",
          repo: "myrepo",
          itemType: "discussion",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        const parsed = new URL(url);
        expect(parsed.searchParams.get("type")).toBe("discussions");

        const query = parsed.searchParams.get("q");
        expect(query).not.toContain("is:issue");
        expect(query).not.toContain("is:pr");
        expect(query).toContain('"gh-aw-workflow-id: my-workflow"');
      });

      it("should generate a complete discussion_comment search URL", () => {
        const url = generateHistoryUrl({
          owner: "myowner",
          repo: "myrepo",
          itemType: "discussion_comment",
          workflowId: "my-workflow",
          serverUrl: "https://github.com",
        });

        const parsed = new URL(url);
        expect(parsed.searchParams.get("type")).toBe("discussions");

        const query = parsed.searchParams.get("q");
        expect(query).not.toContain("is:issue");
        expect(query).not.toContain("is:pr");
        expect(query).toContain('"gh-aw-workflow-id: my-workflow"');
      });

      it("should generate correct URL with workflowCallId", () => {
        const url = generateHistoryUrl({
          owner: "myowner",
          repo: "myrepo",
          itemType: "issue",
          workflowCallId: "callerowner/callerrepo/WorkflowName",
          serverUrl: "https://github.com",
        });

        const parsed = new URL(url);
        const query = parsed.searchParams.get("q");
        expect(query).toContain('"gh-aw-workflow-call-id: callerowner/callerrepo/WorkflowName"');
        expect(query).not.toContain("gh-aw-workflow-id");
      });
    });
  });

  describe("generateHistoryLink", () => {
    it("should return null when no workflow ID is available", () => {
      const link = generateHistoryLink({
        owner: "testowner",
        repo: "testrepo",
        itemType: "issue",
        serverUrl: "https://github.com",
      });

      expect(link).toBeNull();
    });

    it("should return a markdown link with clock symbol", () => {
      const link = generateHistoryLink({
        owner: "testowner",
        repo: "testrepo",
        itemType: "issue",
        workflowId: "my-workflow",
        serverUrl: "https://github.com",
      });

      expect(link).toMatch(/^\[history\]\(https:\/\/github\.com\/search\?/);
    });

    it("should use 'history' as the link label", () => {
      const link = generateHistoryLink({
        owner: "testowner",
        repo: "testrepo",
        itemType: "issue",
        workflowId: "my-workflow",
        serverUrl: "https://github.com",
      });

      expect(link).not.toContain("🕐");
      expect(link).toContain("history");
    });

    it("should wrap the search URL in markdown link syntax", () => {
      const link = generateHistoryLink({
        owner: "testowner",
        repo: "testrepo",
        itemType: "issue",
        workflowId: "my-workflow",
        serverUrl: "https://github.com",
      });

      // Should be valid markdown link: [text](url)
      expect(link).toMatch(/^\[.+\]\(https?:\/\/.+\)$/);
    });

    it("should generate link with correct search URL for issue", () => {
      const link = generateHistoryLink({
        owner: "myowner",
        repo: "myrepo",
        itemType: "issue",
        workflowId: "my-workflow",
        serverUrl: "https://github.com",
      });

      expect(link).toContain("type=issues");
      expect(link).toContain("is%3Aissue");
    });

    it("should generate link with correct search URL for pull_request", () => {
      const link = generateHistoryLink({
        owner: "myowner",
        repo: "myrepo",
        itemType: "pull_request",
        workflowId: "my-workflow",
        serverUrl: "https://github.com",
      });

      expect(link).toContain("type=pullrequests");
    });

    it("should generate link with correct search URL for discussion", () => {
      const link = generateHistoryLink({
        owner: "myowner",
        repo: "myrepo",
        itemType: "discussion",
        workflowId: "my-workflow",
        serverUrl: "https://github.com",
      });

      expect(link).toContain("type=discussions");
    });

    it("should support enterprise URLs in the history link", () => {
      const link = generateHistoryLink({
        owner: "myowner",
        repo: "myrepo",
        itemType: "issue",
        workflowId: "my-workflow",
        serverUrl: "https://github.mycompany.com",
      });

      expect(link).toContain("https://github.mycompany.com/search");
    });
  });
});
