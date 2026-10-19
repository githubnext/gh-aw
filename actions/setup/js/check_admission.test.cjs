import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { main } = require("./check_admission.cjs");
const { buildWorkflowCallId } = require("./aw_context.cjs");
const FAILURE = "ERR_ADMISSION: Workflow execution was not admitted. Contact the repository administrator.";

describe("workflow admission", () => {
  let fetchMock;
  let coreMock;
  let body;

  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(Math, "random").mockReturnValue(0);
    coreMock = { setFailed: vi.fn(), warning: vi.fn() };
    body = { cancel: vi.fn().mockResolvedValue(undefined) };
    fetchMock = vi.fn().mockResolvedValue({ status: 204, body });
    vi.stubGlobal("core", coreMock);
    vi.stubGlobal("fetch", fetchMock);
    vi.stubGlobal("context", {
      repo: { owner: "acme", repo: "project" },
      runId: 1234,
      ref: "refs/heads/main",
      eventName: "push",
      payload: {},
    });
    vi.stubEnv("GH_AW_ADMISSION_URL", "https://cao.example/api/admission");
    vi.stubEnv("GH_AW_ADMISSION_TOKEN", "actions-token");
    vi.stubEnv("GH_AW_ADMISSION_ON_OUTAGE", "");
    vi.stubEnv("GH_AW_WORKFLOW_ID", "test-workflow");
    vi.stubEnv("GITHUB_RUN_ATTEMPT", "2");
    vi.stubEnv("GH_AW_GITHUB_TOKEN", "must-not-send-pat");
    vi.stubEnv("COPILOT_GITHUB_TOKEN", "must-not-send-copilot");
    vi.stubEnv("GH_AW_GITHUB_MCP_SERVER_TOKEN", "must-not-send-mcp");
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    vi.unstubAllEnvs();
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it.each([200, 204])("admits only explicit HTTP %s responses", async status => {
    fetchMock.mockResolvedValue({ status, body });
    await main();
    expect(coreMock.setFailed).not.toHaveBeenCalled();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url.origin + url.pathname).toBe("https://cao.example/api/admission");
    expect(Object.fromEntries(url.searchParams)).toEqual({
      repo: "acme/project",
      workflow_id: "test-workflow",
      workflow_call_id: buildWorkflowCallId("1234", "2", "test-workflow"),
      run_id: "1234",
      run_attempt: "2",
      ref: "refs/heads/main",
      event_type: "push",
    });
    expect(options).toEqual({
      method: "GET",
      headers: { Authorization: "Bearer actions-token", "Cache-Control": "no-store" },
      redirect: "manual",
      cache: "no-store",
      signal: expect.any(AbortSignal),
    });
    expect(url.toString()).not.toContain("actions-token");
    expect(JSON.stringify([url, options])).not.toContain("must-not-send");
    expect(body.cancel).toHaveBeenCalledOnce();
  });

  it.each([201, 202, 206, 301, 302, 307, 308, 400, 401, 403, 404, 409])("reports the same opaque failure for HTTP %s without reading diagnostics", async status => {
    const json = vi.fn();
    const text = vi.fn();
    fetchMock.mockResolvedValue({ status, body, json, text, headers: { "Retry-After": "budget-reset-time" } });
    await main();
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
    expect(fetchMock).toHaveBeenCalledOnce();
    expect(json).not.toHaveBeenCalled();
    expect(text).not.toHaveBeenCalled();
    expect(body.cancel).toHaveBeenCalledOnce();
    expect(coreMock.warning).not.toHaveBeenCalled();
  });

  it.each(["budget exhausted: 1234 credits", "actions-token", "DNS or TLS error"])("never exposes transport details: %s", async detail => {
    vi.stubEnv("GH_AW_ADMISSION_ON_OUTAGE", "block");
    fetchMock.mockRejectedValue(new Error(detail));
    const pending = main();
    await vi.runAllTimersAsync();
    await pending;
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(coreMock.warning).not.toHaveBeenCalled();
  });

  it("bounds the request to ten seconds and fails closed on timeout", async () => {
    vi.stubEnv("GH_AW_ADMISSION_ON_OUTAGE", "block");
    const timeout = vi.spyOn(AbortSignal, "timeout").mockImplementation(ms => {
      const controller = new AbortController();
      setTimeout(() => controller.abort(), ms);
      return controller.signal;
    });
    fetchMock.mockImplementation(
      (_url, options) =>
        new Promise((_resolve, reject) => {
          options.signal.addEventListener("abort", () => reject(new Error("private timeout diagnostics")));
        })
    );
    const pending = main();
    await vi.runAllTimersAsync();
    await pending;
    expect(timeout).toHaveBeenCalledTimes(3);
    expect(timeout.mock.calls).toEqual([[10000], [10000], [10000]]);
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
  });

  it.each([undefined, "ignore", "block"])("applies outage policy %s only after all network retries", async policy => {
    vi.stubEnv("GH_AW_ADMISSION_ON_OUTAGE", policy);
    fetchMock.mockRejectedValue(new Error("private outage diagnostics"));
    const pending = main();
    await vi.runAllTimersAsync();
    await pending;
    expect(fetchMock).toHaveBeenCalledTimes(3);
    if (policy === "block") {
      expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
      expect(coreMock.warning).not.toHaveBeenCalled();
    } else {
      expect(coreMock.setFailed).not.toHaveBeenCalled();
      expect(coreMock.warning).toHaveBeenCalledExactlyOnceWith("ERR_ADMISSION: Admission check did not complete. Continuing under the configured outage policy.");
    }
  });

  it.each([408, 429, 500, 502, 503, 504, 599])("retries transient HTTP %s and uses outage policy", async status => {
    fetchMock.mockResolvedValue({ status, body });
    const pending = main();
    await vi.runAllTimersAsync();
    await pending;
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(coreMock.setFailed).not.toHaveBeenCalled();
    expect(coreMock.warning).toHaveBeenCalledOnce();
  });

  it("recovers from a network glitch with exponential backoff", async () => {
    fetchMock.mockRejectedValueOnce(new Error("network glitch")).mockResolvedValueOnce({ status: 503, body });
    const pending = main();
    await vi.advanceTimersByTimeAsync(249);
    expect(fetchMock).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(1);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(499);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    await vi.advanceTimersByTimeAsync(1);
    await pending;
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(coreMock.setFailed).not.toHaveBeenCalled();
    expect(coreMock.warning).not.toHaveBeenCalled();
  });

  it("does not ignore or retry a denial after a transient failure", async () => {
    fetchMock.mockRejectedValueOnce(new Error("network glitch")).mockResolvedValueOnce({ status: 403, body });
    const pending = main();
    await vi.runAllTimersAsync();
    await pending;
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
    expect(coreMock.warning).not.toHaveBeenCalled();
  });

  it("does not let a body cancellation error convert denial to an outage", async () => {
    body.cancel.mockRejectedValue(new Error("private cleanup failure"));
    fetchMock.mockResolvedValue({ status: 403, body });
    await main();
    expect(fetchMock).toHaveBeenCalledOnce();
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
    expect(coreMock.warning).not.toHaveBeenCalled();
  });

  it("blocks transient HTTP failures when the outage policy is block", async () => {
    vi.stubEnv("GH_AW_ADMISSION_ON_OUTAGE", "block");
    fetchMock.mockResolvedValue({ status: 503, body });
    const pending = main();
    await vi.runAllTimersAsync();
    await pending;
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
    expect(coreMock.warning).not.toHaveBeenCalled();
  });

  it("rejects invalid outage policies without contacting the service", async () => {
    vi.stubEnv("GH_AW_ADMISSION_ON_OUTAGE", "allow");
    await main();
    expect(fetchMock).not.toHaveBeenCalled();
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
  });

  it.each([
    "",
    "http://cao.example/admission",
    "https://",
    "https://user:password@cao.example/admission",
    "https://cao.example/admission?secret=value",
    "https://cao.example/admission?",
    "https://cao.example/admission#fragment",
    "https://cao.example/admission#",
    "https://cao.example/ bad",
    "https://cao.example/\\evil",
    "https://${{ secrets.ENDPOINT }}/admission",
  ])("does not send a token to an invalid endpoint: %s", async endpoint => {
    vi.stubEnv("GH_AW_ADMISSION_URL", endpoint);
    await main();
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("does not fall back to other credentials when the Actions token is missing", async () => {
    vi.stubEnv("GH_AW_ADMISSION_TOKEN", "");
    await main();
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("fails closed when the current workflow identity is missing", async () => {
    vi.stubEnv("GH_AW_WORKFLOW_ID", "");
    await main();
    expect(coreMock.setFailed).toHaveBeenCalledExactlyOnceWith(FAILURE);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("encodes refs as data and never sends arbitrary payload or inbound aw_context fields", async () => {
    context.ref = "refs/heads/feature/a&run_id=attacker";
    context.eventName = "workflow_dispatch";
    context.payload = {
      inputs: { aw_context: JSON.stringify({ repo: "attacker/repo", run_id: "attacker", workflow_id: "attacker", event_type: "schedule", secret: "must-not-send" }) },
      private_data: "must-not-send",
    };
    await main();
    const [url] = fetchMock.mock.calls[0];
    expect(url.searchParams.get("ref")).toBe(context.ref);
    expect(url.searchParams.getAll("run_id")).toEqual(["1234"]);
    expect(url.searchParams.get("repo")).toBe("acme/project");
    expect(url.searchParams.get("event_type")).toBe("workflow_dispatch");
    expect(url.toString()).not.toContain("must-not-send");
    expect(coreMock.setFailed).not.toHaveBeenCalled();
  });

  it("checks every reusable workflow hop and rerun independently", async () => {
    context.eventName = "workflow_call";
    await main();
    vi.stubEnv("GH_AW_WORKFLOW_ID", "child-workflow");
    await main();
    vi.stubEnv("GITHUB_RUN_ATTEMPT", "3");
    await main();
    const calls = fetchMock.mock.calls.map(([url]) => url.searchParams.get("workflow_call_id"));
    expect(new Set(calls).size).toBe(3);
    expect(coreMock.setFailed).not.toHaveBeenCalled();
  });
});
