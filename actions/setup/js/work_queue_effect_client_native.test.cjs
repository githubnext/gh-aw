// @ts-check
import { afterEach, describe, expect, it, vi } from "vitest";
import { getOctokit } from "@actions/github";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { withClaimExecution, closeClaimEffectChannel } = require("./work_queue_claim_scope.cjs");
const { wrapClaimEffectClient } = require("./work_queue_effect_client.cjs");
const { createRateLimitAwareGithub } = require("./github_rate_limit_logger.cjs");
const { queueFixture } = require("./work_queue_lifecycle.test_helpers.cjs");
const { settleClaimDelivery } = require("./safe_output_handler_manager.cjs");
const store = require("./work_queue_store.cjs");

afterEach(() => {
  vi.unstubAllEnvs();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

describe("native Octokit Claim readback", () => {
  it.each([false, true])("reads repository identity through the instrumented client with a verification signal=%s", async bounded => {
    const fixture = queueFixture({ count: 1, bound: true });
    const fetch = vi.fn(async (url, options) => {
      expect(String(url)).toBe("https://api.github.com/repos/owner/repo");
      expect(options.method).toBe("GET");
      return new Response(JSON.stringify({ id: 7, full_name: "owner/repo" }), { status: 200, headers: { "content-type": "application/json" } });
    });
    const source = createRateLimitAwareGithub(getOctokit("fixture-token", { request: { fetch } }));
    await withClaimExecution({ assignment: fixture.assignment, claim_handle: "h1", effects: [] }, async () => {
      const client = wrapClaimEffectClient(source, { claim_handle: "h1", ...(bounded ? { signal: new AbortController().signal } : {}) });
      closeClaimEffectChannel();
      const response = await client.rest.repos.get({ owner: "owner", repo: "repo", headers: { "X-GitHub-Api-Version": "2026-03-10" }, request: { retries: 0, timeout: 15000 } });
      expect(response.data.id).toBe(7);
    });
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it("reads the manager's native control inventory without replacing its verifier", async () => {
    const fixture = queueFixture({ count: 1, bound: true });
    fixture.append("finish", { dispatch_id: fixture.assignment.dispatch_id, claim_handle: "h1", outcome: "completed" }, { ...fixture.workerActor, dispatch_id: fixture.assignment.dispatch_id, claim_handle: "h1" });
    const fetch = vi.fn(async (url, options) => {
      expect(options.method).toBe("GET");
      const pathname = new URL(String(url)).pathname;
      expect(new URL(String(url)).origin).toBe("https://api.github.com");
      expect(["/repos/owner/repo", "/repos/owner/repo/actions/runs/42", "/repos/owner/repo/actions/runs/42/attempts/1"]).toContain(pathname);
      return new Response(JSON.stringify(pathname === "/repos/owner/repo" ? { id: 7, full_name: "owner/repo" } : fixture.nativeRun()), { status: 200, headers: { "content-type": "application/json" } });
    });
    const source = createRateLimitAwareGithub(getOctokit("fixture-token", { request: { fetch } }));
    vi.spyOn(store, "readWorkQueueLog").mockImplementation(fixture.readWorkQueueLog);
    vi.stubEnv("GITHUB_WORKFLOW_REF", `owner/repo/.github/workflows/worker.lock.yml@${"a".repeat(40)}`);
    vi.stubGlobal("context", fixture.workerContext);
    vi.stubGlobal("core", { debug: vi.fn(), info: vi.fn() });
    await withClaimExecution({ assignment: fixture.assignment, claim_handle: "h1", effects: [] }, async () => {
      const client = wrapClaimEffectClient(source, { claim_handle: "h1", authorizeGithub: source, context: fixture.workerContext });
      vi.stubGlobal("github", client);
      const delivery = await settleClaimDelivery({ assignment: fixture.assignment }, [], [], { github: client, context: fixture.workerContext });
      expect(delivery.verification, delivery.reason).toBe("verified");
    });
    expect(fetch).toHaveBeenCalled();
  });
});
