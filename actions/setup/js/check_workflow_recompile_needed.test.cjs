// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("check_workflow_recompile_needed", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let mockExec;
  let originalGlobals;

  beforeEach(() => {
    // Save original globals
    originalGlobals = {
      core: global.core,
      github: global.github,
      context: global.context,
      exec: global.exec,
    };

    // Setup mock core module
    mockCore = {
      info: vi.fn(),
      warning: vi.fn(),
      error: vi.fn(),
      summary: {
        addHeading: vi.fn().mockReturnThis(),
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    // Setup mock github module
    mockGithub = {
      rest: {
        search: {
          issuesAndPullRequests: vi.fn(),
        },
        issues: {
          create: vi.fn(),
          createComment: vi.fn(),
        },
      },
    };

    // Setup mock context
    mockContext = {
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      runId: 123456,
      payload: {
        repository: {
          html_url: "https://github.com/testowner/testrepo",
        },
      },
    };

    // Setup mock exec module
    mockExec = {
      exec: vi.fn(),
    };

    // Set globals for the module
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
    global.exec = mockExec;
  });

  afterEach(() => {
    // Restore original globals
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.exec = originalGlobals.exec;
  });

  it("should report no changes when workflows are up to date", async () => {
    // Mock exec to return no changes (empty diff output)
    mockExec.exec.mockResolvedValue(0);

    const { main } = await import("./check_workflow_recompile_needed.cjs");
    await main();

    expect(mockCore.info).toHaveBeenCalledWith("✓ All workflow lock files are up to date");
    expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
  });

  it("should add comment to existing issue when workflows are out of sync", async () => {
    // Mock exec to return changes (non-empty diff output)
    mockExec.exec
      .mockImplementationOnce(async (cmd, args, options) => {
        if (options?.listeners?.stdout) {
          options.listeners.stdout(Buffer.from("diff content"));
        }
        return 1; // Non-zero exit code indicates changes
      })
      .mockImplementationOnce(async (cmd, args, options) => {
        if (options?.listeners?.stdout) {
          options.listeners.stdout(Buffer.from("detailed diff content"));
        }
        return 0;
      });

    // Mock search to return existing issue
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [
          {
            number: 42,
            html_url: "https://github.com/testowner/testrepo/issues/42",
          },
        ],
      },
    });

    mockGithub.rest.issues.createComment.mockResolvedValue({});

    const { main } = await import("./check_workflow_recompile_needed.cjs");
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Found existing issue"));
    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      issue_number: 42,
      body: expect.stringContaining("Workflows are still out of sync"),
    });
    expect(mockGithub.rest.issues.create).not.toHaveBeenCalled();
  });

  it("should create new issue when workflows are out of sync and no issue exists", async () => {
    // Mock exec to return changes (non-empty diff output)
    mockExec.exec
      .mockImplementationOnce(async (cmd, args, options) => {
        if (options?.listeners?.stdout) {
          options.listeners.stdout(Buffer.from("diff content"));
        }
        return 1;
      })
      .mockImplementationOnce(async (cmd, args, options) => {
        if (options?.listeners?.stdout) {
          options.listeners.stdout(Buffer.from("detailed diff content"));
        }
        return 0;
      });

    // Mock search to return no existing issue
    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 0,
        items: [],
      },
    });

    mockGithub.rest.issues.create.mockResolvedValue({
      data: {
        number: 43,
        html_url: "https://github.com/testowner/testrepo/issues/43",
      },
    });

    const { main } = await import("./check_workflow_recompile_needed.cjs");
    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No existing issue found"));
    expect(mockGithub.rest.issues.create).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      title: "Workflows need recompilation",
      body: expect.stringContaining("Instructions for GitHub Copilot"),
      labels: ["maintenance", "workflows"],
    });
  });

  it("should handle errors gracefully", async () => {
    // Mock exec to throw error
    mockExec.exec.mockRejectedValue(new Error("Git command failed"));

    const { main } = await import("./check_workflow_recompile_needed.cjs");

    await expect(main()).rejects.toThrow("Git command failed");
    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to check for workflow changes"));
  });
});
