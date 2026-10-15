// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { main, parseDispatchRef } from "./resolve_host_repo.cjs";
import { main as dispatchWorkflow } from "./dispatch_workflow.cjs";

const identityToken = claims => `header.${Buffer.from(JSON.stringify(claims)).toString("base64url")}.signature`;

describe("parseDispatchRef", () => {
  it("parses refs/heads/main from a standard workflow_ref", () => {
    expect(parseDispatchRef("owner/repo/.github/workflows/file.yml@refs/heads/main")).toBe("refs/heads/main");
  });

  it("parses refs/tags/v1.2.3 from a tag-triggered workflow_ref", () => {
    expect(parseDispatchRef("owner/repo/.github/workflows/file.yml@refs/tags/v1.2.3")).toBe("refs/tags/v1.2.3");
  });

  it("parses refs/heads/feature/my-branch from a nested branch name", () => {
    expect(parseDispatchRef("org/repo/.github/workflows/ci.yml@refs/heads/feature/my-branch")).toBe("refs/heads/feature/my-branch");
  });

  it("returns empty string when JOB_WORKFLOW_REF is empty", () => {
    expect(parseDispatchRef("")).toBe("");
  });

  it("returns empty string when JOB_WORKFLOW_REF has no '@' separator", () => {
    expect(parseDispatchRef("owner/repo/.github/workflows/file.yml")).toBe("");
  });

  it("returns empty string when JOB_WORKFLOW_REF ends with a 40-hex commit SHA", () => {
    // workflow_call can reference a workflow by commit SHA, but createWorkflowDispatch
    // rejects SHAs — parseDispatchRef must return "" to prevent dispatch failures.
    const sha = "abc123def456abc123def456abc123def456abc1";
    expect(parseDispatchRef(`owner/repo/.github/workflows/file.yml@${sha}`)).toBe("");
  });

  it("uses lastIndexOf so an '@' in the workflow path does not mis-parse the ref", () => {
    // Pathological but valid: if the path segment contained '@', lastIndexOf ensures
    // we capture the ref portion after the final '@'.
    expect(parseDispatchRef("owner/repo@org/.github/workflows/file.yml@refs/heads/main")).toBe("refs/heads/main");
  });

  it("does not return a SHA-like value for a tag ref", () => {
    const result = parseDispatchRef("owner/repo/.github/workflows/file.yml@refs/tags/v2.0.0");
    expect(result).toBe("refs/tags/v2.0.0");
    // Must not look like a SHA (40 hex chars)
    expect(result).not.toMatch(/^[0-9a-f]{40}$/);
  });
});

describe("resolve_host_repo main", () => {
  let outputs;
  let warnings;
  let infos;
  let originalEnv;

  beforeEach(() => {
    outputs = {};
    warnings = [];
    infos = [];

    global.core = {
      info: vi.fn(msg => infos.push(msg)),
      warning: vi.fn(msg => warnings.push(msg)),
      error: vi.fn(),
      getIDToken: vi.fn().mockRejectedValue(new Error("OIDC unavailable")),
      setSecret: vi.fn(),
      setOutput: vi.fn((key, value) => {
        outputs[key] = value;
      }),
    };

    originalEnv = {
      JOB_WORKFLOW_REPOSITORY: process.env.JOB_WORKFLOW_REPOSITORY,
      JOB_WORKFLOW_SHA: process.env.JOB_WORKFLOW_SHA,
      JOB_WORKFLOW_REF: process.env.JOB_WORKFLOW_REF,
      JOB_WORKFLOW_FILE_PATH: process.env.JOB_WORKFLOW_FILE_PATH,
      GITHUB_REPOSITORY: process.env.GITHUB_REPOSITORY,
      GITHUB_WORKFLOW_REF: process.env.GITHUB_WORKFLOW_REF,
      GITHUB_WORKFLOW_SHA: process.env.GITHUB_WORKFLOW_SHA,
      GITHUB_REF: process.env.GITHUB_REF,
      GITHUB_HEAD_REF: process.env.GITHUB_HEAD_REF,
    };
    for (const key of Object.keys(originalEnv)) delete process.env[key];
  });

  afterEach(() => {
    for (const [key, value] of Object.entries(originalEnv)) {
      if (value === undefined) {
        delete process.env[key];
      } else {
        process.env[key] = value;
      }
    }
  });

  it("emits target_checkout_ref as the commit SHA from JOB_WORKFLOW_SHA", async () => {
    process.env.JOB_WORKFLOW_REPOSITORY = "owner/platform-repo";
    process.env.JOB_WORKFLOW_SHA = "abc123def456abc123def456abc123def456abc1";
    process.env.JOB_WORKFLOW_REF = "owner/platform-repo/.github/workflows/orchestrator.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "owner/platform-repo";

    await main();

    expect(outputs["target_checkout_ref"]).toBe("abc123def456abc123def456abc123def456abc1");
    expect(core.getIDToken).not.toHaveBeenCalled();
  });

  it("emits target_ref as the dispatch-compatible branch ref from JOB_WORKFLOW_REF", async () => {
    process.env.JOB_WORKFLOW_REPOSITORY = "owner/platform-repo";
    process.env.JOB_WORKFLOW_SHA = "abc123def456abc123def456abc123def456abc1";
    process.env.JOB_WORKFLOW_REF = "owner/platform-repo/.github/workflows/orchestrator.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "owner/platform-repo";

    await main();

    expect(outputs["target_ref"]).toBe("refs/heads/main");
  });

  it("emits a tag as target_ref when workflow_ref contains a tag", async () => {
    process.env.JOB_WORKFLOW_REPOSITORY = "owner/platform-repo";
    process.env.JOB_WORKFLOW_SHA = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef";
    process.env.JOB_WORKFLOW_REF = "owner/platform-repo/.github/workflows/release.yml@refs/tags/v1.2.3";
    process.env.GITHUB_REPOSITORY = "owner/platform-repo";

    await main();

    expect(outputs["target_ref"]).toBe("refs/tags/v1.2.3");
  });

  it("target_ref is never a SHA when JOB_WORKFLOW_REF is provided", async () => {
    const sha = "abc123def456abc123def456abc123def456abc1";
    process.env.JOB_WORKFLOW_REPOSITORY = "owner/platform-repo";
    process.env.JOB_WORKFLOW_SHA = sha;
    process.env.JOB_WORKFLOW_REF = "owner/platform-repo/.github/workflows/file.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "owner/platform-repo";

    await main();

    expect(outputs["target_ref"]).not.toBe(sha);
    expect(outputs["target_ref"]).toBe("refs/heads/main");
  });

  it("emits target_ref as empty string and warns when JOB_WORKFLOW_REF is missing", async () => {
    process.env.JOB_WORKFLOW_REPOSITORY = "owner/platform-repo";
    process.env.JOB_WORKFLOW_SHA = "abc123def456abc123def456abc123def456abc1";
    delete process.env.JOB_WORKFLOW_REF;
    process.env.GITHUB_REPOSITORY = "owner/platform-repo";

    await main();

    expect(outputs["target_ref"]).toBe("");
    expect(warnings.length).toBeGreaterThan(0);
    expect(warnings[0]).toContain("Could not parse");
  });

  it("emits target_ref as empty string and warns when JOB_WORKFLOW_REF has no '@' separator", async () => {
    process.env.JOB_WORKFLOW_REPOSITORY = "owner/platform-repo";
    process.env.JOB_WORKFLOW_SHA = "abc123def456abc123def456abc123def456abc1";
    process.env.JOB_WORKFLOW_REF = "owner/platform-repo/.github/workflows/file.yml";
    process.env.GITHUB_REPOSITORY = "owner/platform-repo";

    await main();

    expect(outputs["target_ref"]).toBe("");
    expect(warnings.length).toBeGreaterThan(0);
  });

  it("emits target_ref as empty string and warns when JOB_WORKFLOW_REF ends with a commit SHA", async () => {
    // workflow_call can pin to a SHA; createWorkflowDispatch cannot accept a SHA as ref.
    const sha = "abc123def456abc123def456abc123def456abc1";
    process.env.JOB_WORKFLOW_REPOSITORY = "owner/platform-repo";
    process.env.JOB_WORKFLOW_SHA = sha;
    process.env.JOB_WORKFLOW_REF = `owner/platform-repo/.github/workflows/file.yml@${sha}`;
    process.env.GITHUB_REPOSITORY = "owner/platform-repo";

    await main();

    expect(outputs["target_ref"]).toBe("");
    expect(warnings.length).toBeGreaterThan(0);
    expect(outputs["target_checkout_ref"]).toBe(sha);
  });

  it("emits target_repo and target_repo_name correctly", async () => {
    process.env.JOB_WORKFLOW_REPOSITORY = "my-org/my-platform";
    process.env.JOB_WORKFLOW_SHA = "abc123def456abc123def456abc123def456abc1";
    process.env.JOB_WORKFLOW_REF = "my-org/my-platform/.github/workflows/file.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/my-caller";

    await main();

    expect(outputs["target_repo"]).toBe("my-org/my-platform");
    expect(outputs["target_repo_name"]).toBe("my-platform");
  });

  it.each(["", "missing-owner", "org/repo/extra", "org/repo\ninjected=value"])("rejects missing or malformed host metadata without selecting the caller (%j)", async repository => {
    process.env.JOB_WORKFLOW_REPOSITORY = repository;
    process.env.GITHUB_REPOSITORY = "caller-org/caller-repo";

    await expect(main()).rejects.toThrow("Refusing to fall back to the caller repository");
    expect(outputs).toEqual({});
  });

  it.each(["host-org/platform", "caller-org/project"])("resolves missing GHES job metadata from runner identity, not caller context (%s)", async repository => {
    process.env.GITHUB_REPOSITORY = "caller-org/project";
    process.env.GITHUB_WORKFLOW_REF = "caller-org/project/.github/workflows/caller.yml@refs/heads/caller";
    process.env.GITHUB_WORKFLOW_SHA = "a".repeat(40);
    const claims = {
      job_workflow_ref: `${repository}/.github/workflows/relay.lock.yml@refs/heads/platform`,
      job_workflow_sha: "b".repeat(40),
      workflow_ref: process.env.GITHUB_WORKFLOW_REF,
      workflow_sha: process.env.GITHUB_WORKFLOW_SHA,
      repository: process.env.GITHUB_REPOSITORY,
    };
    const token = identityToken(claims);
    core.getIDToken.mockResolvedValue(token);

    await main();

    expect(core.getIDToken).toHaveBeenCalledWith("gh-aw-host-repository");
    expect(core.setSecret).toHaveBeenCalledWith(token);
    expect(outputs).toEqual({
      target_repo: repository,
      target_repo_name: repository.split("/")[1],
      target_checkout_ref: "b".repeat(40),
      target_ref: "refs/heads/platform",
    });
    expect([...infos, ...warnings].join("\n")).not.toContain(token);
  });

  it.each([
    {},
    { workflow_ref: "caller-org/project/.github/workflows/caller.yml@refs/heads/main", workflow_sha: "a".repeat(40) },
    { job_workflow_ref: "host/repo/.github/workflows/relay.yml@refs/heads/main" },
    { job_workflow_ref: "host/repo/.github/workflows/relay.yml@refs/heads/main", job_workflow_sha: "refs/heads/main" },
    { job_workflow_ref: "host/repo/extra/.github/workflows/relay.yml@refs/heads/main", job_workflow_sha: "b".repeat(40) },
    { job_workflow_ref: "host/repo/.github/workflows/relay.yml@refs/heads/main\ninjected=value", job_workflow_sha: "b".repeat(40) },
  ])("rejects absent or invalid reusable-workflow claims without falling back to caller identity (%j)", async claims => {
    process.env.GITHUB_REPOSITORY = "caller-org/project";
    core.getIDToken.mockResolvedValue(identityToken(claims));

    await expect(main()).rejects.toThrow("Refusing to fall back to the caller repository");
    expect(outputs).toEqual({});
  });

  it("keeps a GHES reusable workflow's immutable SHA separate from its dispatch ref", async () => {
    const sha = "b".repeat(40);
    core.getIDToken.mockResolvedValue(identityToken({ job_workflow_ref: `host-org/platform/.github/workflows/relay.lock.yml@${sha}`, job_workflow_sha: sha }));
    await main();
    expect(outputs.target_checkout_ref).toBe(sha);
    expect(outputs.target_ref).toBe("");
    expect(warnings[0]).toContain("Could not parse");
  });

  it("does not expose token material from a failed runner identity request", async () => {
    const secret = "private-token-response";
    core.getIDToken.mockRejectedValue(new Error(secret));

    await expect(main()).rejects.toThrow("Grant id-token: write");
    await expect(main()).rejects.not.toThrow(secret);
    expect([...infos, ...warnings].join("\n")).not.toContain(secret);
    expect(outputs).toEqual({});
  });

  it.each(["not-a-token", "header.not-json.signature"])("fails closed on malformed runner identity tokens (%s)", async token => {
    core.getIDToken.mockResolvedValue(token);
    await expect(main()).rejects.toThrow("Refusing to fall back to the caller repository");
    expect(outputs).toEqual({});
  });

  it.each(["pull_request", "issue_comment"])("keeps inline dispatch defaults PR-aware after local identity resolution (%s)", async eventName => {
    process.env.JOB_WORKFLOW_REPOSITORY = "caller-org/project";
    process.env.GITHUB_REPOSITORY = "caller-org/project";
    process.env.JOB_WORKFLOW_SHA = "a".repeat(40);
    process.env.JOB_WORKFLOW_REF = `caller-org/project/.github/workflows/relay.lock.yml@${eventName === "pull_request" ? "refs/pull/42/merge" : "refs/heads/main"}`;
    process.env.GITHUB_REF = eventName === "pull_request" ? "refs/pull/42/merge" : "refs/heads/main";
    if (eventName === "pull_request") process.env.GITHUB_HEAD_REF = "feature/pr-head";
    const previousContext = global.context;
    const previousGithub = global.github;
    const previousGetOctokit = global.getOctokit;
    global.context = {
      repo: { owner: "caller-org", repo: "project" },
      eventName,
      ref: process.env.GITHUB_REF,
      payload: { issue: { number: 42, pull_request: {} } },
    };
    global.github = {
      rest: {
        actions: { createWorkflowDispatch: vi.fn().mockResolvedValue({ data: {} }) },
        pulls: { get: vi.fn().mockResolvedValue({ data: { head: { ref: "feature/pr-head" } } }) },
      },
    };
    delete global.getOctokit;
    try {
      await main();
      // The compiler contract omits target-ref for inlined imports (covered in Go).
      const handler = await dispatchWorkflow({ workflows: ["worker"], workflow_files: { worker: ".lock.yml" }, "target-repo": outputs.target_repo });
      const result = await handler({ type: "dispatch_workflow", workflow_name: "worker" }, {});
      expect(result.success).toBe(true);
      expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(expect.objectContaining({ owner: "caller-org", repo: "project", ref: "refs/heads/feature/pr-head" }));
      expect(github.rest.pulls.get).toHaveBeenCalledTimes(eventName === "issue_comment" ? 1 : 0);
    } finally {
      global.context = previousContext;
      global.github = previousGithub;
      global.getOctokit = previousGetOctokit;
    }
  });

  it("resolves the host locally even when activation job outputs are masked", async () => {
    vi.stubEnv("GH_AW_NEEDS_ACTIVATION_OUTPUTS_TARGET_REPO", "");
    vi.stubEnv("GH_AW_NEEDS_ACTIVATION_OUTPUTS_TARGET_REPO_NAME", "");
    vi.stubEnv("GH_AW_NEEDS_ACTIVATION_OUTPUTS_TARGET_REF", "");
    process.env.JOB_WORKFLOW_REPOSITORY = "host-org/platform";
    process.env.JOB_WORKFLOW_REF = "host-org/platform/.github/workflows/relay.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "caller-org/project";

    try {
      await main();
      expect(outputs["target_repo"]).toBe("host-org/platform");
      expect(outputs["target_repo_name"]).toBe("platform");
      expect(outputs["target_ref"]).toBe("refs/heads/main");
    } finally {
      vi.unstubAllEnvs();
    }
  });
});
