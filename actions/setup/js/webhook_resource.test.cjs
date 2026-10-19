import { afterEach, describe, expect, it, vi } from "vitest";
import { createRequire } from "node:module";
import { readFileSync } from "node:fs";

const require = createRequire(import.meta.url);
const { WEBHOOK_PAYLOAD_POLICY, WEBHOOK_TYPES, resolveWebhookResource } = require("./webhook_resource.cjs");
const { normalizeWebhookContext } = require("./webhook_context.cjs");
const { validateWebhook } = require("./validate_webhook.cjs");

const examples = [
  ["projects_v2_item", "https://api.github.com/orgs/org/projectsV2/1/items/123", ["edited"], "123", "https://api.github.com/orgs/org/projectsV2/1"],
  ["code_scanning_alert", "https://api.github.com/repos/org/repo/code-scanning/alerts/123", ["created"], "123", undefined],
  ["dependabot_alert", "https://api.github.com/repos/org/repo/dependabot/alerts/123", ["created"], "123", undefined],
  ["jira_issue", "https://tenant.atlassian.net/rest/api/3/issue/123", ["edited", "closed", "labeled"], "123", "https://tenant.atlassian.net"],
  ["linear_issue", "https://linear.app/workspace/issue/APP-123/example-title", ["edited", "closed"], "APP-123", "https://linear.app/workspace"],
  ["azure_devops_issue", "https://dev.azure.com/organization/project/_apis/wit/workItems/123", ["edited", "closed"], "123", "https://dev.azure.com/organization"],
];

afterEach(() => vi.unstubAllGlobals());

describe("minimal resource notification contract", () => {
  it("explicitly versions the projection independently of the batch envelope", () => {
    expect(WEBHOOK_PAYLOAD_POLICY).toBe("resource-url-types-filters/v2");
  });

  const githubCatalog = JSON.parse(readFileSync(new URL("../../../pkg/workflow/data/webhook_events.json", import.meta.url), "utf8"));
  const providerCatalog = JSON.parse(readFileSync(new URL("../../../pkg/workflow/data/provider_issue_webhooks.json", import.meta.url), "utf8"));
  it.each(Object.entries(WEBHOOK_TYPES))("keeps %s activities identical to the compiler catalog", (event, types) => {
    expect([...types].sort()).toEqual([...(githubCatalog[event] ?? providerCatalog[event]).actions].sort());
  });

  it("validates every complete specification payload against the actual runtime", () => {
    const specification = readFileSync(new URL("../../../docs/src/content/docs/specs/webhook-gateway-specification.md", import.meta.url), "utf8");
    const payloads = [...specification.matchAll(/```json title="([^"]+)\.payload\.json"\n([\s\S]*?)\n```/g)];
    expect(payloads.map(match => match[1]).sort()).toEqual([...Object.keys(WEBHOOK_TYPES), "webhook"].sort());
    for (const [, event, json] of payloads) {
      const payload = JSON.parse(json);
      expect(Object.keys(payload).sort()).toEqual(payload.filters ? ["filters", "type", "url"] : ["type", "url"]);
      expect(normalizeWebhookContext({ webhook: { event, deliveries: [{ delivery_id: "example", payload }] } }).items).toHaveLength(1);
      const definition = githubCatalog[event] ?? providerCatalog[event];
      const subscription = {
        schema_version: 2,
        event,
        types: payload.type,
        conditions: Object.entries(payload.filters ?? {})
          .filter(([filter]) => !(event === "azure_devops_issue" && filter === "closed-states"))
          .map(([filter, values]) => ({
            filter,
            path: event === "webhook" ? filter.split(".") : definition.filters[filter].path,
            operator: "equals",
            value_type: event === "webhook" ? typeof values[0] : definition.filters[filter].type,
            values,
          })),
        dispatch: { event_type: "specification", payload_projection: { policy: WEBHOOK_PAYLOAD_POLICY } },
        ...(definition?.provider
          ? {
              provider: definition.provider,
              native_events: [...new Set(Object.values(definition.native_events))],
              activity_normalization: { policy: `${definition.provider}-issue/v1`, ...(event === "azure_devops_issue" ? { closed_states: ["Done"] } : {}) },
            }
          : {}),
      };
      expect(
        validateWebhook(
          { action: "specification", client_payload: { schema_version: 2, aw_context: { webhook: { event, deliveries: [{ delivery_id: "example", payload }] } } } },
          { GITHUB_EVENT_NAME: "repository_dispatch", GH_AW_WEBHOOK_SUBSCRIPTIONS: JSON.stringify([subscription]) }
        ).skipped
      ).toBe(false);
    }
  });

  it("derives generic references only from canonical URLs without guessing issue identities", () => {
    const url = "https://service.example/releases/42";
    expect(resolveWebhookResource("webhook", { url, type: ["com.example.release"] })).toEqual({
      item_type: "webhook",
      item_number: "",
      repo: "",
      source: url,
      url,
    });
    const result = normalizeWebhookContext({
      webhook: {
        event: "webhook",
        deliveries: [url, url, `${url}/other`].map((url, index) => ({
          delivery_id: `generic-${index}`,
          payload: { url, type: ["com.example.release"] },
        })),
      },
    });
    expect(result.items).toHaveLength(2);
    expect(result.webhook.deliveries).toHaveLength(3);
    expect(() => resolveWebhookResource("webhook", { url, type: ["com.example.release"], state: "private" })).toThrow(/accepts only/);
    for (const invalid of ["http://service.example/releases/42", `${url}?token=private`, `${url}#fragment`, "https://user:private@service.example/releases/42"]) {
      expect(() => resolveWebhookResource("webhook", { url: invalid, type: ["com.example.release"] })).toThrow(/canonical/);
    }
  });

  it("requires homogeneous literal generic event identifiers, not activity/body side channels", () => {
    const url = "https://service.example/releases/42";
    for (const type of [[], [""], ["*"], ["event type"], ["${{ github.actor }}"], ["event", "other"], "event", [false], ["a".repeat(101)]]) {
      expect(() => resolveWebhookResource("webhook", { url, type })).toThrow(/type/);
    }
    expect(() =>
      normalizeWebhookContext({
        webhook: {
          event: "webhook",
          deliveries: ["event", "other"].map(type => ({
            delivery_id: type,
            payload: { url, type: [type] },
          })),
        },
      })
    ).toThrow(/one event identifier/);
  });

  it.each(examples)("derives %s identity from only URL and activities, without network access", (event, url, type, number, source) => {
    const fetch = vi.fn(() => {
      throw new Error("must not fetch");
    });
    vi.stubGlobal("fetch", fetch);
    const payload = { url, type };
    const before = JSON.stringify(payload);
    const item = resolveWebhookResource(event, payload);
    expect(item).toMatchObject({ item_type: event, item_number: number, url });
    expect(item.source).toBe(source);
    expect(JSON.stringify(payload)).toBe(before);
    expect(fetch).not.toHaveBeenCalled();
  });

  it.each(examples)("rejects resource details in %s instead of silently stripping them", (event, url, type) => {
    for (const key of ["title", "body", "description", "labels", "sender", "actor", "state", "issue", "data", "resource", "action", "organizationId", "project_id"]) {
      const payload = { url, type, [key]: "private-content-must-not-be-logged" };
      expect(() => resolveWebhookResource(event, payload)).toThrow(/accepts only/);
      expect(() => resolveWebhookResource(event, payload)).not.toThrow(/private-content-must-not-be-logged/);
    }
  });

  it.each(examples)("preserves only scalar filter projections without changing %s resource identity", (event, url, type) => {
    const filters = { state: ["ready"], approved: [false], attempt: [1] };
    const payload = { url, type, filters };
    expect(resolveWebhookResource(event, payload)).toEqual(resolveWebhookResource(event, { url, type }));
    expect(normalizeWebhookContext({ webhook: { event, deliveries: [{ delivery_id: "filtered", payload }] } }).webhook.deliveries[0].payload.filters).toEqual(filters);
    for (const invalid of [null, [], {}, { state: "ready" }, { state: ["ready", null] }, { state: [{ body: "private" }] }, { state: ["ready", true] }, { count: [Infinity] }, JSON.parse('{"__proto__":["private"]}')]) {
      expect(() => resolveWebhookResource(event, { url, type, filters: invalid })).toThrow(/filters/);
    }
  });
  it.each(examples)("requires known distinct nonempty activities for %s", (event, url) => {
    for (const type of [undefined, null, "", "edited", [], [""], [{}], ["not-a-type"], ["closed", "closed"], ["created", "private-description"]]) {
      expect(() => resolveWebhookResource(event, { url, type })).toThrow(/type/);
    }
  });

  it.each(examples)("rejects noncanonical, credentialed, queried and fragmented %s URLs", (event, url, type) => {
    const parsed = new URL(url);
    for (const invalid of [undefined, "", {}, url.replace("https:", "http:"), `${url}?token=private`, `${url}#details`, `${url}/`, `${parsed.origin}/other/resource/123`, url.replace("://", "://user:private@")]) {
      expect(() => resolveWebhookResource(event, { url: invalid, type })).toThrow(/canonical/);
      expect(() => resolveWebhookResource(event, { url: invalid, type })).not.toThrow(/private/);
    }
  });

  it("accepts user-owned and Enterprise Projects API URLs, including IDs beyond JavaScript integer precision", () => {
    const url = "https://github.example.com/api/v3/users/alice/projectsV2/7/items/9007199254740993";
    expect(resolveWebhookResource("projects_v2_item", { url, type: ["deleted"] }, { GITHUB_API_URL: "https://github.example.com/api/v3" })).toMatchObject({
      item_number: "9007199254740993",
      source: "https://github.example.com/api/v3/users/alice/projectsV2/7",
    });
  });

  it.each([
    "https://linear.app/issue/APP-123",
    "https://other.example/workspace/issue/APP-123",
    "https://linear.app/workspace/project/APP-123",
    "https://linear.app/workspace/issue/APP-0",
    "https://linear.app/workspace/issue/APP-123#comment",
  ])("rejects ambiguous or non-issue Linear URL %#", url => {
    expect(() => resolveWebhookResource("linear_issue", { url, type: ["opened"] })).toThrow(/canonical/);
  });

  it("derives legacy Azure collection identities without treating project or update IDs as resource identity", () => {
    const url = "https://organization.visualstudio.com/DefaultCollection/project/_apis/wit/workItems/123";
    expect(resolveWebhookResource("azure_devops_issue", { url, type: ["restored"] })).toMatchObject({ item_number: "123", source: "https://organization.visualstudio.com/DefaultCollection" });
    expect(() => resolveWebhookResource("azure_devops_issue", { url: `${url}/updates/5`, type: ["edited"] })).toThrow(/canonical/);
  });

  it.each([
    ["linear_issue", "https://linear.app/workspace/issue/APP-123/%E6%BC%A2%E5%AD%97", "APP-123", "https://linear.app/workspace"],
    ["azure_devops_issue", "https://dev.azure.com/organization/My%20Project/_apis/wit/workItems/123", "123", "https://dev.azure.com/organization"],
  ])("accepts canonical percent-encoded %s URL segments without changing identity", (event, url, number, source) => {
    expect(resolveWebhookResource(event, { url, type: ["edited"] })).toMatchObject({ item_number: number, source, url });
  });

  it.each([
    ["linear_issue", "https://linear.app/workspace/issue/APP-123/title%"],
    ["linear_issue", "https://linear.app/workspace/issue/APP-123/title%GG"],
    ["azure_devops_issue", "https://dev.azure.com/organization/My%Project/_apis/wit/workItems/123"],
  ])("rejects malformed escapes in %s URL segments", (event, url) => {
    expect(() => resolveWebhookResource(event, { url, type: ["edited"] })).toThrow(/canonical/);
  });

  it("keeps Projects identities scoped to their project URL", () => {
    const result = normalizeWebhookContext({
      webhook: {
        event: "projects_v2_item",
        deliveries: ["1", "2"].map(project => ({ delivery_id: project, payload: { url: `https://api.github.com/orgs/org/projectsV2/${project}/items/123`, type: ["edited"] } })),
      },
    });
    expect(result.items).toHaveLength(2);
    expect(result.items[0]).not.toHaveProperty("project_id");
  });

  it.each([[["projects_v2_item"]], [{ event: "projects_v2_item" }], ["constructor"], ["__proto__"], ["unsupported_issue"]])("rejects non-string and unsupported event families %#", event => {
    expect(() => normalizeWebhookContext({ webhook: { event, deliveries: [{ delivery_id: "d", payload: { url: examples[0][1], type: ["edited"] } }] } })).toThrow(/supported webhook event/);
  });

  it("does not split Linear identities when only a readable URL slug changes", () => {
    const result = normalizeWebhookContext({
      webhook: {
        event: "linear_issue",
        deliveries: ["before", "after"].map(slug => ({ delivery_id: slug, payload: { url: `https://linear.app/workspace/issue/APP-123/${slug}`, type: ["edited"] } })),
      },
    });
    expect(result.items).toHaveLength(1);
    expect(result.webhook.deliveries).toHaveLength(2);
  });

  it("deduplicates Azure collection-scoped work items across optional project URL segments", () => {
    const result = normalizeWebhookContext({
      webhook: {
        event: "azure_devops_issue",
        deliveries: ["", "project/"].map((project, index) => ({ delivery_id: String(index), payload: { url: `https://dev.azure.com/organization/${project}_apis/wit/workItems/123`, type: ["edited"] } })),
      },
    });
    expect(result.items).toHaveLength(1);
    expect(result.webhook.deliveries).toHaveLength(2);
  });
});
