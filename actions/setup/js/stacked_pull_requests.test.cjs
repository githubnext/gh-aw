// @ts-check
import { describe, it, expect, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);

// Set up globals required by modules that reference `core` at load time.
global.core = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

const {
  isStackedEnabled,
  parseStackMetadata,
  hasCircularStackDependency,
  buildStackMetadataLines,
  stackedDisabledError,
  missingStackBaseError,
  circularStackError,
  verifyStackBaseBranchExists,
  createStackTracker,
} = require("./stacked_pull_requests.cjs");

describe("stacked_pull_requests - configuration", () => {
  it("is enabled by default", () => {
    expect(isStackedEnabled({})).toBe(true);
    expect(isStackedEnabled({ stacked: true })).toBe(true);
  });

  it("is disabled with stacked: false", () => {
    expect(isStackedEnabled({ stacked: false })).toBe(false);
  });
});

describe("stacked_pull_requests - stack metadata", () => {
  it("parses and normalizes stack metadata fields", () => {
    const result = parseStackMetadata({
      stack_position: "2",
      stack_root: "  feature/base  ",
      dependencies: ["feature/one", "feature/one", "feature/two", 42],
    });
    expect(result).toEqual({ position: 2, root: "feature/base", dependencies: ["feature/one", "feature/two"] });
  });

  it("returns empty metadata when no stack fields are present", () => {
    expect(parseStackMetadata({})).toEqual({ position: null, root: null, dependencies: [] });
  });

  it("ignores invalid stack positions", () => {
    expect(parseStackMetadata({ stack_position: 0 }).position).toBeNull();
    expect(parseStackMetadata({ stack_position: -3 }).position).toBeNull();
    expect(parseStackMetadata({ stack_position: "not-a-number" }).position).toBeNull();
  });

  it("normalizes branch-like metadata so it cannot break out of the HTML comment", () => {
    const result = parseStackMetadata({ stack_root: "evil --> <script>", dependencies: ["dep --> <img>"] });
    expect(result.root).not.toContain("-->");
    expect(result.dependencies[0]).not.toContain("-->");
  });

  it("detects a direct circular dependency", () => {
    expect(hasCircularStackDependency(["feature-1"], "feature-1", new Map())).toBe(true);
  });

  it("detects a transitive circular dependency", () => {
    const parents = new Map([
      ["feature-2", "feature-1"],
      ["feature-1", "feature-3"],
    ]);
    expect(hasCircularStackDependency(["feature-3"], "feature-2", parents)).toBe(true);
  });

  it("does not report a cycle for a valid stack", () => {
    const parents = new Map([["feature-1", "main"]]);
    expect(hasCircularStackDependency(["feature-2"], "feature-1", parents)).toBe(false);
  });

  it("builds stack metadata lines with dependency references", () => {
    const lines = buildStackMetadataLines({
      base: "feature-1",
      position: 2,
      root: "main",
      dependencies: ["feature-1"],
      dependsOnPullRequests: [42],
    });
    expect(lines[0]).toBe("Depends on #42");
    expect(lines[1]).toContain("<!-- gh-aw-stack:");
    expect(JSON.parse(lines[1].replace("<!-- gh-aw-stack: ", "").replace(" -->", ""))).toEqual({
      base: "feature-1",
      position: 2,
      root: "main",
      dependencies: ["feature-1"],
      depends_on: [42],
    });
  });

  it("returns no lines when there is no stack information", () => {
    expect(buildStackMetadataLines({ base: null, position: null, root: null, dependencies: [], dependsOnPullRequests: [] })).toEqual([]);
  });

  it("records metadata even when the pull request is not stacked", () => {
    const lines = buildStackMetadataLines({ base: null, position: 1, root: "main", dependencies: [], dependsOnPullRequests: [] });
    expect(lines).toHaveLength(1);
    expect(lines[0]).toContain('"position":1');
  });
});

describe("stacked_pull_requests - stack tracker", () => {
  it("records a pull request under both the effective and agent branch names", () => {
    const tracker = createStackTracker();
    tracker.record({ branch: "salted-feature-1", number: 7, url: "https://example.com/7", repo: "owner/repo" }, { agentBranch: "feature-1", baseBranch: "main" });

    expect(tracker.get("feature-1", "owner/repo")?.number).toBe(7);
    expect(tracker.get("salted-feature-1", "owner/repo")?.number).toBe(7);
    expect(tracker.parents.get("salted-feature-1")).toBe("main");
  });

  it("scopes lookups to the target repository", () => {
    const tracker = createStackTracker();
    tracker.record({ branch: "feature-1", number: 7, url: "https://example.com/7", repo: "owner/repo" }, { agentBranch: null, baseBranch: "main" });

    expect(tracker.get("feature-1", "other/repo")).toBeNull();
  });

  it("resolves dependency branches to pull request numbers without duplicates", () => {
    const tracker = createStackTracker();
    tracker.record({ branch: "salted-feature-1", number: 7, url: "https://example.com/7", repo: "owner/repo" }, { agentBranch: "feature-1", baseBranch: "main" });
    tracker.record({ branch: "feature-2", number: 8, url: "https://example.com/8", repo: "owner/repo" }, { agentBranch: null, baseBranch: "salted-feature-1" });

    expect(tracker.resolveDependencies(["feature-1", "salted-feature-1", "feature-2", "unknown"], "owner/repo")).toEqual([7, 8]);
  });
});

describe("stacked_pull_requests - base branch verification", () => {
  const repoParts = { owner: "owner", repo: "repo" };

  it("succeeds when the base branch exists", async () => {
    const getBranch = vi.fn().mockResolvedValue({ data: {} });
    const client = { rest: { repos: { getBranch } } };
    await expect(verifyStackBaseBranchExists(client, repoParts, "feature-1", "owner/repo", "main")).resolves.toEqual({ success: true });
    expect(getBranch).toHaveBeenCalledWith({ owner: "owner", repo: "repo", branch: "feature-1" });
  });

  it("fails with dependency-order guidance when the base branch is missing", async () => {
    const client = { rest: { repos: { getBranch: vi.fn().mockRejectedValue(Object.assign(new Error("Not Found"), { status: 404 })) } } };
    const result = await verifyStackBaseBranchExists(client, repoParts, "feature-1", "owner/repo", "main");
    expect(result.success).toBe(false);
    expect(result.error).toBe(missingStackBaseError("feature-1", "owner/repo", "main"));
  });

  it("degrades to a warning for other API errors", async () => {
    const client = { rest: { repos: { getBranch: vi.fn().mockRejectedValue(Object.assign(new Error("boom"), { status: 500 })) } } };
    await expect(verifyStackBaseBranchExists(client, repoParts, "feature-1", "owner/repo", "main")).resolves.toEqual({ success: true });
  });

  it("degrades to a warning when the getBranch API is unavailable", async () => {
    await expect(verifyStackBaseBranchExists({ rest: { repos: {} } }, repoParts, "feature-1", "owner/repo", "main")).resolves.toEqual({ success: true });
  });
});

describe("stacked_pull_requests - error messages", () => {
  it("names the default base branch and the stacked option when stacking is disabled", () => {
    const message = stackedDisabledError("feature-1", "main");
    expect(message).toContain("feature-1");
    expect(message).toContain("'main'");
    expect(message).toContain("stacked: false");
  });

  it("names both branches for a circular dependency", () => {
    const message = circularStackError("feature-1", "feature-2");
    expect(message).toContain("feature-1");
    expect(message).toContain("feature-2");
  });
});
