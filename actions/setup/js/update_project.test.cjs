import { describe, it, expect, beforeAll, beforeEach, afterEach, vi } from "vitest";

let updateProject;
let parseProjectInput;
let generateCampaignId;

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

const issueResponse = id => ({ repository: { issue: { id } } });

const pullRequestResponse = id => ({ repository: { pullRequest: { id } } });

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
    ]);

    await updateProject(output);

    // Should NOT auto-populate any date fields
    expect(mockCore.info).not.toHaveBeenCalledWith(expect.stringContaining("Auto-populating"));

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
});
