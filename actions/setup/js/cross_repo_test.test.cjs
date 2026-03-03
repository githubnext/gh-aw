import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

const mockGithub = {
  rest: {
    issues: {
      get: vi.fn(),
      update: vi.fn(),
    },
  },
};

const mockContext = {
  eventName: "issues",
  repo: {
    owner: "current-owner",
    repo: "current-repo",
  },
  serverUrl: "https://github.com",
  runId: 12345,
  payload: {
    issue: {
      number: 100,
    },
  },
};

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("update_issue cross-repo + operation integration", () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    vi.resetModules();
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_WORKFLOW_ID = "test-workflow";
  });

  it("should route API calls to the correct cross-repo target", async () => {
    let capturedGetOwner, capturedGetRepo;
    let capturedUpdateOwner, capturedUpdateRepo;
    let capturedBody;

    mockGithub.rest.issues.get.mockImplementation(async ({ owner, repo, issue_number }) => {
      capturedGetOwner = owner;
      capturedGetRepo = repo;
      return {
        data: {
          number: issue_number,
          title: "Test Issue",
          body: "Original body",
          html_url: `https://github.com/${owner}/${repo}/issues/${issue_number}`,
        },
      };
    });

    mockGithub.rest.issues.update.mockImplementation(async ({ owner, repo, issue_number, ...data }) => {
      capturedUpdateOwner = owner;
      capturedUpdateRepo = repo;
      capturedBody = data.body;
      return {
        data: {
          number: issue_number,
          title: "Test Issue",
          body: data.body || "Updated body",
          html_url: `https://github.com/${owner}/${repo}/issues/${issue_number}`,
        },
      };
    });

    const { main } = await import("/home/runner/work/gh-aw/gh-aw/actions/setup/js/update_issue.cjs");
    const handler = await main({
      "target-repo": "cross-owner/cross-repo",
      allowed_repos: ["cross-owner/cross-repo"],
      target: "*",
    });

    const message = {
      type: "update_issue",
      issue_number: 42,
      repo: "cross-owner/cross-repo",
      operation: "replace",
      body: "New body content",
    };

    const result = await handler(message, {});
    
    console.log("Result:", JSON.stringify(result));
    
    // API should go to cross-repo target
    expect(capturedGetOwner).toBe("cross-owner");
    expect(capturedGetRepo).toBe("cross-repo");
    expect(capturedUpdateOwner).toBe("cross-owner");
    expect(capturedUpdateRepo).toBe("cross-repo");
    
    // Body should contain the run URL - check if it's correct
    console.log("Captured body:", capturedBody);
    // The attribution URL should point to the CURRENT workflow's repo, not the cross-repo target
    if (capturedBody.includes("cross-owner/cross-repo/actions")) {
      console.error("BUG: Attribution URL points to cross-repo target instead of current workflow!");
      console.error("Body:", capturedBody);
    } else if (capturedBody.includes("current-owner/current-repo/actions")) {
      console.log("CORRECT: Attribution URL points to current workflow's repo");
    } else {
      console.log("INFO: Body does not contain expected run URL pattern");
    }
    
    expect(result.success).toBe(true);
  });

  it("should apply 'replace' operation correctly", async () => {
    mockGithub.rest.issues.get.mockResolvedValue({
      data: { number: 100, title: "Test", body: "Old body", html_url: "https://example.com/issues/100" },
    });
    mockGithub.rest.issues.update.mockResolvedValue({
      data: { number: 100, title: "Test", body: "New body", html_url: "https://example.com/issues/100" },
    });

    const { main } = await import("/home/runner/work/gh-aw/gh-aw/actions/setup/js/update_issue.cjs");
    const handler = await main({ target: "*" });

    const result = await handler({ issue_number: 100, operation: "replace", body: "New body" }, {});
    
    expect(result.success).toBe(true);
    const updateCall = mockGithub.rest.issues.update.mock.calls[0][0];
    console.log("Update called with body:", updateCall.body);
    // Replace should NOT include old body
    expect(updateCall.body).not.toContain("Old body");
    expect(updateCall.body).toContain("New body");
  });
});
