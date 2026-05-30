// @ts-check

const { getWorkflowMetadata, buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");

describe("getWorkflowMetadata", () => {
  let originalEnv;
  let originalContext;

  beforeEach(() => {
    // Save original environment
    originalEnv = { ...process.env };

    // Save and mock global context
    originalContext = global.context;
    global.context = {
      runId: 123456,
      payload: {
        repository: {
          html_url: "https://github.com/test-owner/test-repo",
        },
      },
    };
  });

  afterEach(() => {
    // Restore environment by mutating process.env in place
    for (const key of Object.keys(process.env)) {
      if (!(key in originalEnv)) {
        delete process.env[key];
      }
    }
    Object.assign(process.env, originalEnv);

    // Restore original context
    global.context = originalContext;
  });

  it("should extract workflow metadata from environment and context", () => {
    // Set environment variables
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_WORKFLOW_ID = "test-workflow-id";
    process.env.GITHUB_SERVER_URL = "https://github.com";

    const metadata = getWorkflowMetadata("test-owner", "test-repo");

    expect(metadata).toEqual({
      workflowName: "Test Workflow",
      workflowId: "test-workflow-id",
      runId: 123456,
      runUrl: "https://github.com/test-owner/test-repo/actions/runs/123456",
    });
  });

  it("should use defaults when environment variables are missing", () => {
    // Clear environment variables
    delete process.env.GH_AW_WORKFLOW_NAME;
    delete process.env.GH_AW_WORKFLOW_ID;
    delete process.env.GITHUB_SERVER_URL;

    const metadata = getWorkflowMetadata("test-owner", "test-repo");

    expect(metadata.workflowName).toBe("Workflow");
    expect(metadata.workflowId).toBe("");
    expect(metadata.runId).toBe(123456);
    expect(metadata.runUrl).toBe("https://github.com/test-owner/test-repo/actions/runs/123456");
  });

  it("should construct runUrl from githubServer when repository payload is missing", () => {
    // Mock context without repository payload
    global.context = {
      runId: 789012,
      payload: {},
    };

    process.env.GITHUB_SERVER_URL = "https://github.enterprise.com";

    const metadata = getWorkflowMetadata("enterprise-owner", "enterprise-repo");

    expect(metadata.runUrl).toBe("https://github.enterprise.com/enterprise-owner/enterprise-repo/actions/runs/789012");
  });

  it("should handle missing context gracefully", () => {
    // Mock context with missing runId
    global.context = {
      payload: {
        repository: {
          html_url: "https://github.com/test-owner/test-repo",
        },
      },
    };

    const metadata = getWorkflowMetadata("test-owner", "test-repo");

    expect(metadata.runId).toBe(0);
    expect(metadata.runUrl).toBe("https://github.com/test-owner/test-repo/actions/runs/0");
  });
});

describe("buildWorkflowRunUrl", () => {
  it("should build run URL from context.serverUrl and explicit workflowRepo", () => {
    const ctx = { serverUrl: "https://github.com", runId: 42000 };
    const url = buildWorkflowRunUrl(ctx, { owner: "wf-owner", repo: "wf-repo" });
    expect(url).toBe("https://github.com/wf-owner/wf-repo/actions/runs/42000");
  });

  it("should fall back to GITHUB_SERVER_URL when context.serverUrl is absent", () => {
    const originalEnv = process.env.GITHUB_SERVER_URL;
    process.env.GITHUB_SERVER_URL = "https://ghes.example.com";
    const ctx = { runId: 99 };
    const url = buildWorkflowRunUrl(ctx, { owner: "ent-owner", repo: "ent-repo" });
    expect(url).toBe("https://ghes.example.com/ent-owner/ent-repo/actions/runs/99");
    if (originalEnv === undefined) {
      delete process.env.GITHUB_SERVER_URL;
    } else {
      process.env.GITHUB_SERVER_URL = originalEnv;
    }
  });

  it("should use the workflowRepo, not a cross-repo target", () => {
    // Simulates the cross-repo case: context.repo is the target but workflowRepo is the workflow owner
    const ctx = { serverUrl: "https://github.com", runId: 7777, repo: { owner: "cross-owner", repo: "cross-repo" } };
    const workflowRepo = { owner: "wf-owner", repo: "wf-repo" };
    const url = buildWorkflowRunUrl(ctx, workflowRepo);
    expect(url).toBe("https://github.com/wf-owner/wf-repo/actions/runs/7777");
    expect(url).not.toContain("cross-owner");
    expect(url).not.toContain("cross-repo");
  });
});
