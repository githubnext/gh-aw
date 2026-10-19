import { createPrivateKey, sign } from "node:crypto";
import { canonical, GatewayError, hash, nonempty, numericId, parseJSON, requireThat } from "./core.mjs";

export class DispatchError extends GatewayError {
  constructor(code, { retryable = false, uncertain = false, dispatched = false } = {}) {
    super(code, 502);
    this.retryable = retryable;
    this.uncertain = uncertain;
    this.dispatched = dispatched;
  }
}

export function appJWT(appId, privateKey, now) {
  const header = Buffer.from(JSON.stringify({ alg: "RS256", typ: "JWT" })).toString("base64url");
  const time = Math.floor(now / 1000);
  const payload = Buffer.from(JSON.stringify({ iat: time - 60, exp: time + 540, iss: appId })).toString("base64url");
  const key = createPrivateKey(privateKey);
  requireThat(key.asymmetricKeyType === "rsa" && key.asymmetricKeyDetails.modulusLength >= 2048, "invalid-app-signing-key", 503);
  const unsigned = `${header}.${payload}`;
  return `${unsigned}.${sign("RSA-SHA256", Buffer.from(unsigned), key).toString("base64url")}`;
}

async function boundedBody(response) {
  const reader = response.body?.getReader();
  const chunks = [];
  let size = 0;
  if (!reader) return Buffer.alloc(0);
  while (true) {
    const { value, done } = await reader.read();
    if (done) break;
    size += value.length;
    if (size > 2 * 1024 * 1024) {
      await reader.cancel();
      throw new DispatchError("github-response-too-large");
    }
    chunks.push(value);
  }
  return Buffer.concat(chunks);
}

export async function dispatchApproved({ github, destination, request, env, assertCurrent, now = Date.now, fetchImpl = fetch }) {
  let token;
  let outcome;
  let failure;
  let expiresAt;
  const api = async (method, path, credential, body, expected, dispatching = false, cleanup = false) => {
    if (!cleanup) assertCurrent();
    let response;
    try {
      response = await fetchImpl(`${github.apiUrl}${path}`, {
        method,
        headers: { authorization: `Bearer ${credential}`, accept: "application/vnd.github+json", "x-github-api-version": "2022-11-28", ...(body === undefined ? {} : { "content-type": "application/json" }) },
        ...(body === undefined ? {} : { body: JSON.stringify(body) }),
        redirect: "manual",
        signal: AbortSignal.timeout(15000),
      });
    } catch {
      throw new DispatchError(dispatching ? "dispatch-outcome-unknown" : "github-transport-failed", { uncertain: dispatching, retryable: !dispatching });
    }
    if (response.status !== expected) {
      await response.body?.cancel();
      // A GitHub 5xx may follow a committed dispatch; only 429 is a safe send retry.
      throw new DispatchError(dispatching ? "dispatch-http-failed" : "github-http-failed", { retryable: response.status === 429 || (!dispatching && response.status >= 500), uncertain: dispatching && response.status >= 500 });
    }
    if (expected === 204) {
      await response.body?.cancel();
      return undefined;
    }
    try {
      return parseJSON(await boundedBody(response));
    } catch (error) {
      if (error instanceof DispatchError) throw error;
      throw new DispatchError("invalid-github-response");
    }
  };
  try {
    assertCurrent();
    const privateKey = env[github.privateKeyEnv];
    requireThat(nonempty(privateKey), "app-key-unavailable", 503);
    let jwt;
    try {
      jwt = appJWT(github.appId, privateKey, now());
    } catch {
      throw new DispatchError("app-jwt-signing-failed");
    }
    const repoPath = `/repos/${destination.repository}`;
    const installation = await api("GET", `${repoPath}/installation`, jwt, undefined, 200);
    requireThat(
      numericId(installation.id) === destination.installationId &&
        numericId(installation.app_id) === github.appId &&
        installation.suspended_at === null &&
        numericId(installation.account?.id) === destination.accountId &&
        installation.account.login === destination.accountLogin &&
        canonical(installation.permissions) === canonical(destination.installationPermissions),
      "installation-binding-mismatch",
      403
    );
    requireThat(Number.isSafeInteger(Number(destination.repositoryId)) && Number(destination.repositoryId) > 0, "unsupported-repository-id-range", 422);
    const minted = await api("POST", `/app/installations/${destination.installationId}/access_tokens`, jwt, { repository_ids: [Number(destination.repositoryId)], permissions: { contents: "write" } }, 201);
    if (nonempty(minted.token)) token = minted.token;
    requireThat(token, "missing-installation-token", 502);
    const permissions = minted.permissions;
    requireThat(permissions?.contents === "write" && Object.keys(permissions).every(key => key === "contents" || (key === "metadata" && permissions.metadata === "read")), "excessive-installation-token-permissions", 403);
    requireThat(
      minted.repository_selection === "selected" &&
        Array.isArray(minted.repositories) &&
        minted.repositories.length === 1 &&
        numericId(minted.repositories[0].id) === destination.repositoryId &&
        minted.repositories[0].full_name === destination.repository,
      "installation-token-scope-mismatch",
      403
    );
    expiresAt = Date.parse(minted.expires_at);
    requireThat(typeof minted.expires_at === "string" && Number.isFinite(expiresAt) && expiresAt > now() + 5000 && expiresAt <= now() + 3660000, "installation-token-expiry-invalid", 403);
    const repository = await api("GET", repoPath, token, undefined, 200);
    requireThat(
      numericId(repository.id) === destination.repositoryId && repository.full_name === destination.repository && repository.default_branch === destination.defaultBranch && numericId(repository.owner?.id) === destination.accountId,
      "destination-repository-mismatch",
      403
    );
    const revision = await api("GET", `${repoPath}/commits/${encodeURIComponent(destination.defaultBranch)}`, token, undefined, 200);
    requireThat(revision.sha === destination.workflowRevision, "destination-revision-changed", 403);
    const path = destination.subscriptions[0].dispatch.workflow;
    const artifact = await api("GET", `${repoPath}/contents/${path}?ref=${destination.workflowRevision}`, token, undefined, 200);
    requireThat(artifact.encoding === "base64" && typeof artifact.content === "string", "missing-approved-workflow", 403);
    const bytes = Buffer.from(artifact.content, "base64");
    requireThat(hash(bytes) === destination.artifactSha256, "destination-artifact-changed", 403);
    const manifests = bytes
      .toString("utf8")
      .split("\n")
      .filter(line => line.startsWith("# gh-aw-manifest: "));
    requireThat(manifests.length === 1, "ambiguous-approved-manifest", 403);
    const manifest = parseJSON(Buffer.from(manifests[0].slice("# gh-aw-manifest: ".length)));
    requireThat(canonical(manifest.webhook_subscriptions) === canonical(destination.subscriptions), "approved-manifest-subscriptions-mismatch", 403);
    assertCurrent();
    requireThat(now() < expiresAt, "installation-token-expired", 403);
    await api("POST", `${repoPath}/dispatches`, token, request, 204, true);
    outcome = { dispatched: true };
  } catch (error) {
    failure = error instanceof GatewayError ? error : new DispatchError("github-operation-failed");
  } finally {
    if (token) {
      try {
        await api("DELETE", "/installation/token", token, undefined, 204, false, true);
      } catch {
        failure = new DispatchError("installation-token-revocation-failed", { dispatched: outcome?.dispatched === true, uncertain: failure?.uncertain === true });
      }
      token = undefined;
    }
  }
  if (failure) throw failure;
  return outcome;
}
