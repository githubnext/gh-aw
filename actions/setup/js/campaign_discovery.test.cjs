// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { normalizeItem, loadCursor, saveCursor, searchByTrackerId, searchByLabel, discover } from "./campaign_discovery.cjs";
import fs from "fs";
import path from "path";

// Mock fs
vi.mock("fs");

// Mock core and github
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  getInput: vi.fn(),
  setOutput: vi.fn(),
};

global.github = {};

describe("campaign_discovery", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("normalizeItem", () => {
    it("should normalize an issue", () => {
      const issue = {
        html_url: "https://github.com/owner/repo/issues/1",
        number: 1,
        repository: { full_name: "owner/repo" },
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
        state: "open",
        title: "Test Issue",
      };

      const normalized = normalizeItem(issue, "issue");

      expect(normalized).toEqual({
        url: "https://github.com/owner/repo/issues/1",
        content_type: "issue",
        number: 1,
        repo: "owner/repo",
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
        state: "open",
        title: "Test Issue",
      });
    });

    it("should normalize a pull request with merged_at", () => {
      const pr = {
        html_url: "https://github.com/owner/repo/pull/2",
        number: 2,
        repository: { full_name: "owner/repo" },
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
        state: "closed",
        title: "Test PR",
        merged_at: "2025-01-03T00:00:00Z",
      };

      const normalized = normalizeItem(pr, "pull_request");

      expect(normalized).toEqual({
        url: "https://github.com/owner/repo/pull/2",
        content_type: "pull_request",
        number: 2,
        repo: "owner/repo",
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
        state: "closed",
        title: "Test PR",
        merged_at: "2025-01-03T00:00:00Z",
      });
    });

    it("should normalize a closed issue with closed_at", () => {
      const issue = {
        html_url: "https://github.com/owner/repo/issues/3",
        number: 3,
        repository: { full_name: "owner/repo" },
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
        state: "closed",
        title: "Closed Issue",
        closed_at: "2025-01-03T00:00:00Z",
      };

      const normalized = normalizeItem(issue, "issue");

      expect(normalized).toEqual({
        url: "https://github.com/owner/repo/issues/3",
        content_type: "issue",
        number: 3,
        repo: "owner/repo",
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
        state: "closed",
        title: "Closed Issue",
        closed_at: "2025-01-03T00:00:00Z",
      });
    });
  });

  describe("loadCursor", () => {
    it("should handle missing cursor file gracefully", () => {
      // Since we can't easily mock fs in vitest with CommonJS,
      // we'll just test that the function doesn't throw
      const cursor = loadCursor("/nonexistent/path.json");
      expect(cursor).toBeNull();
    });
  });

  describe("saveCursor", () => {
    it("should be defined as a function", () => {
      expect(typeof saveCursor).toBe("function");
    });
  });

  describe("searchByTrackerId", () => {
    it("should search for items by tracker-id", async () => {
      const octokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: {
                items: [
                  {
                    html_url: "https://github.com/owner/repo/issues/1",
                    number: 1,
                    repository: { full_name: "owner/repo" },
                    created_at: "2025-01-01T00:00:00Z",
                    updated_at: "2025-01-02T00:00:00Z",
                    state: "open",
                    title: "Test Issue",
                  },
                ],
              },
            }),
          },
        },
      };

      const result = await searchByTrackerId(octokit, "workflow-1", ["owner/repo"], [], 100, 10, null);

      expect(result.items).toHaveLength(1);
      expect(result.items[0].content_type).toBe("issue");
      expect(result.items[0].number).toBe(1);
      expect(result.itemsScanned).toBe(1);
      expect(result.pagesScanned).toBe(1);
    });

    it("should respect max items budget", async () => {
      const items = Array.from({ length: 10 }, (_, i) => ({
        html_url: `https://github.com/owner/repo/issues/${i + 1}`,
        number: i + 1,
        repository: { full_name: "owner/repo" },
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
        state: "open",
        title: `Issue ${i + 1}`,
      }));

      const octokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: { items },
            }),
          },
        },
      };

      const result = await searchByTrackerId(
        octokit,
        "workflow-1",
        ["owner/repo"],
        [],
        5, // max 5 items
        10,
        null
      );

      expect(result.items).toHaveLength(5);
      expect(result.itemsScanned).toBe(5);
    });

    it("should handle pagination", async () => {
      const page1Items = Array.from({ length: 100 }, (_, i) => ({
        html_url: `https://github.com/owner/repo/issues/${i + 1}`,
        number: i + 1,
        repository: { full_name: "owner/repo" },
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
        state: "open",
        title: `Issue ${i + 1}`,
      }));

      const page2Items = Array.from({ length: 50 }, (_, i) => ({
        html_url: `https://github.com/owner/repo/issues/${i + 101}`,
        number: i + 101,
        repository: { full_name: "owner/repo" },
        created_at: "2025-01-01T00:00:00Z",
        updated_at: "2025-01-02T00:00:00Z",
        state: "open",
        title: `Issue ${i + 101}`,
      }));

      const octokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi
              .fn()
              .mockResolvedValueOnce({ data: { items: page1Items } })
              .mockResolvedValueOnce({ data: { items: page2Items } }),
          },
        },
      };

      const result = await searchByTrackerId(octokit, "workflow-1", ["owner/repo"], [], 150, 10, null);

      expect(result.items).toHaveLength(150);
      expect(result.pagesScanned).toBe(2);
    });

    it("should build query with orgs when provided", async () => {
      const octokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: { items: [] },
            }),
          },
        },
      };

      await searchByTrackerId(octokit, "workflow-1", [], ["myorg"], 100, 10, null);

      const call = octokit.rest.search.issuesAndPullRequests.mock.calls[0][0];
      expect(call.q).toContain('"gh-aw-tracker-id: workflow-1"');
      expect(call.q).toContain("org:myorg");
      expect(call.q).not.toContain("(");
      expect(call.q).not.toContain(")");
    });

    it("should build query with both repos and orgs", async () => {
      const octokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: { items: [] },
            }),
          },
        },
      };

      await searchByTrackerId(octokit, "workflow-1", ["owner/repo1"], ["myorg"], 100, 10, null);

      const call = octokit.rest.search.issuesAndPullRequests.mock.calls[0][0];
      expect(call.q).toContain('"gh-aw-tracker-id: workflow-1"');
      expect(call.q).toContain("repo:owner/repo1");
      expect(call.q).toContain("org:myorg");
      expect(call.q).not.toContain("(");
      expect(call.q).not.toContain(")");
    });
  });

  describe("searchByLabel", () => {
    it("should search for items by label", async () => {
      const octokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: {
                items: [
                  {
                    html_url: "https://github.com/owner/repo/issues/1",
                    number: 1,
                    repository: { full_name: "owner/repo" },
                    created_at: "2025-01-01T00:00:00Z",
                    updated_at: "2025-01-02T00:00:00Z",
                    state: "open",
                    title: "Test Issue",
                  },
                ],
              },
            }),
          },
        },
      };

      const result = await searchByLabel(octokit, "z_campaign_test", ["owner/repo"], [], 100, 10, null);

      expect(result.items).toHaveLength(1);
      expect(result.items[0].content_type).toBe("issue");
    });

    it("should build repo-specific query when repos provided", async () => {
      const octokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: { items: [] },
            }),
          },
        },
      };

      await searchByLabel(octokit, "z_campaign_test", ["owner/repo1", "owner/repo2"], [], 100, 10, null);

      const call = octokit.rest.search.issuesAndPullRequests.mock.calls[0][0];
      expect(call.q).toContain('label:"z_campaign_test"');
      expect(call.q).toContain("repo:owner/repo1");
      expect(call.q).toContain("repo:owner/repo2");
      expect(call.q).not.toContain("(");
      expect(call.q).not.toContain(")");
    });

    it("should build org-specific query when orgs provided", async () => {
      const octokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: { items: [] },
            }),
          },
        },
      };

      await searchByLabel(octokit, "z_campaign_test", [], ["myorg", "anotherorg"], 100, 10, null);

      const call = octokit.rest.search.issuesAndPullRequests.mock.calls[0][0];
      expect(call.q).toContain('label:"z_campaign_test"');
      expect(call.q).toContain("org:myorg");
      expect(call.q).toContain("org:anotherorg");
      expect(call.q).not.toContain("(");
      expect(call.q).not.toContain(")");
    });

    it("should build combined query when both repos and orgs provided", async () => {
      const octokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: { items: [] },
            }),
          },
        },
      };

      await searchByLabel(octokit, "z_campaign_test", ["owner/repo1"], ["myorg"], 100, 10, null);

      const call = octokit.rest.search.issuesAndPullRequests.mock.calls[0][0];
      expect(call.q).toContain('label:"z_campaign_test"');
      expect(call.q).toContain("repo:owner/repo1");
      expect(call.q).toContain("org:myorg");
      expect(call.q).not.toContain("(");
      expect(call.q).not.toContain(")");
    });
  });

  describe("discover", () => {
    it("should discover items and generate manifest", async () => {
      const mockOctokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: {
                items: [
                  {
                    html_url: "https://github.com/owner/repo/issues/1",
                    number: 1,
                    repository: { full_name: "owner/repo" },
                    created_at: "2025-01-01T00:00:00Z",
                    updated_at: "2025-01-02T00:00:00Z",
                    state: "open",
                    title: "Test Issue",
                  },
                ],
              },
            }),
          },
        },
      };

      global.github = mockOctokit;

      const config = {
        campaignId: "test-campaign",
        workflows: ["workflow-1"],
        maxDiscoveryItems: 100,
        maxDiscoveryPages: 10,
      };

      const manifest = await discover(config);

      expect(manifest.schema_version).toBe("v1");
      expect(manifest.campaign_id).toBe("test-campaign");
      expect(manifest.discovery.total_items).toBe(1);
      expect(manifest.summary.needs_add_count).toBe(1);
      expect(manifest.items).toHaveLength(1);
    });

    it("should sort items deterministically", async () => {
      const mockOctokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: {
                items: [
                  {
                    html_url: "https://github.com/owner/repo/issues/3",
                    number: 3,
                    repository: { full_name: "owner/repo" },
                    created_at: "2025-01-01T00:00:00Z",
                    updated_at: "2025-01-03T00:00:00Z",
                    state: "open",
                    title: "Issue 3",
                  },
                  {
                    html_url: "https://github.com/owner/repo/issues/1",
                    number: 1,
                    repository: { full_name: "owner/repo" },
                    created_at: "2025-01-01T00:00:00Z",
                    updated_at: "2025-01-01T00:00:00Z",
                    state: "open",
                    title: "Issue 1",
                  },
                  {
                    html_url: "https://github.com/owner/repo/issues/2",
                    number: 2,
                    repository: { full_name: "owner/repo" },
                    created_at: "2025-01-01T00:00:00Z",
                    updated_at: "2025-01-02T00:00:00Z",
                    state: "open",
                    title: "Issue 2",
                  },
                ],
              },
            }),
          },
        },
      };

      global.github = mockOctokit;

      const config = {
        campaignId: "test-campaign",
        workflows: ["workflow-1"],
        maxDiscoveryItems: 100,
        maxDiscoveryPages: 10,
      };

      const manifest = await discover(config);

      expect(manifest.items).toHaveLength(3);
      expect(manifest.items[0].number).toBe(1);
      expect(manifest.items[1].number).toBe(2);
      expect(manifest.items[2].number).toBe(3);
    });

    it("should calculate summary counts correctly", async () => {
      const mockOctokit = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn().mockResolvedValue({
              data: {
                items: [
                  {
                    html_url: "https://github.com/owner/repo/issues/1",
                    number: 1,
                    repository: { full_name: "owner/repo" },
                    created_at: "2025-01-01T00:00:00Z",
                    updated_at: "2025-01-02T00:00:00Z",
                    state: "open",
                    title: "Open Issue",
                  },
                  {
                    html_url: "https://github.com/owner/repo/issues/2",
                    number: 2,
                    repository: { full_name: "owner/repo" },
                    created_at: "2025-01-01T00:00:00Z",
                    updated_at: "2025-01-03T00:00:00Z",
                    state: "closed",
                    title: "Closed Issue",
                    closed_at: "2025-01-03T00:00:00Z",
                  },
                  {
                    html_url: "https://github.com/owner/repo/pull/3",
                    number: 3,
                    repository: { full_name: "owner/repo" },
                    created_at: "2025-01-01T00:00:00Z",
                    updated_at: "2025-01-04T00:00:00Z",
                    state: "closed",
                    title: "Merged PR",
                    pull_request: {},
                    merged_at: "2025-01-04T00:00:00Z",
                  },
                ],
              },
            }),
          },
        },
      };

      global.github = mockOctokit;

      const config = {
        campaignId: "test-campaign",
        workflows: ["workflow-1"],
        maxDiscoveryItems: 100,
        maxDiscoveryPages: 10,
      };

      const manifest = await discover(config);

      expect(manifest.summary.open_count).toBe(1);
      expect(manifest.summary.closed_count).toBe(1);
      expect(manifest.summary.merged_count).toBe(1);
      expect(manifest.summary.needs_add_count).toBe(1); // open items
      expect(manifest.summary.needs_update_count).toBe(2); // closed + merged
    });
  });
});
