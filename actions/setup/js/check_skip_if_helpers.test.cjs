// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

/** @type {ReturnType<typeof vi.fn>} */
let infoFn;

/** @type {typeof global.core} */
let mockCore;

/** @type {typeof global.context} */
let mockContext;

beforeEach(() => {
  infoFn = vi.fn();
  mockCore = {
    info: infoFn,
    warning: vi.fn(),
    error: vi.fn(),
    setFailed: vi.fn(),
    setOutput: vi.fn(),
    debug: vi.fn(),
    summary: {
      addRaw: vi.fn().mockReturnThis(),
      write: vi.fn().mockResolvedValue(undefined),
    },
  };
  mockContext = {
    repo: {
      owner: "testowner",
      repo: "testrepo",
    },
    eventName: "issues",
    actor: "testuser",
    runId: 1,
    workflow: "test-workflow",
  };
  global.core = mockCore;
  global.context = mockContext;
});

describe("check_skip_if_helpers.cjs - buildSearchQuery", () => {
  /** @returns {Promise<{buildSearchQuery: (query: string, scope: string|undefined) => string}>} */
  const loadModule = () => import("./check_skip_if_helpers.cjs");

  describe("scope: none", () => {
    it("returns the raw query unchanged when skipScope is 'none'", async () => {
      const { buildSearchQuery } = await loadModule();
      const result = buildSearchQuery("is:issue is:open", "none");
      expect(result).toBe("is:issue is:open");
    });

    it("logs 'Using raw query' message when skipScope is 'none'", async () => {
      const { buildSearchQuery } = await loadModule();
      buildSearchQuery("is:issue", "none");
      expect(infoFn).toHaveBeenCalledWith("Using raw query (scope: none): is:issue");
    });

    it("does not append repo to the query when skipScope is 'none'", async () => {
      const { buildSearchQuery } = await loadModule();
      const result = buildSearchQuery("label:bug is:open", "none");
      expect(result).not.toContain("repo:");
    });

    it("handles complex multi-word queries with scope: none", async () => {
      const { buildSearchQuery } = await loadModule();
      const query = "is:issue is:open label:bug assignee:testuser";
      const result = buildSearchQuery(query, "none");
      expect(result).toBe(query);
    });
  });

  describe("check_skip_if_helpers.cjs - runSkipQueryGate", () => {
    /** @returns {Promise<{runSkipQueryGate: (options: any) => Promise<void>}>} */
    const loadModule = () => import("./check_skip_if_helpers.cjs");

    beforeEach(() => {
      global.github = {
        rest: {
          search: {
            issuesAndPullRequests: vi.fn(),
          },
        },
      };
    });

    it("uses thresholdEnvVar in validation errors", async () => {
      const { runSkipQueryGate } = await loadModule();

      await runSkipQueryGate({
        skipQuery: "is:issue is:open",
        workflowName: "test-workflow",
        thresholdStr: "invalid",
        thresholdEnvVar: "GH_AW_SKIP_MAX_MATCHES",
        thresholdLabel: "Maximum matches threshold",
        checkLabel: "skip-if-match",
        outputName: "skip_check_ok",
        skipScope: undefined,
        shouldSkip: () => false,
        warningMessage: () => "",
        successMessage: () => "",
        denialSummaryMessage: () => "",
        denialSummaryNextStep: "",
      });

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_SKIP_MAX_MATCHES must be a positive integer"));
      expect(global.github.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
    });

    it("applies policy decision and output name when skip condition matches", async () => {
      global.github.rest.search.issuesAndPullRequests.mockResolvedValue({ data: { total_count: 2 } });
      const { runSkipQueryGate } = await loadModule();

      await runSkipQueryGate({
        skipQuery: "is:issue is:open",
        workflowName: "test-workflow",
        thresholdStr: "1",
        thresholdEnvVar: "GH_AW_SKIP_MAX_MATCHES",
        thresholdLabel: "Maximum matches threshold",
        checkLabel: "skip-if-match",
        outputName: "skip_check_ok",
        skipScope: undefined,
        shouldSkip: (totalCount, threshold) => totalCount >= threshold,
        warningMessage: () => "skip warning",
        successMessage: () => "success message",
        denialSummaryMessage: () => "summary message",
        denialSummaryNextStep: "summary next step",
      });

      expect(mockCore.warning).toHaveBeenCalledWith("skip warning");
      expect(mockCore.setOutput).toHaveBeenCalledWith("skip_check_ok", "false");
    });
  });

  describe("scope: default (repo-scoped)", () => {
    it("appends repo:owner/repo when skipScope is undefined", async () => {
      const { buildSearchQuery } = await loadModule();
      const result = buildSearchQuery("is:issue", undefined);
      expect(result).toBe("is:issue repo:testowner/testrepo");
    });

    it("appends repo:owner/repo when skipScope is empty string", async () => {
      const { buildSearchQuery } = await loadModule();
      const result = buildSearchQuery("is:issue", "");
      expect(result).toBe("is:issue repo:testowner/testrepo");
    });

    it("appends repo:owner/repo when skipScope is any non-'none' value", async () => {
      const { buildSearchQuery } = await loadModule();
      const result = buildSearchQuery("is:issue", "repo");
      expect(result).toBe("is:issue repo:testowner/testrepo");
    });

    it("logs 'Scoped query' message when applying repo scope", async () => {
      const { buildSearchQuery } = await loadModule();
      buildSearchQuery("is:pr is:open", undefined);
      expect(infoFn).toHaveBeenCalledWith("Scoped query: is:pr is:open repo:testowner/testrepo");
    });

    it("uses owner and repo from context.repo", async () => {
      mockContext.repo = { owner: "myorg", repo: "myrepo" };
      global.context = mockContext;
      const { buildSearchQuery } = await loadModule();
      const result = buildSearchQuery("is:issue", undefined);
      expect(result).toBe("is:issue repo:myorg/myrepo");
    });

    it("handles multi-word queries with repo scoping", async () => {
      const { buildSearchQuery } = await loadModule();
      const result = buildSearchQuery("is:issue is:open label:enhancement author:user", undefined);
      expect(result).toBe("is:issue is:open label:enhancement author:user repo:testowner/testrepo");
    });
  });
});
