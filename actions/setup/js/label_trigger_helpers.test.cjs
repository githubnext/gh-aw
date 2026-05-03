// @ts-check

import { describe, it, expect, beforeEach, vi } from "vitest";
import { createRequire } from "module";

const req = createRequire(import.meta.url);
const { ensureLabelExists, validateLabeledIssueEvent, removeLabelSafely } = req("./label_trigger_helpers.cjs");

// ─── global mocks ────────────────────────────────────────────────────────────

const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
};

const mockGithub = {
  rest: {
    issues: {
      createLabel: vi.fn(),
      removeLabel: vi.fn(),
      createComment: vi.fn(),
    },
  },
};

global.core = mockCore;
global.github = mockGithub;

// Default context — will be overridden per test where needed
global.context = {
  eventName: "issues",
  repo: { owner: "test-owner", repo: "test-repo" },
  payload: {
    issue: { number: 42, body: "<!-- gh-aw-workflow-id: my-workflow -->" },
    label: { name: "agentic-workflows:disable" },
  },
};

// ─── ensureLabelExists ────────────────────────────────────────────────────────

describe("ensureLabelExists", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("creates the label when it does not exist", async () => {
    mockGithub.rest.issues.createLabel.mockResolvedValue({});

    await ensureLabelExists("owner", "repo", "my-label", "8250df", "My label description");

    expect(mockGithub.rest.issues.createLabel).toHaveBeenCalledWith({
      owner: "owner",
      repo: "repo",
      name: "my-label",
      color: "8250df",
      description: "My label description",
    });
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Created label 'my-label'"));
  });

  it("logs info when label already exists (422)", async () => {
    const alreadyExists = Object.assign(new Error("Unprocessable Entity"), { status: 422 });
    mockGithub.rest.issues.createLabel.mockRejectedValue(alreadyExists);

    await ensureLabelExists("owner", "repo", "my-label", "8250df", "desc");

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("already exists"));
    expect(mockCore.warning).not.toHaveBeenCalled();
  });

  it("logs a warning for unexpected errors (non-fatal)", async () => {
    mockGithub.rest.issues.createLabel.mockRejectedValue(new Error("Network error"));

    await ensureLabelExists("owner", "repo", "my-label", "8250df", "desc");

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to ensure label 'my-label' exists"));
  });
});

// ─── validateLabeledIssueEvent ────────────────────────────────────────────────

describe("validateLabeledIssueEvent", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.context = {
      eventName: "issues",
      repo: { owner: "test-owner", repo: "test-repo" },
      payload: {
        issue: { number: 42, body: "my body" },
        label: { name: "agentic-workflows:disable" },
      },
    };
  });

  it("returns issue context when event and label match", () => {
    const result = validateLabeledIssueEvent("agentic-workflows:disable");

    expect(result).not.toBeNull();
    expect(result?.owner).toBe("test-owner");
    expect(result?.repo).toBe("test-repo");
    expect(result?.issueNumber).toBe(42);
    expect(result?.body).toBe("my body");
  });

  it("returns null and logs info when event type is not issues", () => {
    global.context = { ...global.context, eventName: "pull_request" };

    const result = validateLabeledIssueEvent("agentic-workflows:disable");

    expect(result).toBeNull();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping"));
  });

  it("returns null and logs warning when no issue in payload", () => {
    global.context = { ...global.context, payload: { label: { name: "agentic-workflows:disable" } } };

    const result = validateLabeledIssueEvent("agentic-workflows:disable");

    expect(result).toBeNull();
    expect(mockCore.warning).toHaveBeenCalledWith("No issue found in event payload");
  });

  it("returns null and logs info when label does not match", () => {
    global.context = {
      ...global.context,
      payload: {
        issue: { number: 42, body: "" },
        label: { name: "some-other-label" },
      },
    };

    const result = validateLabeledIssueEvent("agentic-workflows:disable");

    expect(result).toBeNull();
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Skipping"));
  });

  it("returns empty string body when issue body is null", () => {
    global.context = {
      ...global.context,
      payload: {
        issue: { number: 42, body: null },
        label: { name: "agentic-workflows:disable" },
      },
    };

    const result = validateLabeledIssueEvent("agentic-workflows:disable");

    expect(result).not.toBeNull();
    expect(result?.body).toBe("");
  });

  it("works with apply-safe-outputs label too", () => {
    global.context = {
      ...global.context,
      payload: {
        issue: { number: 7, body: "some body" },
        label: { name: "agentic-workflows:apply-safe-outputs" },
      },
    };

    const result = validateLabeledIssueEvent("agentic-workflows:apply-safe-outputs");

    expect(result).not.toBeNull();
    expect(result?.issueNumber).toBe(7);
  });
});

// ─── removeLabelSafely ────────────────────────────────────────────────────────

describe("removeLabelSafely", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("removes the label and logs info", async () => {
    mockGithub.rest.issues.removeLabel.mockResolvedValue({});

    await removeLabelSafely("owner", "repo", 42, "my-label");

    expect(mockGithub.rest.issues.removeLabel).toHaveBeenCalledWith({
      owner: "owner",
      repo: "repo",
      issue_number: 42,
      name: "my-label",
    });
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Removed label 'my-label' from issue #42"));
  });

  it("logs a warning when removal fails (non-fatal — does not throw)", async () => {
    mockGithub.rest.issues.removeLabel.mockRejectedValue(new Error("Not Found"));

    await expect(removeLabelSafely("owner", "repo", 42, "my-label")).resolves.toBeUndefined();

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to remove label 'my-label'"));
  });
});
