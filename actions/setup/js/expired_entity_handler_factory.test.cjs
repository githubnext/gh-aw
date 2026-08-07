// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
};

global.core = mockCore;

describe("expired_entity_handler_factory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_DEFAULT_UTC;
  });

  it("creates a handler that comments, closes, and returns a closed record", async () => {
    const { createExpiredEntityHandler } = await import("./expired_entity_handler_factory.cjs");
    const addComment = vi.fn().mockResolvedValue({ id: 1 });
    const closeEntity = vi.fn().mockResolvedValue({ state: "closed" });

    const handler = createExpiredEntityHandler({
      github: {},
      owner: "testowner",
      repo: "testrepo",
      workflowName: "Expired Cleanup",
      workflowId: "expired-cleanup",
      runUrl: "https://github.com/testowner/testrepo/actions/runs/1",
      entityNoun: "issue",
      entityLabel: "Issue",
      addComment,
      closeEntity,
    });

    const result = await handler({
      number: 42,
      title: "Expired issue",
      url: "https://github.com/testowner/testrepo/issues/42",
      expirationDate: new Date("2020-01-20T09:20:00.000Z"),
    });

    expect(addComment).toHaveBeenCalledWith(expect.objectContaining({ number: 42 }), expect.stringContaining("This issue was automatically closed because it expired on"));
    expect(closeEntity).toHaveBeenCalledWith(expect.objectContaining({ number: 42 }));
    expect(result).toEqual({
      status: "closed",
      record: {
        number: 42,
        title: "Expired issue",
        url: "https://github.com/testowner/testrepo/issues/42",
      },
    });
  });

  it("allows a pre-close hook to skip the shared comment flow", async () => {
    const { createExpiredEntityHandler } = await import("./expired_entity_handler_factory.cjs");
    const addComment = vi.fn();
    const closeEntity = vi.fn();

    const handler = createExpiredEntityHandler({
      github: {},
      owner: "testowner",
      repo: "testrepo",
      workflowName: "Expired Cleanup",
      workflowId: "expired-cleanup",
      runUrl: "https://github.com/testowner/testrepo/actions/runs/1",
      entityNoun: "discussion",
      entityLabel: "Discussion",
      beforeComment: async entity => ({
        status: "skipped",
        record: { number: entity.number, title: entity.title, url: entity.url },
      }),
      addComment,
      closeEntity,
    });

    const result = await handler({
      number: 7,
      title: "Already handled discussion",
      url: "https://github.com/testowner/testrepo/discussions/7",
      expirationDate: new Date("2020-01-20T09:20:00.000Z"),
    });

    expect(addComment).not.toHaveBeenCalled();
    expect(closeEntity).not.toHaveBeenCalled();
    expect(result).toEqual({
      status: "skipped",
      record: {
        number: 7,
        title: "Already handled discussion",
        url: "https://github.com/testowner/testrepo/discussions/7",
      },
    });
  });
});
