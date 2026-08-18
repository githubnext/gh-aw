// @ts-check
import { beforeEach, describe, expect, it, vi } from "vitest";

const { createOrUpdatePullRequest } = await import("./create_or_update_pull_request.cjs");

describe("create_pull_request pre-created PR reuse", () => {
  beforeEach(() => {
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
    };
  });

  it("updates the allocated PR instead of creating a second PR", async () => {
    const pulls = {
      get: vi.fn().mockResolvedValue({
        data: { state: "open", draft: true, node_id: "PR_node", head: { ref: "gh-aw/pre-created/123-1" } },
      }),
      update: vi.fn().mockResolvedValue({
        data: { number: 42, html_url: "https://github.com/owner/repo/pull/42" },
      }),
      create: vi.fn(),
    };

    const result = await createOrUpdatePullRequest({
      githubClient: { rest: { pulls } },
      repoParts: { owner: "owner", repo: "repo" },
      title: "Final title",
      body: "Final body",
      branchName: "gh-aw/pre-created/123-1",
      baseBranch: "main",
      draft: true,
      preCreatedPullRequestNumber: 42,
      preCreatedBranch: "gh-aw/pre-created/123-1",
    });

    expect(pulls.get).toHaveBeenCalledWith(expect.objectContaining({ pull_number: 42 }));
    expect(pulls.update).toHaveBeenCalledWith(
      expect.objectContaining({
        pull_number: 42,
        title: "Final title",
        body: "Final body",
      })
    );
    expect(pulls.create).not.toHaveBeenCalled();
    expect(result.data.number).toBe(42);
  });

  it("marks the allocated PR ready for review when draft is disabled", async () => {
    const pulls = {
      get: vi.fn().mockResolvedValue({
        data: { state: "open", draft: true, node_id: "PR_node", head: { ref: "gh-aw/pre-created/123-1" } },
      }),
      update: vi.fn().mockResolvedValue({
        data: { number: 42, html_url: "https://github.com/owner/repo/pull/42" },
      }),
      create: vi.fn(),
    };
    const graphql = vi.fn().mockResolvedValue({ markPullRequestReadyForReview: { pullRequest: { isDraft: false } } });

    await createOrUpdatePullRequest({
      githubClient: { rest: { pulls }, graphql },
      repoParts: { owner: "owner", repo: "repo" },
      title: "Final title",
      body: "Final body",
      branchName: "gh-aw/pre-created/123-1",
      baseBranch: "main",
      draft: false,
      preCreatedPullRequestNumber: 42,
      preCreatedBranch: "gh-aw/pre-created/123-1",
    });

    expect(graphql).toHaveBeenCalledWith(expect.stringContaining("markPullRequestReadyForReview"), { pullRequestId: "PR_node" });
  });

  it("leaves the allocated PR as a draft when the draft policy is enabled", async () => {
    const pulls = {
      get: vi.fn().mockResolvedValue({
        data: { state: "open", draft: true, node_id: "PR_node", head: { ref: "gh-aw/pre-created/123-1" } },
      }),
      update: vi.fn().mockResolvedValue({
        data: { number: 42, html_url: "https://github.com/owner/repo/pull/42" },
      }),
      create: vi.fn(),
    };
    const graphql = vi.fn();

    for (const draft of [true, undefined]) {
      await createOrUpdatePullRequest({
        githubClient: { rest: { pulls }, graphql },
        repoParts: { owner: "owner", repo: "repo" },
        title: "Final title",
        body: "Final body",
        branchName: "gh-aw/pre-created/123-1",
        baseBranch: "main",
        draft,
        preCreatedPullRequestNumber: 42,
        preCreatedBranch: "gh-aw/pre-created/123-1",
      });
    }

    expect(graphql).not.toHaveBeenCalled();
  });

  it("warns without failing when the allocated PR cannot be marked ready for review", async () => {
    const pulls = {
      get: vi.fn().mockResolvedValue({
        data: { state: "open", draft: true, node_id: "PR_node", head: { ref: "gh-aw/pre-created/123-1" } },
      }),
      update: vi.fn().mockResolvedValue({
        data: { number: 42, html_url: "https://github.com/owner/repo/pull/42" },
      }),
      create: vi.fn(),
    };
    const graphql = vi.fn().mockRejectedValue(new Error("graphql boom"));

    const result = await createOrUpdatePullRequest({
      githubClient: { rest: { pulls }, graphql },
      repoParts: { owner: "owner", repo: "repo" },
      title: "Final title",
      body: "Final body",
      branchName: "gh-aw/pre-created/123-1",
      baseBranch: "main",
      draft: false,
      preCreatedPullRequestNumber: 42,
      preCreatedBranch: "gh-aw/pre-created/123-1",
    });

    expect(result.data.number).toBe(42);
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("graphql boom"));
  });

  it("converts the allocated PR back to draft when draft is required", async () => {
    const pulls = {
      get: vi.fn().mockResolvedValue({
        data: { state: "open", draft: false, node_id: "PR_node", head: { ref: "gh-aw/pre-created/123-1" } },
      }),
      update: vi.fn().mockResolvedValue({
        data: { number: 42, html_url: "https://github.com/owner/repo/pull/42" },
      }),
      create: vi.fn(),
    };
    const graphql = vi.fn().mockResolvedValue({ convertPullRequestToDraft: { pullRequest: { isDraft: true } } });

    await createOrUpdatePullRequest({
      githubClient: { rest: { pulls }, graphql },
      repoParts: { owner: "owner", repo: "repo" },
      title: "Final title",
      body: "Final body",
      branchName: "gh-aw/pre-created/123-1",
      baseBranch: "main",
      draft: true,
      preCreatedPullRequestNumber: 42,
      preCreatedBranch: "gh-aw/pre-created/123-1",
    });

    expect(graphql).toHaveBeenCalledWith(expect.stringContaining("convertPullRequestToDraft"), { pullRequestId: "PR_node" });
  });

  it("fails immediately without retrying when the allocated PR is no longer usable", async () => {
    const pulls = {
      get: vi.fn().mockResolvedValue({
        data: { state: "closed", draft: true, node_id: "PR_node", head: { ref: "gh-aw/pre-created/123-1" } },
      }),
      update: vi.fn(),
      create: vi.fn(),
    };

    await expect(
      createOrUpdatePullRequest({
        githubClient: { rest: { pulls } },
        repoParts: { owner: "owner", repo: "repo" },
        title: "Final title",
        body: "Final body",
        branchName: "gh-aw/pre-created/123-1",
        baseBranch: "main",
        draft: true,
        preCreatedPullRequestNumber: 42,
        preCreatedBranch: "gh-aw/pre-created/123-1",
      })
    ).rejects.toThrow(/is not open on branch/);

    expect(pulls.get).toHaveBeenCalledOnce();
    expect(pulls.update).not.toHaveBeenCalled();
    expect(pulls.create).not.toHaveBeenCalled();
  });

  it("creates a PR when the allocated PR number is not positive", async () => {
    const pulls = {
      get: vi.fn(),
      update: vi.fn(),
      create: vi.fn().mockResolvedValue({
        data: { number: 43, html_url: "https://github.com/owner/repo/pull/43" },
      }),
    };

    const result = await createOrUpdatePullRequest({
      githubClient: { rest: { pulls } },
      repoParts: { owner: "owner", repo: "repo" },
      title: "Final title",
      body: "Final body",
      branchName: "gh-aw/pre-created/123-1",
      baseBranch: "main",
      draft: true,
      preCreatedPullRequestNumber: 0,
      preCreatedBranch: "gh-aw/pre-created/123-1",
    });

    expect(pulls.create).toHaveBeenCalledOnce();
    expect(pulls.get).not.toHaveBeenCalled();
    expect(pulls.update).not.toHaveBeenCalled();
    expect(result.data.number).toBe(43);
  });
});
