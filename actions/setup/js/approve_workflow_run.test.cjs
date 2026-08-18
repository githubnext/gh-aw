// @ts-check
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGetWorkflowRun = vi.fn();
const mockGetWorkflow = vi.fn();
const mockApproveWorkflowRun = vi.fn();
const mockListFiles = vi.fn();
const mockGetPullRequest = vi.fn();

global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

global.context = {
  repo: { owner: "test-owner", repo: "test-repo" },
  payload: { pull_request: { number: 42 } },
};

global.github = {
  rest: {
    actions: {
      getWorkflowRun: mockGetWorkflowRun,
      getWorkflow: mockGetWorkflow,
      approveWorkflowRun: mockApproveWorkflowRun,
    },
    pulls: {
      get: mockGetPullRequest,
      listFiles: mockListFiles,
    },
  },
  paginate: vi.fn(async (method, params) => (await method(params)).data),
};
global.getOctokit = vi.fn(() => global.github);

const pendingPullRequestRun = {
  event: "pull_request",
  status: "waiting",
  conclusion: null,
  html_url: "https://github.com/test-owner/test-repo/actions/runs/123",
  workflow_id: 456,
  pull_requests: [{ number: 42 }],
};
const externalTokenConfig = { "github-token": "external-token", allowed_workflows: ["test.yml"] };

describe("approve_workflow_run", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.context.eventName = undefined;
    global.context.payload = { pull_request: { number: 42 } };
    mockGetWorkflowRun.mockResolvedValue({ data: pendingPullRequestRun });
    mockGetWorkflow.mockResolvedValue({ data: { path: ".github/workflows/test.yaml" } });
    mockApproveWorkflowRun.mockResolvedValue({ status: 201 });
    mockGetPullRequest.mockResolvedValue({ data: { head: { repo: { fork: false } } } });
    mockListFiles.mockResolvedValue({ data: [] });
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

  it("matches allowed workflow filenames after normalizing the extension", async () => {
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, allowed_workflows: ["test.yaml"] });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(true);
    expect(mockGetWorkflow).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: 456,
    });
  });

  it("matches allowed workflow filename wildcards", async () => {
    mockGetWorkflow.mockResolvedValue({ data: { path: ".github/workflows/pull-request-ci.yml" } });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, allowed_workflows: ["pull-request-*.yaml"] });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(true);
  });

  it("rejects runs from workflows outside the allowed list", async () => {
    mockGetWorkflow.mockResolvedValue({ data: { path: ".github/workflows/release.yml" } });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("does not match an allowed workflow");
    expect(mockGetPullRequest).not.toHaveBeenCalled();
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("rejects path-containing allowed workflow patterns", async () => {
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, allowed_workflows: [".github/workflows/test.yml"] });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("does not match an allowed workflow");
    expect(mockGetPullRequest).not.toHaveBeenCalled();
  });

  it("rejects fork pull requests by default", async () => {
    mockGetPullRequest.mockResolvedValue({ data: { head: { repo: { fork: true } } } });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("set fork: true");
    expect(mockListFiles).not.toHaveBeenCalled();
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("allows fork pull requests when explicitly enabled", async () => {
    mockGetPullRequest.mockResolvedValue({ data: { head: { repo: { fork: true } } } });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, fork: true });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(true);
    expect(mockApproveWorkflowRun).toHaveBeenCalledTimes(1);
  });

  it("rejects a run when any associated pull request is from a fork by default", async () => {
    mockGetWorkflowRun.mockResolvedValue({
      data: { ...pendingPullRequestRun, pull_requests: [{ number: 42 }, { number: 43 }] },
    });
    mockGetPullRequest.mockImplementation(async ({ pull_number: pullRequestNumber }) => ({
      data: { head: { repo: { fork: pullRequestNumber === 43 } } },
    }));
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, allowed_pull_requests: ["43"] });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("set fork: true");
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("rejects pull requests with an unavailable fork repository", async () => {
    mockGetPullRequest.mockResolvedValue({ data: { head: { repo: null } } });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("fork repository is unavailable");
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("rejects pull_request_target events before accessing GitHub", async () => {
    global.context.eventName = "pull_request_target";
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("pull_request_target");
    expect(mockGetWorkflowRun).not.toHaveBeenCalled();
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("rejects a run for a pull request other than the current pull request", async () => {
    mockGetWorkflowRun.mockResolvedValue({
      data: { ...pendingPullRequestRun, pull_requests: [{ number: 43 }] },
    });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not associated");
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("rejects a run when its pull request modifies protected files", async () => {
    mockListFiles.mockResolvedValue({ data: [{ filename: ".github/workflows/ci.yml" }] });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, protected_path_prefixes: [".github/"] });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("modifies protected files");
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("allows files absent from the handler's protected-file configuration", async () => {
    mockListFiles.mockResolvedValue({ data: [{ filename: "package.json" }] });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, protected_files: [] });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(true);
    expect(mockApproveWorkflowRun).toHaveBeenCalledTimes(1);
  });

  it("rejects malformed workflow run pull request data without a triggering pull request", async () => {
    global.context.payload = {};
    mockGetWorkflowRun.mockResolvedValue({
      data: { ...pendingPullRequestRun, pull_requests: [{}] },
    });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not associated");
    expect(mockApproveWorkflowRun).not.toHaveBeenCalled();
  });

  it("approves a run for an explicitly allowed pull request", async () => {
    mockGetWorkflowRun.mockResolvedValue({
      data: { ...pendingPullRequestRun, pull_requests: [{ number: 43 }] },
    });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, allowed_pull_requests: ["43"] });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(true);
    expect(mockApproveWorkflowRun).toHaveBeenCalledTimes(1);
  });

  it("approves a run for a pull request allowed by a JSON-string list", async () => {
    global.context.payload = {};
    mockGetWorkflowRun.mockResolvedValue({
      data: { ...pendingPullRequestRun, pull_requests: [{ number: 43 }] },
    });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, allowed_pull_requests: '["43"]' });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(true);
    expect(mockApproveWorkflowRun).toHaveBeenCalledTimes(1);
  });

  it("approves a run when every associated pull request is triggering or explicitly allowed", async () => {
    mockGetWorkflowRun.mockResolvedValue({
      data: { ...pendingPullRequestRun, pull_requests: [{ number: 42 }, { number: 43 }] },
    });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, allowed_pull_requests: ["43"] });

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(true);
    expect(mockApproveWorkflowRun).toHaveBeenCalledTimes(1);
  });

  it("rejects a run when any associated pull request is not authorized", async () => {
    mockGetWorkflowRun.mockResolvedValue({
      data: { ...pendingPullRequestRun, pull_requests: [{ number: 42 }, { number: 43 }] },
    });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main(externalTokenConfig);

    const result = await handler({ run_id: 123 }, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not associated exclusively");
    expect(mockGetPullRequest).not.toHaveBeenCalled();
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

  it("does not consume the maximum when approval fails", async () => {
    mockApproveWorkflowRun.mockRejectedValueOnce(new Error("temporary API failure")).mockResolvedValueOnce({ status: 201 });
    const { main } = require("./approve_workflow_run.cjs");
    const handler = await main({ ...externalTokenConfig, max: 1 });

    expect((await handler({ run_id: 122 }, {})).success).toBe(false);
    expect((await handler({ run_id: 123 }, {})).success).toBe(true);
    expect(mockApproveWorkflowRun).toHaveBeenCalledTimes(2);
  });
});
