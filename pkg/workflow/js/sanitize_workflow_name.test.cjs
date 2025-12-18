import { describe, it as test, expect } from "vitest";
const { sanitizeWorkflowName } = require("./sanitize_workflow_name.cjs");
describe("sanitize_workflow_name.cjs", () => {
  describe("sanitizeWorkflowName", () => {
    (test("should convert to lowercase", () => {
      expect(sanitizeWorkflowName("MyWorkflow")).toBe("myworkflow");
    }),
      test("should replace colons with hyphens", () => {
        expect(sanitizeWorkflowName("workflow:name")).toBe("workflow-name");
      }),
      test("should replace backslashes with hyphens", () => {
        expect(sanitizeWorkflowName("workflow\\name")).toBe("workflow-name");
      }),
      test("should replace forward slashes with hyphens", () => {
        expect(sanitizeWorkflowName("workflow/name")).toBe("workflow-name");
      }),
      test("should replace spaces with hyphens", () => {
        expect(sanitizeWorkflowName("workflow name")).toBe("workflow-name");
      }),
      test("should replace invalid characters with hyphens", () => {
        expect(sanitizeWorkflowName("workflow@name!")).toBe("workflow-name-");
      }),
      test("should preserve dots, underscores, and hyphens", () => {
        expect(sanitizeWorkflowName("my-workflow_v1.0")).toBe("my-workflow_v1.0");
      }),
      test("should handle complex workflow names", () => {
        expect(sanitizeWorkflowName("My Workflow: v1.0 (test)")).toBe("my-workflow--v1.0--test-");
      }),
      test("should handle empty string", () => {
        expect(sanitizeWorkflowName("")).toBe("");
      }),
      test("should handle already sanitized names", () => {
        expect(sanitizeWorkflowName("my-workflow")).toBe("my-workflow");
      }));
  });
});
