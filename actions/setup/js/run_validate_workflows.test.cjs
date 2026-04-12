// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

describe("run_validate_workflows", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let mockExec;
  let originalGlobals;
  let originalEnv;

  beforeEach(() => {
    originalEnv = { ...process.env };

    process.env.GH_AW_CMD_PREFIX = "./gh-aw";

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
      setFailed: vi.fn(),
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
    process.env = originalEnv;

    // Restore original globals
    global.core = originalGlobals.core;
    global.github = originalGlobals.github;
    global.context = originalGlobals.context;
    global.exec = originalGlobals.exec;

    vi.clearAllMocks();
  });

  it("should pass when validation has no errors or warnings", async () => {
    mockExec.exec.mockImplementation(async (_cmd, _args, options) => {
      if (options?.listeners?.stderr) {
        options.listeners.stderr(Buffer.from("Compiling workflows...\nDone.\n"));
      }
      return 0;
    });

    const { main } = await import("./run_validate_workflows.cjs");
    await main();

    expect(mockCore.info).toHaveBeenCalledWith("✓ All workflow validations passed with no errors or warnings");
    expect(mockGithub.rest.search.issuesAndPullRequests).not.toHaveBeenCalled();
    expect(mockGithub.rest.issues.create).not.toHaveBeenCalled();
  });

  it("should create an issue when validation fails with errors", async () => {
    mockExec.exec.mockImplementation(async (_cmd, _args, options) => {
      if (options?.listeners?.stderr) {
        options.listeners.stderr(Buffer.from("Error: schema validation failed for workflow.md\n"));
      }
      return 1;
    });

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 0, items: [] },
    });

    mockGithub.rest.issues.create.mockResolvedValue({
      data: { number: 42, html_url: "https://github.com/testowner/testrepo/issues/42" },
    });

    const { main } = await import("./run_validate_workflows.cjs");
    await main();

    expect(mockGithub.rest.search.issuesAndPullRequests).toHaveBeenCalled();
    expect(mockGithub.rest.issues.create).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
        title: "[aw] workflow validation findings",
        labels: ["agentic-workflows", "maintenance"],
      })
    );
    expect(mockCore.setFailed).toHaveBeenCalled();
  });

  it("should create an issue when validation has warnings", async () => {
    mockExec.exec.mockImplementation(async (_cmd, _args, options) => {
      if (options?.listeners?.stderr) {
        options.listeners.stderr(Buffer.from("Warning: action not pinned to SHA\n"));
      }
      return 0;
    });

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 0, items: [] },
    });

    mockGithub.rest.issues.create.mockResolvedValue({
      data: { number: 43, html_url: "https://github.com/testowner/testrepo/issues/43" },
    });

    const { main } = await import("./run_validate_workflows.cjs");
    await main();

    expect(mockGithub.rest.issues.create).toHaveBeenCalledWith(
      expect.objectContaining({
        title: "[aw] workflow validation findings",
      })
    );
    // Warnings don't call setFailed
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should add comment to existing issue when one already exists", async () => {
    mockExec.exec.mockImplementation(async (_cmd, _args, options) => {
      if (options?.listeners?.stderr) {
        options.listeners.stderr(Buffer.from("Error: validation failed\n"));
      }
      return 1;
    });

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: {
        total_count: 1,
        items: [{ number: 10, html_url: "https://github.com/testowner/testrepo/issues/10" }],
      },
    });

    const { main } = await import("./run_validate_workflows.cjs");
    await main();

    expect(mockGithub.rest.issues.createComment).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 10,
      })
    );
    expect(mockGithub.rest.issues.create).not.toHaveBeenCalled();
    expect(mockCore.setFailed).toHaveBeenCalled();
  });

  it("should use correct command prefix from environment", async () => {
    process.env.GH_AW_CMD_PREFIX = "gh aw";

    mockExec.exec.mockImplementation(async (_cmd, _args, options) => {
      if (options?.listeners?.stderr) {
        options.listeners.stderr(Buffer.from("Done.\n"));
      }
      return 0;
    });

    const { main } = await import("./run_validate_workflows.cjs");
    await main();

    const expectedArgs = ["aw", "compile", "--validate", "--no-emit", "--zizmor", "--actionlint", "--poutine", "--verbose"];
    expect(mockExec.exec).toHaveBeenCalledWith("gh", expect.arrayContaining(expectedArgs), expect.any(Object));
  });

  it("should truncate very large output in issues", async () => {
    const largeOutput = "x".repeat(60000);
    mockExec.exec.mockImplementation(async (_cmd, _args, options) => {
      if (options?.listeners?.stderr) {
        options.listeners.stderr(Buffer.from(largeOutput));
      }
      return 1;
    });

    mockGithub.rest.search.issuesAndPullRequests.mockResolvedValue({
      data: { total_count: 0, items: [] },
    });

    mockGithub.rest.issues.create.mockResolvedValue({
      data: { number: 44, html_url: "https://github.com/testowner/testrepo/issues/44" },
    });

    const { main } = await import("./run_validate_workflows.cjs");
    await main();

    const createCall = mockGithub.rest.issues.create.mock.calls[0][0];
    expect(createCall.body).toContain("... (output truncated)");
  });
});
