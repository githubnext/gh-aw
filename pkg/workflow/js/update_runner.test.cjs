import { describe, it, expect, beforeEach, vi } from "vitest";

describe("update_runner.cjs", () => {
  let helpers;

  beforeEach(async () => {
    helpers = await import("./update_runner.cjs");
  });

  describe("resolveTargetNumber", () => {
    it("should resolve explicit number from item with wildcard target", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "*",
        item: { issue_number: 123 },
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(true);
      expect(result.number).toBe(123);
    });

    it("should fail with wildcard target but no explicit number", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "*",
        item: {},
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(false);
      expect(result.error).toContain('Target is "*"');
      expect(result.error).toContain("issue_number");
    });

    it("should fail with wildcard target and invalid number", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "*",
        item: { issue_number: "invalid" },
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid issue_number");
    });

    it("should fail with wildcard target and zero number", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "*",
        item: { issue_number: 0 },
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(false);
      // 0 is falsy, so it's treated as "no number specified"
      expect(result.error).toContain("issue_number");
    });

    it("should resolve explicit target number", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "789",
        item: {},
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(true);
      expect(result.number).toBe(789);
    });

    it("should fail with invalid explicit target", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "invalid",
        item: {},
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid issue number");
    });

    it("should fail with zero explicit target", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "0",
        item: {},
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(false);
      expect(result.error).toContain("Invalid issue number");
    });

    it("should resolve from triggering context", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "triggering",
        item: {},
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(true);
      expect(result.number).toBe(456);
    });

    it("should resolve from triggering context when target is empty", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "",
        item: {},
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(true);
      expect(result.number).toBe(456);
    });

    it("should fail when triggering but no context number", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "triggering",
        item: {},
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: undefined,
        displayName: "issue",
      });
      expect(result.success).toBe(false);
      expect(result.error).toContain("Could not determine issue number");
    });

    it("should fail when triggering but context is invalid", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "triggering",
        item: {},
        numberField: "issue_number",
        isValidContext: false,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(false);
      expect(result.error).toContain("Could not determine issue number");
    });

    it("should work with pull_request_number field", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "*",
        item: { pull_request_number: 999 },
        numberField: "pull_request_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "pull request",
      });
      expect(result.success).toBe(true);
      expect(result.number).toBe(999);
    });

    it("should handle string number in item", () => {
      const result = helpers.resolveTargetNumber({
        updateTarget: "*",
        item: { issue_number: "123" },
        numberField: "issue_number",
        isValidContext: true,
        contextNumber: 456,
        displayName: "issue",
      });
      expect(result.success).toBe(true);
      expect(result.number).toBe(123);
    });
  });

  describe("buildUpdateData", () => {
    it("should include status when allowed and valid", () => {
      const result = helpers.buildUpdateData({
        item: { status: "closed" },
        canUpdateStatus: true,
        canUpdateTitle: false,
        canUpdateBody: false,
        supportsStatus: true,
      });
      expect(result.hasUpdates).toBe(true);
      expect(result.updateData.state).toBe("closed");
      expect(result.logMessages).toContain("Will update status to: closed");
    });

    it("should include status open when allowed and valid", () => {
      const result = helpers.buildUpdateData({
        item: { status: "open" },
        canUpdateStatus: true,
        canUpdateTitle: false,
        canUpdateBody: false,
        supportsStatus: true,
      });
      expect(result.hasUpdates).toBe(true);
      expect(result.updateData.state).toBe("open");
    });

    it("should reject invalid status value", () => {
      const result = helpers.buildUpdateData({
        item: { status: "invalid" },
        canUpdateStatus: true,
        canUpdateTitle: false,
        canUpdateBody: false,
        supportsStatus: true,
      });
      expect(result.hasUpdates).toBe(false);
      expect(result.updateData.state).toBeUndefined();
      expect(result.logMessages.some(m => m.includes("Invalid status value"))).toBe(true);
    });

    it("should skip status when not allowed", () => {
      const result = helpers.buildUpdateData({
        item: { status: "closed" },
        canUpdateStatus: false,
        canUpdateTitle: false,
        canUpdateBody: false,
        supportsStatus: true,
      });
      expect(result.hasUpdates).toBe(false);
      expect(result.updateData.state).toBeUndefined();
    });

    it("should skip status when not supported", () => {
      const result = helpers.buildUpdateData({
        item: { status: "closed" },
        canUpdateStatus: true,
        canUpdateTitle: false,
        canUpdateBody: false,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(false);
      expect(result.updateData.state).toBeUndefined();
    });

    it("should include title when allowed and valid", () => {
      const result = helpers.buildUpdateData({
        item: { title: "New Title" },
        canUpdateStatus: false,
        canUpdateTitle: true,
        canUpdateBody: false,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(true);
      expect(result.updateData.title).toBe("New Title");
      expect(result.logMessages).toContain("Will update title to: New Title");
    });

    it("should trim title whitespace", () => {
      const result = helpers.buildUpdateData({
        item: { title: "  Trimmed Title  " },
        canUpdateStatus: false,
        canUpdateTitle: true,
        canUpdateBody: false,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(true);
      expect(result.updateData.title).toBe("Trimmed Title");
    });

    it("should reject empty title", () => {
      const result = helpers.buildUpdateData({
        item: { title: "   " },
        canUpdateStatus: false,
        canUpdateTitle: true,
        canUpdateBody: false,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(false);
      expect(result.updateData.title).toBeUndefined();
      expect(result.logMessages.some(m => m.includes("Invalid title value"))).toBe(true);
    });

    it("should reject non-string title", () => {
      const result = helpers.buildUpdateData({
        item: { title: 123 },
        canUpdateStatus: false,
        canUpdateTitle: true,
        canUpdateBody: false,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(false);
      expect(result.updateData.title).toBeUndefined();
    });

    it("should skip title when not allowed", () => {
      const result = helpers.buildUpdateData({
        item: { title: "New Title" },
        canUpdateStatus: false,
        canUpdateTitle: false,
        canUpdateBody: false,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(false);
      expect(result.updateData.title).toBeUndefined();
    });

    it("should include body when allowed and valid", () => {
      const result = helpers.buildUpdateData({
        item: { body: "New body content" },
        canUpdateStatus: false,
        canUpdateTitle: false,
        canUpdateBody: true,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(true);
      expect(result.updateData.body).toBe("New body content");
      expect(result.logMessages.some(m => m.includes("Will update body"))).toBe(true);
    });

    it("should include empty string body", () => {
      const result = helpers.buildUpdateData({
        item: { body: "" },
        canUpdateStatus: false,
        canUpdateTitle: false,
        canUpdateBody: true,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(true);
      expect(result.updateData.body).toBe("");
    });

    it("should reject non-string body", () => {
      const result = helpers.buildUpdateData({
        item: { body: 123 },
        canUpdateStatus: false,
        canUpdateTitle: false,
        canUpdateBody: true,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(false);
      expect(result.updateData.body).toBeUndefined();
      expect(result.logMessages.some(m => m.includes("Invalid body value"))).toBe(true);
    });

    it("should skip body when not allowed", () => {
      const result = helpers.buildUpdateData({
        item: { body: "New body" },
        canUpdateStatus: false,
        canUpdateTitle: false,
        canUpdateBody: false,
        supportsStatus: false,
      });
      expect(result.hasUpdates).toBe(false);
      expect(result.updateData.body).toBeUndefined();
    });

    it("should handle multiple fields", () => {
      const result = helpers.buildUpdateData({
        item: { title: "New Title", body: "New body", status: "closed" },
        canUpdateStatus: true,
        canUpdateTitle: true,
        canUpdateBody: true,
        supportsStatus: true,
      });
      expect(result.hasUpdates).toBe(true);
      expect(result.updateData.title).toBe("New Title");
      expect(result.updateData.body).toBe("New body");
      expect(result.updateData.state).toBe("closed");
      expect(result.logMessages.length).toBe(3);
    });

    it("should return false hasUpdates when no valid updates", () => {
      const result = helpers.buildUpdateData({
        item: {},
        canUpdateStatus: true,
        canUpdateTitle: true,
        canUpdateBody: true,
        supportsStatus: true,
      });
      expect(result.hasUpdates).toBe(false);
      expect(Object.keys(result.updateData).length).toBe(0);
    });

    it("should not include fields when item value is undefined", () => {
      const result = helpers.buildUpdateData({
        item: { title: undefined, body: undefined, status: undefined },
        canUpdateStatus: true,
        canUpdateTitle: true,
        canUpdateBody: true,
        supportsStatus: true,
      });
      expect(result.hasUpdates).toBe(false);
    });
  });

  describe("createRenderStagedItem", () => {
    it("should render issue update with explicit number", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Issue",
        numberField: "issue_number",
        targetLabel: "Target Issue:",
        currentTargetText: "Current issue",
        includeOperation: false,
      });

      const result = render({ issue_number: 123, title: "New Title" }, 0);

      expect(result).toContain("#### Issue Update 1");
      expect(result).toContain("**Target Issue:** #123");
      expect(result).toContain("**New Title:** New Title");
    });

    it("should render issue update without explicit number", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Issue",
        numberField: "issue_number",
        targetLabel: "Target Issue:",
        currentTargetText: "Current issue",
        includeOperation: false,
      });

      const result = render({ title: "New Title" }, 0);

      expect(result).toContain("#### Issue Update 1");
      expect(result).toContain("**Target:** Current issue");
      expect(result).toContain("**New Title:** New Title");
    });

    it("should render PR update with operation field", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Pull Request",
        numberField: "pull_request_number",
        targetLabel: "Target PR:",
        currentTargetText: "Current pull request",
        includeOperation: true,
      });

      const result = render({ pull_request_number: 456, body: "New body content", operation: "prepend" }, 0);

      expect(result).toContain("### Pull Request Update 1");
      expect(result).toContain("**Target PR:** #456");
      expect(result).toContain("**Operation:** prepend");
      expect(result).toContain("**Body Content:**\nNew body content");
    });

    it("should render body without operation when includeOperation is false", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Issue",
        numberField: "issue_number",
        targetLabel: "Target Issue:",
        currentTargetText: "Current issue",
        includeOperation: false,
      });

      const result = render({ body: "New body content" }, 0);

      expect(result).toContain("**New Body:**\nNew body content");
      expect(result).not.toContain("**Operation:");
    });

    it("should render status when present", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Issue",
        numberField: "issue_number",
        targetLabel: "Target Issue:",
        currentTargetText: "Current issue",
        includeOperation: false,
      });

      const result = render({ status: "closed" }, 0);

      expect(result).toContain("**New Status:** closed");
    });

    it("should use default operation when includeOperation is true but operation not specified", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Pull Request",
        numberField: "pull_request_number",
        targetLabel: "Target PR:",
        currentTargetText: "Current pull request",
        includeOperation: true,
      });

      const result = render({ body: "New body content" }, 0);

      expect(result).toContain("**Operation:** append");
      expect(result).toContain("**Body Content:**\nNew body content");
    });

    it("should increment index correctly", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Issue",
        numberField: "issue_number",
        targetLabel: "Target Issue:",
        currentTargetText: "Current issue",
        includeOperation: false,
      });

      const result = render({ title: "Title" }, 4);

      expect(result).toContain("#### Issue Update 5");
    });
  });

  describe("createGetSummaryLine", () => {
    it("should generate issue summary line", () => {
      const getSummaryLine = helpers.createGetSummaryLine({
        entityPrefix: "Issue",
      });

      const result = getSummaryLine({
        number: 123,
        title: "Test Issue",
        html_url: "https://github.com/owner/repo/issues/123",
      });

      expect(result).toBe("- Issue #123: [Test Issue](https://github.com/owner/repo/issues/123)\n");
    });

    it("should generate PR summary line", () => {
      const getSummaryLine = helpers.createGetSummaryLine({
        entityPrefix: "PR",
      });

      const result = getSummaryLine({
        number: 456,
        title: "Test PR",
        html_url: "https://github.com/owner/repo/pull/456",
      });

      expect(result).toBe("- PR #456: [Test PR](https://github.com/owner/repo/pull/456)\n");
    });

    it("should handle special characters in title", () => {
      const getSummaryLine = helpers.createGetSummaryLine({
        entityPrefix: "Issue",
      });

      const result = getSummaryLine({
        number: 789,
        title: "Fix [bug] with <special> chars",
        html_url: "https://github.com/owner/repo/issues/789",
      });

      expect(result).toBe("- Issue #789: [Fix [bug] with <special> chars](https://github.com/owner/repo/issues/789)\n");
    });
  });

  describe("Default operation behavior", () => {
    it("should default to append when operation is not specified in renderStagedItem", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Pull Request",
        numberField: "pull_request_number",
        targetLabel: "Target PR:",
        currentTargetText: "Current pull request",
        includeOperation: true,
      });

      const result = render({ body: "New body content" }, 0);

      // Should show "append" as the default operation
      expect(result).toContain("**Operation:** append");
    });

    it("should respect explicit append operation in renderStagedItem", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Pull Request",
        numberField: "pull_request_number",
        targetLabel: "Target PR:",
        currentTargetText: "Current pull request",
        includeOperation: true,
      });

      const result = render({ body: "New body content", operation: "append" }, 0);

      expect(result).toContain("**Operation:** append");
    });

    it("should respect explicit prepend operation in renderStagedItem", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Pull Request",
        numberField: "pull_request_number",
        targetLabel: "Target PR:",
        currentTargetText: "Current pull request",
        includeOperation: true,
      });

      const result = render({ body: "New body content", operation: "prepend" }, 0);

      expect(result).toContain("**Operation:** prepend");
    });

    it("should respect explicit replace operation in renderStagedItem", () => {
      const render = helpers.createRenderStagedItem({
        entityName: "Pull Request",
        numberField: "pull_request_number",
        targetLabel: "Target PR:",
        currentTargetText: "Current pull request",
        includeOperation: true,
      });

      const result = render({ body: "New body content", operation: "replace" }, 0);

      expect(result).toContain("**Operation:** replace");
    });
  });
});
