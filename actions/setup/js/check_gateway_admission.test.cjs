import { afterEach, describe, expect, it, vi } from "vitest";
import { createHash } from "node:crypto";
import { once } from "node:events";
import { createServer } from "node:http";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { main, verifyGatewayAdmission } = require("./check_gateway_admission.cjs");
const revision = "a".repeat(40);
const workflow = Buffer.from("# gh-aw header\r\nname: Version check\n# café\n", "utf8");

function hash(bytes) {
  return createHash("sha256").update(bytes).digest("hex");
}

function fixture(bytes = workflow) {
  const env = {
    GH_AW_GATEWAY_GET_TOKEN_URL: "https://gateway.example/getToken",
    GH_AW_GATEWAY_OIDC_AUDIENCE: "https://gateway.example",
    GH_AW_GATEWAY_REPOSITORY: "owner/repository",
    GH_AW_GATEWAY_REPOSITORY_ID: "42",
    GH_AW_GATEWAY_OWNER_ID: "7",
    GH_AW_GATEWAY_WORKFLOW_PATH: ".github/workflows/version-check.lock.yml",
    GH_AW_GATEWAY_WORKFLOW_REF: "owner/repository/.github/workflows/version-check.lock.yml@refs/heads/main",
    GH_AW_GATEWAY_WORKFLOW_SHA: revision,
    GH_AW_GATEWAY_RUN_ID: "101",
    GH_AW_GATEWAY_RUN_ATTEMPT: "2",
    GH_AW_GATEWAY_ACTOR_ID: "99",
    GH_AW_GATEWAY_REF: "refs/heads/main",
    GH_AW_GATEWAY_EVENT_NAME: "repository_dispatch",
    GH_AW_GATEWAY_EVENT_TYPE: `gh-aw-webhook-${hash("version-check").slice(0, 24)}`,
    GITHUB_API_URL: "https://api.github.com",
    // Neither the checked-out revision nor payload-owned data is an authority.
    GITHUB_SHA: "b".repeat(40),
    GITHUB_WORKSPACE: "/untrusted-checkout",
    GITHUB_EVENT_PATH: "/untrusted-payload",
  };
  const claims = {
    repository: env.GH_AW_GATEWAY_REPOSITORY,
    repository_id: env.GH_AW_GATEWAY_REPOSITORY_ID,
    repository_owner_id: env.GH_AW_GATEWAY_OWNER_ID,
    workflow_ref: env.GH_AW_GATEWAY_WORKFLOW_REF,
    workflow_sha: revision,
    run_id: env.GH_AW_GATEWAY_RUN_ID,
    run_attempt: env.GH_AW_GATEWAY_RUN_ATTEMPT,
    actor_id: env.GH_AW_GATEWAY_ACTOR_ID,
    ref: env.GH_AW_GATEWAY_REF,
    event_name: env.GH_AW_GATEWAY_EVENT_NAME,
    check_run_id: "123",
    exp: Math.floor(Date.now() / 1000) + 300,
    nbf: Math.floor(Date.now() / 1000) - 30,
  };
  const core = {
    getInput: vi.fn(() => "github-secret"),
    getIDToken: vi.fn(async audience => `${Buffer.from('{"alg":"RS256"}').toString("base64url")}.${Buffer.from(JSON.stringify({ ...claims, aud: audience })).toString("base64url")}.signature`),
    setSecret: vi.fn(),
    setOutput: vi.fn(),
    setFailed: vi.fn(),
    info: vi.fn(),
  };
  const fetchImpl = vi.fn(async (url, options) => {
    if (url === `${env.GITHUB_API_URL}/repos/owner/repository`) return Response.json({ id: 42, full_name: env.GH_AW_GATEWAY_REPOSITORY });
    if (url === `${env.GITHUB_API_URL}/repos/owner/repository/contents/${env.GH_AW_GATEWAY_WORKFLOW_PATH}?ref=${revision}`) return new Response(bytes);
    if (url === env.GH_AW_GATEWAY_GET_TOKEN_URL) return Response.json({ ...JSON.parse(options.body), allowed: true });
    throw new Error("unexpected credential-bearing request: github-secret");
  });
  return { env, core, claims, fetchImpl, run: () => verifyGatewayAdmission(core, env, fetchImpl) };
}

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("hash-bound gateway workflow admission", () => {
  it("hashes exact immutable Git bytes, binds OIDC and the closed POST, and never forwards repository credentials", async () => {
    const f = fixture();
    await expect(f.run()).resolves.toEqual({ verified: true, workflow_hash: hash(workflow) });
    const audience = `https://gateway.example/gh-aw/gateway/v1/workflow/sha256/${hash(workflow)}`;
    expect(f.core.getIDToken).toHaveBeenCalledExactlyOnceWith(audience);
    expect(f.fetchImpl).toHaveBeenCalledTimes(3);
    const [url, options] = f.fetchImpl.mock.calls[1];
    expect(url).toBe(`https://api.github.com/repos/owner/repository/contents/.github/workflows/version-check.lock.yml?ref=${revision}`);
    expect(options.headers).toEqual({ Authorization: "Bearer github-secret", Accept: "application/vnd.github.raw+json" });
    for (const [, request] of f.fetchImpl.mock.calls) {
      expect(request.redirect).toBe("error");
      expect(request.cache).toBe("no-store");
      expect(request.signal).toBeInstanceOf(AbortSignal);
    }
    const [endpoint, request] = f.fetchImpl.mock.calls[2];
    expect(endpoint).toBe(f.env.GH_AW_GATEWAY_GET_TOKEN_URL);
    expect(request.method).toBe("POST");
    const token = await f.core.getIDToken.mock.results[0].value;
    expect(request.headers.Authorization).toBe(`Bearer ${token}`);
    expect(f.core.setSecret).toHaveBeenCalledWith(token);
    expect(f.core.setSecret).toHaveBeenCalledWith("github-secret");
    expect(request.body).not.toContain("github-secret");
    expect(request.body).not.toContain(token);
    expect(JSON.parse(request.body)).toEqual({
      schema_version: 1,
      repository: "owner/repository",
      repository_id: "42",
      repository_owner_id: "7",
      workflow_path: ".github/workflows/version-check.lock.yml",
      workflow_ref: f.env.GH_AW_GATEWAY_WORKFLOW_REF,
      workflow_sha: revision,
      run_id: "101",
      run_attempt: "2",
      actor_id: "99",
      ref: "refs/heads/main",
      event_name: "repository_dispatch",
      event_type: f.env.GH_AW_GATEWAY_EVENT_TYPE,
      workflow_hash: hash(workflow),
      check_run_id: "123",
    });
  });

  it.each([
    Buffer.from(workflow.toString().replace("# gh-aw header", "# changed header")),
    Buffer.from(workflow.toString().replace("\r\n", "\n")),
    Buffer.from(workflow.toString().replace("Version check", "Changed body")),
    Buffer.concat([Buffer.from([0xef, 0xbb, 0xbf]), workflow]),
    Buffer.concat([workflow, Buffer.from([0xff])]),
  ])("does not normalize or exclude any compiled bytes", async bytes => {
    const f = fixture(bytes);
    const result = await f.run();
    expect(result.workflow_hash).toBe(hash(bytes));
    expect(result.workflow_hash).not.toBe(hash(workflow));
    expect(f.core.getIDToken).toHaveBeenCalledWith(expect.stringContaining(hash(bytes)));
  });

  it("preserves a GitHub Enterprise API prefix and a path-qualified audience", async () => {
    const f = fixture();
    f.env.GITHUB_API_URL = "https://github.example/api/v3";
    f.env.GH_AW_GATEWAY_OIDC_AUDIENCE = "https://gateway.example/tenant";
    await f.run();
    expect(f.fetchImpl.mock.calls[1][0]).toContain("https://github.example/api/v3/repos/");
    expect(f.core.getIDToken).toHaveBeenCalledWith(`https://gateway.example/tenant/gh-aw/gateway/v1/workflow/sha256/${hash(workflow)}`);
  });

  it.each([
    ["GH_AW_GATEWAY_GET_TOKEN_URL", ""],
    ["GH_AW_GATEWAY_GET_TOKEN_URL", "http://gateway.example/getToken"],
    ["GH_AW_GATEWAY_GET_TOKEN_URL", "https://user:password@gateway.example/getToken"],
    ["GH_AW_GATEWAY_GET_TOKEN_URL", "https://gateway.example/getToken?"],
    ["GH_AW_GATEWAY_GET_TOKEN_URL", "https://gateway.example/getToken#"],
    ["GH_AW_GATEWAY_GET_TOKEN_URL", "https://GATEWAY.example/getToken"],
    ["GH_AW_GATEWAY_GET_TOKEN_URL", "https://gateway.example:443/getToken"],
    ["GH_AW_GATEWAY_OIDC_AUDIENCE", "https://gateway.example/"],
    ["GH_AW_GATEWAY_OIDC_AUDIENCE", "https://gateway.example/tenant/"],
    ["GH_AW_GATEWAY_OIDC_AUDIENCE", "https://gateway.example/tenant/../other"],
    ["GITHUB_API_URL", "http://api.github.com"],
    ["GH_AW_GATEWAY_REPOSITORY_ID", "042"],
    ["GH_AW_GATEWAY_OWNER_ID", "0"],
    ["GH_AW_GATEWAY_RUN_ID", ""],
    ["GH_AW_GATEWAY_RUN_ATTEMPT", "1e2"],
    ["GH_AW_GATEWAY_ACTOR_ID", "-1"],
    ["GH_AW_GATEWAY_WORKFLOW_SHA", "main"],
    ["GH_AW_GATEWAY_WORKFLOW_PATH", ".github/workflows/../other.lock.yml"],
    ["GH_AW_GATEWAY_WORKFLOW_REF", "other/repo/.github/workflows/version-check.lock.yml@refs/heads/main"],
    ["GH_AW_GATEWAY_EVENT_NAME", "workflow_dispatch"],
    ["GH_AW_GATEWAY_EVENT_TYPE", "forged"],
    ["GH_AW_GATEWAY_EVENT_TYPE", `gh-aw-webhook-${"0".repeat(24)}`],
    ["GH_AW_GATEWAY_REF", "refs/pull/1/merge"],
  ])("rejects invalid configuration/context %s before any credentialed request", async (name, value) => {
    const f = fixture();
    f.env[name] = value;
    await expect(f.run()).rejects.toThrow(/ERR_CONFIG|ERR_PERMISSION/);
    expect(f.fetchImpl).not.toHaveBeenCalled();
    expect(f.core.getIDToken).not.toHaveBeenCalled();
  });

  it.each([
    { id: 43, full_name: "owner/repository" },
    { id: 42, full_name: "other/repository" },
    { id: 9007199254740992, full_name: "owner/repository" },
    { id: "42", full_name: "owner/repository" },
  ])("rejects a mismatched or unsafe authenticated repository identity", async repository => {
    const f = fixture();
    f.fetchImpl.mockResolvedValueOnce(Response.json(repository));
    await expect(f.run()).rejects.toThrow("authenticated repository lookup");
    expect(f.fetchImpl).toHaveBeenCalledOnce();
    expect(f.core.getIDToken).not.toHaveBeenCalled();
  });

  it.each(["aud", "repository", "repository_id", "repository_owner_id", "workflow_ref", "workflow_sha", "run_id", "run_attempt", "actor_id", "ref", "event_name", "check_run_id", "exp", "nbf"])(
    "rejects an inconsistent or absent issued %s claim without sending the JWT to the backend",
    async field => {
      const f = fixture();
      if (field === "aud") f.core.getIDToken.mockImplementationOnce(async () => "e30.eyJhdWQiOiJ3cm9uZyJ9.signature");
      else delete f.claims[field];
      await expect(f.run()).rejects.toThrow("issued OIDC audience and workflow/run claims");
      expect(f.fetchImpl).toHaveBeenCalledTimes(2);
    }
  );

  it.each(["", "not-a-jwt", "a".repeat(32769), "e30.W10.signature"])("rejects a malformed or oversized JWT", async token => {
    const f = fixture();
    f.core.getIDToken.mockResolvedValueOnce(token);
    await expect(f.run()).rejects.toThrow(/OIDC|JSON object/);
    expect(f.fetchImpl).toHaveBeenCalledTimes(2);
  });

  it.each(["expired", "future"])("rejects %s issued JWT validity without imposing a renewal policy", async validity => {
    const f = fixture();
    if (validity === "expired") f.claims.exp = Math.floor(Date.now() / 1000) - 1;
    else f.claims.nbf = Math.floor(Date.now() / 1000) + 60;
    await expect(f.run()).rejects.toThrow("issued OIDC");
  });

  it.each(["schema_version", "repository", "repository_id", "repository_owner_id", "workflow_path", "workflow_ref", "workflow_sha", "workflow_hash", "run_id", "run_attempt", "check_run_id", "actor_id", "ref", "event_name", "event_type"])(
    "requires exact %s correlation in the explicit backend decision",
    async field => {
      const f = fixture();
      f.fetchImpl.mockImplementationOnce(async () => Response.json({ id: 42, full_name: "owner/repository" }));
      f.fetchImpl.mockImplementationOnce(async () => new Response(workflow));
      f.fetchImpl.mockImplementationOnce(async (_url, options) => Response.json({ ...JSON.parse(options.body), allowed: true, [field]: "different" }));
      await expect(f.run()).rejects.toThrow("exact closed workflow/run binding");
    }
  );

  it.each([{ allowed: true }, { allowed: false }, { allowed: "true" }, { allowed: true, access_token: "must-not-be-returned" }, { allowed: true, pending: true }, [], null])(
    "rejects unbound, denied, deferred, or credential-bearing success responses",
    async decision => {
      const f = fixture();
      f.fetchImpl.mockImplementationOnce(async () => Response.json({ id: 42, full_name: "owner/repository" }));
      f.fetchImpl.mockImplementationOnce(async () => new Response(workflow));
      f.fetchImpl.mockImplementationOnce(async (_url, options) => Response.json(decision?.access_token || decision?.pending || typeof decision?.allowed === "string" ? { ...JSON.parse(options.body), ...decision } : decision));
      await expect(f.run()).rejects.toThrow(/closed workflow\/run|JSON object/);
    }
  );

  it.each([202, 204, 301, 307, 400, 401, 403, 409, 410, 429, 500, 503])("keeps admission blocked on HTTP %i without automatic retries or forwarding error bodies", async status => {
    const f = fixture();
    f.fetchImpl.mockImplementationOnce(async () => Response.json({ id: 42, full_name: "owner/repository" }));
    f.fetchImpl.mockImplementationOnce(async () => new Response(workflow));
    f.fetchImpl.mockResolvedValueOnce(new Response(status === 204 ? null : "secret-server-detail", { status, headers: { Location: "https://other.example" } }));
    await expect(f.run()).rejects.toThrow(`Gateway getToken admission failed (HTTP ${status})`);
    expect(f.fetchImpl).toHaveBeenCalledTimes(3);
  });

  it.each(["workflow-limit", "decision-limit", "declared-length", "invalid-json", "invalid-utf8", "empty-workflow"])("enforces streaming byte limits and parsing on %s", async failure => {
    const f = fixture(failure === "workflow-limit" ? Buffer.alloc(1048577) : failure === "empty-workflow" ? Buffer.alloc(0) : workflow);
    if (failure !== "workflow-limit" && failure !== "empty-workflow") {
      f.fetchImpl.mockImplementationOnce(async () => Response.json({ id: 42, full_name: "owner/repository" }));
      f.fetchImpl.mockImplementationOnce(async () => new Response(workflow));
      f.fetchImpl.mockResolvedValueOnce(
        failure === "decision-limit"
          ? new Response(Buffer.alloc(16385))
          : failure === "declared-length"
            ? new Response("{}", { headers: { "content-length": "16385" } })
            : failure === "invalid-utf8"
              ? new Response(Buffer.from([0xff]))
              : new Response("{")
      );
    }
    await expect(f.run()).rejects.toThrow(/limits|UTF-8 JSON|must not be empty/);
  });

  it("cancels an oversized streaming response and redacts transport/issuance errors", async () => {
    const f = fixture();
    const cancel = vi.fn();
    f.fetchImpl.mockResolvedValueOnce(
      new Response(
        new ReadableStream({
          pull(controller) {
            controller.enqueue(new Uint8Array(1048577));
          },
          cancel,
        })
      )
    );
    await expect(f.run()).rejects.toThrow("response limits");
    expect(cancel).toHaveBeenCalledOnce();
    for (const phase of ["github", "oidc", "backend"]) {
      const g = fixture();
      if (phase === "oidc") g.core.getIDToken.mockRejectedValueOnce(new Error("secret-id-token"));
      else {
        if (phase === "backend") {
          g.fetchImpl.mockImplementationOnce(async () => Response.json({ id: 42, full_name: "owner/repository" }));
          g.fetchImpl.mockImplementationOnce(async () => new Response(workflow));
        }
        g.fetchImpl.mockRejectedValueOnce(new Error("secret-id-token github-secret"));
      }
      await expect(g.run()).rejects.not.toThrow("secret-");
    }
  });

  it("applies the request abort deadline through streamed body reads", async () => {
    const controller = new AbortController();
    const timeout = vi.spyOn(AbortSignal, "timeout").mockReturnValue(controller.signal);
    const f = fixture();
    let signal;
    f.fetchImpl.mockImplementationOnce(async (_url, options) => {
      signal = options.signal;
      return new Response(
        new ReadableStream({
          start(controller) {
            signal.addEventListener("abort", () => controller.error(new Error("secret-deadline-detail")), { once: true });
          },
        })
      );
    });
    const result = expect(f.run()).rejects.toThrow("admission remains blocked");
    await vi.waitFor(() => expect(signal).toBeInstanceOf(AbortSignal));
    expect(timeout).toHaveBeenCalledExactlyOnceWith(30000);
    controller.abort();
    await result;
  });

  it("accepts the exact workflow/decision byte ceilings without an off-by-one rejection", async () => {
    const bytes = Buffer.alloc(1048576, 35);
    const f = fixture(bytes);
    f.fetchImpl.mockImplementationOnce(async () => Response.json({ id: 42, full_name: "owner/repository" }));
    f.fetchImpl.mockImplementationOnce(async () => new Response(bytes));
    f.fetchImpl.mockImplementationOnce(async (_url, options) => new Response(JSON.stringify({ ...JSON.parse(options.body), allowed: true }).padEnd(16384, " ")));
    await expect(f.run()).resolves.toEqual({ verified: true, workflow_hash: hash(bytes) });
  });

  it("bounds the serialized outbound UTF-8 request before presenting the JWT", async () => {
    const f = fixture();
    f.env.GH_AW_GATEWAY_REF = `refs/heads/${"é".repeat(4300)}`;
    f.env.GH_AW_GATEWAY_WORKFLOW_REF = `owner/repository/${f.env.GH_AW_GATEWAY_WORKFLOW_PATH}@${f.env.GH_AW_GATEWAY_REF}`;
    f.claims.ref = f.env.GH_AW_GATEWAY_REF;
    f.claims.workflow_ref = f.env.GH_AW_GATEWAY_WORKFLOW_REF;
    await expect(f.run()).rejects.toThrow("request exceeds its UTF-8 byte limit");
    expect(f.fetchImpl).toHaveBeenCalledTimes(2);
  });
});

describe("gateway admission action entrypoint", () => {
  it.each([true, false])("publishes verified=true only for a complete accepted admission (%s)", async accepted => {
    const f = fixture();
    for (const [key, value] of Object.entries(f.env)) vi.stubEnv(key, value);
    vi.stubGlobal("core", f.core);
    vi.stubGlobal("fetch", f.fetchImpl);
    if (!accepted) f.core.getIDToken.mockRejectedValueOnce(new Error("secret-id-token"));
    await main();
    expect(f.core.setOutput.mock.calls).toEqual(
      accepted
        ? [
            ["verified", "false"],
            ["verified", "true"],
          ]
        : [["verified", "false"]]
    );
    expect(f.core.setFailed).toHaveBeenCalledTimes(accepted ? 0 : 1);
    expect(JSON.stringify([...f.core.setOutput.mock.calls, ...f.core.setFailed.mock.calls, ...f.core.info.mock.calls])).not.toContain("secret-");
  });

  describe("gateway admission HTTP interoperability", () => {
    it.each([200, 307, 503])("exercises actual loopback HTTP serialization and response handling for HTTP %i", async status => {
      const f = fixture();
      const requests = [];
      const server = createServer(async (request, response) => {
        const chunks = [];
        for await (const chunk of request) chunks.push(chunk);
        const body = Buffer.concat(chunks).toString("utf8");
        requests.push({ url: request.url, headers: request.headers, body });
        if (request.url === "/repos/owner/repository") {
          response.setHeader("Content-Type", "application/json");
          response.end(JSON.stringify({ id: 42, full_name: "owner/repository" }));
        } else if (request.url === `/repos/owner/repository/contents/.github/workflows/version-check.lock.yml?ref=${revision}`) {
          response.setHeader("Content-Type", "application/vnd.github.raw+json");
          response.end(workflow);
        } else if (request.url === "/getToken") {
          response.writeHead(status, { "Content-Type": "application/json", Location: "/stolen" });
          response.end(JSON.stringify({ ...JSON.parse(body), allowed: true }));
        } else {
          response.writeHead(404);
          response.end();
        }
      });
      server.listen(0, "127.0.0.1");
      await once(server, "listening");
      const address = server.address();
      const transport = async (url, options) => {
        const original = new URL(url);
        expect(["api.github.com", "gateway.example"]).toContain(original.host);
        expect(original.protocol).toBe("https:");
        // Only the test adapter maps the approved HTTPS destinations to loopback.
        return fetch(`http://127.0.0.1:${address.port}${original.pathname}${original.search}`, options);
      };
      try {
        const result = verifyGatewayAdmission(f.core, f.env, transport);
        if (status === 200) await expect(result).resolves.toEqual({ verified: true, workflow_hash: hash(workflow) });
        else await expect(result).rejects.toThrow("Gateway getToken admission failed");
        expect(requests).toHaveLength(3);
        expect(requests[1].headers.accept).toBe("application/vnd.github.raw+json");
        expect(requests[1].headers.authorization).toBe("Bearer github-secret");
        expect(requests[2].headers.authorization).toBe(`Bearer ${await f.core.getIDToken.mock.results[0].value}`);
        expect(requests[2].headers["content-type"]).toBe("application/json");
        expect(JSON.parse(requests[2].body).workflow_hash).toBe(hash(workflow));
        expect(JSON.parse(requests[2].body).run_attempt).toBe("2");
        expect(requests[2].body).not.toContain("github-secret");
      } finally {
        await new Promise((resolve, reject) => server.close(error => (error ? reject(error) : resolve())));
      }
    });
  });
});
