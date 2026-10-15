import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { createWorkQueueLogger, debugEnabled, queueControlErrorCode, queueControlErrorExplanation } = require("./work_queue_logging.cjs");
const { fakeGitHub, options } = require("./work_queue_store_checks.cjs");
const { genesis, submission, producer } = require("./work_queue_test_helpers.cjs");
const { publishWorkQueueRequest } = require("./work_queue_store.cjs");
const { planDispatch } = require("./work_queue_scheduler.cjs");
const { replayTransactions } = require("./work_queue_replay.cjs");
const { performance } = require("node:perf_hooks");

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

  it.each([
    [401, "GitHub rejected the credential", "valid, and unexpired"],
    [403, "GitHub denied the queue API request", "contents: write and actions: write"],
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
