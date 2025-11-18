import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),

  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),

  // Input/state functions
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  eventName: "issues",
  payload: {
    issue: {
      number: 42,
    },
  },
};

const mockGithub = {
  rest: {
    issues: {
      update: vi.fn(),
      listMilestones: vi.fn(),
    },
  },
};

// Set up global mocks
global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;

describe("add_milestone", () => {
  let addMilestoneScript;
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset all mocks before each test
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_MILESTONES_ALLOWED;
    delete process.env.GH_AW_MILESTONE_TARGET;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.payload.issue = { number: 42 };

    // Reset mock implementations
    mockGithub.rest.issues.update.mockResolvedValue({});
    mockGithub.rest.issues.listMilestones.mockResolvedValue({ data: [] });

    // Read the script content
    const scriptPath = path.join(process.cwd(), "add_milestone.cjs");
    addMilestoneScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up temporary file
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  it("should warn when no add-milestone item found", async () => {
    setAgentOutput({
      items: [],
      errors: [],
    });

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith("No add-milestone item found in agent output");
  });

  it("should generate staged preview in staged mode", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: "v1.0",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should fail when no allowed milestones configured", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: "v1.0",
        },
      ],
      errors: [],
    });

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockCore.setFailed).toHaveBeenCalledWith(
      "No allowed milestones configured. Please configure safe-outputs.add-milestone.allowed in your workflow."
    );
  });

  it("should fail when milestone not in allowed list", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: "v2.0",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_MILESTONES_ALLOWED = "v1.0,v1.1";

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Milestone 'v2.0' is not in the allowed list"));
  });

  it("should add milestone by number when allowed", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: 5,
        },
      ],
      errors: [],
    });
    process.env.GH_AW_MILESTONES_ALLOWED = "5,6";

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 42,
      milestone: 5,
    });
    expect(mockCore.setOutput).toHaveBeenCalledWith("milestone_added", "5");
    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", "42");
  });

  it("should resolve milestone title to number", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: "v1.0",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_MILESTONES_ALLOWED = "v1.0";

    mockGithub.rest.issues.listMilestones.mockResolvedValue({
      data: [
        { number: 10, title: "v1.0" },
        { number: 11, title: "v1.1" },
      ],
    });

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockGithub.rest.issues.listMilestones).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      state: "open",
      per_page: 100,
    });
    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 42,
      milestone: 10,
    });
  });

  it("should resolve milestone title case-insensitively", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: "V1.0",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_MILESTONES_ALLOWED = "v1.0";

    mockGithub.rest.issues.listMilestones.mockResolvedValue({
      data: [{ number: 10, title: "v1.0" }],
    });

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 42,
      milestone: 10,
    });
  });

  it("should use item_number when target is *", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: 5,
          item_number: 123,
        },
      ],
      errors: [],
    });
    process.env.GH_AW_MILESTONES_ALLOWED = "5";
    process.env.GH_AW_MILESTONE_TARGET = "*";

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 123,
      milestone: 5,
    });
  });

  it("should fail when target is * but no item_number provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: 5,
        },
      ],
      errors: [],
    });
    process.env.GH_AW_MILESTONES_ALLOWED = "5";
    process.env.GH_AW_MILESTONE_TARGET = "*";

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockCore.setFailed).toHaveBeenCalledWith('Target is "*" but no item_number specified in milestone item');
  });

  it("should fail when milestone not found in repository", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: "nonexistent",
        },
      ],
      errors: [],
    });
    process.env.GH_AW_MILESTONES_ALLOWED = "nonexistent";

    mockGithub.rest.issues.listMilestones
      .mockResolvedValueOnce({ data: [] }) // open milestones
      .mockResolvedValueOnce({ data: [] }); // closed milestones

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Milestone 'nonexistent' not found in repository"));
  });

  it("should handle API errors gracefully", async () => {
    setAgentOutput({
      items: [
        {
          type: "add_milestone",
          milestone: 5,
        },
      ],
      errors: [],
    });
    process.env.GH_AW_MILESTONES_ALLOWED = "5";

    mockGithub.rest.issues.update.mockRejectedValue(new Error("API Error"));

    await eval(`(async () => { ${addMilestoneScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith("Failed to add milestone: API Error");
    expect(mockCore.setFailed).toHaveBeenCalledWith("Failed to add milestone: API Error");
  });
});
