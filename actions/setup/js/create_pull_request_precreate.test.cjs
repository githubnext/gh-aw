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
        data: { state: "open", head: { ref: "gh-aw/pre-created/123-1" } },
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
