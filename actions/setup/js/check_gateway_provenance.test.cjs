// @ts-check

import { afterEach, describe, expect, it, vi } from "vitest";
import fs from "node:fs";
import { createRequire } from "node:module";
import { getOctokit } from "@actions/github";

const require = createRequire(import.meta.url);
const { main, verifyGatewayProvenance } = require("./check_gateway_provenance.cjs");
const { validateWebhook } = require("./validate_webhook.cjs");
const route = "gh-aw-webhook-0123456789abcdef01234567";

function environment(overrides = {}) {
  return {
    GH_AW_GATEWAY_APP_ID: "123",
    GH_AW_GATEWAY_APP_BOT_ID: "456",
    GH_AW_GATEWAY_EVENT_NAME: "repository_dispatch",
    GH_AW_GATEWAY_EVENT_TYPE: route,
    GH_AW_GATEWAY_ACTOR_ID: "456",
    GH_AW_GATEWAY_REPOSITORY_ID: "789",
    GH_AW_GATEWAY_REPOSITORY: "destination/workflows",
    ...overrides,
  };
}

function dispatch() {
  return {
    action: route,
    sender: { type: "Bot", id: 456, login: "cao[bot]" },
    repository: { id: 789, full_name: "destination/workflows" },
    client_payload: {
      schema_version: 2,
      aw_context: {
        webhook: {
          event: "projects_v2_item",
          deliveries: [{ delivery_id: "delivery-1", payload: { url: "https://api.github.com/orgs/source/projectsV2/1/items/2", type: ["created"] } }],
        },
      },
    },
  };
}

function client() {
  return {
    rest: {
      apps: { getBySlug: vi.fn().mockResolvedValue({ status: 200, data: { id: 123, slug: "cao" } }) },
      users: { getByUsername: vi.fn().mockResolvedValue({ status: 200, data: { id: 456, type: "Bot", login: "cao[bot]" } }) },
    },
  };
}

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
});

describe("verifyGatewayProvenance", () => {
  it("binds the original Bot sender and actor to the centrally approved App and bot", async () => {
    const event = dispatch();
    const before = JSON.stringify(event);
    const github = client();
    await expect(verifyGatewayProvenance(event, github, environment())).resolves.toEqual({ verified: true });
    expect(JSON.stringify(event)).toBe(before);
    const request = { retries: 0, timeout: 15000, redirect: "error" };
    expect(github.rest.apps.getBySlug).toHaveBeenCalledExactlyOnceWith({ app_slug: "cao", request });
    expect(github.rest.users.getByUsername).toHaveBeenCalledExactlyOnceWith({ username: "cao[bot]", request });
  });

  it.each(["", "0", "-1", "01", "+123", "1.2", "1e2", " 123", "123\n", "123456789012345678901", "${{ vars.APP_ID }}", undefined])("fails closed for malformed or absent central identity %s", async value => {
    for (const name of ["GH_AW_GATEWAY_APP_ID", "GH_AW_GATEWAY_APP_BOT_ID"]) {
      const github = client();
      await expect(verifyGatewayProvenance(dispatch(), github, environment({ [name]: value }))).rejects.toThrow(/ERR_CONFIG.*central/);
      expect(github.rest.apps.getBySlug).not.toHaveBeenCalled();
    }
  });

  it.each([
    ["human", { type: "User", id: 456, login: "cao[bot]" }],
    ["other App", { type: "Bot", id: 999, login: "other[bot]" }],
    ["workflow token", { type: "Bot", id: 41898282, login: "github-actions[bot]" }],
    ["missing sender", null],
    ["missing type", { id: 456, login: "cao[bot]" }],
    ["invalid login", { type: "Bot", id: 456, login: "cao" }],
    ["URL login", { type: "Bot", id: 456, login: "https://attacker.example/cao[bot]" }],
    ["unsafe numeric ID", { type: "Bot", id: Number.MAX_SAFE_INTEGER + 1, login: "cao[bot]" }],
  ])("rejects %s regardless of payload identity or allowed roles/bots", async (_name, sender) => {
    const event = dispatch();
    event.sender = sender;
    event.client_payload.sender = { type: "Bot", id: 456, login: "cao[bot]" };
    event.client_payload.app = { id: 123 };
    event.installation = { id: 123 };
    const github = client();
    await expect(verifyGatewayProvenance(event, github, environment({ GH_AW_REQUIRED_ROLES: "admin", GH_AW_ALLOWED_BOTS: "other,github-actions" }))).rejects.toThrow(/provenance rejected/);
    expect(github.rest.apps.getBySlug).not.toHaveBeenCalled();
  });

  it.each(["", "999", "0456", "456\n", undefined])("rejects mismatching original actor %s, including on reruns", async actor => {
    await expect(verifyGatewayProvenance(dispatch(), client(), environment({ GH_AW_GATEWAY_ACTOR_ID: actor, GITHUB_TRIGGERING_ACTOR: "cao[bot]" }))).rejects.toThrow(/original actor/);
  });

  it("does not authenticate the rerun requester", async () => {
    await expect(verifyGatewayProvenance(dispatch(), client(), environment({ GITHUB_TRIGGERING_ACTOR: "human-admin" }))).resolves.toEqual({ verified: true });
  });

  it.each([
    { GH_AW_GATEWAY_EVENT_NAME: "workflow_dispatch" },
    { GH_AW_GATEWAY_EVENT_TYPE: "other-route" },
    { GH_AW_GATEWAY_EVENT_TYPE: undefined },
    { GH_AW_GATEWAY_REPOSITORY_ID: "999" },
    { GH_AW_GATEWAY_REPOSITORY_ID: undefined },
    { GH_AW_GATEWAY_REPOSITORY: "other/repository" },
    { GH_AW_GATEWAY_REPOSITORY: "destination/workflows\n" },
    { GH_AW_GATEWAY_REPOSITORY: undefined },
  ])("rejects wrong or missing route/destination %#", async overrides => {
    const github = client();
    await expect(verifyGatewayProvenance(dispatch(), github, environment(overrides))).rejects.toThrow(/provenance rejected/);
    expect(github.rest.apps.getBySlug).not.toHaveBeenCalled();
  });

  it.each([undefined, null, {}, { action: route }, { ...dispatch(), action: `${route}0` }, { ...dispatch(), repository: null }])("rejects malformed GitHub events %#", async event => {
    await expect(verifyGatewayProvenance(event, client(), environment())).rejects.toThrow(/provenance rejected/);
  });

  it.each([
    ["apps", "getBySlug", undefined],
    ["apps", "getBySlug", { status: 200, data: {} }],
    ["apps", "getBySlug", { status: 200, data: { id: 999, slug: "cao" } }],
    ["apps", "getBySlug", { status: 200, data: { id: 123, slug: "other" } }],
    ["apps", "getBySlug", { status: 302, data: { id: 123, slug: "cao" } }],
    ["users", "getByUsername", { status: 200, data: { id: 999, type: "Bot", login: "cao[bot]" } }],
    ["users", "getByUsername", { status: 200, data: { id: 456, type: "User", login: "cao[bot]" } }],
    ["users", "getByUsername", { status: 200, data: { id: 456, type: "Bot", login: "other[bot]" } }],
    ["users", "getByUsername", { status: 200, data: null }],
    ["users", "getByUsername", undefined],
  ])("rejects unbound or missing %s metadata %#", async (group, method, response) => {
    const github = client();
    github.rest[group][method].mockResolvedValue(response);
    await expect(verifyGatewayProvenance(dispatch(), github, environment())).rejects.toThrow(/metadata do not match/);
  });

  it.each(["403", "404", "500", "timeout"])("fails closed without disclosing API credentials on %s", async status => {
    for (const [group, method] of [
      ["apps", "getBySlug"],
      ["users", "getByUsername"],
    ]) {
      const github = client();
      github.rest[group][method].mockRejectedValue(new Error(`${status}: test-secret-must-not-appear`));
      const promise = verifyGatewayProvenance(dispatch(), github, environment());
      await expect(promise).rejects.toThrow(/ERR_API.*Unable to verify/);
      await expect(promise).rejects.not.toThrow(/test-secret-must-not-appear/);
    }
  });

  it.each(["https://api.github.com", "https://github.example.com/api/v3"])("uses only authenticated fixed metadata endpoints at %s", async baseUrl => {
    const fetch = vi.fn().mockImplementation(async url => {
      const data = String(url).endsWith("/apps/cao") ? { id: 123, slug: "cao" } : { id: 456, type: "Bot", login: "cao[bot]" };
      return new Response(JSON.stringify(data), { status: 200, headers: { "content-type": "application/json" } });
    });
    const github = getOctokit("fixture-token", { baseUrl, request: { fetch } });
    const event = dispatch();
    event.sender.url = "https://attacker.example/sender";
    event.repository.url = "https://attacker.example/repository";
    await expect(verifyGatewayProvenance(event, github, environment())).resolves.toEqual({ verified: true });
    expect(fetch.mock.calls.map(([url]) => url)).toEqual([`${baseUrl}/apps/cao`, `${baseUrl}/users/cao%5Bbot%5D`]);
    for (const [, options] of fetch.mock.calls) {
      expect(options.method).toBe("GET");
      expect(options.headers.authorization).toBe("token fixture-token");
      expect(options.redirect).toBe("error");
    }
  });

  it("still requires the minimal payload validator after authenticating the App", async () => {
    const event = dispatch();
    await expect(verifyGatewayProvenance(event, client(), environment())).resolves.toEqual({ verified: true });
    const env = {
      GITHUB_EVENT_NAME: "repository_dispatch",
      GITHUB_API_URL: "https://api.github.com",
      GH_AW_WEBHOOK_SUBSCRIPTIONS: JSON.stringify([
        {
          event: "projects_v2_item",
          schema_version: 2,
          types: ["created"],
          conditions: [],
          dispatch: { method: "repository_dispatch", event_type: route, payload_projection: { policy: "resource-url-types-filters/v2" } },
        },
      ]),
    };
    expect(validateWebhook(event, env)).toMatchObject({ skipped: false, delivery_ids: ["delivery-1"] });
    event.client_payload.aw_context.webhook.deliveries[0].payload.body = "untrusted source body";
    expect(() => validateWebhook(event, env)).toThrow();
  });
});

describe("main", () => {
  function setup(event) {
    const github = client();
    const core = {
      info: vi.fn(),
      warning: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue(undefined) },
    };
    vi.stubGlobal("github", github);
    vi.stubGlobal("core", core);
    for (const [name, value] of Object.entries(environment())) vi.stubEnv(name, value);
    vi.stubEnv("RUNNER_TEMP", "");
    vi.stubEnv("GITHUB_EVENT_PATH", "/runner-owned/event.json");
    vi.spyOn(fs, "readFileSync").mockReturnValue(JSON.stringify(event));
    return { core, github };
  }

  it("publishes verified only after reading and verifying the runner event", async () => {
    const { core } = setup(dispatch());
    await main();
    expect(fs.readFileSync).toHaveBeenCalledWith("/runner-owned/event.json", "utf8");
    expect(core.setOutput.mock.calls).toEqual([
      ["verified", "false"],
      ["verified", "true"],
    ]);
    expect(core.setFailed).not.toHaveBeenCalled();
  });

  it.each(["unreadable", "malformed", "missing path", "sender mismatch", "missing config", "API failure"])("denies %s without publishing verified=true", async failure => {
    const event = dispatch();
    if (failure === "sender mismatch") event.sender.type = "User";
    const { core, github } = setup(event);
    if (failure === "unreadable")
      vi.mocked(fs.readFileSync).mockImplementation(() => {
        throw new Error("secret-path-detail");
      });
    if (failure === "malformed") vi.mocked(fs.readFileSync).mockReturnValue("{");
    if (failure === "missing path") vi.stubEnv("GITHUB_EVENT_PATH", "");
    if (failure === "missing config") vi.stubEnv("GH_AW_GATEWAY_APP_ID", "");
    if (failure === "API failure") github.rest.apps.getBySlug.mockRejectedValue(new Error("secret-token-detail"));
    await main();
    expect(core.setOutput.mock.calls).toEqual([["verified", "false"]]);
    expect(core.setFailed).toHaveBeenCalledOnce();
    expect(core.summary.addRaw).toHaveBeenCalledOnce();
    expect(JSON.stringify(core.setFailed.mock.calls)).not.toContain("secret-");
    expect(JSON.stringify(core.summary.addRaw.mock.calls)).not.toContain("secret-");
  });
});
