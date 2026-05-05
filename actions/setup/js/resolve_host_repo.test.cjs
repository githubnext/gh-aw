// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { main, parseDispatchRef } from "./resolve_host_repo.cjs";

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
    };
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

  it("emits target_repo and target_repo_name correctly", async () => {
    process.env.JOB_WORKFLOW_REPOSITORY = "my-org/my-platform";
    process.env.JOB_WORKFLOW_SHA = "abc123def456abc123def456abc123def456abc1";
    process.env.JOB_WORKFLOW_REF = "my-org/my-platform/.github/workflows/file.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/my-caller";

    await main();

    expect(outputs["target_repo"]).toBe("my-org/my-platform");
    expect(outputs["target_repo_name"]).toBe("my-platform");
  });
});
