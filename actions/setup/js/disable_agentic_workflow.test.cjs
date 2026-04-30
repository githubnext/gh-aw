// @ts-check

import { describe, it, expect } from "vitest";
import { extractWorkflowId } from "./disable_agentic_workflow.cjs";

describe("extractWorkflowId", () => {
  it("returns null for null body", () => {
    expect(extractWorkflowId(null)).toBeNull();
  });

  it("returns null for undefined body", () => {
    expect(extractWorkflowId(undefined)).toBeNull();
  });

  it("returns null for empty body", () => {
    expect(extractWorkflowId("")).toBeNull();
  });

  it("returns null when no marker is present", () => {
    expect(extractWorkflowId("This is a normal issue body with no markers.")).toBeNull();
  });

  it("extracts workflow ID from standalone marker", () => {
    const body = "Some issue text\n\n<!-- gh-aw-workflow-id: my-workflow -->";
    expect(extractWorkflowId(body)).toBe("my-workflow");
  });

  it("extracts workflow ID from standalone marker with extra whitespace", () => {
    const body = "<!-- gh-aw-workflow-id:   code-review   -->";
    expect(extractWorkflowId(body)).toBe("code-review");
  });

  it("extracts workflow ID from combined agentic-workflow marker (comma-separated)", () => {
    const body = "Issue body\n" + "<!-- gh-aw-agentic-workflow: My Workflow, gh-aw-tracker-id: abc123, engine: copilot, workflow_id: ci-doctor, run: https://github.com/owner/repo/actions/runs/123 -->";
    expect(extractWorkflowId(body)).toBe("ci-doctor");
  });

  it("extracts workflow ID from combined marker when workflow_id is last before closing -->", () => {
    const body = "<!-- gh-aw-agentic-workflow: My Workflow, workflow_id: auto-fix, run: https://example.com -->";
    expect(extractWorkflowId(body)).toBe("auto-fix");
  });

  it("prefers standalone marker over combined marker when both are present", () => {
    const body = "<!-- gh-aw-workflow-id: standalone-workflow -->\n" + "<!-- gh-aw-agentic-workflow: Name, workflow_id: combined-workflow, run: https://example.com -->";
    expect(extractWorkflowId(body)).toBe("standalone-workflow");
  });

  it("handles workflow IDs with dots", () => {
    const body = "<!-- gh-aw-workflow-id: my.workflow.v2 -->";
    expect(extractWorkflowId(body)).toBe("my.workflow.v2");
  });

  it("handles workflow IDs with underscores", () => {
    const body = "<!-- gh-aw-workflow-id: code_review_bot -->";
    expect(extractWorkflowId(body)).toBe("code_review_bot");
  });

  it("extracts from body with substantial content before marker", () => {
    const body = [
      "## Issue Title",
      "",
      "This is a long description of the issue created by an agentic workflow.",
      "",
      "> Closed by [My Workflow](https://github.com/owner/repo/actions/runs/123)",
      "",
      "<!-- gh-aw-expired-comments -->",
      "<!-- gh-aw-workflow-id: expired-issue-workflow -->",
      "<!-- gh-aw-agentic-workflow: My Workflow, workflow_id: expired-issue-workflow, run: https://github.com/owner/repo/actions/runs/123 -->",
    ].join("\n");
    expect(extractWorkflowId(body)).toBe("expired-issue-workflow");
  });

  it("returns null for workflow_id outside of an XML comment block", () => {
    // workflow_id: appearing outside a gh-aw-agentic-workflow comment should NOT be extracted
    const body = "The workflow_id: my-injected-id is mentioned in user text.";
    expect(extractWorkflowId(body)).toBeNull();
  });

  it("returns null for workflow ID with path traversal attempt", () => {
    const body = "<!-- gh-aw-workflow-id: ../secrets -->";
    expect(extractWorkflowId(body)).toBeNull();
  });

  it("returns null for workflow ID with shell-special characters", () => {
    // The regex won't match ';' since it requires [\w.-]+ followed by whitespace/-->
    const body = "<!-- gh-aw-workflow-id: my;workflow -->";
    expect(extractWorkflowId(body)).toBeNull();
  });
});
