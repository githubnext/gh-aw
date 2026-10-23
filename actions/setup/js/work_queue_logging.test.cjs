import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { createWorkQueueLogger, debugEnabled, queueControlErrorCode, queueControlErrorExplanation } = require("./work_queue_logging.cjs");
const { fakeGitHub, options } = require("./work_queue_store_checks.cjs");
const { genesis, submission, producer } = require("./work_queue_test_helpers.cjs");
const { publishWorkQueueRequest } = require("./work_queue_store.cjs");
const { planDispatch } = require("./work_queue_scheduler.cjs");
const { replayTransactions } = require("./work_queue_replay.cjs");
const { performance } = require("node:perf_hooks");
const { resolveDispatchCredentialPrincipal } = require("./work_queue_dispatch_credential.cjs");
const { inspectClaimDelivery, createClaimDeliveryVerifier, verifyClaimDelivery } = require("./work_queue_delivery.cjs");
const { withProjectionLocks } = require("./work_queue_issue_coordination.cjs");
const { registerQueueLedgerClient } = require("./work_queue_git_transport.cjs");

describe("privacy-preserving work queue logging", () => {
  let stderr;
  const output = () => stderr.mock.calls.map(call => call[0]).join("");

  beforeEach(() => {
    vi.stubEnv("DEBUG", "");
    vi.stubEnv("ACTIONS_RUNNER_DEBUG", "");
    vi.stubEnv("RUNNER_DEBUG", "");
    stderr = vi.spyOn(process.stderr, "write").mockReturnValue(true);
  });
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
  });

  it("is silent by default, including failures", () => {
    const logger = createWorkQueueLogger("store");
    logger.debug("request.publish.start", { attempts: 1 });
    logger.failure("request.publish.failed", new Error("secret"));
    expect(stderr).not.toHaveBeenCalled();
  });

  it("explains a delivery contract cardinality failure using only counts", async () => {
    vi.stubEnv("DEBUG", "work-queue:delivery");
    const assignment = {
      version: 3,
      dispatch_id: "private-dispatch",
      request_id: "private-request",
      commit_id: "private-commit",
      policy_epoch: "private-policy",
      pool: "private-pool",
      worker_profile: "private-profile",
      claims: [{ handle: "private-handle", claim_id: "private-claim", work_id: "private-work", result_refs: [], work: { effect_contract: { version: 1, outputs: [{ type: "private-output", min: 1, max: 2 }] } } }],
    };
    const member = assignment.claims[0];
    const delivery = await inspectClaimDelivery({ assignment, claim_handle: member.handle, authorize: async () => ({ authorized: true, claim_handle: member.handle }), messages: [], results: [] });
    expect(delivery.verification).toBe("unknown");
    expect(output()).toContain("delivery.contract.cardinality_mismatch index=0 count=0 min=1 max=2");
    expect(output()).not.toContain("private-");
  });

  it("distinguishes missing protected context from invalid bindings and unattested receipts", async () => {
    vi.stubEnv("DEBUG", "work-queue:delivery");
    expect(await verifyClaimDelivery({}, {})).toEqual({ verified: false, effects: "unknown" });
    expect(output()).toContain("delivery.verify.trusted_context_missing");
    const assignment = {
      version: 3,
      dispatch_id: "private-dispatch",
      request_id: "private-request",
      commit_id: "private-commit",
      policy_epoch: "private-policy",
      pool: "private-pool",
      worker_profile: "private-profile",
      claims: [{ handle: "private-handle", claim_id: "private-claim", work_id: "private-work", result_refs: [], work: { effect_contract: { kind: "none" } } }],
    };
    const member = assignment.claims[0];
    const verify = createClaimDeliveryVerifier({ assignment, recheck: async () => ({ verification: "verified", private_data: "private-token" }) });
    const verification = { assignment, contract: member.work.effect_contract, attempt: 1, run: { run_id: "42", run_attempt: 1 } };
    expect(await verify(member, { ...verification, attempt: 0 })).toEqual({ verified: false, effects: "unknown" });
    expect(output()).toContain("delivery.verify.binding_invalid original_claim=true contract_present=true attempt=0");
    expect(await verify(member, verification)).toEqual({ verified: false, effects: "unknown" });
    expect(output()).toContain("delivery.verify.receipt_rejected receipt_present=false readback_verified=true");
    expect(output()).not.toContain("private-");
  });

  it.each([
    ["work-queue:*", true, true],
    ["work-queue:store", true, false],
    ["work-queue:store work-queue:dispatch", true, false],
    ["work-queue:store,work-queue:replay", true, true],
    ["*,-work-queue:replay", true, false],
    ["-work-queue:replay,*", true, false],
    ["other:*", false, false],
    ["work-queue:st.re", false, false],
  ])("filters namespaces with DEBUG=%s", (pattern, store, replay) => {
    vi.stubEnv("DEBUG", pattern);
    expect(debugEnabled("work-queue:store")).toBe(store);
    expect(debugEnabled("work-queue:replay")).toBe(replay);
  });

  it.each([
    ["ACTIONS_RUNNER_DEBUG", "true"],
    ["RUNNER_DEBUG", "1"],
  ])("supports %s=%s", (name, value) => {
    vi.stubEnv("DEBUG", "-work-queue:*");
    vi.stubEnv(name, value);
    createWorkQueueLogger("store").debug("enabled");
    expect(output()).toContain("[work-queue:store] enabled");
  });

  it("logs timestamps, counts and flags without serializing sensitive data", () => {
    vi.stubEnv("DEBUG", "work-queue:*");
    const secret = "private-token-payload";
    const toString = vi.fn(() => secret);
    createWorkQueueLogger("store").debug("candidate.checked", {
      operations: 2,
      reused: false,
      payload: secret,
      token: secret,
      nested: { secret },
      object: { toString },
      array: [secret],
      invalid: NaN,
      infinite: Infinity,
      negative: -1,
      fractional: 0.5,
      huge: Number.MAX_SAFE_INTEGER + 1,
    });
    expect(output()).toMatch(/\[\d{4}-\d{2}-\d{2}T.*Z\] \[work-queue:store\] candidate.checked operations=2 reused=false \+\d+ms\n/);
    expect(output()).not.toContain(secret);
    expect(toString).not.toHaveBeenCalled();
  });

  it("measures monotonic deltas per namespace across logger instances and emission types", () => {
    vi.stubEnv("DEBUG", "work-queue:*");
    const clock = vi.spyOn(performance, "now");
    clock.mockReturnValue(100);
    const first = createWorkQueueLogger("timing");
    first.debug("first");
    clock.mockReturnValue(125);
    createWorkQueueLogger("other-timing").diagnostic("independent");
    clock.mockReturnValue(142);
    createWorkQueueLogger("timing").diagnostic("second");
    clock.mockReturnValue(150);
    first.failure("third", { status: 503 }, true);
    expect(output()).toContain("[work-queue:timing] first +0ms\n");
    expect(output()).toContain("[work-queue:other-timing] independent +0ms\n");
    expect(output()).toContain("[work-queue:timing] second +42ms\n");
    expect(output()).toMatch(/\[work-queue:timing\] third failed=true http_status=503 queue_control_github_api_unavailable=true -- .* \+8ms\n/);
  });

  it("does not advance the delta clock for suppressed events or use wall-clock time", () => {
    const clock = vi.spyOn(performance, "now").mockReturnValue(100);
    const logger = createWorkQueueLogger("suppressed-timing");
    logger.diagnostic("first");
    clock.mockReturnValue(110);
    logger.debug("suppressed");
    logger.failure("suppressed.failure", new Error("private"));
    vi.spyOn(Date, "now").mockReturnValue(-1000);
    clock.mockReturnValue(130);
    logger.failure("visible.failure", { status: 500 }, true);
    expect(output()).toContain("first +0ms\n");
    expect(output()).toMatch(/visible.failure failed=true http_status=500 queue_control_github_api_unavailable=true -- .* \+30ms\n/);
    expect(output()).not.toContain("suppressed.failure");
  });

  it("does not log error messages, stack traces, causes, headers or responses", () => {
    vi.stubEnv("DEBUG", "work-queue:*");
    const secret = "private-error-request-body";
    const error = Object.assign(new Error(secret, { cause: new Error(secret) }), {
      status: 502,
      code: secret,
      response: { data: secret, headers: { authorization: secret } },
    });
    createWorkQueueLogger("store").failure("publication.failed", error);
    expect(output()).toContain("publication.failed failed=true http_status=502");
    expect(output()).not.toContain(secret);
    expect(output()).not.toContain("Stack");
  });

  it("distinguishes request and transport metadata without logging URLs, bodies or credentials", () => {
    createWorkQueueLogger("native").failure(
      "repository.read.failed",
      {
        status: 500,
        request: { method: "GET", url: "https://private-host/private-path", headers: { authorization: "private-token" } },
        response: { status: 500, headers: {}, data: "private-body" },
        cause: { code: "UND_ERR_SOCKET", message: "private-cause" },
      },
      true
    );
    expect(output()).toContain("http_method_get=true");
    expect(output()).toContain("request_present=true response_present=true response_headers_present=true github_request_id_present=false");
    expect(output()).toContain("transport_und_err_socket=true");
    expect(output()).toContain("GitHub request ID: unknown.");
    expect(output()).not.toContain("private-");
  });

  it("does not invoke request or cause getters, including cyclic transport errors", () => {
    const getter = vi.fn(() => {
      throw new Error("private-getter");
    });
    const failure = Object.defineProperties({ status: 500 }, { request: { get: getter }, cause: { get: getter } });
    createWorkQueueLogger("native").failure("repository.read.failed", failure, true);
    const cyclic = { status: 500, code: "ECONNRESET", cause: null };
    cyclic.cause = cyclic;
    createWorkQueueLogger("native").failure("repository.read.failed", cyclic, true);
    expect(getter).not.toHaveBeenCalled();
    expect(output()).toContain("transport_econnreset=true");
    expect(output()).not.toContain("private");
  });

  it("emits explicitly requested failure diagnostics without DEBUG, exposing only status and fixed categories", () => {
    const secret = "private-token-and-assignment";
    const logger = createWorkQueueLogger("dispatch");
    for (const message of [`No ref found for: ${secret}`, `Unexpected inputs provided: ${secret}`, "Workflow does not have 'workflow_dispatch' trigger", secret]) {
      logger.failure("launch.post.failed", { status: 422, response: { data: { message, errors: [secret] }, headers: { authorization: secret } } }, true);
    }
    expect(output()).toContain("http_status=422");
    expect(output()).toContain("ref_not_found=true");
    expect(output()).toContain("unexpected_inputs=true");
    expect(output()).toContain("dispatch_trigger_missing=true");
    expect(output()).not.toContain(secret);
  });

  it("prints only numeric and boolean metadata in always-visible diagnostics", () => {
    createWorkQueueLogger("native").diagnostic("dispatch.post.start", { input_count: 1, named_ref: true, assignment: "private", token: "private", ref: "private" });
    expect(output()).toContain("dispatch.post.start input_count=1 named_ref=true");
    expect(output()).not.toContain("private");
  });

  it("includes only a bounded canonical GitHub request ID from an HTTP failure", () => {
    const requestId = "1B41:A4F74:930C3C:1DF33D1:6AD4920C";
    createWorkQueueLogger("provisioning").failure(
      "deployment.repository.read.failed",
      {
        status: 500,
        response: { headers: { "x-github-request-id": requestId, authorization: "private-token" }, data: "private-body" },
      },
      true
    );
    expect(output()).toContain(`GitHub request ID: ${requestId}.`);
    expect(output()).not.toContain("private");
  });

  it.each(["private-token", "1:2:3:4:5\nprivate", "1:2:3:4:5\n", "1:2:3:4:5\r", "1:2:3:4:5\u2028", "a".repeat(256), null, 123])("withholds malformed GitHub request IDs: %j", requestId => {
    createWorkQueueLogger("provisioning").failure("deployment.reference.read.failed", { status: 503, response: { headers: { "x-github-request-id": requestId } } }, true);
    expect(output()).toContain("GitHub request ID: unknown.");
    expect(output()).not.toContain("private");
    expect(output()).not.toContain("a".repeat(256));
  });

  it("does not invoke request-ID or header getters", () => {
    const getter = vi.fn(() => {
      throw new Error("private-token");
    });
    const logger = createWorkQueueLogger("provisioning");
    logger.failure("deployment.repository.read.failed", { status: 500, response: Object.defineProperty({}, "headers", { get: getter }) }, true);
    logger.failure("deployment.reference.read.failed", { status: 500, response: { headers: Object.defineProperty({}, "x-github-request-id", { get: getter }) } }, true);
    expect(getter).not.toHaveBeenCalled();
    expect(output()).toContain("GitHub request ID: unknown.");
    expect(output()).not.toContain("private");
  });

  it("reports only allowlisted static error codes without DEBUG or invoking getters", () => {
    const logger = createWorkQueueLogger("dispatch");
    logger.failure("launch.preflight.blocked", new Error("work_queue_dispatch_credential_client_mismatch"), true);
    expect(output()).toContain("work_queue_dispatch_credential_client_mismatch=true");
    expect(output()).toContain("could not be bound to the compiler-selected token");
    expect(output()).toContain("credential validation and workflow dispatch use the same selected token");
    const getter = vi.fn(() => "private-token");
    const error = Object.defineProperties({}, { code: { get: getter }, message: { get: getter } });
    expect(queueControlErrorCode(error)).toBe("queue_control_authority_or_publication_failed");
    expect(getter).not.toHaveBeenCalled();
    expect(queueControlErrorCode(new Error("work_queue_dispatch_credential_client_mismatch: private-token"))).toBe("queue_control_authority_or_publication_failed");
    expect(queueControlErrorCode({ code: "private-token" })).toBe("queue_control_authority_or_publication_failed");
  });

  it.each([
    ["work_queue_pool_not_installed", "automatic installation adds only new pools"],
    ["work_queue_profile_not_approved", "quiescent queue"],
    ["work_queue_worker_not_compiler_approved", "do not bypass the worker allowlist"],
  ])("explains installed routing failures without exposing task data: %s", (code, recovery) => {
    const error = Object.assign(new Error("private-task-and-token"), { code });
    expect(queueControlErrorCode(error)).toBe(code);
    expect(queueControlErrorExplanation(code)).toContain(recovery);
    createWorkQueueLogger("dispatch").failure("intent.process.blocked", error, true);
    expect(output()).toContain(`${code}=true`);
    expect(output()).toContain(recovery);
    expect(output()).not.toContain("private-");
    expect(queueControlErrorCode(new Error(`${code}: private-task-and-token`))).toBe("queue_control_authority_or_publication_failed");
    expect(output()).not.toContain("private-token");
  });

  it.each([
    ["work_queue_dispatch_repository_access_unavailable", "cannot check the target repository", "authenticated GitHub client"],
    ["work_queue_dispatch_credential_client_mismatch", "could not be bound", "same selected token"],
  ])("explains credential error %s without implying repository-role permission enforcement", (code, problem, action) => {
    expect(queueControlErrorCode(new Error(code))).toBe(code);
    createWorkQueueLogger("dispatch").failure("launch.preflight.blocked", new Error(code), true);
    expect(output()).toContain(`${code}=true`);
    expect(output()).toContain(problem);
    expect(output()).toContain(action);
  });

  it("preserves the structured code from an unavailable opaque-token identity API", async () => {
    let failure;
    try {
      await resolveDispatchCredentialPrincipal({}, { kind: "authenticated" });
    } catch (error) {
      failure = error;
    }
    expect(failure).toMatchObject({ code: "work_queue_dispatch_credential_identity_unavailable" });
    expect(queueControlErrorCode(failure)).toBe("work_queue_dispatch_credential_identity_unavailable");
    createWorkQueueLogger("dispatch").failure("launch.preflight.blocked", failure, true);
    expect(output()).toContain("work_queue_dispatch_credential_identity_unavailable=true");
    expect(output()).toContain("Check the selected token's validity and GitHub API access");
    expect(output()).not.toContain("an opaque token requires");
  });

  it("uses an allowlisted structured code without exposing or parsing the raw message", () => {
    const failure = Object.assign(new Error("private-token-and-task-payload"), { code: "work_queue_dispatch_credential_identity_unavailable" });
    expect(queueControlErrorCode(failure)).toBe("work_queue_dispatch_credential_identity_unavailable");
    createWorkQueueLogger("dispatch").failure("launch.preflight.blocked", failure, true);
    expect(output()).toContain("work_queue_dispatch_credential_identity_unavailable=true");
    expect(output()).not.toContain("private-token-and-task-payload");
  });

  it.each([
    [401, "GitHub rejected the credential", "valid, and unexpired"],
    [403, "GitHub denied the queue API request", "does not distinguish permission denial from throttling"],
    [404, "GitHub could not find or expose", "selected token's access"],
    [429, "GitHub rate-limited", "reconcile existing reservations"],
    [503, "GitHub returned a server error", "launch may already have happened"],
    [422, "GitHub rejected the queue API request", "workflow_dispatch trigger"],
  ])("explains HTTP %s and a safe recovery action without raw provider data", (status, problem, action) => {
    const error = { status, response: { data: { message: "private-provider-body" } } };
    const explanation = queueControlErrorExplanation(queueControlErrorCode(error));
    expect(explanation).toContain(problem);
    expect(explanation).toContain(action);
    createWorkQueueLogger("dispatch").failure("launch.post.failed", error, true);
    expect(output()).toContain(explanation);
    expect(output()).not.toContain("private-provider-body");
  });

  it("does not invoke nested response getters or print oversized error bodies in always-visible diagnostics", () => {
    const getter = vi.fn(() => {
      throw new Error("private");
    });
    const logger = createWorkQueueLogger("dispatch");
    logger.failure("launch.post.failed", { status: 422, response: Object.defineProperty({}, "data", { get: getter }) }, true);
    logger.failure("launch.post.failed", Object.defineProperty({}, "response", { get: getter }), true);
    logger.failure("launch.post.failed", { status: 422, response: { data: { message: "No ref found for:" + "private".repeat(2000) } } }, true);
    expect(getter).not.toHaveBeenCalled();
    expect(output()).not.toContain("ref_not_found=true");
    expect(output()).not.toContain("private");
  });

  it.each([
    { headers: { "x-ratelimit-remaining": "0" } },
    { headers: { "retry-after": "60" } },
    { message: "API rate limit exceeded for installation ID private-installation" },
    { message: "You have exceeded a secondary rate limit. private-provider-body" },
    { message: "You have triggered an abuse detection mechanism. private-provider-body" },
  ])("distinguishes HTTP 403 rate-limit evidence without printing provider data: %j", ({ headers = {}, message = "private-provider-body" }) => {
    const error = { status: 403, response: { headers, data: { message } } };
    expect(queueControlErrorCode(error)).toBe("queue_control_github_rate_limited");
    createWorkQueueLogger("native").failure("run.read.failed", error, true);
    expect(output()).toContain("http_status=403");
    expect(output()).toContain("queue_control_github_rate_limited=true");
    expect(output()).not.toContain("queue_control_github_access_denied=true");
    expect(output()).not.toContain("private-");
  });

  it("extracts failure metadata once for classification and logging", () => {
    const getOwnPropertyDescriptor = vi.fn(Reflect.getOwnPropertyDescriptor);
    const headers = new Proxy({ "x-ratelimit-remaining": "0" }, { getOwnPropertyDescriptor });
    createWorkQueueLogger("native").failure("run.read.failed", { status: 403, response: { headers } }, true);
    expect(getOwnPropertyDescriptor.mock.calls.filter(([, key]) => key === "x-ratelimit-remaining")).toHaveLength(1);
    expect(output()).toContain("rate_limit_remaining=0");
    expect(output()).toContain("queue_control_github_rate_limited=true");
  });

  it.each(["Resource not accessible by integration", "Resource not accessible by personal access token"])("preserves explicit permission evidence with an exhausted quota: %s", message => {
    const error = { status: 403, response: { headers: { "x-ratelimit-remaining": "0" }, data: { message } } };
    expect(queueControlErrorCode(error)).toBe("queue_control_github_access_denied");
    createWorkQueueLogger("native").failure("run.read.failed", error, true);
    expect(output()).toContain("rate_limit_remaining=0");
    expect(output()).toContain("resource_not_accessible=true");
    expect(output()).toContain("queue_control_github_access_denied=true");
    expect(output()).not.toContain("queue_control_github_rate_limited=true");
    expect(output()).not.toContain(message);
  });

  it.each([{ headers: { "retry-after": "60" } }, { errors: [{ type: "RATE_LIMITED", message: "private-provider-message" }] }])(
    "retains explicit throttling evidence alongside permission denial and an exhausted quota: %j",
    ({ headers = {}, errors }) => {
      const error = {
        status: 403,
        response: { headers: { "x-ratelimit-remaining": "0", ...headers }, data: { message: "Resource not accessible by integration" } },
        errors,
      };
      expect(queueControlErrorCode(error)).toBe("queue_control_github_rate_limited");
      createWorkQueueLogger("native").failure("run.read.failed", error, true);
      expect(output()).toContain("resource_not_accessible=true");
      expect(output()).toContain("queue_control_github_rate_limited=true");
      expect(output()).not.toContain("private-");
    }
  );

  it("logs only validated quota counters, retry timing and fixed resource flags", () => {
    createWorkQueueLogger("native").failure(
      "run.read.failed",
      {
        status: 403,
        response: {
          headers: {
            "x-ratelimit-limit": "15000",
            "x-ratelimit-remaining": "0",
            "x-ratelimit-used": "15000",
            "x-ratelimit-reset": "1792675926",
            "retry-after": "60",
            "x-ratelimit-resource": "core",
            authorization: "private-token",
          },
          data: { message: "API rate limit exceeded for installation ID private-installation" },
        },
      },
      true
    );
    expect(output()).toContain("rate_limit_limit=15000 rate_limit_remaining=0 rate_limit_used=15000 rate_limit_reset_unix=1792675926 retry_after_seconds=60");
    expect(output()).toContain("rate_limit_resource_core=true");
    expect(output()).toContain("installation_rate_limit=true");
    expect(output()).not.toContain("private-");
  });

  it("preserves uncertainty for a 403 with quota headroom and reports SSO hints without authorization URLs", () => {
    const error = {
      status: 403,
      response: {
        headers: { "x-ratelimit-remaining": "14904", "x-github-sso": "required; url=https://private.example/authorize?token=private-token" },
        data: { message: "Resource not accessible by integration" },
      },
    };
    expect(queueControlErrorCode(error)).toBe("queue_control_github_access_denied");
    createWorkQueueLogger("native").failure("run.read.failed", error, true);
    expect(output()).toContain("rate_limit_remaining=14904");
    expect(output()).toContain("sso_required=true");
    expect(output()).toContain("resource_not_accessible=true");
    expect(output()).toContain("does not distinguish permission denial from throttling");
    expect(output()).not.toContain("private");
  });

  it.each(["private-token", "-1", "1.5", "1e3", " 0", "0\n", "00", "9007199254740992", "1".repeat(256), 0, null])("withholds malformed numeric headers without coercion: %j", value => {
    const error = {
      status: 403,
      response: { headers: { "x-ratelimit-remaining": value, "retry-after": value, "x-ratelimit-resource": "private-resource" } },
    };
    expect(queueControlErrorCode(error)).toBe("queue_control_github_access_denied");
    createWorkQueueLogger("native").failure("run.read.failed", error, true);
    expect(output()).not.toContain("rate_limit_remaining=");
    expect(output()).not.toContain("retry_after_seconds=");
    expect(output()).not.toContain("private");
  });

  it("never invokes metadata getters or serializes unknown GraphQL error types", () => {
    const getter = vi.fn(() => {
      throw new Error("private-token");
    });
    const headers = Object.defineProperties({}, { "x-ratelimit-remaining": { get: getter }, "retry-after": { get: getter }, "x-github-sso": { get: getter } });
    const error = { status: 403, response: { headers }, errors: [{ type: "private-type", message: "private-message" }, Object.defineProperty({}, "type", { get: getter })] };
    createWorkQueueLogger("native").failure("run.read.failed", error, true);
    createWorkQueueLogger("native").failure("run.read.failed", Object.defineProperty({}, "errors", { get: getter }), true);
    createWorkQueueLogger("native").failure("run.read.failed", Object.defineProperty({}, "headers", { get: getter }), true);
    createWorkQueueLogger("native").failure("run.read.failed", { headers }, true);
    expect(getter).not.toHaveBeenCalled();
    expect(output()).toContain("graphql_errors=2");
    expect(output()).not.toContain("private");
  });

  it("logs validated quota metadata from GraphqlResponseError top-level headers without its response body", () => {
    const errors = [{ type: "RATE_LIMITED", message: "private-message", path: ["private-alias"] }];
    const error = {
      name: "GraphqlResponseError",
      headers: {
        "x-ratelimit-limit": "5000",
        "x-ratelimit-remaining": "0",
        "x-ratelimit-used": "5000",
        "x-ratelimit-reset": "1792675926",
        "retry-after": "60",
        "x-ratelimit-resource": "graphql",
        authorization: "private-token",
        location: "https://private.example/?token=private-token",
      },
      errors,
      response: { data: { "private-alias": null }, errors },
    };
    expect(queueControlErrorCode(error)).toBe("queue_control_github_rate_limited");
    createWorkQueueLogger("issue_coordination").failure("projection.locks.acquire.failed", error, true);
    expect(output()).toContain("rate_limit_limit=5000 rate_limit_remaining=0 rate_limit_used=5000 rate_limit_reset_unix=1792675926 retry_after_seconds=60");
    expect(output()).toContain("rate_limit_resource_graphql=true");
    expect(output()).toContain("graphql_rate_limited=true");
    expect(output()).not.toContain("private");
  });

  it.each([
    ["RATE_LIMITED", "graphql_rate_limited", "queue_control_github_rate_limited"],
    ["FORBIDDEN", "graphql_forbidden", "queue_control_authority_or_publication_failed"],
    ["UNPROCESSABLE", "graphql_unprocessable", "queue_control_authority_or_publication_failed"],
    ["NOT_FOUND", "graphql_not_found", "queue_control_authority_or_publication_failed"],
    ["GRAPHQL_VALIDATION_FAILED", "graphql_validation_failed", "queue_control_authority_or_publication_failed"],
  ])("logs the fixed GraphQL category %s without raw errors", (type, field, code) => {
    const error = { errors: [{ type, message: "private-message", path: ["private-alias"] }], data: { private: "private-response" } };
    expect(queueControlErrorCode(error)).toBe(code);
    createWorkQueueLogger("issue_coordination").failure("projection.locks.acquire.failed", error, true);
    expect(output()).toContain(`${field}=true`);
    expect(output()).not.toContain("private");
  });

  it("surfaces a failed GraphQL lock acquisition without authorizing projection or removing uncertain locks", async () => {
    const operation = vi.fn();
    const githubClient = {
      graphql: vi.fn().mockRejectedValue({
        status: 403,
        response: { headers: { "x-ratelimit-remaining": "0", "x-github-request-id": "1:2:3:4:5" }, data: { message: "private-provider-body" } },
      }),
    };
    const result = await withProjectionLocks({ githubClient, repositoryId: "private-repository", head: "a".repeat(40), keys: ["work:private-work"], scopes: [{ work_id: "private-work", keys: ["work:private-work"] }] }, operation);
    expect(result.pending).toHaveLength(1);
    expect(operation).not.toHaveBeenCalled();
    expect(githubClient.graphql).toHaveBeenCalledTimes(1);
    expect(output()).toContain("projection.locks.acquire.failed failed=true http_status=403 rate_limit_remaining=0");
    expect(output()).toContain("GitHub request ID: 1:2:3:4:5.");
    expect(output()).toContain("projection.locks.acquire.pending requested=1 acknowledged=0 pending_targets=1");
    expect(output()).not.toContain("private");
  });

  it("reports partial lock acknowledgements and retains only uncertain writes", async () => {
    const githubClient = {
      graphql: vi.fn().mockImplementation(async (query, variables) => {
        expect(query).toContain("WorkQueueProjectionLocks");
        const ref = { id: "private-ref", name: variables.l0.name.slice("refs/heads/".length), prefix: "refs/heads/", target: { oid: variables.l0.oid } };
        throw { data: { l0: { ref }, l1: null }, errors: [{ type: "UNPROCESSABLE", path: ["l1"], message: "private-provider-message" }] };
      }),
    };
    const operation = vi.fn(async accepted => {
      expect(accepted).toEqual(["private-a"]);
      return { pending: [], ambiguous: true, retain_keys: ["work:private-a"] };
    });
    const result = await withProjectionLocks(
      {
        githubClient,
        repositoryId: "private-repository",
        head: "a".repeat(40),
        keys: ["work:private-a", "work:private-b"],
        scopes: [
          { work_id: "private-a", keys: ["work:private-a"] },
          { work_id: "private-b", keys: ["work:private-b"] },
        ],
      },
      operation
    );
    expect(result.pending).toHaveLength(1);
    expect(operation).toHaveBeenCalledTimes(1);
    expect(githubClient.graphql).toHaveBeenCalledTimes(1);
    expect(output()).toContain("graphql_unprocessable=true");
    expect(output()).toContain("projection.locks.acquire.pending requested=2 acknowledged=1 pending_targets=1");
    expect(output()).toContain("projection.locks.retained locks=1");
    expect(output()).not.toContain("private");
  });

  it("surfaces REST lock acquisition errors without changing pending coordination semantics", async () => {
    const createRef = vi.fn().mockRejectedValue({ status: 422, response: { data: { message: "private-provider-body" } } });
    const operation = vi.fn();
    const githubClient = { rest: { git: { createRef } } };
    registerQueueLedgerClient(githubClient, githubClient);
    await expect(withProjectionLocks({ githubClient, owner: "private-owner", repo: "private-repo", head: "a".repeat(40), keys: ["work:private-work"] }, operation)).rejects.toMatchObject({ code: "projection_coordination_pending" });
    expect(operation).not.toHaveBeenCalled();
    expect(output()).toContain("projection.locks.acquire.failed failed=true http_status=422");
    expect(output()).not.toContain("private");
  });

  it("does not invoke an error status getter or coerce invalid status values", () => {
    vi.stubEnv("DEBUG", "work-queue:*");
    const getter = vi.fn(() => {
      throw new Error("private");
    });
    const logger = createWorkQueueLogger("store");
    logger.failure("read.failed", Object.defineProperty({}, "status", { get: getter }));
    for (const status of ["private", 999, -1, NaN, null]) logger.failure("read.failed", { status });
    expect(getter).not.toHaveBeenCalled();
    expect(output()).not.toContain("http_status");
    expect(output()).not.toContain("private");
  });

  it("logs a failure without propagating descriptor trap errors", () => {
    vi.stubEnv("DEBUG", "work-queue:*");
    const getOwnPropertyDescriptor = vi.fn(() => {
      throw new Error("private-descriptor-error");
    });
    const error = new Proxy({}, { getOwnPropertyDescriptor });
    expect(() => createWorkQueueLogger("store").failure("read.failed", error)).not.toThrow();
    expect(getOwnPropertyDescriptor).toHaveBeenCalledWith({}, "status");
    expect(output()).toContain("read.failed failed=true");
    expect(output()).not.toContain("http_status");
    expect(output()).not.toContain("private-descriptor-error");
  });

  it("logs a failure without propagating revoked proxy errors", () => {
    vi.stubEnv("DEBUG", "work-queue:*");
    const { proxy, revoke } = Proxy.revocable({}, {});
    revoke();
    expect(() => createWorkQueueLogger("store").failure("read.failed", proxy)).not.toThrow();
    expect(output()).toContain("read.failed failed=true");
    expect(output()).not.toContain("http_status");
    expect(output()).not.toContain("TypeError");
  });

  it("does not inspect a proxy when debug logging is disabled", () => {
    const getOwnPropertyDescriptor = vi.fn(() => {
      throw new Error("private");
    });
    const error = new Proxy({}, { getOwnPropertyDescriptor });
    expect(() => createWorkQueueLogger("store").failure("read.failed", error)).not.toThrow();
    expect(getOwnPropertyDescriptor).not.toHaveBeenCalled();
    expect(stderr).not.toHaveBeenCalled();
  });

  it("traces durable publication and idempotent recovery without Work payloads or identities", async () => {
    vi.stubEnv("DEBUG", "work-queue:store");
    const initial = [genesis()];
    const transaction = submission(initial, ["secret-task"], { graph: "private-graph" });
    const fake = fakeGitHub(initial);
    const first = await publishWorkQueueRequest(options(fake, transaction.request, producer));
    const second = await publishWorkQueueRequest(options(fake, transaction.request, producer));
    expect(first.persisted).toBe(true);
    expect(second.recovered).toBe(true);
    expect(output()).toContain("request.publish.complete");
    expect(output()).toContain("request.recovered");
    for (const secret of ["secret-task", "private-graph", "owner/repo", transaction.request.id, transaction.operations[0].work_id]) expect(output()).not.toContain(secret);
    expect(output()).not.toContain("[work-queue:replay]");
  });

  it("traces ambiguous-write recovery without changing publication semantics", async () => {
    vi.stubEnv("DEBUG", "work-queue:store");
    const initial = [genesis()];
    const transaction = submission(initial, ["private"]);
    const fake = fakeGitHub(initial);
    fake.state.ambiguousOnce = true;
    const result = await publishWorkQueueRequest(options(fake, transaction.request, producer));
    expect(result.recovered).toBe(true);
    expect(fake.state.updates).toBe(1);
    expect(output()).toContain("request.publish.write_failed failed=true http_status=502");
    expect(output()).toContain("request.publish.recovery conflict=false ambiguous=true");
    expect(output()).toContain("request.recovered");
    expect(output()).not.toContain("private");
  });

  it("reports no-grant scheduling without exposing pool or Work identities", () => {
    vi.stubEnv("DEBUG", "work-queue:scheduler");
    const state = replayTransactions([genesis()]);
    const decision = planDispatch(state, { pool: "default", max_claims: 1, max_dispatches: 1, max_bytes: 49152 }, { requestId: "private-request", commitId: "private-commit", at: 10 });
    expect(decision.reason).toBe("no_work");
    expect(output()).toContain("selection.no_grant pending=0 ineligible=false");
    expect(output()).toContain("dispatch.plan.complete claims=0 assignments=0 no_grant=true");
    expect(output()).not.toContain("private");
    expect(output()).not.toContain("default");
  });
});
