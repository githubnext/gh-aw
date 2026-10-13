// @ts-check
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockCore = { info: vi.fn() };
const mockContext = {
  repo: { owner: "testowner", repo: "testrepo" },
  serverUrl: "https://github.com",
};

describe("cleanup_stale_steering_issues", () => {
  let mockGithub;
  const oldCreatedAt = new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString();
  const makeIssue = (overrides = {}) => ({
    number: 101,
    title: "[WIP] Example: work in progress",
    body: "This issue was created by [the workflow run](https://github.com/testowner/testrepo/actions/runs/12345).",
    created_at: oldCreatedAt,
    ...overrides,
  });

  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
    global.core = mockCore;
    global.context = mockContext;
    mockGithub = {
      paginate: vi.fn(),
      rest: {
        search: {
          issuesAndPullRequests: vi.fn(),
        },
        issues: {
          listComments: vi.fn(),
          createComment: vi.fn(),
          update: vi.fn(),
        },
        actions: {
          getWorkflowRun: vi.fn(),
        },
      },
    };
    global.github = mockGithub;
  });

  it("comments on and closes stale steering issues linked to successful runs", async () => {
    mockGithub.paginate.mockResolvedValueOnce([makeIssue()]).mockResolvedValueOnce([]);
    mockGithub.rest.actions.getWorkflowRun.mockResolvedValue({ data: { status: "completed", conclusion: "success" } });

    const { main, CLEANUP_COMMENT } = await import("./cleanup_stale_steering_issues.cjs");
    await main();

    expect(mockGithub.paginate).toHaveBeenCalledWith(mockGithub.rest.search.issuesAndPullRequests, {
      q: 'repo:testowner/testrepo is:issue is:open in:title "[WIP]"',
      per_page: 100,
      sort: "created",
      order: "asc",
    });
    expect(mockGithub.rest.actions.getWorkflowRun).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      run_id: 12345,
    });
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 101,
      body: CLEANUP_COMMENT,
    });
    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 101,
      state: "closed",
      state_reason: "completed",
    });
  });

  it("does not touch active, failed, malformed, or unrelated issues", async () => {
    const issues = [
      makeIssue({ number: 1, created_at: new Date().toISOString() }),
      makeIssue({ number: 2, title: "Unrelated issue" }),
      makeIssue({ number: 3, body: "No workflow run link" }),
      makeIssue({ number: 4, body: "https://example.com/owner/repo/actions/runs/12345" }),
      makeIssue({ number: 5 }),
      makeIssue({ number: 6 }),
      makeIssue({ number: 7, pull_request: { url: "https://github.com/testowner/testrepo/pull/7" } }),
    ];
    mockGithub.paginate.mockResolvedValueOnce(issues);
    mockGithub.rest.actions.getWorkflowRun.mockResolvedValueOnce({ data: { status: "completed", conclusion: "failure" } }).mockResolvedValueOnce({ data: { status: "in_progress", conclusion: null } });

    const { main } = await import("./cleanup_stale_steering_issues.cjs");
    await main();

    expect(mockGithub.rest.actions.getWorkflowRun).toHaveBeenCalledTimes(2);
    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
  });

  it("does not duplicate the cleanup comment when closure needs another attempt", async () => {
    mockGithub.paginate.mockResolvedValueOnce([makeIssue()]).mockResolvedValueOnce([{ body: "Already handled <!-- gh-aw:stale-steering-issue-cleanup -->" }]);
    mockGithub.rest.actions.getWorkflowRun.mockResolvedValue({ data: { status: "completed", conclusion: "success" } });

    const { main } = await import("./cleanup_stale_steering_issues.cjs");
    await main();

    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
    expect(mockGithub.rest.issues.update).toHaveBeenCalledTimes(1);
  });

  it("extracts only run IDs belonging to the current repository", async () => {
    const { getWorkflowRunId } = await import("./cleanup_stale_steering_issues.cjs");

    expect(getWorkflowRunId("Run: [the workflow run](https://github.com/testowner/testrepo/actions/runs/12345)", "testowner", "testrepo")).toBe(12345);
    expect(getWorkflowRunId("[the workflow run](https://github.com/other/repo/actions/runs/12345)", "testowner", "testrepo")).toBeNull();
    expect(getWorkflowRunId("[the workflow run](https://github.com/testowner/testrepo/actions/runs/not-a-number)", "testowner", "testrepo")).toBeNull();
  });
});
