// @ts-check
// NOTE: This file uses ESM `import` syntax even though it has a `.cjs` extension.
// Vitest's bundler transforms `.cjs` test files to ESM at test time; `require("vitest")`
// is not supported by Vitest and would fail at runtime. Plain-Node execution of this
// file outside Vitest is not supported — run tests with `vitest run`.
import { describe, it, expect, beforeEach, vi } from "vitest";
import { createExpiredEntityHandler } from "./expired_entity_handler_factory.cjs";

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
};

describe("expired_entity_handler_factory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_DEFAULT_UTC;
  });

  it("creates a handler that comments, closes, and returns a closed record", async () => {
    const addComment = vi.fn().mockResolvedValue({ id: 1 });
    const closeEntity = vi.fn().mockResolvedValue({ state: "closed" });

    const handler = createExpiredEntityHandler({
      core: mockCore,
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
    const addComment = vi.fn();
    const closeEntity = vi.fn();

    const handler = createExpiredEntityHandler({
      core: mockCore,
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

  it("runs normal flow when beforeComment returns undefined", async () => {
    const addComment = vi.fn().mockResolvedValue({});
    const closeEntity = vi.fn().mockResolvedValue({});

    const handler = createExpiredEntityHandler({
      core: mockCore,
      workflowName: "Expired Cleanup",
      workflowId: "expired-cleanup",
      runUrl: "https://github.com/testowner/testrepo/actions/runs/1",
      entityNoun: "discussion",
      entityLabel: "Discussion",
      beforeComment: async () => undefined,
      addComment,
      closeEntity,
    });

    const result = await handler({
      number: 1,
      title: "T",
      url: "https://github.com/testowner/testrepo/discussions/1",
      expirationDate: new Date("2020-01-20T09:20:00.000Z"),
    });

    expect(addComment).toHaveBeenCalledTimes(1);
    expect(closeEntity).toHaveBeenCalledTimes(1);
    expect(result.status).toBe("closed");
  });
});
