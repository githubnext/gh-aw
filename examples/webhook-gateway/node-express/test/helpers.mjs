import { createHmac, generateKeyPairSync, verify } from "node:crypto";
import { createServer } from "node:http";
import { mkdtempSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import assert from "node:assert/strict";
import { contextMapping, definitions, notifications, providers } from "../src/contracts.mjs";
import { hash } from "../src/core.mjs";
import { Store } from "../src/store.mjs";
import { createGateway } from "../src/server.mjs";

export const clock = 1792440000000;
const keys = generateKeyPairSync("rsa", { modulusLength: 2048 });
export const env = {
  GH_AW_GATEWAY_APP_ID: "123",
  GH_AW_GATEWAY_APP_BOT_ID: "456",
  GH_AW_GATEWAY_APP_PRIVATE_KEY: keys.privateKey.export({ type: "pkcs8", format: "pem" }),
  SOURCE_SIGNING_SECRET: "synthetic-fixture-not-a-real-provider-secret",
};
export function condition(event, filter, values) {
  const selector = definitions[event].filters[filter];
  return { filter, path: selector.path, operator: "equals", value_type: selector.type, values };
}
export function subscription(event = "issues", types = ["opened"], conditions = []) {
  const definition = definitions[event];
  const workflow = "reference";
  const result = {
    schema_version: 2,
    event,
    types,
    conditions,
    dispatch: {
      repository: "${{ github.repository }}",
      method: "repository_dispatch",
      event_type: `gh-aw-webhook-${hash(workflow).slice(0, 24)}`,
      workflow: `.github/workflows/${workflow}.lock.yml`,
      payload_projection: { policy: "resource-url-types-filters/v2" },
      context: contextMapping,
      batch: { mode: "same-event", deduplicate_by: "delivery_id" },
      max_payload_bytes: 65535,
      max_request_bytes: 65535,
    },
  };
  if (definition?.native_event) result.native_events = [definition.native_event];
  if (definition?.provider) {
    result.provider = definition.provider;
    result.activity_normalization = { policy: `${definition.provider}-issue/v1` };
    result.native_events = [...new Set(types.map(type => definition.native_events[type]))].sort();
  }
  return result;
}
export function configuration(subscriptions = [subscription()]) {
  return {
    github: { apiUrl: "https://api.github.com", serverUrl: "https://github.com", appId: "123", botId: "456", privateKeyEnv: "GH_AW_GATEWAY_APP_PRIVATE_KEY" },
    sources: [
      { id: "source", provider: "github", approved: true, revision: "review-1", credentialEnv: "SOURCE_SIGNING_SECRET", scope: { repositoryId: "12", repository: "source/repo", ownerId: "11", organization: { id: "11", login: "source" } } },
    ],
    destinations: [
      {
        id: "destination",
        approved: true,
        revision: "approval-1",
        repository: "dest/repo",
        repositoryId: "789",
        installationId: "321",
        accountId: "99",
        accountLogin: "dest",
        installationPermissions: { contents: "write", metadata: "read" },
        defaultBranch: "main",
        workflowRevision: "a".repeat(40),
        artifactSha256: "",
        gateVerified: true,
        sourceIds: ["source"],
        subscriptions,
      },
    ],
  };
}
export function artifactFor(config) {
  const artifact = `# gh-aw-manifest: ${JSON.stringify({ version: "1", webhook_subscriptions: config.destinations[0].subscriptions })}\non:\n  repository_dispatch: {}\n`;
  config.destinations[0].artifactSha256 = hash(artifact);
  return artifact;
}
export function issue(action = "opened") {
  return {
    action,
    repository: { id: 12, full_name: "source/repo", owner: { id: 11 } },
    organization: { id: 11, login: "source" },
    sender: { login: "author" },
    issue: { number: 42, html_url: "https://github.com/source/repo/issues/42", body: "private body that must never escape", labels: [{ name: "ready" }, { name: "private-label" }] },
  };
}
export function notification(event, action = notifications.events[event].actions[0]) {
  const p = issue(action);
  const definition = notifications.events[event];
  if (event === "pull_request_comment") p.issue.pull_request = {};
  if (event.startsWith("pull_request") && event !== "pull_request_comment") p.pull_request = { number: 42, html_url: "https://github.com/source/repo/pull/42" };
  if (event.startsWith("discussion")) p.discussion = { number: 42, html_url: "https://github.com/source/repo/discussions/42" };
  const base = `https://github.com/source/repo/${definition.path}/${event === "commit_comment" ? "a".repeat(40) : "42"}`;
  if (definition.anchor) {
    const comment = { id: 123, html_url: `${base}#${definition.anchor}123`, body: "/review private arguments", commit_id: "a".repeat(40), path: "file.js", user: { login: "reviewer" } };
    if (event === "pull_request_review") p.review = comment;
    else if (event === "pull_request_review_thread") p.thread = { comments: [comment] };
    else p.comment = comment;
  }
  return p;
}
export function providerSource(config, provider) {
  const source = config.sources[0];
  source.provider = provider;
  source.scope = {
    jira: { origin: "https://example.atlassian.net", project: "APP" },
    linear: { organizationId: "org-id", workspace: "workspace", teamId: "team-id" },
    "azure-devops": { collectionId: "collection-id", projectId: "project-id", collectionUrl: "https://dev.azure.com/company" },
  }[provider];
}
export async function listen(server) {
  await new Promise(resolve => server.listen(0, "127.0.0.1", resolve));
  return `http://127.0.0.1:${server.address().port}`;
}
export async function harness(t, config = configuration(), options = {}) {
  const artifact = artifactFor(config);
  const directory = mkdtempSync(join(tmpdir(), "express-gateway-"));
  const filename = join(directory, "state.sqlite");
  const store = new Store(filename);
  const calls = [];
  const logs = [];
  const testEnv = { ...env, ...options.env };
  const state = {
    config,
    dispatchStatus: 204,
    tokenStatus: 201,
    installationStatus: 200,
    revokeStatus: 204,
    tokenPermissions: { contents: "write", metadata: "read" },
    tokenRepo: "dest/repo",
    tokenRepoId: 789,
    tokenSelection: "selected",
    tokenExpiry: new Date(clock + 3600000).toISOString(),
    artifact,
    installationAppId: 123,
  };
  const api = createServer(async (req, res) => {
    const chunks = [];
    for await (const chunk of req) chunks.push(chunk);
    const body = chunks.length ? JSON.parse(Buffer.concat(chunks)) : undefined;
    calls.push({ method: req.method, path: req.url, authorization: req.headers.authorization, body });
    const token = "opaque.synthetic.installation.credential";
    if (req.url.includes("/installation") || req.url.includes("/access_tokens")) {
      if (req.url !== "/installation/token") {
        const jwt = req.headers.authorization?.slice(7);
        const [head, claim, signature] = jwt.split(".");
        assert.equal(verify("RSA-SHA256", Buffer.from(`${head}.${claim}`), keys.publicKey, Buffer.from(signature, "base64url")), true);
        const claims = JSON.parse(Buffer.from(claim, "base64url"));
        assert.equal(claims.iss, "123");
        assert.ok(claims.exp <= Math.floor((state.now ?? clock) / 1000) + 600);
        assert.equal(claims.iat, Math.floor((state.now ?? clock) / 1000) - 60);
      }
    } else assert.equal(req.headers.authorization, `Bearer ${token}`);
    let status = 200;
    let result;
    if (req.url === "/repos/dest/repo/installation") {
      status = state.installationStatus;
      result = { id: 321, app_id: state.installationAppId, account: { id: 99, login: "dest" }, suspended_at: null, permissions: { contents: "write", metadata: "read" } };
    } else if (req.url === "/app/installations/321/access_tokens") {
      assert.deepEqual(body, { repository_ids: [789], permissions: { contents: "write" } });
      status = state.tokenStatus;
      result = { token, expires_at: state.tokenExpiry, permissions: state.tokenPermissions, repository_selection: state.tokenSelection, repositories: [{ id: state.tokenRepoId, full_name: state.tokenRepo }] };
    } else if (req.url === "/repos/dest/repo") result = { id: 789, full_name: "dest/repo", owner: { id: 99 }, default_branch: "main" };
    else if (req.url === "/repos/dest/repo/commits/main") result = { sha: state.revision ?? "a".repeat(40) };
    else if (req.url.startsWith("/repos/dest/repo/contents/")) result = { encoding: "base64", content: Buffer.from(state.artifact).toString("base64") };
    else if (req.url === "/repos/dest/repo/dispatches") {
      if (state.abortDispatch) {
        req.socket.destroy();
        return;
      }
      status = state.dispatchStatus;
    } else if (req.url === "/installation/token") {
      assert.equal(req.headers.authorization, `Bearer ${token}`);
      status = state.revokeStatus;
    } else {
      status = 404;
      result = { message: "synthetic provider response must not enter errors" };
    }
    if (options.afterCall) options.afterCall(req.url, state);
    res.writeHead(status, { "content-type": "application/json", ...(status === 302 ? { location: "https://untrusted.example/token" } : {}) });
    res.end(status === 204 ? undefined : JSON.stringify(result ?? { message: token }));
  });
  const apiUrl = await listen(api);
  let ingress;
  t.after(async () => {
    ingress?.closeAllConnections();
    api.closeAllConnections();
    await Promise.all([...(ingress ? [new Promise(resolve => ingress.close(resolve))] : []), new Promise(resolve => api.close(resolve))]);
    store.close();
    rmSync(directory, { recursive: true });
  });
  const gateway = createGateway({
    store,
    env: testEnv,
    loadConfiguration: () => state.config,
    now: () => state.now ?? clock,
    testHTTP: true,
    audit: event => logs.push(event),
    fetchImpl: (url, init) => {
      assert.ok(url.startsWith("https://api.github.com/"));
      return fetch(`${apiUrl}${url.slice("https://api.github.com".length)}`, init);
    },
  });
  ingress = createServer(gateway.app);
  const ingressUrl = await listen(ingress);
  async function send(payload = issue(), { nativeEvent = "issues", delivery = "original-id", raw = JSON.stringify(payload), signatureSecret = testEnv.SOURCE_SIGNING_SECRET, headers = {} } = {}) {
    const provider = state.config.sources[0].provider;
    const digest = createHmac("sha256", signatureSecret).update(raw).digest("hex");
    const auth =
      provider === "github"
        ? { "x-hub-signature-256": `sha256=${digest}` }
        : provider === "jira"
          ? { "x-hub-signature": `sha256=${digest}` }
          : provider === "linear"
            ? { "linear-signature": digest }
            : { authorization: `Basic ${Buffer.from(signatureSecret).toString("base64")}` };
    return fetch(`${ingressUrl}/ingress/source`, {
      method: "POST",
      headers: { "content-type": "application/json", "x-github-event": nativeEvent, "x-github-delivery": delivery, "x-atlassian-webhook-identifier": delivery, ...auth, ...headers },
      body: raw,
    });
  }
  return { ...gateway, store, filename, calls, logs, state, send, env: testEnv };
}
