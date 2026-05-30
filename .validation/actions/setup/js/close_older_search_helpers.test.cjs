// @ts-check

import { describe, it, expect, beforeEach, vi } from "vitest";
import { buildMarkerSearchQuery, filterByMarker, logFilterSummary } from "./close_older_search_helpers.cjs";

// Mock globals
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

describe("close_older_search_helpers", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("buildMarkerSearchQuery", () => {
    it("should build query with close-older-key when provided", () => {
      const { searchQuery, exactMarker } = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        closeOlderKey: "my-stable-key",
      });

      expect(searchQuery).toBe('repo:owner/repo is:open "gh-aw-close-key: my-stable-key" in:body');
      expect(exactMarker).toBe("<!-- gh-aw-close-key: my-stable-key -->");
    });

    it("should build query with workflow-id when no close-older-key", () => {
      const { searchQuery, exactMarker } = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
      });

      expect(searchQuery).toBe('repo:owner/repo is:open "gh-aw-workflow-id: test-workflow" in:body');
      expect(exactMarker).toBe("<!-- gh-aw-workflow-id: test-workflow -->");
    });

    it("should use callerWorkflowId for exact marker when provided", () => {
      const { searchQuery, exactMarker } = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "my-reusable-workflow",
        callerWorkflowId: "owner/repo/CallerA",
      });

      expect(searchQuery).toBe('repo:owner/repo is:open "gh-aw-workflow-id: my-reusable-workflow" in:body');
      expect(exactMarker).toBe("<!-- gh-aw-workflow-call-id: owner/repo/CallerA -->");
    });

    it("should append entityQualifier when provided", () => {
      const { searchQuery } = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        entityQualifier: "is:issue",
      });

      expect(searchQuery).toBe('repo:owner/repo is:issue is:open "gh-aw-workflow-id: test-workflow" in:body');
    });

    it("should append entityQualifier with close-older-key", () => {
      const { searchQuery } = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        closeOlderKey: "my-key",
        entityQualifier: "is:issue",
      });

      expect(searchQuery).toBe('repo:owner/repo is:issue is:open "gh-aw-close-key: my-key" in:body');
    });

    it("should prefer close-older-key over callerWorkflowId when both are provided", () => {
      const { searchQuery, exactMarker } = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "my-workflow",
        callerWorkflowId: "owner/repo/CallerA",
        closeOlderKey: "shared-key",
      });

      expect(searchQuery).toContain("gh-aw-close-key: shared-key");
      expect(exactMarker).toBe("<!-- gh-aw-close-key: shared-key -->");
    });

    it("should escape quotes in workflow ID to prevent query injection", () => {
      const { searchQuery } = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: 'workflow"with"quotes',
      });

      expect(searchQuery).toContain('workflow\\"with\\"quotes');
    });

    it("should escape quotes in close-older-key to prevent query injection", () => {
      const { searchQuery } = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        closeOlderKey: 'key"with"quotes',
      });

      expect(searchQuery).toContain('key\\"with\\"quotes');
    });
  });

  describe("filterByMarker", () => {
    it("should exclude items matching excludeNumber", () => {
      const items = [
        { number: 1, body: "<!-- gh-aw-workflow-id: test -->", title: "Item 1" },
        { number: 2, body: "<!-- gh-aw-workflow-id: test -->", title: "Item 2" },
      ];

      const { filtered, counters } = filterByMarker({
        items,
        excludeNumber: 1,
        exactMarker: "<!-- gh-aw-workflow-id: test -->",
        entityType: "issue",
      });

      expect(filtered).toHaveLength(1);
      expect(filtered[0].number).toBe(2);
      expect(counters.excludedCount).toBe(1);
    });

    it("should exclude items without exact marker in body", () => {
      const items = [
        { number: 1, body: "<!-- gh-aw-workflow-id: test -->", title: "Match" },
        { number: 2, body: "<!-- gh-aw-workflow-id: test-extended -->", title: "No match" },
        { number: 3, body: "No marker at all", title: "No marker" },
      ];

      const { filtered, counters } = filterByMarker({
        items,
        excludeNumber: 999,
        exactMarker: "<!-- gh-aw-workflow-id: test -->",
        entityType: "issue",
      });

      expect(filtered).toHaveLength(1);
      expect(filtered[0].number).toBe(1);
      expect(counters.markerMismatchCount).toBe(2);
    });

    it("should skip null/undefined items", () => {
      const items = [null, undefined, { number: 1, body: "<!-- gh-aw-workflow-id: test -->", title: "Valid" }];

      const { filtered } = filterByMarker({
        items,
        excludeNumber: 999,
        exactMarker: "<!-- gh-aw-workflow-id: test -->",
        entityType: "discussion",
      });

      expect(filtered).toHaveLength(1);
      expect(filtered[0].number).toBe(1);
    });

    it("should apply additionalFilter before standard checks", () => {
      const items = [
        { number: 1, body: "<!-- gh-aw-workflow-id: test -->", title: "Issue", pull_request: undefined },
        { number: 2, body: "<!-- gh-aw-workflow-id: test -->", title: "PR", pull_request: {} },
      ];

      const { filtered, counters } = filterByMarker({
        items,
        excludeNumber: 999,
        exactMarker: "<!-- gh-aw-workflow-id: test -->",
        entityType: "issue",
        additionalFilter: (item, extra) => {
          if (item.pull_request) {
            extra.pullRequestCount = (extra.pullRequestCount || 0) + 1;
            return false;
          }
          return true;
        },
      });

      expect(filtered).toHaveLength(1);
      expect(filtered[0].number).toBe(1);
      expect(counters.pullRequestCount).toBe(1);
    });

    it("should handle items with missing body gracefully", () => {
      const items = [{ number: 1, title: "No body" }];

      const { filtered, counters } = filterByMarker({
        items,
        excludeNumber: 999,
        exactMarker: "<!-- gh-aw-workflow-id: test -->",
        entityType: "issue",
      });

      expect(filtered).toHaveLength(0);
      expect(counters.markerMismatchCount).toBe(1);
    });

    it("should return all matching items when no exclusions apply", () => {
      const items = [
        { number: 1, body: "Content <!-- gh-aw-workflow-id: test --> more", title: "Item 1" },
        { number: 2, body: "Other <!-- gh-aw-workflow-id: test --> text", title: "Item 2" },
      ];

      const { filtered, counters } = filterByMarker({
        items,
        excludeNumber: 999,
        exactMarker: "<!-- gh-aw-workflow-id: test -->",
        entityType: "issue",
      });

      expect(filtered).toHaveLength(2);
      expect(counters.filteredCount).toBe(2);
      expect(counters.excludedCount).toBe(0);
      expect(counters.markerMismatchCount).toBe(0);
    });

    it("should work with discussion-specific additional filters", () => {
      const items = [
        { number: 1, body: "<!-- gh-aw-workflow-id: test -->", title: "Open", closed: false, category: { id: "CAT1" } },
        { number: 2, body: "<!-- gh-aw-workflow-id: test -->", title: "Closed", closed: true, category: { id: "CAT1" } },
        { number: 3, body: "<!-- gh-aw-workflow-id: test -->", title: "Wrong cat", closed: false, category: { id: "CAT2" } },
      ];

      const categoryId = "CAT1";
      const { filtered, counters } = filterByMarker({
        items,
        excludeNumber: 999,
        exactMarker: "<!-- gh-aw-workflow-id: test -->",
        entityType: "discussion",
        additionalFilter: (d, extra) => {
          if (d.closed) {
            extra.closedCount = (extra.closedCount || 0) + 1;
            return false;
          }
          if (categoryId && (!d.category || d.category.id !== categoryId)) {
            return false;
          }
          return true;
        },
      });

      expect(filtered).toHaveLength(1);
      expect(filtered[0].number).toBe(1);
      expect(counters.closedCount).toBe(1);
    });
  });

  describe("logFilterSummary", () => {
    it("should log basic summary without extra labels", () => {
      logFilterSummary({
        entityTypePlural: "issues",
        counters: { filteredCount: 5, excludedCount: 1, markerMismatchCount: 2 },
      });

      expect(global.core.info).toHaveBeenCalledWith("Filtering complete:");
      expect(global.core.info).toHaveBeenCalledWith("  - Matched issues: 5");
      expect(global.core.info).toHaveBeenCalledWith("  - Excluded new issue: 1");
      expect(global.core.info).toHaveBeenCalledWith("  - Excluded marker mismatch: 2");
    });

    it("should log extra labels when provided", () => {
      logFilterSummary({
        entityTypePlural: "issues",
        counters: { filteredCount: 3, excludedCount: 0, markerMismatchCount: 1, pullRequestCount: 2 },
        extraLabels: [["pullRequestCount", "Excluded pull requests"]],
      });

      expect(global.core.info).toHaveBeenCalledWith("  - Excluded pull requests: 2");
    });

    it("should log zero for missing extra counter keys", () => {
      logFilterSummary({
        entityTypePlural: "discussions",
        counters: { filteredCount: 3, excludedCount: 0, markerMismatchCount: 0 },
        extraLabels: [["closedCount", "Excluded closed discussions"]],
      });

      expect(global.core.info).toHaveBeenCalledWith("  - Excluded closed discussions: 0");
    });
  });

  describe("parity: issues and discussions produce equivalent queries", () => {
    it("should produce equivalent queries for close-older-key (issue vs discussion)", () => {
      const issueResult = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        closeOlderKey: "my-key",
        entityQualifier: "is:issue",
      });

      const discussionResult = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        closeOlderKey: "my-key",
      });

      // Both should use the same exact marker
      expect(issueResult.exactMarker).toBe(discussionResult.exactMarker);
      // Queries should differ only in the entity qualifier
      expect(issueResult.searchQuery).toContain("is:issue");
      expect(discussionResult.searchQuery).not.toContain("is:issue");
      // Both should contain the close-key marker
      expect(issueResult.searchQuery).toContain("gh-aw-close-key: my-key");
      expect(discussionResult.searchQuery).toContain("gh-aw-close-key: my-key");
    });

    it("should produce equivalent queries for workflow-id (issue vs discussion)", () => {
      const issueResult = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
        entityQualifier: "is:issue",
      });

      const discussionResult = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "test-workflow",
      });

      expect(issueResult.exactMarker).toBe(discussionResult.exactMarker);
      expect(issueResult.searchQuery).toContain("is:issue");
      expect(discussionResult.searchQuery).not.toContain("is:issue");
    });

    it("should produce equivalent exact markers for callerWorkflowId (issue vs discussion)", () => {
      const issueResult = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "my-reusable",
        callerWorkflowId: "org/repo/Caller",
        entityQualifier: "is:issue",
      });

      const discussionResult = buildMarkerSearchQuery({
        owner: "owner",
        repo: "repo",
        workflowId: "my-reusable",
        callerWorkflowId: "org/repo/Caller",
      });

      expect(issueResult.exactMarker).toBe(discussionResult.exactMarker);
      expect(issueResult.exactMarker).toBe("<!-- gh-aw-workflow-call-id: org/repo/Caller -->");
    });

    it("should apply filterByMarker identically for issues and discussions", () => {
      const sharedItems = [
        { number: 1, body: "<!-- gh-aw-workflow-id: test -->", title: "A" },
        { number: 2, body: "<!-- gh-aw-workflow-id: test-extended -->", title: "B" },
        { number: 3, body: "<!-- gh-aw-workflow-id: test -->", title: "C" },
      ];

      const issueResult = filterByMarker({
        items: sharedItems,
        excludeNumber: 3,
        exactMarker: "<!-- gh-aw-workflow-id: test -->",
        entityType: "issue",
      });

      const discResult = filterByMarker({
        items: sharedItems,
        excludeNumber: 3,
        exactMarker: "<!-- gh-aw-workflow-id: test -->",
        entityType: "discussion",
      });

      expect(issueResult.filtered.map(i => i.number)).toEqual(discResult.filtered.map(i => i.number));
      expect(issueResult.counters.filteredCount).toBe(discResult.counters.filteredCount);
      expect(issueResult.counters.excludedCount).toBe(discResult.counters.excludedCount);
      expect(issueResult.counters.markerMismatchCount).toBe(discResult.counters.markerMismatchCount);
    });
  });
});
