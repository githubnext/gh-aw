// @ts-check
import { beforeEach, describe, expect, it, vi } from "vitest";

describe("check_cooldown", () => {
  let mockCore;
  let mockGithub;
  let checkCooldown;

  beforeEach(async () => {
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      setOutput: vi.fn(),
    };
    mockGithub = {
      paginate: vi.fn(),
      rest: {
        actions: {
          listJobsForWorkflowRun: vi.fn(),
          listWorkflowRuns: vi.fn(),
        },
        rateLimit: {
          get: vi.fn().mockResolvedValue({ data: { resources: {} } }),
        },
      },
    };
    global.core = mockCore;
    global.github = mockGithub;
    global.context = {
      repo: { owner: "octo-org", repo: "octo-repo" },
      workflow: "Cooldown workflow",
      runId: 100,
    };
    process.env.GH_AW_COOLDOWN_SECONDS = "3600";
    process.env.GITHUB_WORKFLOW_REF = "octo-org/octo-repo/.github/workflows/cooldown.lock.yml@refs/heads/main";

    vi.resetModules();
    checkCooldown = await import("./check_cooldown.cjs");
  });

  it("allows the first agent execution", async () => {
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({ data: { workflow_runs: [] } });

    await checkCooldown.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("cooldown_ok", "true");
    expect(mockGithub.rest.actions.listWorkflowRuns).toHaveBeenCalledWith({
      owner: "octo-org",
      repo: "octo-repo",
      workflow_id: "cooldown.lock.yml",
      status: "completed",
      per_page: 100,
      page: 1,
    });
  });

  it("blocks when the last agent run completed within the cooldown", async () => {
    const completedAt = new Date(Date.now() - 10 * 60 * 1000).toISOString();
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({
      data: { workflow_runs: [{ id: 99, completed_at: completedAt }] },
    });
    mockGithub.paginate.mockResolvedValue([{ name: "agent", conclusion: "success", started_at: completedAt }]);

    await checkCooldown.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("cooldown_ok", "false");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("completed within the cooldown period"));
  });

  it("allows execution when the last agent run completed before the cooldown", async () => {
    const completedAt = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({
      data: { workflow_runs: [{ id: 98, completed_at: completedAt }] },
    });
    mockGithub.paginate.mockResolvedValue([{ name: "agent", conclusion: "failure", started_at: completedAt }]);

    await checkCooldown.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("cooldown_ok", "true");
  });

  it("ignores completed runs where the agent job was skipped", async () => {
    const completedAt = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    mockGithub.rest.actions.listWorkflowRuns.mockResolvedValue({
      data: { workflow_runs: [{ id: 97, completed_at: completedAt }] },
    });
    mockGithub.paginate.mockResolvedValue([{ name: "agent", conclusion: "skipped", started_at: null }]);

    await checkCooldown.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("cooldown_ok", "true");
  });

  it("fails open when run history cannot be queried", async () => {
    mockGithub.rest.actions.listWorkflowRuns.mockRejectedValue(new Error("API unavailable"));

    await checkCooldown.main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("cooldown_ok", "true");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Cooldown check failed"));
  });
});
