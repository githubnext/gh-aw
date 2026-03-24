import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("check_skip_if_check_failing.cjs", () => {
  let mockCore;
  let mockGithub;
  let mockContext;

  beforeEach(() => {
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
    };

    mockGithub = {
      rest: {
        checks: {
          listForRef: vi.fn(),
        },
        actions: {
          listJobsForWorkflowRun: vi.fn(),
        },
      },
      paginate: vi.fn(),
    };

    mockContext = {
      repo: { owner: "test-owner", repo: "test-repo" },
      ref: "refs/heads/main",
      payload: {},
    };

    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;

    vi.resetModules();
  });

  afterEach(() => {
    vi.clearAllMocks();
    delete global.core;
    delete global.github;
    delete global.context;
    delete process.env.GH_AW_SKIP_BRANCH;
    delete process.env.GH_AW_SKIP_CHECK_INCLUDE;
    delete process.env.GH_AW_SKIP_CHECK_EXCLUDE;
    delete process.env.GITHUB_BASE_REF;
    delete process.env.GH_AW_SKIP_CHECK_ALLOW_PENDING;
    delete process.env.GITHUB_RUN_ID;
  });

  it("should allow workflow when all checks pass", async () => {
    mockGithub.paginate.mockResolvedValue([
      { name: "build", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z" },
      { name: "test", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z" },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should cancel workflow when a check has failed", async () => {
    mockGithub.paginate.mockResolvedValue([
      { name: "build", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
      { name: "test", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z" },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failing CI checks detected"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("build"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should cancel workflow when a check was cancelled", async () => {
    mockGithub.paginate.mockResolvedValue([{ name: "ci", status: "completed", conclusion: "cancelled", started_at: "2024-01-01T00:00:00Z" }]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
  });

  it("should cancel workflow when a check timed out", async () => {
    mockGithub.paginate.mockResolvedValue([{ name: "ci", status: "completed", conclusion: "timed_out", started_at: "2024-01-01T00:00:00Z" }]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
  });

  it("should cancel workflow when checks are still in progress (pending treated as failing by default)", async () => {
    mockGithub.paginate.mockResolvedValue([{ name: "build", status: "in_progress", conclusion: null, started_at: "2024-01-01T00:00:00Z" }]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("build (in_progress)"));
  });

  it("should cancel workflow when checks are queued (pending treated as failing by default)", async () => {
    mockGithub.paginate.mockResolvedValue([{ name: "test", status: "queued", conclusion: null, started_at: null }]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("test (queued)"));
  });

  it("should allow workflow when checks are in progress and allow-pending is true", async () => {
    process.env.GH_AW_SKIP_CHECK_ALLOW_PENDING = "true";
    mockGithub.paginate.mockResolvedValue([{ name: "build", status: "in_progress", conclusion: null, started_at: "2024-01-01T00:00:00Z" }]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should cancel when a completed check fails even with allow-pending true", async () => {
    process.env.GH_AW_SKIP_CHECK_ALLOW_PENDING = "true";
    mockGithub.paginate.mockResolvedValue([
      { name: "build", status: "in_progress", conclusion: null, started_at: "2024-01-01T00:00:00Z" },
      { name: "lint", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // lint failed → cancel; build pending but ignored due to allow-pending
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("lint"));
  });

  it("should allow workflow when no checks exist", async () => {
    mockGithub.paginate.mockResolvedValue([]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
  });

  it("should use the PR base branch when triggered by pull_request event", async () => {
    mockContext.payload = {
      pull_request: {
        base: { ref: "main" },
      },
    };
    mockContext.ref = "refs/pull/42/merge";

    mockGithub.paginate.mockResolvedValue([]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockGithub.paginate).toHaveBeenCalledWith(mockGithub.rest.checks.listForRef, expect.objectContaining({ ref: "main" }));
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
  });

  it("should use GH_AW_SKIP_BRANCH when explicitly configured", async () => {
    process.env.GH_AW_SKIP_BRANCH = "release/v2";
    mockGithub.paginate.mockResolvedValue([]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockGithub.paginate).toHaveBeenCalledWith(mockGithub.rest.checks.listForRef, expect.objectContaining({ ref: "release/v2" }));
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
  });

  it("should use GITHUB_BASE_REF when set (standard pull_request event env)", async () => {
    process.env.GITHUB_BASE_REF = "develop";
    mockContext.payload = {};

    mockGithub.paginate.mockResolvedValue([]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockGithub.paginate).toHaveBeenCalledWith(mockGithub.rest.checks.listForRef, expect.objectContaining({ ref: "develop" }));
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
  });

  it("should only check included checks when GH_AW_SKIP_CHECK_INCLUDE is set", async () => {
    process.env.GH_AW_SKIP_CHECK_INCLUDE = JSON.stringify(["build", "test"]);
    mockGithub.paginate.mockResolvedValue([
      { name: "build", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
      { name: "lint", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // build is in include list and failed → should cancel
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("build"));
    // lint is not in include list → should NOT appear in warning
    const warningCalls = mockCore.warning.mock.calls.flat().join(" ");
    expect(warningCalls).not.toContain("lint");
  });

  it("should allow workflow when failing check is not in include list", async () => {
    process.env.GH_AW_SKIP_CHECK_INCLUDE = JSON.stringify(["build"]);
    mockGithub.paginate.mockResolvedValue([
      { name: "lint", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
      { name: "build", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z" },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // lint is not in include list, build passed → allow
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
  });

  it("should ignore excluded checks when GH_AW_SKIP_CHECK_EXCLUDE is set", async () => {
    process.env.GH_AW_SKIP_CHECK_EXCLUDE = JSON.stringify(["lint"]);
    mockGithub.paginate.mockResolvedValue([
      { name: "lint", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
      { name: "build", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z" },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // lint is excluded, build passed → allow
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
  });

  it("should cancel when non-excluded check fails", async () => {
    process.env.GH_AW_SKIP_CHECK_EXCLUDE = JSON.stringify(["lint"]);
    mockGithub.paginate.mockResolvedValue([
      { name: "lint", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
      { name: "build", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // build not excluded and failed → cancel
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
  });

  it("should use the latest run for each check name", async () => {
    // Two runs for the same check name, the newer one passes
    mockGithub.paginate.mockResolvedValue([
      { name: "build", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
      { name: "build", status: "completed", conclusion: "success", started_at: "2024-01-02T00:00:00Z" },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // Latest run passed → allow
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
  });

  it("should fail with error message when API call fails", async () => {
    mockGithub.paginate.mockRejectedValue(new Error("Rate limit exceeded"));

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to fetch check runs"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Rate limit exceeded"));
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });

  it("should ignore deployment gate checks from github-deployments app", async () => {
    mockGithub.paginate.mockResolvedValue([
      // Regular CI check that passes
      { name: "build", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z", app: { slug: "github-actions" } },
      // Deployment gate (waiting for approval) — should be ignored even if it shows as failing
      { name: "production", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z", app: { slug: "github-deployments" } },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // Deployment gate is ignored, build passed → allow
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping 1 deployment gate check(s)"));
  });

  it("should allow workflow when only deployment checks are failing", async () => {
    mockGithub.paginate.mockResolvedValue([
      { name: "staging", status: "completed", conclusion: "cancelled", started_at: "2024-01-01T00:00:00Z", app: { slug: "github-deployments" } },
      { name: "production", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z", app: { slug: "github-deployments" } },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // All checks are deployment gates → no CI checks to evaluate → allow
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping 2 deployment gate check(s)"));
  });

  it("should still cancel when a non-deployment check fails alongside a deployment gate", async () => {
    mockGithub.paginate.mockResolvedValue([
      { name: "build", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z", app: { slug: "github-actions" } },
      { name: "production", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z", app: { slug: "github-deployments" } },
    ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // build failed (not a deployment gate) → cancel
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("build"));
  });

  it("should filter out check runs from the current workflow run", async () => {
    process.env.GITHUB_RUN_ID = "99999";

    // paginate is called twice: first for listForRef (check runs), then for listJobsForWorkflowRun (current run jobs)
    mockGithub.paginate
      .mockResolvedValueOnce([
        // current run's own pre_activation job (in_progress) — should be filtered
        { id: 1001, name: "pre_activation", status: "in_progress", conclusion: null, started_at: "2024-01-01T00:00:00Z" },
        // regular CI check that passed
        { id: 2001, name: "build", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z" },
      ])
      .mockResolvedValueOnce([
        // jobs of the current workflow run
        { id: 1001 },
      ]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // pre_activation from current run is filtered out; build passed → allow
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping 1 check run(s) from the current workflow run"));
  });

  it("should allow workflow when all in-progress checks belong to the current run", async () => {
    process.env.GITHUB_RUN_ID = "99999";

    mockGithub.paginate
      .mockResolvedValueOnce([
        { id: 1001, name: "pre_activation", status: "in_progress", conclusion: null, started_at: "2024-01-01T00:00:00Z" },
        { id: 1002, name: "agent", status: "in_progress", conclusion: null, started_at: "2024-01-01T00:00:00Z" },
      ])
      .mockResolvedValueOnce([{ id: 1001 }, { id: 1002 }]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // all checks belong to current run → filtered out → no checks to evaluate → allow
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping 2 check run(s) from the current workflow run"));
  });

  it("should still cancel when a non-current-run check fails", async () => {
    process.env.GITHUB_RUN_ID = "99999";

    mockGithub.paginate
      .mockResolvedValueOnce([
        { id: 1001, name: "pre_activation", status: "in_progress", conclusion: null, started_at: "2024-01-01T00:00:00Z" },
        { id: 2001, name: "build", status: "completed", conclusion: "failure", started_at: "2024-01-01T00:00:00Z" },
      ])
      .mockResolvedValueOnce([{ id: 1001 }]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // pre_activation filtered (current run); build failed → cancel
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "false");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("build"));
  });

  it("should not call listJobsForWorkflowRun when GITHUB_RUN_ID is not set", async () => {
    // GITHUB_RUN_ID is not set (already cleared in afterEach)
    mockGithub.paginate.mockResolvedValue([{ name: "build", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z" }]);

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
    // Only one paginate call (listForRef), not two
    expect(mockGithub.paginate).toHaveBeenCalledTimes(1);
  });

  it("should continue gracefully when listJobsForWorkflowRun API call fails", async () => {
    process.env.GITHUB_RUN_ID = "99999";

    mockGithub.paginate.mockResolvedValueOnce([{ id: 2001, name: "build", status: "completed", conclusion: "success", started_at: "2024-01-01T00:00:00Z" }]).mockRejectedValueOnce(new Error("API error fetching jobs"));

    const { main } = await import("./check_skip_if_check_failing.cjs");
    await main();

    // API error for current run jobs → warning emitted, but workflow still evaluates remaining checks
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not fetch jobs for current workflow run"));
    // build passed → allow
    expect(mockCore.setOutput).toHaveBeenCalledWith("skip_if_check_failing_ok", "true");
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });
});
