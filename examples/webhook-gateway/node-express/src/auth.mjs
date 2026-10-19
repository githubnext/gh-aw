import { createHmac } from "node:crypto";
import { canonical, equalSecret, hash, nonempty, requireThat } from "./core.mjs";

export function header(request, name) {
  const count = request.rawHeaders.filter((_, index) => index % 2 === 0 && request.rawHeaders[index].toLowerCase() === name).length;
  requireThat(count <= 1, "duplicate-authentication-header");
  const value = request.headers[name];
  requireThat(value === undefined || typeof value === "string", "invalid-authentication-header");
  return value;
}

export function authenticate(request, source, env, raw) {
  const secret = env[source.credentialEnv];
  requireThat(nonempty(secret), "source-credential-unavailable", 503);
  if (source.provider === "azure-devops") {
    requireThat(secret.includes(":") && equalSecret(header(request, "authorization") ?? "", `Basic ${Buffer.from(secret).toString("base64")}`), "invalid-provider-authentication", 401);
    return;
  }
  const name = { github: "x-hub-signature-256", jira: "x-hub-signature", linear: "linear-signature" }[source.provider];
  const supplied = header(request, name);
  const prefix = source.provider === "linear" ? "" : "sha256=";
  requireThat(typeof supplied === "string" && new RegExp(`^${prefix}[0-9a-fA-F]{64}$`).test(supplied), "invalid-provider-authentication", 401);
  const expected = createHmac("sha256", secret).update(raw).digest("hex");
  requireThat(equalSecret(supplied.slice(prefix.length).toLowerCase(), expected), "invalid-provider-authentication", 401);
}

export function deliveryIdentity(request, source, payload, now) {
  let id;
  let identityPayload = payload;
  if (source.provider === "github") id = header(request, "x-github-delivery");
  else if (source.provider === "jira") id = header(request, "x-atlassian-webhook-identifier") ?? `sha256:${hash(canonical(payload))}`;
  else if (source.provider === "linear") {
    requireThat(Number.isSafeInteger(payload.webhookTimestamp) && Math.abs(now - payload.webhookTimestamp) <= 60000, "stale-linear-delivery", 401);
    // webhookId identifies the configured webhook, not an original delivery.
    const { webhookTimestamp, ...stable } = payload;
    identityPayload = stable;
    requireThat(nonempty(payload.createdAt), "linear-event-time-required");
    id = `sha256:${hash(canonical(stable))}`;
  } else id = payload.id;
  requireThat(typeof id === "string" && id.trim() === id && id.length > 0 && Buffer.byteLength(id) <= 256 && !/[\u0000-\u001f\u007f]/.test(id), "original-delivery-id-required");
  return { id, digest: hash(canonical(identityPayload)) };
}
