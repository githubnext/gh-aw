import { describe, expect, it } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { validateWebhook } = require("./validate_webhook.cjs");
const { normalizeWebhookContext } = require("./webhook_context.cjs");
const { GITHUB_NOTIFICATION_EVENTS } = require("./github_notification_webhook.cjs");

function fixture(event = "issue_comment", action = GITHUB_NOTIFICATION_EVENTS[event].actions[0]) {
  const definition = GITHUB_NOTIFICATION_EVENTS[event];
  const id = definition.path === "commit" ? "a".repeat(40) : "42";
  const payload = { url: `https://github.com/Org/repo/${definition.path}/${id}${definition.anchor ? `#${definition.anchor}123` : ""}`, type: [action] };
  const subscription = {
    schema_version: 2,
    event,
    native_events: [definition.native_event],
    types: [...definition.actions],
    conditions: [],
    dispatch: { event_type: "notifications", payload_projection: { policy: "resource-url-types-filters/v2" } },
  };
  const request = { action: "notifications", client_payload: { schema_version: 2, aw_context: { webhook: { event, deliveries: [{ delivery_id: "notification", payload }] } } } };
  const env = { GITHUB_EVENT_NAME: "repository_dispatch" };
  return { payload, subscription, request, env, validate: () => validateWebhook(request, { ...env, GH_AW_WEBHOOK_SUBSCRIPTIONS: JSON.stringify([subscription]) }) };
}

describe("first-class GitHub gateway notifications", () => {
  it.each(Object.entries(GITHUB_NOTIFICATION_EVENTS).flatMap(([event, definition]) => definition.actions.map(action => [event, action])))("validates %s:%s without original bodies or command tokens", (event, action) => {
    const test = fixture(event, action);
    expect(test.validate().skipped).toBe(false);
    const normalized = normalizeWebhookContext(test.request.client_payload.aw_context);
    expect(normalized.items).toHaveLength(1);
    expect(normalized.items[0]).toMatchObject({ item_type: event, repo: "Org/repo", source: test.payload.url.split("#")[0] });
    expect(normalized.webhook.deliveries[0].payload).toEqual(test.payload);
  });

  it.each(Object.keys(GITHUB_NOTIFICATION_EVENTS))("bounds %s source scope and every selected activity", event => {
    const test = fixture(event);
    test.subscription.repos = ["org/*"];
    test.subscription.orgs = ["or*"];
    expect(test.validate().skipped).toBe(false);
    test.subscription.repos = ["other/*"];
    expect(test.validate).toThrow(/repository does not match/);
    test.subscription.repos = ["org/*"];
    test.subscription.orgs = ["other"];
    expect(test.validate).toThrow(/organization does not match/);
    test.subscription.orgs = ["org"];
    test.payload.type = ["not-an-action"];
    expect(test.validate).toThrow(/type/);
    test.payload.type = [...test.subscription.types, "not-an-action"];
    expect(test.validate).toThrow(/type/);
  });

  it("bounds projected selectors without disclosing whole labels or source bodies", () => {
    const test = fixture("pull_request", "closed");
    test.subscription.conditions = [
      { filter: "merged", path: ["pull_request", "merged"], operator: "equals", value_type: "boolean", values: [true] },
      { filter: "labels", path: ["pull_request", "labels", "*", "name"], operator: "equals", value_type: "string", values: ["ready"] },
    ];
    test.payload.filters = { merged: [true], labels: ["ready"] };
    expect(test.validate().skipped).toBe(false);
    test.payload.filters.labels = ["ready", "private-label"];
    expect(test.validate).toThrow(/declared matching/);
    test.payload.filters.labels = ["ready"];
    delete test.payload.filters.merged;
    expect(test.validate).toThrow(/declared matching/);
    test.payload.filters.merged = [true];
    for (const field of ["body", "issue", "pull_request", "comment", "sender", "arguments", "labels"]) {
      test.payload[field] = "private-data";
      expect(test.validate).toThrow(/accepts only/);
      expect(test.validate).not.toThrow(/private-data/);
      delete test.payload[field];
    }
  });

  it.each([
    ["issue_comment", "pull/42#issuecomment-123"],
    ["pull_request_comment", "issues/42#issuecomment-123"],
    ["pull_request_review", "pull/42#discussion_r123"],
    ["pull_request_review_comment", "pull/42#pullrequestreview-123"],
    ["discussion_comment", "discussions/42#issuecomment-123"],
    ["commit_comment", "commit/123#commitcomment-123"],
    ["issues", "issues/42#issuecomment-123"],
    ["pull_request_review_thread", "pull/42"],
  ])("rejects crossed or absent canonical anchors for %s", (event, path) => {
    const test = fixture(event);
    test.payload.url = `https://github.com/Org/repo/${path}`;
    expect(test.validate).toThrow(/canonical/);
  });

  it.each(Object.keys(GITHUB_NOTIFICATION_EVENTS))("binds %s HTML URLs to the approved origin and Enterprise prefix", event => {
    const test = fixture(event);
    test.payload.url = test.payload.url.replace("https://github.com", "https://github.example/base");
    expect(test.validate).toThrow(/canonical/);
    test.env.GITHUB_SERVER_URL = "https://github.example/base";
    expect(test.validate().skipped).toBe(false);
    test.payload.url = test.payload.url.replace("/base/", "/");
    expect(test.validate).toThrow(/canonical/);
  });

  it.each(Object.keys(GITHUB_NOTIFICATION_EVENTS))("rejects empty query or fragment delimiters for %s", event => {
    const test = fixture(event);
    const [parent, anchor] = test.payload.url.split("#");
    for (const url of [`${parent}?${anchor ? `#${anchor}` : ""}`, `${parent}#`]) {
      test.payload.url = url;
      expect(test.validate).toThrow(/canonical/);
    }
  });

  it("rejects malformed notification metadata and noncanonical URL forms", () => {
    for (const change of [
      { native_events: ["pull_request"] },
      { native_events: [] },
      { types: ["invalid"] },
      { types: ["created", "created"] },
      { provider: "github" },
      { command_filter: {} },
      { activity_normalization: {} },
      { any_of: [[]] },
    ]) {
      const test = fixture();
      Object.assign(test.subscription, change);
      expect(test.validate).toThrow(/valid version 2/);
    }
    for (const url of [
      "http://github.com/Org/repo/issues/42#issuecomment-123",
      "https://github.com/Org/repo/issues/042#issuecomment-123",
      "https://github.com/Org/repo/issues/42#issuecomment-0123",
      "https://github.com/Org/repo/issues/42?token=private#issuecomment-123",
      "https://user:private@github.com/Org/repo/issues/42#issuecomment-123",
      "https://other.example/Org/repo/issues/42#issuecomment-123",
    ]) {
      const test = fixture();
      test.payload.url = url;
      expect(test.validate).toThrow(/canonical/);
      expect(test.validate).not.toThrow(/private/);
    }
  });

  it("preserves all comment deliveries while deduplicating their parent reference", () => {
    const test = fixture();
    const webhook = test.request.client_payload.aw_context.webhook;
    webhook.deliveries.push({ delivery_id: "second", payload: { ...test.payload, url: test.payload.url.replace("123", "456") } });
    expect(test.validate().delivery_ids).toEqual(["notification", "second"]);
    const normalized = normalizeWebhookContext(test.request.client_payload.aw_context);
    expect(normalized.webhook.deliveries).toHaveLength(2);
    expect(normalized.items).toHaveLength(1);
  });
});
