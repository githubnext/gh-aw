// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

describe("close_issue.cjs", () => {
  let mockCore;
  let mockGithub;
  let mockContext;
  let originalEnv;

  beforeEach(() => {
    // Save original environment
    originalEnv = { ...process.env };

    // Reset environment variables
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_CLOSE_TARGET;
    delete process.env.GH_AW_REQUIRED_LABELS;
    delete process.env.GH_AW_ALLOWED_OUTCOMES;

    // Setup mocks
    mockCore = {
      info: vi.fn(),
      error: vi.fn(),
      setFailed: vi.fn(),
      setOutput: vi.fn(),
      summary: {
        addRaw: vi.fn().mockReturnThis(),
        write: vi.fn().mockResolvedValue(undefined),
      },
    };

    mockGithub = {
      rest: {
        issues: {
          get: vi.fn(),
          update: vi.fn(),
        },
      },
    };

    mockContext = {
      eventName: "issues",
      repo: {
        owner: "test-owner",
        repo: "test-repo",
      },
      payload: {
        issue: {
          number: 123,
        },
      },
    };

    // Set global objects
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;
  });

  afterEach(() => {
    // Restore environment
    process.env = originalEnv;
    vi.clearAllMocks();
  });

  it("should close an issue with default outcome when no output items", async () => {
    const agentOutput = {
      items: [],
    };

    const tempFile = path.join(__dirname, "test-data", "close-issue-empty.json");
    fs.mkdirSync(path.dirname(tempFile), { recursive: true });
    fs.writeFileSync(tempFile, JSON.stringify(agentOutput));

    process.env.GH_AW_AGENT_OUTPUT = tempFile;

    const script = fs.readFileSync(path.join(__dirname, "close_issue.cjs"), "utf8");
    await eval(`(async () => { ${script} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No close-issue items found in agent output");

    fs.unlinkSync(tempFile);
  });

  it("should close an issue with completed outcome", async () => {
    const agentOutput = {
      items: [
        {
          type: "close_issue",
          outcome: "completed",
          reason: "Fixed in PR #123",
        },
      ],
    };

    const tempFile = path.join(__dirname, "test-data", "close-issue-completed.json");
    fs.mkdirSync(path.dirname(tempFile), { recursive: true });
    fs.writeFileSync(tempFile, JSON.stringify(agentOutput));

    process.env.GH_AW_AGENT_OUTPUT = tempFile;

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 123,
        labels: [],
      },
    });

    mockGithub.rest.issues.update.mockResolvedValue({
      data: {
        number: 123,
        html_url: "https://github.com/test-owner/test-repo/issues/123",
        title: "Test Issue",
      },
    });

    const script = fs.readFileSync(path.join(__dirname, "close_issue.cjs"), "utf8");
    await eval(`(async () => { ${script} })()`);

    expect(mockGithub.rest.issues.update).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      issue_number: 123,
      state: "closed",
      state_reason: "completed",
    });

    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_number", 123);
    expect(mockCore.setOutput).toHaveBeenCalledWith("issue_url", "https://github.com/test-owner/test-repo/issues/123");

    fs.unlinkSync(tempFile);
  });

  it("should validate required labels before closing", async () => {
    const agentOutput = {
      items: [
        {
          type: "close_issue",
          outcome: "not_planned",
        },
      ],
    };

    const tempFile = path.join(__dirname, "test-data", "close-issue-labels.json");
    fs.mkdirSync(path.dirname(tempFile), { recursive: true });
    fs.writeFileSync(tempFile, JSON.stringify(agentOutput));

    process.env.GH_AW_AGENT_OUTPUT = tempFile;
    process.env.GH_AW_REQUIRED_LABELS = "stale,wontfix";

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 123,
        labels: [{ name: "stale" }], // Missing "wontfix" label
      },
    });

    const script = fs.readFileSync(path.join(__dirname, "close_issue.cjs"), "utf8");
    await eval(`(async () => { ${script} })()`);

    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("does not have required labels"));

    fs.unlinkSync(tempFile);
  });

  it("should handle staged mode by showing preview", async () => {
    const agentOutput = {
      items: [
        {
          type: "close_issue",
          issue_number: 456,
          outcome: "not_planned",
          reason: "Out of scope",
        },
      ],
    };

    const tempFile = path.join(__dirname, "test-data", "close-issue-staged.json");
    fs.mkdirSync(path.dirname(tempFile), { recursive: true });
    fs.writeFileSync(tempFile, JSON.stringify(agentOutput));

    process.env.GH_AW_AGENT_OUTPUT = tempFile;
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    const script = fs.readFileSync(path.join(__dirname, "close_issue.cjs"), "utf8");
    await eval(`(async () => { ${script} })()`);

    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Staged Mode: Close Issues Preview"));
    expect(mockCore.summary.write).toHaveBeenCalled();
    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();

    fs.unlinkSync(tempFile);
  });

  it("should validate allowed outcomes", async () => {
    const agentOutput = {
      items: [
        {
          type: "close_issue",
          outcome: "duplicate", // Not in allowed outcomes
        },
      ],
    };

    const tempFile = path.join(__dirname, "test-data", "close-issue-invalid-outcome.json");
    fs.mkdirSync(path.dirname(tempFile), { recursive: true });
    fs.writeFileSync(tempFile, JSON.stringify(agentOutput));

    process.env.GH_AW_AGENT_OUTPUT = tempFile;
    process.env.GH_AW_ALLOWED_OUTCOMES = "completed,not_planned";

    mockGithub.rest.issues.get.mockResolvedValue({
      data: {
        number: 123,
        labels: [],
      },
    });

    const script = fs.readFileSync(path.join(__dirname, "close_issue.cjs"), "utf8");
    await eval(`(async () => { ${script} })()`);

    expect(mockGithub.rest.issues.update).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("is not in allowed outcomes"));

    fs.unlinkSync(tempFile);
  });
});
