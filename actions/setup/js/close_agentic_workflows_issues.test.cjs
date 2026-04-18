// @ts-check
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockCore = {
  info: vi.fn(),
};

const mockContext = {
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
};

global.core = mockCore;
global.context = mockContext;

describe("close_agentic_workflows_issues", () => {
  let mockGithub;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
    mockGithub = {
      paginate: vi.fn(),
      graphql: vi.fn(),
      rest: {
        issues: {
          listForRepo: vi.fn(),
          createComment: vi.fn(),
        },
      },
    };
    global.github = mockGithub;
  });

  it("closes only open issues (not pull requests) with not_planned state reason via GraphQL", async () => {
    mockGithub.paginate.mockResolvedValueOnce([
      { number: 101, title: "Issue A", node_id: "I_101" },
      { number: 102, title: "PR B", node_id: "PR_102", pull_request: { url: "https://example.com/pr/102" } },
    ]);

    const module = await import("./close_agentic_workflows_issues.cjs");
    await module.main();

    expect(mockGithub.paginate).toHaveBeenCalledWith(mockGithub.rest.issues.listForRepo, {
      owner: "testowner",
      repo: "testrepo",
      labels: "agentic-workflows",
      state: "open",
      per_page: 100,
    });

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledTimes(1);
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 101,
      body: module.NO_REPRO_MESSAGE,
    });

    expect(mockGithub.graphql).toHaveBeenCalledTimes(1);
    expect(mockGithub.graphql).toHaveBeenCalledWith(module.CLOSE_ISSUE_MUTATION, {
      issueId: "I_101",
      stateReason: "NOT_PLANNED",
    });
  });

  it("does nothing when no target issues are found", async () => {
    mockGithub.paginate.mockResolvedValueOnce([]);

    const { main } = await import("./close_agentic_workflows_issues.cjs");
    await main();

    expect(mockGithub.rest.issues.createComment).not.toHaveBeenCalled();
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });
});
