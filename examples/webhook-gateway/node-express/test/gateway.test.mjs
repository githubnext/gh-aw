import test from "node:test";
import assert from "node:assert/strict";
import { canonical, envelope, httpsURL, parseJSON } from "../src/core.mjs";
import { notifications, validateConfiguration, validateSubscription } from "../src/contracts.mjs";
import { project } from "../src/matching.mjs";
import { Store } from "../src/store.mjs";
import { artifactFor, clock, condition, configuration, env, harness, issue, notification, providerSource, subscription } from "./helpers.mjs";

test("real HTTP ingress authenticates original bytes, durably admits, scopes App token and dispatches minimal payload", async t => {
  const sub = subscription("issues", ["opened"], [condition("issues", "labels", ["ready", "absent"])]);
  const h = await harness(t, configuration([sub]));
  const raw = ` \n${JSON.stringify(issue(), null, 2)}\n`;
  const response = await h.send(issue(), { raw });
  assert.equal(response.status, 200);
  assert.deepEqual(await response.json(), { duplicate: false, admitted: 1 });
  assert.equal(h.calls.length, 0);
  assert.equal(h.store.status()[0].state, "pending");
  assert.equal(await h.drainOne(), true);
  assert.equal(h.store.status()[0].state, "dispatched");
  assert.deepEqual(
    h.calls.map(call => `${call.method} ${call.path.split("?")[0]}`),
    [
      "GET /repos/dest/repo/installation",
      "POST /app/installations/321/access_tokens",
      "GET /repos/dest/repo",
      "GET /repos/dest/repo/commits/main",
      "GET /repos/dest/repo/contents/.github/workflows/reference.lock.yml",
      "POST /repos/dest/repo/dispatches",
      "DELETE /installation/token",
    ]
  );
  const dispatch = h.calls.find(call => call.path.endsWith("/dispatches"));
  assert.deepEqual(dispatch.body.client_payload.aw_context.webhook.deliveries, [
    {
      delivery_id: "original-id",
      payload: { url: "https://github.com/source/repo/issues/42", type: ["opened"], filters: { labels: ["ready"] } },
    },
  ]);
  assert.ok(!JSON.stringify(dispatch.body).includes("private"));
  assert.ok(!JSON.stringify(h.logs).includes("credential"));
  assert.ok(!JSON.stringify(h.logs).includes(env.SOURCE_SIGNING_SECRET));
});

test("all ten notification families bind their original identity and cataloged anchor", async t => {
  for (const [event, definition] of Object.entries(notifications.events)) {
    await t.test(event, async t => {
      const h = await harness(t, configuration([subscription(event, [definition.actions[0]])]));
      const response = await h.send(notification(event), { nativeEvent: definition.native_event });
      assert.equal(response.status, 200, await response.text());
      assert.equal(h.store.status().length, 1);
      await h.drainOne();
      assert.equal(h.store.status()[0].state, "dispatched");
    });
  }
});

test("authenticated unapproved, forged, malformed, and spoofed sources never create jobs", async t => {
  const h = await harness(t);
  let response = await h.send(issue(), { signatureSecret: "wrong" });
  assert.equal(response.status, 401);
  h.state.config.sources[0].approved = false;
  response = await h.send();
  assert.equal(response.status, 403);
  h.state.config.sources[0].approved = true;
  const cases = [
    '{"action":"opened","action":"closed"}',
    '{"constructor":{}}',
    '{"repository":{"id":9007199254740993}}',
    '{"repository":{"id":9007199254740991.4}}',
    '{"repository":{"id":1.00000000000000001}}',
    '{"n":1e999}',
    '{"n":01}',
    '{"x":[1,]}',
    '{"x":true} trailing',
  ];
  for (const raw of cases) assert.equal((await h.send(undefined, { raw })).status, 400);
  const spoofed = issue();
  spoofed.repository.id = 999;
  assert.equal((await h.send(spoofed)).status, 403);
  assert.equal(h.store.status().length, 0);
  assert.equal(h.calls.length, 0);
});

test("delivery deduplication is durable, content-based and concurrency-safe; conflicting IDs reject", async t => {
  const h = await harness(t);
  const responses = await Promise.all(Array.from({ length: 12 }, () => h.send()));
  const results = await Promise.all(responses.map(response => response.json()));
  assert.equal(results.filter(result => !result.duplicate).length, 1);
  assert.equal(h.store.status().length, 1);
  assert.deepEqual(await (await h.send(issue(), { delivery: "rewritten-unsigned-id" })).json(), { duplicate: true, admitted: 0 });
  const changed = issue();
  changed.issue.number = 43;
  changed.issue.html_url = "https://github.com/source/repo/issues/43";
  assert.equal((await h.send(changed)).status, 409);
  const second = new Store(h.filename);
  t.after(() => second.close());
  const claimed = second.claim(clock);
  assert.equal(claimed.id, h.store.status()[0].id);
  assert.equal(h.store.claim(clock), undefined);
  assert.equal(h.store.status()[0].state, "in_flight");
  assert.equal(await h.drainOne(), false, "an interrupted claim is not automatically replayed");
});

test("approval withdrawal or protected revision change blocks admitted work before minting", async t => {
  for (const mutate of [
    config => {
      config.sources[0].approved = false;
    },
    config => {
      config.destinations[0].gateVerified = false;
    },
    config => {
      config.destinations[0].revision = "new-review";
    },
  ]) {
    await t.test(mutate.toString(), async t => {
      const h = await harness(t);
      await h.send();
      mutate(h.state.config);
      await h.drainOne();
      assert.equal(h.store.status()[0].state, "blocked");
      assert.equal(h.calls.length, 0);
    });
  }
});

test("revocation during lookup prevents mint, and revocation after mint still cleans up", async t => {
  for (const path of ["/repos/dest/repo/installation", "/app/installations/321/access_tokens"]) {
    await t.test(path, async t => {
      const h = await harness(t, configuration(), {
        afterCall: (called, state) => {
          if (called === path) state.config.sources[0].approved = false;
        },
      });
      await h.send();
      await h.drainOne();
      assert.equal(h.store.status()[0].state, "blocked");
      assert.equal(
        h.calls.some(call => call.path.endsWith("/dispatches")),
        false
      );
      assert.equal(
        h.calls.some(call => call.path === "/installation/token"),
        path.includes("access_tokens")
      );
    });
  }
});

test("mint, installation, scope, expiry, workflow drift and redirects fail closed with credential-safe errors", async t => {
  const cases = [
    ["installationStatus", 302, "failed"],
    ["installationAppId", 999, "blocked"],
    ["tokenStatus", 500, "pending"],
    ["tokenPermissions", { contents: "write", issues: "write" }, "blocked"],
    ["tokenRepoId", 790, "blocked"],
    ["tokenSelection", "all", "blocked"],
    ["tokenExpiry", new Date(clock - 1000).toISOString(), "blocked"],
    ["revision", "b".repeat(40), "blocked"],
    ["artifact", "tampered", "blocked"],
  ];
  for (const [key, value, expected] of cases) {
    await t.test(key, async t => {
      const h = await harness(t);
      h.state[key] = value;
      await h.send();
      await h.drainOne();
      assert.equal(h.store.status()[0].state, expected);
      assert.equal(
        h.calls.some(call => call.path.endsWith("/dispatches")),
        false
      );
      assert.ok(!JSON.stringify(h.store.status()).includes("opaque.synthetic"));
      assert.ok(!JSON.stringify(h.logs).includes("opaque.synthetic"));
      assert.ok(!JSON.stringify(h.logs).includes("BEGIN PRIVATE"));
      assert.ok(h.calls.every(call => !call.path.startsWith("https:")));
    });
  }
});

test("explicit dispatch rate limit retries with bounded backoff; uncertain sends never retry", async t => {
  for (const [status, state] of [
    [429, "pending"],
    [500, "uncertain"],
    [403, "failed"],
    [302, "failed"],
  ]) {
    await t.test(String(status), async t => {
      const h = await harness(t);
      h.state.dispatchStatus = status;
      await h.send();
      await h.drainOne();
      assert.equal(h.store.status()[0].state, state);
      assert.equal(await h.drainOne(), false);
      h.state.now = clock + 10000;
      h.state.dispatchStatus = 204;
      assert.equal(await h.drainOne(), state === "pending");
      assert.equal(h.store.status()[0].state, state === "pending" ? "dispatched" : state);
    });
  }
  await t.test("connection loss after send", async t => {
    const h = await harness(t);
    h.state.abortDispatch = true;
    await h.send();
    await h.drainOne();
    assert.equal(h.store.status()[0].state, "uncertain");
    assert.equal(await h.drainOne(), false);
  });
});

test("cleanup failure is explicit, including after a successful dispatch", async t => {
  const h = await harness(t);
  h.state.revokeStatus = 500;
  await h.send();
  await h.drainOne();
  assert.equal(h.store.status()[0].state, "dispatched_cleanup_failed");
  assert.equal(h.store.status()[0].error, "installation-token-revocation-failed");
});

test("cleanup failure preserves an uncertain dispatch outcome and cannot trigger replay", async t => {
  const h = await harness(t);
  h.state.revokeStatus = 500;
  h.state.abortDispatch = true;
  await h.send();
  await h.drainOne();
  assert.equal(h.store.status()[0].state, "uncertain_cleanup_failed");
  assert.equal(await h.drainOne(), false);
});

test("parallel workers cannot send one admission twice and retries stop at three attempts", async t => {
  const h = await harness(t);
  await h.send();
  h.state.dispatchStatus = 429;
  assert.deepEqual((await Promise.all([h.drainOne(), h.drainOne()])).sort(), [false, true]);
  for (let attempt = 1; attempt < 3; attempt++) {
    h.state.now = clock + 10000 * attempt;
    assert.equal(await h.drainOne(), true);
  }
  assert.equal(h.store.status()[0].attempts, 3);
  assert.equal(h.store.status()[0].state, "failed");
  h.state.now = clock + 100000;
  assert.equal(await h.drainOne(), false);
  assert.equal(h.calls.filter(call => call.path.endsWith("/dispatches")).length, 3);
});

test("closed subscription shape, policy, paths, central identity and label limitations are explicit", () => {
  const base = configuration();
  artifactFor(base);
  for (const mutate of [
    sub => {
      sub.dispatch.payload_projection.policy = "resource-url-types/v1";
    },
    sub => {
      sub.dispatch.extra = true;
    },
    sub => {
      sub.extra = true;
    },
    sub => {
      sub.native_events = ["pull_request"];
    },
    sub => {
      sub.conditions = [{ ...condition("issues", "labels", ["ready"]), path: ["issue", "body"] }];
    },
    sub => {
      sub.dispatch.event_type = "caller-selected-route";
    },
  ]) {
    const copy = structuredClone(base.destinations[0].subscriptions[0]);
    mutate(copy);
    assert.throws(() => validateSubscription(copy));
  }
  const label = subscription();
  label.event = "label_command";
  assert.throws(() => validateSubscription(label), /label-consumption-unsupported/);
  assert.throws(() => validateConfiguration(base, { ...env, GH_AW_GATEWAY_APP_ID: "00123" }), /central-identity-mismatch/);
});

test("typed conditions, complete alternatives, approved org scope, and minimal intersections", () => {
  const sub = subscription(
    "projects_v2_item",
    ["edited"],
    [condition("projects_v2_item", "project_id", ["PVT_test"]), condition("projects_v2_item", "field.name", ["Status"]), condition("projects_v2_item", "field.type", ["single_select"])]
  );
  sub.any_of = [
    [condition(sub.event, "field.from.name", ["Todo"]), condition(sub.event, "field.to.name", ["Ready"])],
    [condition(sub.event, "field.from.name", ["Doing"]), condition(sub.event, "field.to.name", ["Done"])],
  ];
  sub.orgs = ["sour*"];
  validateSubscription(sub);
  const config = configuration([sub]);
  config.sources[0].scope.project = { nodeId: "PVT_test", ownerKind: "orgs", owner: "source", number: "7" };
  const payload = { ...issue("edited"), projects_v2_item: { id: 123, project_node_id: "PVT_test" }, changes: { field_value: { field_name: "Status", field_type: "single_select", from: { name: "Todo" }, to: { name: "Ready" } } } };
  const projectIt = () => project(sub, config.sources[0], payload, "projects_v2_item", config.github);
  assert.deepEqual(projectIt().type, ["edited"]);
  assert.equal(projectIt().url, "https://api.github.com/orgs/source/projectsV2/7/items/123");
  assert.equal(Object.keys(projectIt().filters).length, 5);
  payload.changes.field_value.to.name = "Done";
  assert.equal(projectIt(), undefined, "cannot combine Todo with another alternative's Done");
  delete payload.changes.field_value.from;
  assert.equal(projectIt(), undefined, "cannot hydrate missing transition evidence");
  payload.organization.id = 99;
  assert.throws(projectIt, /source-organization-mismatch/);
});

test("slash matching is position-zero, case-sensitive and never forwards arguments", async t => {
  const sub = subscription("slash_command", ["created", "edited"]);
  sub.native_events = ["issue_comment"];
  sub.command_filter = { policy: "slash-command/v1", names: ["review", "fix*"], events: ["issue_comment"], strategy: "inline" };
  const h = await harness(t, configuration([sub]));
  for (const [index, body] of [" /review", "\n/review", "/Review", "/reviewer", "text\n/review", "/review!"].entries()) {
    const payload = notification("issue_comment");
    payload.comment.body = body;
    assert.equal((await h.send(payload, { nativeEvent: "issue_comment", delivery: `nonmatch-${index}` })).status, 200);
  }
  assert.equal(h.store.status().length, 0);
  const payload = notification("issue_comment");
  payload.comment.body = "/fix private arguments";
  assert.equal((await h.send(payload, { nativeEvent: "issue_comment" })).status, 200);
  await h.drainOne();
  const delivery = h.calls.find(call => call.path.endsWith("/dispatches")).body.client_payload.aw_context.webhook.deliveries[0];
  assert.deepEqual(delivery.payload.filters, { event: ["issue_comment"], command: ["fix"] });
  assert.ok(!JSON.stringify(delivery).includes("private"));
});

test("original URL/parent/comment evidence rejects mismatches and malformed PR split", async t => {
  const sub = subscription("issue_comment", ["created"]);
  const h = await harness(t, configuration([sub]));
  const payload = notification("issue_comment");
  payload.comment.id = 124;
  assert.equal((await h.send(payload, { nativeEvent: "issue_comment" })).status, 400);
  payload.comment.id = 123;
  payload.issue.pull_request = null;
  assert.equal((await h.send(payload, { nativeEvent: "issue_comment" })).status, 400);
  delete payload.issue.pull_request;
  payload.comment.html_url += "?credential=untrusted";
  assert.equal((await h.send(payload, { nativeEvent: "issue_comment" })).status, 400);
});

test("UTF-8 limits cover the exact complete serialized wrapper, not just client payload", () => {
  const sub = subscription();
  const base = { url: "https://github.com/source/repo/issues/42", type: ["opened"], filters: { labels: [""] } };
  const empty = envelope(sub, "id", base);
  const space = 65535 - Buffer.byteLength(JSON.stringify(empty));
  base.filters.labels[0] = "a".repeat(space);
  assert.equal(Buffer.byteLength(JSON.stringify(envelope(sub, "id", base))), 65535);
  base.filters.labels[0] += "a";
  assert.throws(() => envelope(sub, "id", base), /dispatch-too-large/);
  base.filters.labels[0] = "é".repeat(Math.floor(space / 2));
  assert.ok(Buffer.byteLength(JSON.stringify(envelope(sub, "id", base))) <= 65535);
  base.filters.labels[0] += "é";
  assert.throws(() => envelope(sub, "id", base), /dispatch-too-large/);
});

test("strict parser rejects non-UTF8, nesting, duplicate keys, unsafe integers and accepts distinct typed values", () => {
  assert.throws(() => parseJSON(Buffer.from([0xff])), /invalid-utf8/);
  assert.throws(() => parseJSON(Buffer.from("[".repeat(66) + "]".repeat(66))), /json-too-deep/);
  assert.equal(canonical(parseJSON(Buffer.from('{"b":false,"a":[1,"1"]}'))), '{"a":[1,"1"],"b":false}');
  assert.equal(parseJSON(Buffer.from("1e0")), 1);
});

test("WHATWG canonical references reject scheme/host/port/path rewrites and bare delimiters", () => {
  for (const url of [
    "HTTPS://github.com/source/repo/issues/42",
    "https://GITHUB.COM/source/repo/issues/42",
    "https://github.com:443/source/repo/issues/42",
    "https://github.com/source/./repo/issues/42",
    "https://github.com/source/repo/issues/42?",
    "https://github.com/source/repo/issues/42#",
    "https://github.com/source/repo/issues/42?#",
  ])
    assert.throws(() => httpsURL(url), /invalid-resource-url/);
  assert.equal(httpsURL("https://github.com/source/repo/pull/42#pullrequestreview-123").hash, "#pullrequestreview-123");
});

test("oversized matching-literal projection is rejected before durable admission, not truncated", async t => {
  const literal = "é".repeat(40000);
  const sub = subscription("issues", ["opened"], [condition("issues", "labels", [literal])]);
  const h = await harness(t, configuration([sub]));
  const payload = issue();
  payload.issue.labels = [{ name: literal }];
  assert.equal((await h.send(payload)).status, 413);
  assert.equal(h.store.status().length, 0);
  assert.equal(h.store.db.prepare("SELECT count(*) AS n FROM receipts").get().n, 0);
});

test("notification and slash subscriptions may independently admit the same delivery to one route", async t => {
  const slash = subscription("slash_command", ["created", "edited"]);
  slash.native_events = ["issue_comment"];
  slash.command_filter = { policy: "slash-command/v1", names: ["review"], events: ["issue_comment"], strategy: "inline" };
  const h = await harness(t, configuration([subscription("issue_comment", ["created"]), slash]));
  assert.deepEqual(await (await h.send(notification("issue_comment"), { nativeEvent: "issue_comment" })).json(), { duplicate: false, admitted: 2 });
  await h.drainOne();
  await h.drainOne();
  assert.equal(h.store.status().filter(job => job.state === "dispatched").length, 2);
});
