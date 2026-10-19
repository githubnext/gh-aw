import test from "node:test";
import assert from "node:assert/strict";
import { condition, configuration, harness, providerSource, subscription, clock, issue } from "./helpers.mjs";
import { project } from "../src/matching.mjs";

function jira(webhookEvent = "jira:issue_updated") {
  return {
    webhookEvent,
    issue: { id: "10001", key: "APP-42", self: "https://example.atlassian.net/rest/api/3/issue/10001", fields: { project: { key: "APP" }, labels: ["ready"], summary: "private title" } },
    changelog: {
      items: [
        { fieldtype: "jira", field: "resolution", from: null, to: "1" },
        { fieldtype: "jira", field: "labels", fromString: null, toString: "ready" },
      ],
    },
  };
}
function linear(action = "update") {
  return {
    action,
    type: "Issue",
    organizationId: "org-id",
    webhookId: "configured-webhook",
    webhookTimestamp: clock,
    createdAt: "2026-10-19T12:00:00.000Z",
    url: "https://linear.app/workspace/issue/APP-42/readable-slug",
    data: { id: "immutable-issue-id", identifier: "APP-42", teamId: "team-id", completedAt: "2026-10-19T12:00:00.000Z", canceledAt: null, labelIds: ["new", "kept"], description: "private description" },
    updatedFrom: { completedAt: null, labelIds: ["old", "kept"] },
  };
}
function azure(eventType = "workitem.updated") {
  return {
    id: "original-event-uuid",
    eventType,
    resourceContainers: { collection: { id: "collection-id" }, project: { id: "project-id" } },
    resource: {
      id: 999,
      workItemId: 42,
      revision: { id: 42 },
      url: "https://dev.azure.com/company/_apis/wit/workItems/42/updates/999",
      fields: {
        "System.State": { oldValue: "Active", newValue: "Done" },
        "System.Tags": { oldValue: "old; kept", newValue: " new;kept;new " },
      },
    },
  };
}

test("Jira signed original bytes normalize and intersect activities without source bodies", async t => {
  const sub = subscription("jira_issue", ["closed"], [condition("jira_issue", "project", ["APP"])]);
  const config = configuration([sub]);
  providerSource(config, "jira");
  const h = await harness(t, config);
  assert.equal((await h.send(jira())).status, 200);
  await h.drainOne();
  const payload = h.calls.find(call => call.path.endsWith("/dispatches")).body.client_payload.aw_context.webhook.deliveries[0].payload;
  assert.deepEqual(payload, { url: "https://example.atlassian.net/rest/api/3/issue/10001", type: ["closed"], filters: { project: ["APP"] } });
  assert.equal(h.store.status()[0].state, "dispatched");
  const statusOnly = jira();
  statusOnly.changelog.items = [{ fieldtype: "jira", field: "status", fromString: "Active", toString: "Done" }];
  assert.deepEqual(await (await h.send(statusOnly, { delivery: "other" })).json(), { duplicate: false, admitted: 0 });
  const wrong = jira();
  wrong.issue.self = "https://example.atlassian.net/rest/api/3/issue/10002";
  assert.equal((await h.send(wrong, { delivery: "wrong" })).status, 400);
  wrong.issue.fields.project.key = "UNAPPROVED";
  assert.equal((await h.send(wrong, { delivery: "unapproved" })).status, 403);
  for (const suffix of ["?", "#", "?#"]) {
    const delimiter = jira();
    delimiter.issue.self += suffix;
    assert.equal((await h.send(delimiter, { delivery: `delimiter-${suffix}` })).status, 400);
  }
});

test("provider evidence remains conservative for ambiguous Jira and Linear changes", () => {
  const jiraSub = subscription("jira_issue", ["edited", "closed", "reopened", "labeled", "unlabeled"], [condition("jira_issue", "project", ["APP"])]);
  const config = configuration();
  providerSource(config, "jira");
  const payload = jira();
  assert.deepEqual(project(jiraSub, config.sources[0], payload, undefined, config.github).type, ["closed", "edited", "labeled"]);
  payload.issue.fields.labels = ["ready", "also"];
  assert.deepEqual(project(jiraSub, config.sources[0], payload, undefined, config.github).type, ["closed", "edited"]);
  payload.changelog.items.push({ fieldtype: "jira", field: "resolution", from: "1", to: null });
  assert.throws(() => project(jiraSub, config.sources[0], payload, undefined, config.github), /contradictory-lifecycle-evidence/);
  providerSource(config, "linear");
  const linearSub = subscription("linear_issue", ["edited", "closed", "reopened", "labeled", "unlabeled"], [condition("linear_issue", "organization-id", ["org-id"])]);
  const p = linear();
  assert.deepEqual(project(linearSub, config.sources[0], p, undefined, config.github).type, ["closed", "edited", "labeled", "unlabeled"]);
  delete p.data.canceledAt;
  assert.deepEqual(project(linearSub, config.sources[0], p, undefined, config.github).type, ["edited", "labeled", "unlabeled"]);
  p.updatedFrom = { completedAt: p.data.completedAt, canceledAt: null };
  p.data.completedAt = null;
  p.data.canceledAt = null;
  assert.deepEqual(project(linearSub, config.sources[0], p, undefined, config.github).type, ["edited", "reopened"]);
  p.updatedFrom = { completedAt: "2026-10-19T11:00:00.000Z", canceledAt: null };
  p.data.canceledAt = "2026-10-19T12:00:00.000Z";
  assert.deepEqual(project(linearSub, config.sources[0], p, undefined, config.github).type, ["edited"], "terminal-to-terminal does not close or reopen");
});

test("Linear signature, timestamp, entity, URL binding and stable retry identity", async t => {
  const sub = subscription("linear_issue", ["closed", "labeled", "unlabeled"], [condition("linear_issue", "organization-id", ["org-id"]), condition("linear_issue", "team-id", ["team-id"])]);
  const config = configuration([sub]);
  providerSource(config, "linear");
  const h = await harness(t, config);
  const p = linear();
  assert.equal((await h.send(p)).status, 200);
  const retry = structuredClone(p);
  retry.webhookTimestamp += 1000;
  assert.deepEqual(await (await h.send(retry)).json(), { duplicate: true, admitted: 0 });
  await h.drainOne();
  assert.equal(h.store.status()[0].state, "dispatched");
  const forwarded = h.calls.find(call => call.path.endsWith("/dispatches")).body.client_payload.aw_context.webhook.deliveries[0].payload;
  assert.deepEqual(forwarded.type, ["closed", "labeled", "unlabeled"]);
  assert.ok(!JSON.stringify(forwarded).includes("immutable-issue-id"));
  p.webhookTimestamp -= 61000;
  assert.equal((await h.send(p)).status, 401);
  p.webhookTimestamp = clock;
  p.url = "https://linear.app/issue/APP-42";
  assert.equal((await h.send(p)).status, 400);
  p.url = "https://linear.app/workspace/issue/APP-43";
  assert.equal((await h.send(p)).status, 400);
  p.type = "IssueLabel";
  assert.deepEqual(await (await h.send(p)).json(), { duplicate: false, admitted: 0 });
  p.type = "Issue";
  p.organizationId = "unapproved";
  assert.equal((await h.send(p)).status, 403);
  p.organizationId = "org-id";
  p.url = "https://linear.app/workspace/issue/APP-42";
  p.data.url = "https://linear.app/workspace/issue/APP-43";
  assert.equal((await h.send(p)).status, 400, "contradictory delivered links cannot be ignored");
  p.data.url = p.url;
  delete p.url;
  assert.equal((await h.send(p)).status, 200, "data.url fallback retains identical identity binding");
});

test("Azure configured Basic authentication and complete state/tag deltas project only matching terminal", async t => {
  const sub = subscription("azure_devops_issue", ["closed", "labeled", "unlabeled"], [condition("azure_devops_issue", "project_id", ["project-id"])]);
  sub.activity_normalization.closed_states = ["Done", "Closed"];
  const config = configuration([sub]);
  providerSource(config, "azure-devops");
  const h = await harness(t, config, { env: { SOURCE_SIGNING_SECRET: "synthetic-user:synthetic-password" } });
  assert.equal((await h.send(azure())).status, 200);
  await h.drainOne();
  const forwarded = h.calls.find(call => call.path.endsWith("/dispatches")).body.client_payload.aw_context.webhook.deliveries[0].payload;
  assert.deepEqual(forwarded, { url: "https://dev.azure.com/company/_apis/wit/workItems/42", type: ["closed", "labeled", "unlabeled"], filters: { project_id: ["project-id"], "closed-states": ["Done"] } });
  const wrongId = azure();
  delete wrongId.resource.workItemId;
  assert.equal((await h.send(wrongId)).status, 400, "must not accept the update record's id");
  const wrongScope = azure();
  wrongScope.resourceContainers.collection.id = "other-collection";
  assert.equal((await h.send(wrongScope)).status, 403);
  assert.equal((await h.send(azure(), { signatureSecret: "wrong:credential" })).status, 401);
  const contradictory = azure();
  contradictory.resource.revision.id = 43;
  assert.equal((await h.send(contradictory)).status, 400);
  contradictory.resource.revision.id = 42;
  contradictory.resource._links = { workItem: { href: "https://dev.azure.com/company/_apis/wit/workItems/43" } };
  assert.equal((await h.send(contradictory)).status, 400);
  contradictory.resource._links.workItem.href = "https://dev.azure.com/another/_apis/wit/workItems/42";
  assert.equal((await h.send(contradictory)).status, 400);
});

test("Azure snapshots, reopening, terminal-to-terminal and nonterminal edited projection", () => {
  const sub = subscription("azure_devops_issue", ["edited", "closed", "reopened", "labeled", "unlabeled"], [condition("azure_devops_issue", "project_id", ["project-id"])]);
  sub.activity_normalization.closed_states = ["Done", "Closed"];
  const config = configuration([sub]);
  providerSource(config, "azure-devops");
  const p = azure();
  const match = () => project(sub, config.sources[0], p, undefined, config.github);
  p.resource.fields["System.State"] = { oldValue: "Done", newValue: "Active" };
  assert.deepEqual(match().filters["closed-states"], ["Done"]);
  p.resource.fields["System.State"].newValue = "Closed";
  assert.equal(match().filters["closed-states"], undefined);
  p.resource.fields["System.Tags"] = { oldValue: "same; same", newValue: "same" };
  assert.deepEqual(match().type, ["edited"]);
  p.resource.fields = { "System.State": "Done" };
  assert.deepEqual(match().type, ["edited"]);
  sub.types = ["edited"];
  assert.equal(match().filters["closed-states"], undefined);
  p.resourceVersion = "1.0";
  delete p.resource.workItemId;
  delete p.resource.revision;
  p.resource.id = 42;
  assert.equal(match().url, "https://dev.azure.com/company/_apis/wit/workItems/42");
});

test("security and Projects ingress use approved resource identity and selector-only projection", async t => {
  for (const event of ["dependabot_alert", "code_scanning_alert", "projects_v2_item"]) {
    await t.test(event, async t => {
      const sub = subscription(event, ["created"], event === "projects_v2_item" ? [condition(event, "project_id", ["PVT_test"])] : [condition(event, "severity", [event === "dependabot_alert" ? "high" : "error"])]);
      sub.orgs = ["source"];
      if (event !== "projects_v2_item") sub.repos = ["SOURCE/*"];
      const config = configuration([sub]);
      config.sources[0].scope.project = { nodeId: "PVT_test", ownerKind: "orgs", owner: "source", number: "7" };
      const h = await harness(t, config);
      const p = issue("created");
      p.projects_v2_item = { id: 123, project_node_id: "PVT_test" };
      p.alert = {
        number: 42,
        url: `https://api.github.com/repos/source/repo/${event === "dependabot_alert" ? "dependabot" : "code-scanning"}/alerts/42`,
        rule: { severity: "error" },
        security_advisory: { severity: "high", description: "private" },
      };
      assert.equal((await h.send(p, { nativeEvent: event })).status, 200);
      await h.drainOne();
      assert.equal(h.store.status()[0].state, "dispatched");
      assert.ok(!JSON.stringify(h.calls.find(call => call.path.endsWith("/dispatches")).body).includes("private"));
    });
  }
});

test("provider create/delete/restore are native lifecycle events, not inferred closure", () => {
  for (const [event, provider, build, pairs] of [
    [
      "jira_issue",
      "jira",
      jira,
      [
        ["jira:issue_created", "opened"],
        ["jira:issue_deleted", "deleted"],
      ],
    ],
    [
      "linear_issue",
      "linear",
      linear,
      [
        ["create", "opened"],
        ["remove", "deleted"],
      ],
    ],
    [
      "azure_devops_issue",
      "azure-devops",
      azure,
      [
        ["workitem.created", "opened"],
        ["workitem.deleted", "deleted"],
        ["workitem.restored", "restored"],
      ],
    ],
  ]) {
    const filter = { jira: ["project", "APP"], linear: ["organization-id", "org-id"], "azure-devops": ["project_id", "project-id"] }[provider];
    const sub = subscription(
      event,
      pairs.map(([, type]) => type),
      [condition(event, filter[0], [filter[1]])]
    );
    const config = configuration([sub]);
    providerSource(config, provider);
    for (const [native, type] of pairs) {
      const payload = build(native);
      if (provider === "azure-devops") payload.resource.id = 42;
      assert.deepEqual(project(sub, config.sources[0], payload, undefined, config.github).type, [type]);
    }
  }
});

test("catalogued body/tag scalar selectors forward only exact configured matching literals", () => {
  const sub = subscription("projects_v2_item", ["edited"], [condition("projects_v2_item", "project_id", ["PVT_test"]), condition("projects_v2_item", "body.from", ["reviewed literal"])]);
  const config = configuration([sub]);
  config.sources[0].scope.project = { nodeId: "PVT_test", ownerKind: "orgs", owner: "source", number: "7" };
  const p = { ...issue("edited"), projects_v2_item: { id: 123, project_node_id: "PVT_test" }, changes: { body: { from: "reviewed literal", to: "private new text" } } };
  const forwarded = project(sub, config.sources[0], p, "projects_v2_item", config.github);
  assert.deepEqual(forwarded.filters["body.from"], ["reviewed literal"]);
  assert.ok(!JSON.stringify(forwarded).includes("private"));
  p.changes.body.from = "unselected text";
  assert.equal(project(sub, config.sources[0], p, "projects_v2_item", config.github), undefined);
  const azureSub = subscription("azure_devops_issue", ["edited"], [condition("azure_devops_issue", "project_id", ["project-id"]), condition("azure_devops_issue", "tags.from", ["old; kept"])]);
  providerSource(config, "azure-devops");
  assert.deepEqual(project(azureSub, config.sources[0], azure(), undefined, config.github).filters["tags.from"], ["old; kept"]);
});
