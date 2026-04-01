// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

// messages_core.cjs calls core.warning on parse failures - provide a stub
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};
global.core = mockCore;

const { getRunStartedMessage, getRunSuccessMessage, getRunFailureMessage, getDetectionFailureMessage, getPullRequestCreatedMessage, getIssueCreatedMessage, getCommitPushedMessage } = require("./messages_run_status.cjs");

const WORKFLOW = "My Workflow";
const RUN_URL = "https://github.com/owner/repo/actions/runs/99";

describe("messages_run_status", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
  });

  describe("getRunStartedMessage", () => {
    it("returns default template with all placeholders substituted", () => {
      const msg = getRunStartedMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, eventType: "issue" });
      expect(msg).toBe(`🚀 [${WORKFLOW}](${RUN_URL}) has started processing this issue`);
    });

    it("supports different event types", () => {
      expect(getRunStartedMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, eventType: "pull request" })).toContain("pull request");
      expect(getRunStartedMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, eventType: "discussion" })).toContain("discussion");
    });

    it("uses custom template from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ runStarted: "Custom: {workflow_name} started" });
      const msg = getRunStartedMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, eventType: "issue" });
      expect(msg).toBe(`Custom: ${WORKFLOW} started`);
    });

    it("substitutes camelCase keys as well as snake_case", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ runStarted: "{workflowName} at {runUrl}" });
      const msg = getRunStartedMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, eventType: "issue" });
      expect(msg).toBe(`${WORKFLOW} at ${RUN_URL}`);
    });
  });

  describe("getRunSuccessMessage", () => {
    it("returns default template with placeholders substituted", () => {
      const msg = getRunSuccessMessage({ workflowName: WORKFLOW, runUrl: RUN_URL });
      expect(msg).toBe(`✅ [${WORKFLOW}](${RUN_URL}) completed successfully!`);
    });

    it("uses custom template from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ runSuccess: "Done: {workflow_name}" });
      const msg = getRunSuccessMessage({ workflowName: WORKFLOW, runUrl: RUN_URL });
      expect(msg).toBe(`Done: ${WORKFLOW}`);
    });

    it("ignores unrelated config keys and uses default", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ runStarted: "overridden" });
      const msg = getRunSuccessMessage({ workflowName: WORKFLOW, runUrl: RUN_URL });
      expect(msg).toContain("completed successfully");
    });
  });

  describe("getRunFailureMessage", () => {
    it("returns default template with status substituted", () => {
      const msg = getRunFailureMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, status: "failed" });
      expect(msg).toBe(`❌ [${WORKFLOW}](${RUN_URL}) failed. Please review the logs for details.`);
    });

    it("handles different status values", () => {
      expect(getRunFailureMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, status: "was cancelled" })).toContain("was cancelled");
      expect(getRunFailureMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, status: "timed out" })).toContain("timed out");
    });

    it("uses custom template from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ runFailure: "FAILED: {workflow_name} - {status}" });
      const msg = getRunFailureMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, status: "failed" });
      expect(msg).toBe(`FAILED: ${WORKFLOW} - failed`);
    });
  });

  describe("getDetectionFailureMessage", () => {
    it("returns default template with placeholders substituted", () => {
      const msg = getDetectionFailureMessage({ workflowName: WORKFLOW, runUrl: RUN_URL });
      expect(msg).toBe(`⚠️ Security scanning failed for [${WORKFLOW}](${RUN_URL}). Review the logs for details.`);
    });

    it("uses custom template from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ detectionFailure: "Security alert for {workflow_name}" });
      const msg = getDetectionFailureMessage({ workflowName: WORKFLOW, runUrl: RUN_URL });
      expect(msg).toBe(`Security alert for ${WORKFLOW}`);
    });
  });

  describe("getPullRequestCreatedMessage", () => {
    it("returns default template with item_number and item_url substituted", () => {
      const msg = getPullRequestCreatedMessage({ itemNumber: 42, itemUrl: "https://github.com/owner/repo/pull/42" });
      expect(msg).toBe("Pull request created: [#42](https://github.com/owner/repo/pull/42)");
    });

    it("uses custom template from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ pullRequestCreated: "PR #{item_number} ready" });
      const msg = getPullRequestCreatedMessage({ itemNumber: 7, itemUrl: "https://github.com/owner/repo/pull/7" });
      expect(msg).toBe("PR #7 ready");
    });
  });

  describe("getIssueCreatedMessage", () => {
    it("returns default template with item_number and item_url substituted", () => {
      const msg = getIssueCreatedMessage({ itemNumber: 15, itemUrl: "https://github.com/owner/repo/issues/15" });
      expect(msg).toBe("Issue created: [#15](https://github.com/owner/repo/issues/15)");
    });

    it("uses custom template from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ issueCreated: "New issue #{item_number}" });
      const msg = getIssueCreatedMessage({ itemNumber: 3, itemUrl: "https://github.com/owner/repo/issues/3" });
      expect(msg).toBe("New issue #3");
    });
  });

  describe("getCommitPushedMessage", () => {
    const SHA = "abc1234def5678901234567890123456789012ab";
    const SHORT = "abc1234";
    const COMMIT_URL = `https://github.com/owner/repo/commit/${SHA}`;

    it("returns default template with short_sha and commit_url substituted", () => {
      const msg = getCommitPushedMessage({ commitSha: SHA, shortSha: SHORT, commitUrl: COMMIT_URL });
      expect(msg).toBe(`Commit pushed: [\`${SHORT}\`](${COMMIT_URL})`);
    });

    it("uses custom template from config", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ commitPushed: "Pushed {short_sha} to repo" });
      const msg = getCommitPushedMessage({ commitSha: SHA, shortSha: SHORT, commitUrl: COMMIT_URL });
      expect(msg).toBe(`Pushed ${SHORT} to repo`);
    });

    it("supports full SHA in custom template", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ commitPushed: "{commit_sha}" });
      const msg = getCommitPushedMessage({ commitSha: SHA, shortSha: SHORT, commitUrl: COMMIT_URL });
      expect(msg).toBe(SHA);
    });
  });

  describe("fallback when config is missing keys", () => {
    it("uses default template when only unrelated config keys are set", () => {
      process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({ footer: "custom footer" });
      expect(getRunStartedMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, eventType: "issue" })).toContain("has started processing");
      expect(getRunSuccessMessage({ workflowName: WORKFLOW, runUrl: RUN_URL })).toContain("completed successfully");
      expect(getRunFailureMessage({ workflowName: WORKFLOW, runUrl: RUN_URL, status: "failed" })).toContain("failed");
    });
  });
});
