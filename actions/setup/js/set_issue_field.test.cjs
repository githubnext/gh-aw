import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
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
      number: 123,
    },
  },
};

const mockGraphql = vi.fn();

const mockGithub = {
  rest: {
    issues: {
      get: vi.fn(),
    },
  },
  graphql: mockGraphql,
};

global.core = mockCore;
global.context = mockContext;
global.github = mockGithub;

describe("set_issue_field (Handler Factory Architecture)", () => {
  let handler;

  const issueNodeId = "I_kwDOABCD123456";
  const textFieldId = "IF_kwDO_text";
  const statusFieldId = "IF_kwDO_status";
  const effortFieldId = "IF_kwDO_direct";

  const mockIssueFieldsQuery = {
    repository: {
      issueFields: {
        nodes: [
          { id: textFieldId, name: "Customer Impact", __typename: "IssueFieldText" },
          {
            id: statusFieldId,
            name: "Status",
            __typename: "IssueFieldSingleSelect",
            options: [
              { id: "IFOPT_open", name: "Open" },
              { id: "IFOPT_closed", name: "Closed" },
            ],
          },
          { id: effortFieldId, name: "Effort", __typename: "IssueFieldNumber" },
        ],
      },
      owner: {
        __typename: "Organization",
        issueFields: {
          nodes: [],
        },
      },
    },
  };

  beforeEach(async () => {
    vi.clearAllMocks();

    mockGithub.rest.issues.get.mockResolvedValue({ data: { node_id: issueNodeId } });
    mockGraphql.mockImplementation(query => {
      if (query.includes("issueFields")) {
        return Promise.resolve(mockIssueFieldsQuery);
      }
      if (query.includes("setIssueFieldValue")) {
        return Promise.resolve({ setIssueFieldValue: { issue: { id: issueNodeId } } });
      }
      return Promise.resolve({});
    });

    const { main } = require("./set_issue_field.cjs");
    handler = await main({ max: 5 });
  });

  it("should return a function from main()", async () => {
    const { main } = require("./set_issue_field.cjs");
    const result = await main({});
    expect(typeof result).toBe("function");
  });

  it("should set issue text field successfully", async () => {
    const message = {
      type: "set_issue_field",
      issue_number: 42,
      field_name: "Customer Impact",
      value: "High",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.issue_number).toBe(42);
    expect(result.field_name).toBe("Customer Impact");
    expect(result.field_node_id).toBe(textFieldId);
    expect(mockGraphql).toHaveBeenCalledWith(
      expect.stringContaining("setIssueFieldValue"),
      expect.objectContaining({
        issueId: issueNodeId,
        issueFields: [expect.objectContaining({ fieldId: textFieldId, textValue: "High" })],
      })
    );
  });

  it("should set single-select field by option name", async () => {
    const message = {
      type: "set_issue_field",
      issue_number: 42,
      field_name: "Status",
      value: "Closed",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(mockGraphql).toHaveBeenCalledWith(
      expect.stringContaining("setIssueFieldValue"),
      expect.objectContaining({
        issueFields: [expect.objectContaining({ fieldId: statusFieldId, singleSelectOptionId: "IFOPT_closed" })],
      })
    );
  });

  it("should error with actionable message for unknown field name", async () => {
    const message = {
      type: "set_issue_field",
      issue_number: 42,
      field_name: "Unknown Field",
      value: "foo",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
    expect(result.error).toContain("Available fields");
    expect(result.error).toContain("field_node_id");
  });

  it("should error with actionable message for invalid single-select value", async () => {
    const message = {
      type: "set_issue_field",
      issue_number: 42,
      field_name: "Status",
      value: "Invalid",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid value");
    expect(result.error).toContain("Available options");
  });

  it("should resolve field type when field_node_id is provided", async () => {
    const message = {
      type: "set_issue_field",
      issue_number: 42,
      field_node_id: effortFieldId,
      value: "3.5",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.field_node_id).toBe(effortFieldId);
    expect(mockGraphql).toHaveBeenCalledWith(expect.stringContaining("repository(owner"), expect.anything());
    expect(mockGraphql).toHaveBeenCalledWith(
      expect.stringContaining("setIssueFieldValue"),
      expect.objectContaining({
        issueFields: [expect.objectContaining({ fieldId: effortFieldId, numberValue: 3.5 })],
      })
    );
  });

  it("should error when provided field_node_id is unknown", async () => {
    const message = {
      type: "set_issue_field",
      issue_number: 42,
      field_node_id: "IF_kwDO_missing",
      value: "3.5",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
    expect(result.error).toContain("Available fields");
  });

  it("should enforce configured allowed-fields list", async () => {
    const { main } = require("./set_issue_field.cjs");
    const restrictedHandler = await main({
      allowed_fields: ["Status"],
    });

    const result = await restrictedHandler({
      type: "set_issue_field",
      issue_number: 42,
      field_name: "Customer Impact",
      value: "High",
    });

    expect(result.success).toBe(false);
    expect(result.error).toContain('issue field "Customer Impact" is not in the allowed-fields list: Status');
  });

  it("should allow any field when allowed-fields includes wildcard", async () => {
    const { main } = require("./set_issue_field.cjs");
    const unrestrictedHandler = await main({
      allowed_fields: ["*"],
    });

    const result = await unrestrictedHandler({
      type: "set_issue_field",
      issue_number: 42,
      field_name: "Customer Impact",
      value: "High",
    });

    expect(result.success).toBe(true);
  });

  it("should skip gracefully with a warning when no issue fields are discovered", async () => {
    mockGraphql.mockImplementation(query => {
      if (query.includes("issueFields")) {
        return Promise.resolve({
          repository: {
            issueFields: { nodes: [] },
            owner: {
              __typename: "Organization",
              issueFields: { nodes: [] },
            },
          },
        });
      }
      return Promise.resolve({});
    });

    const { main } = require("./set_issue_field.cjs");
    const noFieldsHandler = await main({ max: 5 });

    const result = await noFieldsHandler({
      type: "set_issue_field",
      issue_number: 42,
      field_name: "Priority",
      value: "High",
    });

    expect(result.success).toBe(false);
    expect(result.skipped).toBe(true);
    expect(result.error).toContain("No issue fields were discovered");
    expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("No issue fields were discovered"));
    expect(mockCore.error).not.toHaveBeenCalled();
  });

  it("fetchIssueFields query uses only concrete issue field type fragments", async () => {
    let capturedQuery = "";
    mockGraphql.mockImplementation(query => {
      if (query.includes("repository(owner")) {
        capturedQuery = query;
        return Promise.resolve(mockIssueFieldsQuery);
      }
      if (query.includes("setIssueFieldValue")) {
        return Promise.resolve({ setIssueFieldValue: { issue: { id: issueNodeId } } });
      }
      return Promise.resolve({});
    });

    const { main } = require("./set_issue_field.cjs");
    const h = await main({ max: 5 });
    await h({ type: "set_issue_field", issue_number: 42, field_name: "Customer Impact", value: "High" }, {});

    // The query must not select id or name directly on the union — only inside concrete fragments
    // Bare "id" or "name" would appear on their own line; inside a fragment they appear as "{ id name }"
    expect(capturedQuery).not.toMatch(/^\s+id\s*$/m);
    expect(capturedQuery).not.toMatch(/^\s+name\s*$/m);
    // The query must not include the non-existent IssueField type fragment
    expect(capturedQuery).not.toMatch(/\.\.\.\s+on\s+IssueField\s*\{/);
    // The query must include at least the concrete IssueFieldText fragment
    expect(capturedQuery).toContain("... on IssueFieldText");
    // The query must include the IssueFieldSingleSelect fragment with options
    expect(capturedQuery).toContain("... on IssueFieldSingleSelect");
    expect(capturedQuery).toContain("options");
  });

  it("fetchIssueFields query does not include a User.issueFields selection", async () => {
    let capturedQuery = "";
    mockGraphql.mockImplementation(query => {
      if (query.includes("repository(owner")) {
        capturedQuery = query;
        return Promise.resolve(mockIssueFieldsQuery);
      }
      if (query.includes("setIssueFieldValue")) {
        return Promise.resolve({ setIssueFieldValue: { issue: { id: issueNodeId } } });
      }
      return Promise.resolve({});
    });

    const { main } = require("./set_issue_field.cjs");
    const h = await main({ max: 5 });
    await h({ type: "set_issue_field", issue_number: 42, field_name: "Customer Impact", value: "High" }, {});

    expect(capturedQuery).not.toContain("... on User");
  });

  it("fetchIssueFields filters out nodes missing id or name (null entries and unknown types)", async () => {
    mockGraphql.mockImplementation(query => {
      if (query.includes("issueFields")) {
        return Promise.resolve({
          repository: {
            issueFields: {
              nodes: [
                null,
                { __typename: "IssueFieldText", id: textFieldId, name: "Customer Impact" },
                { __typename: "IssueFieldUnknown" }, // missing id and name
                { __typename: "IssueFieldUnknown", id: "IF_x" }, // missing name
                { __typename: "IssueFieldUnknown", name: "Orphan" }, // missing id
              ],
            },
            owner: { __typename: "Organization", issueFields: { nodes: [] } },
          },
        });
      }
      if (query.includes("setIssueFieldValue")) {
        return Promise.resolve({ setIssueFieldValue: { issue: { id: issueNodeId } } });
      }
      return Promise.resolve({});
    });

    const { main } = require("./set_issue_field.cjs");
    const h = await main({ max: 5 });
    // Only the IssueFieldText node (with valid id+name) should survive filtering
    const result = await h({ type: "set_issue_field", issue_number: 42, field_name: "Customer Impact", value: "High" }, {});
    expect(result.success).toBe(true);
    expect(result.field_node_id).toBe(textFieldId);
  });

  it("GraphQL discovery errors propagate as actionable errors, not as 'No issue fields were discovered'", async () => {
    const graphqlError = new Error("Selections can't be made directly on unions (see selections on IssueFields)");
    mockGraphql.mockImplementation(query => {
      if (query.includes("repository(owner")) {
        return Promise.reject(graphqlError);
      }
      if (query.includes("setIssueFieldValue")) {
        return Promise.resolve({ setIssueFieldValue: { issue: { id: issueNodeId } } });
      }
      return Promise.resolve({});
    });

    const { main } = require("./set_issue_field.cjs");
    const errorHandler = await main({ max: 5 });

    const result = await errorHandler({
      type: "set_issue_field",
      issue_number: 42,
      field_name: "Priority",
      value: "High",
    });

    expect(result.success).toBe(false);
    // Must NOT be silently treated as "no fields discovered"
    expect(result.skipped).not.toBe(true);
    expect(result.error).not.toContain("No issue fields were discovered");
    // Must surface the actual error
    expect(result.error).toContain("Selections can't be made directly on unions");
    expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("No issue fields were discovered"));
  });
});
