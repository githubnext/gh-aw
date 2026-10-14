// @ts-check
"use strict";

const { createHash } = require("node:crypto");

/** @type {WeakMap<object, {auth: Function, hook: unknown, fingerprint: string, scope: object}>} */
const staticAuthentication = new WeakMap();

function fingerprint(token) {
  return createHash("sha256")
    .update(token.replace(/^(token|bearer) +/i, ""))
    .digest("hex");
}

// Register only at a known string-token constructor, never from auth() alone:
// a dynamic auth strategy can return the same token before later rotating it.
function registerStaticQueueAuthentication(client, token) {
  if (typeof token !== "string" || !token || typeof client.auth !== "function") return;
  staticAuthentication.set(client, { auth: client.auth, hook: client.auth.hook, fingerprint: fingerprint(token), scope: {} });
}

async function queueAuthenticationScope(client) {
  const registered = staticAuthentication.get(client);
  if (!registered) return undefined;
  try {
    if (client.auth === registered.auth && client.auth.hook === registered.hook) {
      const authentication = await client.auth({ type: "token" });
      if (authentication?.type === "token" && typeof authentication.token === "string" && fingerprint(authentication.token) === registered.fingerprint) return registered.scope;
    }
  } catch {
    // Authentication failure is handled by the fresh API probes, not a cache.
  }
  staticAuthentication.delete(client);
  return undefined;
}

module.exports = { registerStaticQueueAuthentication, queueAuthenticationScope };
