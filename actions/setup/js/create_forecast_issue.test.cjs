// @ts-check
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
};

const mockContext = {
  repo: {
    owner: "octo",
    repo: "repo",
  },
  serverUrl: "https://github.com",
};

global.core = mockCore;
global.context = mockContext;

describe("create_forecast_issue", () => {
  let mockGithub;
  let mockFs;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
    process.env.GITHUB_RUN_ID = "123456";
    mockFs = {
      existsSync: vi.fn(),
      readFileSync: vi.fn(),
    };
    vi.doMock("node:fs", () => mockFs);
    mockGithub = {
      rest: {
        issues: {
          create: vi.fn().mockResolvedValue({
            data: {
              number: 42,
              html_url: "https://github.com/octo/repo/issues/42",
            },
          }),
        },
      },
    };
    global.github = mockGithub;
  });

  it("renders markdown forecast issue body with pretty ET and source run footnote", async () => {
    const module = await import("./create_forecast_issue.cjs");
    const body = module.buildForecastIssueBody(
      {
        period: "month",
        workflows: [
          {
            workflow_id: "wf|a",
            sampled_runs: 3,
            monte_carlo: {
              p50_projected_effective_tokens: 12345.6,
            },
          },
          {
            workflow_id: "wf-b",
            sampled_runs: 5,
            projected_effective_tokens: 0,
          },
        ],
      },
      {
        owner: "octo",
        repo: "repo",
        serverUrl: "https://github.com",
        runID: "123456",
        generatedAtISO: "2026-01-01T00:00:00.000Z",
      },
    );

    expect(body).toContain("| Workflow | Sampled runs | Forecast ET (P50) |");
    expect(body).toContain("| wf\\|a | 3 | 12,346 |");
    expect(body).toContain("> 1 workflow has sampled runs but forecast ET is 0. This usually indicates missing token usage in cached run summaries for sampled runs.");
    expect(body).toContain("_Forecast source run: [#123456](https://github.com/octo/repo/actions/runs/123456)._");
  });

  it("adds all-projected-zero diagnostics when every projected ET is zero", async () => {
    const module = await import("./create_forecast_issue.cjs");
    const body = module.buildForecastIssueBody(
      {
        period: "month",
        workflows: [
          { workflow_id: "wf-1", sampled_runs: 2, projected_effective_tokens: 0 },
          { workflow_id: "wf-2", sampled_runs: 0, projected_effective_tokens: 0 },
        ],
      },
      {
        owner: "octo",
        repo: "repo",
        serverUrl: "https://github.com",
        generatedAtISO: "2026-01-01T00:00:00.000Z",
      },
    );

    expect(body).toContain("All projected ET values are 0 even after cache warm-up.");
  });

  it("warns and skips when report file is missing", async () => {
    mockFs.existsSync.mockReturnValue(false);

    const module = await import("./create_forecast_issue.cjs");
    await module.main();

    expect(mockCore.warning).toHaveBeenCalledWith("Forecast report JSON not found at ./.cache/gh-aw/forecast/report.json; skipping issue creation.");
    expect(mockGithub.rest.issues.create).not.toHaveBeenCalled();
  });
});
