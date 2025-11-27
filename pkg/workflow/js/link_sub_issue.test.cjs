import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import os from "os";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    issues: {
      get: vi.fn(),
    },
  },
  graphql: vi.fn(),
};

const mockContext = {
  eventName: "workflow_dispatch",
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("link_sub_issue.cjs", () => {
  let tempDir;
  let outputFile;
  let linkSubIssueScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Create a temp directory for each test
    tempDir = fs.mkdtempSync(path.join(os.tmpdir(), "link-sub-issue-test-"));
    outputFile = path.join(tempDir, "agent-output.json");

    // Reset environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_LINK_SUB_ISSUE_MAX_COUNT;
    delete process.env.GH_AW_LINK_SUB_ISSUE_PARENT_REQUIRED_LABELS;
    delete process.env.GH_AW_LINK_SUB_ISSUE_PARENT_TITLE_PREFIX;
    delete process.env.GH_AW_LINK_SUB_ISSUE_SUB_REQUIRED_LABELS;
    delete process.env.GH_AW_LINK_SUB_ISSUE_SUB_TITLE_PREFIX;

    // Read the script content
    const scriptPath = path.join(process.cwd(), "link_sub_issue.cjs");
    linkSubIssueScript = fs.readFileSync(scriptPath, "utf8");
  });

  // Helper to set agent output in correct format
  const setAgentOutput = items => {
    const output = { items };
    fs.writeFileSync(outputFile, JSON.stringify(output));
    process.env.GH_AW_AGENT_OUTPUT = outputFile;
  };

  afterEach(() => {
    // Clean up temp directory
    if (tempDir && fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true });
    }
  });

  async function runScript() {
    // Execute the script using eval like other test files
    await eval(`(async () => { ${linkSubIssueScript} })()`);
  }

  it("should skip sub-issue that already has a parent", async () => {
    // Create agent output with link_sub_issue item
    setAgentOutput([{ type: "link_sub_issue", parent_issue_number: 100, sub_issue_number: 50 }]);

    // Mock parent issue fetch
    mockGithub.rest.issues.get
      .mockResolvedValueOnce({
        data: {
          number: 100,
          title: "Parent Issue",
          node_id: "I_parent_100",
          labels: [],
        },
      })
      // Mock sub-issue fetch
      .mockResolvedValueOnce({
        data: {
          number: 50,
          title: "Sub Issue",
          node_id: "I_sub_50",
          labels: [],
        },
      });

    // Mock GraphQL to return that sub-issue already has a parent
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        issue: {
          parent: {
            number: 99,
            title: "Existing Parent Issue",
          },
        },
      },
    });

    await runScript();

    // Verify warning was logged about existing parent
    expect(mockCore.warning).toHaveBeenCalledWith(
      expect.stringContaining('Sub-issue #50 is already a sub-issue of #99 ("Existing Parent Issue"). Skipping.')
    );

    // Verify no addSubIssue mutation was called (only the parent check query was called)
    expect(mockGithub.graphql).toHaveBeenCalledTimes(1);
    expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("parent {"), expect.any(Object));

    // Verify summary was written with failure
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryCall).toContain("Sub-issue is already a sub-issue of #99");
  });

  it("should proceed with linking when sub-issue has no parent", async () => {
    // Create agent output with link_sub_issue item
    setAgentOutput([{ type: "link_sub_issue", parent_issue_number: 100, sub_issue_number: 50 }]);

    // Mock parent issue fetch
    mockGithub.rest.issues.get
      .mockResolvedValueOnce({
        data: {
          number: 100,
          title: "Parent Issue",
          node_id: "I_parent_100",
          labels: [],
        },
      })
      // Mock sub-issue fetch
      .mockResolvedValueOnce({
        data: {
          number: 50,
          title: "Sub Issue",
          node_id: "I_sub_50",
          labels: [],
        },
      });

    // Mock GraphQL to return null parent (no existing parent)
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          issue: {
            parent: null,
          },
        },
      })
      // Mock addSubIssue mutation success
      .mockResolvedValueOnce({
        addSubIssue: {
          issue: {
            id: "I_parent_100",
            number: 100,
          },
          subIssue: {
            id: "I_sub_50",
            number: 50,
          },
        },
      });

    await runScript();

    // Verify no warning about existing parent
    expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("already a sub-issue"));

    // Verify addSubIssue mutation was called
    expect(mockGithub.graphql).toHaveBeenCalledTimes(2);
    expect(mockGithub.graphql).toHaveBeenLastCalledWith(expect.stringContaining("addSubIssue"), expect.any(Object));

    // Verify success was logged
    expect(mockCore.info).toHaveBeenCalledWith("Successfully linked issue #50 as sub-issue of #100");
  });

  it("should continue with linking if parent check query fails", async () => {
    // Create agent output with link_sub_issue item
    setAgentOutput([{ type: "link_sub_issue", parent_issue_number: 100, sub_issue_number: 50 }]);

    // Mock parent issue fetch
    mockGithub.rest.issues.get
      .mockResolvedValueOnce({
        data: {
          number: 100,
          title: "Parent Issue",
          node_id: "I_parent_100",
          labels: [],
        },
      })
      // Mock sub-issue fetch
      .mockResolvedValueOnce({
        data: {
          number: 50,
          title: "Sub Issue",
          node_id: "I_sub_50",
          labels: [],
        },
      });

    // Mock GraphQL to fail for parent check (e.g., field not available)
    mockGithub.graphql
      .mockRejectedValueOnce(new Error("Field 'parent' doesn't exist on type 'Issue'"))
      // Mock addSubIssue mutation success
      .mockResolvedValueOnce({
        addSubIssue: {
          issue: {
            id: "I_parent_100",
            number: 100,
          },
          subIssue: {
            id: "I_sub_50",
            number: 50,
          },
        },
      });

    await runScript();

    // Verify warning was logged about parent check failure
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not check if sub-issue #50 has a parent"));
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Proceeding with link attempt"));

    // Verify addSubIssue mutation was still called
    expect(mockGithub.graphql).toHaveBeenCalledTimes(2);
    expect(mockGithub.graphql).toHaveBeenLastCalledWith(expect.stringContaining("addSubIssue"), expect.any(Object));

    // Verify success was logged
    expect(mockCore.info).toHaveBeenCalledWith("Successfully linked issue #50 as sub-issue of #100");
  });

  it("should skip if no link_sub_issue items in output", async () => {
    // Create agent output with non-link_sub_issue item
    setAgentOutput([{ type: "create_issue", title: "New Issue" }]);

    await runScript();

    // Verify info message about no items
    expect(mockCore.info).toHaveBeenCalledWith("No link_sub_issue items found in agent output");

    // Verify no GitHub API calls
    expect(mockGithub.rest.issues.get).not.toHaveBeenCalled();
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });

  it("should handle multiple link requests with mixed existing parents", async () => {
    // Create agent output with multiple link_sub_issue items
    setAgentOutput([
      { type: "link_sub_issue", parent_issue_number: 100, sub_issue_number: 50 },
      { type: "link_sub_issue", parent_issue_number: 100, sub_issue_number: 51 },
    ]);

    // Mock parent issue fetch (called twice, once for each link)
    mockGithub.rest.issues.get
      .mockResolvedValueOnce({
        data: { number: 100, title: "Parent Issue", node_id: "I_parent_100", labels: [] },
      })
      .mockResolvedValueOnce({
        data: { number: 50, title: "Sub Issue 50", node_id: "I_sub_50", labels: [] },
      })
      .mockResolvedValueOnce({
        data: { number: 100, title: "Parent Issue", node_id: "I_parent_100", labels: [] },
      })
      .mockResolvedValueOnce({
        data: { number: 51, title: "Sub Issue 51", node_id: "I_sub_51", labels: [] },
      });

    // Mock GraphQL calls
    mockGithub.graphql
      // First sub-issue has existing parent
      .mockResolvedValueOnce({
        repository: {
          issue: {
            parent: { number: 99, title: "Existing Parent" },
          },
        },
      })
      // Second sub-issue has no parent
      .mockResolvedValueOnce({
        repository: {
          issue: {
            parent: null,
          },
        },
      })
      // addSubIssue for second sub-issue
      .mockResolvedValueOnce({
        addSubIssue: {
          issue: { id: "I_parent_100", number: 100 },
          subIssue: { id: "I_sub_51", number: 51 },
        },
      });

    await runScript();

    // First should be skipped
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Sub-issue #50 is already a sub-issue"));

    // Second should succeed
    expect(mockCore.info).toHaveBeenCalledWith("Successfully linked issue #51 as sub-issue of #100");

    // Verify summary contains both results
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    const summaryCall = mockCore.summary.addRaw.mock.calls[0][0];
    expect(summaryCall).toContain("Successfully linked 1 sub-issue");
    expect(summaryCall).toContain("Failed to link 1 sub-issue");
  });
});
