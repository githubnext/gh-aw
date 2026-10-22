import { describe, expect, it } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { validateWebhook } = require("./validate_webhook.cjs");
const { normalizeWebhookContext } = require("./webhook_context.cjs");
const { COMMAND_SOURCE_ACTIONS } = require("./gateway_command_webhook.cjs");

function fixture(label = false, overrides = {}) {
  const event = label ? "label_command" : "slash_command";
  const subscription = {
    schema_version: 2,
    event,
    native_events: ["issues"],
    types: label ? ["labeled"] : ["opened", "edited", "reopened"],
    conditions: [],
    command_filter: { policy: label ? "label-command/v1" : "slash-command/v1", names: ["review"], events: ["issues"], strategy: "inline", ...(label ? { remove_label: true } : {}) },
    dispatch: { event_type: "commands", payload_projection: { policy: "resource-url-types-filters/v2" } },
    ...overrides,
  };
  const payload = { url: "https://github.com/Org/repo/issues/42", type: [label ? "labeled" : "opened"], filters: { event: ["issues"], [label ? "label" : "command"]: ["review"] } };
  const request = { action: "commands", client_payload: { schema_version: 2, aw_context: { webhook: { event, deliveries: [{ delivery_id: "command", payload }] } } } };
  return { subscription, payload, request, validate: () => validateWebhook(request, { GITHUB_EVENT_NAME: "repository_dispatch", GH_AW_WEBHOOK_SUBSCRIPTIONS: JSON.stringify([subscription]) }) };
}

describe("minimal gateway command notifications", () => {
  it.each([false, true])("validates command family label=%s without original bodies or write operations", label => {
    const test = fixture(label, { orgs: ["org"], repos: ["org/*"] });
    expect(test.validate().skipped).toBe(false);
    test.subscription.repos = ["other/*"];
    expect(test.validate).toThrow(/repository does not match/);
    delete test.subscription.repos;
    test.subscription.orgs = ["other"];
    expect(test.validate).toThrow(/organization does not match/);
  });

  it.each(Object.entries(COMMAND_SOURCE_ACTIONS))("binds %s source, action, and exact canonical comment identity", (event, actions) => {
    const locations = {
      issues: "issues/42",
      issue_comment: "issues/42#issuecomment-123",
      pull_request_comment: "pull/42#issuecomment-123",
      pull_request: "pull/42",
      pull_request_review_comment: "pull/42#discussion_r123",
      discussion: "discussions/42",
      discussion_comment: "discussions/42#discussioncomment-123",
    };
    const test = fixture();
    test.subscription.command_filter.events = [event];
    test.subscription.native_events = [event === "pull_request_comment" ? "issue_comment" : event];
    test.subscription.types = actions;
    test.payload.filters.event = [event];
    test.payload.url = `https://github.com/Org/repo/${locations[event]}`;
    for (const action of actions) {
      test.payload.type = [action];
      expect(test.validate().skipped).toBe(false);
    }
    test.payload.type = ["labeled"];
    expect(test.validate).toThrow(/type/);
    test.payload.type = [actions[0]];
    for (const url of [
      "https://evil.example/Org/repo/" + locations[event],
      test.payload.url + "?token=private",
      test.payload.url.split("#")[0] + "#unknown-123",
      test.payload.url.replace(/42/, "042"),
      test.payload.url.replace("https:", "http:"),
      test.payload.url.replace("github.com", "user:private@github.com"),
    ]) {
      const saved = test.payload.url;
      test.payload.url = url;
      expect(test.validate).toThrow(/canonical/);
      expect(test.validate).not.toThrow(/private/);
      test.payload.url = saved;
    }
  });

  it.each([
    [["review"], "review", true],
    [["review"], "reviewer", false],
    [["review"], "Review", false],
    [["fix*"], "fix", true],
    [["fix*"], "fix-build", true],
    [["*"], "any.command_1", true],
    [["*"], "review private arguments", false],
    [["*"], "/review", false],
    [["*"], "", false],
  ])("bounds projected command %s / %s", (names, command, accepted) => {
    const test = fixture();
    test.subscription.command_filter.names = names;
    test.payload.filters.command = [command];
    if (accepted) expect(test.validate().skipped).toBe(false);
    else expect(test.validate).toThrow(/declared matching/);
  });

  it("matches labels exactly, not with slash-command wildcard semantics", () => {
    const test = fixture(true);
    test.subscription.command_filter.names = ["deploy*"];
    test.payload.filters.label = ["deploy-prod"];
    expect(test.validate).toThrow(/declared matching/);
    test.payload.filters.label = ["deploy*"];
    expect(test.validate().skipped).toBe(false);
    test.subscription.command_filter.remove_label = false;
    expect(test.validate).toThrow(/valid version 2/);
  });

  it("does not import native bodyless activation bypasses into gateway admission", () => {
    const test = fixture();
    for (const action of ["ready_for_review", "review_requested", "submitted"]) {
      test.payload.type = [action];
      expect(test.validate).toThrow(/type/);
    }
    test.subscription.command_filter.events = ["pull_request_review"];
    expect(test.validate).toThrow(/valid version 2/);
  });

  it("binds command HTML URLs to the configured Enterprise origin and base path", () => {
    const test = fixture();
    test.payload.url = "https://github.example/base/Org/repo/issues/42";
    expect(test.validate).toThrow(/canonical/);
    const env = { GITHUB_EVENT_NAME: "repository_dispatch", GITHUB_SERVER_URL: "https://github.example/base", GH_AW_WEBHOOK_SUBSCRIPTIONS: JSON.stringify([test.subscription]) };
    expect(validateWebhook(test.request, env).skipped).toBe(false);
    test.payload.url = "https://github.example/Org/repo/issues/42";
    expect(() => validateWebhook(test.request, env)).toThrow(/canonical/);
  });

  it("rejects malformed filtering metadata explicitly", () => {
    for (const change of [{ policy: "slash-command/v2" }, { names: [] }, { names: [null] }, { events: ["push"] }, { events: null }, { strategy: "decentralized" }, { remove_label: true }, { extra: true }]) {
      const test = fixture();
      Object.assign(test.subscription.command_filter, change);
      expect(test.validate).toThrow(/valid version 2/);
    }
    for (const override of [{ conditions: [{ filter: "body" }] }, { native_events: [] }, { types: ["labeled"] }, { provider: "github" }, { any_of: [[]] }, { activity_normalization: {} }]) {
      expect(fixture(false, override).validate).toThrow(/valid version 2/);
    }
    const label = fixture(true);
    delete label.subscription.command_filter.remove_label;
    expect(label.validate).toThrow(/valid version 2/);
  });

  it("rejects undeclared or multi-valued projections and all resource details", () => {
    for (const filters of [
      { event: ["issues"] },
      { event: ["issues"], command: ["review", "other"] },
      { event: ["issues", "pull_request"], command: ["review"] },
      { event: ["push"], command: ["review"] },
      { event: ["issues"], command: ["review"], arguments: ["private"] },
    ]) {
      const test = fixture();
      test.payload.filters = filters;
      expect(test.validate).toThrow(/declared matching|canonical/);
    }
    for (const field of ["body", "arguments", "sender", "labels", "issue", "comment"]) {
      const test = fixture();
      test.payload[field] = "private";
      expect(test.validate).toThrow(/accepts only/);
      expect(test.validate).not.toThrow(/private/);
    }
  });

  it("preserves distinct comment deliveries while deduplicating parent items without issue/discussion collisions", () => {
    const deliveries = [
      ["issue_comment", "issues/42#issuecomment-1"],
      ["issue_comment", "issues/42#issuecomment-2"],
      ["discussion_comment", "discussions/42#discussioncomment-3"],
    ].map(([event, path], index) => ({ delivery_id: `command-${index}`, payload: { url: `https://github.com/org/repo/${path}`, type: ["created"], filters: { event: [event], command: ["review"] } } }));
    const normalized = normalizeWebhookContext({ webhook: { event: "slash_command", deliveries } });
    expect(normalized.webhook.deliveries).toHaveLength(3);
    expect(normalized.items).toHaveLength(2);
    expect(normalized.items.map(item => item.source)).toEqual(["https://github.com/org/repo/issues/42", "https://github.com/org/repo/discussions/42"]);
  });
});
