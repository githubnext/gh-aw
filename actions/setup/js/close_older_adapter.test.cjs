// @ts-check

import { describe, it, expect, beforeEach, vi } from "vitest";
import { closeOlderWithAdapter } from "./close_older_adapter.cjs";

describe("close_older_adapter", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
    };
  });

  it("forwards callerWorkflowId and closeOlderKey to searchOlderEntities", async () => {
    const github = {};
    const searchOlderEntities = vi.fn().mockResolvedValue([
      {
        number: 12,
        title: "Older issue",
        html_url: "https://example/12",
      },
    ]);
    const getCloseMessage = vi.fn().mockReturnValue("close message");
    const addComment = vi.fn().mockResolvedValue({ id: 1 });
    const closeEntity = vi.fn().mockResolvedValue({ number: 12, html_url: "https://example/12" });

    const result = await closeOlderWithAdapter({
      github,
      owner: "owner",
      repo: "repo",
      workflowId: "workflow",
      newEntity: { number: 55, html_url: "https://example/new" },
      workflowName: "Workflow",
      runUrl: "https://example/run",
      callerWorkflowId: "caller-id",
      closeOlderKey: "close-key",
      entityType: "issue",
      entityTypePlural: "issues",
      searchOlderEntities,
      getCloseMessage,
      addComment,
      closeEntity,
      delayMs: 1,
      messageParams: params => ({ transformedEntityUrl: params.newEntityUrl, transformedEntityNumber: params.newEntityNumber }),
    });

    expect(searchOlderEntities).toHaveBeenCalledWith(github, "owner", "repo", "workflow", 55, "caller-id", "close-key");
    expect(result).toEqual([{ number: 12, html_url: "https://example/12" }]);
  });

  it("applies messageParams transformation before getCloseMessage", async () => {
    const github = {};
    const getCloseMessage = vi.fn().mockReturnValue("close message");
    const addComment = vi.fn().mockResolvedValue({ id: 1 });
    const closeEntity = vi.fn().mockResolvedValue({ number: 12, html_url: "https://example/12" });

    await closeOlderWithAdapter({
      github,
      owner: "owner",
      repo: "repo",
      workflowId: "workflow",
      newEntity: { number: 55, html_url: "https://example/new" },
      workflowName: "Workflow",
      runUrl: "https://example/run",
      callerWorkflowId: "caller-id",
      closeOlderKey: "close-key",
      entityType: "issue",
      entityTypePlural: "issues",
      searchOlderEntities: vi.fn().mockResolvedValue([{ number: 12, title: "Older issue", html_url: "https://example/12" }]),
      getCloseMessage,
      addComment,
      closeEntity,
      delayMs: 1,
      messageParams: params => ({ transformedEntityUrl: params.newEntityUrl, transformedEntityNumber: params.newEntityNumber }),
    });

    expect(getCloseMessage).toHaveBeenCalledWith({ transformedEntityUrl: "https://example/new", transformedEntityNumber: 55 });
    expect(addComment).toHaveBeenCalledWith(github, "owner", "repo", 12, "close message");
    expect(closeEntity).toHaveBeenCalledWith(github, "owner", "repo", 12);
  });

  it("normalizes missing html_url in mapped result", async () => {
    const result = await closeOlderWithAdapter({
      github: {},
      owner: "owner",
      repo: "repo",
      workflowId: "workflow",
      newEntity: { number: 10, html_url: "https://example/new" },
      workflowName: "Workflow",
      runUrl: "https://example/run",
      callerWorkflowId: undefined,
      closeOlderKey: undefined,
      entityType: "issue",
      entityTypePlural: "issues",
      searchOlderEntities: vi.fn().mockResolvedValue([{ number: 22, title: "Older issue" }]),
      getCloseMessage: vi.fn().mockReturnValue("close message"),
      addComment: vi.fn().mockResolvedValue({ id: 1 }),
      closeEntity: vi.fn().mockResolvedValue({ number: 22 }),
      delayMs: 1,
      messageParams: params => params,
    });

    expect(result).toEqual([{ number: 22, html_url: "" }]);
  });
});
