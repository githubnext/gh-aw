// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
};

// Set up global mocks
global.core = mockCore;

describe("update_discussion.cjs - buildDiscussionUpdateData", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.resetModules();
  });

  describe("title permission gating", () => {
    it("should block title when allow_title is not set", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ title: "New Title" }, {});
      expect(result.success).toBe(true);
      expect(result.data.title).toBeUndefined();
    });

    it("should block title when allow_title is false", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ title: "New Title" }, { allow_title: false });
      expect(result.success).toBe(true);
      expect(result.data.title).toBeUndefined();
    });

    it("should allow title when allow_title is true", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ title: "New Title" }, { allow_title: true });
      expect(result.success).toBe(true);
      expect(result.data.title).toBe("New Title");
    });
  });

  describe("body permission gating", () => {
    it("should block body when allow_body is not set", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ body: "New body" }, {});
      expect(result.success).toBe(true);
      expect(result.data.body).toBeUndefined();
    });

    it("should block body when allow_body is false", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ body: "New body" }, { allow_body: false });
      expect(result.success).toBe(true);
      expect(result.data.body).toBeUndefined();
    });

    it("should allow body when allow_body is true", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ body: "New body" }, { allow_body: true });
      expect(result.success).toBe(true);
      expect(result.data.body).toBe("New body");
    });
  });

  describe("label permission gating", () => {
    it("should block labels when allow_labels is not set", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ labels: ["bug"] }, {});
      expect(result.success).toBe(true);
      expect(result.data.labels).toBeUndefined();
    });

    it("should allow labels when allow_labels is true", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ labels: ["bug"] }, { allow_labels: true });
      expect(result.success).toBe(true);
      expect(result.data.labels).toEqual(["bug"]);
    });

    it("should fail when labels is not an array", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ labels: "bug" }, { allow_labels: true });
      expect(result.success).toBe(false);
      expect(result.error).toContain("must be an array");
    });

    it("should filter labels by allowed_labels list", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ labels: ["bug", "wontfix", "enhancement"] }, { allow_labels: true, allowed_labels: ["bug", "enhancement"] });
      expect(result.success).toBe(true);
      expect(result.data.labels).toEqual(["bug", "enhancement"]);
    });

    it("should fail when all labels are filtered out by allowed_labels", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ labels: ["wontfix"] }, { allow_labels: true, allowed_labels: ["bug", "enhancement"] });
      expect(result.success).toBe(false);
    });

    it("should accept up to MAX_LABELS (10) labels without truncation", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const tenLabels = ["l1", "l2", "l3", "l4", "l5", "l6", "l7", "l8", "l9", "l10"];
      const result = buildDiscussionUpdateData({ labels: tenLabels }, { allow_labels: true });
      expect(result.success).toBe(true);
      expect(result.data.labels).toHaveLength(10);
      expect(result.data.labels).toEqual(tenLabels);
    });

    it("should accept 4+ labels (regression: was silently truncated to 3)", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const fiveLabels = ["bug", "enhancement", "documentation", "help wanted", "question"];
      const result = buildDiscussionUpdateData({ labels: fiveLabels }, { allow_labels: true });
      expect(result.success).toBe(true);
      expect(result.data.labels).toHaveLength(5);
    });

    it("should reject more than MAX_LABELS (10) labels", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const elevenLabels = ["l1", "l2", "l3", "l4", "l5", "l6", "l7", "l8", "l9", "l10", "l11"];
      const result = buildDiscussionUpdateData({ labels: elevenLabels }, { allow_labels: true });
      expect(result.success).toBe(false);
      expect(result.error).toContain("E003");
    });
  });

  describe("labels-only config does not set title or body", () => {
    it("should produce updateData with labels but no title or body", async () => {
      const { buildDiscussionUpdateData } = await import("./update_discussion.cjs");
      const result = buildDiscussionUpdateData({ title: "Some Title", body: "Some Body", labels: ["bug"] }, { allow_labels: true });
      expect(result.success).toBe(true);
      expect(result.data.title).toBeUndefined();
      expect(result.data.body).toBeUndefined();
      expect(result.data.labels).toEqual(["bug"]);
    });
  });
});
