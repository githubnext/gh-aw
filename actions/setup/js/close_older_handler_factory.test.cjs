import { beforeEach, describe, expect, it, vi } from "vitest";

global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

describe("close_older_handler_factory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("createCloseOlderSearchAdapter places excludeNumber between leading and trailing args", async () => {
    const { createCloseOlderSearchAdapter } = await import("./close_older_handler_factory.cjs");
    const searchFn = vi.fn().mockResolvedValue([]);
    const adaptedSearch = createCloseOlderSearchAdapter(searchFn, ["category-id"], ["caller-id", "close-key"]);

    await adaptedSearch("github", "owner", "repo", "workflow-id", 99);

    expect(searchFn).toHaveBeenCalledWith("github", "owner", "repo", "workflow-id", "category-id", 99, "caller-id", "close-key");
  });

  it("closeOlderWithDescriptor delegates to closeOlderEntities and maps results", async () => {
    const { closeOlderWithDescriptor } = await import("./close_older_handler_factory.cjs");
    const searchOlderEntities = vi.fn().mockResolvedValue([{ number: 123, title: "Old issue", html_url: "https://github.com/owner/repo/issues/123" }]);
    const addComment = vi.fn().mockResolvedValue({ id: 1, html_url: "https://github.com/owner/repo/issues/123#issuecomment-1" });
    const closeEntity = vi.fn().mockResolvedValue({ number: 123, html_url: "https://github.com/owner/repo/issues/123" });

    const result = await closeOlderWithDescriptor({
      github: "github",
      owner: "owner",
      repo: "repo",
      workflowId: "workflow-id",
      newEntity: { number: 124, html_url: "https://github.com/owner/repo/issues/124" },
      workflowName: "Workflow",
      runUrl: "https://github.com/owner/repo/actions/runs/1",
      entityType: "issue",
      entityTypePlural: "issues",
      getCloseMessage: () => "close message",
      searchOlderEntities,
      addComment,
      closeEntity,
      delayMs: 500,
      getEntityId: entity => entity.number,
      getEntityUrl: entity => entity.html_url,
      mapClosedEntity: item => ({ number: item.number, html_url: item.html_url || "" }),
    });

    expect(result).toEqual([{ number: 123, html_url: "https://github.com/owner/repo/issues/123" }]);
    expect(searchOlderEntities).toHaveBeenCalledWith("github", "owner", "repo", "workflow-id", 124);
    expect(addComment).toHaveBeenCalledWith("github", "owner", "repo", 123, "close message");
    expect(closeEntity).toHaveBeenCalledWith("github", "owner", "repo", 123);
  });
});
