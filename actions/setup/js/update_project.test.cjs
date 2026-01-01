import { describe, it, expect, beforeAll, beforeEach, afterEach, vi } from "vitest";

let updateProject;
let parseProjectInput;
let generateCampaignId;
let extractDateFromTimestamp;
let extractWorkerWorkflowFromBody;
let hasExistingFieldValue;

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  getInput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    issues: {
      addLabels: vi.fn().mockResolvedValue({}),
    },
  },
  graphql: vi.fn(),
};

const mockContext = {
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

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

beforeAll(async () => {
  const mod = await import("./update_project.cjs");
  const exports = mod.default || mod;
  updateProject = exports.updateProject;
  parseProjectInput = exports.parseProjectInput;
  generateCampaignId = exports.generateCampaignId;
  extractDateFromTimestamp = exports.extractDateFromTimestamp;
  extractWorkerWorkflowFromBody = exports.extractWorkerWorkflowFromBody;
  hasExistingFieldValue = exports.hasExistingFieldValue;
  // Call main to execute the module
  if (exports.main) {
    await exports.main();
  }
});

function clearMock(fn) {
  if (fn && typeof fn.mockClear === "function") {
    fn.mockClear();
  }
}

function clearCoreMocks() {
  clearMock(mockCore.debug);
  clearMock(mockCore.info);
  clearMock(mockCore.notice);
  clearMock(mockCore.warning);
  clearMock(mockCore.error);
  clearMock(mockCore.setFailed);
  clearMock(mockCore.setOutput);
  clearMock(mockCore.exportVariable);
  clearMock(mockCore.getInput);
  clearMock(mockCore.summary.addRaw);
  clearMock(mockCore.summary.write);
}

beforeEach(() => {
  mockGithub.graphql.mockReset();
  mockGithub.rest.issues.addLabels.mockClear();
  clearCoreMocks();
  vi.useRealTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

const repoResponse = (ownerType = "Organization") => ({
  repository: {
    id: "repo123",
    owner: {
      id: ownerType === "User" ? "owner-user-123" : "owner123",
      __typename: ownerType,
    },
  },
});

const viewerResponse = (login = "test-bot") => ({
  viewer: {
    login,
  },
});

const orgProjectV2Response = (url, number = 60, id = "project123", orgLogin = "testowner") => ({
  organization: {
    projectV2: {
      id,
      number,
      title: "Test Project",
      url,
      owner: { __typename: "Organization", login: orgLogin },
    },
  },
});

const userProjectV2Response = (url, number = 60, id = "project123", userLogin = "testowner") => ({
  user: {
    projectV2: {
      id,
      number,
      title: "Test Project",
      url,
      owner: { __typename: "User", login: userLogin },
    },
  },
});

const orgProjectNullResponse = () => ({ organization: { projectV2: null } });
const userProjectNullResponse = () => ({ user: { projectV2: null } });

const issueResponse = (id, body = null) => ({ repository: { issue: { id, body } } });

const pullRequestResponse = (id, body = null) => ({ repository: { pullRequest: { id, body } } });

const emptyItemsResponse = () => ({
  node: {
    items: {
      nodes: [],
      pageInfo: { hasNextPage: false, endCursor: null },
    },
  },
});

const existingItemResponse = (contentId, itemId = "existing-item") => ({
  node: {
    items: {
      nodes: [{ id: itemId, content: { id: contentId } }],
      pageInfo: { hasNextPage: false, endCursor: null },
    },
  },
});

const fieldsResponse = nodes => ({ node: { fields: { nodes } } });

const updateFieldValueResponse = () => ({
  updateProjectV2ItemFieldValue: {
    projectV2Item: {
      id: "item123",
    },
  },
});

const addDraftIssueResponse = (itemId = "draft-item") => ({
  addProjectV2DraftIssue: {
    projectItem: {
      id: itemId,
    },
  },
});

function queueResponses(responses) {
  responses.forEach(response => {
    mockGithub.graphql.mockResolvedValueOnce(response);
  });
}

function getOutput(name) {
  const call = mockCore.setOutput.mock.calls.find(([key]) => key === name);
  return call ? call[1] : undefined;
}

describe("parseProjectInput", () => {
  it("extracts the project number from a GitHub URL", () => {
    expect(parseProjectInput("https://github.com/orgs/acme/projects/42")).toBe("42");
  });

  it("rejects a numeric string", () => {
    expect(() => parseProjectInput("17")).toThrow(/full GitHub project URL/);
  });

  it("rejects a project name", () => {
    expect(() => parseProjectInput("Engineering Roadmap")).toThrow(/full GitHub project URL/);
  });

  it("throws when the project input is missing", () => {
    expect(() => parseProjectInput(undefined)).toThrow(/Invalid project input/);
  });
});

describe("generateCampaignId", () => {
  it("builds a slug with a timestamp suffix", () => {
    const nowSpy = vi.spyOn(Date, "now").mockReturnValue(1734470400000);
    const id = generateCampaignId("https://github.com/orgs/acme/projects/42", "42");
    expect(id).toBe("acme-project-42-m4syw5xc");
    nowSpy.mockRestore();
  });
});

describe("extractDateFromTimestamp", () => {
  it("extracts YYYY-MM-DD from ISO 8601 timestamp", () => {
    expect(extractDateFromTimestamp("2025-12-15T10:30:00Z")).toBe("2025-12-15");
    expect(extractDateFromTimestamp("2025-01-01T00:00:00.000Z")).toBe("2025-01-01");
    expect(extractDateFromTimestamp("2024-06-30T23:59:59+00:00")).toBe("2024-06-30");
  });

  it("returns null for invalid timestamps", () => {
    expect(extractDateFromTimestamp(null)).toBe(null);
    expect(extractDateFromTimestamp(undefined)).toBe(null);
    expect(extractDateFromTimestamp("")).toBe(null);
    expect(extractDateFromTimestamp("invalid")).toBe(null);
    expect(extractDateFromTimestamp("12/15/2025")).toBe(null);
  });

  it("returns null for non-string values", () => {
    expect(extractDateFromTimestamp(123)).toBe(null);
    expect(extractDateFromTimestamp({})).toBe(null);
    expect(extractDateFromTimestamp([])).toBe(null);
  });
});

describe("extractWorkerWorkflowFromBody", () => {
  it("extracts workflow name from XML comment", () => {
    const body = "Some issue text\n\n<!-- agentic-workflow: Daily Updater, tracker-id: abc123 -->\n\nMore text";
    expect(extractWorkerWorkflowFromBody(body)).toBe("Daily Updater");
  });

  it("extracts workflow name with special characters", () => {
    const body = "<!-- agentic-workflow: Test-Workflow_2024, tracker-id: xyz -->";
    expect(extractWorkerWorkflowFromBody(body)).toBe("Test-Workflow_2024");
  });

  it("handles XML comment at start of body", () => {
    const body = "<!-- agentic-workflow: First Workflow, run: https://example.com -->\nIssue content";
    expect(extractWorkerWorkflowFromBody(body)).toBe("First Workflow");
  });

  it("extracts workflow name without tracker-id", () => {
    const body = "Text\n<!-- agentic-workflow: Simple Workflow -->\nMore text";
    expect(extractWorkerWorkflowFromBody(body)).toBe("Simple Workflow");
  });

  it("returns null for invalid or missing XML comment", () => {
    expect(extractWorkerWorkflowFromBody("No XML comment here")).toBe(null);
    expect(extractWorkerWorkflowFromBody("<!-- other-comment: value -->")).toBe(null);
    expect(extractWorkerWorkflowFromBody(null)).toBe(null);
    expect(extractWorkerWorkflowFromBody(undefined)).toBe(null);
    expect(extractWorkerWorkflowFromBody("")).toBe(null);
  });

  it("returns null for non-string values", () => {
    expect(extractWorkerWorkflowFromBody(123)).toBe(null);
    expect(extractWorkerWorkflowFromBody({})).toBe(null);
    expect(extractWorkerWorkflowFromBody([])).toBe(null);
  });
});

describe("updateProject", () => {
  it("fails when the project URL cannot be resolved", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = { type: "update_project", project: projectUrl };

    queueResponses([repoResponse(), viewerResponse(), orgProjectNullResponse()]);

    await expect(updateProject(output)).rejects.toThrow(/not found or not accessible/);
  });

  it("respects a custom campaign id", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      campaign_id: "custom-id-2025",
      content_type: "issue",
      content_number: 42,
    };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project456"), issueResponse("issue-id-42"), emptyItemsResponse(), { addProjectV2ItemById: { item: { id: "item-custom" } } }]);

    await updateProject(output);

    const labelCall = mockGithub.rest.issues.addLabels.mock.calls[0][0];
    expect(labelCall.labels).toEqual(["campaign:custom-id-2025"]);
    expect(getOutput("item-id")).toBe("item-custom");
  });

  it("adds an issue to a project board", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = { type: "update_project", project: projectUrl, content_type: "issue", content_number: 42 };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project123"), issueResponse("issue-id-42"), emptyItemsResponse(), { addProjectV2ItemById: { item: { id: "item123" } } }]);

    await updateProject(output);

    // No campaign label should be added when campaign_id is not provided
    expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    expect(getOutput("item-id")).toBe("item123");
  });

  it("adds an issue to a project board with campaign label when campaign_id provided", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = { type: "update_project", project: projectUrl, content_type: "issue", content_number: 42, campaign_id: "my-campaign" };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project123"), issueResponse("issue-id-42"), emptyItemsResponse(), { addProjectV2ItemById: { item: { id: "item123" } } }]);

    await updateProject(output);

    const labelCall = mockGithub.rest.issues.addLabels.mock.calls[0][0];
    expect(labelCall).toEqual(
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
        issue_number: 42,
      })
    );
    expect(labelCall.labels).toEqual(["campaign:my-campaign"]);
    expect(getOutput("item-id")).toBe("item123");
  });

  it("adds a draft issue to a project board", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "draft_issue",
      draft_title: "Draft title",
      draft_body: "Draft body",
    };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project-draft"), addDraftIssueResponse("draft-item-1")]);

    await updateProject(output);

    expect(mockGithub.graphql.mock.calls.some(([query]) => query.includes("addProjectV2DraftIssue"))).toBe(true);
    expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    expect(getOutput("item-id")).toBe("draft-item-1");
  });

  it("rejects draft issues without a title", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "draft_issue",
      draft_title: "   ",
    };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project-draft")]);

    await expect(updateProject(output)).rejects.toThrow(/draft_title/);
  });

  it("skips adding an issue that already exists on the board", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = { type: "update_project", project: projectUrl, content_type: "issue", content_number: 99 };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project123"), issueResponse("issue-id-99"), existingItemResponse("issue-id-99", "item-existing")]);

    await updateProject(output);

    expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith("✓ Item already on board");
    expect(getOutput("item-id")).toBe("item-existing");
  });

  it("adds a pull request to the project board", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = { type: "update_project", project: projectUrl, content_type: "pull_request", content_number: 17 };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project-pr"), pullRequestResponse("pr-id-17"), emptyItemsResponse(), { addProjectV2ItemById: { item: { id: "pr-item" } } }]);

    await updateProject(output);

    // No campaign label should be added when campaign_id is not provided
    expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
  });

  it("falls back to legacy issue field when content_number missing", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = { type: "update_project", project: projectUrl, issue: "101" };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "legacy-project"), issueResponse("issue-id-101"), emptyItemsResponse(), { addProjectV2ItemById: { item: { id: "legacy-item" } } }]);

    await updateProject(output);

    expect(mockCore.warning).toHaveBeenCalledWith('Field "issue" deprecated; use "content_number" instead.');

    // No campaign label should be added when campaign_id is not provided
    expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    expect(getOutput("item-id")).toBe("legacy-item");
  });

  it("rejects invalid content numbers", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = { type: "update_project", project: projectUrl, content_number: "ABC" };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "invalid-project")]);

    await expect(updateProject(output)).rejects.toThrow(/Invalid content number/);
  });

  it("updates an existing text field", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 10,
      fields: { Status: "In Progress" },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-field"),
      issueResponse("issue-id-10"),
      existingItemResponse("issue-id-10", "item-field"),
      fieldsResponse([{ id: "field-status", name: "Status" }]),
      updateFieldValueResponse(),
    ]);

    await updateProject(output);

    const updateCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCall).toBeDefined();
    expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
  });

  it("updates fields on a draft issue item", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "draft_issue",
      draft_title: "Draft title",
      fields: { Status: "In Progress" },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-draft-fields"),
      addDraftIssueResponse("draft-item-fields"),
      fieldsResponse([{ id: "field-status", name: "Status" }]),
      updateFieldValueResponse(),
    ]);

    await updateProject(output);

    const updateCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCall).toBeDefined();
    expect(mockGithub.rest.issues.addLabels).not.toHaveBeenCalled();
    expect(getOutput("item-id")).toBe("draft-item-fields");
  });

  it("updates a single select field when the option exists", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 15,
      fields: { Priority: "High" },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-priority"),
      issueResponse("issue-id-15"),
      existingItemResponse("issue-id-15", "item-priority"),
      fieldsResponse([
        {
          id: "field-priority",
          name: "Priority",
          options: [
            { id: "opt-low", name: "Low" },
            { id: "opt-high", name: "High" },
          ],
        },
      ]),
      updateFieldValueResponse(),
    ]);

    await updateProject(output);

    const updateCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCall).toBeDefined();
  });

  it("creates a new option in single select field with colors for existing options", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 16,
      fields: { Status: "Closed - Not Planned" },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-status"),
      issueResponse("issue-id-16"),
      existingItemResponse("issue-id-16", "item-status"),
      fieldsResponse([
        {
          id: "field-status",
          name: "Status",
          options: [
            { id: "opt-todo", name: "Todo", color: "GRAY" },
            { id: "opt-in-progress", name: "In Progress", color: "YELLOW" },
            { id: "opt-done", name: "Done", color: "GREEN" },
            { id: "opt-closed", name: "Closed", color: "PURPLE" },
          ],
        },
      ]),
      // Response for updateProjectV2Field mutation
      {
        updateProjectV2Field: {
          projectV2Field: {
            id: "field-status",
            options: [
              { id: "opt-todo", name: "Todo" },
              { id: "opt-in-progress", name: "In Progress" },
              { id: "opt-done", name: "Done" },
              { id: "opt-closed", name: "Closed" },
              { id: "opt-closed-not-planned", name: "Closed - Not Planned" },
            ],
          },
        },
      },
      updateFieldValueResponse(),
    ]);

    await updateProject(output);

    // Find the updateProjectV2Field mutation call
    const updateFieldCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("updateProjectV2Field"));
    expect(updateFieldCall).toBeDefined();

    // Verify that the mutation includes color for all options
    const options = updateFieldCall[1].options;
    expect(options).toHaveLength(5); // 4 existing + 1 new

    // Check that all existing options have their colors preserved
    expect(options[0]).toEqual({ name: "Todo", description: "", color: "GRAY" });
    expect(options[1]).toEqual({ name: "In Progress", description: "", color: "YELLOW" });
    expect(options[2]).toEqual({ name: "Done", description: "", color: "GREEN" });
    expect(options[3]).toEqual({ name: "Closed", description: "", color: "PURPLE" });

    // Check that the new option has a default color
    expect(options[4]).toEqual({ name: "Closed - Not Planned", description: "", color: "GRAY" });
  });

  it("warns when a field cannot be created", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 20,
      fields: { NonExistentField: "Some Value" },
    };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project-test"), issueResponse("issue-id-20"), existingItemResponse("issue-id-20", "item-test"), fieldsResponse([])]);

    mockGithub.graphql.mockRejectedValueOnce(new Error("Failed to create field"));

    await updateProject(output);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining('Failed to create field "NonExistentField"'));
  });

  it("warns when adding the campaign label fails", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = { type: "update_project", project: projectUrl, content_type: "issue", content_number: 50, campaign_id: "test-campaign" };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project-label"), issueResponse("issue-id-50"), emptyItemsResponse(), { addProjectV2ItemById: { item: { id: "item-label" } } }]);

    mockGithub.rest.issues.addLabels.mockRejectedValueOnce(new Error("Labels disabled"));

    await updateProject(output);

    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Failed to add campaign label"));
  });

  it("rejects non-URL project identifier", async () => {
    const output = { type: "update_project", project: "My Campaign", campaign_id: "my-campaign-123" };
    await expect(updateProject(output)).rejects.toThrow(/full GitHub project URL/);
  });

  it("accepts URL project identifier when campaign_id is present", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      campaign_id: "my-campaign-123",
    };

    queueResponses([repoResponse(), viewerResponse(), orgProjectV2Response(projectUrl, 60, "project123")]);

    await updateProject(output);

    expect(mockCore.error).not.toHaveBeenCalled();
  });

  it("updates date fields only when provided", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 50,
      campaign_id: "test-campaign",
      fields: {
        start_date: "2025-12-01",
        end_date: "2025-12-20",
      },
    };

    const issueWithTimestamps = {
      repository: {
        issue: {
          id: "issue-id-50",
          createdAt: "2025-12-15T10:30:00Z",
          closedAt: "2025-12-18T16:45:00Z",
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-override"),
      issueWithTimestamps,
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "item-override" } } },
      fieldsResponse([
        { id: "field-start", name: "Start Date", dataType: "DATE" },
        { id: "field-end", name: "End Date", dataType: "DATE" },
      ]),
      updateFieldValueResponse(),
      updateFieldValueResponse(),
      // Auto-population logic queries fields again
      fieldsResponse([
        { id: "field-start", name: "Start Date", dataType: "DATE" },
        { id: "field-end", name: "End Date", dataType: "DATE" },
      ]),
    ]);

    await updateProject(output);

    // Should skip auto-populating because user explicitly provided the fields
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Start Date was explicitly provided, skipping auto-population"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("End Date was explicitly provided, skipping auto-population"));

    // Verify user-provided values are used
    const updateCalls = mockGithub.graphql.mock.calls.filter(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCalls).toHaveLength(2);
    expect(updateCalls[0][1].value).toEqual({ date: "2025-12-01" });
    expect(updateCalls[1][1].value).toEqual({ date: "2025-12-20" });
  });

  it("correctly identifies DATE fields and uses date format (not singleSelectOptionId)", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 75,
      fields: {
        deadline: "2025-12-31",
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-date-field"),
      issueResponse("issue-id-75"),
      existingItemResponse("issue-id-75", "item-date-field"),
      // DATE field with dataType explicitly set to "DATE"
      // This tests that the code checks dataType before checking for options
      fieldsResponse([{ id: "field-deadline", name: "Deadline", dataType: "DATE" }]),
      updateFieldValueResponse(),
    ]);

    await updateProject(output);

    // Verify the field value is set using date format, not singleSelectOptionId
    const updateCall = mockGithub.graphql.mock.calls.find(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCall).toBeDefined();
    expect(updateCall[1].value).toEqual({ date: "2025-12-31" });
    // Explicitly verify it's NOT using singleSelectOptionId
    expect(updateCall[1].value).not.toHaveProperty("singleSelectOptionId");
  });

  it("auto-populates Start Date and End Date from issue timestamps", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 100,
      campaign_id: "test-campaign",
    };

    const issueWithTimestamps = {
      repository: {
        issue: {
          id: "issue-id-100",
          createdAt: "2025-12-15T10:30:00Z",
          closedAt: "2025-12-18T16:45:00Z",
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-auto-date"),
      issueWithTimestamps,
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "item-auto-date" } } },
      fieldsResponse([
        { id: "field-start-auto", name: "Start Date", dataType: "DATE" },
        { id: "field-end-auto", name: "End Date", dataType: "DATE" },
      ]),
      updateFieldValueResponse(),
      updateFieldValueResponse(),
    ]);

    await updateProject(output);

    // Verify auto-population messages
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Auto-populating Start Date: 2025-12-15"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Auto-populating End Date: 2025-12-18"));

    // Verify the field values were set with correct dates
    const updateCalls = mockGithub.graphql.mock.calls.filter(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCalls).toHaveLength(2);
    expect(updateCalls[0][1].value).toEqual({ date: "2025-12-15" });
    expect(updateCalls[1][1].value).toEqual({ date: "2025-12-18" });
  });

  it("auto-populates only Start Date when issue is not closed", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 101,
      campaign_id: "test-campaign",
    };

    const openIssue = {
      repository: {
        issue: {
          id: "issue-id-101",
          createdAt: "2025-12-20T14:00:00Z",
          closedAt: null,
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-open-issue"),
      openIssue,
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "item-open-issue" } } },
      fieldsResponse([
        { id: "field-start-open", name: "Start Date", dataType: "DATE" },
        { id: "field-end-open", name: "End Date", dataType: "DATE" },
      ]),
      updateFieldValueResponse(),
    ]);

    await updateProject(output);

    // Verify only Start Date is auto-populated
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Auto-populating Start Date: 2025-12-20"));
    expect(mockCore.info).not.toHaveBeenCalledWith(expect.stringContaining("Auto-populating End Date"));

    // Verify only one field update call
    const updateCalls = mockGithub.graphql.mock.calls.filter(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCalls).toHaveLength(1);
    expect(updateCalls[0][1].value).toEqual({ date: "2025-12-20" });
  });

  it("does not auto-populate when user explicitly provides date fields", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 102,
      campaign_id: "test-campaign",
      fields: {
        start_date: "2025-11-01",
        end_date: "2025-11-30",
      },
    };

    const issueWithTimestamps = {
      repository: {
        issue: {
          id: "issue-id-102",
          createdAt: "2025-12-01T10:00:00Z",
          closedAt: "2025-12-05T18:00:00Z",
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-user-dates"),
      issueWithTimestamps,
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "item-user-dates" } } },
      fieldsResponse([
        { id: "field-start-user", name: "Start Date", dataType: "DATE" },
        { id: "field-end-user", name: "End Date", dataType: "DATE" },
      ]),
      updateFieldValueResponse(),
      updateFieldValueResponse(),
      fieldsResponse([
        { id: "field-start-user", name: "Start Date", dataType: "DATE" },
        { id: "field-end-user", name: "End Date", dataType: "DATE" },
      ]),
    ]);

    await updateProject(output);

    // Verify skipping messages
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Start Date was explicitly provided, skipping auto-population"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("End Date was explicitly provided, skipping auto-population"));

    // Verify user-provided values are used (not the auto-populated timestamps)
    const updateCalls = mockGithub.graphql.mock.calls.filter(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCalls).toHaveLength(2);
    expect(updateCalls[0][1].value).toEqual({ date: "2025-11-01" });
    expect(updateCalls[1][1].value).toEqual({ date: "2025-11-30" });
  });

  it("auto-populates Worker Workflow field from issue body", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 200,
      campaign_id: "test-campaign",
    };

    const issueWithWorkerWorkflow = {
      repository: {
        issue: {
          id: "issue-id-200",
          createdAt: "2025-12-20T10:00:00Z",
          closedAt: null,
          body: "Issue description\n\n<!-- agentic-workflow: Daily Updater, tracker-id: daily-123 -->\n\nMore content",
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-worker"),
      issueWithWorkerWorkflow,
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "item-worker" } } },
      fieldsResponse([
        { id: "field-start", name: "Start Date", dataType: "DATE" },
        { id: "field-worker", name: "Worker Workflow", dataType: "SINGLE_SELECT", options: [] },
      ]),
      updateFieldValueResponse(), // Start Date update
      fieldsResponse([
        { id: "field-start", name: "Start Date", dataType: "DATE" },
        { id: "field-worker", name: "Worker Workflow", dataType: "SINGLE_SELECT", options: [] },
      ]),
      // Worker Workflow option creation
      {
        updateProjectV2Field: {
          projectV2Field: {
            id: "field-worker",
            options: [{ id: "option-daily", name: "Daily Updater" }],
          },
        },
      },
      updateFieldValueResponse(), // Worker Workflow update
    ]);

    await updateProject(output);

    // Verify Worker Workflow auto-population message
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Auto-populating Worker Workflow: Daily Updater"));

    // Verify the field was updated
    const updateCalls = mockGithub.graphql.mock.calls.filter(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCalls.length).toBeGreaterThanOrEqual(2); // At least Start Date and Worker Workflow
  });

  it("skips Worker Workflow auto-population when field doesn't exist", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 201,
      campaign_id: "test-campaign",
    };

    const issueWithWorkerWorkflow = {
      repository: {
        issue: {
          id: "issue-id-201",
          createdAt: "2025-12-20T10:00:00Z",
          closedAt: null,
          body: "<!-- agentic-workflow: Test Workflow, tracker-id: test-123 -->",
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-no-worker"),
      issueWithWorkerWorkflow,
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "item-no-worker" } } },
      fieldsResponse([{ id: "field-start", name: "Start Date", dataType: "DATE" }]),
      updateFieldValueResponse(), // Start Date update
      fieldsResponse([{ id: "field-start", name: "Start Date", dataType: "DATE" }]), // No Worker Workflow field
    ]);

    await updateProject(output);

    // Verify no Worker Workflow field message
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No Worker Workflow field found on project board"));
  });

  it("skips Worker Workflow when no XML comment in body", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 202,
      campaign_id: "test-campaign",
    };

    const issueWithoutWorkerWorkflow = {
      repository: {
        issue: {
          id: "issue-id-202",
          createdAt: "2025-12-20T10:00:00Z",
          closedAt: null,
          body: "Regular issue with no workflow marker",
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-no-marker"),
      issueWithoutWorkerWorkflow,
      emptyItemsResponse(),
      { addProjectV2ItemById: { item: { id: "item-no-marker" } } },
      fieldsResponse([{ id: "field-start", name: "Start Date", dataType: "DATE" }]),
      updateFieldValueResponse(), // Start Date update
    ]);

    await updateProject(output);

    // Verify skip message
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No workflow name found in issue/PR body"));
  });

  it("does not auto-populate Start Date when existing item already has it set", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 300,
      campaign_id: "test-campaign",
    };

    const issueWithTimestamps = {
      repository: {
        issue: {
          id: "issue-id-300",
          createdAt: "2025-12-25T10:00:00Z",
          closedAt: null,
          body: "Issue with existing Start Date",
        },
      },
    };

    // Existing item with Start Date already set
    const existingItemWithStartDate = {
      node: {
        items: {
          nodes: [
            {
              id: "existing-item-300",
              content: { id: "issue-id-300" },
              fieldValues: {
                nodes: [
                  {
                    field: { id: "field-start", name: "Start Date" },
                    date: "2025-12-20",
                  },
                ],
              },
            },
          ],
          pageInfo: { hasNextPage: false, endCursor: null },
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-existing-start"),
      issueWithTimestamps,
      existingItemWithStartDate,
      fieldsResponse([
        { id: "field-start", name: "Start Date", dataType: "DATE" },
        { id: "field-end", name: "End Date", dataType: "DATE" },
      ]),
    ]);

    await updateProject(output);

    // Verify skip message for existing Start Date
    expect(mockCore.info).toHaveBeenCalledWith("✓ Item already on board");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Start Date already set on existing item, skipping auto-population"));

    // Verify no Start Date update call was made
    const updateCalls = mockGithub.graphql.mock.calls.filter(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCalls).toHaveLength(0);
  });

  it("does not auto-populate End Date when existing item already has it set", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 301,
      campaign_id: "test-campaign",
    };

    const issueWithTimestamps = {
      repository: {
        issue: {
          id: "issue-id-301",
          createdAt: "2025-12-20T10:00:00Z",
          closedAt: "2025-12-25T16:00:00Z",
          body: "Issue with existing End Date",
        },
      },
    };

    // Existing item with End Date already set
    const existingItemWithEndDate = {
      node: {
        items: {
          nodes: [
            {
              id: "existing-item-301",
              content: { id: "issue-id-301" },
              fieldValues: {
                nodes: [
                  {
                    field: { id: "field-end", name: "End Date" },
                    date: "2025-12-26",
                  },
                ],
              },
            },
          ],
          pageInfo: { hasNextPage: false, endCursor: null },
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-existing-end"),
      issueWithTimestamps,
      existingItemWithEndDate,
      fieldsResponse([
        { id: "field-start", name: "Start Date", dataType: "DATE" },
        { id: "field-end", name: "End Date", dataType: "DATE" },
      ]),
      updateFieldValueResponse(), // Start Date update only
    ]);

    await updateProject(output);

    // Verify skip message for existing End Date
    expect(mockCore.info).toHaveBeenCalledWith("✓ Item already on board");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("End Date already set on existing item, skipping auto-population"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Auto-populating Start Date: 2025-12-20"));

    // Verify only Start Date update call was made
    const updateCalls = mockGithub.graphql.mock.calls.filter(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCalls).toHaveLength(1);
    expect(updateCalls[0][1].value).toEqual({ date: "2025-12-20" });
  });

  it("does not auto-populate Worker Workflow when existing item already has it set", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 302,
      campaign_id: "test-campaign",
    };

    const issueWithWorkerWorkflow = {
      repository: {
        issue: {
          id: "issue-id-302",
          createdAt: "2025-12-20T10:00:00Z",
          closedAt: null,
          body: "<!-- agentic-workflow: WorkflowA, run 123 -->\nIssue with worker workflow",
        },
      },
    };

    // Existing item with Worker Workflow already set
    const existingItemWithWorkerWorkflow = {
      node: {
        items: {
          nodes: [
            {
              id: "existing-item-302",
              content: { id: "issue-id-302" },
              fieldValues: {
                nodes: [
                  {
                    field: { id: "field-worker", name: "Worker Workflow" },
                    name: "WorkflowB",
                  },
                ],
              },
            },
          ],
          pageInfo: { hasNextPage: false, endCursor: null },
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-existing-worker"),
      issueWithWorkerWorkflow,
      existingItemWithWorkerWorkflow,
      fieldsResponse([{ id: "field-start", name: "Start Date", dataType: "DATE" }]),
      updateFieldValueResponse(), // Start Date update only
    ]);

    await updateProject(output);

    // Verify skip message for existing Worker Workflow
    expect(mockCore.info).toHaveBeenCalledWith("✓ Item already on board");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Worker Workflow already set on existing item, skipping auto-population"));

    // Verify no Worker Workflow update call was made (only Start Date)
    const updateCalls = mockGithub.graphql.mock.calls.filter(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCalls).toHaveLength(1);
    expect(updateCalls[0][1].value).toEqual({ date: "2025-12-20" });
  });

  it("auto-populates empty date and workflow fields on existing items", async () => {
    const projectUrl = "https://github.com/orgs/testowner/projects/60";
    const output = {
      type: "update_project",
      project: projectUrl,
      content_type: "issue",
      content_number: 303,
      campaign_id: "test-campaign",
    };

    const issueWithTimestamps = {
      repository: {
        issue: {
          id: "issue-id-303",
          createdAt: "2025-12-20T10:00:00Z",
          closedAt: "2025-12-25T16:00:00Z",
          body: "<!-- agentic-workflow: WorkflowC, run 456 -->\nIssue with empty fields",
        },
      },
    };

    // Existing item with no field values set
    const existingItemWithNoFields = {
      node: {
        items: {
          nodes: [
            {
              id: "existing-item-303",
              content: { id: "issue-id-303" },
              fieldValues: {
                nodes: [],
              },
            },
          ],
          pageInfo: { hasNextPage: false, endCursor: null },
        },
      },
    };

    queueResponses([
      repoResponse(),
      viewerResponse(),
      orgProjectV2Response(projectUrl, 60, "project-empty-fields"),
      issueWithTimestamps,
      existingItemWithNoFields,
      fieldsResponse([
        { id: "field-start", name: "Start Date", dataType: "DATE" },
        { id: "field-end", name: "End Date", dataType: "DATE" },
      ]),
      updateFieldValueResponse(), // Start Date update
      updateFieldValueResponse(), // End Date update
      fieldsResponse([{ id: "field-worker", name: "Worker Workflow", dataType: "SINGLE_SELECT", options: [] }]),
      {
        updateProjectV2Field: {
          projectV2Field: {
            id: "field-worker",
            options: [{ id: "option-workflowc", name: "WorkflowC" }],
          },
        },
      },
      updateFieldValueResponse(), // Worker Workflow update
    ]);

    await updateProject(output);

    // Verify auto-population messages
    expect(mockCore.info).toHaveBeenCalledWith("✓ Item already on board");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Auto-populating Start Date: 2025-12-20"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Auto-populating End Date: 2025-12-25"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Auto-populating Worker Workflow: WorkflowC"));

    // Verify all three field updates were made
    const updateCalls = mockGithub.graphql.mock.calls.filter(([query]) => query.includes("updateProjectV2ItemFieldValue"));
    expect(updateCalls).toHaveLength(3);
    expect(updateCalls[0][1].value).toEqual({ date: "2025-12-20" });
    expect(updateCalls[1][1].value).toEqual({ date: "2025-12-25" });
    expect(updateCalls[2][1].value).toEqual({ singleSelectOptionId: "option-workflowc" });
  });
});

describe("hasExistingFieldValue", () => {
  it("returns false when existingItem is null", () => {
    expect(hasExistingFieldValue(null, "Start Date")).toBe(false);
  });

  it("returns false when existingItem has no fieldValues", () => {
    const item = { id: "item-1", content: { id: "content-1" } };
    expect(hasExistingFieldValue(item, "Start Date")).toBe(false);
  });

  it("returns false when fieldValues.nodes is not an array", () => {
    const item = { id: "item-1", fieldValues: { nodes: null } };
    expect(hasExistingFieldValue(item, "Start Date")).toBe(false);
  });

  it("returns false when fieldValues.nodes is empty", () => {
    const item = { id: "item-1", fieldValues: { nodes: [] } };
    expect(hasExistingFieldValue(item, "Start Date")).toBe(false);
  });

  it("returns true when date field has a value", () => {
    const item = {
      id: "item-1",
      fieldValues: {
        nodes: [
          {
            field: { id: "field-1", name: "Start Date" },
            date: "2025-12-20",
          },
        ],
      },
    };
    expect(hasExistingFieldValue(item, "Start Date")).toBe(true);
  });

  it("returns true when single-select field has a value", () => {
    const item = {
      id: "item-1",
      fieldValues: {
        nodes: [
          {
            field: { id: "field-1", name: "Worker Workflow" },
            name: "WorkflowA",
          },
        ],
      },
    };
    expect(hasExistingFieldValue(item, "Worker Workflow")).toBe(true);
  });

  it("returns true when text field has a non-empty value", () => {
    const item = {
      id: "item-1",
      fieldValues: {
        nodes: [
          {
            field: { id: "field-1", name: "Status" },
            text: "In Progress",
          },
        ],
      },
    };
    expect(hasExistingFieldValue(item, "Status")).toBe(true);
  });

  it("returns false when text field has an empty value", () => {
    const item = {
      id: "item-1",
      fieldValues: {
        nodes: [
          {
            field: { id: "field-1", name: "Status" },
            text: "   ",
          },
        ],
      },
    };
    expect(hasExistingFieldValue(item, "Status")).toBe(false);
  });

  it("performs case-insensitive field name matching", () => {
    const item = {
      id: "item-1",
      fieldValues: {
        nodes: [
          {
            field: { id: "field-1", name: "Start Date" },
            date: "2025-12-20",
          },
        ],
      },
    };
    expect(hasExistingFieldValue(item, "start date")).toBe(true);
    expect(hasExistingFieldValue(item, "START DATE")).toBe(true);
  });

  it("returns false when field name does not match", () => {
    const item = {
      id: "item-1",
      fieldValues: {
        nodes: [
          {
            field: { id: "field-1", name: "Start Date" },
            date: "2025-12-20",
          },
        ],
      },
    };
    expect(hasExistingFieldValue(item, "End Date")).toBe(false);
  });

  it("skips nodes with missing field data", () => {
    const item = {
      id: "item-1",
      fieldValues: {
        nodes: [
          null,
          { field: null },
          { field: { name: null } },
          {
            field: { id: "field-1", name: "Start Date" },
            date: "2025-12-20",
          },
        ],
      },
    };
    expect(hasExistingFieldValue(item, "Start Date")).toBe(true);
  });
});
