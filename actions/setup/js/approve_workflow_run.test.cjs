// @ts-check
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGetWorkflowRun = vi.fn();
const mockApproveWorkflowRun = vi.fn();

global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

global.context = {
  repo: { owner: "test-owner", repo: "test-repo" },
};

global.github = {
  rest: {
    actions: {
      getWorkflowRun: mockGetWorkflowRun,
      approveWorkflowRun: mockApproveWorkflowRun,
    },
  },
};
global.getOctokit = vi.fn(() => global.github);

const pendingPullRequestRun = {
  event: "pull_request",
  status: "waiting",
  conclusion: null,
  html_url: "https://github.com/test-owner/test-repo/actions/runs/123",
  pull_requests: [{ number: 42 }],
};
const externalTokenConfig = { "github-token": "external-token" };

describe("approve_workflow_run", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetWorkflowRun.mockResolvedValue({ data: pendingPullRequestRun });
    mockApproveWorkflowRun.mockResolvedValue({ status: 201 });
  });

  it("approves an eligible pull request workflow run", async () => {
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result).toEqual({
      success: true,
      run_id: 123,
      url: pendingPullRequestRun.html_url,
    });
    expect(mockApproveWorkflowRun).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      run_id: 123,
    });
  });

  it("accepts a decimal run ID string", async () => {
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: "123" }, {});

    expect(result.success).toBe(true);
    expect(mockGetWorkflowRun).toHaveBeenCalledWith(expect.objectContaining({ run_id: 123 }));
  });

  it.each([undefined, "", 0, -1, 1.5, "abc", "12abc"])("rejects invalid run ID %j", async runId => {
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: runId }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("positive integer");
    expect(mockGetWorkflowRun).not.toHaveBeenCalled();
  });

  it("rejects runs that are not associated with a pull request", async () => {
    mockGetWorkflowRun.mockResolvedValue({
      data: { ...pendingPullRequestRun, event: "push", pull_requests: [] },
    });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not associated with a pull request");
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("rejects runs that are not awaiting approval", async () => {
    mockGetWorkflowRun.mockResolvedValue({
      data: { ...pendingPullRequestRun, status: "completed" },
    });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not awaiting approval");
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("previews without approving in staged mode", async () => {
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ staged: true });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(true);
    expect(result.staged).toBe(true);
    expect(mockGetWorkflowRun).not.toHaveBeenCalled();
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("does not apply the maximum to staged previews", async () => {
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ staged: true, max: 1 });

    expect((await handler({ run_id: 123 }, {})).success).toBe(true);
    expect((await handler({ run_id: 124 }, {})).success).toBe(true);
    expect(mockGetWorkflowRun).not.toHaveBeenCalled();
  });

  it("rejects live approvals without an external token", async () => {
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main();

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("external github-token or GitHub App token");
    expect(mockGetWorkflowRun).not.toHaveBeenCalled();
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("enforces the configured maximum", async () => {
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, max: 1 });

    expect((await handler({ run_id: 123 }, {})).success).toBe(true);
    const result = await handler({ run_id: 124 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Max count of 1 reached");
  });

  it("does not consume the maximum for an ineligible run", async () => {
    mockGetWorkflowRun
      .mockResolvedValueOnce({
        data: { ...pendingPullRequestRun, status: "completed" },
      })
      .mockResolvedValueOnce({ data: pendingPullRequestRun });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, max: 1 });

    expect((await handler({ run_id: 122 }, {})).success).toBe(false);
    expect((await handler({ run_id: 123 }, {})).success).toBe(true);
    expect(mockApproveWorkflowRun).toHaveBeenCalledTimes(1);
  });
});
