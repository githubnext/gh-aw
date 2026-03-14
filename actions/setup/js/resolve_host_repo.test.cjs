// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
};

// Set up global mocks before importing the module
global.core = mockCore;

describe("resolve_host_repo.cjs", () => {
  let main;

  beforeEach(async () => {
    vi.clearAllMocks();
    mockCore.summary.addRaw.mockReturnThis();
    mockCore.summary.write.mockResolvedValue(undefined);

    const module = await import("./resolve_host_repo.cjs");
    main = module.main;
  });

  afterEach(() => {
    delete process.env.GITHUB_WORKFLOW_REF;
    delete process.env.GITHUB_REPOSITORY;
    delete process.env.GITHUB_REF;
  });

  it("should output the platform repo when invoked cross-repo", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
  });

  it("should log a cross-repo detection message and write step summary", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Cross-repo invocation detected"));
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should output the current repo when same-repo invocation", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/platform-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Same-repo invocation"));
  });

  it("should not write step summary for same-repo invocations", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/platform-repo";

    await main();

    expect(mockCore.summary.write).not.toHaveBeenCalled();
  });

  it("should fall back to GITHUB_REPOSITORY when GITHUB_WORKFLOW_REF is empty", async () => {
    process.env.GITHUB_WORKFLOW_REF = "";
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/fallback-repo");
  });

  it("should fall back to GITHUB_REPOSITORY when GITHUB_WORKFLOW_REF has unexpected format", async () => {
    process.env.GITHUB_WORKFLOW_REF = "not-a-valid-ref";
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/fallback-repo");
  });

  it("should handle event-driven relay (issue_comment) that calls a cross-repo workflow", async () => {
    // This is the exact scenario from the bug report:
    // An issue_comment event in app-repo triggers a relay that calls the platform workflow.
    // GITHUB_WORKFLOW_REF reflects the platform workflow, GITHUB_REPOSITORY is the caller.
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/my-workflow.lock.yml@main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Cross-repo invocation detected"));
  });

  it("should fall back to empty string when GITHUB_REPOSITORY is also undefined", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    delete process.env.GITHUB_REPOSITORY;

    await main();

    // workflowRepo parsed from GITHUB_WORKFLOW_REF is "my-org/platform-repo"
    // currentRepo is "" since env var is deleted
    // targetRepo = workflowRepo || currentRepo = "my-org/platform-repo"
    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
  });

  it("should log GITHUB_WORKFLOW_REF and GITHUB_REPOSITORY", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("GITHUB_WORKFLOW_REF:"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("GITHUB_REPOSITORY:"));
  });

  it("should output target_ref extracted from GITHUB_WORKFLOW_REF", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/feature-branch";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "refs/heads/feature-branch");
  });

  it("should output target_ref for a short branch ref (not refs/heads/...)", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "main");
  });

  it("should output target_ref for a feature branch in a caller-hosted relay", async () => {
    // This is the exact scenario from the bug report:
    // relay is pinned to @feature-branch, activation should check out feature-branch
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/platform-gateway.lock.yml@refs/heads/my-feature";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "refs/heads/my-feature");
  });

  it("should output empty target_ref when GITHUB_WORKFLOW_REF has no @ segment (no GITHUB_REF fallback)", async () => {
    // When GITHUB_WORKFLOW_REF has no '@', we cannot determine the callee ref.
    // We intentionally do NOT fall back to GITHUB_REF because in cross-repo scenarios
    // GITHUB_REF is the caller's ref, not the callee's. Empty string tells actions/checkout
    // to use the repository's default branch.
    process.env.GITHUB_WORKFLOW_REF = "not-a-valid-ref";
    process.env.GITHUB_REF = "refs/heads/fallback-branch";
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "");
  });

  it("should output empty target_ref when GITHUB_WORKFLOW_REF is empty", async () => {
    process.env.GITHUB_WORKFLOW_REF = "";
    delete process.env.GITHUB_REF;
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "");
  });

  it("should output target_ref for a tag ref", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/tags/v1.0.0";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "refs/tags/v1.0.0");
  });

  it("should output target_ref for a commit SHA", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@abc123def456";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "abc123def456");
  });

  it("should output target_repo_name when invoked cross-repo", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo_name", "platform-repo");
  });

  it("should output target_repo_name when same-repo invocation", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/platform-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo_name", "platform-repo");
  });

  it("should output target_repo_name without owner prefix when falling back to GITHUB_REPOSITORY", async () => {
    process.env.GITHUB_WORKFLOW_REF = "";
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo_name", "fallback-repo");
  });

  it("should include target_ref in step summary for cross-repo invocations", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/feature-branch";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("refs/heads/feature-branch"));
    expect(mockCore.summary.write).toHaveBeenCalled();
  });
});
