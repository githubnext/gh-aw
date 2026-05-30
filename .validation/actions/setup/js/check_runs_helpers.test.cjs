// @ts-check
import { describe, it, expect } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { isDeploymentCheck, selectLatestRelevantChecks, getFailingChecks } = req("./check_runs_helpers.cjs");

describe("check_runs_helpers", () => {
  describe("isDeploymentCheck", () => {
    it("returns true when app slug is github-deployments", () => {
      expect(isDeploymentCheck({ app: { slug: "github-deployments" } })).toBe(true);
    });

    it("returns false for non-deployment check runs", () => {
      expect(isDeploymentCheck({ app: { slug: "github-actions" } })).toBe(false);
    });

    it("returns false when app is absent", () => {
      expect(isDeploymentCheck({ name: "CI" })).toBe(false);
    });

    it("returns false for null/undefined", () => {
      expect(isDeploymentCheck(null)).toBe(false);
      expect(isDeploymentCheck(undefined)).toBe(false);
    });
  });

  describe("selectLatestRelevantChecks", () => {
    /** @type {any[]} */
    const runs = [
      { id: 1, name: "CI", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z", app: { slug: "github-actions" } },
      { id: 2, name: "CI", status: "completed", conclusion: "failure", started_at: "2024-01-02T00:00:00Z", app: { slug: "github-actions" } },
      { id: 3, name: "Lint", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z", app: { slug: "github-actions" } },
      { id: 4, name: "Deploy", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z", app: { slug: "github-deployments" } },
    ];

    it("returns the latest run per name", () => {
      const { relevant } = selectLatestRelevantChecks(runs);
      const ci = relevant.find(r => r.name === "CI");
      expect(ci?.id).toBe(2);
    });

    it("excludes deployment checks and reports count", () => {
      const { relevant, deploymentCheckCount } = selectLatestRelevantChecks(runs);
      expect(relevant.every(r => r.app?.slug !== "github-deployments")).toBe(true);
      expect(deploymentCheckCount).toBe(1);
    });

    it("excludes runs in excludedCheckRunIds", () => {
      const { relevant, currentRunFilterCount } = selectLatestRelevantChecks(runs, {
        excludedCheckRunIds: new Set([2]),
      });
      expect(relevant.find(r => r.id === 2)).toBeUndefined();
      expect(currentRunFilterCount).toBe(1);
    });

    it("filters by includeList", () => {
      const { relevant } = selectLatestRelevantChecks(runs, { includeList: ["Lint"] });
      expect(relevant.length).toBe(1);
      expect(relevant[0].name).toBe("Lint");
    });

    it("filters by excludeList", () => {
      const { relevant } = selectLatestRelevantChecks(runs, { excludeList: ["Lint"] });
      expect(relevant.find(r => r.name === "Lint")).toBeUndefined();
    });

    it("returns empty array for empty input", () => {
      const { relevant, deploymentCheckCount, currentRunFilterCount } = selectLatestRelevantChecks([]);
      expect(relevant).toEqual([]);
      expect(deploymentCheckCount).toBe(0);
      expect(currentRunFilterCount).toBe(0);
    });

    it("ignores empty includeList (includes everything)", () => {
      const { relevant } = selectLatestRelevantChecks(runs, { includeList: [] });
      expect(relevant.length).toBe(2); // CI (latest) + Lint
    });
  });

  describe("getFailingChecks", () => {
    /** @type {any[]} */
    const runs = [
      { id: 1, status: "completed", conclusion: "success" },
      { id: 2, status: "completed", conclusion: "failure" },
      { id: 3, status: "completed", conclusion: "cancelled" },
      { id: 4, status: "completed", conclusion: "timed_out" },
      { id: 5, status: "in_progress", conclusion: null },
      { id: 6, status: "completed", conclusion: null },
    ];

    it("includes failure, cancelled, and timed_out conclusions", () => {
      const failing = getFailingChecks(runs);
      const ids = failing.map(r => r.id);
      expect(ids).toContain(2);
      expect(ids).toContain(3);
      expect(ids).toContain(4);
    });

    it("excludes success conclusion", () => {
      const failing = getFailingChecks(runs);
      expect(failing.find(r => r.id === 1)).toBeUndefined();
    });

    it("includes pending runs when allowPending is false (default)", () => {
      const failing = getFailingChecks(runs);
      expect(failing.find(r => r.id === 5)).toBeDefined();
    });

    it("excludes pending runs when allowPending is true", () => {
      const failing = getFailingChecks(runs, { allowPending: true });
      expect(failing.find(r => r.id === 5)).toBeUndefined();
    });

    it("excludes completed run with null conclusion", () => {
      const failing = getFailingChecks(runs);
      expect(failing.find(r => r.id === 6)).toBeUndefined();
    });

    it("returns empty array when no checks fail", () => {
      const passing = [{ id: 1, status: "completed", conclusion: "success" }];
      expect(getFailingChecks(passing)).toEqual([]);
    });
  });
});
