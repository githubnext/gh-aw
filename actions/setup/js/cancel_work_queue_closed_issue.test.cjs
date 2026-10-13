import { describe, expect, it, vi } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { main, matchingWork, closedIssue, readIssuesConfig } = require("./cancel_work_queue_closed_issue.cjs");

const resource = { kind: "issue", host: "github.com", repository: "owner/repo", repository_id: "12", resource_id: "34", number: "7" };
const work = { work_id: "work_1", backing_issue: resource, state: "available" };
const context = {
  repo: { owner: "owner", repo: "repo" },
  issue: { number: 7 },
  sha: "a".repeat(40),
  eventName: "issues",
  payload: { action: "closed", issue: { id: 34, number: 7, labels: [{ name: "work" }, { name: "work:queued" }] }, repository: { id: 12, full_name: "owner/repo" } },
};
const trusted = {
  role: "administrator",
  principal: "2",
  repository: "owner/repo",
  workflow: ".github/workflows/agentic-workqueue.yml",
  run_id: "99",
  run_attempt: 1,
  authenticated: true,
  roles: ["administrator"],
  event: "issues",
  native_run: { repository: { id: 12 } },
};

function fixture(initialWork = work) {
  const state = { works: new Map([[initialWork.work_id, initialWork]]), requests: new Map() };
  const confirmed = { works: new Map([[initialWork.work_id, initialWork]]), requests: new Map() };
  const githubClient = {
    rest: {
      issues: { get: vi.fn(async () => ({ data: { id: 34, number: 7, state: "closed" } })), listComments: vi.fn(async () => ({ data: [] })), createComment: vi.fn(async () => ({})) },
    },
  };
  const publishWorkQueueRequest = vi.fn(async args => {
    const decision = await args.generateOperations(state, args.request, args.actor, Date.now(), "commit", []);
    if (decision.operations.length) {
      confirmed.works.set(initialWork.work_id, { ...initialWork, state: "cancelled", cancellation_reason: "related_issue_closed", cancellation_commit_id: "commit" });
      confirmed.requests.set(args.request.id, { request: args.request, operations: decision.operations });
    }
    return { publishedNow: decision.operations.length > 0 };
  });
  const options = {
    context,
    githubClient,
    issues: true,
    authenticatePublisher: vi.fn(async () => trusted),
    readWorkQueueLog: vi
      .fn()
      .mockResolvedValueOnce({ sha: "abc", state })
      .mockImplementation(async () => ({ sha: "def", state: confirmed })),
    closedIssue: vi.fn(async () => true),
    publishWorkQueueRequest,
    core: { info: vi.fn() },
  };
  return { githubClient, options, state, confirmed, publishWorkQueueRequest };
}

describe("closed backing Issue cancellation", () => {
  it.each([[], [{ name: "bug" }], [{ name: "work" }], [{ name: "work:queued" }], [{ name: "tasks" }, { name: "work:queued" }]].map(labels => ({ labels })))(
    "rejects unbound Issues before configuration and authentication reads: $labels",
    async ({ labels }) => {
      const event = fixture({ ...work, backing_issue: { ...resource, resource_id: "35" } });
      event.options.context = { ...context, payload: { ...context.payload, issue: { ...context.payload.issue, labels } } };
      delete event.options.issues;
      event.githubClient.rest.repos = { getContent: vi.fn().mockRejectedValue(new Error("API rate limit exceeded for installation")) };
      expect(await main(event.options)).toEqual({ status: "unbound" });
      expect(event.githubClient.rest.repos.getContent).not.toHaveBeenCalled();
      expect(event.options.authenticatePublisher).not.toHaveBeenCalled();
      expect(event.githubClient.rest.issues.get).not.toHaveBeenCalled();
      expect(event.options.readWorkQueueLog).toHaveBeenCalledOnce();
      expect(event.publishWorkQueueRequest).not.toHaveBeenCalled();
      expect(event.githubClient.rest.issues.createComment).not.toHaveBeenCalled();
    }
  );

  it.each(["work", "tasks", "team work"])("preserves custom tracking/status configuration for bound Issues: %s", async label => {
    const event = fixture();
    event.options.issues = { label };
    event.options.context = { ...context, payload: { ...context.payload, issue: { ...context.payload.issue, labels: [{ name: label }, { name: `${label.replace(/\s+/g, "-")}:needs-attention` }] } } };
    expect((await main(event.options)).status).toBe("cancelled");
    expect(event.options.authenticatePublisher).toHaveBeenCalledOnce();
    expect(event.options.closedIssue).toHaveBeenCalledTimes(2);
    expect(event.githubClient.rest.issues.get).not.toHaveBeenCalled();
  });

  it.each([[], undefined, null, [{ name: null }]].map(labels => ({ labels })))("does not reject a bound event with missing label hints: $labels", async ({ labels }) => {
    const event = fixture();
    event.options.context = { ...context, payload: { ...context.payload, issue: { ...context.payload.issue, labels } } };
    expect((await main(event.options)).status).toBe("cancelled");
    expect(event.options.closedIssue).toHaveBeenCalledTimes(2);
  });

  it("propagates API quota failures for candidate Issues without publishing or retrying", async () => {
    const event = fixture();
    delete event.options.issues;
    event.githubClient.rest.repos = { getContent: vi.fn().mockRejectedValue(Object.assign(new Error("API rate limit exceeded for installation"), { status: 403 })) };
    await expect(main(event.options)).rejects.toThrow("API rate limit exceeded");
    expect(event.githubClient.rest.repos.getContent).toHaveBeenCalledOnce();
    expect(event.options.authenticatePublisher).not.toHaveBeenCalled();
    expect(event.publishWorkQueueRequest).not.toHaveBeenCalled();
  });

  it("rejects ledger read failures before configuration, authentication or mutation", async () => {
    const event = fixture();
    event.options.readWorkQueueLog.mockReset().mockRejectedValue(new Error("ledger_invalid"));
    await expect(main(event.options)).rejects.toThrow("ledger_invalid");
    expect(event.options.authenticatePublisher).not.toHaveBeenCalled();
    expect(event.publishWorkQueueRequest).not.toHaveBeenCalled();
    expect(event.githubClient.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it.each([false, true])("uses one checked GraphQL read for an unrelated closure and no REST reads (initialized=%s)", async initialized => {
    const event = fixture();
    delete event.options.readWorkQueueLog;
    delete event.options.issues;
    const text = JSON.stringify(require("./work_queue_test_helpers.cjs").genesis()) + "\n";
    event.githubClient.graphql = vi.fn(async () => ({
      repository: {
        id: "repo-node",
        nameWithOwner: "owner/repo",
        ref: initialized ? { target: { oid: "b".repeat(40), tree: { oid: "c".repeat(40), entries: [{ name: "work-queue.jsonl", type: "blob", mode: 33188, oid: "d".repeat(40) }] } } } : null,
        log: initialized ? { oid: "d".repeat(40), text, byteSize: Buffer.byteLength(text), isTruncated: false } : null,
        isEmpty: false,
        defaultBranchRef: { target: { oid: "a".repeat(40) } },
        legacy0: null,
        legacy1: null,
      },
    }));
    event.githubClient.rest.repos = { getContent: vi.fn().mockRejectedValue(new Error("API rate limit exceeded for installation")) };
    expect(await main(event.options)).toEqual({ status: initialized ? "unbound" : "queue_missing" });
    expect(event.githubClient.graphql).toHaveBeenCalledOnce();
    expect(event.githubClient.graphql.mock.calls[0][0]).toContain("query CheckedWorkQueue");
    expect(event.githubClient.rest.repos.getContent).not.toHaveBeenCalled();
    expect(event.githubClient.rest.issues.get).not.toHaveBeenCalled();
    expect(event.options.authenticatePublisher).not.toHaveBeenCalled();
    expect(event.publishWorkQueueRequest).not.toHaveBeenCalled();
  });

  it("ignores non-closed issue events and is disabled without Issue mirroring", async () => {
    const event = fixture();
    event.options.context = { ...context, payload: { ...context.payload, action: "labeled" } };
    expect(await main(event.options)).toEqual({ status: "ignored" });
    expect(event.options.authenticatePublisher).not.toHaveBeenCalled();

    const disabled = fixture();
    disabled.options.issues = false;
    expect(await main(disabled.options)).toEqual({ status: "disabled" });
    expect(disabled.options.authenticatePublisher).not.toHaveBeenCalled();

    const comment = fixture();
    comment.options.context = { ...context, eventName: "issue_comment", payload: { ...context.payload, action: "created" } };
    expect(await main(comment.options)).toEqual({ status: "ignored" });
    expect(comment.options.authenticatePublisher).not.toHaveBeenCalled();
  });

  it("uses repository Issue settings from the event revision", async () => {
    const githubClient = {
      rest: {
        repos: {
          getContent: vi.fn(async () => ({
            data: { type: "file", encoding: "base64", size: 49, content: Buffer.from('{"work_queue":{"issues":{"label":"tasks"}}}').toString("base64") },
          })),
        },
      },
    };
    expect(await readIssuesConfig(githubClient, context)).toMatchObject({ label: "tasks" });
    expect(githubClient.rest.repos.getContent).toHaveBeenCalledWith({ ...context.repo, path: ".github/workflows/aw.json", ref: context.sha });
  });
  it("binds exact immutable identity rather than an Issue number or label", () => {
    const state = { works: new Map([["w", { ...work, backing_issue: { ...resource, resource_id: "35" } }]]) };
    expect(matchingWork(state, resource)).toBeUndefined();
    expect(matchingWork({ works: new Map([["w", { ...work, backing_issue: undefined, issue_link: resource }]]) }, resource)).toBeUndefined();
  });

  it("selects an exact Issue subject but never a dependency-only Issue", async () => {
    const subject = fixture({ ...work, backing_issue: undefined, subject: resource });
    expect((await main(subject.options)).status).toBe("cancelled");
    expect(subject.publishWorkQueueRequest.mock.calls[0][0].request.parameters.operations[0].reason).toBe("related_issue_closed");
    const dependency = fixture({ ...work, backing_issue: undefined, subject: undefined, dependencies: [resource] });
    expect((await main(dependency.options)).status).toBe("unbound");
    expect(dependency.publishWorkQueueRequest).not.toHaveBeenCalled();
  });

  it("refuses to pick one of several Work items with the same Issue subject", async () => {
    const ambiguous = fixture({ ...work, backing_issue: undefined, subject: resource });
    ambiguous.state.works.set("work_2", { ...work, work_id: "work_2", backing_issue: undefined, subject: resource });
    await expect(main(ambiguous.options)).rejects.toThrow("multiple Work items");
    expect(ambiguous.publishWorkQueueRequest).not.toHaveBeenCalled();
  });

  it("checks closed state, tracking and status labels against fresh Issue identity", async () => {
    const githubClient = {
      graphql: vi.fn(async () => ({
        i0: {
          issue: {
            databaseId: 34,
            number: 7,
            state: "CLOSED",
            repository: { databaseId: 12, nameWithOwner: "owner/repo" },
            labels: { nodes: [{ name: "work" }, { name: "work:running" }], pageInfo: { hasNextPage: false } },
          },
        },
      })),
    };
    expect(await closedIssue(githubClient, context, resource, { label: "work" })).toBe(true);
    githubClient.graphql.mockResolvedValueOnce({ i0: { issue: { databaseId: 35, number: 7, state: "CLOSED", repository: { databaseId: 12, nameWithOwner: "owner/repo" }, labels: { nodes: [], pageInfo: { hasNextPage: false } } } } });
    await expect(closedIssue(githubClient, context, resource, { label: "work" })).rejects.toThrow();
  });

  it.each([
    { state: "OPEN", labels: ["work", "work:queued"] },
    { state: "CLOSED", labels: ["tasks", "tasks:queued"] },
    { state: "CLOSED", labels: ["work"] },
  ])("does not authorize cancellation from event label hints when the fresh Issue is $state with $labels", async ({ state, labels }) => {
    const event = fixture();
    delete event.options.closedIssue;
    event.githubClient.graphql = vi.fn(async () => ({
      i0: { issue: { databaseId: 34, number: 7, state, repository: { databaseId: 12, nameWithOwner: "owner/repo" }, labels: { nodes: labels.map(name => ({ name })), pageInfo: { hasNextPage: false } } } },
    }));
    expect(await main(event.options)).toEqual({ status: "stale" });
    expect(event.githubClient.rest.issues.get).not.toHaveBeenCalled();
    expect(event.publishWorkQueueRequest).not.toHaveBeenCalled();
    expect(event.githubClient.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it.each([{ databaseId: 35 }, { number: 8 }, { repository: { databaseId: 13, nameWithOwner: "owner/repo" } }, { repository: { databaseId: 12, nameWithOwner: "other/repo" } }])(
    "rejects changed immutable Issue identity using the fresh preflight alone: %j",
    async changed => {
      const event = fixture();
      delete event.options.closedIssue;
      event.githubClient.graphql = vi.fn(async () => ({
        i0: {
          issue: { databaseId: 34, number: 7, state: "CLOSED", repository: { databaseId: 12, nameWithOwner: "owner/repo" }, labels: { nodes: [{ name: "work" }, { name: "work:queued" }], pageInfo: { hasNextPage: false } }, ...changed },
        },
      }));
      await expect(main(event.options)).rejects.toThrow("backing Issue was deleted, transferred, or changed identity");
      expect(event.githubClient.rest.issues.get).not.toHaveBeenCalled();
      expect(event.publishWorkQueueRequest).not.toHaveBeenCalled();
      expect(event.githubClient.rest.issues.createComment).not.toHaveBeenCalled();
    }
  );

  it("commits cancellation through the administrator and confirms before commenting", async () => {
    const { options, githubClient, publishWorkQueueRequest } = fixture();
    expect(await main(options)).toMatchObject({ status: "cancelled", publishedNow: true });
    expect(publishWorkQueueRequest).toHaveBeenCalledOnce();
    const args = publishWorkQueueRequest.mock.calls[0][0];
    expect(args.actor.role).toBe("administrator");
    expect(args.request.kind).toBe("cancel_work");
    expect(args.request.parameters.operations).toEqual([{ kind: "WorkCancellation", work_id: "work_1", reason: "related_issue_closed" }]);
    expect(githubClient.rest.issues.createComment).toHaveBeenCalledOnce();
    expect(githubClient.rest.issues.createComment.mock.calls[0][0].body).toContain("Cancellation was recorded in the work queue ledger. New state: **cancelled**. The work is now canceled.");
  });

  it("does not publish for a stale or unbound Issue", async () => {
    const stale = fixture();
    stale.options.closedIssue.mockResolvedValue(false);
    expect((await main(stale.options)).status).toBe("stale");
    expect(stale.publishWorkQueueRequest).not.toHaveBeenCalled();
    const unbound = fixture({ ...work, backing_issue: { ...resource, resource_id: "35" } });
    expect((await main(unbound.options)).status).toBe("unbound");
    expect(unbound.githubClient.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("does not comment on unresolved races or uncertain writes without a terminal commit", async () => {
    const race = fixture();
    race.options.closedIssue.mockResolvedValueOnce(true).mockResolvedValueOnce(false);
    expect((await main(race.options)).status).toBe("pending");
    expect(race.githubClient.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("comments on an earlier verified completed or cancelled outcome without claiming causation", async () => {
    const completed = fixture({ ...work, state: "completed", completion_id: "commit" });
    expect((await main(completed.options)).status).toBe("completed");
    expect(completed.publishWorkQueueRequest).not.toHaveBeenCalled();
    expect(completed.githubClient.rest.issues.createComment.mock.calls[0][0].body).toContain("New state: **completed**");
    const old = { ...work, state: "cancelled", cancellation_reason: "other_request", cancellation_commit_id: "commit" };
    const recovered = fixture(old);
    expect((await main(recovered.options)).status).toBe("cancelled");
    expect(recovered.publishWorkQueueRequest).not.toHaveBeenCalled();
    expect(recovered.githubClient.rest.issues.createComment).toHaveBeenCalledOnce();
    expect(recovered.githubClient.rest.issues.createComment.mock.calls[0][0].body).toContain("this Issue closure did not change it");
    const duplicate = fixture(old);
    duplicate.githubClient.rest.issues.listComments.mockResolvedValueOnce({ data: [{ body: recovered.githubClient.rest.issues.createComment.mock.calls[0][0].body }] });
    expect((await main(duplicate.options)).status).toBe("cancelled");
    expect(duplicate.githubClient.rest.issues.createComment).not.toHaveBeenCalled();
  });

  it("never renders an untrusted Work ID in the confirmation", async () => {
    const unsafe = fixture({ ...work, work_id: "`@example`" });
    expect((await main(unsafe.options)).status).toBe("cancelled");
    expect(unsafe.githubClient.rest.issues.createComment.mock.calls[0][0].body).not.toContain("`@example`");
  });

  it("checks an uncertain comment response rather than blindly retrying it", async () => {
    const uncertain = fixture({ ...work, state: "completed", completion_id: "commit" });
    uncertain.githubClient.rest.issues.createComment.mockRejectedValueOnce(new Error("response lost"));
    uncertain.githubClient.rest.issues.listComments
      .mockResolvedValueOnce({ data: [] })
      .mockResolvedValueOnce({ data: [{ body: "<!-- gh-aw-issue-closure:" + require("./work_queue_codec.cjs").digest({ resource, work_id: "work_1" }) + " -->" }] });
    expect((await main(uncertain.options)).status).toBe("completed");
    expect(uncertain.githubClient.rest.issues.createComment).toHaveBeenCalledOnce();
    const unresolved = fixture({ ...work, state: "completed", completion_id: "commit" });
    unresolved.githubClient.rest.issues.createComment.mockRejectedValueOnce(new Error("response lost"));
    await expect(main(unresolved.options)).rejects.toThrow("response lost");
    expect(unresolved.githubClient.rest.issues.createComment).toHaveBeenCalledOnce();
  });

  it("requires exact authenticated repository and event identity", async () => {
    const invalid = fixture();
    invalid.options.authenticatePublisher.mockResolvedValue({ ...trusted, native_run: { repository: { id: 13 } } });
    await expect(main(invalid.options)).rejects.toThrow("repository");
    expect(invalid.publishWorkQueueRequest).not.toHaveBeenCalled();
  });

  it("requires the authenticated primary maintenance workflow", async () => {
    const invalid = fixture();
    invalid.options.authenticatePublisher = vi.fn(async () => ({ ...trusted, workflow: ".github/workflows/agentics-maintenance.yml" }));
    await expect(main(invalid.options)).rejects.toThrow("authenticated Agentic Workqueue workflow");
    expect(invalid.publishWorkQueueRequest).not.toHaveBeenCalled();
  });

  it("scopes request and confirmation identities to Work when an Issue is reused", async () => {
    const first = fixture({ ...work, work_id: "first_work" });
    const second = fixture({ ...work, work_id: "second_work" });
    await main(first.options);
    await main(second.options);
    const firstRequest = first.publishWorkQueueRequest.mock.calls[0][0].request;
    const secondRequest = second.publishWorkQueueRequest.mock.calls[0][0].request;
    const firstComment = first.githubClient.rest.issues.createComment.mock.calls[0][0].body;
    const secondComment = second.githubClient.rest.issues.createComment.mock.calls[0][0].body;
    expect(firstRequest.id).not.toBe(secondRequest.id);
    expect(firstComment.match(/gh-aw-issue-closure:[^ ]+/)[0]).not.toBe(secondComment.match(/gh-aw-issue-closure:[^ ]+/)[0]);
  });
});
