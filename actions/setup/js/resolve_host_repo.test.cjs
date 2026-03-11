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
});
