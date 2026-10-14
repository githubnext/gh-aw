// @ts-check
import { describe, expect, it } from "vitest";
import { createRequire } from "node:module";
import { readWorkQueueLog } from "./work_queue_store.cjs";
import { fakeGitHub } from "./work_queue_store_checks.cjs";
import { genesis } from "./work_queue_test_helpers.cjs";

const require = createRequire(import.meta.url);
const { registerStaticQueueAuthentication, queueAuthenticationScope } = require("./work_queue_auth_cache.cjs");

describe("work queue static-auth cache scope", () => {
  it("never infers static authentication from an auth() token result", async () => {
    const client = { auth: async () => ({ type: "token", token: "unknown-token" }) };
    expect(await queueAuthenticationScope(client)).toBeUndefined();
  });

  it("invalidates a registered scope permanently after token rotation, auth replacement or resolution failure", async () => {
    for (const change of ["rotation", "replacement", "failure", "unauthenticated", "hook"]) {
      let token = "original-token";
      let fail = false;
      let type = "token";
      const auth = Object.assign(
        async () => {
          if (fail) throw new Error("Authentication unavailable");
          return { type, token };
        },
        { hook: () => {} }
      );
      const client = { auth };
      registerStaticQueueAuthentication(client, token);
      const scope = await queueAuthenticationScope(client);
      expect(scope).toBeDefined();
      expect(await queueAuthenticationScope(client)).toBe(scope);
      if (change === "rotation") token = "replacement-token";
      if (change === "replacement") client.auth = Object.assign(async () => ({ type, token }), { hook: auth.hook });
      if (change === "failure") fail = true;
      if (change === "unauthenticated") type = "unauthenticated";
      if (change === "hook") auth.hook = () => {};
      expect(await queueAuthenticationScope(client)).toBeUndefined();
      client.auth = auth;
      token = "original-token";
      fail = false;
      type = "token";
      expect(await queueAuthenticationScope(client)).toBeUndefined();
    }
  });

  it.each(["getCommit", "getTree", "getBlob"])("does not replay cached contents when same-client credentials lose %s access", async method => {
    for (const authentication of ["rotated-token", "replaced-auth", "reregistered-auth", "dynamic-auth", "unknown-auth"]) {
      const fake = fakeGitHub([genesis()]);
      let token = "initial-token";
      if (authentication !== "unknown-auth") Object.assign(fake.githubClient, { auth: async () => ({ type: "token", token }) });
      if (authentication === "rotated-token" || authentication === "replaced-auth" || authentication === "reregistered-auth") registerStaticQueueAuthentication(fake.githubClient, token);
      const input = { githubClient: fake.githubClient, owner: "owner", repo: "repo" };
      await readWorkQueueLog(input);
      if (authentication === "rotated-token" || authentication === "reregistered-auth") token = "downgraded-token";
      if (authentication === "replaced-auth") Object.assign(fake.githubClient, { auth: async () => ({ type: "token", token: "replacement-token" }) });
      if (authentication === "reregistered-auth") registerStaticQueueAuthentication(fake.githubClient, token);
      const git = fake.githubClient.rest.git;
      const original = git[method];
      let denied = 0;
      git[method] = async () => {
        denied++;
        throw Object.assign(new Error("Contents access denied"), { status: 403 });
      };
      for (let attempt = 0; attempt < 2; attempt++) await expect(readWorkQueueLog(input)).rejects.toThrow("Contents access denied");
      expect(denied).toBe(2);
      git[method] = original;
      expect((await readWorkQueueLog(input)).transactions).toHaveLength(1);
      expect(fake.state.updates).toBe(0);
    }
  });
});
