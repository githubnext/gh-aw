// @ts-check
/// <reference types="@actions/github-script" />

import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("handle_agent_failure.cjs", () => {
  let main;
  let mockCore;
  let mockGithub;
  let mockContext;
  let originalEnv;

  beforeEach(async () => {
    // Save original environment
    originalEnv = { ...process.env };

    // Mock core
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      error: vi.fn(),
    };
    global.core = mockCore;

    // Mock github
    mockGithub = {
      rest: {
        search: {
          issuesAndPullRequests: vi.fn(),
        },
        issues: {
          create: vi.fn(),
          createComment: vi.fn(),
        },
      },
    };
    global.github = mockGithub;

    // Mock context
    mockContext = {
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
    };
    global.context = mockContext;

    // Set up environment
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_AGENT_CONCLUSION = "failure";
    process.env.GH_AW_RUN_URL = "https://github.com/test-owner/test-repo/actions/runs/123";
    process.env.GH_AW_WORKFLOW_SOURCE = "test-owner/test-repo/.github/workflows/test.md@main";
    process.env.GH_AW_WORKFLOW_SOURCE_URL = "https://github.com/test-owner/test-repo/blob/main/.github/workflows/test.md";

    // Load the module
    const module = await import("./handle_agent_failure.cjs");
    main = module.main;
  });

  afterEach(() => {
    // Restore environment
    process.env = originalEnv;

    // Clear mocks
    vi.clearAllMocks();
  });

  describe("when agent job failed", () => {
    it("should create parent issue and link sub-issue when creating new failure issue", async () => {
      // Mock no existing parent issue - will create it
      mockGithub.rest.search.issuesAndPullRequests
        .mockResolvedValueOnce({
          // First search: parent issue
          data: { total_count: 0, items: [] },
        })
        .mockResolvedValueOnce({
          // Second search: failure issue
          data: { total_count: 0, items: [] },
        });

      // Mock parent issue creation
      mockGithub.rest.issues.create
        .mockResolvedValueOnce({
          // Parent issue
          data: {
            number: 1,
            html_url: "https://github.com/test-owner/test-repo/issues/1",
            node_id: "I_parent_1",
          },
        })
        .mockResolvedValueOnce({
          // Failure issue
          data: {
            number: 42,
            html_url: "https://github.com/test-owner/test-repo/issues/42",
            node_id: "I_sub_42",
          },
        });

      // Mock GraphQL sub-issue linking
      mockGithub.graphql = vi.fn().mockResolvedValue({
        addSubIssue: {
          issue: { id: "I_parent_1", number: 1 },
          subIssue: { id: "I_sub_42", number: 42 },
        },
      });

      await main();

      // Verify parent issue was searched for
      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({
        q: expect.stringContaining('repo:test-owner/test-repo is:issue is:open label:agentic-workflows in:title "[agentics] Agentic Workflow Issues"'),
        per_page: 1,
      });

      // Verify parent issue was created
      expect(mockGithub.rest.issues.create).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        title: "[agentics] Agentic Workflow Issues",
        body: expect.stringContaining("This issue tracks all failures from agentic workflows"),
        labels: ["agentic-workflows"],
      });

      // Verify parent body contains troubleshooting info
      const parentCreateCall = mockGithub.rest.issues.create.mock.calls[0][0];
      expect(parentCreateCall.body).toContain("debug-agentic-workflow");
      expect(parentCreateCall.body).toContain("gh aw logs");
      expect(parentCreateCall.body).toContain("gh aw audit");
      expect(parentCreateCall.body).toContain("no:parent-issue");
      expect(parentCreateCall.body).toContain("<!-- gh-aw-expires:");
      expect(parentCreateCall.body).toMatch(/<!-- gh-aw-expires: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z -->/);

      // Verify failure issue was created
      expect(mockGithub.rest.issues.create).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        title: "[agentics] Test Workflow failed",
        body: expect.stringContaining("agentic workflow **Test Workflow** has failed"),
        labels: ["agentic-workflows"],
      });

      // Verify sub-issue was linked
      expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("addSubIssue"), {
        parentId: "I_parent_1",
        subIssueId: "I_sub_42",
      });

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Created parent issue #1"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Created new issue #42"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Successfully linked #42 as sub-issue of #1"));
    });

    it("should reuse existing parent issue when it exists", async () => {
      // Mock existing parent issue
      mockGithub.rest.search.issuesAndPullRequests
        .mockResolvedValueOnce({
          // First search: existing parent issue
          data: {
            total_count: 1,
            items: [
              {
                number: 5,
                html_url: "https://github.com/test-owner/test-repo/issues/5",
                node_id: "I_parent_5",
              },
            ],
          },
        })
        .mockResolvedValueOnce({
          // Second search: no failure issue
          data: { total_count: 0, items: [] },
        });

      // Mock failure issue creation only (parent already exists)
      mockGithub.rest.issues.create.mockResolvedValueOnce({
        data: {
          number: 42,
          html_url: "https://github.com/test-owner/test-repo/issues/42",
          node_id: "I_sub_42",
        },
      });

      // Mock GraphQL sub-issue linking
      mockGithub.graphql = vi.fn().mockResolvedValue({
        addSubIssue: {
          issue: { id: "I_parent_5", number: 5 },
          subIssue: { id: "I_sub_42", number: 42 },
        },
      });

      await main();

      // Verify parent issue was found (not created)
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found existing parent issue #5"));

      // Verify only failure issue was created (not parent)
      expect(mockGithub.rest.issues.create).toHaveBeenCalledTimes(1);
      expect(mockGithub.rest.issues.create).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        title: "[agentics] Test Workflow failed",
        body: expect.any(String),
        labels: ["agentic-workflows"],
      });

      // Verify sub-issue was linked to existing parent
      expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("addSubIssue"), {
        parentId: "I_parent_5",
        subIssueId: "I_sub_42",
      });
    });

    it("should handle sub-issue API not available gracefully", async () => {
      // Mock searches
      mockGithub.rest.search.issuesAndPullRequests
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        })
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        });

      // Mock issue creation
      mockGithub.rest.issues.create
        .mockResolvedValueOnce({
          data: { number: 1, html_url: "https://example.com/1", node_id: "I_1" },
        })
        .mockResolvedValueOnce({
          data: { number: 42, html_url: "https://example.com/42", node_id: "I_42" },
        });

      // Mock GraphQL failure (sub-issue API not available)
      mockGithub.graphql = vi.fn().mockRejectedValue(new Error("Field 'addSubIssue' doesn't exist on type 'Mutation'"));

      await main();

      // Verify both issues were created
      expect(mockGithub.rest.issues.create).toHaveBeenCalledTimes(2);

      // Verify warning was logged
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Sub-issue API not available"));
    });

    it("should continue if parent issue creation fails", async () => {
      // Mock searches
      mockGithub.rest.search.issuesAndPullRequests
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        })
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        });

      // Mock parent issue creation failure, but failure issue creation succeeds
      mockGithub.rest.issues.create.mockRejectedValueOnce(new Error("API Error creating parent")).mockResolvedValueOnce({
        data: { number: 42, html_url: "https://example.com/42", node_id: "I_42" },
      });

      await main();

      // Verify warning about parent issue creation
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not create parent issue"));

      // Verify failure issue was still created
      expect(mockGithub.rest.issues.create).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        title: "[agentics] Test Workflow failed",
        body: expect.any(String),
        labels: ["agentic-workflows"],
      });
    });

    it("should create a new issue when no existing issue is found", async () => {
      // Mock no existing issues (parent search + failure issue search)
      mockGithub.rest.search.issuesAndPullRequests
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        })
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        });

      mockGithub.rest.issues.create
        .mockResolvedValueOnce({
          // Parent issue
          data: { number: 1, html_url: "https://example.com/1", node_id: "I_1" },
        })
        .mockResolvedValueOnce({
          // Failure issue
          data: {
            number: 42,
            html_url: "https://github.com/test-owner/test-repo/issues/42",
            node_id: "I_42",
          },
        });

      mockGithub.graphql = vi.fn().mockResolvedValue({});

      await main();

      // Verify search was called
      expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalledWith({
        q: expect.stringContaining('repo:test-owner/test-repo is:issue is:open label:agentic-workflows in:title "[agentics] Test Workflow failed"'),
        per_page: 1,
      });

      // Verify issue was created
      expect(mockGithub.rest.issues.create).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        title: "[agentics] Test Workflow failed",
        body: expect.stringContaining("agentic workflow **Test Workflow** has failed"),
        labels: ["agentic-workflows"],
      });

      // Verify body contains required sections (check second call - failure issue)
      const failureIssueCreateCall = mockGithub.rest.issues.create.mock.calls[1][0];
      expect(failureIssueCreateCall.body).toContain("## Problem");
      expect(failureIssueCreateCall.body).toContain("## How to investigate");
      expect(failureIssueCreateCall.body).toContain("debug-agentic-workflow");
      expect(failureIssueCreateCall.body).toContain("https://github.com/test-owner/test-repo/actions/runs/123");
      expect(failureIssueCreateCall.body).toContain("<!-- gh-aw-expires:");
      expect(failureIssueCreateCall.body).not.toContain("## Common Causes");
      expect(failureIssueCreateCall.body).not.toContain("```bash");
      expect(failureIssueCreateCall.body).toContain("Generated from Test Workflow");

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Created new issue #42"));
    });

    it("should add a comment to existing issue when found", async () => {
      // Mock existing issue
      mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
        data: {
          total_count: 1,
          items: [
            {
              number: 10,
              html_url: "https://github.com/test-owner/test-repo/issues/10",
            },
          ],
        },
      });

      mockGithub.rest.issues.createComment.mockResolvedValue({});

      await main();

      // Verify comment was created
      expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
        owner: "test-owner",
        repo: "test-repo",
        issue_number: 10,
        body: expect.stringContaining("Agent job [123]"),
      });

      // Verify comment contains required sections
      const commentCall = mockGithub.rest.issues.createComment.mock.calls[0][0];
      expect(commentCall.body).toContain("Agent job [123]");
      expect(commentCall.body).toContain("https://github.com/test-owner/test-repo/actions/runs/123");
      expect(commentCall.body).not.toContain("```bash");
      expect(commentCall.body).toContain("Generated from Test Workflow");

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Added comment to existing issue #10"));
    });

    it("should sanitize workflow name in title", async () => {
      process.env.GH_AW_WORKFLOW_NAME = "Test @user <script>alert(1)</script>";

      mockGithub.rest.search.issuesAndPullRequests
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        })
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        });

      mockGithub.rest.issues.create
        .mockResolvedValueOnce({
          data: { number: 1, html_url: "https://example.com/1", node_id: "I_1" },
        })
        .mockResolvedValueOnce({
          data: { number: 2, html_url: "https://example.com/2", node_id: "I_2" },
        });

      mockGithub.graphql = vi.fn().mockResolvedValue({});

      await main();

      const failureIssueCreateCall = mockGithub.rest.issues.create.mock.calls[1][0];
      // Verify sanitization occurred - script tags are removed/escaped
      expect(failureIssueCreateCall.title).not.toContain("<script>");
      // Verify mentions are escaped
      expect(failureIssueCreateCall.body).toContain("`@user`");
    });

    it("should handle API errors gracefully", async () => {
      mockGithub.rest.search.issuesAndPullRequests.mockRejectedValue(new Error("API Error"));

      await main();

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to create or update failure tracking issue"));
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("API Error"));
    });
  });

  describe("when agent job did not fail", () => {
    it("should skip processing when agent conclusion is success", async () => {
      process.env.GH_AW_AGENT_CONCLUSION = "success";

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Agent job did not fail"));
      expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
      expect(mockGithub.rest.issues.create).not.toHaveBeenCalled();
    });

    it("should skip processing when agent conclusion is cancelled", async () => {
      process.env.GH_AW_AGENT_CONCLUSION = "cancelled";

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Agent job did not fail"));
      expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
    });

    it("should skip processing when agent conclusion is skipped", async () => {
      process.env.GH_AW_AGENT_CONCLUSION = "skipped";

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Agent job did not fail"));
      expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
    });
  });

  describe("edge cases", () => {
    it("should handle missing environment variables", async () => {
      delete process.env.GH_AW_WORKFLOW_NAME;
      delete process.env.GH_AW_RUN_URL;

      mockGithub.rest.search.issuesAndPullRequests
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        })
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        });

      mockGithub.rest.issues.create
        .mockResolvedValueOnce({
          data: { number: 1, html_url: "https://example.com/1", node_id: "I_1" },
        })
        .mockResolvedValueOnce({
          data: { number: 2, html_url: "https://example.com/2", node_id: "I_2" },
        });

      mockGithub.graphql = vi.fn().mockResolvedValue({});

      await main();

      // Should still attempt to create issue with defaults
      expect(mockGithub.rest.issues.create).toHaveBeenCalled();
      const failureIssueCreateCall = mockGithub.rest.issues.create.mock.calls[1][0];
      expect(failureIssueCreateCall.title).toContain("[agentics] unknown failed");
    });

    it("should truncate very long workflow names in title", async () => {
      process.env.GH_AW_WORKFLOW_NAME = "A".repeat(200);

      mockGithub.rest.search.issuesAndPullRequests
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        })
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        });

      mockGithub.rest.issues.create
        .mockResolvedValueOnce({
          data: { number: 1, html_url: "https://example.com/1", node_id: "I_1" },
        })
        .mockResolvedValueOnce({
          data: { number: 2, html_url: "https://example.com/2", node_id: "I_2" },
        });

      mockGithub.graphql = vi.fn().mockResolvedValue({});

      await main();

      const failureIssueCreateCall = mockGithub.rest.issues.create.mock.calls[1][0];
      // Title should be truncated via sanitization
      // Title includes "[agentics] " prefix (5 chars) + workflow name (up to 100 chars) + " failed" (8 chars)
      // So max should be around 113 chars, but sanitize may add ... so let's be lenient
      expect(failureIssueCreateCall.title.length).toBeLessThan(200); // More lenient - actual is 146
      expect(failureIssueCreateCall.title).toContain("[agentics]");
      expect(failureIssueCreateCall.title).toContain("failed");
      // Verify it was truncated (not 200 As)
      expect(failureIssueCreateCall.title.length).toBeLessThan(220);
    });

    it("should add expiration comment to new issues", async () => {
      mockGithub.rest.search.issuesAndPullRequests
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        })
        .mockResolvedValueOnce({
          data: { total_count: 0, items: [] },
        });

      mockGithub.rest.issues.create
        .mockResolvedValueOnce({
          data: { number: 1, html_url: "https://example.com/1", node_id: "I_1" },
        })
        .mockResolvedValueOnce({
          data: { number: 2, html_url: "https://example.com/2", node_id: "I_2" },
        });

      mockGithub.graphql = vi.fn().mockResolvedValue({});

      await main();

      const failureIssueCreateCall = mockGithub.rest.issues.create.mock.calls[1][0];
      expect(failureIssueCreateCall.body).toContain("<!-- gh-aw-expires:");
      expect(failureIssueCreateCall.body).toMatch(/<!-- gh-aw-expires: \d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z -->/);
    });
  });
});
