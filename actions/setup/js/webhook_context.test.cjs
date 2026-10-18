import { describe, it, expect, afterEach, vi } from "vitest";
import fs from "node:fs";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { normalizeWebhookContext, redactWebhookPayloads } = require("./webhook_context.cjs");
const { buildAwContext } = require("./aw_context.cjs");

function project(deliveryID, id, action = "edited") {
  return { delivery_id: deliveryID, payload: { url: `https://api.github.com/orgs/org/projectsV2/1/items/${id}`, type: [action] } };
}

function contextFor(event, deliveries) {
  return { webhook: { event, deliveries } };
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

describe("normalizeWebhookContext", () => {
  it.each([undefined, null, {}, { items: [] }, { actor: "native-user" }])("leaves non-webhook context alone: %j", value => {
    expect(normalizeWebhookContext(value)).toBeNull();
  });

  it("deduplicates deliveries and items independently without mutating or choosing a scalar target", () => {
    const raw = contextFor("projects_v2_item", [project("a", 1), project("b", 2), project("c", 1, "created"), project("a", 1)]);
    raw.items = [{ item_type: "forged", item_number: "99" }];
    const before = JSON.stringify(raw);
    const result = normalizeWebhookContext(raw);
    expect(result.webhook.deliveries.map(value => value.delivery_id)).toEqual(["a", "b", "c"]);
    expect(result.items).toEqual([
      { item_type: "projects_v2_item", item_number: "1", source: "https://api.github.com/orgs/org/projectsV2/1", repo: "", url: project("a", 1).payload.url },
      { item_type: "projects_v2_item", item_number: "2", source: "https://api.github.com/orgs/org/projectsV2/1", repo: "", url: project("b", 2).payload.url },
    ]);
    expect(result).not.toHaveProperty("item_number");
    expect(JSON.stringify(raw)).toBe(before);
    expect(normalizeWebhookContext(result)).toEqual(result);
  });

  it.each(["code_scanning_alert", "dependabot_alert"])("scopes %s identities to the source repository", event => {
    const source = event === "dependabot_alert" ? "dependabot" : "code-scanning";
    const alert = (deliveryID, repo, number) => ({ delivery_id: deliveryID, payload: { url: `https://api.github.com/repos/${repo}/${source}/alerts/${number}`, type: ["created"] } });
    const result = normalizeWebhookContext(contextFor(event, [alert("a", "org/one", 1), alert("b", "org/two", 1), alert("c", "ORG/ONE", 1)]));
    expect(result.items).toEqual([
      { item_type: event, item_number: "1", repo: "org/one", url: alert("a", "org/one", 1).payload.url },
      { item_type: event, item_number: "1", repo: "org/two", url: alert("b", "org/two", 1).payload.url },
    ]);
    expect(result.webhook.deliveries).toHaveLength(3);
  });

  it.each(["undefined", "null", "0", "-1", "1.5", "", "01"])("rejects invalid security alert URL identity %j", number => {
    const raw = contextFor("dependabot_alert", [{ delivery_id: "a", payload: { url: `https://api.github.com/repos/org/repo/dependabot/alerts/${number}`, type: ["created"] } }]);
    expect(() => normalizeWebhookContext(raw)).toThrow(/canonical GitHub API alert URL/);
  });

  it.each([
    "http://api.github.com/repos/org/repo/code-scanning/alerts/1",
    "https://api.github.com/repos/org/repo/dependabot/alerts/1",
    "https://other.example/repos/org/repo/code-scanning/alerts/1",
    "https://api.github.com/repos/org/repo/code-scanning/alerts/1?token=private",
    "https://api.github.com/repos/org/repo/code-scanning/alerts/1#fragment",
    "https://user:password@api.github.com/repos/org/repo/code-scanning/alerts/1",
    "https://api.github.com/repos/org/repo/other/code-scanning/alerts/1",
    "https://api.github.com/repos/org%2Frepo/repo/code-scanning/alerts/1",
  ])("rejects noncanonical, cross-kind or cross-host security alert URL %#", url => {
    const raw = contextFor("code_scanning_alert", [{ delivery_id: "a", payload: { url, type: ["created"] } }]);
    expect(() => normalizeWebhookContext(raw)).toThrow(/canonical GitHub API alert URL/);
  });

  it("reports invalid API configuration without including its value", () => {
    const raw = contextFor("dependabot_alert", [{ delivery_id: "a", payload: { url: "https://api.github.com/repos/o/r/dependabot/alerts/1", type: ["created"] } }]);
    expect(() => normalizeWebhookContext(raw, { GITHUB_API_URL: "private-invalid-value" })).toThrow("GITHUB_API_URL must be a valid URL");
    expect(() => normalizeWebhookContext(raw, { GITHUB_API_URL: "private-invalid-value" })).not.toThrow(/private-invalid-value/);
  });

  it("fails the whole batch on a conflicting duplicate, without including its secret payload in the error", () => {
    const first = project("same", 1);
    const conflicting = project("same", 2);
    conflicting.payload.private = "do-not-log-this";
    expect(() => normalizeWebhookContext(contextFor("projects_v2_item", [first, conflicting]))).toThrow("duplicate webhook delivery_id has conflicting payloads");
    expect(() => normalizeWebhookContext(contextFor("projects_v2_item", [first, conflicting]))).not.toThrow(/do-not-log-this/);
  });
});

describe("buildAwContext webhook batch propagation", () => {
  it("preserves unique elements across repository dispatch, workflow dispatch, and workflow call hops", () => {
    vi.spyOn(fs, "readFileSync").mockImplementation(() => {
      throw Object.assign(new Error("missing"), { code: "ENOENT" });
    });

    vi.stubEnv("GITHUB_RUN_ATTEMPT", "1");
    const inbound = contextFor("projects_v2_item", [project("a", 1), project("b", 2), project("c", 1), project("a", 1)]);
    vi.stubGlobal("context", { repo: { owner: "org", repo: "destination" }, eventName: "repository_dispatch", actor: "cao[bot]", runId: 1, payload: { client_payload: { schema_version: 2, aw_context: inbound } } });
    const first = buildAwContext();
    expect(first.actor).toBe("cao[bot]");
    expect(first.event_type).toBe("repository_dispatch");
    expect(first.items).toHaveLength(2);
    expect(first.webhook.deliveries).toHaveLength(3);
    expect(first.item_type).toBe("");
    expect(first.item_number).toBe("");
    for (const eventName of ["workflow_dispatch", "workflow_call"]) {
      vi.stubGlobal("context", { repo: { owner: "org", repo: "child" }, eventName, actor: "github-actions[bot]", runId: 2, payload: { inputs: { aw_context: JSON.stringify(first) } } });
      const child = buildAwContext();
      expect(child.webhook).toEqual(first.webhook);
      expect(child.items).toEqual(first.items);
      expect(child.actor).toBe("github-actions[bot]");
      expect(child.item_number).toBe("");
    }
  });
});

describe("redactWebhookPayloads", () => {
  it.each([false, true])("omits raw payloads without modifying %s serialized context", serialized => {
    const raw = normalizeWebhookContext(contextFor("projects_v2_item", [project("a", 1)]));
    const input = serialized ? JSON.stringify(raw) : raw;
    const safe = redactWebhookPayloads(input);
    expect(typeof safe).toBe(typeof input);
    expect(JSON.stringify(safe)).not.toContain("payload");
    const decoded = serialized ? JSON.parse(safe) : safe;
    expect(decoded).toEqual({ ...raw, webhook: { event: raw.webhook.event, deliveries: [{ delivery_id: "a" }] } });
    expect(redactWebhookPayloads(safe)).toEqual(safe);
    expect(raw.webhook.deliveries[0].payload).toEqual(project("a", 1).payload);
  });

  it.each([null, {}, { repo: "org/native" }, '{"repo":"org/native"}', "not-json"])("preserves native context %#", value => {
    expect(redactWebhookPayloads(value)).toBe(value);
  });

  it("tolerates malformed unvalidated log context", () => {
    expect(() => redactWebhookPayloads({ webhook: { deliveries: [null, {}] } })).not.toThrow();
  });
});
