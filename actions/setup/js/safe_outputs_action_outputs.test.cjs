// @ts-check
import { describe, it, expect, beforeEach } from "vitest";
import { emitSafeOutputActionOutputs } from "./safe_outputs_action_outputs.cjs";

describe("emitSafeOutputActionOutputs", () => {
  /** @type {Record<string, string>} */
  let outputs;
  /** @type {string[]} */
  let infoMessages;

  beforeEach(() => {
    outputs = {};
    infoMessages = [];

    global.core = {
      info: (/** @type {string} */ msg) => {
        infoMessages.push(msg);
      },
      setOutput: (/** @type {string} */ name, /** @type {string} */ value) => {
        outputs[name] = value;
      },
    };
  });

  it("emits created_issue_number and created_issue_url for create_issue result", () => {
    emitSafeOutputActionOutputs({
      results: [{ success: true, type: "create_issue", result: { number: 42, url: "https://github.com/owner/repo/issues/42" } }],
    });

    expect(outputs["created_issue_number"]).toBe("42");
    expect(outputs["created_issue_url"]).toBe("https://github.com/owner/repo/issues/42");
  });

  it("emits only the first successful create_issue result", () => {
    emitSafeOutputActionOutputs({
      results: [
        { success: true, type: "create_issue", result: { number: 1, url: "https://github.com/owner/repo/issues/1" } },
        { success: true, type: "create_issue", result: { number: 2, url: "https://github.com/owner/repo/issues/2" } },
      ],
    });

    expect(outputs["created_issue_number"]).toBe("1");
    expect(outputs["created_issue_url"]).toBe("https://github.com/owner/repo/issues/1");
  });

  it("skips failed create_issue results", () => {
    emitSafeOutputActionOutputs({
      results: [
        { success: false, type: "create_issue", result: { number: 99, url: "https://github.com/owner/repo/issues/99" } },
        { success: true, type: "create_issue", result: { number: 5, url: "https://github.com/owner/repo/issues/5" } },
      ],
    });

    expect(outputs["created_issue_number"]).toBe("5");
  });

  it("emits created_pr_number and created_pr_url for create_pull_request result", () => {
    emitSafeOutputActionOutputs({
      results: [{ success: true, type: "create_pull_request", result: { pull_request_number: 7, pull_request_url: "https://github.com/owner/repo/pull/7" } }],
    });

    expect(outputs["created_pr_number"]).toBe("7");
    expect(outputs["created_pr_url"]).toBe("https://github.com/owner/repo/pull/7");
  });

  it("emits comment_id and comment_url for add_comment result", () => {
    emitSafeOutputActionOutputs({
      results: [{ success: true, type: "add_comment", result: { commentId: 123, url: "https://github.com/owner/repo/issues/1#issuecomment-123" } }],
    });

    expect(outputs["comment_id"]).toBe("123");
    expect(outputs["comment_url"]).toBe("https://github.com/owner/repo/issues/1#issuecomment-123");
  });

  it("unwraps array result for add_comment and uses the first element", () => {
    emitSafeOutputActionOutputs({
      results: [
        {
          success: true,
          type: "add_comment",
          result: [
            { commentId: 10, url: "https://github.com/owner/repo/issues/1#issuecomment-10" },
            { commentId: 20, url: "https://github.com/owner/repo/issues/1#issuecomment-20" },
          ],
        },
      ],
    });

    expect(outputs["comment_id"]).toBe("10");
    expect(outputs["comment_url"]).toBe("https://github.com/owner/repo/issues/1#issuecomment-10");
  });

  it("emits push_commit_sha and push_commit_url for push_to_pull_request_branch result", () => {
    emitSafeOutputActionOutputs({
      results: [{ success: true, type: "push_to_pull_request_branch", result: { commit_sha: "abc123", commit_url: "https://github.com/owner/repo/commit/abc123" } }],
    });

    expect(outputs["push_commit_sha"]).toBe("abc123");
    expect(outputs["push_commit_url"]).toBe("https://github.com/owner/repo/commit/abc123");
  });

  it("emits outputs for multiple different types in a single run", () => {
    emitSafeOutputActionOutputs({
      results: [
        { success: true, type: "create_issue", result: { number: 1, url: "https://github.com/owner/repo/issues/1" } },
        { success: true, type: "add_comment", result: { commentId: 200, url: "https://github.com/owner/repo/issues/1#issuecomment-200" } },
        { success: true, type: "push_to_pull_request_branch", result: { commit_sha: "sha1", commit_url: "https://github.com/owner/repo/commit/sha1" } },
      ],
    });

    expect(outputs["created_issue_number"]).toBe("1");
    expect(outputs["comment_id"]).toBe("200");
    expect(outputs["push_commit_sha"]).toBe("sha1");
  });

  it("emits no outputs when there are no successful results", () => {
    emitSafeOutputActionOutputs({
      results: [{ success: false, type: "create_issue", result: null }],
    });

    expect(Object.keys(outputs)).toHaveLength(0);
  });

  it("emits no outputs when results array is empty", () => {
    emitSafeOutputActionOutputs({ results: [] });

    expect(Object.keys(outputs)).toHaveLength(0);
  });

  it("skips create_issue result with array result (not a single issue)", () => {
    emitSafeOutputActionOutputs({
      results: [{ success: true, type: "create_issue", result: [{ number: 1 }] }],
    });

    expect(outputs["created_issue_number"]).toBeUndefined();
  });

  it("logs info messages for each emitted output", () => {
    emitSafeOutputActionOutputs({
      results: [{ success: true, type: "create_issue", result: { number: 3, url: "https://github.com/owner/repo/issues/3" } }],
    });

    expect(infoMessages.some(m => m.includes("created_issue_number"))).toBe(true);
    expect(infoMessages.some(m => m.includes("created_issue_url"))).toBe(true);
  });
});
