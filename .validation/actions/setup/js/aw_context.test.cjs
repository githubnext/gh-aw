import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import fs from "fs";

// resolveItemContext does not depend on global context — it only operates on
// the plain payload object, so we can test it directly without any mocking.
const { resolveItemContext } = await import("./aw_context.cjs");
const { EXPERIMENT_ASSIGNMENTS_PATH } = await import("./experiment_helpers.cjs");

describe("resolveItemContext", () => {
  it("returns issue type and number for issues events", () => {
    const payload = { issue: { number: 42 } };
    expect(resolveItemContext(payload)).toEqual({ item_type: "issue", item_number: "42", comment_id: "", comment_node_id: "" });
  });

  it("returns issue type with comment_id for issue_comment events", () => {
    const payload = { issue: { number: 7 }, comment: { id: 99001122 } };
    expect(resolveItemContext(payload)).toEqual({ item_type: "issue", item_number: "7", comment_id: "99001122", comment_node_id: "" });
  });

  it("returns pull_request type with comment_id for issue_comment events on pull requests", () => {
    // GitHub sends issue_comment events for PR comments with payload.issue.pull_request set
    const payload = { issue: { number: 7, pull_request: {} }, comment: { id: 99001122 } };
    expect(resolveItemContext(payload)).toEqual({ item_type: "pull_request", item_number: "7", comment_id: "99001122", comment_node_id: "" });
  });

  it("returns pull_request type and number for pull_request events", () => {
    const payload = { pull_request: { number: 100 } };
    expect(resolveItemContext(payload)).toEqual({ item_type: "pull_request", item_number: "100", comment_id: "", comment_node_id: "" });
  });

  it("returns pull_request type with review id for pull_request_review events", () => {
    const payload = { pull_request: { number: 100 }, review: { id: 55667788 } };
    expect(resolveItemContext(payload)).toEqual({
      item_type: "pull_request",
      item_number: "100",
      comment_id: "55667788",
      comment_node_id: "",
    });
  });

  it("returns pull_request type with comment_id for pull_request_review_comment events", () => {
    const payload = { pull_request: { number: 100 }, comment: { id: 11223344 }, review: { id: 55667788 } };
    // comment.id takes priority over review.id
    expect(resolveItemContext(payload)).toEqual({
      item_type: "pull_request",
      item_number: "100",
      comment_id: "11223344",
      comment_node_id: "",
    });
  });

  it("returns discussion type and number for discussion events", () => {
    const payload = { discussion: { number: 5 } };
    expect(resolveItemContext(payload)).toEqual({ item_type: "discussion", item_number: "5", comment_id: "", comment_node_id: "" });
  });

  it("returns discussion type with comment_id for discussion_comment events", () => {
    const payload = { discussion: { number: 5 }, comment: { id: 77889900 } };
    expect(resolveItemContext(payload)).toEqual({
      item_type: "discussion",
      item_number: "5",
      comment_id: "77889900",
      comment_node_id: "",
    });
  });

  it("returns discussion type with comment_node_id for discussion_comment events with node_id", () => {
    const payload = { discussion: { number: 5 }, comment: { id: 77889900, node_id: "DC_kwDOParentComment456" } };
    expect(resolveItemContext(payload)).toEqual({
      item_type: "discussion",
      item_number: "5",
      comment_id: "77889900",
      comment_node_id: "DC_kwDOParentComment456",
    });
  });

  it("returns check_run type and id for check_run events", () => {
    const payload = { check_run: { id: 7654321 } };
    expect(resolveItemContext(payload)).toEqual({ item_type: "check_run", item_number: "7654321", comment_id: "", comment_node_id: "" });
  });

  it("returns check_suite type and id for check_suite events", () => {
    const payload = { check_suite: { id: 9988776 } };
    expect(resolveItemContext(payload)).toEqual({ item_type: "check_suite", item_number: "9988776", comment_id: "", comment_node_id: "" });
  });

  it("returns empty strings for push/workflow_dispatch events (no item payload)", () => {
    expect(resolveItemContext({})).toEqual({ item_type: "", item_number: "", comment_id: "", comment_node_id: "" });
    expect(resolveItemContext(null)).toEqual({ item_type: "", item_number: "", comment_id: "", comment_node_id: "" });
    expect(resolveItemContext(undefined)).toEqual({ item_type: "", item_number: "", comment_id: "", comment_node_id: "" });
  });

  it("returns empty item_number when number is null", () => {
    const payload = { issue: { number: null } };
    expect(resolveItemContext(payload)).toEqual({ item_type: "issue", item_number: "", comment_id: "", comment_node_id: "" });
  });
});

describe("buildAwContext experiments field", () => {
  let readFileSpy;
  const savedStateDir = process.env.GH_AW_EXPERIMENT_STATE_DIR;

  beforeEach(() => {
    delete process.env.GH_AW_EXPERIMENT_STATE_DIR;
    // Set up a minimal global context required by buildAwContext
    global.context = {
      repo: { owner: "test-owner", repo: "test-repo" },
      runId: 12345,
      actor: "octocat",
      eventName: "issues",
      payload: { issue: { number: 1 } },
    };
    readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
  });

  afterEach(() => {
    readFileSpy.mockRestore();
    if (savedStateDir !== undefined) {
      process.env.GH_AW_EXPERIMENT_STATE_DIR = savedStateDir;
    } else {
      delete process.env.GH_AW_EXPERIMENT_STATE_DIR;
    }
    delete global.context;
  });

  it("includes experiments as empty string when no assignments file exists", async () => {
    const { buildAwContext } = await import("./aw_context.cjs");
    const result = buildAwContext();
    expect(result.experiments).toBe("");
  });

  it("includes experiments as compact JSON string when assignments file exists", async () => {
    readFileSpy.mockImplementation(filePath => {
      if (filePath === EXPERIMENT_ASSIGNMENTS_PATH) {
        return JSON.stringify({ caveman: "yes", style: "detailed" });
      }
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
    const { buildAwContext } = await import("./aw_context.cjs");
    const result = buildAwContext();
    expect(result.experiments).toBe(JSON.stringify({ caveman: "yes", style: "detailed" }));
  });

  it("experiments is a string primitive (not an object) to satisfy aw_context validation", async () => {
    readFileSpy.mockReturnValue(JSON.stringify({ feature: "beta" }));
    const { buildAwContext } = await import("./aw_context.cjs");
    const result = buildAwContext();
    expect(typeof result.experiments).toBe("string");
  });
});

describe("buildAwContext allow_bot_authored_trigger_comment field", () => {
  let readFileSpy;

  beforeEach(() => {
    readFileSpy = vi.spyOn(fs, "readFileSync").mockImplementation(() => {
      throw Object.assign(new Error("ENOENT"), { code: "ENOENT" });
    });
  });

  afterEach(() => {
    readFileSpy.mockRestore();
    delete global.context;
  });

  it("is false for non-issue_comment events", async () => {
    global.context = {
      repo: { owner: "o", repo: "r" },
      runId: 1,
      actor: "octocat",
      eventName: "pull_request",
      payload: { action: "synchronize", pull_request: { user: { login: "octocat" } } },
    };
    const { buildAwContext } = await import("./aw_context.cjs");
    expect(buildAwContext().allow_bot_authored_trigger_comment).toBe(false);
  });

  it("is false for issue_comment:created (not edited)", async () => {
    global.context = {
      repo: { owner: "o", repo: "r" },
      runId: 1,
      actor: "theletterf",
      eventName: "issue_comment",
      payload: { action: "created", comment: { user: { login: "github-actions[bot]" } } },
    };
    const { buildAwContext } = await import("./aw_context.cjs");
    expect(buildAwContext().allow_bot_authored_trigger_comment).toBe(false);
  });

  it("is false for issue_comment:edited when actor matches comment author", async () => {
    global.context = {
      repo: { owner: "o", repo: "r" },
      runId: 1,
      actor: "octocat",
      eventName: "issue_comment",
      payload: { action: "edited", comment: { user: { login: "octocat" } } },
    };
    const { buildAwContext } = await import("./aw_context.cjs");
    expect(buildAwContext().allow_bot_authored_trigger_comment).toBe(false);
  });

  it("is true for issue_comment:edited when actor differs from [bot]-authored comment (bot-menu pattern)", async () => {
    // A workflow posted a checkbox-menu comment as github-actions[bot].
    // A human maintainer (theletterf) edited it to tick a box.
    global.context = {
      repo: { owner: "o", repo: "r" },
      runId: 1,
      actor: "theletterf",
      eventName: "issue_comment",
      payload: { action: "edited", comment: { user: { login: "github-actions[bot]" } } },
    };
    const { buildAwContext } = await import("./aw_context.cjs");
    expect(buildAwContext().allow_bot_authored_trigger_comment).toBe(true);
  });

  it("is false for issue_comment:edited when comment author is a human (not a [bot])", async () => {
    // A maintainer editing another human's comment should NOT set the flag —
    // only bot-authored comments qualify for the bot-menu exception.
    global.context = {
      repo: { owner: "o", repo: "r" },
      runId: 1,
      actor: "theletterf",
      eventName: "issue_comment",
      payload: { action: "edited", comment: { user: { login: "another-human" } } },
    };
    const { buildAwContext } = await import("./aw_context.cjs");
    expect(buildAwContext().allow_bot_authored_trigger_comment).toBe(false);
  });

  it("is false for issue_comment:edited when comment.user.login is absent from payload", async () => {
    global.context = {
      repo: { owner: "o", repo: "r" },
      runId: 1,
      actor: "theletterf",
      eventName: "issue_comment",
      payload: { action: "edited", comment: {} },
    };
    const { buildAwContext } = await import("./aw_context.cjs");
    expect(buildAwContext().allow_bot_authored_trigger_comment).toBe(false);
  });
});
